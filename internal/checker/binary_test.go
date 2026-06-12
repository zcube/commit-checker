package checker_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func binaryCheckConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.BinaryFile.Enabled = &t
	return cfg
}

// TestCheckBinaryFiles_Disabled: binary check disabled returns no errors.
func TestCheckBinaryFiles_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	// Stage a binary file
	writeBinaryFile(t, dir, "app.exe")
	gitMust(t, dir, "git", "add", "app.exe")

	f := false
	cfg := &config.Config{}
	cfg.BinaryFile.Enabled = &f

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled binary check should return no errors, got: %v", errs)
	}
}

// TestCheckBinaryFiles_DetectsBinary: binary file in staged changes is flagged.
func TestCheckBinaryFiles_DetectsBinary(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "app.exe")
	gitMust(t, dir, "git", "add", "app.exe")

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for binary file, got none")
	}
	found := false
	for _, e := range errs {
		if findSubstr(e, "app.exe") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error mentioning app.exe, got: %v", errs)
	}
}

// TestCheckBinaryFiles_TextFilePass: text files are not flagged.
func TestCheckBinaryFiles_TextFilePass(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "readme.txt", "this is a text file\n")
	gitMust(t, dir, "git", "add", "readme.txt")

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("text file should not be flagged, got: %v", errs)
	}
}

// TestCheckBinaryFiles_IgnoreFiles: binary files matching ignore_files are skipped.
func TestCheckBinaryFiles_IgnoreFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "assets/logo.png")
	gitMust(t, dir, "git", "add", "assets/logo.png")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.IgnoreFiles = []string{"**/*.png"}

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ignored binary file should not be flagged, got: %v", errs)
	}
}

// TestCheckBinaryFiles_GlobalIgnore: binary files matching global_ignore are skipped.
func TestCheckBinaryFiles_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "vendor/lib.so")
	gitMust(t, dir, "git", "add", "vendor/lib.so")

	cfg := binaryCheckConfig()
	cfg.Exceptions.GlobalIgnore = []string{"vendor/**"}

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("globally ignored binary file should not be flagged, got: %v", errs)
	}
}

// TestCheckBinaryFiles_MultipleBinaries: multiple binary files are all detected.
func TestCheckBinaryFiles_MultipleBinaries(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "bin/server")
	writeBinaryFile(t, dir, "bin/client")
	gitMust(t, dir, "git", "add", "bin/")

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 2 {
		t.Errorf("expected 2 errors for 2 binary files, got %d: %v", len(errs), errs)
	}
}

// TestCheckBinaryFiles_MixedIgnore: some binaries ignored, some flagged.
func TestCheckBinaryFiles_MixedIgnore(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "icon.png")
	writeBinaryFile(t, dir, "malware.exe")
	gitMust(t, dir, "git", "add", ".")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.IgnoreFiles = []string{"**/*.png"}

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error (only malware.exe), got %d: %v", len(errs), errs)
	}
	if len(errs) == 1 && !findSubstr(errs[0], "malware.exe") {
		t.Errorf("expected error for malware.exe, got: %s", errs[0])
	}
}

// TestCheckBinaryFiles_DefaultEnabled: default config (nil Enabled) should enable check.
func TestCheckBinaryFiles_DefaultEnabled(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "app")
	gitMust(t, dir, "git", "add", "app")

	cfg := &config.Config{} // Enabled is nil, default should be true

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("default config should detect binary files, got none")
	}
}

// writeBinaryFile creates a file with binary content (ELF-like header).
func writeBinaryFile(t *testing.T, dir, relPath string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	// ELF magic bytes followed by null bytes — git will classify this as binary.
	content := []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(full, content, 0755); err != nil {
		t.Fatal(err)
	}
}

// TestCheckBinaryFiles_NoStagedChanges: no staged files means no errors.
func TestCheckBinaryFiles_NoStagedChanges(t *testing.T) {
	_ = newGitRepo(t)

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("no staged changes should produce no errors, got: %v", errs)
	}
}

// TestCheckBinaryFiles_ConfigLoad: binary_file config loads from YAML.
func TestCheckBinaryFiles_ConfigLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	cfgContent := `
binary_file:
  enabled: true
  ignore_files:
    - "**/*.png"
    - "**/*.jpg"
    - "assets/**"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.BinaryFile.IsEnabled() {
		t.Error("expected binary_file.enabled=true")
	}
	if len(cfg.BinaryFile.IgnoreFiles) != 3 {
		t.Errorf("expected 3 ignore_files, got %v", cfg.BinaryFile.IgnoreFiles)
	}
}

// TestCheckBinaryFiles_ConfigLoadDisabled: binary_file can be disabled via YAML.
func TestCheckBinaryFiles_ConfigLoadDisabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	cfgContent := `
binary_file:
  enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BinaryFile.IsEnabled() {
		t.Error("expected binary_file.enabled=false")
	}
}

// TestCheckBinaryFiles_ImageDefaultAllow: 기본적으로 이미지는 허가됨.
func TestCheckBinaryFiles_ImageDefaultAllow(t *testing.T) {
	dir := newGitRepo(t)
	full := filepath.Join(dir, "photo.png")
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(full, png, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "photo.png")

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("PNG should be allowed by default, got: %v", errs)
	}
}

// TestCheckBinaryFiles_ImageRuleBlock: 이미지를 명시적으로 block 정책으로 설정 → 차단.
func TestCheckBinaryFiles_ImageRuleBlock(t *testing.T) {
	dir := newGitRepo(t)
	full := filepath.Join(dir, "photo.png")
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(full, png, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "photo.png")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.Rules = []config.BinaryFilePolicyRule{
		{Extensions: []string{".png"}, Policy: "block"},
	}

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("PNG should be blocked when policy is 'block'")
	}
}

// TestCheckBinaryFiles_LfsPolicy_NotTracked: lfs 정책이지만 LFS 추적되지 않음 → 차단.
func TestCheckBinaryFiles_LfsPolicy_NotTracked(t *testing.T) {
	dir := newGitRepo(t)
	full := filepath.Join(dir, "movie.mp4")
	if err := os.WriteFile(full, []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 0, 0, 0, 0}, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "movie.mp4")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.Rules = []config.BinaryFilePolicyRule{
		{Extensions: []string{".mp4"}, Policy: "lfs"},
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("non-LFS-tracked binary should fail under lfs policy")
	}
	if len(errs) > 0 && !findSubstr(errs[0], "LFS") && !findSubstr(errs[0], "lfs") {
		t.Errorf("expected LFS-related message, got: %s", errs[0])
	}
}

// TestCheckBinaryFiles_LfsPolicy_Tracked: lfs 정책 + .gitattributes 에 filter=lfs → 허가.
func TestCheckBinaryFiles_LfsPolicy_Tracked(t *testing.T) {
	dir := newGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.psd filter=lfs diff=lfs merge=lfs -text\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", ".gitattributes")

	full := filepath.Join(dir, "design.psd")
	if err := os.WriteFile(full, []byte{'8', 'B', 'P', 'S', 0, 1, 0, 0, 0, 0, 0, 0}, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "design.psd")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.Rules = []config.BinaryFilePolicyRule{
		{Extensions: []string{".psd"}, Policy: "lfs"},
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("LFS-tracked file under lfs policy should pass, got: %v", errs)
	}
}

// TestCheckBinaryFiles_DefaultPolicyAllow: default_policy=allow → 모든 비매칭 바이너리 허가.
func TestCheckBinaryFiles_DefaultPolicyAllow(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "app.exe")
	gitMust(t, dir, "git", "add", "app.exe")

	cfg := binaryCheckConfig()
	cfg.BinaryFile.DefaultPolicy = "allow"

	errs, err := checker.CheckBinaryFiles(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("default_policy=allow should permit binaries, got: %v", errs)
	}
}

// TestCheckBinaryFiles_DeletedBinaryIgnored: deleted binary files should not be flagged.
func TestCheckBinaryFiles_DeletedBinaryIgnored(t *testing.T) {
	dir := newGitRepo(t)
	// Commit a binary file first
	writeBinaryFile(t, dir, "old.bin")
	gitMust(t, dir, "git", "add", "old.bin")
	gitMust(t, dir, "git", "commit", "-m", "add binary")

	// Delete the binary file and stage the deletion
	_ = os.Remove(filepath.Join(dir, "old.bin"))
	cmd := exec.Command("git", "add", "old.bin")
	cmd.Dir = dir
	_ = cmd.Run()

	errs, err := checker.CheckBinaryFiles(t.Context(), binaryCheckConfig())
	if err != nil {
		t.Fatalf("CheckBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("deleted binary file should not be flagged, got: %v", errs)
	}
}
