// Command unitycheck performs the licence-free Unity project validation used by
// CI: structure, pinned packages, meta-file coverage, scene GUID integrity,
// assembly-definition boundaries and ignore rules.
//
// It is NOT a substitute for running EditMode/PlayMode tests in the real Unity
// Editor; it only proves that the project is internally consistent enough to
// open. The runtime gate lives in tooling/unity-test.ps1|sh and stays BLOCKED
// wherever no licensed editor is available.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	requiredEditorVersion = "6000.6.7f1"
	unityDir              = "unity"
	assetsDir             = "unity/Assets"
	ttgAssetsDir          = "unity/Assets/_TTGQLDL"
)

var (
	guidPattern      = regexp.MustCompile(`guid:\s*([0-9a-f]{32})`)
	bomStripper      = regexp.MustCompile(`^\xEF\xBB\xBF`)
	requiredPackages = map[string]string{
		"com.unity.ugui":           "2.6.0",
		"com.unity.test-framework": "1.8.0",
	}
	requiredDirs = []string{
		"unity/Assets/_TTGQLDL/Runtime/Bootstrap",
		"unity/Assets/_TTGQLDL/Runtime/Core",
		"unity/Assets/_TTGQLDL/Runtime/Presentation",
		"unity/Assets/_TTGQLDL/Runtime/Networking",
		"unity/Assets/_TTGQLDL/Runtime/UI",
		"unity/Assets/_TTGQLDL/Runtime/Diagnostics",
		"unity/Assets/_TTGQLDL/Editor",
		"unity/Assets/_TTGQLDL/Scenes",
		"unity/Assets/_TTGQLDL/Prefabs",
		"unity/Assets/_TTGQLDL/Art",
		"unity/Assets/_TTGQLDL/Resources",
		"unity/Assets/_TTGQLDL/Tests/EditMode",
		"unity/Assets/_TTGQLDL/Tests/PlayMode",
		"unity/Packages",
		"unity/ProjectSettings",
	}
	approvedScenes = []string{
		"unity/Assets/_TTGQLDL/Scenes/00_Bootstrap.unity",
		"unity/Assets/_TTGQLDL/Scenes/01_BroadcastWorld.unity",
		"unity/Assets/_TTGQLDL/Scenes/02_OfflineRecoveryPreview.unity",
	}
	gitignorePatterns = []string{
		"/unity/[Ll]ibrary/",
		"/unity/[Tt]emp/",
		"/unity/[Ll]ogs/",
		"/unity/[Uu]ser[Ss]ettings/",
		"/unity/[Bb]uild/",
		"/unity/TestResults/",
	}
)

// Issue is one validation finding.
type Issue struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// Report is the machine-readable result.
type Report struct {
	Root       string   `json:"root"`
	Checks     []string `json:"checks"`
	Issues     []Issue  `json:"issues"`
	Scenes     []string `json:"scenes"`
	Assemblies []string `json:"assemblies"`
	Errors     int      `json:"errors"`
	Passed     bool     `json:"passed"`
}

type checker struct {
	root   string
	issues []Issue
	checks []string
	report Report
	guids  map[string]string
}

func main() {
	root := flag.String("root", "..", "repository root")
	asJSON := flag.Bool("json", false, "emit a machine-readable report")
	flag.Parse()

	c := &checker{root: *root}
	c.run()

	rep := c.report
	rep.Root = c.root
	rep.Checks = c.checks
	rep.Issues = c.issues
	for _, issue := range c.issues {
		if issue.Severity == "ERROR" {
			rep.Errors++
		}
	}
	rep.Passed = rep.Errors == 0

	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(rep)
	} else {
		for _, issue := range c.issues {
			location := issue.Path
			if location == "" {
				location = "-"
			}
			fmt.Printf("%-5s %-22s %s: %s\n", issue.Severity, issue.Check, location, issue.Message)
		}
		fmt.Printf("scenes: %d, assemblies: %d\n", len(rep.Scenes), len(rep.Assemblies))
		if rep.Passed {
			fmt.Printf("PASS: %d checks, 0 errors\n", len(c.checks))
		} else {
			fmt.Printf("FAIL: %d checks, %d error(s)\n", len(c.checks), rep.Errors)
		}
	}
	if !rep.Passed {
		os.Exit(1)
	}
}

func (c *checker) add(check, severity, path, message string) {
	c.issues = append(c.issues, Issue{Check: check, Severity: severity, Path: path, Message: message})
}

func (c *checker) errorf(check, path, format string, args ...any) {
	c.add(check, "ERROR", path, fmt.Sprintf(format, args...))
}

func (c *checker) abs(rel string) string { return filepath.Join(c.root, filepath.FromSlash(rel)) }

func (c *checker) read(rel string) ([]byte, bool) {
	data, err := os.ReadFile(c.abs(rel))
	if err != nil {
		return nil, false
	}
	return bomStripper.ReplaceAll(data, nil), true
}

func (c *checker) run() {
	c.checkStructure()
	c.checkEditorVersion()
	c.checkPackages()
	c.checkMetaCoverage()
	c.checkAsmdefs()
	c.checkScenes()
	c.checkGitignore()
}

func (c *checker) checkStructure() {
	c.checks = append(c.checks, "unity-structure")
	for _, dir := range requiredDirs {
		info, err := os.Stat(c.abs(dir))
		if err != nil || !info.IsDir() {
			c.errorf("unity-structure", dir, "required directory is missing")
		}
	}
	if _, ok := c.read(unityDir + "/ProjectSettings/ProjectVersion.txt"); !ok {
		c.errorf("unity-structure", unityDir+"/ProjectSettings/ProjectVersion.txt", "missing project version file")
	}
}

func (c *checker) checkEditorVersion() {
	c.checks = append(c.checks, "unity-editor-version")
	data, ok := c.read(unityDir + "/ProjectSettings/ProjectVersion.txt")
	if !ok {
		return
	}
	text := string(data)
	if !strings.Contains(text, "m_EditorVersion: "+requiredEditorVersion) {
		c.errorf("unity-editor-version", unityDir+"/ProjectSettings/ProjectVersion.txt",
			"editor version must be %s", requiredEditorVersion)
	}
}

func (c *checker) checkPackages() {
	c.checks = append(c.checks, "unity-packages")
	manifestRaw, ok := c.read(unityDir + "/Packages/manifest.json")
	if !ok {
		c.errorf("unity-packages", unityDir+"/Packages/manifest.json", "missing manifest")
		return
	}
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		c.errorf("unity-packages", unityDir+"/Packages/manifest.json", "invalid JSON: %v", err)
		return
	}

	lockRaw, lockOK := c.read(unityDir + "/Packages/packages-lock.json")
	var lock struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
			Source  string `json:"source"`
		} `json:"dependencies"`
	}
	if lockOK {
		if err := json.Unmarshal(lockRaw, &lock); err != nil {
			c.errorf("unity-packages", unityDir+"/Packages/packages-lock.json", "invalid JSON: %v", err)
			lockOK = false
		}
	} else {
		c.errorf("unity-packages", unityDir+"/Packages/packages-lock.json", "missing lockfile")
	}

	for name, want := range requiredPackages {
		got, present := manifest.Dependencies[name]
		if !present {
			c.errorf("unity-packages", unityDir+"/Packages/manifest.json", "required package %s is not pinned", name)
			continue
		}
		if got != want {
			c.errorf("unity-packages", unityDir+"/Packages/manifest.json",
				"package %s must be pinned to %s, found %s", name, want, got)
		}
		if lockOK {
			entry, locked := lock.Dependencies[name]
			if !locked {
				c.errorf("unity-packages", unityDir+"/Packages/packages-lock.json", "%s is missing from the lockfile", name)
			} else if entry.Version != want {
				c.errorf("unity-packages", unityDir+"/Packages/packages-lock.json",
					"%s locks %s, expected %s", name, entry.Version, want)
			}
		}
	}

	// Every manifest dependency that is not a built-in module must be locked.
	if lockOK {
		for name := range manifest.Dependencies {
			if strings.HasPrefix(name, "com.unity.modules.") {
				continue
			}
			if _, locked := lock.Dependencies[name]; !locked {
				c.errorf("unity-packages", unityDir+"/Packages/packages-lock.json",
					"%s is used by the manifest but missing from the lockfile", name)
			}
		}
	}
}

func (c *checker) checkMetaCoverage() {
	c.checks = append(c.checks, "unity-meta-files")
	guides := map[string]string{}
	c.guids = guides

	err := filepath.WalkDir(c.abs(assetsDir), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if name == ".gitkeep" {
			return nil
		}
		rel, relErr := filepath.Rel(c.root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(name, ".meta") {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				c.errorf("unity-meta-files", rel, "cannot read meta file")
				return nil
			}
			match := guidPattern.FindSubmatch(data)
			if match == nil {
				c.errorf("unity-meta-files", rel, "meta file has no guid")
				return nil
			}
			guid := string(match[1])
			if previous, duplicate := guides[guid]; duplicate {
				c.errorf("unity-meta-files", rel, "guid %s is duplicated (also %s)", guid, previous)
			} else {
				guides[guid] = rel
			}
			return nil
		}
		// Every asset needs a sibling meta file.
		if _, statErr := os.Stat(path + ".meta"); statErr != nil {
			c.errorf("unity-meta-files", rel, "asset has no .meta file")
		}
		return nil
	})
	if err != nil {
		c.errorf("unity-meta-files", assetsDir, "cannot walk assets: %v", err)
	}

	// Directories need meta files too, otherwise Unity regenerates guids.
	err = filepath.WalkDir(c.abs(assetsDir), func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || path == c.abs(assetsDir) {
			return nil
		}
		if _, statErr := os.Stat(path + ".meta"); statErr != nil {
			rel, _ := filepath.Rel(c.root, path)
			c.errorf("unity-meta-files", filepath.ToSlash(rel), "folder has no .meta file")
		}
		return nil
	})
	if err != nil {
		c.errorf("unity-meta-files", assetsDir, "cannot walk folders: %v", err)
	}

}

func (c *checker) checkAsmdefs() {
	c.checks = append(c.checks, "unity-assemblies")
	var asmdefs []string
	err := filepath.WalkDir(c.abs(assetsDir), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".asmdef") {
			asmdefs = append(asmdefs, path)
		}
		return nil
	})
	if err != nil {
		c.errorf("unity-assemblies", assetsDir, "cannot walk assets: %v", err)
		return
	}
	sort.Strings(asmdefs)

	type definition struct {
		path            string
		name            string
		references      []string
		includePlatform []string
		constraints     []string
		noEngine        bool
	}

	defs := make([]definition, 0, len(asmdefs))
	names := map[string]string{}
	for _, path := range asmdefs {
		rel, _ := filepath.Rel(c.root, path)
		rel = filepath.ToSlash(rel)
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			c.errorf("unity-assemblies", rel, "cannot read asmdef")
			continue
		}
		var parsed struct {
			Name               string   `json:"name"`
			References         []string `json:"references"`
			IncludePlatforms   []string `json:"includePlatforms"`
			DefineConstraints  []string `json:"defineConstraints"`
			NoEngineReferences bool     `json:"noEngineReferences"`
		}
		if err := json.Unmarshal(bomStripper.ReplaceAll(raw, nil), &parsed); err != nil {
			c.errorf("unity-assemblies", rel, "invalid JSON: %v", err)
			continue
		}
		if parsed.Name == "" {
			c.errorf("unity-assemblies", rel, "assembly has no name")
			continue
		}
		if previous, duplicate := names[parsed.Name]; duplicate {
			c.errorf("unity-assemblies", rel, "duplicate assembly name %s (also %s)", parsed.Name, previous)
		}
		names[parsed.Name] = rel
		defs = append(defs, definition{
			path: rel, name: parsed.Name, references: parsed.References,
			includePlatform: parsed.IncludePlatforms, constraints: parsed.DefineConstraints,
			noEngine: parsed.NoEngineReferences,
		})
		c.report.Assemblies = append(c.report.Assemblies, parsed.Name)
	}

	enginePrefixes := []string{"UnityEngine", "UnityEditor", "nunit", "Unity.", "System", "mscorlib"}
	for _, def := range defs {
		editorOnly := len(def.includePlatform) == 1 && def.includePlatform[0] == "Editor"
		isTest := strings.Contains(def.name, ".Tests.")
		for _, reference := range def.references {
			if _, ok := names[reference]; ok {
				if editorOnly {
					continue
				}
				// Runtime assemblies must never depend on editor-only code.
				if reference == "Ttgqldl.Editor" || reference == "UnityEditor.TestRunner" {
					c.errorf("unity-assemblies", def.path,
						"runtime assembly %s references editor-only assembly %s", def.name, reference)
				}
				continue
			}
			allowed := false
			for _, prefix := range enginePrefixes {
				if strings.HasPrefix(reference, prefix) {
					allowed = true
					break
				}
			}
			if !allowed {
				c.errorf("unity-assemblies", def.path,
					"reference %q is neither a defined assembly nor a known engine assembly", reference)
			}
		}
		if isTest && !contains(def.constraints, "UNITY_INCLUDE_TESTS") {
			c.errorf("unity-assemblies", def.path,
				"test assembly %s must be gated by UNITY_INCLUDE_TESTS", def.name)
		}
	}

	// The core assembly must stay pure: no engine reference at all.
	for _, def := range defs {
		if def.name == "Ttgqldl.Core" && !def.noEngine {
			c.errorf("unity-assemblies", def.path, "Ttgqldl.Core must set noEngineReferences=true")
		}
	}
}

func (c *checker) checkScenes() {
	c.checks = append(c.checks, "unity-scenes")
	for _, scene := range approvedScenes {
		if _, ok := c.read(scene); !ok {
			c.errorf("unity-scenes", scene, "approved scene is missing")
			continue
		}
		c.report.Scenes = append(c.report.Scenes, filepath.Base(scene))
	}

	entries, err := os.ReadDir(c.abs(ttgAssetsDir + "/Scenes"))
	if err != nil {
		c.errorf("unity-scenes", ttgAssetsDir+"/Scenes", "cannot list scenes: %v", err)
		return
	}
	count := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".unity") {
			count++
		}
	}
	if count != len(approvedScenes) {
		c.errorf("unity-scenes", ttgAssetsDir+"/Scenes", "expected %d scenes, found %d", len(approvedScenes), count)
	}

	// Editor build settings must list the three scenes, using project-relative
	// paths ("Assets/..."), not repository-relative ones.
	settings, ok := c.read(unityDir + "/ProjectSettings/EditorBuildSettings.asset")
	if !ok {
		c.errorf("unity-scenes", unityDir+"/ProjectSettings/EditorBuildSettings.asset", "missing build settings")
		return
	}
	text := string(settings)
	for _, scene := range approvedScenes {
		projectPath := strings.TrimPrefix(scene, unityDir+"/")
		if !strings.Contains(text, projectPath) {
			c.errorf("unity-scenes", unityDir+"/ProjectSettings/EditorBuildSettings.asset",
				"build settings do not include %s", projectPath)
		}
	}

	c.checkSceneGuids()
}

func (c *checker) checkGitignore() {
	c.checks = append(c.checks, "unity-ignore-rules")
	data, ok := c.read(".gitignore")
	if !ok {
		c.errorf("unity-ignore-rules", ".gitignore", "missing .gitignore")
		return
	}
	text := string(data)
	for _, pattern := range gitignorePatterns {
		if !strings.Contains(text, pattern) {
			c.errorf("unity-ignore-rules", ".gitignore", "missing Unity ignore rule %q", pattern)
		}
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// checkSceneGuids verifies that a scene does not reference a deleted project
// asset and that no MonoBehaviour lost its script.
//
// Script guids from packages (uGUI, EventSystem) live in the package cache,
// which is not committed. When the cache exists the check is strict; in a bare
// CI checkout package guids cannot be resolved, so they are reported as WARN
// instead of failing a build that actually compiles. The authoritative
// missing-script gate is the PlayMode test that walks the real scene.
func (c *checker) checkSceneGuids() {
	resolvable := make(map[string]bool, len(c.guids))
	for guid := range c.guids {
		resolvable[guid] = true
	}

	packageCache := c.abs(unityDir + "/Library/PackageCache")
	strict := false
	if info, err := os.Stat(packageCache); err == nil && info.IsDir() {
		strict = true
		_ = filepath.WalkDir(packageCache, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".meta") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			if match := guidPattern.FindSubmatch(data); match != nil {
				resolvable[string(match[1])] = true
			}
			return nil
		})
	}

	builtIn := regexp.MustCompile(`^0000000000000000[0-9a-f]{16}$`)
	missingScript := regexp.MustCompile(`m_Script:\s*\{fileID:\s*0\}`)

	for _, scene := range approvedScenes {
		data, ok := c.read(scene)
		if !ok {
			continue
		}
		seen := map[string]bool{}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if missingScript.MatchString(line) {
				c.errorf("unity-scenes", scene, "MonoBehaviour with a null m_Script reference (missing script)")
			}
		}
		for _, match := range guidPattern.FindAllSubmatch(data, -1) {
			guid := string(match[1])
			if seen[guid] || guid == strings.Repeat("0", 32) || builtIn.MatchString(guid) {
				continue
			}
			seen[guid] = true
			if resolvable[guid] {
				continue
			}
			if strict {
				c.errorf("unity-scenes", scene, "scene references unknown guid %s", guid)
			} else {
				c.add("unity-scenes", "WARN", scene,
					"guid "+guid+" is not resolvable without the package cache; verify in the editor")
			}
		}
	}
}
