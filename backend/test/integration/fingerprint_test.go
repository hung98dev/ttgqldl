package integration

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/schema"
)

var schemaTablePattern = regexp.MustCompile(`(?m)^CREATE TABLE ttg\.([a-z_]+)`)

// TestSchemaFingerprintMatches is the drift gate for the sqlc input. It compares
// the committed fingerprint with a freshly migrated database, so a column type,
// nullability, default, constraint or index change that was not accompanied by a
// regenerated snapshot fails the build.
func TestSchemaFingerprintMatches(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	raw, err := os.ReadFile("../../db/schema.fingerprint.json")
	if err != nil {
		t.Fatalf("read committed fingerprint: %v (run tooling/schema-dump.ps1|sh)", err)
	}
	var want schema.Fingerprint
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("parse committed fingerprint: %v", err)
	}
	if want.Version != schema.FingerprintVersion {
		t.Fatalf("fingerprint version %d, code expects %d", want.Version, schema.FingerprintVersion)
	}

	got, err := schema.Read(ctx, db)
	if err != nil {
		t.Fatalf("read live schema: %v", err)
	}
	if len(got.Tables) == 0 {
		t.Fatal("live schema has no tables; migrations are not applied")
	}

	if diffs := schema.Diff(want, got); len(diffs) > 0 {
		t.Fatalf("schema drift detected — the committed snapshot is stale.\n"+
			"Run tooling/migrate-up.ps1 then tooling/schema-dump.ps1 and regenerate sqlc:\n  %s",
			strings.Join(diffs, "\n  "))
	}
}

// TestSchemaSnapshotHasEveryTable cross-checks the sqlc snapshot against the
// live database independently of the fingerprint.
func TestSchemaSnapshotHasEveryTable(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	raw, err := os.ReadFile("../../db/schema.sql")
	if err != nil {
		t.Fatalf("read schema snapshot: %v", err)
	}
	declared := map[string]bool{}
	for _, m := range schemaTablePattern.FindAllStringSubmatch(string(raw), -1) {
		declared[m[1]] = true
	}

	rows, err := db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'ttg' AND table_name <> 'goose_db_version'`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer func() { _ = rows.Close() }()
	actual := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		actual[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	for name := range declared {
		if !actual[name] {
			t.Errorf("schema.sql declares ttg.%s but the database does not have it", name)
		}
	}
	for name := range actual {
		if !declared[name] {
			t.Errorf("migrated table ttg.%s is missing from schema.sql", name)
		}
	}
}
