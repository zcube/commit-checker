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

	locale := "ko"
	if _, ok := localeToLanguageName[lang]; ok {
		locale = lang
	}

	return fmt.Sprintf(`# yaml-language-server: $schema=./.commit-checker.schema.json
# commit-checker configuration
# All fields are optional. Defaults apply when omitted.

comment_language:
  enabled: true
  locale: %s                  # BCP-47 (ko/en/ja/zh) 또는 legacy (korean/english/japanese/chinese/any)
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

  # file_languages: 파일별 로케일 규칙 (첫 번째 일치 패턴 적용).
  # file_languages:
  #   - pattern: "locales/**"
  #     locale: any
  #   - pattern: "i18n/**"
  #     locale: en

binary_file:
  enabled: true
  # default_policy: block         # 기본 정책 — block | allow | lfs (기본값: block)
  # 내장 이미지 확장자(.png .jpg .jpeg .gif .webp .bmp .ico .tiff .tif .heic .heif .avif)는
  # 별도 규칙이 없으면 자동으로 allow 가 적용됩니다.

  # rules: 확장자별 정책 (첫 번째 매칭 규칙 적용).
  # rules:
  #   # 이미지를 LFS 로 강제하려면:
  #   - extensions: [.png, .jpg, .jpeg, .gif, .webp]
  #     policy: lfs
  #   # 디자인 원본/동영상 등 큰 바이너리:
  #   - extensions: [.psd, .ai, .sketch]
  #     policy: lfs
  #   - extensions: [.mp4, .mov, .webm]
  #     policy: lfs
  #   # 폰트는 그냥 허가:
  #   - extensions: [.woff2, .ttf, .otf]
  #     policy: allow

  # ignore_files: 정책 검사 자체를 건너뛸 glob 패턴.
  # ignore_files:
  #   - "assets/icons/**"

lint:
  enabled: true
  yaml:
    enabled: true
  json:
    enabled: true
    # allow_json5: false     # JSON5 형식 허용 (주석, trailing comma)
  xml:
    enabled: true
  toml:
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

# append_only: DB 마이그레이션 등 한 번 커밋된 내용을 변경하면 안 되는 경로.
# 기본 비활성화 — paths 를 지정하면서 enabled: true 로 켜세요.
# append_only:
#   enabled: true
#   paths:
#     - "migrations/**"
#     - "db/migrations/**"
#   # filename_order: numeric 이 기본. 순서 검사를 끄려면 none 지정.
#   # filename_order: none

# cache_dir: 빌드 산출물·캐시 디렉터리(node_modules, dist, build, target,
# __pycache__, .venv 등) 안의 파일이 커밋되는 것을 차단합니다.
# 부모 디렉터리 인디케이터(go.mod, package.json, Cargo.toml 등) 기반으로 검증.
cache_dir:
  enabled: true
  # ignore_dirs: 의도적으로 커밋하는 디렉터리 (예: Go vendor).
  # ignore_dirs:
  #   - vendor

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
    locale: %s    # 비어두면 commit_message.locale 에서 자동 유도
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

# custom_rules: 정규식 기반 커스텀 검사 규칙.
# custom_rules:
#   commit_message:
#     - name: no-wip
#       pattern: "(?i)^WIP"
#       message: "WIP 접두사를 제거하고 커밋하세요"
#     - name: need-ticket
#       pattern: "\\[PROJ-\\d+\\]"
#       message: "커밋 제목에 티켓 ID를 포함하세요 (예: [PROJ-123])"
#       required: true
#   diff:
#     - name: no-api-key
#       pattern: "(?i)(api_key|secret_key)\\s*=\\s*['\"][^'\"]{10,}"
#       message: "API 키가 감지되었습니다 — 커밋에 포함하지 마세요"
`, locale, locale, locale, localizedExample(locale), locale)
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
