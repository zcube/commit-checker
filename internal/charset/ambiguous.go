// 모호한 문자 감지 로직은 Gitea (MIT License) 에서 응용했습니다:
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

// AmbiguousTable 는 특정 로케일에서 ASCII 문자와 혼동될 수 있는 룬을 매핑합니다.
type AmbiguousTable struct {
	Confusable []rune           // 혼동 가능한 룬 코드포인트의 정렬된 슬라이스
	With       []rune           // 병렬 슬라이스: Confusable[i] 가 닮은 ASCII 문자
	Locale     string
	RangeTable *unicode.RangeTable // 이진 탐색 전 빠른 사전 필터
}

// TablesForLocale 는 주어진 로케일의 모호한 문자 테이블을 반환합니다.
// locale 은 "ko", "ja", "zh-hans", "en" 등 BCP 47 태그여야 합니다.
// 반환 슬라이스는 항상 로케일별 테이블(또는 _default 폴백)과 _common 테이블을 포함합니다.
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

// IsAmbiguous 는 r 이 주어진 테이블에서 모호한 룬인지 확인합니다.
// 해당하면 *confusableTo 에 시각적으로 닮은 ASCII 문자를 설정합니다.
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
