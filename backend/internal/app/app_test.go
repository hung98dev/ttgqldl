package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestApp(t *testing.T, checks []Check, epoch func() int64) *App {
	t.Helper()
	a, err := New(Options{
		Logger: discardLogger(),
		Config: Config{
			HTTPAddr:         "127.0.0.1:0",
			ReadinessTimeout: 200 * time.Millisecond,
			ShutdownTimeout:  2 * time.Second,
		},
		Checks:      checks,
		WriterEpoch: epoch,
		Version:     "test",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func doRequest(t *testing.T, a *App, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestNewValidatesOptions(t *testing.T) {
	cases := map[string]Options{
		"nil logger":     {Config: Config{HTTPAddr: "127.0.0.1:0", ReadinessTimeout: time.Second, ShutdownTimeout: time.Second}},
		"empty addr":     {Logger: discardLogger(), Config: Config{ReadinessTimeout: time.Second, ShutdownTimeout: time.Second}},
		"zero readiness": {Logger: discardLogger(), Config: Config{HTTPAddr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		"zero shutdown":  {Logger: discardLogger(), Config: Config{HTTPAddr: "127.0.0.1:0", ReadinessTimeout: time.Second}},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(opts); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestHealthzReportsAliveWithoutTouchingDependencies(t *testing.T) {
	var called atomic.Int32
	a := newTestApp(t, []Check{{
		Name: "postgres",
		Run: func(context.Context) error {
			called.Add(1)
			return errors.New("database is down")
		},
	}}, nil)

	rec := doRequest(t, a, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	body := decodeBody[map[string]any](t, rec)
	if body["status"] != "alive" {
		t.Errorf("body = %v", body)
	}
	if called.Load() != 0 {
		t.Error("liveness probe must not execute readiness checks")
	}
	if id := rec.Header().Get("X-Request-Id"); len(id) != 36 {
		t.Errorf("X-Request-Id = %q, want a UUID", id)
	}
}

func TestReadyzReadyReportsWriterEpoch(t *testing.T) {
	a := newTestApp(t, nil, func() int64 { return 42 })
	rec := doRequest(t, a, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	got := decodeBody[readyResponse](t, rec)
	if got.Status != "ready" || got.WriterEpoch != 42 {
		t.Fatalf("body = %+v", got)
	}
}

func TestReadyzDefaultsWriterEpochToZero(t *testing.T) {
	a := newTestApp(t, nil, nil)
	rec := doRequest(t, a, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeBody[readyResponse](t, rec)
	if got.WriterEpoch != 0 || got.Status != "ready" {
		t.Fatalf("body = %+v", got)
	}
}

func TestReadyzFailsWhenDependencyIsUnavailable(t *testing.T) {
	a := newTestApp(t, []Check{{Name: "postgres", Run: func(context.Context) error {
		return errors.New("connection refused")
	}}}, nil)

	rec := doRequest(t, a, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := decodeBody[errorEnvelope](t, rec)
	if body.Error.Code != "UNAVAILABLE" {
		t.Errorf("code = %q", body.Error.Code)
	}
	if !body.Error.Retryable {
		t.Error("readiness failure must be retryable")
	}
	if !strings.Contains(body.Error.Message, "postgres") {
		t.Errorf("message = %q, want the failing dependency named", body.Error.Message)
	}
	if len(body.Error.RequestID) != 36 {
		t.Errorf("request_id = %q, want a UUID", body.Error.RequestID)
	}
	if body.Error.RequestID != rec.Header().Get("X-Request-Id") {
		t.Error("request_id must match the X-Request-Id header")
	}
}

func TestReadyzHonoursReadinessTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	a, err := New(Options{
		Logger: discardLogger(),
		Config: Config{
			HTTPAddr:         "127.0.0.1:0",
			ReadinessTimeout: 25 * time.Millisecond,
			ShutdownTimeout:  time.Second,
		},
		Checks: []Check{{Name: "postgres", Run: func(ctx context.Context) error {
			select {
			case <-release:
				return nil
			case <-time.After(5 * time.Second):
				return nil
			}
		}}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	started := time.Now()
	rec := doRequest(t, a, http.MethodGet, "/readyz", nil)
	elapsed := time.Since(started)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if elapsed > time.Second {
		t.Fatalf("readiness probe blocked for %s, must honour the timeout", elapsed)
	}
	body := decodeBody[errorEnvelope](t, rec)
	if !strings.Contains(body.Error.Message, "timeout") {
		t.Errorf("message = %q, want a timeout report", body.Error.Message)
	}
}

func TestReadyzRecoversPanickingCheck(t *testing.T) {
	a := newTestApp(t, []Check{{Name: "postgres", Run: func(context.Context) error {
		panic("driver exploded")
	}}}, nil)
	rec := doRequest(t, a, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := decodeBody[errorEnvelope](t, rec)
	if !strings.Contains(body.Error.Message, "postgres") {
		t.Errorf("message = %q", body.Error.Message)
	}
}

func TestUnknownRouteAndWrongMethod(t *testing.T) {
	a := newTestApp(t, nil, nil)

	rec := doRequest(t, a, http.MethodGet, "/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", rec.Code)
	}
	if body := decodeBody[errorEnvelope](t, rec); body.Error.Code != "NOT_FOUND" {
		t.Errorf("code = %q", body.Error.Code)
	}

	rec = doRequest(t, a, http.MethodPost, "/healthz", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Errorf("Allow = %q", allow)
	}
	if body := decodeBody[errorEnvelope](t, rec); body.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("code = %q", body.Error.Code)
	}

	rec = doRequest(t, a, http.MethodHead, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", rec.Code)
	}
}

func TestPanicIsRecoveredAsInternalError(t *testing.T) {
	a := newTestApp(t, nil, nil)
	handler := a.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := decodeBody[errorEnvelope](t, rec)
	if body.Error.Code != "INTERNAL" || body.Error.Retryable {
		t.Errorf("body = %+v", body.Error)
	}
}

func TestRequestIDPropagation(t *testing.T) {
	a := newTestApp(t, nil, nil)

	rec := doRequest(t, a, http.MethodGet, "/healthz", map[string]string{"X-Request-Id": "abc-123"})
	if got := rec.Header().Get("X-Request-Id"); got != "abc-123" {
		t.Fatalf("X-Request-Id = %q, want the supplied value", got)
	}

	long := strings.Repeat("a", maxRequestIDLength+1)
	rec = doRequest(t, a, http.MethodGet, "/healthz", map[string]string{"X-Request-Id": long})
	if got := rec.Header().Get("X-Request-Id"); got == long || len(got) != 36 {
		t.Fatalf("oversized request id was not replaced: %q", got)
	}

	rec = doRequest(t, a, http.MethodGet, "/healthz", map[string]string{"X-Request-Id": "bad\nid"})
	if got := rec.Header().Get("X-Request-Id"); got == "bad\nid" {
		t.Fatal("control characters were reflected into the response header")
	}
}

func waitForHealth(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become healthy", url)
}

func TestServeShutsDownGracefullyOnContextCancel(t *testing.T) {
	a := newTestApp(t, nil, nil)
	ln, err := a.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	baseURL := "http://" + ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx, ln) }()

	waitForHealth(t, baseURL)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil after a requested shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after context cancellation")
	}

	client := &http.Client{Timeout: 300 * time.Millisecond}
	if resp, err := client.Get(baseURL + "/healthz"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("server still accepts connections after shutdown")
	}
}

func TestServeReturnsImmediatelyWhenContextAlreadyCancelled(t *testing.T) {
	a := newTestApp(t, nil, nil)
	ln, err := a.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx, ln) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return for an already cancelled context")
	}
}

func TestListenReportsBindFailure(t *testing.T) {
	a := newTestApp(t, nil, nil)
	ln, err := a.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	// A second listener on the same concrete address must fail.
	second, err := New(Options{
		Logger: discardLogger(),
		Config: Config{
			HTTPAddr:         ln.Addr().String(),
			ReadinessTimeout: time.Second,
			ShutdownTimeout:  time.Second,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := second.Listen(); err == nil {
		t.Fatal("expected a bind error on an address already in use")
	}
}
