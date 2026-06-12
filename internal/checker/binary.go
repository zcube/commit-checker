package checker

import (
	"context"
	"os/exec"
	"strings"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckBinaryFiles: 스테이지된 diff에서 바이너리 파일을 검사.
// 위반 없으면 빈 슬라이스 반환.
func CheckBinaryFiles(ctx context.Context, cfg *config.Config) ([]string, error) {
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
		// 취소 시 남은 파일 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}
		if msg := evaluateBinaryPolicy(&cfg.BinaryFile, path); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs, nil
}

// evaluateBinaryPolicy 는 path 에 적용할 정책에 따라 i18n 처리된 에러 메시지를 반환합니다.
// allow 또는 lfs(LFS-tracked) 인 경우 빈 문자열을 반환합니다.
func evaluateBinaryPolicy(cfg *config.BinaryFileConfig, path string) string {
	policy := cfg.PolicyFor(path)
	switch policy {
	case "allow":
		return ""
	case "lfs":
		if isLFSTracked(path) {
			return ""
		}
		return i18n.T("diff.binary_file_lfs_required", map[string]any{"Path": path})
	case "block":
		fallthrough
	default:
		return i18n.T("diff.binary_file_error", map[string]any{"Path": path})
	}
}

// isLFSTracked 는 path 가 git LFS filter 로 추적되는지 확인합니다.
// `.gitattributes` 에 `path filter=lfs` 가 등록되어 있으면 true.
func isLFSTracked(path string) bool {
	out, err := exec.Command("git", "check-attr", "-z", "filter", "--", path).Output()
	if err != nil {
		return false
	}
	// check-attr -z 출력 형식: <path>\0filter\0<value>\0
	parts := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	return len(parts) >= 3 && parts[2] == "lfs"
}

// getStagedBinaryFiles: git이 바이너리로 판별한 스테이지된 파일 목록 반환.
// git diff --staged --numstat에서 바이너리 파일은 "-\t-\tpath" 형식으로 표시됨.
func getStagedBinaryFiles() ([]string, error) {
	// -z: NUL 구분 출력으로 비ASCII 경로의 C-스타일 인용(core.quotePath)을 회피.
	cmd := exec.Command("git", "diff", "--staged", "--numstat", "-z", "--diff-filter=d")
	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return nil, err
		}
	}

	// numstat -z 레코드: "added\tdeleted\t<path>" (바이너리는 added/deleted 가 "-").
	// rename 은 경로가 비고 뒤따르는 두 NUL 필드가 (이전, 새) 경로.
	fields := gitdiff.SplitNullSeparated(out)
	var binaries []string
	for i := 0; i < len(fields); i++ {
		parts := strings.SplitN(fields[i], "\t", 3)
		if len(parts) != 3 {
			continue
		}
		path := parts[2]
		if path == "" && i+2 < len(fields) {
			path = fields[i+2] // rename: 새 경로 사용
			i += 2
		}
		if parts[0] == "-" && parts[1] == "-" && path != "" {
			binaries = append(binaries, path)
		}
	}
	return binaries, nil
}
