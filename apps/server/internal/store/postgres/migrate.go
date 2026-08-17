package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var applied string
		err := s.pool.QueryRow(ctx, `SELECT version FROM schema_migrations WHERE version = $1`, name).Scan(&applied)
		if err == nil {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if err := s.execScript(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			return err
		}
	}
	return nil
}

// execScript runs multi-statement SQL. pgx's default extended protocol rejects that.
func (s *Store) execScript(ctx context.Context, sql string) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	batch := conn.Conn().PgConn().Exec(ctx, "BEGIN;\n"+sql+"\nCOMMIT;")
	if _, err := batch.ReadAll(); err != nil {
		_, _ = conn.Conn().PgConn().Exec(ctx, "ROLLBACK;").ReadAll()
		return err
	}
	return nil
}
