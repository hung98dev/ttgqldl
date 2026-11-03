package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
)

type requestIDKey struct{}

// requestIDFrom returns the correlation id stored on the request context.
func requestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// newRequestID returns a random UUIDv4 in canonical form.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// wrap applies the middleware chain: request id, access log, panic recovery.
func (a *App) wrap(next http.Handler) http.Handler {
	return a.withRequestID(a.accessLog(a.recoverer(next)))
}

// withRequestID reuses a well-formed inbound X-Request-Id and otherwise
// generates one, echoing it on the response and storing it on the context.
func (a *App) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get("X-Request-Id"))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// accessLog emits one structured entry per request.
func (a *App) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		a.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int64("bytes", rec.written),
			slog.Duration("duration", now().Sub(started)),
			slog.String("request_id", requestIDFrom(r.Context())),
		)
	})
}

// recoverer converts a handler panic into a 500 error envelope and keeps the
// process serving other requests.
func (a *App) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				a.logger.Error("panic recovered",
					slog.String("error", fmt.Sprint(rec)),
					slog.String("path", r.URL.Path),
					slog.String("request_id", requestIDFrom(r.Context())),
				)
				a.writeError(w, r, http.StatusInternalServerError, "INTERNAL", "internal error", false)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the status code and byte count for access logging.
// Unwrap keeps http.ResponseController (and therefore future streaming and
// WebSocket upgrade support) working through the wrapper.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written int64
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(p)
	s.written += int64(n)
	return n, err
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// errPanic converts a recovered value into an error.
func errPanic(v any) error {
	if err, ok := v.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", v)
}
