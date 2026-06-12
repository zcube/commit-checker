// merge.go: 전역/프리셋/프로젝트 설정을 우선순위에 따라 병합하는 로직.
package config

// mergeConfigs: global을 기반으로 project 설정을 덮어씌워 병합된 Config를 반환합니다.
// project에서 명시적으로 설정된 값이 우선합니다.
// 목록 필드(allowed_words, global_ignore, custom_rules 등)는 global + project를 합칩니다.
func mergeConfigs(global, project *Config) Config {
	result := *project

	// 주석 언어 검사 설정 병합
	mergeBoolPtr(&result.CommentLanguage.Enabled, global.CommentLanguage.Enabled)
	mergeBoolPtr(&result.CommentLanguage.NoEmoji, global.CommentLanguage.NoEmoji)
	mergeBoolPtr(&result.CommentLanguage.CheckStrings, global.CommentLanguage.CheckStrings)
	mergeBoolPtr(&result.CommentLanguage.SkipTechnicalStrings, global.CommentLanguage.SkipTechnicalStrings)
	mergeString(&result.CommentLanguage.RequiredLanguage, global.CommentLanguage.RequiredLanguage)
	mergeString(&result.CommentLanguage.CheckMode, global.CommentLanguage.CheckMode)
	mergeString(&result.CommentLanguage.Locale, global.CommentLanguage.Locale)
	mergeString(&result.CommentLanguage.AllowedWordsFile, global.CommentLanguage.AllowedWordsFile)
	mergeString(&result.CommentLanguage.AllowedWordsURL, global.CommentLanguage.AllowedWordsURL)
	mergeInt(&result.CommentLanguage.MinLength, global.CommentLanguage.MinLength)
	result.CommentLanguage.AllowedWords = append(global.CommentLanguage.AllowedWords, result.CommentLanguage.AllowedWords...)
	result.CommentLanguage.SkipDirectives = append(global.CommentLanguage.SkipDirectives, result.CommentLanguage.SkipDirectives...)
	result.CommentLanguage.IgnoreFiles = append(global.CommentLanguage.IgnoreFiles, result.CommentLanguage.IgnoreFiles...)
	if len(result.CommentLanguage.Languages) == 0 {
		result.CommentLanguage.Languages = global.CommentLanguage.Languages
	}
	if len(result.CommentLanguage.Extensions) == 0 {
		result.CommentLanguage.Extensions = global.CommentLanguage.Extensions
	}
	if len(result.CommentLanguage.FileLanguages) == 0 {
		result.CommentLanguage.FileLanguages = global.CommentLanguage.FileLanguages
	}

	// 커밋 메시지 설정 병합
	mergeBoolPtr(&result.CommitMessage.Enabled, global.CommitMessage.Enabled)
	mergeBoolPtr(&result.CommitMessage.NoAICoauthor, global.CommitMessage.NoAICoauthor)
	mergeBoolPtr(&result.CommitMessage.NoUnicodeSpaces, global.CommitMessage.NoUnicodeSpaces)
	mergeBoolPtr(&result.CommitMessage.NoAmbiguousChars, global.CommitMessage.NoAmbiguousChars)
	mergeBoolPtr(&result.CommitMessage.NoBadRunes, global.CommitMessage.NoBadRunes)
	mergeBoolPtr(&result.CommitMessage.NoEmoji, global.CommitMessage.NoEmoji)
	mergeString(&result.CommitMessage.Locale, global.CommitMessage.Locale)
	result.CommitMessage.CoauthorRemoveEmails = append(global.CommitMessage.CoauthorRemoveEmails, result.CommitMessage.CoauthorRemoveEmails...)

	// 바이너리 파일 설정 병합
	mergeBoolPtr(&result.BinaryFile.Enabled, global.BinaryFile.Enabled)
	result.BinaryFile.IgnoreFiles = append(global.BinaryFile.IgnoreFiles, result.BinaryFile.IgnoreFiles...)

	// 인코딩 설정 병합
	mergeBoolPtr(&result.Encoding.Enabled, global.Encoding.Enabled)
	mergeBoolPtr(&result.Encoding.RequireUTF8, global.Encoding.RequireUTF8)
	mergeBoolPtr(&result.Encoding.NoInvisibleChars, global.Encoding.NoInvisibleChars)
	mergeBoolPtr(&result.Encoding.NoAmbiguousChars, global.Encoding.NoAmbiguousChars)
	mergeString(&result.Encoding.Locale, global.Encoding.Locale)
	result.Encoding.IgnoreFiles = append(global.Encoding.IgnoreFiles, result.Encoding.IgnoreFiles...)

	// EditorConfig 설정 병합
	mergeBoolPtr(&result.EditorConfig.Enabled, global.EditorConfig.Enabled)
	result.EditorConfig.IgnoreFiles = append(global.EditorConfig.IgnoreFiles, result.EditorConfig.IgnoreFiles...)

	// Exceptions: 항상 합치기
	result.Exceptions.GlobalIgnore = append(global.Exceptions.GlobalIgnore, result.Exceptions.GlobalIgnore...)
	result.Exceptions.CommentLanguageIgnore = append(global.Exceptions.CommentLanguageIgnore, result.Exceptions.CommentLanguageIgnore...)

	// CustomRules: 항상 합치기 (global 규칙 먼저)
	result.CustomRules.CommitMessage = append(global.CustomRules.CommitMessage, result.CustomRules.CommitMessage...)
	result.CustomRules.Diff = append(global.CustomRules.Diff, result.CustomRules.Diff...)

	// 개선 가이드 설정 병합
	mergeBoolPtr(&result.Guide.Enabled, global.Guide.Enabled)

	return result
}

// mergeBoolPtr: dst가 nil이면 src 값을 사용합니다.
func mergeBoolPtr(dst **bool, src *bool) {
	if *dst == nil && src != nil {
		*dst = src
	}
}

// mergeString: dst가 빈 문자열이면 src 값을 사용합니다.
func mergeString(dst *string, src string) {
	if *dst == "" && src != "" {
		*dst = src
	}
}

// mergeInt: dst가 0이면 src 값을 사용합니다.
func mergeInt(dst *int, src int) {
	if *dst == 0 && src != 0 {
		*dst = src
	}
}
