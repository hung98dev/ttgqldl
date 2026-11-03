// Package config loads and validates the backend runtime configuration.
//
// The register of environment variables, their defaults and the fields that are
// mandatory in every mode is defined by docs/plan/08_OPERATIONS.md
// ("Required configuration register"). This package is the single place where
// that register is turned into typed values; nothing else may read the process
// environment directly.
//
// Validation is deliberately strict and fails closed: an invalid enum, a weak
// secret, a missing key or an unusable PostgreSQL URL aborts startup with a
// message that names every offending variable.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Environment is the deployment mode.
type Environment string

// Deployment modes accepted by the register.
const (
	EnvDev   Environment = "dev"
	EnvStage Environment = "stage"
	EnvProd  Environment = "prod"
)

// SourceMode selects how gameplay input is acquired.
type SourceMode string

// Source modes accepted by the register.
const (
	SourceMock           SourceMode = "mock"
	SourceReplay         SourceMode = "replay"
	SourceAuthorizedLive SourceMode = "authorized_live"
)

// Defaults from docs/plan/08_OPERATIONS.md.
const (
	DefaultEnv              = EnvDev
	DefaultSource           = SourceMock
	DefaultHTTPAddr         = "127.0.0.1:8080"
	DefaultLogLevel         = "info"
	DefaultMetricsAddr      = "127.0.0.1:9090"
	DefaultShutdownTimeout  = 10 * time.Second
	DefaultReadinessTimeout = 2 * time.Second
	DefaultMaxQueue         = 20000
	DefaultStreamRateHz     = 5
	DefaultSnapshotTicks    = 600

	// MinSecretBytes is the minimum length of a signing or HMAC secret.
	MinSecretBytes = 32

	maxShutdownTimeout  = 5 * time.Minute
	maxReadinessTimeout = 30 * time.Second
	maxStreamRateHz     = 10
	minSnapshotTicks    = 1
)

// ErrInvalid reports that at least one configuration value failed validation.
// Callers can test for it with errors.Is and inspect the joined field errors.
var ErrInvalid = errors.New("invalid configuration")

// FieldError names a single offending environment variable.
type FieldError struct {
	Key    string
	Reason string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Reason)
}

// Identity holds the HMAC material used to derive scoped pseudonymous actor
// keys. Version and secret rotate together; see docs/plan/06_LIVE_SECURITY.md.
type Identity struct {
	HMACKeyVersion string
	HMACSecret     string
}

// Secrets holds credentials that later milestones consume. They are validated
// at startup now so that a misconfigured deployment fails immediately instead
// of at the first privileged request.
type Secrets struct {
	AdminPasswordHash   string
	StreamSigningSecret string
	SourceCredentialRef string
}

// Config is the fully validated runtime configuration.
type Config struct {
	Env              Environment
	Source           SourceMode
	HTTPAddr         string
	DatabaseURL      string
	LogLevel         string
	MetricsAddr      string
	MaxQueue         int
	StreamRateHz     int
	SnapshotTicks    int
	ShutdownTimeout  time.Duration
	ReadinessTimeout time.Duration
	Identity         Identity
	Secrets          Secrets
}

// IsDevelopment reports whether the configuration is a local development one.
func (c Config) IsDevelopment() bool { return c.Env == EnvDev }

// Load reads the register through getenv and validates it.
//
// getenv is injected so that the loader itself never depends on the process
// environment and can be tested exhaustively.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("%w: getenv must not be nil", ErrInvalid)
	}
	raw := func(key string) string { return getenv(key) }
	trimmed := func(key string) string { return strings.TrimSpace(getenv(key)) }

	var problems []error
	fail := func(key, reason string) {
		problems = append(problems, &FieldError{Key: key, Reason: reason})
	}

	cfg := Config{
		Env:              DefaultEnv,
		Source:           DefaultSource,
		HTTPAddr:         DefaultHTTPAddr,
		LogLevel:         DefaultLogLevel,
		MetricsAddr:      DefaultMetricsAddr,
		MaxQueue:         DefaultMaxQueue,
		StreamRateHz:     DefaultStreamRateHz,
		SnapshotTicks:    DefaultSnapshotTicks,
		ShutdownTimeout:  DefaultShutdownTimeout,
		ReadinessTimeout: DefaultReadinessTimeout,
	}

	switch env := Environment(trimmed("TTG_ENV")); env {
	case "":
		cfg.Env = DefaultEnv
	case EnvDev, EnvStage, EnvProd:
		cfg.Env = env
	default:
		fail("TTG_ENV", fmt.Sprintf("must be one of dev|stage|prod, got %q", string(env)))
	}

	switch src := SourceMode(trimmed("TTG_SOURCE")); src {
	case "":
		cfg.Source = DefaultSource
	case SourceMock, SourceReplay:
		cfg.Source = src
	case SourceAuthorizedLive:
		// Fails closed: the authorized LIVE adapter needs GATE-LIVE evidence and
		// the M10 adapter, neither of which exists in this build.
		fail("TTG_SOURCE", "authorized_live is not available in this build: GATE-LIVE is BLOCKED (docs/plan/11_EXTERNAL_GATES.md); use mock or replay")
	default:
		fail("TTG_SOURCE", fmt.Sprintf("must be one of mock|replay|authorized_live, got %q", string(src)))
	}

	if v := trimmed("TTG_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if err := validateHostPort(cfg.HTTPAddr); err != nil {
		fail("TTG_HTTP_ADDR", err.Error())
	}

	if v := trimmed("TTG_METRICS_ADDR"); v != "" {
		cfg.MetricsAddr = v
	}
	if err := validateHostPort(cfg.MetricsAddr); err != nil {
		fail("TTG_METRICS_ADDR", err.Error())
	}

	if v := trimmed("TTG_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		fail("TTG_LOG_LEVEL", fmt.Sprintf("must be one of debug|info|warn|error, got %q", cfg.LogLevel))
	}

	cfg.DatabaseURL = strings.TrimSpace(raw("TTG_DATABASE_URL"))
	if cfg.DatabaseURL == "" {
		fail("TTG_DATABASE_URL", "is required in every mode")
	} else if err := validateDatabaseURL(cfg.DatabaseURL, cfg.Env); err != nil {
		fail("TTG_DATABASE_URL", err.Error())
	}

	cfg.ShutdownTimeout = parseDuration(trimmed("TTG_SHUTDOWN_TIMEOUT"), DefaultShutdownTimeout, "TTG_SHUTDOWN_TIMEOUT", fail)
	if cfg.ShutdownTimeout <= 0 || cfg.ShutdownTimeout > maxShutdownTimeout {
		fail("TTG_SHUTDOWN_TIMEOUT", fmt.Sprintf("must be >0 and <=%s, got %s", maxShutdownTimeout, cfg.ShutdownTimeout))
	}
	cfg.ReadinessTimeout = parseDuration(trimmed("TTG_READINESS_TIMEOUT"), DefaultReadinessTimeout, "TTG_READINESS_TIMEOUT", fail)
	if cfg.ReadinessTimeout <= 0 || cfg.ReadinessTimeout > maxReadinessTimeout {
		fail("TTG_READINESS_TIMEOUT", fmt.Sprintf("must be >0 and <=%s, got %s", maxReadinessTimeout, cfg.ReadinessTimeout))
	}

	cfg.MaxQueue = parsePositiveInt(trimmed("TTG_MAX_QUEUE"), DefaultMaxQueue, "TTG_MAX_QUEUE", fail)
	if cfg.MaxQueue <= 0 {
		fail("TTG_MAX_QUEUE", "must be a positive integer")
	}
	cfg.StreamRateHz = parsePositiveInt(trimmed("TTG_STREAM_RATE_HZ"), DefaultStreamRateHz, "TTG_STREAM_RATE_HZ", fail)
	if cfg.StreamRateHz < 1 || cfg.StreamRateHz > maxStreamRateHz {
		fail("TTG_STREAM_RATE_HZ", fmt.Sprintf("must be between 1 and %d", maxStreamRateHz))
	}
	cfg.SnapshotTicks = parsePositiveInt(trimmed("TTG_SNAPSHOT_TICKS"), DefaultSnapshotTicks, "TTG_SNAPSHOT_TICKS", fail)
	if cfg.SnapshotTicks < minSnapshotTicks {
		fail("TTG_SNAPSHOT_TICKS", fmt.Sprintf("must be >= %d", minSnapshotTicks))
	}

	cfg.Identity.HMACKeyVersion = trimmed("TTG_HMAC_KEY_VERSION")
	if cfg.Identity.HMACKeyVersion == "" {
		fail("TTG_HMAC_KEY_VERSION", "is required in every mode")
	}
	cfg.Identity.HMACSecret = raw("TTG_HMAC_SECRET")
	if len(cfg.Identity.HMACSecret) == 0 {
		fail("TTG_HMAC_SECRET", "is required in every mode")
	} else if len(cfg.Identity.HMACSecret) < MinSecretBytes {
		fail("TTG_HMAC_SECRET", fmt.Sprintf("must be at least %d bytes", MinSecretBytes))
	}

	cfg.Secrets.AdminPasswordHash = strings.TrimSpace(raw("TTG_ADMIN_PASSWORD_HASH"))
	cfg.Secrets.StreamSigningSecret = raw("TTG_STREAM_SIGNING_SECRET")
	cfg.Secrets.SourceCredentialRef = strings.TrimSpace(raw("TTG_SOURCE_CREDENTIAL_REF"))

	if cfg.Env != EnvDev {
		if cfg.Secrets.AdminPasswordHash == "" {
			fail("TTG_ADMIN_PASSWORD_HASH", "is required in stage and prod")
		} else if !strings.HasPrefix(cfg.Secrets.AdminPasswordHash, "$argon2id$") {
			fail("TTG_ADMIN_PASSWORD_HASH", "must be an Argon2id PHC string beginning with $argon2id$")
		}
		if len(cfg.Secrets.StreamSigningSecret) < MinSecretBytes {
			fail("TTG_STREAM_SIGNING_SECRET", fmt.Sprintf("must be at least %d bytes in stage and prod", MinSecretBytes))
		}
	}

	if len(problems) > 0 {
		return Config{}, errors.Join(append([]error{ErrInvalid}, problems...)...)
	}
	return cfg, nil
}

func validateHostPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("must be host:port, got %q", addr)
	}
	if host == "" {
		return fmt.Errorf("must name an explicit host, got %q", addr)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must use a port between 1 and 65535, got %q", port)
	}
	return nil
}

func validateDatabaseURL(raw string, env Environment) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("must be a valid URL: %w", err)
	}
	switch u.Scheme {
	case "postgres", "postgresql":
	default:
		return fmt.Errorf("must use the postgres:// or postgresql:// scheme, got %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return errors.New("must include a host")
	}
	if strings.Trim(u.Path, "/") == "" {
		return errors.New("must include a database name")
	}
	if env == EnvDev {
		return nil
	}
	// Production/stage must not silently disable TLS for a non-loopback server.
	sslmode := u.Query().Get("sslmode")
	if sslmode == "disable" && !isLoopback(u.Hostname()) {
		return fmt.Errorf("sslmode=disable is refused in %s for non-loopback host %q", env, u.Hostname())
	}
	return nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func parseDuration(raw string, def time.Duration, key string, fail func(key, reason string)) time.Duration {
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		fail(key, fmt.Sprintf("must be a Go duration such as 10s, got %q", raw))
		return def
	}
	return d
}

func parsePositiveInt(raw string, def int, key string, fail func(key, reason string)) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		fail(key, fmt.Sprintf("must be an integer, got %q", raw))
		return def
	}
	return n
}
