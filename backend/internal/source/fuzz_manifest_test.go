package source

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzReplayManifest exercises the manifest decoder and its plain-name/duplicate
// validation with bounded temporary files. It never opens a fixture or an arbitrary
// production path.
func FuzzReplayManifest(f *testing.F) {
	f.Add([]byte(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","records":1}]}`))
	f.Add([]byte(`{"schema":2,"fixtures":[]}`))
	f.Add([]byte(`{"schema":1,"fixtures":[{"name":"../escape","schema_version":1,"sha256":"x","records":1}]}`))
	f.Add([]byte(`{"schema":1,"fixtures":[{"name":"a","schema_version":1,"sha256":"x","records":1},{"name":"a","schema_version":1,"sha256":"y","records":2}]}`))
	f.Add([]byte(`{"schema":1,"fixtures":[]} trailing`))
	f.Add([]byte(``))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 4096 {
			return
		}
		directory := t.TempDir()
		path := filepath.Join(directory, "manifest.json")
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		manifest, err := loadReplayManifest(path)
		if err != nil {
			return
		}
		if manifest.Schema != replayManifestSchema {
			t.Fatalf("accepted unsupported schema %d", manifest.Schema)
		}
		seen := make(map[string]struct{}, len(manifest.Fixtures))
		for _, entry := range manifest.Fixtures {
			if entry.Name == "" || filepath.Base(entry.Name) != entry.Name || len(entry.Name) > 256 {
				t.Fatalf("accepted unsafe fixture name %q", entry.Name)
			}
			if _, exists := seen[entry.Name]; exists {
				t.Fatalf("accepted duplicate fixture name %q", entry.Name)
			}
			seen[entry.Name] = struct{}{}
		}
	})
}
