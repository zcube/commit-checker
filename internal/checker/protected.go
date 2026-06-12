package checker

import (
	"context"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckProtectedPaths 는 스테이지된 diff 에서 보호 경로(protected_paths) 위반을 검사합니다.
// diffs 는 gitdiff.GetStagedDiff 결과를 커맨드 레벨에서 1회 조회해 전달합니다.
//
// append-only 보다 강한 "완전 동결" 정책으로, 보호 경로에 매칭되는 파일은
// 추가·수정·삭제 등 어떤 staged 변경도 허용되지 않습니다.
// exceptions.global_ignore 에 매칭되는 파일은 검사에서 제외합니다.
func CheckProtectedPaths(ctx context.Context, cfg *config.Config, diffs []gitdiff.FileDiff) ([]string, error) {
	if !cfg.ProtectedPaths.IsEnabled() {
		return nil, nil
	}

	ignorePatterns := cfg.Exceptions.GlobalIgnore

	var errs []string
	for _, d := range diffs {
		// 취소 시 남은 파일 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !pathutil.MatchesAny(d.Path, cfg.ProtectedPaths.Paths) {
			continue
		}
		if pathutil.MatchesAny(d.Path, ignorePatterns) {
			continue
		}

		// 변경 유형(추가/삭제/수정)별로 구분된 메시지 출력
		var key string
		switch {
		case d.IsNew:
			key = "diff.protected_path_added"
		case d.IsDeleted:
			key = "diff.protected_path_deleted"
		default:
			key = "diff.protected_path_modified"
		}
		errs = append(errs, i18n.T(key, map[string]any{
			"Path": d.Path,
		}))
	}

	return errs, nil
}
