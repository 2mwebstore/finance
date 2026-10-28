package database

import (
	"os"
	"strings"
	"testing"
)

// The startup check and the versioned migration must create the same tables and keys.
func TestPlatformMigrationMatchesStartupSQL(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000013_platforms.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	squash := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	migration := squash(string(raw))
	// The migration builds each table as a string (to add the branches collation), so its
	// quotes are doubled.
	for _, stmt := range platformTables {
		if !strings.Contains(migration, squash(strings.ReplaceAll(stmt, "'", "''"))) {
			t.Errorf("migration 000013 is missing or differs from:\n%s", stmt)
		}
	}
	for _, fk := range platformForeignKeys {
		if !strings.Contains(migration, squash(fk.sql())) {
			t.Errorf("migration 000013 is missing foreign key %s", fk.name)
		}
	}
	if !strings.Contains(migration, squash(seedPlatforms)) {
		t.Error("migration 000013 is missing the platform seed")
	}
}
