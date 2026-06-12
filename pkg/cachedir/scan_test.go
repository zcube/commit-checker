package cachedir

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo 는 t.TempDir() 안에 git 저장소를 초기화하고 루트 경로를 반환합니다.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	// 커밋에 필요한 최소 사용자 설정 (전역 설정에 의존하지 않음)
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	// 전역 설정의 commit.gpgsign 영향을 받지 않도록 비활성화
	runGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

// runGit 은 repoRoot 에서 git 명령을 실행하고 실패 시 테스트를 중단합니다.
func runGit(t *testing.T, repoRoot string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// ── FindCacheDirsInRepo ──

// TestFindCacheDirsInRepo_EmptyDir: 빈 디렉터리에서는 아무것도 발견하지 않음.
func TestFindCacheDirsInRepo_EmptyDir(t *testing.T) {
	root := t.TempDir()
	if dirs := FindCacheDirsInRepo(root); len(dirs) != 0 {
		t.Errorf("expected no cache dirs in empty dir, got %v", dirs)
	}
}

// TestFindCacheDirsInRepo_NonexistentDir: 존재하지 않는 경로는 빈 결과.
func TestFindCacheDirsInRepo_NonexistentDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "no-such-dir")
	if dirs := FindCacheDirsInRepo(root); len(dirs) != 0 {
		t.Errorf("expected no cache dirs for nonexistent path, got %v", dirs)
	}
}

// TestFindCacheDirsInRepo_SkipsProtectedDirs: .git 등 보호 디렉터리는 진입하지 않음.
func TestFindCacheDirsInRepo_SkipsProtectedDirs(t *testing.T) {
	root := t.TempDir()
	// .git 내부에 유효한 node_modules 구조를 만들어도 발견되면 안 됨
	writeFile(t, filepath.Join(root, ".git", "package.json"), "{}")
	if err := os.MkdirAll(filepath.Join(root, ".git", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if dirs := FindCacheDirsInRepo(root); len(dirs) != 0 {
		t.Errorf("nothing inside .git should be found, got %v", dirs)
	}
}

// TestFindCacheDirsInRepo_SkipsUnknownHiddenDirs: .idea 같은 미등록 hidden 디렉터리는 진입하지 않음.
func TestFindCacheDirsInRepo_SkipsUnknownHiddenDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".idea", "package.json"), "{}")
	if err := os.MkdirAll(filepath.Join(root, ".idea", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if dirs := FindCacheDirsInRepo(root); len(dirs) != 0 {
		t.Errorf("unknown hidden dirs should not be entered, got %v", dirs)
	}
}

// TestFindCacheDirsInRepo_NestedStructure: 깊은 중첩 구조에서도 유효한 캐시 디렉터리를 발견.
func TestFindCacheDirsInRepo_NestedStructure(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "apps", "web", "frontend")
	writeFile(t, filepath.Join(pkg, "package.json"), "{}")
	if err := os.MkdirAll(filepath.Join(pkg, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	dirs := FindCacheDirsInRepo(root)
	if len(dirs) != 1 {
		t.Fatalf("expected exactly 1 cache dir, got %v", dirs)
	}
	if dirs[0] != filepath.Join(pkg, "node_modules") {
		t.Errorf("unexpected path: %s", dirs[0])
	}
}

// TestFindCacheDirsInRepo_KnownNameNotValidated: 이름은 등록되었지만 검증 실패한
// 디렉터리는 결과에서 제외되며 그 하위로도 재귀하지 않음.
func TestFindCacheDirsInRepo_KnownNameNotValidated(t *testing.T) {
	root := t.TempDir()
	// 인디케이터 없는 build/ 아래에 유효한 node_modules 를 둠
	inner := filepath.Join(root, "build", "sub")
	writeFile(t, filepath.Join(inner, "package.json"), "{}")
	if err := os.MkdirAll(filepath.Join(inner, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if dirs := FindCacheDirsInRepo(root); len(dirs) != 0 {
		t.Errorf("recursion should stop at known-name dir even if not validated, got %v", dirs)
	}
}

// ── HasUntrackedEntries ──

// TestHasUntrackedEntries_Untracked: 미추적 파일이 있으면 true.
func TestHasUntrackedEntries_Untracked(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "node_modules")
	writeFile(t, filepath.Join(target, "lib", "index.js"), "")

	if !HasUntrackedEntries(root, target) {
		t.Error("dir with untracked files should report true")
	}
}

// TestHasUntrackedEntries_AllTracked: 모든 파일이 커밋되어 있으면 false.
func TestHasUntrackedEntries_AllTracked(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "node_modules")
	writeFile(t, filepath.Join(target, "lib", "index.js"), "")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")

	if HasUntrackedEntries(root, target) {
		t.Error("fully committed dir should report false")
	}
}

// TestHasUntrackedEntries_NotARepo: git 저장소가 아니면 false (에러 경로).
func TestHasUntrackedEntries_NotARepo(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "node_modules")
	writeFile(t, filepath.Join(target, "index.js"), "")

	if HasUntrackedEntries(root, target) {
		t.Error("non-git dir should report false")
	}
}

// ── ListUntrackedEntries ──

// TestListUntrackedEntries_Mixed: 미추적 파일만 절대 경로로 반환하고 추적 파일은 제외.
func TestListUntrackedEntries_Mixed(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(target, "tracked.js"), "")
	runGit(t, root, "add", "dist/tracked.js")
	runGit(t, root, "commit", "-q", "-m", "init")
	writeFile(t, filepath.Join(target, "untracked.js"), "")
	// targetDir 밖의 미추적 파일은 결과에 포함되지 않아야 함
	writeFile(t, filepath.Join(root, "outside.txt"), "")

	entries, err := ListUntrackedEntries(root, target)
	if err != nil {
		t.Fatalf("ListUntrackedEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 untracked entry, got %v", entries)
	}
	want := filepath.Join(target, "untracked.js")
	if entries[0] != want {
		t.Errorf("expected %s, got %s", want, entries[0])
	}
}

// TestListUntrackedEntries_AllTracked: 모두 커밋된 디렉터리는 빈 결과.
func TestListUntrackedEntries_AllTracked(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(target, "a.js"), "")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")

	entries, err := ListUntrackedEntries(root, target)
	if err != nil {
		t.Fatalf("ListUntrackedEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no untracked entries, got %v", entries)
	}
}

// TestListUntrackedEntries_NotARepo: git 저장소가 아니면 에러 반환.
func TestListUntrackedEntries_NotARepo(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(target, "a.js"), "")

	if _, err := ListUntrackedEntries(root, target); err == nil {
		t.Error("expected error for non-git dir")
	}
}

// ── ListTrackedEntries ──

// TestListTrackedEntries: 추적 중인 파일만 절대 경로로 반환.
func TestListTrackedEntries(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(target, "tracked.js"), "")
	runGit(t, root, "add", "dist/tracked.js")
	runGit(t, root, "commit", "-q", "-m", "init")
	writeFile(t, filepath.Join(target, "untracked.js"), "")

	entries, err := ListTrackedEntries(root, target)
	if err != nil {
		t.Fatalf("ListTrackedEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 tracked entry, got %v", entries)
	}
	want := filepath.Join(target, "tracked.js")
	if entries[0] != want {
		t.Errorf("expected %s, got %s", want, entries[0])
	}
}

// TestListTrackedEntries_Empty: 추적 파일이 없으면 빈 결과.
func TestListTrackedEntries_Empty(t *testing.T) {
	root := initGitRepo(t)
	target := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(target, "untracked.js"), "")

	entries, err := ListTrackedEntries(root, target)
	if err != nil {
		t.Fatalf("ListTrackedEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no tracked entries, got %v", entries)
	}
}

// TestListTrackedEntries_NotARepo: git 저장소가 아니면 에러 반환.
func TestListTrackedEntries_NotARepo(t *testing.T) {
	root := t.TempDir()
	if _, err := ListTrackedEntries(root, root); err == nil {
		t.Error("expected error for non-git dir")
	}
}

// ── GetDirSize ──

// TestGetDirSize: 파일이 있는 디렉터리는 0 보다 큰 크기를 반환.
func TestGetDirSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "data.bin"), "hello world")
	if size := GetDirSize(dir); size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

// TestGetDirSize_Nonexistent: 존재하지 않는 경로는 0 반환 (에러 경로).
func TestGetDirSize_Nonexistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir")
	if size := GetDirSize(path); size != 0 {
		t.Errorf("expected 0 for nonexistent path, got %d", size)
	}
}

// ── FormatBytes ──

// TestFormatBytes: 단위별 포맷 경계 케이스 검증.
func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
		{2 * 1024 * 1024 * 1024 * 1024, "2.0 TB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── FindRepoRoot ──

// TestFindRepoRoot_AtRoot: .git 이 있는 디렉터리 자체에서 시작.
func TestFindRepoRoot_AtRoot(t *testing.T) {
	root := initGitRepo(t)
	got, err := FindRepoRoot(root)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

// TestFindRepoRoot_FromNestedDir: 하위 디렉터리에서 시작해도 루트를 찾음.
func TestFindRepoRoot_FromNestedDir(t *testing.T) {
	root := initGitRepo(t)
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

// TestFindRepoRoot_GitFile: 워크트리처럼 .git 이 파일이어도 루트로 인식.
func TestFindRepoRoot_GitFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git"), "gitdir: /somewhere/else\n")
	got, err := FindRepoRoot(root)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

// TestFindRepoRoot_NotARepo: git 저장소가 아니면 에러 반환.
func TestFindRepoRoot_NotARepo(t *testing.T) {
	// 시스템 임시 디렉터리는 상위로 올라가도 .git 이 없다고 가정
	root := t.TempDir()
	if _, err := FindRepoRoot(root); err == nil {
		t.Error("expected error outside a git repository")
	}
}
