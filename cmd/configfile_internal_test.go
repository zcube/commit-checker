package cmd

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigFilePath_DefaultPrefersYaml(t *testing.T) {
	dir := chdirTemp(t)
	yamlPath := filepath.Join(dir, ".commit-checker.yaml")
	ymlPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, ymlPath, "enabled: false\n")
	writeTestFile(t, yamlPath, "enabled: true\n")

	got := resolveConfigFilePath(".commit-checker.yml")
	if got != ".commit-checker.yaml" {
		t.Fatalf("expected yaml to be preferred, got %q", got)
	}
}

func TestResolveConfigFilePath_DefaultFallsBackToYml(t *testing.T) {
	dir := chdirTemp(t)
	ymlPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, ymlPath, "enabled: false\n")

	got := resolveConfigFilePath(".commit-checker.yml")
	if got != ".commit-checker.yml" {
		t.Fatalf("expected yml fallback, got %q", got)
	}
}
