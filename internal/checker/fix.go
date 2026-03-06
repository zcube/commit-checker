package checker

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
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

// fixCoauthor: allow list에 없는 Co-authored-by: 트레일러 줄을 제거.
func fixCoauthor(content string, changes []string, cfg *config.CommitMessageConfig) (string, []string) {
	lines := strings.Split(content, "\n")
	var kept []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "co-authored-by:") {
			// 허용된 이메일이면 유지
			if len(cfg.CoauthorAllowEmails) > 0 {
				email := config.ExtractCoauthorEmail(trimmed)
				if cfg.CoauthorEmailAllowed(email) {
					kept = append(kept, line)
					continue
				}
			}
			changes = append(changes, fmt.Sprintf("line %d: removed co-author trailer: %q", i+1, trimmed))
			continue
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
					changes = append(changes, fmt.Sprintf(
						"line %d, col %d: replaced invisible space %s with regular space",
						lineIdx+1, col, desc,
					))
					sb.WriteRune(' ')
				} else {
					changes = append(changes, fmt.Sprintf(
						"line %d, col %d: removed invisible character %s",
						lineIdx+1, col, desc,
					))
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
				changes = append(changes, fmt.Sprintf(
					"line %d, col %d: replaced ambiguous U+%04X %q with ASCII %q (U+%04X)",
					lineIdx+1, col, r, string(r), string(confusableTo), confusableTo,
				))
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
			changes = append(changes, fmt.Sprintf(
				"line %d, col %d: removed invalid UTF-8 byte 0x%02X",
				lineIdx, col, b[0],
			))
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
