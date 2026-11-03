package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOpenRejectsEmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), Options{DSN: "   "})
	if !errors.Is(err, ErrEmptyDSN) {
		t.Fatalf("err = %v, want ErrEmptyDSN", err)
	}
}

func TestOpenRejectsMalformedDSN(t *testing.T) {
	_, err := Open(context.Background(), Options{DSN: "this is not a connection string"})
	if err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestOpenIsLazyAndPingReportsUnreachableServer(t *testing.T) {
	// Port 1 is never a PostgreSQL server; Open must still succeed because the
	// pool connects lazily, and Ping must report the failure.
	pool, err := Open(context.Background(), Options{
		DSN:            "postgres://app:pw@127.0.0.1:1/ttgqldl?sslmode=disable",
		ConnectTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Open must not contact the server: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err == nil {
		t.Fatal("Ping reported success against a closed port")
	}
	if _, err := pool.ServerVersion(ctx); err == nil {
		t.Fatal("ServerVersion reported success against a closed port")
	}
}

func TestNilPoolIsSafe(t *testing.T) {
	var pool *Pool
	pool.Close()
	if err := pool.Ping(context.Background()); err == nil {
		t.Fatal("nil pool Ping must fail")
	}
	if _, err := pool.ServerVersion(context.Background()); err == nil {
		t.Fatal("nil pool ServerVersion must fail")
	}
	if stats := pool.Stats(); len(stats) != 0 {
		t.Fatalf("nil pool stats = %v", stats)
	}
}

func TestStatsOnOpenPool(t *testing.T) {
	pool, err := Open(context.Background(), Options{DSN: "postgres://app:pw@127.0.0.1:1/ttgqldl?sslmode=disable"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()
	stats := pool.Stats()
	if stats["max_conns"] <= 0 {
		t.Fatalf("stats = %v", stats)
	}
}
