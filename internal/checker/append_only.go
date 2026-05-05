package checker

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"

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
//   - 새 파일 추가 (filename_order 옵션 적용 시 이름 순서도 검사)
//   - 기존 파일 끝에 내용 추가
//
// 차단:
//   - 파일 삭제
//   - 기존 줄 수정·삭제
//   - 파일 중간에 내용 삽입
//   - filename_order=numeric: 기존 파일보다 앞에 오는 이름의 파일 추가
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

	tree, err := headTree(repo)
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

// headTree 는 HEAD 커밋의 트리를 반환합니다.
// HEAD 가 없는 빈 저장소면 nil, nil 을 반환합니다.
func headTree(repo *gogit.Repository) (*object.Tree, error) {
	ref, err := repo.Head()
	if err != nil {
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
