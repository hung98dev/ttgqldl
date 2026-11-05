// Package storage owns PostgreSQL access for the backend.
//
// Only this package may construct a connection pool. Domain packages receive
// narrow interfaces or plain values, never a *pgxpool.Pool, so database
// infrastructure cannot leak into gameplay logic.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// Options configures the PostgreSQL pool.
type Options struct {
	// DSN is the postgres:// connection string from TTG_DATABASE_URL.
	DSN string
	// MaxConns bounds the pool. Zero leaves the pgx default in place.
	MaxConns int32
	// ConnectTimeout bounds a single connection attempt.
	ConnectTimeout time.Duration
	// MaxConnLifetime and MaxConnIdleTime recycle connections.
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Pool wraps a pgx connection pool. The pool connects lazily, so a server can
// start while PostgreSQL is down and report the truth through readiness.
type Pool struct {
	pool *pgxpool.Pool
}

// ErrEmptyDSN is returned when no connection string is configured.
var ErrEmptyDSN = errors.New("postgres: empty DSN")

// Open parses the DSN and creates a lazily connecting pool. It does not contact
// the server; use Ping (directly or through a readiness check) for that.
func Open(ctx context.Context, opts Options) (*Pool, error) {
	if strings.TrimSpace(opts.DSN) == "" {
		return nil, ErrEmptyDSN
	}
	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse DSN: %w", err)
	}
	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}
	if opts.ConnectTimeout > 0 {
		cfg.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}
	return &Pool{pool: pool}, nil
}

// OpenSQL returns a database/sql handle backed by the pgx driver, for tooling
// that cannot use pgxpool (goose migrations). The caller must Close it.
func OpenSQL(dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, ErrEmptyDSN
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open sql handle: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// Ping verifies that a connection can be acquired and used right now.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return errors.New("postgres: pool is not initialised")
	}
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: ping: %w", err)
	}
	return nil
}

// ServerVersion returns the PostgreSQL server version string.
func (p *Pool) ServerVersion(ctx context.Context) (string, error) {
	if p == nil || p.pool == nil {
		return "", errors.New("postgres: pool is not initialised")
	}
	var version string
	if err := p.pool.QueryRow(ctx, "show server_version").Scan(&version); err != nil {
		return "", fmt.Errorf("postgres: server version: %w", err)
	}
	return version, nil
}

// Begin starts one PostgreSQL transaction for the durable ingestion boundary. Domain
// packages never receive the pool; ingestion owns the narrow transactional usage.
func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	if p == nil || p.pool == nil {
		return nil, errors.New("postgres: pool is not initialised")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: begin: %w", err)
	}
	return tx, nil
}

// Close releases every pooled connection. It is safe to call on a nil pool.
func (p *Pool) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

// Stats exposes pool counters for operator telemetry.
func (p *Pool) Stats() map[string]int64 {
	if p == nil || p.pool == nil {
		return map[string]int64{}
	}
	s := p.pool.Stat()
	return map[string]int64{
		"acquire_count":     s.AcquireCount(),
		"acquired_conns":    int64(s.AcquiredConns()),
		"idle_conns":        int64(s.IdleConns()),
		"total_conns":       int64(s.TotalConns()),
		"max_conns":         int64(s.MaxConns()),
		"empty_acquire_cnt": s.EmptyAcquireCount(),
	}
}
