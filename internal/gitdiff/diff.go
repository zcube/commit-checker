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

// FileDiff 는 스테이지된 diff 에서 파일 정보를 담습니다.
type FileDiff struct {
	Path        string
	AddedLines  map[int]bool // 새 파일에서 추가된 줄 번호 집합 (1 기반)
	IsDeleted   bool
	IsSubmodule bool // git mode 160000 (서브모듈)
	IsSymlink   bool // git mode 120000 (심볼릭 링크)
}

// GetStagedDiff 는 git diff --staged 를 실행하고 파싱된 파일 diff 목록을 반환합니다.
func GetStagedDiff() ([]FileDiff, error) {
	cmd := exec.Command("git", "diff", "--staged")
	out, err := cmd.Output()
	if err != nil {
		// git diff 의 종료 코드 1 은 단순히 차이가 있음을 의미하며 오류가 아닙니다.
		// 실제 오류는 출력이 없습니다.
		if len(out) == 0 {
			return nil, fmt.Errorf("git diff --staged failed: %w", err)
		}
	}
	return ParseDiff(string(out)), nil
}

// contentCache: 스테이지된 파일 내용 캐시 (sync.Map으로 동시성 안전).
// 키: "절대작업디렉토리\x00상대경로"
var contentCache sync.Map

// stagedCacheKey: 현재 작업 디렉토리와 경로를 결합한 캐시 키 반환.
func stagedCacheKey(filePath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filePath
	}
	return cwd + "\x00" + filePath
}

// GetStagedContent 는 git show 를 사용하여 파일의 스테이지된(인덱스) 내용을 반환합니다.
// 동일 경로는 캐싱하여 중복 git show 호출을 방지합니다.
func GetStagedContent(filePath string) (string, error) {
	key := stagedCacheKey(filePath)
	if v, ok := contentCache.Load(key); ok {
		return v.(string), nil
	}
	cmd := exec.Command("git", "show", ":"+filePath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show :%s: %w", filePath, err)
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

		case current == nil:
			continue

		case strings.HasPrefix(line, "+++ b/"):
			current.Path = strings.TrimPrefix(line, "+++ b/")

		case line == "+++ /dev/null":
			current.IsDeleted = true

		case strings.HasPrefix(line, "new file mode "), strings.HasPrefix(line, "old mode "),
			strings.HasPrefix(line, "new mode "):
			// 파일 모드 줄: 서브모듈(160000)과 심볼릭 링크(120000) 감지
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
