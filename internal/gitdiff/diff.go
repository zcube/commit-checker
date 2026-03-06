package gitdiff

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FileDiff contains information about a file in the staged diff
type FileDiff struct {
	Path       string
	AddedLines map[int]bool // set of line numbers (1-based) added in the new file
	IsDeleted  bool
}

// GetStagedDiff runs git diff --staged and returns parsed file diffs
func GetStagedDiff() ([]FileDiff, error) {
	cmd := exec.Command("git", "diff", "--staged")
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 from git diff just means there are differences; it's not an error here.
		// Real errors produce no output.
		if len(out) == 0 {
			return nil, fmt.Errorf("git diff --staged failed: %w", err)
		}
	}
	return ParseDiff(string(out)), nil
}

// GetStagedContent returns the staged (index) content of a file using git show
func GetStagedContent(path string) (string, error) {
	cmd := exec.Command("git", "show", ":"+path)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show :%s: %w", path, err)
	}
	return string(out), nil
}

// HasExtension reports whether path has one of the given file extensions
func HasExtension(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	for _, e := range extensions {
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

// ParseDiff parses a unified diff output and returns FileDiffs with added line numbers
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

		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"),
			strings.HasPrefix(line, "rename from"), strings.HasPrefix(line, "rename to"),
			strings.HasPrefix(line, "Binary "):
			// metadata lines, skip

		case strings.HasPrefix(line, "@@"):
			currentNewLine = parseHunkHeader(line)

		case strings.HasPrefix(line, "+"):
			// Added line in the new file
			current.AddedLines[currentNewLine] = true
			currentNewLine++

		case strings.HasPrefix(line, "-"):
			// Removed line: does not exist in the new file, don't increment

		case strings.HasPrefix(line, " "):
			// Context line: exists in both old and new file
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

// parseHunkHeader parses @@ -old[,count] +new[,count] @@ and returns the new file start line
func parseHunkHeader(line string) int {
	// Find the '+' that starts the new-file range
	idx := strings.Index(line, "+")
	if idx < 0 {
		return 0
	}
	rest := line[idx+1:]
	// Take digits up to ',' ' ' or '@'
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
