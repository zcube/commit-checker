package checker

import (
	"regexp"
	"testing"
)

// ---- IsPathLikeString -------------------------------------------------------

func TestIsPathLikeString_SlashPath(t *testing.T) {
	if !IsPathLikeString("/api/v1/users") {
		t.Error("expected /api/v1/users to be path-like")
	}
}

func TestIsPathLikeString_MimeType(t *testing.T) {
	if !IsPathLikeString("application/json") {
		t.Error("expected application/json to be path-like")
	}
}

func TestIsPathLikeString_TextHtml(t *testing.T) {
	if !IsPathLikeString("text/html; charset=utf-8") {
		t.Error("expected text/html; charset=utf-8 to be path-like")
	}
}

func TestIsPathLikeString_NoSlash(t *testing.T) {
	if IsPathLikeString("ERR_TOKEN") {
		t.Error("expected ERR_TOKEN to NOT be path-like")
	}
}

func TestIsPathLikeString_KoreanText(t *testing.T) {
	if IsPathLikeString("안녕하세요 사용자") {
		t.Error("expected Korean text to NOT be path-like")
	}
}

// ---- IsAllUppercaseASCII ----------------------------------------------------

func TestIsAllUppercaseASCII_Constant(t *testing.T) {
	if !IsAllUppercaseASCII("ERR_TOKEN") {
		t.Error("expected ERR_TOKEN to be all-uppercase ASCII")
	}
}

func TestIsAllUppercaseASCII_MaxSize(t *testing.T) {
	if !IsAllUppercaseASCII("MAX_RETRY_COUNT") {
		t.Error("expected MAX_RETRY_COUNT to be all-uppercase ASCII")
	}
}

func TestIsAllUppercaseASCII_StatusOK(t *testing.T) {
	if !IsAllUppercaseASCII("STATUS_OK") {
		t.Error("expected STATUS_OK to be all-uppercase ASCII")
	}
}

func TestIsAllUppercaseASCII_WithLowercase(t *testing.T) {
	if IsAllUppercaseASCII("Hello, World") {
		t.Error("expected Hello, World to NOT be all-uppercase ASCII (has lowercase)")
	}
}

func TestIsAllUppercaseASCII_AllLowercase(t *testing.T) {
	if IsAllUppercaseASCII("hello world") {
		t.Error("expected hello world to NOT be all-uppercase ASCII")
	}
}

func TestIsAllUppercaseASCII_NonASCII(t *testing.T) {
	if IsAllUppercaseASCII("안녕하세요") {
		t.Error("expected Korean text to NOT be all-uppercase ASCII")
	}
}

func TestIsAllUppercaseASCII_UpperWithKorean(t *testing.T) {
	if IsAllUppercaseASCII("ERR_한국어") {
		t.Error("expected mixed ASCII+Korean to NOT be all-uppercase ASCII")
	}
}

// ---- IsTechnicalString ------------------------------------------------------

func TestIsTechnicalString_Path(t *testing.T) {
	if !IsTechnicalString("/api/v1/users") {
		t.Error("expected /api/v1/users to be technical")
	}
}

func TestIsTechnicalString_Mime(t *testing.T) {
	if !IsTechnicalString("application/json") {
		t.Error("expected application/json to be technical")
	}
}

func TestIsTechnicalString_UpperConstant(t *testing.T) {
	if !IsTechnicalString("ERR_TOKEN") {
		t.Error("expected ERR_TOKEN to be technical")
	}
}

func TestIsTechnicalString_KoreanSentence(t *testing.T) {
	if IsTechnicalString("안녕하세요 사용자입니다") {
		t.Error("expected Korean sentence to NOT be technical")
	}
}

func TestIsTechnicalString_EnglishSentence(t *testing.T) {
	if IsTechnicalString("Hello, this is a message") {
		t.Error("expected English sentence to NOT be technical (has lowercase)")
	}
}

func TestIsTechnicalString_Empty(t *testing.T) {
	// 빈 문자열은 min_length 필터에서 걸리지만, 함수 자체는 true 반환 (소문자/비ASCII 없음)
	if !IsTechnicalString("") {
		t.Error("expected empty string to be treated as technical (no letters at all)")
	}
}

// ---- compileSkipStringPatterns ----------------------------------------------

func TestCompileSkipStringPatterns_ValidPatterns(t *testing.T) {
	patterns := compileSkipStringPatterns([]string{`^[A-Z]+$`, `^\d+$`})
	if len(patterns) != 2 {
		t.Fatalf("expected 2 compiled patterns, got %d", len(patterns))
	}
}

func TestCompileSkipStringPatterns_InvalidPattern(t *testing.T) {
	// 잘못된 패턴은 건너뛰고 유효한 패턴만 반환
	patterns := compileSkipStringPatterns([]string{`^[A-Z]+$`, `[invalid`, `^\d+$`})
	if len(patterns) != 2 {
		t.Fatalf("expected 2 valid patterns (invalid skipped), got %d", len(patterns))
	}
}

func TestCompileSkipStringPatterns_Empty(t *testing.T) {
	patterns := compileSkipStringPatterns(nil)
	if len(patterns) != 0 {
		t.Fatalf("expected 0 patterns for nil input, got %d", len(patterns))
	}
}

// ---- matchesSkipPattern -----------------------------------------------------

func TestMatchesSkipPattern_Match(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`),
		regexp.MustCompile(`^\d+(\.\d+)*$`),
	}
	// 대문자 상수
	if !matchesSkipPattern("MAX_RETRY", patterns) {
		t.Error("expected MAX_RETRY to match uppercase constant pattern")
	}
	// 버전 문자열
	if !matchesSkipPattern("1.2.3", patterns) {
		t.Error("expected 1.2.3 to match version pattern")
	}
}

func TestMatchesSkipPattern_NoMatch(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`),
	}
	if matchesSkipPattern("안녕하세요", patterns) {
		t.Error("expected Korean text to NOT match any pattern")
	}
	if matchesSkipPattern("hello world", patterns) {
		t.Error("expected lowercase text to NOT match any pattern")
	}
}

func TestMatchesSkipPattern_EmptyPatterns(t *testing.T) {
	if matchesSkipPattern("anything", nil) {
		t.Error("expected no match with nil patterns")
	}
}

// ---- compileSkipStringPatterns 전체 매칭 동작 테스트 -------------------------

func TestCompileSkipStringPatterns_FullMatch(t *testing.T) {
	// ^$ 없이도 전체 문자열 매칭이 되어야 함
	patterns := compileSkipStringPatterns([]string{`color`})
	if matchesSkipPattern("backgroundColor", patterns) {
		t.Error("expected 'backgroundColor' to NOT match full-match pattern 'color'")
	}
	if !matchesSkipPattern("color", patterns) {
		t.Error("expected 'color' to match full-match pattern 'color'")
	}
}

func TestCompileSkipStringPatterns_FullMatchWithAnchors(t *testing.T) {
	// ^$ 이미 있으면 그대로 전체 매칭
	patterns := compileSkipStringPatterns([]string{`^(true|false|null|nil)$`})
	if matchesSkipPattern("참이 아닌 거짓", patterns) {
		t.Error("expected Korean text to NOT match boolean pattern")
	}
	if !matchesSkipPattern("true", patterns) {
		t.Error("expected 'true' to match boolean pattern")
	}
}

func TestCompileSkipStringPatterns_WildcardPattern(t *testing.T) {
	// .* 사용 시 부분 매칭처럼 동작 가능
	patterns := compileSkipStringPatterns([]string{`(?i).*color.*`})
	if !matchesSkipPattern("backgroundColor", patterns) {
		t.Error("expected 'backgroundColor' to match '(?i).*color.*'")
	}
	if !matchesSkipPattern("color", patterns) {
		t.Error("expected 'color' to match '(?i).*color.*'")
	}
	// 전체 매칭이므로 .* 없이는 부분 매칭 안됨
	patterns2 := compileSkipStringPatterns([]string{`color`})
	if matchesSkipPattern("mycolor", patterns2) {
		t.Error("expected 'mycolor' to NOT match full-match 'color'")
	}
}

func TestCompileSkipStringPatterns_CaseInsensitive(t *testing.T) {
	patterns := compileSkipStringPatterns([]string{`(?i)hello`})
	if !matchesSkipPattern("HELLO", patterns) {
		t.Error("expected 'HELLO' to match case-insensitive 'hello'")
	}
	if matchesSkipPattern("say hello world", patterns) {
		t.Error("expected 'say hello world' to NOT match full-match '(?i)hello'")
	}
}
