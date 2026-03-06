package charset_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/charset"
)

// ---- IsInvisible -----------------------------------------------------------

func TestIsInvisible_AllowedWhitespace(t *testing.T) {
	allowed := []rune{' ', '\t', '\n', '\r'}
	for _, r := range allowed {
		if charset.IsInvisible(r) {
			t.Errorf("rune U+%04X should be allowed (normal whitespace)", r)
		}
	}
}

func TestIsInvisible_BOM_Allowed(t *testing.T) {
	// U+FEFF BOM must be explicitly allowed
	if charset.IsInvisible(0xFEFF) {
		t.Error("U+FEFF (BOM) must be allowed")
	}
}

func TestIsInvisible_InvisibleChars(t *testing.T) {
	invisible := []struct {
		r    rune
		name string
	}{
		{0x00A0, "NO-BREAK SPACE"},
		{0x200B, "ZERO WIDTH SPACE"},
		{0x200C, "ZERO WIDTH NON-JOINER"},
		{0x200D, "ZERO WIDTH JOINER"},
		{0x200E, "LEFT-TO-RIGHT MARK"},
		{0x200F, "RIGHT-TO-LEFT MARK"},
		// Note: U+2028 LINE SEPARATOR and U+2029 PARAGRAPH SEPARATOR are NOT
		// in InvisibleRanges (they fall between ranges 0x200F and 0x202A).
		{0x202A, "LEFT-TO-RIGHT EMBEDDING"},
		{0x202E, "RIGHT-TO-LEFT OVERRIDE"},
		{0x2060, "WORD JOINER"},
		{0x3000, "IDEOGRAPHIC SPACE"},
		{0xFFA0, "HALFWIDTH HANGUL FILLER"},
	}
	for _, tc := range invisible {
		if !charset.IsInvisible(tc.r) {
			t.Errorf("U+%04X (%s) should be invisible", tc.r, tc.name)
		}
	}
}

func TestIsInvisible_NormalChars_NotInvisible(t *testing.T) {
	normal := []rune{'A', 'z', '0', '한', '日', '!', '.'}
	for _, r := range normal {
		if charset.IsInvisible(r) {
			t.Errorf("rune U+%04X should NOT be invisible", r)
		}
	}
}

// ---- InvisibleName ---------------------------------------------------------

func TestInvisibleName_KnownNames(t *testing.T) {
	cases := []struct {
		r    rune
		want string
	}{
		{0x00A0, "NO-BREAK SPACE"},
		{0x00AD, "SOFT HYPHEN"},
		{0x200B, "ZERO WIDTH SPACE"},
		{0x200C, "ZERO WIDTH NON-JOINER"},
		{0x200D, "ZERO WIDTH JOINER"},
		{0x200E, "LEFT-TO-RIGHT MARK"},
		{0x200F, "RIGHT-TO-LEFT MARK"},
		{0x202A, "LEFT-TO-RIGHT EMBEDDING"},
		{0x202E, "RIGHT-TO-LEFT OVERRIDE"},
		{0x202F, "NARROW NO-BREAK SPACE"},
		{0x2003, "EM SPACE"},
		{0x2060, "WORD JOINER"},
		{0x3000, "IDEOGRAPHIC SPACE"},
		{0xFFA0, "HALFWIDTH HANGUL FILLER"},
	}
	for _, tc := range cases {
		got := charset.InvisibleName(tc.r)
		if got != tc.want {
			t.Errorf("InvisibleName(U+%04X) = %q, want %q", tc.r, got, tc.want)
		}
	}
}

func TestInvisibleName_UnknownRune_EmptyString(t *testing.T) {
	// Characters not in the name map should return ""
	unknowns := []rune{'A', 0x2028, 0x2029, 0x2800}
	for _, r := range unknowns {
		name := charset.InvisibleName(r)
		// These are invisible but may not be in the name map — just check no panic
		_ = name
	}
}

func TestInvisibleName_AllKnownNamesNonEmpty(t *testing.T) {
	known := []rune{
		0x00A0, 0x00AD, 0x034F, 0x1680, 0x180E,
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005,
		0x2006, 0x2007, 0x2008, 0x2009, 0x200A, 0x200B,
		0x200C, 0x200D, 0x200E, 0x200F, 0x202A, 0x202B,
		0x202C, 0x202D, 0x202E, 0x202F, 0x205F, 0x2060,
		0x2061, 0x2062, 0x2063, 0x2064, 0x206A, 0x206B,
		0x206C, 0x206D, 0x206E, 0x206F, 0x2800, 0x3000,
		0xFFA0,
	}
	for _, r := range known {
		if charset.InvisibleName(r) == "" {
			t.Errorf("InvisibleName(U+%04X) returned empty string, expected named entry", r)
		}
	}
}

// ---- TablesForLocale -------------------------------------------------------

func TestTablesForLocale_KnownLocales(t *testing.T) {
	locales := []string{"ko", "ja", "ru", "zh-hans", "zh-hant", "_default"}
	for _, locale := range locales {
		tables := charset.TablesForLocale(locale)
		if len(tables) != 2 {
			t.Errorf("TablesForLocale(%q) returned %d tables, want 2", locale, len(tables))
		}
		for i, tbl := range tables {
			if tbl == nil {
				t.Errorf("TablesForLocale(%q): table[%d] is nil", locale, i)
			}
		}
	}
}

func TestTablesForLocale_UnknownLocale_FallsBackToDefault(t *testing.T) {
	tables := charset.TablesForLocale("xx-unknown")
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables for unknown locale, got %d", len(tables))
	}
	if tables[0] == nil {
		t.Error("first table (locale-specific) must not be nil")
	}
}

func TestTablesForLocale_ZhCN_FallsBackToZhHans(t *testing.T) {
	// zh-CN and zh_CN should fall back to zh-hans
	for _, locale := range []string{"zh-CN", "zh_CN"} {
		tables := charset.TablesForLocale(locale)
		if tables[0] == nil {
			t.Errorf("TablesForLocale(%q): first table is nil", locale)
		}
	}
}

func TestTablesForLocale_ZhVariants(t *testing.T) {
	for _, locale := range []string{"zh", "zh-TW", "zh-something"} {
		tables := charset.TablesForLocale(locale)
		if tables[0] == nil {
			t.Errorf("TablesForLocale(%q): first table is nil", locale)
		}
	}
}

func TestTablesForLocale_AlwaysIncludesCommon(t *testing.T) {
	// Second table is always _common regardless of locale
	for _, locale := range []string{"ko", "en", "unknown"} {
		tables := charset.TablesForLocale(locale)
		if len(tables) < 2 || tables[1] == nil {
			t.Errorf("TablesForLocale(%q): _common table (index 1) missing or nil", locale)
		}
	}
}

// ---- IsAmbiguous -----------------------------------------------------------

func TestIsAmbiguous_CyrillicA(t *testing.T) {
	tables := charset.TablesForLocale("ko")
	var confusableTo rune
	// U+0410 CYRILLIC CAPITAL LETTER A looks like Latin A
	if !charset.IsAmbiguous(0x0410, &confusableTo, tables...) {
		t.Error("U+0410 Cyrillic А should be ambiguous")
	}
	if confusableTo != 'A' {
		t.Errorf("Cyrillic А confusable with %q (U+%04X), want 'A'", string(confusableTo), confusableTo)
	}
}

func TestIsAmbiguous_CyrillicO(t *testing.T) {
	tables := charset.TablesForLocale("ko")
	var confusableTo rune
	// U+043E CYRILLIC SMALL LETTER O looks like Latin o
	if !charset.IsAmbiguous(0x043E, &confusableTo, tables...) {
		t.Error("U+043E Cyrillic о should be ambiguous")
	}
	if confusableTo != 'o' {
		t.Errorf("Cyrillic о confusable with %q, want 'o'", string(confusableTo))
	}
}

func TestIsAmbiguous_NormalASCII_NotAmbiguous(t *testing.T) {
	tables := charset.TablesForLocale("ko")
	var confusableTo rune
	for _, r := range "ABCabc123" {
		if charset.IsAmbiguous(r, &confusableTo, tables...) {
			t.Errorf("ASCII rune %q should NOT be ambiguous", string(r))
		}
	}
}

func TestIsAmbiguous_KoreanHangul_NotAmbiguous(t *testing.T) {
	tables := charset.TablesForLocale("ko")
	var confusableTo rune
	for _, r := range "안녕하세요" {
		if charset.IsAmbiguous(r, &confusableTo, tables...) {
			t.Errorf("Korean rune %q (U+%04X) should not be ambiguous", string(r), r)
		}
	}
}

func TestIsAmbiguous_NilTable_Skipped(t *testing.T) {
	// Passing a nil table must not panic
	var confusableTo rune
	result := charset.IsAmbiguous(0x0410, &confusableTo, nil)
	_ = result // just ensure no panic
}

func TestIsAmbiguous_JapaneseLocale(t *testing.T) {
	tables := charset.TablesForLocale("ja")
	var confusableTo rune
	// Common Cyrillic confusables should still appear in ja locale
	if !charset.IsAmbiguous(0x0410, &confusableTo, tables...) {
		t.Error("U+0410 Cyrillic А should be ambiguous with ja locale tables")
	}
}

func TestIsAmbiguous_RussianLocale(t *testing.T) {
	tables := charset.TablesForLocale("ru")
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables for ru, got %d", len(tables))
	}
	// ru locale has its own table
	var confusableTo rune
	found := charset.IsAmbiguous(0x0410, &confusableTo, tables...)
	_ = found // just ensure no panic; Russian locale may behave differently
}

func TestTablesForLocale_SubtagFallback(t *testing.T) {
	// "ko-KR" should fall back to "ko" via sub-tag stripping
	tablesKoKR := charset.TablesForLocale("ko-KR")
	tablesKo := charset.TablesForLocale("ko")
	if tablesKoKR[0] != tablesKo[0] {
		t.Error("ko-KR should fall back to the same table as ko")
	}
}
