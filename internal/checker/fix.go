package checker

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)


// FixResult holds a fixed commit message and a description of every change made.
type FixResult struct {
	Original string
	Fixed    string
	Changes  []string
}

// NeedsFixing reports whether the commit message has any auto-fixable violations.
func (r FixResult) NeedsFixing() bool { return len(r.Changes) > 0 }

// FixMsg applies all auto-fixable corrections to a commit message.
// It only fixes structural violations (co-author, invisible chars, ambiguous chars, bad runes).
// Language violations in the body are NOT auto-fixable.
func FixMsg(cfg *config.Config, content string) FixResult {
	result := FixResult{Original: content, Fixed: content}

	// Bad runes must run first: other fixers use range-over-string which silently
	// replaces invalid UTF-8 bytes with U+FFFD, hiding them from detection.
	if cfg.CommitMessage.IsNoBadRunes() {
		result.Fixed, result.Changes = fixBadRunes(result.Fixed, result.Changes)
	}
	if cfg.CommitMessage.IsNoCoauthor() {
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
				changes = append(changes, i18n.T("fix.removed_ai_coauthor", map[string]interface{}{
					"Line":    i + 1,
					"Trailer": trimmed,
				}))
				continue
			}
		}
		kept = append(kept, line)
	}
	// Remove trailing blank lines that may have been left after removing trailers,
	// but preserve the final newline if the original had one.
	result := strings.Join(kept, "\n")
	// Collapse multiple trailing newlines to at most one.
	for strings.HasSuffix(result, "\n\n") {
		result = result[:len(result)-1]
	}
	return result, changes
}

// fixInvisibleChars replaces invisible/non-standard space characters with a regular space.
// BiDi control characters and zero-width characters are removed entirely.
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
				// Space variants → regular space; control/zero-width chars → removed
				if isSpaceVariant(r) {
					changes = append(changes, i18n.T("fix.replaced_invisible_space", map[string]interface{}{
						"Line": lineIdx + 1,
						"Col":  col,
						"Desc": desc,
					}))
					sb.WriteRune(' ')
				} else {
					changes = append(changes, i18n.T("fix.removed_invisible_char", map[string]interface{}{
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

// isSpaceVariant returns true for invisible characters that are semantically spaces
// (should be replaced with U+0020) rather than control characters (should be removed).
func isSpaceVariant(r rune) bool {
	switch r {
	case 0x00A0, // NO-BREAK SPACE
		0x1680, // OGHAM SPACE MARK
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005,
		0x2006, 0x2007, 0x2008, 0x2009, 0x200A, // various em/en/thin spaces
		0x202F, // NARROW NO-BREAK SPACE
		0x205F, // MEDIUM MATHEMATICAL SPACE
		0x3000: // IDEOGRAPHIC SPACE
		return true
	}
	return false
}

// fixAmbiguousChars replaces ambiguous Unicode characters with their ASCII lookalikes.
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
				changes = append(changes, i18n.T("fix.replaced_ambiguous_char", map[string]interface{}{
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

// fixBadRunes removes invalid UTF-8 byte sequences.
func fixBadRunes(content string, changes []string) (string, []string) {
	b := []byte(content)
	var sb strings.Builder
	lineIdx := 1
	col := 1
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			changes = append(changes, i18n.T("fix.removed_bad_rune", map[string]interface{}{
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
