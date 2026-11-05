package identity

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const (
	vectorSecretV1 = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	vectorSecretV2 = "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"
)

func secretFromHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func keyFromHex(t *testing.T, version, secret string) Key {
	t.Helper()
	key, err := NewKey(version, secretFromHex(t, secret))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestRotationCandidates(t *testing.T) {
	active := keyFromHex(t, "v2", vectorSecretV2)
	prior := keyFromHex(t, "v1", vectorSecretV1)
	keys, err := NewKeySet(active, []Key{prior})
	if err != nil {
		t.Fatal(err)
	}
	if keys.ActiveVersion() != "v2" {
		t.Fatalf("active = %s", keys.ActiveVersion())
	}
	versions := keys.Versions()
	if len(versions) != 2 || versions[0] != "v2" || versions[1] != "v1" {
		t.Fatalf("versions = %v", versions)
	}

	input := testInput()
	candidates, err := keys.Candidates(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d", len(candidates))
	}
	if candidates[0].KeyVersion != "v2" || candidates[1].KeyVersion != "v1" {
		t.Fatalf("candidate order = %v/%v", candidates[0].KeyVersion, candidates[1].KeyVersion)
	}
	if candidates[0].ActorKey == candidates[1].ActorKey {
		t.Fatal("different key versions produced the same actor key")
	}
	activeResult, err := keys.Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	if activeResult != candidates[0] {
		t.Fatal("active derive differs from first candidate")
	}
	for _, candidate := range candidates {
		explicit, err := keys.DeriveWithVersion(input, candidate.KeyVersion)
		if err != nil {
			t.Fatal(err)
		}
		if explicit != candidate {
			t.Fatalf("candidate %s differs from explicit derivation", candidate.KeyVersion)
		}
		if !candidate.ActorKey.Valid() {
			t.Fatalf("candidate %s is not a valid actor key", candidate.KeyVersion)
		}
	}

	if _, err := keys.DeriveWithVersion(input, "v9"); !errors.Is(err, ErrUnknownKeyVersion) {
		t.Fatalf("unknown version error = %v", err)
	}
	missing := input
	missing.ViewerID = ""
	if _, err := keys.Candidates(missing); !errors.Is(err, ErrMissingViewerID) {
		t.Fatalf("candidate input error = %v", err)
	}
	if _, err := keys.DeriveWithVersion(missing, "v1"); !errors.Is(err, ErrMissingViewerID) {
		t.Fatalf("explicit input error = %v", err)
	}
}

func TestRotationOrderingIsIndependentOfInputOrder(t *testing.T) {
	v1 := keyFromHex(t, "v1", vectorSecretV1)
	v2 := keyFromHex(t, "v2", vectorSecretV2)
	v3 := keyFromHex(t, "v3", "404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f")
	first, err := NewKeySet(v2, []Key{v3, v1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewKeySet(v2, []Key{v1, v3})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(first.Versions()) != fmt.Sprint(second.Versions()) {
		t.Fatalf("ordering depends on retained input order: %v vs %v", first.Versions(), second.Versions())
	}
	if fmt.Sprint(first.Versions()) != "[v2 v1 v3]" {
		t.Fatalf("ordering = %v", first.Versions())
	}
	left, err := first.Candidates(testInput())
	if err != nil {
		t.Fatal(err)
	}
	right, err := second.Candidates(testInput())
	if err != nil {
		t.Fatal(err)
	}
	for index := range left {
		if left[index] != right[index] {
			t.Fatalf("candidate %d differs by construction order", index)
		}
	}
}

func TestRotationRejectsDuplicates(t *testing.T) {
	active := keyFromHex(t, "v2", vectorSecretV2)
	if _, err := NewKeySet(active, []Key{keyFromHex(t, "v2", vectorSecretV1)}); !errors.Is(err, ErrDuplicateKeyVersion) {
		t.Fatalf("active duplicate error = %v", err)
	}
	v1 := keyFromHex(t, "v1", vectorSecretV1)
	if _, err := NewKeySet(active, []Key{v1, keyFromHex(t, "v1", vectorSecretV2)}); !errors.Is(err, ErrDuplicateKeyVersion) {
		t.Fatalf("retained duplicate error = %v", err)
	}
}

func TestKeyVersionIsBoundIntoDerivation(t *testing.T) {
	// Same secret bytes, different labels: the key version is part of the HMAC message,
	// so candidate outputs still differ even under a misconfigured shared secret.
	active := keyFromHex(t, "v2", vectorSecretV1)
	prior := keyFromHex(t, "v1", vectorSecretV1)
	keys, err := NewKeySet(active, []Key{prior})
	if err != nil {
		t.Fatal(err)
	}
	first, err := keys.DeriveWithVersion(testInput(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := keys.DeriveWithVersion(testInput(), "v2")
	if err != nil {
		t.Fatal(err)
	}
	if first.ActorKey == second.ActorKey {
		t.Fatal("key version label is not bound into derivation")
	}
}

func TestKeySetRedactsSecrets(t *testing.T) {
	active := keyFromHex(t, "v2", vectorSecretV2)
	keys, err := NewKeySet(active, []Key{keyFromHex(t, "v1", vectorSecretV1)})
	if err != nil {
		t.Fatal(err)
	}
	formatted := fmt.Sprintf("%v %#v", keys, keys)
	if strings.Contains(formatted, vectorSecretV1) || strings.Contains(formatted, vectorSecretV2) {
		t.Fatal("KeySet formatting disclosed secret material")
	}
	if strings.Contains(formatted, "<redacted>") == false {
		t.Fatalf("expected redaction marker in %q", formatted)
	}
}
