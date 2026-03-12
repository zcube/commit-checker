package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/spf13/cobra"
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

binary_file:
  enabled: true

lint:
  enabled: true
  yaml:
    enabled: true
  json:
    enabled: true
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true

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
	Use:   "init",
	Short: "Generate default configuration file",
	Long: `Generate a default .commit-checker.yml configuration file in the current directory.

Use --lang to set the locale (ko, en, ja, zh). If not specified, the system locale is detected automatically.
Use --force to overwrite an existing file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := configFile
		if !initForce {
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("%s already exists. Use --force to overwrite", target)
			}
		}
		content := getDefaultConfig(initLang)
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		fmt.Printf("%s created.\n", target)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing file")
	initCmd.Flags().StringVar(&initLang, "lang", "", "locale for default config (ko, en, ja, zh); auto-detected if omitted")
	rootCmd.AddCommand(initCmd)
}
