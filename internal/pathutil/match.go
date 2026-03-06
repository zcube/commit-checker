package pathutil

import "path/filepath"

// MatchesAny: 경로가 주어진 glob 패턴 중 하나와 일치하는지 확인.
// filepath.Match 의미론으로 패턴을 매칭.
// "**" glob 매칭은 베이스 이름만으로도 추가 테스트하여 지원.
func MatchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// 전체 경로 매칭 시도
		if m, _ := filepath.Match(pattern, path); m {
			return true
		}
		// 베이스 이름만으로 매칭 시도
		if m, _ := filepath.Match(pattern, filepath.Base(path)); m {
			return true
		}
		// 슬래시 정규화 매칭 시도
		if matchDoubleStarGlob(path, pattern) {
			return true
		}
	}
	return false
}

// matchDoubleStarGlob: "**"를 포함하는 패턴을 "/"로 분할하여 경로 세그먼트를
// 점진적으로 매칭. "vendor/**" 및 "**/generated/*.go" 같은 패턴 지원.
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
	// filepath.SplitList는 OS 목록 구분자로 분할; 대신 수동 분할 사용
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
		// **는 0개 이상의 경로 세그먼트와 매칭 가능
		for i := 0; i <= len(pathParts); i++ {
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
