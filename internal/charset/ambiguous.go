// Ambiguous character detection adapted from Gitea (MIT License):
// https://github.com/go-gitea/gitea/blob/main/modules/charset/ambiguous.go
// https://github.com/go-gitea/gitea/blob/main/modules/charset/ambiguous_gen.go
//
// Source data: https://github.com/hediet/vscode-unicode-data/blob/main/out/ambiguous.json

package charset

import (
	"sort"
	"strings"
	"unicode"
)

// AmbiguousTable matches confusable runes with the ASCII characters they resemble
// for a given locale.
type AmbiguousTable struct {
	Confusable []rune           // sorted slice of confusable rune codepoints
	With       []rune           // parallel slice: ASCII char each Confusable[i] looks like
	Locale     string
	RangeTable *unicode.RangeTable // fast pre-filter before binary search
}

// TablesForLocale returns the ambiguous-character tables for the given locale.
// locale should be a BCP 47 tag such as "ko", "ja", "zh-hans", "en", etc.
// The returned slice always includes the locale-specific table (or the _default
// fallback) followed by the _common table.
func TablesForLocale(locale string) []*AmbiguousTable {
	key := locale
	var table *AmbiguousTable
	for len(key) > 0 {
		if t, ok := AmbiguousCharacters[key]; ok {
			table = t
			break
		}
		idx := strings.LastIndexAny(key, "-_")
		if idx < 0 {
			key = ""
		} else {
			key = key[:idx]
		}
	}
	// zh-CN → zh-hans fallback
	if table == nil && (locale == "zh-CN" || locale == "zh_CN") {
		table = AmbiguousCharacters["zh-hans"]
	}
	if table == nil && strings.HasPrefix(locale, "zh") {
		table = AmbiguousCharacters["zh-hant"]
	}
	if table == nil {
		table = AmbiguousCharacters["_default"]
	}
	return []*AmbiguousTable{table, AmbiguousCharacters["_common"]}
}

// IsAmbiguous reports whether r is an ambiguous rune in the given tables.
// If it is, *confusableTo is set to the ASCII character it visually resembles.
func IsAmbiguous(r rune, confusableTo *rune, tables ...*AmbiguousTable) bool {
	for _, table := range tables {
		if table == nil {
			continue
		}
		if !unicode.Is(table.RangeTable, r) {
			continue
		}
		i := sort.Search(len(table.Confusable), func(i int) bool {
			return table.Confusable[i] >= r
		})
		if i < len(table.Confusable) && table.Confusable[i] == r {
			*confusableTo = table.With[i]
			return true
		}
	}
	return false
}
