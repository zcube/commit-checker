package checker

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

// FixResult 는 수정된 커밋 메시지와 변경 사항 설명 목록을 담습니다.
type FixResult struct {
	Original string
	Fixed    string
	Changes  []string
}

// NeedsFixing 는 커밋 메시지에 자동 수정 가능한 위반이 있는지 확인합니다.
func (r FixResult) NeedsFixing() bool { return len(r.Changes) > 0 }

// FixMsg 는 커밋 메시지에 자동 수정 가능한 모든 교정을 적용합니다.
// 구조적 위반(공동 작성자, 비가시 문자, 모호한 문자, 잘못된 룬)만 수정합니다.
// 본문의 언어 위반은 자동 수정되지 않습니다.
func FixMsg(cfg *config.Config, content string) FixResult {
	result := FixResult{Original: content, Fixed: content}

	// 잘못된 룬을 먼저 처리해야 합니다. 다른 수정기는 range-over-string 을 사용하여
	// 잘못된 UTF-8 바이트를 U+FFFD 로 자동 교체하므로 감지가 불가능해집니다.
	if cfg.CommitMessage.IsNoBadRunes() {
		result.Fixed, result.Changes = fixBadRunes(result.Fixed, result.Changes)
	}
	if cfg.CommitMessage.IsNoAICoauthor() {
		result.Fixed, result.Changes = fixCoauthor(result.Fixed, result.Changes, &cfg.CommitMessage)
	}
	if cfg.CommitMessage.IsNoUnicodeSpaces() {
		result.Fixed, result.Changes = fixInvisibleChars(result.Fixed, result.Changes)
	}
	if cfg.CommitMessage.IsNoAmbiguousChars() {
		tables := charset.TablesForLocale(cfg.CommitMessage.Locale)
		result.Fixed, result.Changes = fixAmbiguousChars(result.Fixed, result.Changes, tables)
	}

	return result
}

// fixCoauthor: AI 패턴과 일치하는 Co-authored-by: 트레일러 줄을 제거.
// 패턴에 해당하지 않는 일반 공동 작업자 줄은 유지됨.
func fixCoauthor(content string, changes []string, cfg *config.CommitMessageConfig) (string, []string) {
	lines := strings.Split(content, "\n")
	var kept []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "co-authored-by:") {
			email := config.ExtractCoauthorEmail(trimmed)
			if cfg.CoauthorShouldRemove(email) {
				changes = append(changes, i18n.T("fix.removed_ai_coauthor", map[string]any{
					"Line":    i + 1,
					"Trailer": trimmed,
				}))
				continue
			}
		}
		kept = append(kept, line)
	}
	// 트레일러 제거 후 남은 빈 줄을 정리하되, 원본에 마지막 개행이 있으면 유지합니다.
	result := strings.Join(kept, "\n")
	// 연속된 마지막 개행을 최대 하나로 축소합니다.
	for strings.HasSuffix(result, "\n\n") {
		result = result[:len(result)-1]
	}
	return result, changes
}

// fixInvisibleChars 는 비가시/비표준 공백 문자를 일반 공백으로 교체합니다.
// 양방향 제어 문자와 영폭 문자는 완전히 제거합니다.
func fixInvisibleChars(content string, changes []string) (string, []string) {
	var sb strings.Builder
	for lineIdx, line := range strings.Split(content, "\n") {
		if lineIdx > 0 {
			sb.WriteByte('\n')
		}
		col := 0
		for _, r := range line {
			col++
			if charset.IsInvisible(r) {
				name := charset.InvisibleName(r)
				desc := fmt.Sprintf("U+%04X", r)
				if name != "" {
					desc += " " + name
				}
				// 공백 변형 → 일반 공백; 제어/영폭 문자 → 제거
				if isSpaceVariant(r) {
					changes = append(changes, i18n.T("fix.replaced_invisible_space", map[string]any{
						"Line": lineIdx + 1,
						"Col":  col,
						"Desc": desc,
					}))
					sb.WriteRune(' ')
				} else {
					changes = append(changes, i18n.T("fix.removed_invisible_char", map[string]any{
						"Line": lineIdx + 1,
						"Col":  col,
						"Desc": desc,
					}))
				}
				continue
			}
			sb.WriteRune(r)
		}
	}
	return sb.String(), changes
}

// isSpaceVariant 는 비가시 문자 중 의미적으로 공백인 경우(U+0020 으로 교체) true 를 반환합니다.
// 제어 문자(제거 대상)는 false 를 반환합니다.
func isSpaceVariant(r rune) bool {
	switch r {
	case 0x00A0, // 줄 바꿈 없는 공백
		0x1680,                                          // 오감 공백
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, // em/en/thin 공백 변형
		0x2006, 0x2007, 0x2008, 0x2009, 0x200A,
		0x202F, // 좁은 줄 바꿈 없는 공백
		0x205F, // 수학 중간 공백
		0x3000: // 표의문자 공백
		return true
	}
	return false
}

// fixAmbiguousChars 는 모호한 유니코드 문자를 ASCII 유사 문자로 교체합니다.
func fixAmbiguousChars(content string, changes []string, tables []*charset.AmbiguousTable) (string, []string) {
	var sb strings.Builder
	for lineIdx, line := range strings.Split(content, "\n") {
		if lineIdx > 0 {
			sb.WriteByte('\n')
		}
		col := 0
		for _, r := range line {
			col++
			var confusableTo rune
			if charset.IsAmbiguous(r, &confusableTo, tables...) {
				changes = append(changes, i18n.T("fix.replaced_ambiguous_char", map[string]any{
					"Line":      lineIdx + 1,
					"Col":       col,
					"CharCode":  fmt.Sprintf("%04X", r),
					"Char":      string(r),
					"ASCII":     string(confusableTo),
					"ASCIICode": fmt.Sprintf("%04X", confusableTo),
				}))
				sb.WriteRune(confusableTo)
				continue
			}
			sb.WriteRune(r)
		}
	}
	return sb.String(), changes
}

// fixBadRunes 는 잘못된 UTF-8 바이트 시퀀스를 제거합니다.
func fixBadRunes(content string, changes []string) (string, []string) {
	b := []byte(content)
	var sb strings.Builder
	lineIdx := 1
	col := 1
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			changes = append(changes, i18n.T("fix.removed_bad_rune", map[string]any{
				"Line": lineIdx,
				"Col":  col,
				"Byte": fmt.Sprintf("%02X", b[0]),
			}))
			b = b[size:]
			continue
		}
		if r == '\n' {
			lineIdx++
			col = 1
		} else {
			col++
		}
		sb.WriteRune(r)
		b = b[size:]
	}
	return sb.String(), changes
}
