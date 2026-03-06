package checker

import (
	"os/exec"
	"strings"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckBinaryFiles: 스테이지된 diff에서 바이너리 파일을 검사.
// 위반 없으면 빈 슬라이스 반환.
func CheckBinaryFiles(cfg *config.Config) ([]string, error) {
	if !cfg.BinaryFile.IsEnabled() {
		return nil, nil
	}

	files, err := getStagedBinaryFiles()
	if err != nil {
		return nil, err
	}

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore, cfg.BinaryFile.IgnoreFiles...)

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}
		errs = append(errs, i18n.T("diff.binary_file_error", map[string]interface{}{
			"Path": path,
		}))
	}
	return errs, nil
}

// getStagedBinaryFiles: git이 바이너리로 판별한 스테이지된 파일 목록 반환.
// git diff --staged --numstat에서 바이너리 파일은 "-\t-\tpath" 형식으로 표시됨.
func getStagedBinaryFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--numstat", "--diff-filter=d")
	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return nil, err
		}
	}

	var binaries []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// 바이너리 파일 형식: -\t-\tfilename
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 && parts[0] == "-" && parts[1] == "-" {
			binaries = append(binaries, parts[2])
		}
	}
	return binaries, nil
}
