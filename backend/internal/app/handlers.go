package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// maxRequestIDLength bounds an inbound correlation identifier.
const maxRequestIDLength = 128

// healthResponse is the documented body of GET /healthz.
type healthResponse struct {
	Status string `json:"status"`
}

// readyResponse is the documented body of a successful GET /readyz.
type readyResponse struct {
	Status      string `json:"status"`
	WriterEpoch int64  `json:"writer_epoch"`
}

// errorEnvelope follows docs/plan/05_API_WEBSOCKET.md: every non-2xx response
// carries {"error":{"code","message","request_id","retryable"}}.
type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Retryable bool   `json:"retryable"`
}

// handleHealthz reports process liveness. It deliberately performs no
// dependency I/O: a liveness probe must not fail because PostgreSQL is down.
func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, healthResponse{Status: "alive"})
}

// handleReadyz reports whether the process can serve authoritative traffic.
// Any failing dependency yields 503 with the standard error envelope.
func (a *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if failed := a.runChecks(r.Context()); len(failed) > 0 {
		a.writeError(w, r, http.StatusServiceUnavailable, "UNAVAILABLE",
			"readiness failed: "+strings.Join(failed, ","), true)
		return
	}
	var epoch int64
	if a.writerEpoch != nil {
		epoch = a.writerEpoch()
	}
	a.writeJSON(w, http.StatusOK, readyResponse{Status: "ready", WriterEpoch: epoch})
}

// handleFallback answers unknown paths with 404 and known paths addressed with
// an unsupported method with 405 plus an Allow header.
func (a *App) handleFallback(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/healthz", "/readyz":
		w.Header().Set("Allow", "GET, HEAD")
		a.writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"method not allowed for "+r.URL.Path, false)
	default:
		a.writeError(w, r, http.StatusNotFound, "NOT_FOUND", "no route for "+r.URL.Path, false)
	}
}

// runChecks evaluates every readiness dependency under one bounded deadline.
//
// Checks run concurrently so that one slow dependency cannot hide another, and
// the whole evaluation is capped by ReadinessTimeout even if a check ignores
// its context: the caller must never hang a probe.
func (a *App) runChecks(parent context.Context) []string {
	if len(a.checks) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, a.cfg.ReadinessTimeout)
	defer cancel()

	type outcome struct {
		name string
		err  error
	}
	results := make(chan outcome, len(a.checks))
	for _, check := range a.checks {
		go func(c Check) {
			defer func() {
				if rec := recover(); rec != nil {
					results <- outcome{name: c.Name, err: errPanic(rec)}
				}
			}()
			results <- outcome{name: c.Name, err: c.Run(ctx)}
		}(check)
	}

	var failed []string
	for range a.checks {
		select {
		case res := <-results:
			if res.err != nil {
				failed = append(failed, res.name)
				a.logger.Warn("readiness check failed",
					slog.String("check", res.name),
					slog.String("error", res.err.Error()),
				)
			}
		case <-ctx.Done():
			// Report the timeout once, naming the dependencies still pending.
			failed = append(failed, "timeout")
			return failed
		}
	}
	return failed
}

func (a *App) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		a.logger.Error("write response", slog.String("error", err.Error()))
	}
}

func (a *App) writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, retryable bool) {
	a.writeJSON(w, status, errorEnvelope{Error: errorDetail{
		Code:      code,
		Message:   message,
		RequestID: requestIDFrom(r.Context()),
		Retryable: retryable,
	}})
}

// requestIDLength and requestIDPattern keep an inbound correlation id bounded
// and printable before it is echoed into logs and responses.
func sanitizeRequestID(raw string) string {
	if raw == "" || len(raw) > maxRequestIDLength {
		return ""
	}
	for _, r := range raw {
		if r < 0x20 || r > 0x7e {
			return ""
		}
	}
	return raw
}

// now is a seam for deterministic timing in tests.
var now = time.Now
