// Package directive parses commit-checker inline directives embedded in
// source-code comments. Directives are comment texts that start with the
// prefix "commit-checker:" (case-insensitive).
//
// Supported directives
//
//	commit-checker:disable            disable language checking from this point
//	commit-checker:disable:lang=<L>   disable default; use language L instead
//	commit-checker:enable             re-enable language checking
//	commit-checker:ignore             skip the immediately following comment
//	commit-checker:lang=<L>           switch required language to L from here
//	commit-checker:file-lang=<L>      set required language for the whole file
//
// <L> accepts the same values as required_language (korean, english,
// japanese, chinese, any) and locale codes (ko, en, ja, zh, zh-hans, zh-hant).
package directive

import (
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/langdetect"
)

const prefix = "commit-checker:"

// CommentState describes how a single comment should be handled after
// directive processing.
type CommentState struct {
	// Skip is true when the comment should not be language-checked at all.
	Skip bool
	// Language is the effective required language for this comment.
	// Empty string means "use the caller's default".
	Language string
}

// Analyze walks comments in source order and returns a CommentState for each.
// defaultLang is the per-file required language (already resolved from config).
func Analyze(comments []comment.Comment, defaultLang string) []CommentState {
	states := make([]CommentState, len(comments))

	disabled := false    // commit-checker:disable is in effect
	disabledLang := ""   // language override while disabled (empty = skip entirely)
	skipNext := false    // commit-checker:ignore seen; skip next real comment
	langOverride := ""   // commit-checker:lang= override (empty = use defaultLang)
	fileLang := ""       // commit-checker:file-lang= sets for whole file

	for i, c := range comments {
		text := strings.TrimSpace(c.Text)

		if !isDirective(text) {
			if fileLang != "" {
				// file-lang overrides everything except an active disable
				if disabled {
					states[i] = CommentState{Skip: disabledLang == "", Language: disabledLang}
				} else if skipNext {
					states[i] = CommentState{Skip: true}
					skipNext = false
				} else {
					lang := fileLang
					if langOverride != "" {
						lang = langOverride
					}
					states[i] = CommentState{Language: lang}
				}
			} else if disabled {
				states[i] = CommentState{Skip: disabledLang == "", Language: disabledLang}
				skipNext = false
			} else if skipNext {
				states[i] = CommentState{Skip: true}
				skipNext = false
			} else {
				states[i] = CommentState{Language: langOverride}
			}
			continue
		}

		// It is a directive — always skip it as a checkable comment.
		states[i] = CommentState{Skip: true}

		lower := strings.ToLower(text)

		switch {
		case strings.HasPrefix(lower, prefix+"file-lang="):
			fileLang = resolveLanguage(text[len(prefix+"file-lang="):])

		case strings.HasPrefix(lower, prefix+"disable:lang="):
			disabled = true
			disabledLang = resolveLanguage(text[len(prefix+"disable:lang="):])

		case strings.HasPrefix(lower, prefix+"disable"):
			disabled = true
			disabledLang = ""

		case strings.HasPrefix(lower, prefix+"enable"):
			disabled = false
			disabledLang = ""

		case strings.HasPrefix(lower, prefix+"ignore"):
			skipNext = true

		case strings.HasPrefix(lower, prefix+"lang="):
			langOverride = resolveLanguage(text[len(prefix+"lang="):])
		}
	}

	// Resolve empty Language fields to defaultLang.
	for i := range states {
		if !states[i].Skip && states[i].Language == "" {
			states[i].Language = defaultLang
		}
	}

	return states
}

// IsDirective reports whether comment text is a commit-checker directive.
func IsDirective(text string) bool {
	return isDirective(strings.TrimSpace(text))
}

func isDirective(text string) bool {
	return strings.HasPrefix(strings.ToLower(text), prefix)
}

// resolveLanguage normalises a language value: locale codes (ko, en, ja, zh…)
// are mapped to full names; unknown values are returned as-is (lowercased).
func resolveLanguage(raw string) string {
	raw = strings.TrimSpace(raw)
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(raw)); mapped != "" {
		return mapped
	}
	return strings.ToLower(raw)
}
