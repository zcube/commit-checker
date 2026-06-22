// merge.go: 베이스 설정(preset·include) 위에 상위 설정을 덮어씌워 병합하는 로직.
package config

// mergeConfigs: base를 기반으로 overlay 설정을 덮어씌워 병합된 Config를 반환합니다.
// overlay에서 명시적으로 설정된 값이 우선합니다.
// 목록 필드(allowed_words, global_ignore, custom_rules 등)는 base + overlay를 합칩니다.
func mergeConfigs(base, overlay *Config) Config {
	result := *overlay

	// 최상위 enabled 병합: 본문(overlay) 값이 우선 (리포 단위 opt-out)
	mergeBoolPtr(&result.Enabled, base.Enabled)

	// 주석 언어 검사 설정 병합
	mergeBoolPtr(&result.CommentLanguage.Enabled, base.CommentLanguage.Enabled)
	mergeBoolPtr(&result.CommentLanguage.NoEmoji, base.CommentLanguage.NoEmoji)
	mergeBoolPtr(&result.CommentLanguage.CheckStrings, base.CommentLanguage.CheckStrings)
	mergeBoolPtr(&result.CommentLanguage.SkipTechnicalStrings, base.CommentLanguage.SkipTechnicalStrings)
	mergeString(&result.CommentLanguage.RequiredLanguage, base.CommentLanguage.RequiredLanguage)
	mergeString(&result.CommentLanguage.CheckMode, base.CommentLanguage.CheckMode)
	mergeString(&result.CommentLanguage.Locale, base.CommentLanguage.Locale)
	mergeString(&result.CommentLanguage.AllowedWordsFile, base.CommentLanguage.AllowedWordsFile)
	mergeString(&result.CommentLanguage.AllowedWordsURL, base.CommentLanguage.AllowedWordsURL)
	mergeInt(&result.CommentLanguage.MinLength, base.CommentLanguage.MinLength)
	result.CommentLanguage.AllowedWords = append(base.CommentLanguage.AllowedWords, result.CommentLanguage.AllowedWords...)
	result.CommentLanguage.SkipDirectives = append(base.CommentLanguage.SkipDirectives, result.CommentLanguage.SkipDirectives...)
	result.CommentLanguage.IgnoreFiles = append(base.CommentLanguage.IgnoreFiles, result.CommentLanguage.IgnoreFiles...)
	if len(result.CommentLanguage.Languages) == 0 {
		result.CommentLanguage.Languages = base.CommentLanguage.Languages
	}
	if len(result.CommentLanguage.Extensions) == 0 {
		result.CommentLanguage.Extensions = base.CommentLanguage.Extensions
	}
	if len(result.CommentLanguage.FileLanguages) == 0 {
		result.CommentLanguage.FileLanguages = base.CommentLanguage.FileLanguages
	}

	// 커밋 메시지 설정 병합
	mergeBoolPtr(&result.CommitMessage.Enabled, base.CommitMessage.Enabled)
	mergeBoolPtr(&result.CommitMessage.NoAICoauthor, base.CommitMessage.NoAICoauthor)
	mergeBoolPtr(&result.CommitMessage.NoUnicodeSpaces, base.CommitMessage.NoUnicodeSpaces)
	mergeBoolPtr(&result.CommitMessage.NoAmbiguousChars, base.CommitMessage.NoAmbiguousChars)
	mergeBoolPtr(&result.CommitMessage.NoBadRunes, base.CommitMessage.NoBadRunes)
	mergeBoolPtr(&result.CommitMessage.NoEmoji, base.CommitMessage.NoEmoji)
	mergeString(&result.CommitMessage.Locale, base.CommitMessage.Locale)
	result.CommitMessage.CoauthorRemoveEmails = append(base.CommitMessage.CoauthorRemoveEmails, result.CommitMessage.CoauthorRemoveEmails...)
	mergeBoolPtr(&result.CommitMessage.LanguageCheck.Enabled, base.CommitMessage.LanguageCheck.Enabled)
	mergeString(&result.CommitMessage.LanguageCheck.Locale, base.CommitMessage.LanguageCheck.Locale)
	mergeString(&result.CommitMessage.LanguageCheck.RequiredLanguage, base.CommitMessage.LanguageCheck.RequiredLanguage)
	mergeInt(&result.CommitMessage.LanguageCheck.MinLength, base.CommitMessage.LanguageCheck.MinLength)
	if len(result.CommitMessage.LanguageCheck.SkipPrefixes) == 0 {
		result.CommitMessage.LanguageCheck.SkipPrefixes = base.CommitMessage.LanguageCheck.SkipPrefixes
	}
	mergeBoolPtr(&result.CommitMessage.ConventionalCommit.Enabled, base.CommitMessage.ConventionalCommit.Enabled)
	mergeBoolPtr(&result.CommitMessage.ConventionalCommit.RequireScope, base.CommitMessage.ConventionalCommit.RequireScope)
	mergeBoolPtr(&result.CommitMessage.ConventionalCommit.AllowMergeCommits, base.CommitMessage.ConventionalCommit.AllowMergeCommits)
	mergeBoolPtr(&result.CommitMessage.ConventionalCommit.AllowRevertCommits, base.CommitMessage.ConventionalCommit.AllowRevertCommits)
	mergeString(&result.CommitMessage.ConventionalCommit.Locale, base.CommitMessage.ConventionalCommit.Locale)
	if len(result.CommitMessage.ConventionalCommit.Types) == 0 {
		result.CommitMessage.ConventionalCommit.Types = base.CommitMessage.ConventionalCommit.Types
	}
	if len(result.CommitMessage.ConventionalCommit.TypeAliases) == 0 {
		result.CommitMessage.ConventionalCommit.TypeAliases = base.CommitMessage.ConventionalCommit.TypeAliases
	}
	mergeBoolPtr(&result.CommitMessage.SubjectLimit.Enabled, base.CommitMessage.SubjectLimit.Enabled)
	mergeInt(&result.CommitMessage.SubjectLimit.MaxLength, base.CommitMessage.SubjectLimit.MaxLength)
	mergeBoolPtr(&result.CommitMessage.BodyLineLimit.Enabled, base.CommitMessage.BodyLineLimit.Enabled)
	mergeInt(&result.CommitMessage.BodyLineLimit.MaxLength, base.CommitMessage.BodyLineLimit.MaxLength)

	// 바이너리 파일 설정 병합
	mergeBoolPtr(&result.BinaryFile.Enabled, base.BinaryFile.Enabled)
	result.BinaryFile.IgnoreFiles = append(base.BinaryFile.IgnoreFiles, result.BinaryFile.IgnoreFiles...)

	// 인코딩 설정 병합
	mergeBoolPtr(&result.Encoding.Enabled, base.Encoding.Enabled)
	mergeBoolPtr(&result.Encoding.RequireUTF8, base.Encoding.RequireUTF8)
	mergeBoolPtr(&result.Encoding.NoInvisibleChars, base.Encoding.NoInvisibleChars)
	mergeBoolPtr(&result.Encoding.NoAmbiguousChars, base.Encoding.NoAmbiguousChars)
	mergeString(&result.Encoding.Locale, base.Encoding.Locale)
	result.Encoding.IgnoreFiles = append(base.Encoding.IgnoreFiles, result.Encoding.IgnoreFiles...)

	// EditorConfig 설정 병합
	mergeBoolPtr(&result.EditorConfig.Enabled, base.EditorConfig.Enabled)
	result.EditorConfig.IgnoreFiles = append(base.EditorConfig.IgnoreFiles, result.EditorConfig.IgnoreFiles...)

	// Exceptions: 항상 합치기
	result.Exceptions.GlobalIgnore = append(base.Exceptions.GlobalIgnore, result.Exceptions.GlobalIgnore...)
	result.Exceptions.CommentLanguageIgnore = append(base.Exceptions.CommentLanguageIgnore, result.Exceptions.CommentLanguageIgnore...)

	// CustomRules: 항상 합치기 (base 규칙 먼저)
	result.CustomRules.CommitMessage = append(base.CustomRules.CommitMessage, result.CustomRules.CommitMessage...)
	result.CustomRules.Diff = append(base.CustomRules.Diff, result.CustomRules.Diff...)

	// 보호 경로 설정 병합: enabled 는 한쪽이라도 켜져 있으면 유지, paths 는 합치기
	if !result.ProtectedPaths.Enabled {
		result.ProtectedPaths.Enabled = base.ProtectedPaths.Enabled
	}
	result.ProtectedPaths.Paths = append(base.ProtectedPaths.Paths, result.ProtectedPaths.Paths...)

	// 개선 가이드 설정 병합
	mergeBoolPtr(&result.Guide.Enabled, base.Guide.Enabled)

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
