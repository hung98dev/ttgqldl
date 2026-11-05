package identity

import (
	"strings"
	"testing"
)

func TestSanitizeDisplayAliasAcceptsSafeAliases(t *testing.T) {
	for _, alias := range []string{
		"Villager 0001",
		"Người Chơi",
		"😀😀",
		"Clan_Leader-7",
		strings.Repeat("a", AliasMaxGraphemes),
		"Villager",
	} {
		result := SanitizeDisplayAlias(alias, AliasVisible, "Villager 9999")
		if !result.Visible || result.Fallback || result.Text != alias {
			t.Fatalf("safe alias %q sanitized to %+v", alias, result)
		}
	}
}

func TestSanitizeDisplayAliasRejectsUnsafeAliases(t *testing.T) {
	cases := []struct {
		name  string
		alias string
	}{
		{"bidi", "Villager\u202E0001"},
		{"zero width", "Villager\u200B0001"},
		{"control", "Villager\x00"},
		{"leading mark", "\u0301Villager"},
		{"mark flood", "V" + strings.Repeat("\u0301", AliasMaxMarks+1)},
		{"too many graphemes", strings.Repeat("a", AliasMaxGraphemes+1)},
		{"too many bytes", strings.Repeat("é", AliasMaxBytes)},
		{"url http", "http://example.test/player"},
		{"url https", "https://example.test"},
		{"url www", "www.example.test"},
		{"url tiktok", "tiktok.com/@player"},
		{"markup", "<b>Villager</b>"},
		{"ampersand", "Villager & Co"},
		{"javascript", "javascript:alert(1)"},
		{"invalid utf8", "Villager\xff"},
		{"whitespace", "   "},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := SanitizeDisplayAlias(testCase.alias, AliasVisible, "Villager 0001")
			if !result.Visible || !result.Fallback || result.Text != "Villager 0001" {
				t.Fatalf("unsafe alias %q -> %+v", testCase.alias, result)
			}
		})
	}
}

func TestSanitizeDisplayAliasHonoursVisibility(t *testing.T) {
	for _, visibility := range []AliasVisibility{AliasHidden, AliasErased, AliasOptedOut, AliasVisibility(99)} {
		result := SanitizeDisplayAlias("Villager 0001", visibility, "Villager 9999")
		if result.Visible || result.Text != "" {
			t.Fatalf("visibility %d published %+v", visibility, result)
		}
	}
}

func TestSanitizeDisplayAliasFallbackSafety(t *testing.T) {
	result := SanitizeDisplayAlias("<script>", AliasVisible, "<script>fallback</script>")
	if result.Text != DefaultAliasFallback {
		t.Fatalf("unsafe fallback produced %q", result.Text)
	}
	result = SanitizeDisplayAlias("", AliasVisible, "Villager 0042")
	if result.Text != "Villager 0042" || !result.Fallback {
		t.Fatalf("safe caller fallback not used: %+v", result)
	}
	// An accepted source alias is not automatically publishable.
	result = SanitizeDisplayAlias("https://example.test/player", AliasVisible, "")
	if result.Text != DefaultAliasFallback || !result.Fallback {
		t.Fatalf("URL alias published: %+v", result)
	}
}

func TestAliasNeverEntersIdentityDerivation(t *testing.T) {
	keys, err := NewKeySet(testKey(t, "v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	base := testInput()
	first, err := keys.Derive(base)
	if err != nil {
		t.Fatal(err)
	}
	// The API has no alias field; a caller cannot pass one. Two source events with
	// different aliases but the same stable viewer id must derive the same key.
	second, err := keys.Derive(base)
	if err != nil {
		t.Fatal(err)
	}
	if first.ActorKey != second.ActorKey {
		t.Fatal("identity derivation is not stable")
	}
}
