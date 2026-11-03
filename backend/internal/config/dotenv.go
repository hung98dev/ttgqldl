package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ParseDotEnv reads a minimal .env file.
//
// Supported syntax: blank lines, `#` comments, an optional `export ` prefix,
// `KEY=VALUE` pairs, and values wrapped in single or double quotes. No escape
// sequences, interpolation or multi-line values are supported: a value is taken
// literally after quote stripping. Unknown syntax is an error rather than being
// ignored, so a typo cannot silently drop a required secret.
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		text = strings.TrimPrefix(text, "export ")
		key, value, found := strings.Cut(text, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", line)
		}
		key = strings.TrimSpace(key)
		if !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("line %d: invalid variable name %q", line, key)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	return values, nil
}

// LoadDotEnvFile parses path. It reports existed=false when the file is absent,
// which is a normal condition for stage/prod deployments that inject the
// environment directly.
func LoadDotEnvFile(path string) (values map[string]string, existed bool, err error) {
	// #nosec G304 -- the path is the developer's TTG_ENV_FILE setting for the local
	// development loader. It is never network-controlled and stage/prod inject the
	// environment directly instead of reading a file.
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	values, err = ParseDotEnv(f)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", path, err)
	}
	return values, true, nil
}

// LoadEnvFileIntoProcess parses path (when present) and sets every value that is
// not already present in the process environment. It returns how many variables
// were applied and whether the file existed.
//
// This is the single place that binds a .env file to the process environment;
// every command (server, migrate) goes through it.
func LoadEnvFileIntoProcess(path string) (applied int, existed bool, err error) {
	values, existed, err := LoadDotEnvFile(path)
	if err != nil || !existed {
		return 0, existed, err
	}
	keys, err := ApplyDotEnv(values, os.LookupEnv, os.Setenv)
	if err != nil {
		return 0, true, err
	}
	return len(keys), true, nil
}

// ApplyDotEnv sets every value that is not already present in the process
// environment and returns the keys it actually set. Existing variables always
// win, so an explicit shell export can never be overwritten by a file.
func ApplyDotEnv(values map[string]string, lookup func(string) (string, bool), set func(string, string) error) ([]string, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	// Deterministic order keeps logs and tests stable.
	slices.Sort(keys)
	var applied []string
	for _, key := range keys {
		if _, exists := lookup(key); exists {
			continue
		}
		if err := set(key, values[key]); err != nil {
			return applied, fmt.Errorf("set %s: %w", key, err)
		}
		applied = append(applied, key)
	}
	return applied, nil
}
