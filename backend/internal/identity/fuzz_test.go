package identity

import (
	"testing"
)

func fuzzKeys(tb testing.TB) *KeySet {
	tb.Helper()
	key, err := NewKey("v1", validSecret())
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := NewKeySet(key, nil)
	if err != nil {
		tb.Fatal(err)
	}
	return keys
}

func FuzzDeriveInput(f *testing.F) {
	keys := fuzzKeys(f)
	f.Add("mock", "v1", "village-a", "viewer-0001")
	f.Add("replay", "v1", "village-b", "viewer|0002")
	f.Add("mock", "v2", "a|b", "c")
	f.Add("", "", "", "")
	f.Add("mock", "v1", "village-a", "viewer\u202E")
	f.Add("mock", "v1", "village-a", "người-xem-01")
	f.Fuzz(func(t *testing.T, provider, providerVersion, broadcaster, viewer string) {
		if len(provider) > 256 || len(providerVersion) > 256 || len(broadcaster) > 2048 || len(viewer) > 4096 {
			return // keep the target bounded; the validator enforces the real limits
		}
		input := Input{
			Provider:        provider,
			ProviderVersion: providerVersion,
			BroadcasterKey:  broadcaster,
			ViewerID:        viewer,
		}
		result, err := keys.Derive(input)
		if err != nil {
			return
		}
		if !result.ActorKey.Valid() || len(result.ActorKey) != 64 {
			t.Fatalf("accepted input produced invalid actor key %q", result.ActorKey)
		}
		if result.KeyVersion != "v1" || result.Ephemeral {
			t.Fatalf("unexpected result metadata: %+v", result)
		}
		again, err := keys.Derive(input)
		if err != nil || again != result {
			t.Fatalf("derivation is not deterministic: %+v vs %+v (%v)", result, again, err)
		}
	})
}

func FuzzSanitizeDisplayAlias(f *testing.F) {
	f.Add("Villager 0001", uint8(0), "Villager 9999")
	f.Add("Villager\u202E", uint8(0), "Villager 9999")
	f.Add("<script>", uint8(0), "<fallback>")
	f.Add("", uint8(1), "fallback")
	f.Add("😀😀", uint8(0), "")
	f.Fuzz(func(t *testing.T, alias string, visibility uint8, fallback string) {
		if len(alias) > 4096 || len(fallback) > 4096 {
			return
		}
		result := SanitizeDisplayAlias(alias, AliasVisibility(visibility), fallback)
		if result.Visible {
			if result.Text == "" {
				t.Fatalf("visible alias has empty text")
			}
			if len(result.Text) > AliasMaxBytes {
				t.Fatalf("visible alias exceeds byte bound: %d", len(result.Text))
			}
			if !result.Fallback && result.Text != alias {
				t.Fatalf("non-fallback alias text differs from input")
			}
			if !structurallySafeAlias(result.Text) || unsafeAliasContent(result.Text) {
				t.Fatalf("visible alias is structurally unsafe: %q", result.Text)
			}
		} else if result.Text != "" {
			t.Fatalf("hidden alias leaked text: %q", result.Text)
		}
	})
}
