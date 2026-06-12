package cmd

import (
	_ "embed"
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

// defaultConfigTemplate: 기본 설정 YAML 템플릿 (locale 자리에 fmt 의 %s 사용).
//
//go:embed templates/config.yml.tmpl
var defaultConfigTemplate string

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

	return fmt.Sprintf(defaultConfigTemplate, locale, locale, locale, localizedExample(locale), locale)
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
