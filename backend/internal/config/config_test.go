package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// validEnv is the smallest environment that satisfies every mandatory field in
// dev mode: database URL, HMAC key version and HMAC secret.
func validEnv() map[string]string {
	return map[string]string{
		"TTG_DATABASE_URL":     "postgres://ttg_app:secret@127.0.0.1:5432/ttgqldl?sslmode=disable",
		"TTG_HMAC_KEY_VERSION": "1",
		"TTG_HMAC_SECRET":      "0123456789abcdef0123456789abcdef",
	}
}

func lookupFrom(env map[string]string) func(string) string {
	return func(key string) string { return env[key] }
}

func TestLoadAppliesDocumentedDefaults(t *testing.T) {
	cfg, err := Load(lookupFrom(validEnv()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Env != EnvDev {
		t.Errorf("Env = %q, want dev", cfg.Env)
	}
	if cfg.Source != SourceMock {
		t.Errorf("Source = %q, want mock", cfg.Source)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, DefaultHTTPAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Errorf("ShutdownTimeout = %s, want %s", cfg.ShutdownTimeout, DefaultShutdownTimeout)
	}
	if cfg.ReadinessTimeout != DefaultReadinessTimeout {
		t.Errorf("ReadinessTimeout = %s, want %s", cfg.ReadinessTimeout, DefaultReadinessTimeout)
	}
	if cfg.MaxQueue != DefaultMaxQueue || cfg.StreamRateHz != DefaultStreamRateHz || cfg.SnapshotTicks != DefaultSnapshotTicks {
		t.Errorf("tuning defaults = %d/%d/%d", cfg.MaxQueue, cfg.StreamRateHz, cfg.SnapshotTicks)
	}
}

func TestLoadAcceptsExplicitValues(t *testing.T) {
	env := validEnv()
	env["TTG_ENV"] = "stage"
	env["TTG_SOURCE"] = "replay"
	env["TTG_HTTP_ADDR"] = "0.0.0.0:9091"
	env["TTG_LOG_LEVEL"] = "debug"
	env["TTG_SHUTDOWN_TIMEOUT"] = "25s"
	env["TTG_READINESS_TIMEOUT"] = "1500ms"
	env["TTG_DATABASE_URL"] = "postgres://ttg_app:secret@db.internal:5432/ttgqldl?sslmode=require"
	env["TTG_ADMIN_PASSWORD_HASH"] = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA"
	env["TTG_STREAM_SIGNING_SECRET"] = strings.Repeat("s", 32)

	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Env != EnvStage || cfg.Source != SourceReplay {
		t.Errorf("env/source = %q/%q", cfg.Env, cfg.Source)
	}
	if cfg.ShutdownTimeout != 25*time.Second || cfg.ReadinessTimeout != 1500*time.Millisecond {
		t.Errorf("timeouts = %s/%s", cfg.ShutdownTimeout, cfg.ReadinessTimeout)
	}
	if cfg.IsDevelopment() {
		t.Errorf("stage must not report as development")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"unknown env", "TTG_ENV", "production", "TTG_ENV"},
		{"unknown source", "TTG_SOURCE", "kafka", "TTG_SOURCE"},
		{"authorized live is gated", "TTG_SOURCE", "authorized_live", "GATE-LIVE"},
		{"bad http addr", "TTG_HTTP_ADDR", "localhost", "TTG_HTTP_ADDR"},
		{"http port out of range", "TTG_HTTP_ADDR", "127.0.0.1:70000", "TTG_HTTP_ADDR"},
		{"empty http host", "TTG_HTTP_ADDR", ":8080", "TTG_HTTP_ADDR"},
		{"bad log level", "TTG_LOG_LEVEL", "verbose", "TTG_LOG_LEVEL"},
		{"bad shutdown duration", "TTG_SHUTDOWN_TIMEOUT", "ten seconds", "TTG_SHUTDOWN_TIMEOUT"},
		{"negative shutdown", "TTG_SHUTDOWN_TIMEOUT", "-1s", "TTG_SHUTDOWN_TIMEOUT"},
		{"excessive shutdown", "TTG_SHUTDOWN_TIMEOUT", "10m", "TTG_SHUTDOWN_TIMEOUT"},
		{"zero readiness", "TTG_READINESS_TIMEOUT", "0s", "TTG_READINESS_TIMEOUT"},
		{"bad stream rate", "TTG_STREAM_RATE_HZ", "50", "TTG_STREAM_RATE_HZ"},
		{"non numeric queue", "TTG_MAX_QUEUE", "many", "TTG_MAX_QUEUE"},
		{"short hmac", "TTG_HMAC_SECRET", "short", "TTG_HMAC_SECRET"},
		{"empty hmac version", "TTG_HMAC_KEY_VERSION", "", "TTG_HMAC_KEY_VERSION"},
		{"ws scheme", "TTG_DATABASE_URL", "http://user:pw@127.0.0.1:5432/db", "TTG_DATABASE_URL"},
		{"missing db name", "TTG_DATABASE_URL", "postgres://user:pw@127.0.0.1:5432/", "TTG_DATABASE_URL"},
		{"missing host", "TTG_DATABASE_URL", "postgres:///db", "TTG_DATABASE_URL"},
		{"empty database url", "TTG_DATABASE_URL", "", "TTG_DATABASE_URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validEnv()
			env[tc.key] = tc.value
			_, err := Load(lookupFrom(env))
			if err == nil {
				t.Fatalf("expected an error for %s=%q", tc.key, tc.value)
			}
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("error does not wrap ErrInvalid: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestLoadStageRequiresPrivilegedSecrets(t *testing.T) {
	env := validEnv()
	env["TTG_ENV"] = "prod"
	_, err := Load(lookupFrom(env))
	if err == nil {
		t.Fatal("expected prod to require admin hash and stream signing secret")
	}
	if !strings.Contains(err.Error(), "TTG_ADMIN_PASSWORD_HASH") || !strings.Contains(err.Error(), "TTG_STREAM_SIGNING_SECRET") {
		t.Fatalf("missing fields not reported together: %v", err)
	}

	env["TTG_ADMIN_PASSWORD_HASH"] = "plaintext"
	env["TTG_STREAM_SIGNING_SECRET"] = strings.Repeat("x", 32)
	_, err = Load(lookupFrom(env))
	if err == nil || !strings.Contains(err.Error(), "Argon2id") {
		t.Fatalf("expected Argon2id format rejection, got %v", err)
	}
}

func TestLoadRejectsPlaintextTLSForRemoteHostOutsideDev(t *testing.T) {
	env := validEnv()
	env["TTG_ENV"] = "prod"
	env["TTG_ADMIN_PASSWORD_HASH"] = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA"
	env["TTG_STREAM_SIGNING_SECRET"] = strings.Repeat("x", 32)
	env["TTG_DATABASE_URL"] = "postgres://app:pw@db.example.com:5432/ttgqldl?sslmode=disable"
	_, err := Load(lookupFrom(env))
	if err == nil || !strings.Contains(err.Error(), "sslmode=disable") {
		t.Fatalf("expected sslmode rejection, got %v", err)
	}

	// Loopback stays allowed: the connection never leaves the host.
	env["TTG_DATABASE_URL"] = "postgres://app:pw@127.0.0.1:5432/ttgqldl?sslmode=disable"
	if _, err := Load(lookupFrom(env)); err != nil {
		t.Fatalf("loopback disable must be allowed in prod: %v", err)
	}
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	env := map[string]string{
		"TTG_ENV":         "nope",
		"TTG_LOG_LEVEL":   "loud",
		"TTG_HTTP_ADDR":   "bad",
		"TTG_HMAC_SECRET": "short",
		"TTG_MAX_QUEUE":   "-5",
	}
	_, err := Load(lookupFrom(env))
	if err == nil {
		t.Fatal("expected errors")
	}
	var fieldErrors int
	for _, line := range strings.Split(err.Error(), "\n") {
		if strings.Contains(line, ":") {
			fieldErrors++
		}
	}
	if fieldErrors < 5 {
		t.Fatalf("expected at least 5 field errors, got %d in %q", fieldErrors, err)
	}
}

func TestLoadRejectsNilGetenv(t *testing.T) {
	if _, err := Load(nil); err == nil {
		t.Fatal("expected an error for a nil getenv")
	}
}

func TestIsDevelopment(t *testing.T) {
	env := validEnv()
	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsDevelopment() {
		t.Fatal("dev config must report as development")
	}
}
