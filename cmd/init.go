package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
)

var (
	initForce bool
	initLang  string
)

// localeToLanguageName: 로케일 코드를 required_language 값으로 변환.
var localeToLanguageName = map[string]string{
	"ko": "korean",
	"en": "english",
	"ja": "japanese",
	"zh": "chinese",
}

func getDefaultConfig(lang string) string {
	if lang == "" {
		lang = i18n.DetectLocale()
	}
	lang = strings.ToLower(lang)

	requiredLang := "korean"
	locale := "ko"
	if name, ok := localeToLanguageName[lang]; ok {
		requiredLang = name
		locale = lang
	}

	return fmt.Sprintf(`# yaml-language-server: $schema=./.commit-checker.schema.json
# commit-checker configuration
# All fields are optional. Defaults apply when omitted.

comment_language:
  enabled: true
  required_language: %s   # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff            # diff | full
  extensions:
    - .go
    - .ts
    - .tsx
    - .js
    - .jsx
    - .mjs
    - .java
    - .kt
    - .py
    - .c
    - .cpp
    - .cs
    - .swift
    - .rs
    - dockerfile
    # - .md           # 마크다운 주석 언어 검사 (필요 시 활성화)

  # allowed_words: 언어 검사에서 무시할 영어 단어 목록.
  # 고유명사, 기술 용어 등 원어를 유지해야 하는 단어를 등록하세요.
  # allowed_words:
  #   - TypeScript
  #   - JavaScript
  #   - API
  #   - URL

  # allowed_words_file: 허용 단어가 한 줄에 하나씩 적힌 텍스트 파일 경로.
  # allowed_words_file: .commit-checker-words.txt

  # allowed_words_url: 허용 단어 파일을 HTTP/HTTPS로 가져올 URL.
  # allowed_words_url: https://example.com/allowed-words.txt

  # allowed_words_cache: URL에서 가져온 허용 단어의 로컬 캐싱.
  # allowed_words_cache:
  #   enabled: true
  #   ttl: 24h    # 캐시 유효 기간 (1h, 30m, 7d 등)

  # check_strings: 문자열 리터럴도 언어 검사 (기본값: false).
  # check_strings: false

  # no_emoji: 주석에서 이모지 사용 금지 (기본값: false).
  # no_emoji: false

  # ignore_files: 주석 언어 검사에서 제외할 glob 패턴.
  # ignore_files:
  #   - "**/*_test.go"
  #   - "**/*.generated.go"
  #   - "vendor/**"

  # file_languages: 파일별 언어 규칙 (첫 번째 일치 패턴 적용).
  # file_languages:
  #   - pattern: "locales/**"
  #     language: any
  #   - pattern: "i18n/**"
  #     language: english

binary_file:
  enabled: true
  # ignore_files: 바이너리 검사에서 제외할 파일 (이미지, 폰트 등).
  # ignore_files:
  #   - "**/*.png"
  #   - "**/*.jpg"
  #   - "**/*.woff2"

lint:
  enabled: true
  yaml:
    enabled: true
  json:
    enabled: true
    # allow_json5: false     # JSON5 형식 허용 (주석, trailing comma)
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # 비가시 유니코드 문자 검사 (NBSP, ZWSP 등)
  # no_ambiguous_chars: true   # ASCII와 혼동되는 유니코드 문자 검사

editorconfig:
  enabled: true
  # ignore_files:
  #   - "vendor/**"

commit_message:
  # enabled: true              # false이면 모든 커밋 메시지 검사 비활성화
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  # no_emoji: false
  locale: %s

  conventional_commit:
    enabled: false
    locale: %s    # localized type aliases (e.g., feat -> %s)

  language_check:
    enabled: false
    required_language: %s
    min_length: 5
    skip_prefixes:
      - "Merge"
      - "Revert"
      - "fixup!"
      - "squash!"

# exceptions: 전역 및 기능별 파일 제외.
# exceptions:
#   global_ignore:
#     - "vendor/**"
#     - "third_party/**"
#   comment_language_ignore:
#     - "legacy/**"
`, requiredLang, locale, locale, localizedExample(locale), requiredLang)
}

// localizedExample: 해당 로케일의 대표 타입 별칭 예시를 반환.
func localizedExample(locale string) string {
	switch locale {
	case "ko":
		return "기능"
	case "ja":
		return "機能"
	case "zh":
		return "功能"
	default:
		return "feat"
	}
}

var initCmd = &cobra.Command{
	Use: "init",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := configFile
		if !initForce {
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("%s", i18n.T("init.already_exists", map[string]any{"Path": target}))
			}
		}
		content := getDefaultConfig(initLang)
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return fmt.Errorf("%s", i18n.T("init.fail_write", map[string]any{"Error": err.Error()}))
		}
		fmt.Println(i18n.T("init.created", map[string]any{"Path": target}))
		return nil
	},
}

func init() {
	initCmd.Short = i18n.T("cmd.init.short", nil)
	initCmd.Long = i18n.T("cmd.init.long", nil)
	initCmd.Flags().BoolVar(&initForce, "force", false, i18n.T("flag.init_force", nil))
	initCmd.Flags().StringVar(&initLang, "lang", "", i18n.T("flag.init_lang", nil))
	rootCmd.AddCommand(initCmd)
}
