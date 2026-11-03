package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	input := strings.Join([]string{
		"# comment",
		"",
		"TTG_ENV=dev",
		"export TTG_LOG_LEVEL=debug",
		`QUOTED="value with spaces"`,
		"SINGLE='single quoted'",
		"EMPTY=",
		"  SPACED  =  trimmed  ",
		"# trailing comment line",
	}, "\n")

	values, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDotEnv: %v", err)
	}
	want := map[string]string{
		"TTG_ENV":       "dev",
		"TTG_LOG_LEVEL": "debug",
		"QUOTED":        "value with spaces",
		"SINGLE":        "single quoted",
		"EMPTY":         "",
		"SPACED":        "trimmed",
	}
	for key, expected := range want {
		if got, ok := values[key]; !ok || got != expected {
			t.Errorf("%s = %q (present=%t), want %q", key, got, ok, expected)
		}
	}
	if len(values) != len(want) {
		t.Errorf("parsed %d keys, want %d: %v", len(values), len(want), values)
	}
}

func TestParseDotEnvRejectsMalformedLines(t *testing.T) {
	cases := map[string]string{
		"missing equals": "TTG_ENV dev",
		"invalid name":   "1BAD=value",
		"space in name":  "BAD NAME=value",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseDotEnv(strings.NewReader(input)); err == nil {
				t.Fatalf("expected an error for %q", input)
			}
		})
	}
}

func TestLoadDotEnvFileMissingIsNotAnError(t *testing.T) {
	values, existed, err := LoadDotEnvFile(filepath.Join(t.TempDir(), "absent.env"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existed {
		t.Fatal("reported existed for an absent file")
	}
	if len(values) != 0 {
		t.Fatalf("values = %v, want empty", values)
	}
}

func TestLoadDotEnvFileReadsValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("A=1\nB=two\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	values, existed, err := LoadDotEnvFile(path)
	if err != nil {
		t.Fatalf("LoadDotEnvFile: %v", err)
	}
	if !existed || values["A"] != "1" || values["B"] != "two" {
		t.Fatalf("values = %v existed=%t", values, existed)
	}
}

func TestApplyDotEnvNeverOverridesProcessEnvironment(t *testing.T) {
	existing := map[string]string{"TTG_ENV": "prod"}
	set := map[string]string{}
	lookup := func(key string) (string, bool) {
		v, ok := existing[key]
		return v, ok
	}
	assign := func(key, value string) error {
		set[key] = value
		return nil
	}
	applied, err := ApplyDotEnv(map[string]string{"TTG_ENV": "dev", "TTG_LOG_LEVEL": "debug"}, lookup, assign)
	if err != nil {
		t.Fatalf("ApplyDotEnv: %v", err)
	}
	if _, overridden := set["TTG_ENV"]; overridden {
		t.Fatal("an existing environment variable was overwritten")
	}
	if set["TTG_LOG_LEVEL"] != "debug" {
		t.Fatalf("new variable not applied: %v", set)
	}
	if len(applied) != 1 || applied[0] != "TTG_LOG_LEVEL" {
		t.Fatalf("applied = %v, want [TTG_LOG_LEVEL]", applied)
	}
}
