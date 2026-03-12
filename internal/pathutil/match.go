package pathutil

import "path/filepath"

// MatchesAny 는 path 가 주어진 glob 패턴 중 하나와 일치하는지 확인합니다.
// 패턴은 filepath.Match 의미론으로 경로와 매칭됩니다.
// "**" glob 매칭은 기본 이름만 단독으로 테스트하는 방식으로 지원됩니다.
func MatchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// 전체 경로 매칭 시도
		if m, _ := filepath.Match(pattern, path); m {
			return true
		}
		// 기본 이름만 매칭 시도
		if m, _ := filepath.Match(pattern, filepath.Base(path)); m {
			return true
		}
		// 슬래시 정규화 후 매칭 시도
		if matchDoubleStarGlob(path, pattern) {
			return true
		}
	}
	return false
}

// matchDoubleStarGlob 는 "**" 를 포함한 패턴을 "/" 로 분리하여 경로 세그먼트를 순차적으로 매칭합니다.
// "vendor/**" 나 "**/generated/*.go" 같은 패턴을 지원합니다.
func matchDoubleStarGlob(path, pattern string) bool {
	// 일관된 구분자 처리를 위해 filepath.ToSlash 사용
	pathParts := splitPath(filepath.ToSlash(path))
	patParts := splitPath(filepath.ToSlash(pattern))
	return matchParts(pathParts, patParts)
}

func splitPath(p string) []string {
	var parts []string
	for _, seg := range filepath.SplitList(p) {
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	// filepath.SplitList 은 OS 구분자로 분리하므로 수동 분리 사용
	_ = parts
	result := []string{}
	current := ""
	for _, ch := range p {
		if ch == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func matchParts(pathParts, patParts []string) bool {
	if len(patParts) == 0 {
		return len(pathParts) == 0
	}
	if patParts[0] == "**" {
		// ** 는 0개 이상의 경로 세그먼트와 매칭 가능
		for i := range len(pathParts) + 1 {
			if matchParts(pathParts[i:], patParts[1:]) {
				return true
			}
		}
		return false
	}
	if len(pathParts) == 0 {
		return false
	}
	m, _ := filepath.Match(patParts[0], pathParts[0])
	if !m {
		return false
	}
	return matchParts(pathParts[1:], patParts[1:])
}
