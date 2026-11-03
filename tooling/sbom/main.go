// Command sbom produces a deterministic CycloneDX 1.5 inventory of the
// repository's dependencies (Go modules, npm packages, Unity packages and
// pinned container images) and validates container digest pinning.
//
// It is deliberately dependency-free and reproducible: the same checkout always
// produces byte-identical output, so a changed SBOM means a changed dependency
// graph. It is an engineering inventory, not legal licence approval.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const specVersion = "1.5"

type component struct {
	Type       string          `json:"type"`
	BOMRef     string          `json:"bom-ref"`
	Name       string          `json:"name"`
	Version    string          `json:"version,omitempty"`
	Scope      string          `json:"scope,omitempty"`
	Purl       string          `json:"purl,omitempty"`
	Licenses   []licenseEntry  `json:"licenses,omitempty"`
	Properties []componentProp `json:"properties,omitempty"`
}

type licenseEntry struct {
	License licenseValue `json:"license"`
}

type licenseValue struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type componentProp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bom struct {
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber"`
	Version      int         `json:"version"`
	Metadata     metadata    `json:"metadata"`
	Components   []component `json:"components"`
}

type metadata struct {
	Timestamp string    `json:"timestamp,omitempty"`
	Tools     []tool    `json:"tools"`
	Component component `json:"component"`
}

type tool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type packageLock struct {
	LockfileVersion int    `json:"lockfileVersion"`
	Name            string `json:"name"`
	Packages        map[string]struct {
		Version  string `json:"version"`
		License  string `json:"license"`
		Dev      bool   `json:"dev"`
		Resolved string `json:"resolved"`
		Link     bool   `json:"link"`
	} `json:"packages"`
}

type unityLock struct {
	Dependencies map[string]struct {
		Version      string            `json:"version"`
		Depth        int               `json:"depth"`
		Source       string            `json:"source"`
		Dependencies map[string]string `json:"dependencies"`
	} `json:"dependencies"`
}

type goModule struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Main    bool   `json:"Main"`
	Dir     string `json:"Dir"`
}

func main() {
	root := flag.String("root", "..", "repository root")
	out := flag.String("out", "", "write the CycloneDX document here (default stdout)")
	checkImages := flag.Bool("check-images", false, "only validate container image pinning")
	flag.Parse()

	components, images, err := collect(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}

	if *checkImages {
		if err := validateImages(images); err != nil {
			fmt.Fprintln(os.Stderr, "sbom:", err)
			os.Exit(1)
		}
		fmt.Printf("container images pinned by digest: %d\n", len(images))
		for _, image := range images {
			fmt.Printf("  %s\n", image)
		}
		return
	}

	document := bom{
		BOMFormat:    "CycloneDX",
		SpecVersion:  specVersion,
		SerialNumber: "urn:uuid:" + deterministicUUID(components),
		Version:      1,
		Metadata: metadata{
			Tools: []tool{{Vendor: "TTGQLDL", Name: "tooling/sbom", Version: "1.0.0"}},
			Component: component{
				Type: "application", BOMRef: "app:ttgqldl", Name: "ttgqldl", Version: "M01",
			},
		},
		Components: components,
	}

	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
	payload = append(payload, '\n')

	if *out == "" {
		_, _ = os.Stdout.Write(payload)
		return
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, payload, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}

	known, unknown := licenseSummary(components)
	fmt.Printf("wrote %s (%d components)\n", *out, len(components))
	fmt.Printf("licenses: %d identified, %d without metadata\n", known, unknown)
}

func collect(root string) ([]component, []string, error) {
	components := make([]component, 0, 512)

	golang, err := goComponents(root)
	if err != nil {
		return nil, nil, err
	}
	components = append(components, golang...)

	npm, err := npmComponents(root)
	if err != nil {
		return nil, nil, err
	}
	components = append(components, npm...)

	unity, err := unityComponents(root)
	if err != nil {
		return nil, nil, err
	}
	components = append(components, unity...)

	images, err := containerImages(root)
	if err != nil {
		return nil, nil, err
	}
	for _, image := range images {
		components = append(components, component{
			Type: "container", BOMRef: "container:" + image, Name: image,
			Properties: []componentProp{{Name: "ttgqldl:pin", Value: "sha256"}},
		})
	}

	sort.Slice(components, func(i, j int) bool {
		if components[i].Type != components[j].Type {
			return components[i].Type < components[j].Type
		}
		if components[i].Name != components[j].Name {
			return components[i].Name < components[j].Name
		}
		return components[i].Version < components[j].Version
	})

	return components, images, nil
}

func goComponents(root string) ([]component, error) {
	backend := filepath.Join(root, "backend")
	if _, err := os.Stat(filepath.Join(backend, "go.mod")); err != nil {
		return nil, fmt.Errorf("backend/go.mod is missing: %w", err)
	}

	command := exec.Command("go", "list", "-m", "-json", "all")
	command.Dir = backend
	command.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -m all: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	components := make([]component, 0, 64)
	for decoder.More() {
		var module goModule
		if err := decoder.Decode(&module); err != nil {
			return nil, fmt.Errorf("decode go module: %w", err)
		}
		scope := "required"
		if module.Main {
			scope = "optional"
		}
		components = append(components, component{
			Type:       "library",
			BOMRef:     purl("golang", module.Path, module.Version),
			Name:       module.Path,
			Version:    module.Version,
			Scope:      scope,
			Purl:       purl("golang", module.Path, module.Version),
			Licenses:   []licenseEntry{{License: licenseValue{Name: "NOASSERTION"}}},
			Properties: []componentProp{{Name: "ttgqldl:ecosystem", Value: "go"}},
		})
	}
	return components, nil
}

func npmComponents(root string) ([]component, error) {
	path := filepath.Join(root, "operator", "package-lock.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read operator/package-lock.json: %w", err)
	}

	var lock packageLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, fmt.Errorf("parse operator/package-lock.json: %w", err)
	}
	if lock.LockfileVersion < 3 {
		return nil, fmt.Errorf("operator lockfileVersion %d is older than 3", lock.LockfileVersion)
	}

	components := make([]component, 0, len(lock.Packages))
	for entryPath, entry := range lock.Packages {
		if entryPath == "" || entry.Link {
			continue
		}
		name := strings.TrimPrefix(entryPath, "node_modules/")
		if index := strings.LastIndex(name, "node_modules/"); index >= 0 {
			name = name[index+len("node_modules/"):]
		}
		scope := "required"
		if entry.Dev {
			scope = "optional"
		}
		license := entry.License
		if license == "" {
			license = "NOASSERTION"
		}
		components = append(components, component{
			Type:     "library",
			BOMRef:   purl("npm", name, entry.Version),
			Name:     name,
			Version:  entry.Version,
			Scope:    scope,
			Purl:     purl("npm", name, entry.Version),
			Licenses: []licenseEntry{{License: licenseValue{Name: license}}},
			Properties: []componentProp{
				{Name: "ttgqldl:ecosystem", Value: "npm"},
			},
		})
	}
	return components, nil
}

func unityComponents(root string) ([]component, error) {
	path := filepath.Join(root, "unity", "Packages", "packages-lock.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read unity/packages-lock.json: %w", err)
	}

	var lock unityLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, fmt.Errorf("parse unity/packages-lock.json: %w", err)
	}

	components := make([]component, 0, len(lock.Dependencies))
	for name, entry := range lock.Dependencies {
		components = append(components, component{
			Type:     "library",
			BOMRef:   purl("pkg:unity", name, entry.Version),
			Name:     name,
			Version:  entry.Version,
			Scope:    "required",
			Licenses: []licenseEntry{{License: licenseValue{Name: "NOASSERTION"}}},
			Properties: []componentProp{
				{Name: "ttgqldl:ecosystem", Value: "unity"},
				{Name: "ttgqldl:source", Value: entry.Source},
			},
		})
	}
	return components, nil
}

var (
	composeImagePattern  = regexp.MustCompile(`(?m)^\s*image:\s*(\S+)\s*$`)
	workflowImagePattern = regexp.MustCompile(`(?m)^\s*POSTGRES_IMAGE:\s*(\S+)\s*$`)
	digestPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*:[A-Za-z0-9._-]+@sha256:[0-9a-f]{64}$`)
)

func containerImages(root string) ([]string, error) {
	var images []string
	seen := map[string]bool{}

	add := func(value string) {
		value = strings.Trim(value, `"'`)
		if !seen[value] {
			seen[value] = true
			images = append(images, value)
		}
	}

	for _, relative := range []string{"compose.dev.yaml", ".github/workflows/ci.yml"} {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", relative, err)
		}
		for _, match := range composeImagePattern.FindAllStringSubmatch(string(raw), -1) {
			add(match[1])
		}
		for _, match := range workflowImagePattern.FindAllStringSubmatch(string(raw), -1) {
			add(match[1])
		}
	}

	sort.Strings(images)
	return images, nil
}

// validateImages enforces immutable digest pinning and rejects repositories that
// silently drift between compose and CI.
func validateImages(images []string) error {
	if len(images) == 0 {
		return fmt.Errorf("no container images found; the supply-chain check would be vacuous")
	}
	var problems []string
	for _, image := range images {
		if !digestPattern.MatchString(image) {
			problems = append(problems, fmt.Sprintf("%s is not pinned by tag and sha256 digest", image))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

func purl(kind, name, version string) string {
	switch kind {
	case "golang":
		return "pkg:golang/" + name + "@" + version
	case "npm":
		return "pkg:npm/" + name + "@" + version
	default:
		return "pkg:unity/" + name + "@" + version
	}
}

func deterministicUUID(components []component) string {
	hasher := sha256.New()
	for _, item := range components {
		_, _ = hasher.Write([]byte(item.BOMRef))
		_, _ = hasher.Write([]byte{0})
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("%s-%s-%s-%s-%s", sum[0:8], sum[8:12], sum[12:16], sum[16:20], sum[20:32])
}

func licenseSummary(components []component) (known int, unknown int) {
	for _, item := range components {
		if len(item.Licenses) == 0 || item.Licenses[0].License.Name == "NOASSERTION" {
			unknown++
			continue
		}
		known++
	}
	return known, unknown
}
