package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(abs, "docs", "plan", matrixFile)); err != nil {
		t.Fatalf("not running inside the repository (docs/plan missing under %s)", abs)
	}
	return abs
}

// copyRepo clones the documentation surface the validator reads into dir.
func copyRepo(t *testing.T, src, dst string) {
	t.Helper()
	paths := []string{
		filepath.Join("docs", "plan"),
		filepath.Join("docs", "evidence"),
		filepath.Join("docs", "traceability.csv"),
		"AGENTS.md",
	}
	for _, rel := range paths {
		s := filepath.Join(src, rel)
		d := filepath.Join(dst, rel)
		info, err := os.Stat(s)
		if err != nil {
			t.Fatalf("stat %s: %v", s, err)
		}
		if info.IsDir() {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", d, err)
			}
			entries, err := os.ReadDir(s)
			if err != nil {
				t.Fatalf("readdir %s: %v", s, err)
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(s, e.Name()))
				if err != nil {
					t.Fatalf("read %s: %v", e.Name(), err)
				}
				if err := os.WriteFile(filepath.Join(d, e.Name()), data, 0o644); err != nil {
					t.Fatalf("write %s: %v", e.Name(), err)
				}
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(d), err)
		}
		data, err := os.ReadFile(s)
		if err != nil {
			t.Fatalf("read %s: %v", s, err)
		}
		if err := os.WriteFile(d, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", d, err)
		}
	}
}

func newValidator(root string) *validator {
	return &validator{
		root:     root,
		stats:    map[string]string{},
		files:    map[string][]string{},
		slugSets: map[string]map[string]bool{},
		matrix:   map[string]*milestone{},
		wpOwner:  map[string]string{},
		qaIDs:    map[string]string{},
		reqIDs:   map[string]string{},
		evidence: map[string]*evRecord{},
	}
}

func runOn(root string) *validator {
	v := newValidator(root)
	v.run()
	return v
}

func errorsOf(v *validator) []string {
	var out []string
	for _, is := range v.issues {
		if is.Severity == "ERROR" {
			out = append(out, is.Check+": "+is.Message)
		}
	}
	return out
}

func mutate(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(data)
	if !strings.Contains(text, old) {
		t.Fatalf("fixture text not found in %s: %q", path, old)
	}
	text = strings.Replace(text, old, replacement, 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestCanonicalPlanPasses is the baseline: the committed plan must be clean.
func TestCanonicalPlanPasses(t *testing.T) {
	root := repoRoot(t)
	v := runOn(root)
	if errs := errorsOf(v); len(errs) > 0 {
		t.Fatalf("canonical plan must validate clean, got %d error(s):\n%s", len(errs), strings.Join(errs, "\n"))
	}
	if len(v.matrix) == 0 || len(v.qaIDs) == 0 || len(v.evidence) == 0 {
		t.Fatalf("validator parsed nothing: milestones=%d qa=%d evidence=%d", len(v.matrix), len(v.qaIDs), len(v.evidence))
	}
	if v.matrix["M00"] == nil || v.matrix["M00"].State != "PASS" {
		t.Fatalf("M00 state=%v", v.matrix["M00"])
	}
	if v.matrix["M10"] != nil && v.matrix["M10"].State != "BLOCKED" {
		t.Fatalf("M10 must stay BLOCKED without human evidence, got %s", v.matrix["M10"].State)
	}
}

// TestValidatorCatchesSeededDefects proves the checks are real (QA-GOV-04): the
// validator must fail for each single seeded defect.
func TestValidatorCatchesSeededDefects(t *testing.T) {
	src := repoRoot(t)
	cases := []struct {
		name   string
		check  string
		mutate func(root string, t *testing.T)
	}{
		{
			name:  "broken markdown link",
			check: "markdown-links",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", indexFile),
					"[01_PRODUCT_SCOPE.md](01_PRODUCT_SCOPE.md)", "[01_PRODUCT_SCOPE.md](01_MISSING.md)")
			},
		},
		{
			name:  "duplicate QA id",
			check: "qa-id-definitions",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", qaFile),
					"QA-SIM-11 XP ordering", "QA-SIM-01 XP ordering")
			},
		},
		{
			name:  "undefined QA reference",
			check: "qa-id-references",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", "11_EXTERNAL_GATES.md"),
					"# EXTERNAL GATES AND HONEST LIMITS", "# EXTERNAL GATES AND HONEST LIMITS (QA-ZZZ-99)")
			},
		},
		{
			name:  "retired fighter coefficient",
			check: "constants-gameplay",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", "02_GAME_RULES.md"),
					"Fighter attack = 18*isqrt(F)", "Fighter attack = 14*isqrt(F)")
			},
		},
		{
			name:  "dependency cycle",
			check: "milestone-dag",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", matrixFile),
					"| M01 | M00 |", "| M01 | M09 |")
			},
		},
		{
			name:  "PASS without evidence",
			check: "status-evidence",
			mutate: func(root string, t *testing.T) {
				if err := os.Remove(filepath.Join(root, "docs", "evidence", "M00-W0004.md")); err != nil {
					t.Fatalf("remove evidence: %v", err)
				}
			},
		},
		{
			name:  "traceability row points at missing evidence",
			check: "traceability",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "traceability.csv"),
					"REQ-01-SCOPE,01_PRODUCT_SCOPE.md#0-goals-scope-governance-and-fixed-technology,01,M00-W0001,QA-GOV-01;QA-GOV-02,P0,STATIC,VERIFIED,NOT_STARTED,docs/evidence/M00-W0001.md",
					"REQ-01-SCOPE,01_PRODUCT_SCOPE.md#0-goals-scope-governance-and-fixed-technology,01,M00-W0001,QA-GOV-01;QA-GOV-02,P0,STATIC,VERIFIED,NOT_STARTED,docs/evidence/M00-NOPE.md")
			},
		},
		{
			name:  "narrative status disagrees with matrix",
			check: "status-evidence",
			mutate: func(root string, t *testing.T) {
				mutate(t, filepath.Join(root, "docs", "plan", matrixFile),
					"- Current truth: M00 = PASS;", "- Current truth: M00 = IN_PROGRESS;")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dst := t.TempDir()
			copyRepo(t, src, dst)
			tc.mutate(dst, t)
			v := runOn(dst)
			errs := errorsOf(v)
			if len(errs) == 0 {
				t.Fatalf("seeded defect %q was not detected", tc.name)
			}
			found := false
			for _, e := range errs {
				if strings.HasPrefix(e, tc.check+":") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected a %q error, got:\n%s", tc.check, strings.Join(errs, "\n"))
			}
		})
	}
}
