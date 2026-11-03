package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is two levels up from tooling/unitycheck.
const repoRoot = "../.."

// copyFixture copies the parts of the Unity project the checker reads into a
// temporary directory so each test can break exactly one thing.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()

	files := []string{
		".gitignore",
		"unity/ProjectSettings/ProjectVersion.txt",
		"unity/ProjectSettings/EditorBuildSettings.asset",
		"unity/Packages/manifest.json",
		"unity/Packages/packages-lock.json",
	}
	for _, rel := range files {
		copyFile(t, filepath.Join(repoRoot, rel), filepath.Join(dst, rel))
	}

	assetsSrc := filepath.Join(repoRoot, "unity", "Assets")
	err := filepath.WalkDir(assetsSrc, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return nil
		}
		copyFile(t, path, filepath.Join(dst, rel))
		return nil
	})
	if err != nil {
		t.Fatalf("copy assets: %v", err)
	}
	return dst
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func runChecker(root string) Report {
	c := &checker{root: root}
	c.run()
	rep := c.report
	rep.Issues = c.issues
	for _, issue := range c.issues {
		if issue.Severity == "ERROR" {
			rep.Errors++
		}
	}
	rep.Passed = rep.Errors == 0
	return rep
}

func messages(rep Report) string {
	var b strings.Builder
	for _, issue := range rep.Issues {
		b.WriteString(issue.Path)
		b.WriteString(": ")
		b.WriteString(issue.Message)
		b.WriteString("\n")
	}
	return b.String()
}

func TestRepositoryIsValid(t *testing.T) {
	rep := runChecker(repoRoot)
	if !rep.Passed {
		t.Fatalf("repository violates the unity contract:\n%s", messages(rep))
	}
	if len(rep.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %v", rep.Scenes)
	}
	if len(rep.Assemblies) == 0 {
		t.Fatal("expected assembly definitions to be discovered")
	}
}

func TestMissingMetaIsDetected(t *testing.T) {
	root := copyFixture(t)
	meta := filepath.Join(root, "unity/Assets/_TTGQLDL/Runtime/Core/Enums.cs.meta")
	if err := os.Remove(meta); err != nil {
		t.Fatalf("remove meta: %v", err)
	}

	rep := runChecker(root)
	if rep.Passed {
		t.Fatal("checker accepted an asset without a .meta file")
	}
	if !strings.Contains(messages(rep), "no .meta") {
		t.Fatalf("unexpected findings:\n%s", messages(rep))
	}
}

func TestWrongEditorVersionIsDetected(t *testing.T) {
	root := copyFixture(t)
	path := filepath.Join(root, "unity/ProjectSettings/ProjectVersion.txt")
	if err := os.WriteFile(path, []byte("m_EditorVersion: 2022.3.10f1\n"), 0o644); err != nil {
		t.Fatalf("write version: %v", err)
	}

	rep := runChecker(root)
	if rep.Passed {
		t.Fatal("checker accepted an unpinned editor version")
	}
	if !strings.Contains(messages(rep), "editor version must be") {
		t.Fatalf("unexpected findings:\n%s", messages(rep))
	}
}

func TestUnpinnedPackageIsDetected(t *testing.T) {
	root := copyFixture(t)
	path := filepath.Join(root, "unity/Packages/manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	updated := strings.Replace(string(data), `"com.unity.test-framework": "1.8.0"`,
		`"com.unity.test-framework": "1.4.6"`, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	rep := runChecker(root)
	if rep.Passed {
		t.Fatal("checker accepted a package that is not the pinned version")
	}
	if !strings.Contains(messages(rep), "com.unity.test-framework") {
		t.Fatalf("unexpected findings:\n%s", messages(rep))
	}
}

func TestRuntimeAssemblyCannotReferenceEditorCode(t *testing.T) {
	root := copyFixture(t)
	path := filepath.Join(root, "unity/Assets/_TTGQLDL/Runtime/Core/Ttgqldl.Core.asmdef")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read asmdef: %v", err)
	}
	updated := strings.Replace(string(data), `"references": [],`,
		`"references": ["Ttgqldl.Editor"],`, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write asmdef: %v", err)
	}

	rep := runChecker(root)
	if rep.Passed {
		t.Fatal("checker accepted a runtime assembly that references editor-only code")
	}
	if !strings.Contains(messages(rep), "editor-only assembly") {
		t.Fatalf("unexpected findings:\n%s", messages(rep))
	}
}

func TestSceneWithUnknownProjectGuidIsDetected(t *testing.T) {
	root := copyFixture(t)
	// A package cache enables strict guid resolution (the real repository has
	// one locally; CI falls back to warnings for package guids only).
	cacheMeta := filepath.Join(root, "unity/Library/PackageCache/com.unity.ugui@abc/Runtime/UI.meta")
	if err := os.MkdirAll(filepath.Dir(cacheMeta), 0o755); err != nil {
		t.Fatalf("mkdir package cache: %v", err)
	}
	if err := os.WriteFile(cacheMeta, []byte("fileFormatVersion: 2\nguid: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"), 0o644); err != nil {
		t.Fatalf("write package meta: %v", err)
	}

	// Re-point the scene controller meta at a different guid: the asset still
	// exists, but the scene's reference no longer resolves.
	meta := filepath.Join(root, "unity/Assets/_TTGQLDL/Runtime/Bootstrap/BroadcastSceneController.cs.meta")
	data, err := os.ReadFile(meta)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	updated := []byte("fileFormatVersion: 2\nguid: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n")
	if len(data) == 0 {
		t.Fatal("empty meta")
	}
	if err := os.WriteFile(meta, updated, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	rep := runChecker(root)
	if rep.Passed {
		t.Fatal("checker accepted a scene referencing an unknown guid")
	}
	if !strings.Contains(messages(rep), "unknown guid") {
		t.Fatalf("unexpected findings:\n%s", messages(rep))
	}
}
