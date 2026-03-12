package checker

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/emoji"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/langdetect"
)


// CheckMsg: 설정된 모든 정책 위반 여부를 커밋 메시지에서 검사.
// content는 커밋 메시지 파일(예: .git/COMMIT_EDITMSG)의 원시 텍스트.
func CheckMsg(cfg *config.Config, content string) []string {
	if !cfg.CommitMessage.IsEnabled() {
		return nil
	}

	var errs []string

	if cfg.CommitMessage.IsNoAICoauthor() {
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
	if cfg.CommitMessage.IsNoEmoji() {
		errs = append(errs, checkMsgEmoji(content)...)
	}
	if cfg.CommitMessage.LanguageCheck.IsEnabled() {
		errs = append(errs, checkMsgLanguage(content, &cfg.CommitMessage.LanguageCheck)...)
	}
	if cfg.CommitMessage.ConventionalCommit.IsEnabled() {
		errs = append(errs, checkConventional(content, &cfg.CommitMessage.ConventionalCommit)...)
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
			errs = append(errs, i18n.T("msg.ai_coauthor_error", map[string]interface{}{
				"Line":    i + 1,
				"Trailer": trimmed,
			}))
		}
	}
	return errs
}

// checkInvisibleChars 는 비가시/비표준 공백 문자를 감지합니다.
// Gitea 의 InvisibleRanges 테이블을 기반으로 하며 BOM (U+FEFF) 은 허용됩니다.
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
				errs = append(errs, i18n.T("msg.invisible_char_error", map[string]interface{}{
					"Line": lineIdx + 1,
					"Col":  col,
					"Desc": desc,
				}))
			}
		}
	}
	return errs
}

// checkAmbiguousChars 는 ASCII 문자와 시각적으로 구별할 수 없지만 코드포인트가 다른
// 유니코드 문자를 감지합니다 (예: 키릴 문자 А vs 라틴 문자 A).
// Gitea 의 AmbiguousCharacters 로케일 테이블을 사용합니다.
func checkAmbiguousChars(content string, tables []*charset.AmbiguousTable) []string {
	var errs []string
	for lineIdx, line := range strings.Split(content, "\n") {
		col := 0
		for _, r := range line {
			col++
			var confusableTo rune
			if charset.IsAmbiguous(r, &confusableTo, tables...) {
				errs = append(errs, i18n.T("msg.ambiguous_char_error", map[string]interface{}{
					"Line":      lineIdx + 1,
					"Col":       col,
					"CharCode":  fmt.Sprintf("%04X", r),
					"Char":      string(r),
					"ASCII":     string(confusableTo),
					"ASCIICode": fmt.Sprintf("%04X", confusableTo),
				}))
			}
		}
	}
	return errs
}

// checkBadRunes 는 커밋 메시지에서 잘못된 UTF-8 바이트 시퀀스를 감지합니다.
func checkBadRunes(content string) []string {
	var errs []string
	bytes := []byte(content)
	lineIdx := 1
	col := 1
	for len(bytes) > 0 {
		r, size := utf8.DecodeRune(bytes)
		if r == utf8.RuneError && size == 1 {
			errs = append(errs, i18n.T("msg.bad_rune_error", map[string]interface{}{
				"Line": lineIdx,
				"Col":  col,
				"Byte": fmt.Sprintf("%02X", bytes[0]),
			}))
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

// checkMsgLanguage 는 커밋 메시지 본문이 필수 언어로 작성되었는지 확인합니다.
// 제목 줄(첫 번째 줄)과 빈 구분자 이후의 본문 줄을 모두 검사합니다.
// 설정된 skip_prefixes 로 시작하는 제목 줄은 검사를 건너뜁니다.
func checkMsgLanguage(content string, cfg *config.CommitMessageLanguageConfig) []string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}

	subject := strings.TrimSpace(lines[0])

	// 제목이 건너뛸 접두사로 시작하면 전체 메시지를 건너뜁니다.
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
			errs = append(errs, i18n.T("msg.language_error", map[string]interface{}{
				"Line":     lineNum,
				"Language": required,
				"Detected": detected,
				"Text":     truncate(text, 80),
			}))
		}
	}

	// 제목 줄 검사.
	checkLine(1, subject)

	// 본문 줄 검사 (제목과 본문 사이 빈 구분자 줄은 건너뜁니다).
	for i := 1; i < len(lines); i++ {
		checkLine(i+1, lines[i])
	}

	return errs
}

// checkMsgEmoji: 커밋 메시지에서 이모지 문자를 감지하여 에러 목록 반환.
func checkMsgEmoji(content string) []string {
	var errs []string
	emojis := emoji.FindEmojis(content)
	for _, e := range emojis {
		errs = append(errs, i18n.T("msg.emoji_error", map[string]interface{}{
			"Line":     e.Line,
			"Col":      e.Col,
			"Char":     e.Char,
			"CharCode": fmt.Sprintf("%04X", e.Code),
		}))
	}
	return errs
}
