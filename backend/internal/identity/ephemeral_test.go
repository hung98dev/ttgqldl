package identity

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func ephemeralConfig(t *testing.T) EphemeralConfig {
	t.Helper()
	expiry := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return EphemeralConfig{
		SessionID: "synthetic-session-001",
		Secret:    validSecret(),
		ExpiresAt: expiry,
		Now:       func() time.Time { return expiry.Add(-time.Hour) },
	}
}

func TestEphemeralStableWithinSession(t *testing.T) {
	ephemeral, err := NewEphemeral(ephemeralConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	first, err := ephemeral.Derive("synthetic-viewer-01")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ephemeral.Derive("synthetic-viewer-01")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("ephemeral identity is not stable for one viewer within a session")
	}
	if !first.Ephemeral {
		t.Fatal("ephemeral result is not labelled synthetic")
	}
	if first.KeyVersion != EphemeralKeyVersion {
		t.Fatalf("ephemeral label = %s", first.KeyVersion)
	}
	if !first.ActorKey.Valid() {
		t.Fatalf("actor key = %q", first.ActorKey)
	}
	otherViewer, err := ephemeral.Derive("synthetic-viewer-02")
	if err != nil {
		t.Fatal(err)
	}
	if otherViewer.ActorKey == first.ActorKey {
		t.Fatal("different viewers share an ephemeral actor key")
	}
}

func TestEphemeralIsNotCorrelatableAcrossSessions(t *testing.T) {
	config := ephemeralConfig(t)
	config.SessionID = "synthetic-session-001"
	first, err := NewEphemeral(config)
	if err != nil {
		t.Fatal(err)
	}
	config.SessionID = "synthetic-session-002"
	second, err := NewEphemeral(config)
	if err != nil {
		t.Fatal(err)
	}
	left, err := first.Derive("same-viewer")
	if err != nil {
		t.Fatal(err)
	}
	right, err := second.Derive("same-viewer")
	if err != nil {
		t.Fatal(err)
	}
	if left.ActorKey == right.ActorKey {
		t.Fatal("ephemeral identity correlated the same viewer across sessions")
	}
}

func TestEphemeralExpiry(t *testing.T) {
	config := ephemeralConfig(t)
	ephemeral, err := NewEphemeral(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ephemeral.DeriveAt("viewer", config.ExpiresAt.Add(-time.Nanosecond)); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	if _, err := ephemeral.DeriveAt("viewer", config.ExpiresAt); !errors.Is(err, ErrEphemeralExpired) {
		t.Fatalf("at expiry error = %v", err)
	}
	if _, err := ephemeral.DeriveAt("viewer", config.ExpiresAt.Add(time.Hour)); !errors.Is(err, ErrEphemeralExpired) {
		t.Fatalf("after expiry error = %v", err)
	}

	expiredConfig := config
	expiredConfig.Now = func() time.Time { return config.ExpiresAt.Add(time.Second) }
	expired, err := NewEphemeral(expiredConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expired.Derive("viewer"); !errors.Is(err, ErrEphemeralExpired) {
		t.Fatalf("configured clock expiry error = %v", err)
	}
}

func TestEphemeralConfigurationAndInputValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*EphemeralConfig)
		want   error
	}{
		{"empty session", func(c *EphemeralConfig) { c.SessionID = "" }, ErrInvalidSession},
		{"whitespace session", func(c *EphemeralConfig) { c.SessionID = "  " }, ErrInvalidSession},
		{"bidi session", func(c *EphemeralConfig) { c.SessionID = "session\u202E" }, ErrInvalidSession},
		{"nil secret", func(c *EphemeralConfig) { c.Secret = nil }, ErrWeakKey},
		{"short secret", func(c *EphemeralConfig) { c.Secret = bytes.Repeat([]byte{1}, MinSecretBytes-1) }, ErrWeakKey},
		{"degenerate secret", func(c *EphemeralConfig) { c.Secret = bytes.Repeat([]byte{'a'}, MinSecretBytes) }, ErrWeakKey},
		{"zero expiry", func(c *EphemeralConfig) { c.ExpiresAt = time.Time{} }, ErrInvalidEphemeralConf},
		{"nil clock", func(c *EphemeralConfig) { c.Now = nil }, ErrInvalidEphemeralConf},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			config := ephemeralConfig(t)
			testCase.mutate(&config)
			if _, err := NewEphemeral(config); !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}

	ephemeral, err := NewEphemeral(ephemeralConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ephemeral.Derive(""); !errors.Is(err, ErrInvalidIdentityInput) {
		t.Fatalf("empty viewer error = %v", err)
	}
	if _, err := ephemeral.Derive(" \t"); !errors.Is(err, ErrInvalidIdentityInput) {
		t.Fatalf("blank viewer error = %v", err)
	}
	if _, err := ephemeral.Derive("viewer\x00"); !errors.Is(err, ErrInvalidIdentityInput) {
		t.Fatalf("control viewer error = %v", err)
	}
}

func TestEphemeralRedactsSecret(t *testing.T) {
	config := ephemeralConfig(t)
	ephemeral, err := NewEphemeral(config)
	if err != nil {
		t.Fatal(err)
	}
	formatted := fmt.Sprintf("%v %#v", ephemeral, ephemeral)
	if strings.Contains(formatted, string(config.Secret)) {
		t.Fatal("ephemeral formatting disclosed synthetic secret material")
	}
	if !strings.Contains(formatted, "<redacted>") {
		t.Fatalf("expected redaction marker in %q", formatted)
	}
	result, err := ephemeral.Derive("viewer")
	if err != nil {
		t.Fatal(err)
	}
	formattedResult := fmt.Sprintf("%v %#v", result, result)
	if strings.Contains(formattedResult, string(result.ActorKey)) {
		t.Fatal("Result formatting disclosed the private ActorKey")
	}
}
