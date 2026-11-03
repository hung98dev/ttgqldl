package integration

import (
	"context"
	"regexp"
	"testing"
)

// TestRolesAreLeastPrivileged asserts the role model that the production
// deployment must reproduce: the schema owner is not a superuser, the runtime
// role owns nothing, and only the owner may create schema objects.
func TestRolesAreLeastPrivileged(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	var provisioned int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM pg_roles WHERE rolname IN ('ttg_owner', 'ttg_app')`).Scan(&provisioned); err != nil {
		t.Fatalf("query roles: %v", err)
	}
	if provisioned != 2 {
		t.Skip("development roles ttg_owner/ttg_app are not provisioned in this database")
	}

	for _, role := range []string{"ttg_owner", "ttg_app"} {
		var super, createDB, createRole bool
		if err := db.QueryRowContext(ctx,
			`SELECT rolsuper, rolcreatedb, rolcreaterole FROM pg_roles WHERE rolname = $1`, role).
			Scan(&super, &createDB, &createRole); err != nil {
			t.Fatalf("role %s: %v", role, err)
		}
		if super {
			t.Errorf("%s must not be a superuser", role)
		}
		if createDB {
			t.Errorf("%s must not have CREATEDB", role)
		}
		if createRole {
			t.Errorf("%s must not have CREATEROLE", role)
		}
	}

	var appCanCreate, ownerCanCreate bool
	if err := db.QueryRowContext(ctx,
		`SELECT has_database_privilege('ttg_app', current_database(), 'CREATE')`).Scan(&appCanCreate); err != nil {
		t.Fatalf("app database privilege: %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT has_database_privilege('ttg_owner', current_database(), 'CREATE')`).Scan(&ownerCanCreate); err != nil {
		t.Fatalf("owner database privilege: %v", err)
	}
	if appCanCreate {
		t.Error("ttg_app must not have CREATE on the database")
	}
	if !ownerCanCreate {
		t.Error("ttg_owner must have CREATE on the database so migrations can create schema objects")
	}

	// The application role may read and write rows, never define them.
	var appCanCreateInSchema bool
	if err := db.QueryRowContext(ctx,
		`SELECT has_schema_privilege('ttg_app', 'ttg', 'CREATE')`).Scan(&appCanCreateInSchema); err != nil {
		t.Fatalf("app schema privilege: %v", err)
	}
	if appCanCreateInSchema {
		t.Error("ttg_app must not have CREATE on schema ttg")
	}
}

// TestAppRoleCannotRunDDL proves the prohibition empirically. It only runs when
// the test connection actually is the least-privileged application role.
func TestAppRoleCannotRunDDL(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	var currentUser string
	if err := db.QueryRowContext(ctx, `SELECT current_user`).Scan(&currentUser); err != nil {
		t.Fatalf("current_user: %v", err)
	}
	if currentUser != "ttg_app" {
		t.Skipf("connected as %q; DDL-denial assertions require the ttg_app role", currentUser)
	}

	denied := regexp.MustCompile(`permission denied|must be owner|not have permission`)
	cases := []struct {
		name string
		sql  string
	}{
		{"create table in ttg", `CREATE TABLE ttg.forbidden_ddl (id INT)`},
		{"create table in public", `CREATE TABLE public.forbidden_ddl (id INT)`},
		{"create schema", `CREATE SCHEMA forbidden_schema`},
		{"alter table", `ALTER TABLE ttg.villages ADD COLUMN forbidden_column INT`},
		{"drop table", `DROP TABLE ttg.villages`},
		{"truncate table", `TRUNCATE TABLE ttg.villages`},
		{"create role", `CREATE ROLE forbidden_role`},
		{"change table owner", `ALTER TABLE ttg.villages OWNER TO ttg_app`},
		{"create index on an owned table", `CREATE INDEX forbidden_idx ON ttg.villages (id)`},
		{"create extension", `CREATE EXTENSION IF NOT EXISTS pgcrypto`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.ExecContext(ctx, tc.sql)
			if err == nil {
				t.Fatalf("%s: the application role executed privileged DDL", tc.name)
			}
			if !denied.MatchString(err.Error()) {
				t.Fatalf("%s: rejected, but with an unexpected error: %v", tc.name, err)
			}
		})
	}

	// GRANT by a non-owner is a warning-only no-op rather than an error, so assert
	// the effect instead: PUBLIC must not gain access to an authoritative table.
	t.Run("grant has no effect", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, `GRANT ALL ON ttg.villages TO PUBLIC`); err != nil {
			return // refusing outright is also acceptable
		}
		var publicSelect bool
		if err := db.QueryRowContext(ctx,
			`SELECT has_table_privilege('public', 'ttg.villages', 'SELECT')`).Scan(&publicSelect); err != nil {
			t.Fatalf("privilege probe: %v", err)
		}
		if publicSelect {
			t.Fatal("PUBLIC gained SELECT through a GRANT issued by a non-owner")
		}
	})

	// The rejected statements must not have created anything.
	var leftovers int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema IN ('ttg', 'public') AND table_name = 'forbidden_ddl'`).Scan(&leftovers); err != nil {
		t.Fatalf("leftover check: %v", err)
	}
	if leftovers != 0 {
		t.Fatal("a forbidden table exists after the DDL denial test")
	}
}

// TestAppRoleCanUseRuntimeTable confirms the denial is targeted: the application
// role still reads and writes rows in the migrated tables.
func TestAppRoleCanUseRuntimeTable(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var villageID = newUUID(t)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`,
		villageID, repeatKey("r")); err != nil {
		t.Fatalf("application role cannot insert a village: %v", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM ttg.villages WHERE id = $1`, villageID).Scan(&count); err != nil {
		t.Fatalf("application role cannot select: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the row back, got %d", count)
	}
}

func repeatKey(fill string) string {
	out := make([]byte, 64)
	for i := range out {
		out[i] = fill[0]
	}
	return string(out)
}
