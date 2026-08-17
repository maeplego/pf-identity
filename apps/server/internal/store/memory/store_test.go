package memory

import (
	"testing"

	"github.com/portfolio/pf-identity-server/internal/store/storetest"
)

func TestReposContract(t *testing.T) {
	storetest.Repos(t, NewStore())
}
