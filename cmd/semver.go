package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/gitversion-go/gitversion"
	"gopkg.in/yaml.v3"
)

var (
	semverShowVariable      string
	semverOutput            string
	semverNextVersion       string
	semverBranch            string
	semverNoCache           bool
	semverUpdateBuildNumber bool
)

var semverCmd = &cobra.Command{
	Use:           "semver [path]",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		if abs, err := filepath.Abs(target); err == nil {
			target = abs
		}

		opts := gitversion.Options{
			Path:    target,
			Branch:  semverBranch,
			NoCache: semverNoCache,
		}

		// commit-checker 설정에서 semver.gitversion 내장 설정 확인.
		// 내장 설정이 있으면 GitVersion.yml 자동 탐색보다 우선합니다.
		if cfg, err := config.Load(resolveConfigFilePath(configFile)); err == nil {
			if len(cfg.Semver.Gitversion) > 0 {
				if yamlBytes, err := yaml.Marshal(cfg.Semver.Gitversion); err == nil {
					opts.ConfigYAML = yamlBytes
				}
			}
		}

		// --next-version 은 모든 설정보다 우선 적용합니다.
		if semverNextVersion != "" {
			opts.Overrides = append(opts.Overrides, "next-version="+semverNextVersion)
		}

		vars, err := gitversion.Calculate(opts)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("cmd.semver.error_calc", map[string]any{"Err": err}))
		}

		if semverShowVariable != "" {
			out, err := vars.ShowVariable(semverShowVariable)
			if err != nil {
				return fmt.Errorf("%s", i18n.T("cmd.semver.error_variable", map[string]any{"Err": err}))
			}
			fmt.Println(out)
			return nil
		}

		switch strings.ToLower(semverOutput) {
		case "json":
			j, err := vars.ToJSON()
			if err != nil {
				return fmt.Errorf("%s", i18n.T("cmd.semver.error_variable", map[string]any{"Err": err}))
			}
			fmt.Println(j)
		case "dot-env", "dotenv":
			fmt.Print(vars.ToDotEnv())
		case "full-semver", "fullsemver":
			fmt.Println(vars.FullSemVer)
		case "build-server", "buildserver":
			lines, detected := gitversion.BuildServerOutput(vars, semverUpdateBuildNumber)
			if !detected {
				// CI 환경이 아니면 dot-env 로 폴백합니다.
				fmt.Print(vars.ToDotEnv())
			} else {
				for _, l := range lines {
					fmt.Println(l)
				}
			}
		default:
			fmt.Println(vars.SemVer)
		}
		return nil
	},
}

func init() {
	semverCmd.Short = i18n.T("cmd.semver.short", nil)
	semverCmd.Long = i18n.T("cmd.semver.long", nil)
	f := semverCmd.Flags()
	f.StringVarP(&semverShowVariable, "show-variable", "v", "", "단일 변수만 출력 (예: -v FullSemVer)")
	f.StringVarP(&semverOutput, "output", "o", "semver", "출력 형식: semver (기본), full-semver, json, dot-env, build-server")
	f.StringVarP(&semverNextVersion, "next-version", "n", "", "next-version 오버라이드 (예: -n 2.0.0)")
	f.StringVarP(&semverBranch, "branch", "b", "", "버전을 계산할 브랜치 오버라이드")
	f.BoolVar(&semverNoCache, "no-cache", false, "디스크 캐시 비활성화")
	f.BoolVar(&semverUpdateBuildNumber, "update-build-number", true, "build-server 출력에 빌드번호 갱신 명령 포함")
	rootCmd.AddCommand(semverCmd)
}
