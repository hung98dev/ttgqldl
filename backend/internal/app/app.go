// Package app wires the HTTP surface of the backend: the server lifecycle,
// middleware, and the readiness contract.
//
// The package depends only on narrow function values (readiness checks, writer
// epoch) so that storage, simulation and future operator code can be injected
// without importing this package's consumers.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Check is one readiness dependency. Run must return quickly and must honour
// the context deadline it is given.
type Check struct {
	Name string
	Run  func(context.Context) error
}

// Config holds the HTTP-layer settings derived from the validated
// configuration.
type Config struct {
	// HTTPAddr is the host:port to bind.
	HTTPAddr string
	// ReadinessTimeout bounds the whole readiness evaluation.
	ReadinessTimeout time.Duration
	// ShutdownTimeout bounds graceful shutdown.
	ShutdownTimeout time.Duration
}

// Options are the injected dependencies of an App.
type Options struct {
	Logger *slog.Logger
	Config Config
	Checks []Check
	// WriterEpoch reports the fencing epoch of the active simulation writer.
	// It returns 0 until the M04 writer-lease subsystem exists, which is the
	// truthful value: no lease has been acquired yet.
	WriterEpoch func() int64
	// Version is reported in the startup log entry.
	Version string
}

// App is the HTTP application.
type App struct {
	logger      *slog.Logger
	cfg         Config
	checks      []Check
	writerEpoch func() int64
	version     string
	server      *http.Server
}

// New validates the options and builds the HTTP server. It performs no I/O.
func New(opts Options) (*App, error) {
	switch {
	case opts.Logger == nil:
		return nil, errors.New("app: logger is required")
	case opts.Config.HTTPAddr == "":
		return nil, errors.New("app: HTTPAddr is required")
	case opts.Config.ReadinessTimeout <= 0:
		return nil, errors.New("app: ReadinessTimeout must be positive")
	case opts.Config.ShutdownTimeout <= 0:
		return nil, errors.New("app: ShutdownTimeout must be positive")
	}
	a := &App{
		logger:      opts.Logger,
		cfg:         opts.Config,
		checks:      opts.Checks,
		writerEpoch: opts.WriterEpoch,
		version:     opts.Version,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("GET /readyz", a.handleReadyz)
	mux.HandleFunc("/", a.handleFallback)

	a.server = &http.Server{
		Addr:              opts.Config.HTTPAddr,
		Handler:           a.wrap(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(opts.Logger.Handler(), slog.LevelWarn),
	}
	return a, nil
}

// Handler exposes the fully wrapped handler for tests and for embedding.
func (a *App) Handler() http.Handler { return a.server.Handler }

// Listen binds the configured address. Split from Serve so tests can use an
// ephemeral port and still exercise the real server.
func (a *App) Listen() (net.Listener, error) {
	ln, err := net.Listen("tcp", a.cfg.HTTPAddr)
	if err != nil {
		return nil, fmt.Errorf("app: listen %s: %w", a.cfg.HTTPAddr, err)
	}
	return ln, nil
}

// Serve accepts connections until ctx is cancelled or the listener fails.
//
// Cancellation triggers a graceful shutdown bounded by ShutdownTimeout; nil is
// returned when shutdown completes cleanly, so a supervisor can distinguish a
// requested stop from a crash.
func (a *App) Serve(ctx context.Context, ln net.Listener) error {
	if ctx.Err() != nil {
		_ = ln.Close()
		return nil
	}
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("http server listening",
			slog.String("addr", ln.Addr().String()),
		)
		errCh <- a.server.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("http server shutdown requested", slog.Duration("timeout", a.cfg.ShutdownTimeout))
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.ShutdownTimeout)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			_ = a.server.Close()
			return fmt.Errorf("app: graceful shutdown: %w", err)
		}
		<-errCh
		a.logger.Info("http server stopped")
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("app: serve: %w", err)
	}
}

// Run binds and serves until ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	ln, err := a.Listen()
	if err != nil {
		return err
	}
	return a.Serve(ctx, ln)
}
