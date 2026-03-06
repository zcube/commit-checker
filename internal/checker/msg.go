package checker

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/langdetect"
)


// CheckMsg inspects a commit message for all configured policy violations.
// content is the raw text of the commit message file (e.g. .git/COMMIT_EDITMSG).
func CheckMsg(cfg *config.Config, content string) []string {
	var errs []string

	if cfg.CommitMessage.IsNoCoauthor() {
		errs = append(errs, checkCoauthor(content, &cfg.CommitMessage)...)
	}
	if cfg.CommitMessage.IsNoUnicodeSpaces() {
		errs = append(errs, checkInvisibleChars(content)...)
	}
	if cfg.CommitMessage.IsNoAmbiguousChars() {
		tables := charset.TablesForLocale(cfg.CommitMessage.Locale)
		errs = append(errs, checkAmbiguousChars(content, tables)...)
	}
	if cfg.CommitMessage.IsNoBadRunes() {
		errs = append(errs, checkBadRunes(content)...)
	}
	if cfg.CommitMessage.LanguageCheck.IsEnabled() {
		errs = append(errs, checkMsgLanguage(content, &cfg.CommitMessage.LanguageCheck)...)
	}

	return errs
}

// checkCoauthor: AI 도구 이메일 패턴과 일치하는 Co-authored-by: 트레일러 줄을 에러로 보고.
// 내장 AI 패턴(Copilot, Claude, Cursor 등) + 사용자 정의 패턴이 적용됨.
// 패턴에 해당하지 않는 일반 공동 작업자는 허용됨.
func checkCoauthor(content string, cfg *config.CommitMessageConfig) []string {
	var errs []string
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(trimmed), "co-authored-by:") {
			continue
		}
		email := config.ExtractCoauthorEmail(trimmed)
		if cfg.CoauthorShouldRemove(email) {
			errs = append(errs, fmt.Sprintf(
				"commit message line %d: AI co-author trailer must be removed: %q",
				i+1, trimmed,
			))
		}
	}
	return errs
}

// checkInvisibleChars reports invisible / non-standard space characters.
// Detection is based on Gitea's InvisibleRanges table. BOM (U+FEFF) is allowed.
func checkInvisibleChars(content string) []string {
	var errs []string
	for lineIdx, line := range strings.Split(content, "\n") {
		col := 0
		for _, r := range line {
			col++
			if charset.IsInvisible(r) {
				name := charset.InvisibleName(r)
				desc := fmt.Sprintf("U+%04X", r)
				if name != "" {
					desc += " " + name
				}
				errs = append(errs, fmt.Sprintf(
					"commit message line %d, col %d: invisible/non-standard space character %s is not allowed",
					lineIdx+1, col, desc,
				))
			}
		}
	}
	return errs
}

// checkAmbiguousChars reports Unicode characters that are visually indistinguishable
// from ASCII characters but have different codepoints (e.g. Cyrillic А vs Latin A).
// Detection uses Gitea's AmbiguousCharacters locale tables.
func checkAmbiguousChars(content string, tables []*charset.AmbiguousTable) []string {
	var errs []string
	for lineIdx, line := range strings.Split(content, "\n") {
		col := 0
		for _, r := range line {
			col++
			var confusableTo rune
			if charset.IsAmbiguous(r, &confusableTo, tables...) {
				errs = append(errs, fmt.Sprintf(
					"commit message line %d, col %d: ambiguous character U+%04X %q looks like %q (U+%04X) — replace with the ASCII character",
					lineIdx+1, col, r, string(r), string(confusableTo), confusableTo,
				))
			}
		}
	}
	return errs
}

// checkBadRunes reports invalid UTF-8 byte sequences in the commit message.
func checkBadRunes(content string) []string {
	var errs []string
	bytes := []byte(content)
	lineIdx := 1
	col := 1
	for len(bytes) > 0 {
		r, size := utf8.DecodeRune(bytes)
		if r == utf8.RuneError && size == 1 {
			errs = append(errs, fmt.Sprintf(
				"commit message line %d, col %d: invalid UTF-8 byte sequence 0x%02X",
				lineIdx, col, bytes[0],
			))
		}
		if r == '\n' {
			lineIdx++
			col = 1
		} else {
			col++
		}
		bytes = bytes[size:]
	}
	return errs
}

// checkMsgLanguage checks that the commit message body is written in the required language.
// The subject line (first line) is checked; body lines after a blank separator are also checked.
// Lines starting with configured skip_prefixes on the subject are exempt.
func checkMsgLanguage(content string, cfg *config.CommitMessageLanguageConfig) []string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}

	subject := strings.TrimSpace(lines[0])

	// Skip entire message if subject starts with a skip prefix.
	for _, prefix := range cfg.SkipPrefixes {
		if strings.HasPrefix(subject, prefix) {
			return nil
		}
	}

	required := cfg.RequiredLanguage
	minLength := cfg.MinLength

	var errs []string
	checkLine := func(lineNum int, text string) {
		text = strings.TrimSpace(text)
		ok, hasContent := langdetect.IsRequiredLanguage(text, required, minLength, nil)
		if !hasContent {
			return
		}
		if !ok {
			detected := langdetect.Dominant(text)
			errs = append(errs, fmt.Sprintf(
				"commit message line %d: must be written in %s (detected: %s): %s",
				lineNum, required, detected,
				truncate(text, 80),
			))
		}
	}

	// Check subject line.
	checkLine(1, subject)

	// Check body lines (skip the blank separator line between subject and body).
	for i := 1; i < len(lines); i++ {
		checkLine(i+1, lines[i])
	}

	return errs
}
