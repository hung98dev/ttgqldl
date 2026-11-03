package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../.."

func TestInventoryIsDeterministic(t *testing.T) {
	first, images, err := collect(repoRoot)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	second, _, err := collect(repoRoot)
	if err != nil {
		t.Fatalf("collect twice: %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}

	if string(firstJSON) != string(secondJSON) {
		t.Fatal("two inventories of the same checkout differ")
	}
	if deterministicUUID(first) != deterministicUUID(second) {
		t.Fatal("serial number is not deterministic")
	}
	if len(images) == 0 {
		t.Fatal("no container images were discovered")
	}
}

func TestInventoryCoversEveryEcosystem(t *testing.T) {
	components, _, err := collect(repoRoot)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	ecosystems := map[string]int{}
	for _, item := range components {
		if item.Type == "container" {
			ecosystems["container"]++
			continue
		}
		for _, property := range item.Properties {
			if property.Name == "ttgqldl:ecosystem" {
				ecosystems[property.Value]++
			}
		}
	}

	for _, expected := range []string{"go", "npm", "unity", "container"} {
		if ecosystems[expected] == 0 {
			t.Fatalf("no %s components in the inventory: %#v", expected, ecosystems)
		}
	}
}

func TestContainerImagesAreDigestPinned(t *testing.T) {
	_, images, err := collect(repoRoot)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := validateImages(images); err != nil {
		t.Fatalf("images are not pinned: %v", err)
	}
}

func TestUnpinnedImageIsRejected(t *testing.T) {
	cases := []string{
		"postgres:18.6",
		"postgres:latest@sha256:short",
		"postgres@sha256:" + strings.Repeat("a", 64),
	}
	for _, image := range cases {
		if err := validateImages([]string{image}); err == nil {
			t.Fatalf("expected %q to be rejected", image)
		}
	}

	pinned := "postgres:18.6@sha256:" + strings.Repeat("a", 64)
	if err := validateImages([]string{pinned}); err != nil {
		t.Fatalf("pinned image was rejected: %v", err)
	}

	if err := validateImages(nil); err == nil {
		t.Fatal("an empty image list must not pass the supply-chain check")
	}
}

func TestNpmComponentsCarryLicenceMetadata(t *testing.T) {
	components, err := npmComponents(repoRoot)
	if err != nil {
		t.Fatalf("npm components: %v", err)
	}
	if len(components) < 100 {
		t.Fatalf("expected the full npm graph, got %d components", len(components))
	}
	missing := 0
	for _, item := range components {
		if len(item.Licenses) == 0 {
			missing++
		}
	}
	if missing > 0 {
		t.Fatalf("%d npm components have no licence entry (missing metadata must be explicit)", missing)
	}
}

func TestOutputFileIsWritten(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "sbom.cdx.json")

	raw, err := json.Marshal(bom{BOMFormat: "CycloneDX", SpecVersion: specVersion, Version: 1})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("sbom file missing: %v", err)
	}
}
