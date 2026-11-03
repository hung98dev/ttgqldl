// Package integration holds tests that exercise real dependencies.
//
// Tests that need PostgreSQL read TTG_TEST_DATABASE_URL and skip when it is not
// set, so `go test ./...` stays runnable on a workstation with no database.
// Tests that only need an unreachable server always run.
package integration

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/app"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

const unreachableDSN = "postgres://app:pw@127.0.0.1:1/ttgqldl?sslmode=disable"

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newApp(t *testing.T, pool *storage.Pool) *app.App {
	t.Helper()
	a, err := app.New(app.Options{
		Logger: discardLogger(),
		Config: app.Config{
			HTTPAddr:         "127.0.0.1:0",
			ReadinessTimeout: 2 * time.Second,
			ShutdownTimeout:  2 * time.Second,
		},
		Checks: []app.Check{{Name: "postgres", Run: pool.Ping}},
	})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a
}

func get(t *testing.T, a *app.App, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)
	return rec
}

// TestReadyzFailsWhenPostgreSQLIsUnavailable needs no database: it proves that a
// server with an unreachable dependency stays live but reports not-ready.
func TestReadyzFailsWhenPostgreSQLIsUnavailable(t *testing.T) {
	pool, err := storage.Open(context.Background(), storage.Options{
		DSN:            unreachableDSN,
		ConnectTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer pool.Close()

	a := newApp(t, pool)

	if rec := get(t, a, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rec.Code)
	}
	rec := get(t, a, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz = %d, want 503 when PostgreSQL is unreachable", rec.Code)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "UNAVAILABLE" || !strings.Contains(body.Error.Message, "postgres") {
		t.Fatalf("body = %+v", body)
	}
}

// TestPostgreSQLIntegration exercises the real pinned server. It is skipped
// unless TTG_TEST_DATABASE_URL points at a running instance.
func TestPostgreSQLIntegration(t *testing.T) {
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TTG_TEST_DATABASE_URL is not set; start the development database with tooling/dev-up")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := storage.Open(ctx, storage.Options{
		DSN:            dsn,
		MaxConns:       4,
		ConnectTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	version, err := pool.ServerVersion(ctx)
	if err != nil {
		t.Fatalf("server version: %v", err)
	}
	if !strings.HasPrefix(version, "18.") {
		t.Fatalf("server_version = %q, want the pinned PostgreSQL 18 line", version)
	}
	t.Logf("connected to PostgreSQL server_version=%s", version)

	a := newApp(t, pool)
	rec := get(t, a, "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	t.Logf("readyz body: %s", strings.TrimSpace(rec.Body.String()))

	// After the pool is closed the same server must report not-ready.
	pool.Close()
	rec = get(t, a, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after close = %d, want 503", rec.Code)
	}
}
