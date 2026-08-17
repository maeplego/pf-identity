package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
	"github.com/portfolio/pf-identity-server/internal/store/storetest"
)

var testURL string

func TestMain(m *testing.M) {
	u, cleanup := lookupTestDatabase()
	testURL = u
	code := m.Run()
	if cleanup != nil {
		cleanup()
	}
	os.Exit(code)
}

func TestReposContract(t *testing.T) {
	s := openTestStore(t)
	storetest.Repos(t, s)
}

func TestTakeCodeConcurrent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	hash := id.New()
	if err := s.PutCode(ctx, domain.AuthCode{
		Hash:          hash,
		ClientID:      "c",
		UserID:        "u",
		RedirectURI:   "http://127.0.0.1/cb",
		Scopes:        []string{"openid"},
		CodeChallenge: "ch",
		ExpiresAt:     time.Now().UTC().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := s.TakeCode(ctx, hash)
			errCh <- err
		}()
	}
	var ok, used int
	for i := 0; i < 2; i++ {
		err := <-errCh
		switch {
		case err == nil:
			ok++
		case errors.Is(err, domain.ErrUsed):
			used++
		default:
			t.Fatalf("unexpected take error: %v", err)
		}
	}
	if ok != 1 || used != 1 {
		t.Fatalf("concurrent take ok=%d used=%d", ok, used)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	if testURL == "" {
		t.Skip("set IDENTITY_TEST_DATABASE_URL or install Docker to run Postgres store tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	s, err := Open(ctx, testURL)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func lookupTestDatabase() (string, func()) {
	if u := strings.TrimSpace(os.Getenv("IDENTITY_TEST_DATABASE_URL")); u != "" {
		return u, nil
	}
	if strings.TrimSpace(os.Getenv("IDENTITY_SKIP_DOCKER")) == "1" {
		return "", nil
	}
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintf(os.Stderr, "postgres tests: docker not on PATH: %v\n", err)
		return "", nil
	}
	name := "pf-identity-pg-" + fmt.Sprintf("%d", os.Getpid())
	run := exec.Command("docker", "run", "-d", "--rm", "--name", name,
		"-e", "POSTGRES_USER=idp",
		"-e", "POSTGRES_PASSWORD=idp",
		"-e", "POSTGRES_DB=idp_test",
		"-p", "127.0.0.1::5432",
		"postgres:16-alpine",
	)
	out, err := run.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres tests: docker run failed: %v\n%s\n", err, out)
		return "", nil
	}
	cleanup := func() { _ = exec.Command("docker", "rm", "-f", name).Run() }
	deadline := time.Now().Add(45 * time.Second)
	var url string
	for time.Now().Before(deadline) {
		portOut, err := exec.Command("docker", "port", name, "5432/tcp").Output()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		hostport := strings.TrimSpace(strings.Split(string(portOut), "\n")[0])
		url = "postgres://idp:idp@" + hostport + "/idp_test?sslmode=disable"
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		s, err := Open(ctx, url)
		cancel()
		if err == nil {
			s.Close()
			return url, cleanup
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "postgres tests: container did not become ready; last url=%s\n", url)
	cleanup()
	return "", nil
}
