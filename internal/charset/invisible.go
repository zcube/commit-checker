// Package charset: 유니코드 문자 분류 유틸리티 패키지.
// InvisibleRanges 테이블은 Gitea (MIT 라이선스)에서 적용:
// https://github.com/go-gitea/gitea/blob/main/modules/charset/invisible_gen.go
package charset

import "unicode"

// InvisibleRanges: 비표준 공백, 양방향 제어 문자, 변형 선택자 등
// 시각적으로 공백이나 빈 문자와 구분 불가능한 보이지 않는 / 폭 없는
// 문자를 포함하는 unicode.RangeTable.
var InvisibleRanges = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 11, Hi: 13, Stride: 1},
		{Lo: 127, Hi: 160, Stride: 33},
		{Lo: 173, Hi: 847, Stride: 674},
		{Lo: 1564, Hi: 4447, Stride: 2883},
		{Lo: 4448, Hi: 6068, Stride: 1620},
		{Lo: 6069, Hi: 6155, Stride: 86},
		{Lo: 6156, Hi: 6158, Stride: 1},
		{Lo: 7355, Hi: 7356, Stride: 1},
		{Lo: 8192, Hi: 8207, Stride: 1},
		{Lo: 8234, Hi: 8239, Stride: 1},
		{Lo: 8287, Hi: 8303, Stride: 1},
		{Lo: 10240, Hi: 12288, Stride: 2048},
		{Lo: 12644, Hi: 65024, Stride: 52380},
		{Lo: 65025, Hi: 65039, Stride: 1},
		// 참고: 65279 (U+FEFF, BOM/ZWNBSP)는 의도적으로 제외 — 파일 시작의
		// BOM은 유효한 UTF-8 인코딩 마커이므로 허용됨.
		{Lo: 65440, Hi: 65440, Stride: 1},
		{Lo: 65520, Hi: 65528, Stride: 1},
		{Lo: 65532, Hi: 65532, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 78844, Hi: 119155, Stride: 40311},
		{Lo: 119156, Hi: 119162, Stride: 1},
		{Lo: 917504, Hi: 917631, Stride: 1},
		{Lo: 917760, Hi: 917999, Stride: 1},
	},
	LatinOffset: 2,
}

// IsInvisible: 커밋 메시지에 나타나면 안 되는 보이지 않는 또는 폭 없는 문자인지 확인.
//
// 허용: 일반 공백 U+0020, 탭 U+0009, LF U+000A, CR U+000D,
// 및 U+FEFF (BOM) — 유효한 UTF-8 파일 인코딩 마커.
func IsInvisible(r rune) bool {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	return unicode.Is(InvisibleRanges, r)
}

// InvisibleName: 보이지 않는 룬의 사람이 읽을 수 있는 설명을 반환.
func InvisibleName(r rune) string {
	names := map[rune]string{
		0x00A0: "NO-BREAK SPACE",
		0x00AD: "SOFT HYPHEN",
		0x034F: "COMBINING GRAPHEME JOINER",
		0x1680: "OGHAM SPACE MARK",
		0x180E: "MONGOLIAN VOWEL SEPARATOR",
		0x2000: "EN QUAD",
		0x2001: "EM QUAD",
		0x2002: "EN SPACE",
		0x2003: "EM SPACE",
		0x2004: "THREE-PER-EM SPACE",
		0x2005: "FOUR-PER-EM SPACE",
		0x2006: "SIX-PER-EM SPACE",
		0x2007: "FIGURE SPACE",
		0x2008: "PUNCTUATION SPACE",
		0x2009: "THIN SPACE",
		0x200A: "HAIR SPACE",
		0x200B: "ZERO WIDTH SPACE",
		0x200C: "ZERO WIDTH NON-JOINER",
		0x200D: "ZERO WIDTH JOINER",
		0x200E: "LEFT-TO-RIGHT MARK",
		0x200F: "RIGHT-TO-LEFT MARK",
		0x202A: "LEFT-TO-RIGHT EMBEDDING",
		0x202B: "RIGHT-TO-LEFT EMBEDDING",
		0x202C: "POP DIRECTIONAL FORMATTING",
		0x202D: "LEFT-TO-RIGHT OVERRIDE",
		0x202E: "RIGHT-TO-LEFT OVERRIDE",
		0x202F: "NARROW NO-BREAK SPACE",
		0x205F: "MEDIUM MATHEMATICAL SPACE",
		0x2060: "WORD JOINER",
		0x2061: "FUNCTION APPLICATION",
		0x2062: "INVISIBLE TIMES",
		0x2063: "INVISIBLE SEPARATOR",
		0x2064: "INVISIBLE PLUS",
		0x206A: "INHIBIT SYMMETRIC SWAPPING",
		0x206B: "ACTIVATE SYMMETRIC SWAPPING",
		0x206C: "INHIBIT ARABIC FORM SHAPING",
		0x206D: "ACTIVATE ARABIC FORM SHAPING",
		0x206E: "NATIONAL DIGIT SHAPES",
		0x206F: "NOMINAL DIGIT SHAPES",
		0x2800: "BRAILLE PATTERN BLANK",
		0x3000: "IDEOGRAPHIC SPACE",
		0xFFA0: "HALFWIDTH HANGUL FILLER",
	}
	if name, ok := names[r]; ok {
		return name
	}
	return ""
}
