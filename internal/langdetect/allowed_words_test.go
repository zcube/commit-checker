package langdetect

import "testing"

func TestStripAllowedWords(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		words []string
		want  string
	}{
		{
			name:  "빈 단어 목록 — 원본 그대로",
			text:  "TypeScript 코드입니다",
			words: nil,
			want:  "TypeScript 코드입니다",
		},
		{
			// "TypeScript" = 10 runes → 10 spaces
			name:  "단어 앞뒤 공백",
			text:  "TypeScript 코드입니다",
			words: []string{"TypeScript"},
			want:  "           코드입니다",
		},
		{
			name:  "한국어 조사 앞 단어 (공백 없음)",
			text:  "TypeScript를 사용합니다",
			words: []string{"TypeScript"},
			want:  "          를 사용합니다",
		},
		{
			name:  "대소문자 무관 일치",
			text:  "typescript 코드입니다",
			words: []string{"TypeScript"},
			want:  "           코드입니다",
		},
		{
			name:  "부분 일치 방지: Java ≠ JavaScript",
			text:  "JavaScript 코드입니다",
			words: []string{"Java"},
			want:  "JavaScript 코드입니다",
		},
		{
			// "JavaScript" = 10 runes → 10 spaces
			name:  "긴 단어 우선 처리: JavaScript가 Java보다 먼저",
			text:  "JavaScript 코드입니다",
			words: []string{"Java", "JavaScript"},
			want:  "           코드입니다",
		},
		{
			// "TypeScript" = 10, "JavaScript" = 10
			name:  "여러 단어 동시 제거",
			text:  "TypeScript / JavaScript 지원합니다",
			words: []string{"TypeScript", "JavaScript"},
			want:  "           /            지원합니다",
		},
		{
			// "API" = 3 runes → 3 spaces
			name:  "단어가 문자열 중간에 있는 경우",
			text:  "이 API를 사용하세요",
			words: []string{"API"},
			want:  "이    를 사용하세요",
		},
		{
			// "Python" = 6 runes → 6 spaces
			name:  "문장 끝 단어",
			text:  "사용 언어는 Python",
			words: []string{"Python"},
			want:  "사용 언어는       ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripAllowedWords(tc.text, tc.words)
			if got != tc.want {
				t.Errorf("StripAllowedWords(%q, %v)\n  got:  %q\n  want: %q", tc.text, tc.words, got, tc.want)
			}
		})
	}
}

func TestIsRequiredLanguage_WithAllowedWords(t *testing.T) {
	cases := []struct {
		name         string
		text         string
		allowedWords []string
		wantOK       bool
		wantContent  bool
	}{
		{
			name:         "허용 단어 제거 후 한국어만 남으면 통과",
			text:         "TypeScript 코드를 작성합니다",
			allowedWords: []string{"TypeScript"},
			wantOK:       true,
			wantContent:  true,
		},
		{
			name:         "허용 단어 없으면 실패",
			text:         "TypeScript 코드를 작성합니다",
			allowedWords: nil,
			wantOK:       true, // 한국어가 포함되어 있으므로 통과 (hasScript 기준)
			wantContent:  true,
		},
		{
			name:         "순수 영어 주석은 허용 단어 있어도 실패",
			text:         "This is written in English only",
			allowedWords: []string{"TypeScript"},
			wantOK:       false,
			wantContent:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stripped := StripAllowedWords(tc.text, tc.allowedWords)
			ok, hasContent := IsRequiredLanguage(stripped, Korean, 5, nil)
			if ok != tc.wantOK || hasContent != tc.wantContent {
				t.Errorf("IsRequiredLanguage(StripAllowedWords(%q)) ok=%v hasContent=%v, want ok=%v hasContent=%v",
					tc.text, ok, hasContent, tc.wantOK, tc.wantContent)
			}
		})
	}
}
