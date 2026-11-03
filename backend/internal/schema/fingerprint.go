// Package schema produces a deterministic fingerprint of the authoritative
// PostgreSQL schema.
//
// The fingerprint is the drift contract for the sqlc input: it records every
// table, column (type, nullability, default, length/precision), constraint and
// index in schema `ttg`. Regenerating it after a migration and comparing it in
// CI is what turns "schema.sql looks right" into a verifiable statement.
//
// It deliberately ignores goose's bookkeeping table so that a goose upgrade does
// not churn the fingerprint.
package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// SchemaName is the owned schema from docs/plan/04_DATABASE.md.
const SchemaName = "ttg"

// MetadataTable is tooling bookkeeping and is excluded from the fingerprint.
const MetadataTable = "goose_db_version"

// FingerprintVersion increments when the fingerprint itself changes shape.
const FingerprintVersion = 1

// Fingerprint is the comparable schema description.
type Fingerprint struct {
	Version     int          `json:"version"`
	Schema      string       `json:"schema"`
	Tables      []Table      `json:"tables"`
	Constraints []Constraint `json:"constraints"`
	Indexes     []Index      `json:"indexes"`
}

// Table lists one relation and its ordered columns.
type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Column captures everything that changes generated code or behaviour.
type Column struct {
	Name             string  `json:"name"`
	Ordinal          int     `json:"ordinal"`
	DataType         string  `json:"data_type"`
	UDTName          string  `json:"udt_name"`
	Nullable         bool    `json:"nullable"`
	Default          *string `json:"default"`
	MaxLength        *int64  `json:"max_length,omitempty"`
	NumericPrecision *int64  `json:"numeric_precision,omitempty"`
	NumericScale     *int64  `json:"numeric_scale,omitempty"`
}

// Constraint is a named constraint with its rendered definition.
type Constraint struct {
	Table      string `json:"table"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}

// Index is a named index with its rendered definition.
type Index struct {
	Table      string `json:"table"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Read queries the live catalog. It never writes.
func Read(ctx context.Context, db *sql.DB) (Fingerprint, error) {
	fp := Fingerprint{Version: FingerprintVersion, Schema: SchemaName}

	tables, err := readTables(ctx, db)
	if err != nil {
		return Fingerprint{}, err
	}
	fp.Tables = tables

	constraints, err := readConstraints(ctx, db)
	if err != nil {
		return Fingerprint{}, err
	}
	fp.Constraints = constraints

	indexes, err := readIndexes(ctx, db)
	if err != nil {
		return Fingerprint{}, err
	}
	fp.Indexes = indexes
	return fp, nil
}

func readTables(ctx context.Context, db *sql.DB) ([]Table, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE' AND table_name <> $2
		ORDER BY table_name`, SchemaName, MetadataTable)
	if err != nil {
		return nil, fmt.Errorf("schema: list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("schema: scan table: %w", err)
		}
		tables = append(tables, Table{Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema: tables rows: %w", err)
	}
	for i := range tables {
		columns, err := readColumns(ctx, db, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Columns = columns
	}
	return tables, nil
}

func readColumns(ctx context.Context, db *sql.DB, table string) ([]Column, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, ordinal_position, data_type, udt_name, is_nullable,
		       column_default, character_maximum_length, numeric_precision, numeric_scale
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`, SchemaName, table)
	if err != nil {
		return nil, fmt.Errorf("schema: columns for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	var columns []Column
	for rows.Next() {
		var (
			column    Column
			nullable  string
			def       sql.NullString
			maxLen    sql.NullInt64
			precision sql.NullInt64
			scale     sql.NullInt64
		)
		if err := rows.Scan(&column.Name, &column.Ordinal, &column.DataType, &column.UDTName,
			&nullable, &def, &maxLen, &precision, &scale); err != nil {
			return nil, fmt.Errorf("schema: scan column of %s: %w", table, err)
		}
		column.Nullable = nullable == "YES"
		if def.Valid {
			value := def.String
			column.Default = &value
		}
		if maxLen.Valid {
			value := maxLen.Int64
			column.MaxLength = &value
		}
		if precision.Valid {
			value := precision.Int64
			column.NumericPrecision = &value
		}
		if scale.Valid {
			value := scale.Int64
			column.NumericScale = &value
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema: columns rows: %w", err)
	}
	return columns, nil
}

func readConstraints(ctx context.Context, db *sql.DB) ([]Constraint, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.relname, con.conname, con.contype::text, pg_get_constraintdef(con.oid)
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname <> $2
		ORDER BY c.relname, con.conname`, SchemaName, MetadataTable)
	if err != nil {
		return nil, fmt.Errorf("schema: constraints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var constraints []Constraint
	for rows.Next() {
		var con Constraint
		if err := rows.Scan(&con.Table, &con.Name, &con.Type, &con.Definition); err != nil {
			return nil, fmt.Errorf("schema: scan constraint: %w", err)
		}
		constraints = append(constraints, con)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema: constraints rows: %w", err)
	}
	return constraints, nil
}

func readIndexes(ctx context.Context, db *sql.DB) ([]Index, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT tablename, indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename <> $2
		ORDER BY tablename, indexname`, SchemaName, MetadataTable)
	if err != nil {
		return nil, fmt.Errorf("schema: indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var indexes []Index
	for rows.Next() {
		var idx Index
		if err := rows.Scan(&idx.Table, &idx.Name, &idx.Definition); err != nil {
			return nil, fmt.Errorf("schema: scan index: %w", err)
		}
		indexes = append(indexes, idx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema: indexes rows: %w", err)
	}
	return indexes, nil
}

// Marshal renders the fingerprint deterministically (sorted already by the
// queries, two-space JSON with a trailing newline).
func Marshal(fp Fingerprint) ([]byte, error) {
	if fp.Tables == nil {
		fp.Tables = []Table{}
	}
	if fp.Constraints == nil {
		fp.Constraints = []Constraint{}
	}
	if fp.Indexes == nil {
		fp.Indexes = []Index{}
	}
	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("schema: marshal: %w", err)
	}
	return append(data, '\n'), nil
}

// Diff reports human-readable differences between the committed fingerprint
// (want) and the live database (got). It is intentionally descriptive rather
// than exhaustive: the fix is always to regenerate the snapshot.
func Diff(want, got Fingerprint) []string {
	var out []string
	if len(want.Tables) != len(got.Tables) {
		out = append(out, fmt.Sprintf("table count: committed %d, live %d", len(want.Tables), len(got.Tables)))
	}
	wantTables := map[string]Table{}
	for _, t := range want.Tables {
		wantTables[t.Name] = t
	}
	gotTables := map[string]Table{}
	for _, t := range got.Tables {
		gotTables[t.Name] = t
	}
	names := make([]string, 0, len(wantTables)+len(gotTables))
	for name := range wantTables {
		names = append(names, name)
	}
	for name := range gotTables {
		if _, ok := wantTables[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		wantTable, inWant := wantTables[name]
		gotTable, inGot := gotTables[name]
		switch {
		case !inWant:
			out = append(out, "table added in the database: "+name)
		case !inGot:
			out = append(out, "table missing from the database: "+name)
		default:
			out = append(out, diffColumns(name, wantTable, gotTable)...)
		}
	}
	out = append(out, diffNamed(want.Constraints, got.Constraints, "constraint",
		func(c Constraint) string { return c.Table + "." + c.Name },
		func(c Constraint) string { return c.Definition })...)
	out = append(out, diffNamed(want.Indexes, got.Indexes, "index",
		func(i Index) string { return i.Table + "." + i.Name },
		func(i Index) string { return i.Definition })...)
	return out
}

func diffColumns(table string, want, got Table) []string {
	var out []string
	wantCols := map[string]Column{}
	for _, c := range want.Columns {
		wantCols[c.Name] = c
	}
	gotCols := map[string]Column{}
	for _, c := range got.Columns {
		gotCols[c.Name] = c
	}
	for name, wc := range wantCols {
		gc, ok := gotCols[name]
		if !ok {
			out = append(out, fmt.Sprintf("%s: column removed: %s", table, name))
			continue
		}
		if wc.DataType != gc.DataType || wc.UDTName != gc.UDTName {
			out = append(out, fmt.Sprintf("%s.%s: type %s(%s) -> %s(%s)",
				table, name, wc.DataType, wc.UDTName, gc.DataType, gc.UDTName))
		}
		if wc.Nullable != gc.Nullable {
			out = append(out, fmt.Sprintf("%s.%s: nullable %t -> %t", table, name, wc.Nullable, gc.Nullable))
		}
		if deref(wc.Default) != deref(gc.Default) {
			out = append(out, fmt.Sprintf("%s.%s: default %q -> %q", table, name, deref(wc.Default), deref(gc.Default)))
		}
		if derefInt(wc.MaxLength) != derefInt(gc.MaxLength) {
			out = append(out, fmt.Sprintf("%s.%s: length %d -> %d", table, name, derefInt(wc.MaxLength), derefInt(gc.MaxLength)))
		}
	}
	for name := range gotCols {
		if _, ok := wantCols[name]; !ok {
			out = append(out, fmt.Sprintf("%s: column added: %s", table, name))
		}
	}
	return out
}

func diffNamed[T any](want, got []T, kind string, key func(T) string, shape func(T) string) []string {
	wantMap := map[string]string{}
	for _, item := range want {
		wantMap[key(item)] = shape(item)
	}
	gotMap := map[string]string{}
	for _, item := range got {
		gotMap[key(item)] = shape(item)
	}
	var out []string
	for k, wantDef := range wantMap {
		gotDef, ok := gotMap[k]
		if !ok {
			out = append(out, fmt.Sprintf("%s missing from the database: %s", kind, k))
			continue
		}
		if wantDef != gotDef {
			out = append(out, fmt.Sprintf("%s changed: %s\n  committed: %s\n  live:      %s", kind, k, wantDef, gotDef))
		}
	}
	for k := range gotMap {
		if _, ok := wantMap[k]; !ok {
			out = append(out, fmt.Sprintf("%s added in the database: %s", kind, k))
		}
	}
	sort.Strings(out)
	return out
}

func deref(s *string) string {
	if s == nil {
		return "<null>"
	}
	return *s
}

func derefInt(v *int64) int64 {
	if v == nil {
		return -1
	}
	return *v
}

// WithSearchPath returns dsn with the connection runtime parameter
// `search_path` pinned to schemaName, supporting both URL and keyword DSN forms.
// It lets tooling select the owned schema without relying on a role default.
func WithSearchPath(dsn, schemaName string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		query := parsed.Query()
		query.Set("search_path", schemaName)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	if strings.Contains(dsn, "search_path=") {
		return dsn
	}
	return dsn + " search_path=" + schemaName
}
