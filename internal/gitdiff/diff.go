package gitdiff

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// RefWorktree 는 작업 트리(working tree) 를 가리키는 특수 ref 값입니다.
// git 의 working tree 는 디스크 위 실제 파일 상태를 의미합니다.
const RefWorktree = "worktree"

// Spec 은 git diff 비교 대상을 지정합니다.
//
// 동작 매핑:
//   - From=""     To=""        → HEAD ↔ index (스테이지된 diff, 기본값)
//   - From=""     To=worktree  → index ↔ working tree (커밋 안 한 변경)
//   - From=ref    To=worktree  → ref ↔ working tree
//   - From=ref    To=""        → ref ↔ HEAD
//   - From=ref    To=ref2      → ref ↔ ref2
type Spec struct {
	From string
	To   string
}

// IsDefault 는 기본 staged 모드 (HEAD ↔ index) 인지 확인합니다.
func (s Spec) IsDefault() bool {
	return s.From == "" && s.To == ""
}

// IsWorktree 는 비교 대상이 작업 트리인지 확인합니다.
func (s Spec) IsWorktree() bool {
	return s.To == RefWorktree || s.To == "working-tree" || s.To == "wt"
}

// currentSpec 은 패키지 단위 글로벌 spec 입니다.
// cmd 레이어에서 SetSpec 으로 설정한 후 모든 GetStagedDiff/GetStagedContent 호출에 적용됩니다.
var currentSpec Spec

// SetSpec 은 현재 spec 을 설정합니다 (cmd 진입점에서 1 회 호출).
func SetSpec(s Spec) { currentSpec = s }

// CurrentSpec 은 현재 설정된 spec 을 반환합니다.
func CurrentSpec() Spec { return currentSpec }

// ParseRange 는 "A..B" 또는 "A...B" 형식의 range 문자열을 from, to 로 분리합니다.
// "A...B" 의 경우 git 의 symmetric diff 의미에 맞게 merge-base(A,B) 를 계산하여 from 에 사용합니다.
// range 가 아니면 ok=false 를 반환합니다.
func ParseRange(s string) (from, to string, ok bool) {
	if i := strings.Index(s, "..."); i >= 0 {
		left, right := s[:i], s[i+3:]
		base, err := mergeBase(left, right)
		if err != nil || base == "" {
			// merge-base 실패 시 left 를 그대로 사용 (best-effort).
			base = left
		}
		return base, right, true
	}
	if i := strings.Index(s, ".."); i >= 0 {
		return s[:i], s[i+2:], true
	}
	return "", "", false
}

// mergeBase 는 git merge-base 를 호출하여 두 ref 의 공통 조상을 반환합니다.
func mergeBase(a, b string) (string, error) {
	out, err := exec.Command("git", "merge-base", a, b).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// SpecFromArgs 는 git diff 호환 위치 인자에서 Spec 을 만듭니다.
//   - 0개: 기본 (staged)
//   - 1개 (range): A..B 또는 A...B
//   - 1개 (단일 ref): ref ↔ working tree
//   - 2개: A ↔ B
//
// staged=true 이면 --staged 의미로 해석합니다 (1 개 인자는 ref ↔ index 가 됨).
func SpecFromArgs(args []string, staged bool) (Spec, error) {
	if staged {
		switch len(args) {
		case 0:
			return Spec{}, nil
		case 1:
			// --staged <ref> 형식: ref 와 인덱스 비교
			return Spec{From: args[0]}, nil
		default:
			return Spec{}, fmt.Errorf("--staged 는 인자 0~1 개만 받습니다")
		}
	}
	switch len(args) {
	case 0:
		return Spec{}, nil
	case 1:
		if from, to, ok := ParseRange(args[0]); ok {
			return Spec{From: from, To: to}, nil
		}
		// 단일 ref: ref ↔ working tree
		return Spec{From: args[0], To: RefWorktree}, nil
	case 2:
		return Spec{From: args[0], To: args[1]}, nil
	default:
		return Spec{}, fmt.Errorf("인자가 너무 많습니다 (최대 2 개)")
	}
}

// buildDiffArgs 는 spec 에 해당하는 git diff 인자 목록을 구성합니다.
func buildDiffArgs(s Spec) []string {
	args := []string{"diff"}
	switch {
	case s.IsDefault():
		args = append(args, "--staged")
	case s.IsWorktree():
		// from 과 working tree 비교
		if s.From != "" {
			args = append(args, s.From)
		}
		// from 이 빈 문자열이면 인자 없는 git diff 로 인덱스 ↔ working tree 비교
	default:
		from := s.From
		if from == "" {
			from = "HEAD"
		}
		to := s.To
		if to == "" {
			to = "HEAD"
		}
		args = append(args, from, to)
	}
	return args
}

// FileDiff 는 스테이지된 diff 에서 파일 정보를 담습니다.
type FileDiff struct {
	Path            string
	AddedLines      map[int]bool // 새 파일에서 추가된 줄 번호 집합 (1 기반)
	IsDeleted       bool
	IsNew           bool // 새로 생성된 파일 (new file mode)
	HasRemovedLines bool // diff 에 제거된 줄(-) 이 존재함
	IsSubmodule     bool // git mode 160000 (서브모듈)
	IsSymlink       bool // git mode 120000 (심볼릭 링크)
}

// GetStagedDiff 는 currentSpec 기준으로 git diff 를 실행하고 파싱된 결과를 반환합니다.
// 함수명은 backward compat 를 위해 유지하지만 실제 비교 대상은 SetSpec 으로 결정됩니다.
// Spec 이 기본값(IsDefault) 이면 기존처럼 git diff --staged 를 실행합니다.
func GetStagedDiff() ([]FileDiff, error) {
	args := buildDiffArgs(currentSpec)
	// quotepath=false: diff 헤더의 비ASCII 경로가 C-스타일로 인용되지 않도록 함
	// (인용되면 ParseDiff 가 경로를 원형으로 복원하지 못함).
	cmd := exec.Command("git", append([]string{"-c", "core.quotepath=false"}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		// git diff 의 종료 코드 1 은 단순히 차이가 있음을 의미하며 오류가 아닙니다.
		if len(out) == 0 {
			return nil, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
	}
	return ParseDiff(string(out)), nil
}

// SplitNullSeparated 는 git 의 -z 출력(NUL 구분, 경로 인용 없음)을 경로 목록으로 분리합니다.
func SplitNullSeparated(out []byte) []string {
	raw := strings.TrimRight(string(out), "\x00")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\x00")
}

// contentCache: 스테이지된 파일 내용 캐시 (sync.Map으로 동시성 안전).
// 키: "절대작업디렉토리\x00상대경로"
var contentCache sync.Map

// contentCacheKey 는 (cwd, ref, path) 조합으로 캐시 키를 생성합니다.
func contentCacheKey(ref, filePath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return cwd + "\x00" + ref + "\x00" + filePath
}

// GetStagedContent 는 currentSpec.To 시점의 파일 내용을 반환합니다.
// Spec 기본값에서는 인덱스(스테이지) 내용입니다 (git show :path).
// To 가 worktree 면 디스크 파일을 읽고, 그 외 ref 면 git show <ref>:path 를 사용합니다.
// 동일 (cwd, ref, path) 조합은 캐싱하여 중복 호출을 방지합니다.
func GetStagedContent(filePath string) (string, error) {
	return GetContentAt(currentSpec.To, filePath)
}

// GetContentAt 은 ref 시점의 파일 내용을 반환합니다.
// ref 값:
//   - "" 또는 "index", "staged": 인덱스 (git show :path)
//   - "worktree", "wt", "working-tree": 작업 트리 디스크 파일
//   - 그 외: git show <ref>:path
func GetContentAt(ref, filePath string) (string, error) {
	key := contentCacheKey(ref, filePath)
	if v, ok := contentCache.Load(key); ok {
		return v.(string), nil
	}
	var (
		out []byte
		err error
	)
	switch ref {
	case "", "index", "staged":
		out, err = exec.Command("git", "show", ":"+filePath).Output()
		if err != nil {
			return "", fmt.Errorf("git show :%s: %w", filePath, err)
		}
	case RefWorktree, "wt", "working-tree":
		out, err = os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", filePath, err)
		}
	default:
		out, err = exec.Command("git", "show", ref+":"+filePath).Output()
		if err != nil {
			return "", fmt.Errorf("git show %s:%s: %w", ref, filePath, err)
		}
	}
	result := string(out)
	contentCache.Store(key, result)
	return result, nil
}

// ResetStagedContentCache: 콘텐츠 캐시를 초기화합니다 (테스트용).
func ResetStagedContentCache() {
	contentCache.Range(func(k, _ any) bool {
		contentCache.Delete(k)
		return true
	})
}

// HasExtension 는 path 가 주어진 파일 확장자 목록 중 하나를 가지는지 확인합니다.
// "dockerfile" 은 특수 식별자로, Dockerfile, Dockerfile.*, *.dockerfile 파일명 패턴에 매칭됩니다.
func HasExtension(path string, extensions []string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	ext := filepath.Ext(path)
	for _, e := range extensions {
		if e == "dockerfile" {
			if lower == "dockerfile" ||
				strings.HasPrefix(lower, "dockerfile.") ||
				strings.HasSuffix(lower, ".dockerfile") {
				return true
			}
			continue
		}
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// maxDiffLineBytes: diff 출력용 scanner 버퍼 크기.
// git diff 는 매우 긴 라인을 생성할 수 있음 (minified JS, 생성된 protobuf 등).
// 4MB 면 실제 케이스 대부분을 처리 가능.
const maxDiffLineBytes = 4 * 1024 * 1024

// ParseDiff 는 통합 diff 출력을 파싱하여 추가된 줄 번호를 포함한 FileDiff 목록을 반환합니다.
func ParseDiff(diff string) []FileDiff {
	var result []FileDiff
	var current *FileDiff
	currentNewLine := 0

	scanner := bufio.NewScanner(strings.NewReader(diff))
	scanner.Buffer(make([]byte, maxDiffLineBytes), maxDiffLineBytes)
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "diff --git "):
			if current != nil {
				result = append(result, *current)
			}
			current = &FileDiff{AddedLines: make(map[int]bool)}
			currentNewLine = 0
			// "diff --git a/foo b/foo" 에서 경로 추출 (삭제 파일 포함 모든 케이스 처리)
			if parts := strings.SplitN(line, " b/", 2); len(parts) == 2 {
				current.Path = parts[1]
			}

		case current == nil:
			continue

		case strings.HasPrefix(line, "+++ b/"):
			// 정확한 경로로 덮어씀 (rename 등 대비)
			current.Path = strings.TrimPrefix(line, "+++ b/")

		case line == "+++ /dev/null":
			current.IsDeleted = true

		case strings.HasPrefix(line, "new file mode "), strings.HasPrefix(line, "old mode "),
			strings.HasPrefix(line, "new mode "):
			// 파일 모드 줄: 신규 파일, 서브모듈(160000), 심볼릭 링크(120000) 감지
			if strings.HasPrefix(line, "new file mode ") {
				current.IsNew = true
			}
			for _, prefix := range []string{"new file mode ", "new mode ", "old mode "} {
				if strings.HasPrefix(line, prefix) {
					mode := strings.TrimSpace(strings.TrimPrefix(line, prefix))
					switch mode {
					case "160000":
						current.IsSubmodule = true
					case "120000":
						current.IsSymlink = true
					}
					break
				}
			}

		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"),
			strings.HasPrefix(line, "rename from"), strings.HasPrefix(line, "rename to"),
			strings.HasPrefix(line, "Binary "):
			// 메타데이터 줄, 건너뜀

		case strings.HasPrefix(line, "@@"):
			currentNewLine = parseHunkHeader(line)

		case strings.HasPrefix(line, "+"):
			// 새 파일에 추가된 줄
			current.AddedLines[currentNewLine] = true
			currentNewLine++

		case strings.HasPrefix(line, "-"):
			// 제거된 줄: 새 파일에 존재하지 않으므로 증가하지 않음
			current.HasRemovedLines = true

		case strings.HasPrefix(line, " "):
			// 컨텍스트 줄: 이전 파일과 새 파일 모두에 존재
			currentNewLine++
		}
	}

	if current != nil {
		result = append(result, *current)
	}
	if err := scanner.Err(); err != nil {
		// 에러 발생 전까지 파싱된 결과를 반환. 부분 결과가 크래시보다 나음.
		_ = err
	}
	return result
}

// parseHunkHeader 는 @@ -old[,count] +new[,count] @@ 를 파싱하여 새 파일 시작 줄 번호를 반환합니다.
func parseHunkHeader(line string) int {
	// 새 파일 범위의 시작 '+' 를 찾습니다.
	idx := strings.Index(line, "+")
	if idx < 0 {
		return 0
	}
	rest := line[idx+1:]
	// ',' ' ' '@' 까지의 숫자를 추출합니다.
	end := strings.IndexAny(rest, ", @\t")
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0
	}
	return n
}
