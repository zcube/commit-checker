package cachedir

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile 은 path 에 빈 또는 지정된 내용을 가진 파일을 생성합니다.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestIsCacheDir_NodeModules: 부모에 package.json 있으면 node_modules 인정.
func TestIsCacheDir_NodeModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), "{}")
	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsCacheDir(nm) {
		t.Error("node_modules with package.json parent should be valid cache dir")
	}
}

// TestIsCacheDir_NodeModulesWithoutIndicator: 부모에 package.json 없으면 false.
func TestIsCacheDir_NodeModulesWithoutIndicator(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if IsCacheDir(nm) {
		t.Error("node_modules without package.json should not be cache dir")
	}
}

// TestIsCacheDir_BuildWithCargo: 부모에 Cargo.toml → build 인정.
func TestIsCacheDir_BuildWithCargo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Cargo.toml"), "")
	b := filepath.Join(dir, "build")
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsCacheDir(b) {
		t.Error("build/ with Cargo.toml parent should be valid")
	}
}

// TestIsCacheDir_BuildCMakeOutOfSource: build/ 안 CMakeCache.txt → 인정.
func TestIsCacheDir_BuildCMakeOutOfSource(t *testing.T) {
	dir := t.TempDir()
	b := filepath.Join(dir, "build")
	writeFile(t, filepath.Join(b, "CMakeCache.txt"), "")
	if !IsCacheDir(b) {
		t.Error("build/ with CMakeCache.txt should be valid (out-of-source CMake)")
	}
}

// TestIsCacheDir_UnknownName: 이름이 등록되지 않으면 false.
func TestIsCacheDir_UnknownName(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "myrandomdir")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if IsCacheDir(other) {
		t.Error("unknown name should not be cache dir")
	}
}

// TestIsPythonVirtualenv: pyvenv.cfg 가 있으면 인정.
func TestIsPythonVirtualenv(t *testing.T) {
	dir := t.TempDir()
	v := filepath.Join(dir, "myenv")
	writeFile(t, filepath.Join(v, "pyvenv.cfg"), "home = /usr/bin\n")
	if !IsPythonVirtualenv(v) {
		t.Error("dir with pyvenv.cfg should be virtualenv")
	}
}

// TestFindCacheDirAncestor_Direct: 직계 부모가 캐시 디렉터리.
func TestFindCacheDirAncestor_Direct(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	file := filepath.Join(root, "node_modules", "lodash", "index.js")
	writeFile(t, file, "")

	ancestor, ok := FindCacheDirAncestor(root, file)
	if !ok {
		t.Fatal("expected to find cache dir ancestor")
	}
	if filepath.Base(ancestor) != "node_modules" {
		t.Errorf("expected node_modules, got %s", ancestor)
	}
}

// TestFindCacheDirAncestor_NotFound: 캐시 디렉터리 안에 없으면 false.
func TestFindCacheDirAncestor_NotFound(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "src", "main.go")
	writeFile(t, file, "package main")

	if _, ok := FindCacheDirAncestor(root, file); ok {
		t.Error("source file outside cache dir should not have ancestor")
	}
}

// TestFindCacheDirsInRepo: 검증된 디렉터리만 발견.
func TestFindCacheDirsInRepo(t *testing.T) {
	root := t.TempDir()
	// 유효한 node_modules
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 유효하지 않은 build (인디케이터 없음)
	if err := os.MkdirAll(filepath.Join(root, "subdir", "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 유효한 .venv
	writeFile(t, filepath.Join(root, "myenv", "pyvenv.cfg"), "home = /usr/bin\n")

	dirs := FindCacheDirsInRepo(root)

	foundNodeModules := false
	foundVenv := false
	foundInvalidBuild := false
	for _, d := range dirs {
		switch filepath.Base(d) {
		case "node_modules":
			foundNodeModules = true
		case "myenv":
			foundVenv = true
		case "build":
			foundInvalidBuild = true
		}
	}
	if !foundNodeModules {
		t.Error("expected node_modules to be found")
	}
	if !foundVenv {
		t.Error("expected myenv (virtualenv) to be found")
	}
	if foundInvalidBuild {
		t.Error("build/ without indicator should not be found")
	}
}

// TestFindCacheDirsInRepo_StopsAtKnown: node_modules 안의 중첩 dist 는 발견하지 않음.
func TestFindCacheDirsInRepo_StopsAtKnown(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	// node_modules/some-pkg/dist
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "some-pkg", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "node_modules", "some-pkg", "package.json"), "{}")

	dirs := FindCacheDirsInRepo(root)
	for _, d := range dirs {
		if filepath.Base(d) == "dist" {
			t.Errorf("dist inside node_modules should not be found: %s", d)
		}
	}
}
