package checker

import "testing"

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
