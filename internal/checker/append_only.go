package checker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// errNotInFromTree: 비교 기준(from) 트리에 파일이 존재하지 않음을 나타내는 센티널 에러.
var errNotInFromTree = errors.New("file not in from tree")

// CheckAppendOnly 는 스테이지된 diff 에서 append-only 경로 위반을 검사합니다.
// diffs 는 gitdiff.GetStagedDiff 결과를 커맨드 레벨에서 1회 조회해 전달합니다.
// go-git 으로 HEAD 내용과 staged 내용을 직접 비교합니다.
//
// 허용:
//   - 새 파일 추가 (filename_order 옵션 적용 시 이름 순서도 검사)
//   - 기존 파일 끝에 내용 추가
//
// 차단:
//   - 파일 삭제
//   - 기존 줄 수정·삭제
//   - 파일 중간에 내용 삽입
//   - filename_order=numeric: 기존 파일보다 앞에 오는 이름의 파일 추가
func CheckAppendOnly(ctx context.Context, cfg *config.Config, diffs []gitdiff.FileDiff) ([]string, error) {
	if !cfg.AppendOnly.IsEnabled() {
		return nil, nil
	}

	repo, err := gogit.PlainOpenWithOptions(".", &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}

	// "from" tree 결정: spec.From 이 비어있으면 HEAD 사용.
	spec := gitdiff.CurrentSpec()
	fromRef := spec.From
	if fromRef == "" {
		fromRef = "HEAD"
	}
	tree, err := treeAt(repo, fromRef)
	if err != nil {
		return nil, err
	}

	ignorePatterns := cfg.Exceptions.GlobalIgnore

	var errs []string
	for _, d := range diffs {
		// 취소 시 남은 파일 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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

		if d.IsNew {
			if cfg.AppendOnly.IsFilenameOrderNumeric() {
				if msg := checkFilenameOrder(tree, d.Path, cfg.AppendOnly.Paths); msg != "" {
					errs = append(errs, msg)
				}
			}
			continue
		}

		violation, checkErr := checkFileContent(tree, d.Path)
		if checkErr != nil {
			// from tree 에 없는 파일(rename 등)은 비교 대상이 없으므로 건너뜀
			if errors.Is(checkErr, errNotInFromTree) {
				continue
			}
			return nil, fmt.Errorf("append-only check %s: %w", d.Path, checkErr)
		}
		if violation != "" {
			errs = append(errs, i18n.T(violation, map[string]any{
				"Path": d.Path,
			}))
		}
	}

	return errs, nil
}

// checkFilenameOrder 는 새 파일의 이름이 같은 디렉터리의 기존 파일보다 뒤에 오는지 검사합니다.
// patterns 에 매칭되는 파일만 비교 대상으로 삼습니다.
// 위반 시 i18n 처리된 에러 문자열을 반환합니다.
func checkFilenameOrder(tree *object.Tree, newPath string, patterns []string) string {
	if tree == nil {
		return ""
	}

	newDir := filepath.Dir(newPath)
	newBase := filepath.Base(newPath)

	maxExisting := ""
	_ = tree.Files().ForEach(func(f *object.File) error {
		if filepath.Dir(f.Name) != newDir {
			return nil
		}
		if !pathutil.MatchesAny(f.Name, patterns) {
			return nil
		}
		base := filepath.Base(f.Name)
		if maxExisting == "" || naturalLess(maxExisting, base) {
			maxExisting = base
		}
		return nil
	})

	if maxExisting == "" {
		return ""
	}

	if !naturalLess(maxExisting, newBase) {
		return i18n.T("diff.append_only_filename_order", map[string]any{
			"Path":    newPath,
			"MaxFile": filepath.Join(newDir, maxExisting),
		})
	}
	return ""
}

// treeAt 는 주어진 ref 의 커밋 트리를 반환합니다.
// HEAD 가 없는 빈 저장소이거나 ref 해석에 실패하면 nil, nil 을 반환합니다.
func treeAt(repo *gogit.Repository, ref string) (*object.Tree, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, nil
	}
	commit, err := repo.CommitObject(*hash)
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
		return "", errNotInFromTree
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return "", err
	}

	stagedContent, err := gitdiff.GetStagedContent(path)
	if err != nil {
		return "", err
	}

	// staged 내용이 HEAD 내용으로 시작해야 함 (앞부분 동일 = 삭제·수정·중간삽입 없음)
	if !strings.HasPrefix(stagedContent, headContent) {
		return "diff.append_only_modified", nil
	}

	return "", nil
}

// naturalLess 는 a 가 b 보다 natural sort 순서상 앞에 오면 true 를 반환합니다.
// 숫자 부분은 정수로 비교하고 나머지는 문자열로 비교합니다.
// 예: "9.sql" < "10.sql", "001.sql" < "002.sql"
func naturalLess(a, b string) bool {
	for {
		if b == "" {
			return false
		}
		if a == "" {
			return true
		}

		aIsDigit := unicode.IsDigit(rune(a[0]))
		bIsDigit := unicode.IsDigit(rune(b[0]))

		if aIsDigit && bIsDigit {
			aEnd := numericRunEnd(a)
			bEnd := numericRunEnd(b)
			aNum := parseUint(a[:aEnd])
			bNum := parseUint(b[:bEnd])
			if aNum != bNum {
				return aNum < bNum
			}
			// 숫자 값이 같으면 길이로 구분 (001 < 01 < 1 처럼 다루지 않고 값 우선)
			a = a[aEnd:]
			b = b[bEnd:]
		} else {
			if a[0] != b[0] {
				return a[0] < b[0]
			}
			a = a[1:]
			b = b[1:]
		}
	}
}

func numericRunEnd(s string) int {
	i := 0
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	return i
}

func parseUint(s string) uint64 {
	var n uint64
	for i := 0; i < len(s); i++ {
		n = n*10 + uint64(s[i]-'0')
	}
	return n
}
