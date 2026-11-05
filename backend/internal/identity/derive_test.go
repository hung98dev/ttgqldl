package identity

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func validSecret() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func testKey(t *testing.T, version string) Key {
	t.Helper()
	key, err := NewKey(version, validSecret())
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testInput() Input {
	return Input{
		Provider:        "mock",
		ProviderVersion: "v1",
		BroadcasterKey:  "village-a",
		ViewerID:        "viewer-0001",
	}
}

type vectorFixture struct {
	Spec            string   `json:"spec"`
	EncodingVersion int      `json:"encoding_version"`
	Domain          string   `json:"domain"`
	Layout          []string `json:"layout"`
	LP              string   `json:"lp"`
	Note            string   `json:"note"`
	Vectors         []struct {
		Name            string `json:"name"`
		KeyVersion      string `json:"key_version"`
		SecretHex       string `json:"material_hex"`
		Provider        string `json:"provider"`
		ProviderVersion string `json:"provider_version"`
		BroadcasterKey  string `json:"broadcaster_key"`
		ViewerID        string `json:"viewer_id"`
		ActorKey        string `json:"expected_digest"`
	} `json:"vectors"`
}

func loadVectors(t *testing.T) vectorFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/hmac_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture vectorFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Spec != "ttg-actor-key-v1" || fixture.EncodingVersion != EncodingVersion || fixture.Domain != actorKeyDomain {
		t.Fatalf("unexpected vector specification: %+v", fixture)
	}
	if len(fixture.Vectors) < 10 {
		t.Fatalf("only %d golden vectors", len(fixture.Vectors))
	}
	return fixture
}

// TestIndependentGoldenVectors compares production HMAC output against vectors computed
// independently with Python hmac/hashlib from the documented layout, never by this
// implementation.
func TestIndependentGoldenVectors(t *testing.T) {
	fixture := loadVectors(t)
	seen := make(map[string]string)
	for _, vector := range fixture.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			secret, err := hex.DecodeString(vector.SecretHex)
			if err != nil {
				t.Fatal(err)
			}
			key, err := NewKey(vector.KeyVersion, secret)
			if err != nil {
				t.Fatal(err)
			}
			keys, err := NewKeySet(key, nil)
			if err != nil {
				t.Fatal(err)
			}
			result, err := keys.Derive(Input{
				Provider:        vector.Provider,
				ProviderVersion: vector.ProviderVersion,
				BroadcasterKey:  vector.BroadcasterKey,
				ViewerID:        vector.ViewerID,
			})
			if err != nil {
				t.Fatalf("derive: %v", err)
			}
			if string(result.ActorKey) != vector.ActorKey {
				t.Fatalf("actor key = %s, want %s", result.ActorKey, vector.ActorKey)
			}
			if len(result.ActorKey) != 64 || !result.ActorKey.Valid() {
				t.Fatalf("actor key format = %q", result.ActorKey)
			}
			if result.KeyVersion != KeyVersion(vector.KeyVersion) {
				t.Fatalf("key version = %s", result.KeyVersion)
			}
			// Deliberate cross-vector checks: the vector set includes the same viewer
			// under different scopes and keys; those must not collide.
			tuple := fmt.Sprintf("%s|%s|%s|%s", vector.Provider, vector.BroadcasterKey, vector.ViewerID, vector.KeyVersion)
			if previous, ok := seen[vector.ActorKey]; ok && previous != tuple {
				t.Fatalf("actor key collision between %q and %q", previous, tuple)
			}
			seen[vector.ActorKey] = tuple
		})
	}
}

func TestHMACKnownAnswer(t *testing.T) {
	// RFC 4231 test case 2, used as a standard-library sanity check.
	mac := hmac.New(sha256.New, []byte("Jefe"))
	mac.Write([]byte("what do ya want for nothing?"))
	got := hex.EncodeToString(mac.Sum(nil))
	const want = "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"
	if got != want {
		t.Fatalf("HMAC-SHA256 known answer = %s", got)
	}
}

func TestInputValidation(t *testing.T) {
	cases := []struct {
		name  string
		input Input
		want  error
	}{
		{"valid", testInput(), nil},
		{"empty provider", func() Input { i := testInput(); i.Provider = ""; return i }(), ErrInvalidIdentityInput},
		{"uppercase provider", func() Input { i := testInput(); i.Provider = "Mock"; return i }(), ErrInvalidIdentityInput},
		{"provider too long", func() Input { i := testInput(); i.Provider = strings.Repeat("a", MaxProviderBytes+1); return i }(), ErrInvalidIdentityInput},
		{"empty provider version", func() Input { i := testInput(); i.ProviderVersion = ""; return i }(), ErrInvalidIdentityInput},
		{"provider version space", func() Input { i := testInput(); i.ProviderVersion = "v 1"; return i }(), ErrInvalidIdentityInput},
		{"empty broadcaster", func() Input { i := testInput(); i.BroadcasterKey = ""; return i }(), ErrInvalidIdentityInput},
		{"broadcaster bidi", func() Input { i := testInput(); i.BroadcasterKey = "village\u202Ea"; return i }(), ErrInvalidIdentityInput},
		{"broadcaster control", func() Input { i := testInput(); i.BroadcasterKey = "village\n"; return i }(), ErrInvalidIdentityInput},
		{"broadcaster too long", func() Input {
			i := testInput()
			i.BroadcasterKey = strings.Repeat("a", MaxBroadcasterBytes+1)
			return i
		}(), ErrInvalidIdentityInput},
		{"whitespace broadcaster", func() Input { i := testInput(); i.BroadcasterKey = "  "; return i }(), ErrInvalidIdentityInput},
		{"empty viewer", func() Input { i := testInput(); i.ViewerID = ""; return i }(), ErrMissingViewerID},
		{"whitespace viewer", func() Input { i := testInput(); i.ViewerID = " \t"; return i }(), ErrMissingViewerID},
		{"viewer control", func() Input { i := testInput(); i.ViewerID = "viewer\x00"; return i }(), ErrInvalidIdentityInput},
		{"viewer bidi", func() Input { i := testInput(); i.ViewerID = "viewer\u202E1"; return i }(), ErrInvalidIdentityInput},
		{"viewer zero width", func() Input { i := testInput(); i.ViewerID = "viewer\u200B1"; return i }(), ErrInvalidIdentityInput},
		{"viewer invalid utf8", func() Input { i := testInput(); i.ViewerID = "viewer\xff"; return i }(), ErrInvalidIdentityInput},
		{"viewer too long", func() Input { i := testInput(); i.ViewerID = strings.Repeat("v", MaxViewerIDRunes+1); return i }(), ErrInvalidIdentityInput},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.input.Validate()
			if testCase.want == nil {
				if err != nil {
					t.Fatalf("valid input rejected: %v", err)
				}
				return
			}
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
			if testCase.input.ViewerID != "" && strings.Contains(err.Error(), testCase.input.ViewerID) {
				t.Fatalf("error leaked raw viewer id: %v", err)
			}
		})
	}
}

func TestScopedDerivationInvariants(t *testing.T) {
	key := testKey(t, "v1")
	keys, err := NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := testInput()
	first, err := keys.Derive(base)
	if err != nil {
		t.Fatal(err)
	}
	second, err := keys.Derive(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("derivation is not deterministic")
	}
	if !first.ActorKey.Valid() || len(first.ActorKey) != 64 {
		t.Fatalf("actor key = %q", first.ActorKey)
	}
	if first.Ephemeral {
		t.Fatal("provider-backed derivation labelled ephemeral")
	}

	mutations := []struct {
		name   string
		mutate func(*Input)
	}{
		{"provider", func(i *Input) { i.Provider = "replay" }},
		{"provider version", func(i *Input) { i.ProviderVersion = "v2" }},
		{"broadcaster", func(i *Input) { i.BroadcasterKey = "village-b" }},
		{"viewer", func(i *Input) { i.ViewerID = "viewer-0002" }},
	}
	for _, testCase := range mutations {
		t.Run(testCase.name, func(t *testing.T) {
			input := base
			testCase.mutate(&input)
			other, err := keys.Derive(input)
			if err != nil {
				t.Fatal(err)
			}
			if other.ActorKey == first.ActorKey {
				t.Fatalf("%s does not isolate the actor key", testCase.name)
			}
		})
	}

	// Separator-shaped identifiers must remain distinct through length-prefixing.
	left := base
	left.BroadcasterKey = "a|b"
	left.ViewerID = "c"
	right := base
	right.BroadcasterKey = "a"
	right.ViewerID = "b|c"
	leftResult, err := keys.Derive(left)
	if err != nil {
		t.Fatal(err)
	}
	rightResult, err := keys.Derive(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftResult.ActorKey == rightResult.ActorKey {
		t.Fatal("separator ambiguity in identity tuple")
	}
}

func TestKeyValidationAndRedaction(t *testing.T) {
	secret := validSecret()
	key, err := NewKey("v1", secret)
	if err != nil {
		t.Fatal(err)
	}
	if key.Version() != "v1" {
		t.Fatalf("version = %s", key.Version())
	}
	if strings.Contains(key.String(), hex.EncodeToString(secret)) {
		t.Fatal("key String disclosed secret material")
	}
	if strings.Contains(fmt.Sprintf("%#v", key), hex.EncodeToString(secret)) {
		t.Fatal("key GoString disclosed secret material")
	}

	bad := []struct {
		name    string
		version string
		secret  []byte
		want    error
	}{
		{"empty version", "", secret, ErrInvalidKeyVersion},
		{"bad version char", "v 1", secret, ErrInvalidKeyVersion},
		{"long version", strings.Repeat("v", MaxKeyVersionBytes+1), secret, ErrInvalidKeyVersion},
		{"nil secret", "v1", nil, ErrWeakKey},
		{"short secret", "v1", bytes.Repeat([]byte{1}, MinSecretBytes-1), ErrWeakKey},
		{"all zero secret", "v1", make([]byte, MinSecretBytes), ErrWeakKey},
		{"repeated secret", "v1", bytes.Repeat([]byte{'a'}, MinSecretBytes), ErrWeakKey},
		{"oversized secret", "v1", bytes.Repeat([]byte{1}, MaxSecretBytes+1), ErrInvalidKey},
	}
	for _, testCase := range bad {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewKey(testCase.version, testCase.secret)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
			if testCase.secret != nil && strings.Contains(err.Error(), hex.EncodeToString(testCase.secret)) {
				t.Fatal("error disclosed key material")
			}
		})
	}

	if _, err := NewKeySet(Key{}, nil); !errors.Is(err, ErrNoKeyMaterial) {
		t.Fatalf("empty key set error = %v", err)
	}
	if _, err := (&KeySet{}).Derive(testInput()); !errors.Is(err, ErrNoKeyMaterial) {
		t.Fatalf("nil key set derive error = %v", err)
	}
}

func TestConcurrentDerivation(t *testing.T) {
	keys, err := NewKeySet(testKey(t, "v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	want, err := keys.Derive(testInput())
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	failures := make(chan string, 32)
	for worker := 0; worker < 32; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := 0; index < 100; index++ {
				got, err := keys.Derive(testInput())
				if err != nil || got != want {
					failures <- "concurrent derivation diverged"
					return
				}
			}
		}()
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
}

// TestActorKeyCannotBeSuppliedDirectly documents that the API has no path that accepts
// a caller-provided ActorKey as an identity; only derivation returns domain.ActorKey.
func TestActorKeyCannotBeSuppliedDirectly(t *testing.T) {
	keys, err := NewKeySet(testKey(t, "v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	rawProviderID := domain.ActorKey(strings.Repeat("a", 64))
	result, err := keys.Derive(testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.ActorKey == rawProviderID {
		t.Fatal("derivation copied a supplied value")
	}
}

func TestMixedScriptViewerIdsRemainDistinct(t *testing.T) {
	keys, err := NewKeySet(testKey(t, "v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	base := testInput()
	identities := []string{"XAY", "ХАУ", "ΧΑΥ", "xa\u0302y"}
	seen := make(map[domain.ActorKey]string)
	for _, viewer := range identities {
		input := base
		input.ViewerID = viewer
		result, err := keys.Derive(input)
		if err != nil {
			t.Fatalf("opaque viewer %q rejected: %v", viewer, err)
		}
		if previous, exists := seen[result.ActorKey]; exists {
			t.Fatalf("mixed-script viewers %q and %q collided without transliteration", previous, viewer)
		}
		seen[result.ActorKey] = viewer
	}
}

func TestDerivationErrorsDoNotLeakSecrets(t *testing.T) {
	secret := validSecret()
	key, err := NewKey("v1", secret)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = keys.Derive(Input{})
	if err == nil {
		t.Fatal("empty input accepted")
	}
	if strings.Contains(err.Error(), string(secret)) || strings.Contains(err.Error(), hex.EncodeToString(secret)) {
		t.Fatalf("error leaked key material: %v", err)
	}
}
