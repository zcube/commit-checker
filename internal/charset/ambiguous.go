// 모호 문자 감지, Gitea (MIT 라이선스)에서 적용:
// https://github.com/go-gitea/gitea/blob/main/modules/charset/ambiguous.go
// https://github.com/go-gitea/gitea/blob/main/modules/charset/ambiguous_gen.go
//
// 원본 데이터: https://github.com/hediet/vscode-unicode-data/blob/main/out/ambiguous.json

package charset

import (
	"sort"
	"strings"
	"unicode"
)

// AmbiguousTable: 주어진 로케일에서 ASCII 문자와 혼동 가능한 룬을 매칭하는 테이블.
type AmbiguousTable struct {
	Confusable []rune           // 혼동 가능한 룬 코드포인트의 정렬된 슬라이스
	With       []rune           // 병렬 슬라이스: 각 Confusable[i]가 닮은 ASCII 문자
	Locale     string
	RangeTable *unicode.RangeTable // 이진 검색 전 빠른 사전 필터
}

// TablesForLocale: 주어진 로케일에 대한 모호 문자 테이블을 반환.
// locale은 "ko", "ja", "zh-hans", "en" 등의 BCP 47 태그.
// 반환되는 슬라이스는 항상 로케일별 테이블(또는 _default
// 폴백)과 _common 테이블을 포함.
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
	// zh-CN → zh-hans 폴백
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

// IsAmbiguous: 주어진 테이블에서 r이 모호한 룬인지 확인.
// 모호한 경우 *confusableTo에 시각적으로 닮은 ASCII 문자를 설정.
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
