package langdetect_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/langdetect"
)

func TestIsRequiredLanguage_Korean(t *testing.T) {
	cases := []struct {
		text        string
		wantOK      bool
		wantContent bool
	}{
		// Pure Korean — OK
		{"이것은 한국어 주석입니다", true, true},
		// Mixed Korean/English identifier — OK (Korean is present)
		{"변수 name을 설정합니다", true, true},
		// Pure English sentence — FAIL
		{"This is an English comment", false, true},
		// Technical directive — skip (no content)
		{"TODO", true, false},
		{"nolint:errcheck", true, false},
		// Too short — skip
		{"hi", true, false},
		// URL — skip
		{"https://example.com/path", true, false},
	}

	for _, tc := range cases {
		ok, hasContent := langdetect.IsRequiredLanguage(tc.text, langdetect.Korean, 5, nil)
		if ok != tc.wantOK || hasContent != tc.wantContent {
			t.Errorf("IsRequiredLanguage(%q, korean) = (%v, %v), want (%v, %v)",
				tc.text, ok, hasContent, tc.wantOK, tc.wantContent)
		}
	}
}

func TestIsRequiredLanguage_Any(t *testing.T) {
	ok, hasContent := langdetect.IsRequiredLanguage("anything goes here", langdetect.Any, 5, nil)
	if !ok || !hasContent {
		t.Errorf("required=any should always pass; got ok=%v hasContent=%v", ok, hasContent)
	}
}

func TestIsRequiredLanguage_English(t *testing.T) {
	ok, hasContent := langdetect.IsRequiredLanguage("This is English", langdetect.English, 5, nil)
	if !ok || !hasContent {
		t.Errorf("English text should pass english requirement; got ok=%v hasContent=%v", ok, hasContent)
	}
	ok2, _ := langdetect.IsRequiredLanguage("이것은 한국어입니다", langdetect.English, 5, nil)
	if ok2 {
		t.Error("Korean text should fail english requirement")
	}
}

func TestIsRequiredLanguage_Japanese(t *testing.T) {
	cases := []struct {
		text        string
		wantOK      bool
		wantContent bool
	}{
		// Pure Hiragana — OK
		{"これは日本語のコメントです", true, true},
		// Katakana — OK
		{"ユーザーデータを処理する", true, true},
		// Mixed Japanese/English — OK (Japanese is present)
		{"変数 name を設定する", true, true},
		// Pure Korean — FAIL
		{"이것은 한국어 주석입니다", false, true},
		// Pure English — FAIL
		{"This is an English comment", false, true},
		// Pure Chinese CJK (no Hiragana/Katakana) — FAIL for Japanese requirement
		{"这是一个中文注释内容示例", false, true},
	}
	for _, tc := range cases {
		ok, hasContent := langdetect.IsRequiredLanguage(tc.text, langdetect.Japanese, 5, nil)
		if ok != tc.wantOK || hasContent != tc.wantContent {
			t.Errorf("IsRequiredLanguage(%q, japanese) = (%v, %v), want (%v, %v)",
				tc.text, ok, hasContent, tc.wantOK, tc.wantContent)
		}
	}
}

func TestIsRequiredLanguage_Chinese(t *testing.T) {
	cases := []struct {
		text        string
		wantOK      bool
		wantContent bool
	}{
		// Pure CJK Unified Ideographs — OK
		{"这是一个中文注释内容示例", true, true},
		// Mixed Chinese/English — OK (Chinese is present)
		{"处理 user 数据的函数", true, true},
		// Pure Korean — FAIL
		{"이것은 한국어 주석입니다", false, true},
		// Pure English — FAIL
		{"This is an English comment", false, true},
		// Pure Hiragana (Japanese kana only) — FAIL for Chinese requirement
		{"これはひらがなのコメント", false, true},
	}
	for _, tc := range cases {
		ok, hasContent := langdetect.IsRequiredLanguage(tc.text, langdetect.Chinese, 5, nil)
		if ok != tc.wantOK || hasContent != tc.wantContent {
			t.Errorf("IsRequiredLanguage(%q, chinese) = (%v, %v), want (%v, %v)",
				tc.text, ok, hasContent, tc.wantOK, tc.wantContent)
		}
	}
}

func TestIsRequiredLanguage_CrossLanguage(t *testing.T) {
	// NOTE on Japanese ↔ Chinese overlap:
	// Japanese kanji (e.g. 日本語) occupy the same CJK Unified Ideographs codepoints as
	// Chinese characters. Because of this shared range, Japanese text that contains kanji
	// will also pass a "chinese" requirement. This is a known limitation of codepoint-range
	// detection; distinguishing the two requires a full language model.
	//
	// The table below documents the *actual* expected behaviour including this overlap.
	cases := []struct {
		text     string
		required langdetect.Language
		wantOK   bool
		note     string
	}{
		// Korean samples
		{"이것은 한국어 주석입니다", langdetect.Korean, true, "Korean passes Korean"},
		{"이것은 한국어 주석입니다", langdetect.Japanese, false, "Korean fails Japanese"},
		{"이것은 한국어 주석입니다", langdetect.Chinese, false, "Korean fails Chinese"},
		{"이것은 한국어 주석입니다", langdetect.English, false, "Korean fails English"},

		// Japanese samples (kana-only: no kanji ambiguity)
		{"これはひらがなとカタカナです", langdetect.Japanese, true, "kana passes Japanese"},
		{"これはひらがなとカタカナです", langdetect.Korean, false, "kana fails Korean"},
		{"これはひらがなとカタカナです", langdetect.Chinese, false, "kana fails Chinese"},
		{"これはひらがなとカタカナです", langdetect.English, false, "kana fails English"},

		// Japanese with kanji — passes both Japanese (has kana) AND Chinese (has CJK)
		{"これは日本語のコメントです", langdetect.Japanese, true, "kanji+kana passes Japanese"},
		{"これは日本語のコメントです", langdetect.Chinese, true, "kanji shared with CJK — also passes Chinese"},

		// Chinese samples (pure CJK, no kana)
		{"这是一个中文注释内容示例", langdetect.Chinese, true, "Chinese passes Chinese"},
		{"这是一个中文注释内容示例", langdetect.Japanese, false, "pure CJK (no kana) fails Japanese"},
		{"这是一个中文注释内容示例", langdetect.Korean, false, "Chinese fails Korean"},
		{"这是一个中文注释内容示例", langdetect.English, false, "Chinese fails English"},

		// English samples
		{"This is an English comment here", langdetect.English, true, "English passes English"},
		{"This is an English comment here", langdetect.Korean, false, "English fails Korean"},
		{"This is an English comment here", langdetect.Japanese, false, "English fails Japanese"},
		{"This is an English comment here", langdetect.Chinese, false, "English fails Chinese"},
	}

	for _, tc := range cases {
		ok, hasContent := langdetect.IsRequiredLanguage(tc.text, tc.required, 5, nil)
		if !hasContent {
			t.Errorf("%s: unexpected no-content for %q", tc.note, tc.text)
			continue
		}
		if ok != tc.wantOK {
			t.Errorf("%s: IsRequiredLanguage(%q, %s) = %v, want %v",
				tc.note, tc.text, tc.required, ok, tc.wantOK)
		}
	}
}

func TestDominant(t *testing.T) {
	cases := []struct {
		text string
		want langdetect.Language
	}{
		{"안녕하세요 반갑습니다", langdetect.Korean},
		{"Hello world here", langdetect.English},
		{"これは日本語です", langdetect.Japanese},
		{"这是中文内容示例", langdetect.Chinese},
	}
	for _, tc := range cases {
		got := langdetect.Dominant(tc.text)
		if got != tc.want {
			t.Errorf("Dominant(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}
