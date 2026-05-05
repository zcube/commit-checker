package checker

import (
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckAppendOnly 는 스테이지된 diff 에서 append-only 경로 위반을 검사합니다.
// 위반 유형:
//   - 파일 삭제
//   - 기존 줄 수정·삭제 (- 줄 존재)
//   - 파일 끝이 아닌 중간에 줄 삽입
func CheckAppendOnly(cfg *config.Config) ([]string, error) {
	if !cfg.AppendOnly.IsEnabled() {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	ignorePatterns := cfg.Exceptions.GlobalIgnore

	var errs []string
	for _, d := range diffs {
		if !pathutil.MatchesAny(d.Path, cfg.AppendOnly.Paths) {
			continue
		}
		if pathutil.MatchesAny(d.Path, ignorePatterns) {
			continue
		}

		if d.IsDeleted {
			errs = append(errs, i18n.T("diff.append_only_deleted", map[string]any{
				"Path": d.Path,
			}))
			continue
		}

		// 신규 파일은 전체가 추가이므로 허용
		if d.IsNew {
			continue
		}

		if d.HasRemovedLines {
			errs = append(errs, i18n.T("diff.append_only_modified", map[string]any{
				"Path": d.Path,
			}))
			continue
		}

		if d.HasMiddleInsert {
			errs = append(errs, i18n.T("diff.append_only_middle_insert", map[string]any{
				"Path": d.Path,
			}))
		}
	}

	return errs, nil
}
