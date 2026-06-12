// accessors.go: 설정 구조체들의 getter/조회 메서드 (IsEnabled, GetLocale, PolicyFor 등).
package config

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/zcube/commit-checker/internal/langdetect"
	"github.com/zcube/commit-checker/internal/logger"
)

// IsEnabled: commit-checker 전체 활성화 여부 반환 (기본값: true).
// false 이면 모든 훅 진입 커맨드(run/diff/msg/push)가 검사 없이 성공 종료합니다.
func (c *Config) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// GetLocale 은 정규화된 언어 식별자를 반환합니다 (Locale > Language 순).
func (r *FileLanguageRule) GetLocale() string {
	if v := langdetect.NormalizeLocale(r.Locale); v != "" {
		return v
	}
	return langdetect.NormalizeLocale(r.Language)
}

// IsEnabled: 주석 언어 검사 활성화 여부 반환 (기본값: true).
func (c *CommentLanguageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// GetLocale 은 정규화된 자연어 식별자를 반환합니다.
// Locale 이 비어있으면 RequiredLanguage(legacy)를, 그것도 비어있으면 "korean"을 반환합니다.
func (c *CommentLanguageConfig) GetLocale() string {
	if v := langdetect.NormalizeLocale(c.Locale); v != "" {
		return v
	}
	if v := langdetect.NormalizeLocale(c.RequiredLanguage); v != "" {
		return v
	}
	return langdetect.Korean
}

// IsNoEmoji: 주석 이모지 검사 활성화 여부 반환 (기본값: false).
func (c *CommentLanguageConfig) IsNoEmoji() bool {
	if c.NoEmoji == nil {
		return false
	}
	return *c.NoEmoji
}

// IsFullMode: check_mode가 "full"일 때 true 반환 (전체 스테이지 파일 주석 검사).
func (c *CommentLanguageConfig) IsFullMode() bool {
	return c.CheckMode == "full"
}

// IsCheckStrings 는 문자열 리터럴 언어 검사가 활성화된 경우 true 반환 (기본값: false).
func (c *CommentLanguageConfig) IsCheckStrings() bool {
	if c.CheckStrings == nil {
		return false
	}
	return *c.CheckStrings
}

// IsSkipTechnicalStrings 는 기술적 식별자 문자열을 건너뛰는 경우 true 반환 (기본값: true).
func (c *CommentLanguageConfig) IsSkipTechnicalStrings() bool {
	if c.SkipTechnicalStrings == nil {
		return true
	}
	return *c.SkipTechnicalStrings
}

// IsEnabled: 커밋 메시지 언어 검사 활성화 여부 반환 (기본값: false).
func (c *CommitMessageLanguageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetLocale 은 정규화된 자연어 식별자를 반환합니다.
// Locale > RequiredLanguage > "korean" 순으로 fallback 합니다.
func (c *CommitMessageLanguageConfig) GetLocale() string {
	if v := langdetect.NormalizeLocale(c.Locale); v != "" {
		return v
	}
	if v := langdetect.NormalizeLocale(c.RequiredLanguage); v != "" {
		return v
	}
	return langdetect.Korean
}

// IsEnabled: 컨벤셔널 커밋 검사 활성화 여부 반환 (기본값: false).
func (c *ConventionalCommitConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// IsRequireScope: 스코프 필수 여부 반환 (기본값: false).
func (c *ConventionalCommitConfig) IsRequireScope() bool {
	if c.RequireScope == nil {
		return false
	}
	return *c.RequireScope
}

// IsAllowMergeCommits: Merge 커밋 형식 검사 건너뜀 여부 반환 (기본값: true).
func (c *ConventionalCommitConfig) IsAllowMergeCommits() bool {
	if c.AllowMergeCommits == nil {
		return true
	}
	return *c.AllowMergeCommits
}

// IsAllowRevertCommits: Revert 커밋 형식 검사 건너뜀 여부 반환 (기본값: true).
func (c *ConventionalCommitConfig) IsAllowRevertCommits() bool {
	if c.AllowRevertCommits == nil {
		return true
	}
	return *c.AllowRevertCommits
}

// GetTypes: 설정된 타입 목록을 반환하며, 미설정 시 DefaultConventionalTypes 반환.
func (c *ConventionalCommitConfig) GetTypes() []string {
	if len(c.Types) > 0 {
		return c.Types
	}
	return DefaultConventionalTypes
}

// GetTypeAliases: 타입 별칭 매핑을 반환.
// 사용자 설정 > 로케일 내장 매핑 > 빈 map 순서로 적용.
func (c *ConventionalCommitConfig) GetTypeAliases() map[string]string {
	if len(c.TypeAliases) > 0 {
		return c.TypeAliases
	}
	if c.Locale != "" {
		if m, ok := LocalizedConventionalTypes[c.Locale]; ok {
			return m
		}
	}
	return nil
}

// GetAllAllowedTypes: 허용된 모든 타입 목록을 반환 (표준 + 별칭).
func (c *ConventionalCommitConfig) GetAllAllowedTypes() []string {
	types := c.GetTypes()
	aliases := c.GetTypeAliases()
	if len(aliases) == 0 {
		return types
	}
	all := make([]string, len(types))
	copy(all, types)
	for alias := range aliases {
		all = append(all, alias)
	}
	return all
}

// ResolveType: 타입 문자열을 표준 타입으로 해석.
// 별칭이면 표준 타입 반환, 아니면 입력 그대로 반환.
func (c *ConventionalCommitConfig) ResolveType(commitType string) string {
	aliases := c.GetTypeAliases()
	if aliases != nil {
		if standard, ok := aliases[commitType]; ok {
			return standard
		}
	}
	return commitType
}

// IsEnabled: 제목 길이 검사 활성화 여부 반환 (기본값: false).
func (c *SubjectLimitConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetMaxLength: 제목 최대 글자 수 반환 (기본값: 72).
func (c *SubjectLimitConfig) GetMaxLength() int {
	if c.MaxLength <= 0 {
		return 72
	}
	return c.MaxLength
}

// IsEnabled: 본문 줄 길이 검사 활성화 여부 반환 (기본값: false).
func (c *BodyLineLimitConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetMaxLength: 본문 각 줄 최대 글자 수 반환 (기본값: 100).
func (c *BodyLineLimitConfig) GetMaxLength() int {
	if c.MaxLength <= 0 {
		return 100
	}
	return c.MaxLength
}

// IsEnabled: 커밋 메시지 검사 전체 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsNoAICoauthor: 공동 작성자 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoAICoauthor() bool {
	if c.NoAICoauthor == nil {
		return true
	}
	return *c.NoAICoauthor
}

// IsNoUnicodeSpaces: 유니코드 공백 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoUnicodeSpaces() bool {
	if c.NoUnicodeSpaces == nil {
		return true
	}
	return *c.NoUnicodeSpaces
}

// IsNoAmbiguousChars: 모호한 문자 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoAmbiguousChars() bool {
	if c.NoAmbiguousChars == nil {
		return true
	}
	return *c.NoAmbiguousChars
}

// IsNoBadRunes: 잘못된 UTF-8 룬 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoBadRunes() bool {
	if c.NoBadRunes == nil {
		return true
	}
	return *c.NoBadRunes
}

// IsNoEmoji: 커밋 메시지 이모지 검사 활성화 여부 반환 (기본값: false).
func (c *CommitMessageConfig) IsNoEmoji() bool {
	if c.NoEmoji == nil {
		return false
	}
	return *c.NoEmoji
}

// CoauthorShouldRemove: 주어진 이메일이 내장 AI 패턴 또는 사용자 정의 패턴과 일치하는지 확인.
// 일치하면 해당 Co-authored-by 줄을 제거해야 함을 의미.
func (c *CommitMessageConfig) CoauthorShouldRemove(email string) bool {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	// 내장 AI 패턴 확인
	for _, pattern := range BuiltinAICoauthorPatterns {
		matched, err := path.Match(pattern, emailLower)
		if err == nil && matched {
			return true
		}
	}
	// 사용자 정의 추가 패턴 확인
	for _, pattern := range c.CoauthorRemoveEmails {
		lowPattern := strings.ToLower(strings.TrimSpace(pattern))
		matched, err := path.Match(lowPattern, emailLower)
		if err != nil {
			logger.Warn("invalid glob pattern in coauthor_remove_emails",
				"pattern", pattern, "error", err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// ExtractCoauthorEmail: Co-authored-by: 트레일러에서 이메일 주소를 추출.
// 꺾쇠 괄호 없이 반환. 이메일이 없으면 빈 문자열 반환.
func ExtractCoauthorEmail(line string) string {
	lt := strings.LastIndex(line, "<")
	gt := strings.LastIndex(line, ">")
	if lt >= 0 && gt > lt {
		return line[lt+1 : gt]
	}
	return ""
}

// IsEnabled: 바이너리 파일 감지 활성화 여부 반환 (기본값: true).
func (c *BinaryFileConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// PolicyFor: path 의 확장자에 적용할 정책을 반환합니다.
// 우선순위: 사용자 rules > 내장 이미지 정책(allow) > default_policy(또는 block).
func (c *BinaryFileConfig) PolicyFor(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	for _, r := range c.Rules {
		for _, e := range r.Extensions {
			if strings.EqualFold(ext, e) {
				return normalizePolicy(r.Policy)
			}
		}
	}

	for _, e := range BuiltinImageExtensions {
		if ext == e {
			return "allow"
		}
	}

	return normalizePolicy(c.DefaultPolicy)
}

func normalizePolicy(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "allow":
		return "allow"
	case "lfs":
		return "lfs"
	case "block", "":
		return "block"
	default:
		return "block"
	}
}

// IsEnabled: lint 검사 활성화 여부 반환 (기본값: true).
func (c *LintConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsEnabled: 이 lint 규칙 활성화 여부 반환 (기본값: true).
func (c *LintRuleConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsEnabled: YAML lint 활성화 여부 반환 (기본값: true).
func (c *YAMLLintConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsCommentFilter: comment filter 활성화 여부 반환 (기본값: false).
func (c *YAMLLintConfig) IsCommentFilter() bool {
	if c.CommentFilter == nil {
		return false
	}
	return *c.CommentFilter
}

// IsEnabled: JSON lint 활성화 여부 반환 (기본값: true).
func (c *JSONLintConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsAllowJSON5: JSON5 형식 허용 여부 반환 (기본값: false).
func (c *JSONLintConfig) IsAllowJSON5() bool {
	if c.AllowJSON5 == nil {
		return false
	}
	return *c.AllowJSON5
}

// IsCommentFilter: comment filter 활성화 여부 반환 (기본값: false).
func (c *JSONLintConfig) IsCommentFilter() bool {
	if c.CommentFilter == nil {
		return false
	}
	return *c.CommentFilter
}

// IsEnabled: 인코딩 검사 활성화 여부 반환 (기본값: true).
func (c *EncodingConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsRequireUTF8: UTF-8 인코딩 필수 여부 반환 (기본값: true).
func (c *EncodingConfig) IsRequireUTF8() bool {
	if c.RequireUTF8 == nil {
		return true
	}
	return *c.RequireUTF8
}

// IsNoInvisibleChars: 비가시 유니코드 문자 검사 활성화 여부 반환 (기본값: false).
func (c *EncodingConfig) IsNoInvisibleChars() bool {
	if c.NoInvisibleChars == nil {
		return false
	}
	return *c.NoInvisibleChars
}

// IsNoAmbiguousChars: 모호한 유니코드 문자 검사 활성화 여부 반환 (기본값: false).
func (c *EncodingConfig) IsNoAmbiguousChars() bool {
	if c.NoAmbiguousChars == nil {
		return false
	}
	return *c.NoAmbiguousChars
}

// IsEnabled: editorconfig 검사 활성화 여부 반환 (기본값: true).
func (c *EditorConfigConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsEnabled: 보호 경로 검사 활성화 여부 반환.
func (c *ProtectedPathsConfig) IsEnabled() bool {
	return c.Enabled && len(c.Paths) > 0
}

// IsEnabled: append-only 검사 활성화 여부 반환.
func (c *AppendOnlyConfig) IsEnabled() bool {
	return c.Enabled && len(c.Paths) > 0
}

// IsFilenameOrderNumeric: 파일 이름 numeric sort 순서 검사 활성화 여부 반환 (기본값: true).
// "none"으로 명시적으로 비활성화하지 않는 한 항상 활성화.
func (c *AppendOnlyConfig) IsFilenameOrderNumeric() bool {
	return c.FilenameOrder != "none"
}

// IsEnabled: 캐시/빌드 디렉터리 검사 활성화 여부 반환 (기본값: true).
func (c *CacheDirConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsEnabled: 개선 가이드 출력 활성화 여부 반환 (기본값: true).
func (c *GuideConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}
