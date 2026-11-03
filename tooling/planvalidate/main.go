// Command planvalidate verifies that the canonical TTGQLDL plan is internally
// consistent and that milestone status claims are backed by evidence.
//
// It is deliberately dependency-free: Go standard library only. It reads the
// repository as data and never writes to it.
//
// Usage (from the repository root or from tooling/):
//
//	cd tooling && go run ./planvalidate -root ..
//
// Exit codes: 0 = all checks pass, 1 = at least one error, 2 = usage/IO error.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	planDir       = "docs/plan"
	evidenceDir   = "docs/evidence"
	traceability  = "docs/traceability.csv"
	matrixFile    = "10_IMPLEMENTATION_MATRIX.md"
	qaFile        = "09_QA_ACCEPTANCE.md"
	indexFile     = "00_INDEX.md"
	reviewFile    = "12_REVIEW_FINDINGS.md"
	expectedColsN = 10
)

var canonicalPlanFiles = []string{
	"00_INDEX.md",
	"01_PRODUCT_SCOPE.md",
	"02_GAME_RULES.md",
	"03_SIMULATION.md",
	"04_DATABASE.md",
	"05_API_WEBSOCKET.md",
	"06_LIVE_SECURITY.md",
	"07_UNITY_ART_UX.md",
	"08_OPERATIONS.md",
	"09_QA_ACCEPTANCE.md",
	"10_IMPLEMENTATION_MATRIX.md",
	"11_EXTERNAL_GATES.md",
	"12_REVIEW_FINDINGS.md",
}

// traceabilityColumns is the canonical schema documented in 09_QA_ACCEPTANCE.md.
var traceabilityColumns = []string{
	"requirement_id", "document_anchor", "owner_module", "work_package", "test_id",
	"priority", "verification_method", "spec_status", "impl_status", "evidence_uri",
}

var (
	milestoneStatuses = map[string]bool{
		"NOT_STARTED": true, "IN_PROGRESS": true, "BLOCKED": true, "FAIL": true, "PASS": true,
	}
	specStatuses    = map[string]bool{"VERIFIED": true, "FIXED": true, "BLOCKED": true}
	priorities      = map[string]bool{"P0": true, "P1": true}
	verifyMethods   = map[string]bool{"AUTOMATED": true, "MANUAL": true, "AUTOMATED+MANUAL": true, "STATIC": true}
	evidenceStatus  = map[string]bool{"PASS": true, "FAIL": true, "BLOCKED": true, "IN_PROGRESS": true}
	evidenceKinds   = map[string]bool{"automated": true, "human": true}
	externalGateIDs = map[string]bool{
		"GATE-LIVE": true, "GATE-OBS": true, "GATE-LEGAL": true,
		"GATE-HUMAN-UX": true, "GATE-PERF": true,
	}
)

type severity int

const (
	sevError severity = iota
	sevWarn
)

func (s severity) String() string {
	if s == sevWarn {
		return "WARN"
	}
	return "ERROR"
}

type issue struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

type report struct {
	Root     string            `json:"root"`
	Checks   []string          `json:"checks"`
	Issues   []issue           `json:"issues"`
	Stats    map[string]string `json:"stats"`
	Errors   int               `json:"errors"`
	Warnings int               `json:"warnings"`
	Passed   bool              `json:"passed"`
}

type validator struct {
	root   string
	issues []issue
	checks []string
	stats  map[string]string

	files     map[string][]string // relative path -> lines
	slugSets  map[string]map[string]bool
	matrix    map[string]*milestone
	order     []string
	wpList    []string
	wpOwner   map[string]string
	qaIDs     map[string]string // qa id -> defining file
	reqIDs    map[string]string // requirement id -> traceability row anchor
	evidence  map[string]*evRecord
	execBases []string
}

type milestone struct {
	ID       string
	Deps     []string
	State    string
	Gate     string
	Line     int
	Packages []string
}

type evRecord struct {
	File        string
	WP          string
	Milestone   string
	Status      string
	Commit      string
	Head        string
	UTCTime     string
	Env         string
	Tests       []string
	Requirement []string
	Reviewer    string
	ReviewKind  string
}

func main() {
	root := flag.String("root", ".", "repository root")
	asJSON := flag.Bool("json", false, "emit a machine-readable report")
	verbose := flag.Bool("v", false, "list every executed check")
	flag.Parse()

	v := &validator{
		root:     *root,
		stats:    map[string]string{},
		files:    map[string][]string{},
		slugSets: map[string]map[string]bool{},
		matrix:   map[string]*milestone{},
		wpOwner:  map[string]string{},
		qaIDs:    map[string]string{},
		reqIDs:   map[string]string{},
		evidence: map[string]*evRecord{},
	}
	v.run()

	rep := report{Root: v.root, Checks: v.checks, Issues: v.issues, Stats: v.stats}
	for _, is := range v.issues {
		if is.Severity == "ERROR" {
			rep.Errors++
		} else {
			rep.Warnings++
		}
	}
	rep.Passed = rep.Errors == 0

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	} else {
		if *verbose {
			for _, c := range v.checks {
				fmt.Printf("  check: %s\n", c)
			}
		}
		for _, is := range v.issues {
			loc := is.File
			if is.Line > 0 {
				loc = fmt.Sprintf("%s:%d", is.File, is.Line)
			}
			fmt.Printf("%-5s %-22s %s: %s\n", is.Severity, is.Check, loc, is.Message)
		}
		keys := make([]string, 0, len(v.stats))
		for k := range v.stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-24s %s\n", k+":", v.stats[k])
		}
	}
	if !rep.Passed {
		if !*asJSON {
			fmt.Printf("FAIL: %d error(s), %d warning(s), %d checks\n", rep.Errors, rep.Warnings, len(v.checks))
		}
		os.Exit(1)
	}
	if !*asJSON {
		fmt.Printf("PASS: %d checks, 0 errors, %d warning(s)\n", len(v.checks), rep.Warnings)
	}
}

func (v *validator) addf(check string, sev severity, file string, line int, format string, args ...any) {
	v.issues = append(v.issues, issue{
		Check: check, Severity: sev.String(), File: file, Line: line,
		Message: fmt.Sprintf(format, args...),
	})
}

func (v *validator) begin(name string) { v.checks = append(v.checks, name) }

func (v *validator) abs(rel string) string { return filepath.Join(v.root, filepath.FromSlash(rel)) }

func (v *validator) read(rel string) ([]string, bool) {
	if lines, ok := v.files[rel]; ok {
		return lines, true
	}
	data, err := os.ReadFile(v.abs(rel))
	if err != nil {
		return nil, false
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	v.files[rel] = lines
	return lines, true
}

func (v *validator) run() {
	if _, err := os.Stat(v.abs(planDir)); err != nil {
		fmt.Fprintf(os.Stderr, "planvalidate: %s not found under %q\n", planDir, v.root)
		os.Exit(2)
	}
	v.collectMarkdown()
	v.checkCanonicalFiles()
	v.checkLinks()
	v.parseMatrix()
	v.checkMilestoneGraph()
	v.checkWorkPackages()
	v.parseQAIDs()
	v.checkQAReferences()
	v.checkObsoletePlans()
	v.checkConstants()
	v.checkTraceability()
	v.loadEvidence()
	v.checkEvidence()
	v.checkStatusHonesty()
	v.checkIndexCoverage()
	v.summary()
}

// ---------- file inventory ----------

// generatedDirs are repository-local build outputs that are never committed and
// never subject to plan linting.
var generatedDirs = []string{
	"unity/Library",
	"unity/Temp",
	"unity/Obj",
	"unity/Logs",
	"unity/Build",
	"unity/Builds",
	"unity/TestResults",
	"unity/UserSettings",
}

func (v *validator) collectMarkdown() {
	v.begin("markdown-inventory")
	_ = filepath.WalkDir(v.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == ".dsh" {
				return filepath.SkipDir
			}
			// Generated Unity directories are gitignored build caches; linting
			// package documentation inside them would fail every local run for
			// content the repository does not own.
			if rel, rerr := filepath.Rel(v.root, path); rerr == nil {
				rel = filepath.ToSlash(rel)
				for _, generated := range generatedDirs {
					if rel == generated || strings.HasPrefix(rel, generated+"/") {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}
		rel, rerr := filepath.Rel(v.root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		v.read(rel)
		return nil
	})
	v.stats["markdown_files"] = fmt.Sprint(len(v.files))
	for rel, lines := range v.files {
		v.slugSets[rel] = headingSlugs(lines)
	}
}

func (v *validator) checkCanonicalFiles() {
	v.begin("canonical-plan-files")
	for _, f := range canonicalPlanFiles {
		rel := planDir + "/" + f
		if _, ok := v.read(rel); !ok {
			v.addf("canonical-plan-files", sevError, rel, 0, "required canonical plan module is missing")
		}
	}
	v.stats["canonical_modules"] = fmt.Sprint(len(canonicalPlanFiles))
}

// ---------- markdown links and anchors ----------

var linkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^()\s]+(?:\s+"[^"]*")?)\)`)

func (v *validator) checkLinks() {
	v.begin("markdown-links")
	links := 0
	rels := make([]string, 0, len(v.files))
	for rel := range v.files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		lines := v.files[rel]
		for i, line := range lines {
			for _, m := range linkPattern.FindAllStringSubmatch(line, -1) {
				target := strings.TrimSpace(m[1])
				if idx := strings.Index(target, " \""); idx >= 0 {
					target = target[:idx]
				}
				if target == "" || strings.HasPrefix(target, "#") ||
					strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				links++
				pathPart, anchor := target, ""
				if idx := strings.Index(target, "#"); idx >= 0 {
					pathPart, anchor = target[:idx], target[idx+1:]
				}
				if pathPart == "" {
					continue
				}
				resolved := pathPart
				if !strings.HasPrefix(pathPart, "/") {
					resolved = filepath.ToSlash(filepath.Join(filepath.Dir(rel), pathPart))
				} else {
					resolved = strings.TrimPrefix(pathPart, "/")
				}
				resolved = filepath.ToSlash(filepath.Clean(resolved))
				if _, ok := v.read(resolved); !ok {
					if _, err := os.Stat(v.abs(resolved)); err != nil {
						v.addf("markdown-links", sevError, rel, i+1, "broken link target %q", target)
						continue
					}
				}
				if anchor == "" || !strings.HasSuffix(resolved, ".md") {
					continue
				}
				slugs, ok := v.slugSets[resolved]
				if !ok {
					continue
				}
				if !slugs[anchor] {
					v.addf("markdown-links", sevError, rel, i+1,
						"anchor %q not found in %s", anchor, resolved)
				}
			}
		}
	}
	v.stats["markdown_links"] = fmt.Sprint(links)
}

// headingSlugs approximates the GitHub heading-anchor algorithm (lowercase,
// drop punctuation, spaces to hyphens, duplicate headings get -1, -2, ...).
func headingSlugs(lines []string) map[string]bool {
	out := map[string]bool{}
	seen := map[string]int{}
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "#") {
			continue
		}
		text := strings.TrimLeft(trimmed, "#")
		text = strings.TrimSpace(text)
		slug := slugify(text)
		if slug == "" {
			continue
		}
		if n, ok := seen[slug]; ok {
			seen[slug] = n + 1
			out[fmt.Sprintf("%s-%d", slug, n)] = true
			continue
		}
		seen[slug] = 1
		out[slug] = true
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return strings.ReplaceAll(b.String(), " ", "-")
}

// ---------- implementation matrix ----------

var (
	matrixRow = regexp.MustCompile(`^\|\s*(M\d{2})\s*\|([^|]*)\|([^|]*)\|([^|]*)\|([^|]*)\|`)
	wpLine    = regexp.MustCompile(`(M\d{2})\s+((?:W\d{4}[^;]*;\s*)*W\d{4}[^;]*)`)
	wpToken   = regexp.MustCompile(`W\d{4}`)
	statusRe  = regexp.MustCompile(`M(\d{2})(?:[-–]M(\d{2}))?\s*=\s*(NOT_STARTED|IN_PROGRESS|BLOCKED|PASS|FAIL)`)
)

func (v *validator) parseMatrix() {
	v.begin("matrix-parse")
	rel := planDir + "/" + matrixFile
	lines, ok := v.read(rel)
	if !ok {
		return
	}
	for i, line := range lines {
		if m := matrixRow.FindStringSubmatch(line); m != nil {
			id := m[1]
			if _, dup := v.matrix[id]; dup {
				v.addf("matrix-parse", sevError, rel, i+1, "duplicate milestone id %s", id)
				continue
			}
			ms := &milestone{ID: id, State: strings.TrimSpace(m[4]), Gate: strings.TrimSpace(m[5]), Line: i + 1}
			for _, dep := range strings.Split(m[2], ",") {
				dep = strings.TrimSpace(dep)
				if dep == "" || dep == "none" {
					continue
				}
				ms.Deps = append(ms.Deps, dep)
			}
			if !milestoneStatuses[ms.State] {
				v.addf("matrix-parse", sevError, rel, i+1, "%s has invalid state %q", id, ms.State)
			}
			v.matrix[id] = ms
			v.order = append(v.order, id)
		}
		if m := wpLine.FindStringSubmatch(line); m != nil {
			msID := m[1]
			for _, wp := range wpToken.FindAllString(m[2], -1) {
				if owner, dup := v.wpOwner[wp]; dup {
					v.addf("matrix-parse", sevError, rel, i+1, "work package %s listed twice (already under %s)", wp, owner)
					continue
				}
				v.wpOwner[wp] = msID
				v.wpList = append(v.wpList, wp)
				if ms := v.matrix[msID]; ms != nil {
					ms.Packages = append(ms.Packages, wp)
				}
			}
		}
	}
	if len(v.matrix) == 0 {
		v.addf("matrix-parse", sevError, rel, 0, "no milestone rows parsed")
	}
	v.stats["milestones"] = fmt.Sprint(len(v.matrix))
	v.stats["work_packages"] = fmt.Sprint(len(v.wpList))
}

func (v *validator) checkMilestoneGraph() {
	v.begin("milestone-dag")
	ids := make([]string, 0, len(v.matrix))
	for id := range v.matrix {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, dep := range v.matrix[id].Deps {
			switch {
			case dep == "":
			case strings.HasPrefix(dep, "M"):
				if _, ok := v.matrix[dep]; !ok {
					v.addf("milestone-dag", sevError, planDir+"/"+matrixFile, v.matrix[id].Line,
						"%s depends on undefined milestone %s", id, dep)
				}
			case strings.HasPrefix(dep, "GATE-"):
				if !externalGateIDs[dep] {
					v.addf("milestone-dag", sevError, planDir+"/"+matrixFile, v.matrix[id].Line,
						"%s depends on undefined external gate %s", id, dep)
				}
			default:
				v.addf("milestone-dag", sevError, planDir+"/"+matrixFile, v.matrix[id].Line,
					"%s has unparseable dependency %q (use Mnn or GATE-xxx)", id, dep)
			}
		}
	}
	// Cycle detection (M edges only).
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var visit func(string) bool
	visit = func(id string) bool {
		color[id] = grey
		stack = append(stack, id)
		for _, dep := range v.matrix[id].Deps {
			if !strings.HasPrefix(dep, "M") {
				continue
			}
			if _, ok := v.matrix[dep]; !ok {
				continue
			}
			switch color[dep] {
			case grey:
				v.addf("milestone-dag", sevError, planDir+"/"+matrixFile, v.matrix[id].Line,
					"dependency cycle: %s", strings.Join(append(stack, dep), " -> "))
				return true
			case white:
				if visit(dep) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return false
	}
	for _, id := range ids {
		if color[id] == white {
			if visit(id) {
				break
			}
		}
	}
	// Every milestone must be reachable from M00 through dependencies.
	reached := map[string]bool{"M00": true}
	for changed := true; changed; {
		changed = false
		for id, ms := range v.matrix {
			if reached[id] {
				continue
			}
			for _, dep := range ms.Deps {
				if reached[dep] {
					reached[id] = true
					changed = true
					break
				}
			}
		}
	}
	for _, id := range ids {
		if !reached[id] {
			v.addf("milestone-dag", sevWarn, planDir+"/"+matrixFile, v.matrix[id].Line,
				"%s is not reachable from M00 through milestone dependencies", id)
		}
	}
	v.stats["dag_roots"] = fmt.Sprint(len(v.matrix) - len(reached) + 1)
}

func (v *validator) checkWorkPackages() {
	v.begin("work-packages")
	rel := planDir + "/" + matrixFile
	for _, wp := range v.wpList {
		owner := v.wpOwner[wp]
		if len(wp) != 5 || wp[0] != 'W' {
			v.addf("work-packages", sevError, rel, 0, "malformed work package id %q", wp)
			continue
		}
		want := "M" + wp[1:3]
		if owner != want {
			v.addf("work-packages", sevError, rel, 0,
				"work package %s is listed under %s but its id belongs to %s", wp, owner, want)
		}
	}
	if ms := v.matrix["M00"]; ms != nil && len(ms.Packages) == 0 {
		v.addf("work-packages", sevError, rel, ms.Line, "M00 has no work packages")
	}
	for id, ms := range v.matrix {
		if len(ms.Packages) == 0 && id != "M00" {
			v.addf("work-packages", sevWarn, rel, ms.Line, "%s has no work packages", id)
		}
	}
}

// ---------- QA identifiers ----------

var (
	qaDefinition = regexp.MustCompile(`^\s*(?:[-*]\s*)?(QA-[A-Z]+-\d{2})\b`)
	qaReference  = regexp.MustCompile(`QA-[A-Z]+-\d{2}(?:\.\.\d{2}|/\d{2})*`)
)

func (v *validator) parseQAIDs() {
	v.begin("qa-id-definitions")
	rel := planDir + "/" + qaFile
	lines, ok := v.read(rel)
	if !ok {
		return
	}
	for i, line := range lines {
		m := qaDefinition.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id := m[1]
		if prev, dup := v.qaIDs[id]; dup {
			v.addf("qa-id-definitions", sevError, rel, i+1, "duplicate QA id %s (already defined at %s)", id, prev)
			continue
		}
		v.qaIDs[id] = fmt.Sprintf("%s:%d", rel, i+1)
	}
	v.stats["qa_ids"] = fmt.Sprint(len(v.qaIDs))
}

// expandQA expands the shorthand forms used by the plan: "QA-X-06..09",
// "QA-X-03/04" and chained forms such as "QA-GOV-01/02/03".
func expandQA(token string) []string {
	if m := regexp.MustCompile(`^(QA-[A-Z]+-)(\d{2})\.\.(\d{2})$`).FindStringSubmatch(token); m != nil {
		var out []string
		var lo, hi int
		fmt.Sscanf(m[2], "%d", &lo)
		fmt.Sscanf(m[3], "%d", &hi)
		for i := lo; i <= hi; i++ {
			out = append(out, fmt.Sprintf("%s%02d", m[1], i))
		}
		return out
	}
	if m := regexp.MustCompile(`^(QA-[A-Z]+-)(\d{2})((?:/\d{2})+)$`).FindStringSubmatch(token); m != nil {
		out := []string{m[1] + m[2]}
		for _, seg := range strings.Split(m[3], "/")[1:] {
			out = append(out, m[1]+seg)
		}
		return out
	}
	return []string{token}
}

func (v *validator) checkQAReferences() {
	v.begin("qa-id-references")
	rels := make([]string, 0, len(v.files))
	for rel := range v.files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	refs := map[string]bool{}
	for _, rel := range rels {
		if strings.Contains(rel, "/evidence/") {
			continue // evidence test lists are validated by the evidence check
		}
		for i, line := range v.files[rel] {
			for _, tok := range qaReference.FindAllString(line, -1) {
				for _, id := range expandQA(tok) {
					refs[id] = true
					if _, ok := v.qaIDs[id]; !ok {
						v.addf("qa-id-references", sevError, rel, i+1, "reference to undefined QA id %s", id)
					}
				}
			}
		}
	}
	v.stats["qa_references"] = fmt.Sprint(len(refs))
	for id := range v.qaIDs {
		if !refs[id] {
			v.addf("qa-id-references", sevWarn, planDir+"/"+qaFile, 0, "QA id %s is defined but never referenced", id)
		}
	}
}

// ---------- obsolete parallel plans ----------

var prohibition = regexp.MustCompile(`(?i)\b(no|not|never|do not|don't|forbidden|forbid|avoid|without|instead of|instead)\b`)

func (v *validator) checkObsoletePlans() {
	v.begin("obsolete-parallel-plans")
	for _, p := range []string{"MASTER_PLAN.md", "docs/v2", "docs/v3", "docs/plan_v2", "PLAN.md"} {
		if _, err := os.Stat(v.abs(p)); err == nil {
			v.addf("obsolete-parallel-plans", sevError, p, 0, "obsolete parallel plan artefact exists; only docs/plan is canonical")
		}
	}
	rels := make([]string, 0, len(v.files))
	for rel := range v.files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	mentions := 0
	for _, rel := range rels {
		for i, line := range v.files[rel] {
			lower := strings.ToLower(line)
			for _, bad := range []string{"master_plan", "docs/v2", "docs/v3"} {
				idx := strings.Index(lower, bad)
				if idx < 0 {
					continue
				}
				mentions++
				if !prohibition.MatchString(line) {
					v.addf("obsolete-parallel-plans", sevError, rel, i+1,
						"reference to obsolete plan artefact %q in a non-prohibition sentence", bad)
				}
			}
		}
	}
	v.stats["obsolete_plan_mentions"] = fmt.Sprint(mentions)
}

// ---------- cross-module constant contracts ----------

type rule struct {
	Check   string
	Doc     string
	Must    []string
	MustNot []string
}

// constantContracts pins the cross-module numeric and timing contracts. Every
// entry is a documented value that two or more owner modules must agree on; a
// silent drift (for example a different fighter coefficient) fails the check.
var constantContracts = []rule{
	{"constants-gameplay", "02_GAME_RULES.md", []string{
		`300s`, `10Hz`, `0\.\.2999`,
		`\[0,160s\)`, `\[160,260s\)`, `\[260,280s\)`, `\[280,300s\)`,
		`18\*isqrt`, `8\*isqrt`, `40-4\*\(P-1\)`,
		`ceil\(base_hp/5\)`, `34/100`, `32/100`,
		`200\*wall_tier`, `100\*wall_tier`, `1,000,000`, `600 simulation ticks`,
		`ttg-rules-v1`, `consecutive_wins=min\(10`, `min\(50,1\+floor\(total_wins/3\)\)`,
	}, []string{`14\*isqrt`, `coefficient 14`, `consecutive_wins=min\(100`}},
	{"constants-simulation", "03_SIMULATION.md", []string{
		`0\.\.2999`, `\[0,1600\)`, `\[1600,2600\)`, `\[2600,2800\)`, `\[2800,3000\)`,
		`18\*isqrt`, `8\*isqrt`, `40-4\*\(P-1\)`, `combat_index`, `1200`, `1600`, `2600`, `2800`, `3000`,
		`ttg-rules-v1`, `schema_version`,
	}, []string{`18\*isqrt\(min\(F,9999\)`}},
	{"constants-database", "04_DATABASE.md", []string{
		`1000\+200\*\(wall_tier-1\)`, `1000000`, `600`, `1600`, `2600`, `2800`,
		`privacy_epoch`, `writer_epoch`, `ttg-rules-v1`, `schema_version`, `joined_tick=0`,
	}, nil},
	{"constants-qa", "09_QA_ACCEPTANCE.md", []string{
		`18\*isqrt\(6\)`, `162`, `238`, `2800`, `ttg-rules-v1`, `docs/traceability.csv`,
	}, []string{`07_OPERATIONS`}},
	{"constants-security", "06_LIVE_SECURITY.md", []string{
		`600 ticks`, `HMAC-SHA256`,
	}, nil},
	{"constants-unity", "07_UNITY_ART_UX.md", []string{
		`1080x1920`, `150`, `2600`, `2800`,
	}, nil},
	{"constants-api", "05_API_WEBSOCKET.md", []string{
		`128KiB`, `ttg\.v1`, `base_revision`, `stream_seq`,
	}, nil},
	{"constants-ops", "08_OPERATIONS.md", []string{
		`Go 1\.27\.3`, `PostgreSQL 18\.6`, `Unity 6000\.6\.7f1`, `TTG_MAX_QUEUE=20000`,
	}, nil},
	{"constants-scope", "01_PRODUCT_SCOPE.md", []string{
		`Go 1\.27\.3`, `PostgreSQL 18\.6`, `Unity 6000\.6\.7f1`,
	}, nil},
}

func (v *validator) checkConstants() {
	v.begin("constant-contracts")
	for _, r := range constantContracts {
		rel := planDir + "/" + r.Doc
		lines, ok := v.read(rel)
		if !ok {
			continue
		}
		text := strings.Join(lines, "\n")
		for _, pat := range r.Must {
			re, err := regexp.Compile(pat)
			if err != nil {
				v.addf("constant-contracts", sevError, "tooling/planvalidate", 0, "bad contract regex %q", pat)
				continue
			}
			if !re.MatchString(text) {
				v.addf(r.Check, sevError, rel, 0, "missing required contract token /%s/", pat)
			}
		}
		for _, pat := range r.MustNot {
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			if loc := re.FindStringIndex(text); loc != nil {
				line := 1 + strings.Count(text[:loc[0]], "\n")
				v.addf(r.Check, sevError, rel, line, "forbidden retired contract token /%s/", pat)
			}
		}
	}
}

// ---------- evidence records ----------

var evidenceName = regexp.MustCompile(`^(M\d{2})-(W\d{4})\.md$`)

func (v *validator) loadEvidence() {
	v.begin("evidence-load")
	dir := v.abs(evidenceDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		rel := evidenceDir + "/" + e.Name()
		lines, _ := v.read(rel)
		rec := &evRecord{File: rel}
		m := evidenceName.FindStringSubmatch(e.Name())
		if m == nil {
			v.addf("evidence-load", sevError, rel, 0, "evidence file name must be Mxx-Wxxxx.md")
			continue
		}
		rec.Milestone, rec.WP = m[1], m[2]
		fields := evidenceFields(lines)
		rec.Status = fields["status"]
		rec.Commit = fields["commit"]
		rec.Head = fields["head"]
		rec.UTCTime = fields["utc"]
		rec.Env = fields["environment"]
		rec.Reviewer = fields["reviewer"]
		rec.ReviewKind = fields["reviewer_kind"]
		rec.Tests = splitList(fields["tests"])
		rec.Requirement = splitList(fields["requirements"])
		if wpField := bareWP(fields["wp"]); wpField != "" && wpField != rec.WP {
			v.addf("evidence-load", sevError, rel, 0, "evidence wp=%q disagrees with file name %s", fields["wp"], rec.WP)
		}
		if fields["milestone"] != "" && fields["milestone"] != rec.Milestone {
			v.addf("evidence-load", sevError, rel, 0, "evidence milestone=%q disagrees with file name %s", fields["milestone"], rec.Milestone)
		}
		v.evidence[rec.WP] = rec
	}
	v.stats["evidence_records"] = fmt.Sprint(len(v.evidence))
}

// evidenceFields parses the machine-readable
//
//	<!-- evidence
//	key: value
//	-->
//
// block that 09_QA_ACCEPTANCE.md requires at the top of every evidence file.
func evidenceFields(lines []string) map[string]string {
	out := map[string]string{}
	in := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "<!-- evidence"):
			in = true
			continue
		case in && strings.HasPrefix(trimmed, "-->"):
			return out
		case in:
			idx := strings.Index(trimmed, ":")
			if idx <= 0 {
				continue
			}
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+1:])
			out[key] = val
		}
	}
	return out
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// bareWP accepts both "W0001" and the human-readable "M00-W0001" form used by
// the plan and returns the bare work-package id.
var wpPrefixed = regexp.MustCompile(`^M\d{2}-(W\d{4})$`)

func bareWP(s string) string {
	if m := wpPrefixed.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return s
}

var (
	fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
	treeSHA = regexp.MustCompile(`^WORKTREE@[0-9a-f]{40}$`)
)

func (v *validator) checkEvidence() {
	v.begin("evidence-schema")
	for wp, rec := range v.evidence {
		if !evidenceStatus[rec.Status] {
			v.addf("evidence-schema", sevError, rec.File, 0, "invalid or missing status %q", rec.Status)
		}
		if !evidenceKinds[rec.ReviewKind] {
			v.addf("evidence-schema", sevError, rec.File, 0, "reviewer_kind must be automated or human, got %q", rec.ReviewKind)
		}
		if rec.Reviewer == "" {
			v.addf("evidence-schema", sevError, rec.File, 0, "reviewer must be recorded")
		}
		if !fullSHA.MatchString(rec.Head) {
			v.addf("evidence-schema", sevError, rec.File, 0, "head must be a 40-hex commit sha, got %q", rec.Head)
		}
		if !fullSHA.MatchString(rec.Commit) && !treeSHA.MatchString(rec.Commit) {
			v.addf("evidence-schema", sevError, rec.File, 0,
				"commit must be a 40-hex sha or WORKTREE@<40-hex head>, got %q", rec.Commit)
		}
		if rec.UTCTime == "" || !strings.HasSuffix(rec.UTCTime, "Z") {
			v.addf("evidence-schema", sevError, rec.File, 0, "utc must be an RFC3339 UTC timestamp ending in Z, got %q", rec.UTCTime)
		}
		if rec.Env == "" {
			v.addf("evidence-schema", sevError, rec.File, 0, "environment must be recorded")
		}
		if len(rec.Tests) == 0 {
			v.addf("evidence-schema", sevError, rec.File, 0, "tests list must not be empty")
		}
		for _, t := range rec.Tests {
			for _, id := range expandQA(t) {
				if _, ok := v.qaIDs[id]; !ok {
					v.addf("evidence-schema", sevError, rec.File, 0, "evidence references undefined QA id %s", id)
				}
			}
		}
		if owner, ok := v.wpOwner[wp]; !ok {
			v.addf("evidence-schema", sevError, rec.File, 0, "evidence for %s but the work package is not in the matrix", wp)
		} else if owner != rec.Milestone {
			v.addf("evidence-schema", sevError, rec.File, 0, "%s belongs to %s, evidence claims %s", wp, owner, rec.Milestone)
		}
		for _, r := range rec.Requirement {
			if _, ok := v.reqIDs[r]; !ok {
				v.addf("evidence-schema", sevError, rec.File, 0, "evidence references unknown requirement %s", r)
			}
		}
	}
}

// ---------- status honesty ----------

func (v *validator) checkStatusHonesty() {
	v.begin("status-evidence")
	rel := planDir + "/" + matrixFile
	for id, ms := range v.matrix {
		evidenceForMilestone := 0
		passing := 0
		for _, wp := range ms.Packages {
			rec, ok := v.evidence[wp]
			if !ok {
				continue
			}
			evidenceForMilestone++
			if rec.Status == "PASS" {
				passing++
			}
		}
		switch ms.State {
		case "PASS":
			if len(ms.Packages) == 0 {
				v.addf("status-evidence", sevError, rel, ms.Line, "%s is PASS without work packages", id)
				continue
			}
			if passing != len(ms.Packages) {
				v.addf("status-evidence", sevError, rel, ms.Line,
					"%s is PASS but only %d/%d work packages have PASS evidence", id, passing, len(ms.Packages))
			}
			for _, rec := range v.evidence {
				if rec.Milestone != id {
					continue
				}
				if rec.Status == "PASS" && rec.ReviewKind != "automated" && rec.Reviewer == "" {
					v.addf("status-evidence", sevError, rec.File, 0, "human PASS evidence requires a reviewer")
				}
			}
		case "NOT_STARTED", "IN_PROGRESS", "FAIL":
			if evidenceForMilestone > 0 && passing == len(ms.Packages) && len(ms.Packages) > 0 {
				v.addf("status-evidence", sevWarn, rel, ms.Line,
					"%s has complete PASS evidence but is still marked %s", id, ms.State)
			}
		}
	}
	// M10/M11 are release gates: they may only leave BLOCKED with human evidence.
	for _, id := range []string{"M10", "M11"} {
		ms, ok := v.matrix[id]
		if !ok {
			continue
		}
		if ms.State == "BLOCKED" {
			continue
		}
		ok = false
		for _, wp := range ms.Packages {
			if rec := v.evidence[wp]; rec != nil && rec.Status == "PASS" && rec.ReviewKind == "human" && rec.Reviewer != "" {
				ok = true
			}
		}
		if !ok {
			v.addf("status-evidence", sevError, rel, ms.Line,
				"%s is an external release gate and may not leave BLOCKED without human PASS evidence", id)
		}
	}

	// The narrative "Current truth" statement must agree with the table so the
	// prose can never advertise PASS while the matrix says otherwise.
	lines, _ := v.read(rel)
	claimed := map[string]string{}
	for i, line := range lines {
		if !strings.Contains(line, "Current truth") {
			continue
		}
		for _, m := range statusRe.FindAllStringSubmatch(line, -1) {
			first := m[1]
			last := m[2]
			if last == "" {
				last = first
			}
			var lo, hi int
			fmt.Sscanf(first, "%d", &lo)
			fmt.Sscanf(last, "%d", &hi)
			for n := lo; n <= hi; n++ {
				claimed[fmt.Sprintf("M%02d", n)] = m[3]
			}
			_ = i
		}
	}
	if len(claimed) == 0 {
		v.addf("status-evidence", sevWarn, rel, 0,
			"no machine-readable 'Current truth' status line (expected 'M00 = IN_PROGRESS; ...')")
	}
	for id, state := range claimed {
		ms := v.matrix[id]
		if ms == nil {
			v.addf("status-evidence", sevError, rel, 0, "Current truth line names undefined milestone %s", id)
			continue
		}
		if ms.State != state {
			v.addf("status-evidence", sevError, rel, ms.Line,
				"Current truth says %s = %s but the matrix row says %s", id, state, ms.State)
		}
	}
	for id := range v.matrix {
		if _, ok := claimed[id]; !ok && len(claimed) > 0 {
			v.addf("status-evidence", sevWarn, rel, 0, "Current truth line does not state a status for %s", id)
		}
	}
}

func (v *validator) checkIndexCoverage() {
	v.begin("index-coverage")
	rel := planDir + "/" + indexFile
	lines, ok := v.read(rel)
	if !ok {
		return
	}
	text := strings.Join(lines, "\n")
	for _, f := range canonicalPlanFiles {
		if f == indexFile {
			continue
		}
		if !strings.Contains(text, f) {
			v.addf("index-coverage", sevError, rel, 0, "index does not reference module %s", f)
		}
	}
	if !strings.Contains(text, "12_REVIEW_FINDINGS.md") {
		v.addf("index-coverage", sevError, rel, 0, "index must reference the findings log")
	}
	if !strings.Contains(strings.ToLower(text), "not normative") {
		v.addf("index-coverage", sevWarn, rel, 0, "index should mark 12_REVIEW_FINDINGS.md as non-normative")
	}
}

// ---------- traceability ----------

func (v *validator) checkTraceability() {
	v.begin("traceability")
	rel := traceability
	data, err := os.ReadFile(v.abs(rel))
	if err != nil {
		v.addf("traceability", sevError, rel, 0, "traceability file is missing")
		return
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		v.addf("traceability", sevError, rel, 0, "invalid CSV: %v", err)
		return
	}
	if len(rows) == 0 {
		v.addf("traceability", sevError, rel, 0, "traceability file is empty")
		return
	}
	header := rows[0]
	if len(header) != expectedColsN {
		v.addf("traceability", sevError, rel, 1, "expected %d columns, got %d", expectedColsN, len(header))
		return
	}
	for i, want := range traceabilityColumns {
		if strings.TrimSpace(header[i]) != want {
			v.addf("traceability", sevError, rel, 1, "column %d must be %q, got %q", i+1, want, header[i])
		}
	}
	tested := map[string]bool{}
	rowsByReq := 0
	for i, row := range rows[1:] {
		line := i + 2
		if len(row) < expectedColsN {
			v.addf("traceability", sevError, rel, line, "row has %d columns, want %d", len(row), expectedColsN)
			continue
		}
		req := strings.TrimSpace(row[0])
		if req == "" {
			v.addf("traceability", sevError, rel, line, "empty requirement id")
			continue
		}
		if _, dup := v.reqIDs[req]; dup {
			v.addf("traceability", sevError, rel, line, "duplicate requirement id %s", req)
			continue
		}
		v.reqIDs[req] = fmt.Sprintf("%s:%d", rel, line)
		rowsByReq++
		v.checkAnchor(rel, line, req, row[1])
		wp := bareWP(strings.TrimSpace(row[3]))
		if _, ok := v.wpOwner[wp]; !ok {
			v.addf("traceability", sevError, rel, line, "%s references undefined work package %q", req, wp)
		}
		tests := strings.TrimSpace(row[4])
		if tests == "" {
			v.addf("traceability", sevError, rel, line, "%s has no acceptance test id", req)
		}
		for _, tok := range splitList(tests) {
			for _, id := range expandQA(tok) {
				tested[id] = true
				if _, ok := v.qaIDs[id]; !ok {
					v.addf("traceability", sevError, rel, line, "%s references undefined QA id %s", req, id)
				}
			}
		}
		if p := strings.TrimSpace(row[5]); !priorities[p] {
			v.addf("traceability", sevError, rel, line, "%s has invalid priority %q", req, p)
		}
		if m := strings.TrimSpace(row[6]); !verifyMethods[m] {
			v.addf("traceability", sevError, rel, line, "%s has invalid verification method %q", req, m)
		}
		if s := strings.TrimSpace(row[7]); !specStatuses[s] {
			v.addf("traceability", sevError, rel, line, "%s has invalid spec_status %q", req, s)
		}
		if s := strings.TrimSpace(row[8]); !milestoneStatuses[s] {
			v.addf("traceability", sevError, rel, line, "%s has invalid impl_status %q", req, s)
		}
		ev := strings.TrimSpace(row[9])
		if ev == "" {
			v.addf("traceability", sevError, rel, line, "%s has no evidence location", req)
		} else if _, err := os.Stat(v.abs(ev)); err != nil {
			v.addf("traceability", sevError, rel, line, "%s evidence %q does not exist", req, ev)
		}
	}
	v.stats["requirements"] = fmt.Sprint(rowsByReq)
	for id := range v.qaIDs {
		if !tested[id] {
			v.addf("traceability", sevError, rel, 0, "QA id %s is not mapped to any requirement row", id)
		}
	}
}

var anchorFile = regexp.MustCompile(`^([0-9A-Za-z_.-]+\.md)(?:#(.+))?$`)

func (v *validator) checkAnchor(csvRel string, line int, req, anchor string) {
	anchor = strings.TrimSpace(anchor)
	m := anchorFile.FindStringSubmatch(anchor)
	if m == nil {
		v.addf("traceability", sevError, csvRel, line, "%s anchor %q must be <module>.md#<heading>", req, anchor)
		return
	}
	rel := planDir + "/" + m[1]
	if _, ok := v.read(rel); !ok {
		v.addf("traceability", sevError, csvRel, line, "%s anchor file %q does not exist", req, m[1])
		return
	}
	if m[2] == "" {
		return
	}
	if !v.slugSets[rel][m[2]] {
		v.addf("traceability", sevError, csvRel, line, "%s anchor heading %q not found in %s", req, m[2], m[1])
	}
}

func (v *validator) summary() {
	v.stats["canonical_matrix_status"] = v.matrixStatusSummary()
}

func (v *validator) matrixStatusSummary() string {
	ids := make([]string, 0, len(v.matrix))
	for id := range v.matrix {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id+"="+v.matrix[id].State)
	}
	return strings.Join(parts, " ")
}
