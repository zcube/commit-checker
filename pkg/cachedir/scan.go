package cachedir

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindCacheDirsInRepo 는 repoRoot 안에서 검증된 캐시/빌드 산출물 디렉터리들을 찾아 반환합니다.
// 동작 규칙:
//   - 디렉터리 이름이 dirValidators 에 등록되어 있어야 함
//   - 해당 이름의 validator 중 하나가 통과해야 함
//   - 인지된 디렉터리(검증 여부 무관)에 도달하면 그 하위로 재귀하지 않음
//     (예: node_modules/some-pkg/dist 같은 중첩 발견 방지)
//   - .git, .svn 등 보호 디렉터리는 진입하지 않음
//   - pyvenv.cfg 가 있으면 이름과 무관하게 virtualenv 로 인식
func FindCacheDirsInRepo(repoRoot string) []string {
	var found []string
	var visit func(dir string)
	visit = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			path := filepath.Join(dir, name)

			if ProtectedDirNames[name] {
				continue
			}

			// pyvenv.cfg 기반 virtualenv 감지 (이름 무관)
			if IsPythonVirtualenv(path) {
				found = append(found, path)
				continue
			}

			// dirValidators 에 없는 hidden 디렉터리는 진입하지 않음 (.idea, .vscode 등)
			if strings.HasPrefix(name, ".") {
				if _, known := dirValidators[name]; !known {
					continue
				}
			}

			if _, known := dirValidators[name]; known {
				if IsCacheDir(path) {
					found = append(found, path)
				}
				continue
			}

			visit(path)
		}
	}
	visit(repoRoot)
	return found
}

// HasUntrackedEntries 는 targetDir 에 git 미추적 파일이 하나라도 있으면 true 를 반환합니다.
// 모든 파일이 커밋되어 있으면 false 입니다.
func HasUntrackedEntries(repoRoot, targetDir string) bool {
	rel, err := filepath.Rel(repoRoot, targetDir)
	if err != nil {
		return false
	}
	out, err := exec.Command(
		"git", "-C", repoRoot,
		"ls-files", "--others", "--directory", "-z", rel,
	).Output()
	if err != nil {
		return false
	}
	for _, entry := range strings.Split(string(out), "\x00") {
		if strings.TrimRight(entry, "/") != "" {
			return true
		}
	}
	return false
}

// ListUntrackedEntries 는 targetDir 안에서 git 에 미추적인 파일/디렉터리를 절대 경로로 반환합니다.
// `git ls-files --others` 는 인덱스 항목을 포함하지 않으므로 git add 된 파일은 자동 보존됩니다.
func ListUntrackedEntries(repoRoot, targetDir string) ([]string, error) {
	rel, err := filepath.Rel(repoRoot, targetDir)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command(
		"git", "-C", repoRoot,
		"ls-files", "--others", "--directory", "-z", rel,
	).Output()
	if err != nil {
		return nil, err
	}

	prefix := rel + string(os.PathSeparator)
	var entries []string
	for _, entry := range strings.Split(string(out), "\x00") {
		entry = strings.TrimRight(entry, "/")
		if entry == "" {
			continue
		}
		if entry != rel && !strings.HasPrefix(entry, prefix) {
			continue
		}
		entries = append(entries, filepath.Join(repoRoot, entry))
	}
	return entries, nil
}

// ListTrackedEntries 는 targetDir 안에서 git 에 추적 중인 파일을 절대 경로로 반환합니다.
// 캐시/빌드 디렉터리 안에 이미 커밋된 파일이 있는지 확인하는 용도입니다.
func ListTrackedEntries(repoRoot, targetDir string) ([]string, error) {
	rel, err := filepath.Rel(repoRoot, targetDir)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command(
		"git", "-C", repoRoot,
		"ls-files", "-z", "--", rel,
	).Output()
	if err != nil {
		return nil, err
	}

	var entries []string
	for _, entry := range strings.Split(string(out), "\x00") {
		if entry == "" {
			continue
		}
		entries = append(entries, filepath.Join(repoRoot, entry))
	}
	return entries, nil
}

// GetDirSize 는 du -sk 로 path 의 디스크 사용량(바이트)을 반환합니다.
// 큰 트리에서 filepath.Walk 보다 빠릅니다. 실패 시 0 을 반환합니다.
func GetDirSize(path string) int64 {
	out, err := exec.Command("du", "-sk", path).Output()
	if err != nil {
		return 0
	}
	var kb int64
	_, _ = fmt.Sscanf(string(out), "%d", &kb)
	return kb * 1024
}

// FormatBytes 는 바이트 수를 사람이 읽기 쉬운 단위(KB, MB, GB ...)로 포맷합니다.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FindRepoRoot 는 현재 디렉터리(또는 startDir)에서 시작해 git 저장소 루트를 찾습니다.
// .git 디렉터리/파일이 발견된 디렉터리를 절대 경로로 반환합니다.
func FindRepoRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repository: %s", startDir)
		}
		dir = parent
	}
}
