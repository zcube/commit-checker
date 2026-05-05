package checker

import (
	"errors"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckAppendOnly 는 스테이지된 diff 에서 append-only 경로 위반을 검사합니다.
// go-git 으로 HEAD 내용과 staged 내용을 직접 비교합니다.
//
// 허용:
//   - 새 파일 추가
//   - 기존 파일 끝에 내용 추가
//
// 차단:
//   - 파일 삭제
//   - 기존 줄 수정·삭제
//   - 파일 중간에 내용 삽입
func CheckAppendOnly(cfg *config.Config) ([]string, error) {
	if !cfg.AppendOnly.IsEnabled() {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	repo, err := gogit.PlainOpenWithOptions(".", &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}

	headTree, err := headTree(repo)
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

		violation, checkErr := checkFileContent(headTree, d.Path)
		if checkErr != nil {
			// HEAD에 파일이 없으면 신규 파일로 간주
			continue
		}
		if violation != "" {
			errs = append(errs, i18n.T(violation, map[string]any{
				"Path": d.Path,
			}))
		}
	}

	return errs, nil
}

// headTree 는 HEAD 커밋의 트리를 반환합니다.
// HEAD 가 없는 빈 저장소면 nil, nil 을 반환합니다.
func headTree(repo *gogit.Repository) (*object.Tree, error) {
	ref, err := repo.Head()
	if err != nil {
		// 빈 저장소 (첫 커밋 전)
		return nil, nil
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	return commit.Tree()
}

// checkFileContent 는 HEAD 내용과 staged 내용을 비교하여 위반 i18n 키를 반환합니다.
// 위반 없으면 "" 를 반환합니다.
func checkFileContent(headTree *object.Tree, path string) (string, error) {
	if headTree == nil {
		return "", nil
	}

	headFile, err := headTree.File(path)
	if err != nil {
		// HEAD 에 파일 없음 → 신규 파일
		return "", errors.New("not in HEAD")
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return "", err
	}

	stagedContent, err := gitdiff.GetStagedContent(path)
	if err != nil {
		return "", err
	}

	// staged 내용이 HEAD 내용으로 시작해야 함 (앞부분 동일 = 삭제·수정 없음)
	if !strings.HasPrefix(stagedContent, headContent) {
		return "diff.append_only_modified", nil
	}

	// staged 에 추가된 내용이 있다면 HEAD 내용 바로 뒤에 붙어야 함 (중간 삽입 없음)
	// HasPrefix 통과 = 앞부분 동일 = 수정·중간 삽입 모두 차단됨
	return "", nil
}
