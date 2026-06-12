// config.go: 최상위 Config 구조체 정의와 설정 로드(Load)·프리셋/전역 설정 처리 핵심 로직.
package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/zcube/commit-checker/internal/config/schema"
	"github.com/zcube/commit-checker/internal/logger"
	"gopkg.in/yaml.v3"
)

// PresetConfig: 원격 URL에서 불러올 기본 설정 프리셋.
type PresetConfig struct {
	// URL: 프리셋 설정 파일을 가져올 HTTP/HTTPS URL.
	// 해당 URL에서 .commit-checker.yml 형식의 YAML을 가져와 기본 설정으로 사용합니다.
	// 로컬 설정이 프리셋 설정을 override합니다.
	URL string `yaml:"url"`

	// Cache: URL에서 가져온 프리셋의 로컬 캐싱 설정.
	Cache AllowedWordsCacheConfig `yaml:"cache"`
}

// Config: .commit-checker.yml에서 로드하는 최상위 설정 구조체.
type Config struct {
	// Enabled: commit-checker 전체 활성화 여부 (기본값: true).
	// 전역 훅(git config --global)으로 모든 리포에서 실행되는 환경에서
	// 특정 리포만 `enabled: false` 한 줄로 모든 검사를 비활성화(opt-out)할 때 사용합니다.
	// 병합 시 프로젝트 설정 값이 전역/프리셋보다 우선합니다.
	Enabled *bool `yaml:"enabled"`

	Preset          PresetConfig          `yaml:"preset"`
	CommentLanguage CommentLanguageConfig `yaml:"comment_language"`
	CommitMessage   CommitMessageConfig   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig      `yaml:"binary_file"`
	Lint            LintConfig            `yaml:"lint"`
	Encoding        EncodingConfig        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig    `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig      `yaml:"exceptions"`
	CustomRules     CustomRulesConfig     `yaml:"custom_rules"`
	ProtectedPaths  ProtectedPathsConfig  `yaml:"protected_paths"`
	AppendOnly      AppendOnlyConfig      `yaml:"append_only"`
	CacheDir        CacheDirConfig        `yaml:"cache_dir"`
	Guide           GuideConfig           `yaml:"guide"`
}

// Load: 주어진 YAML 파일에서 설정을 읽음.
// 전역 설정(~/.commit-checker.yml)이 있으면 먼저 로드하고 프로젝트 설정과 병합합니다.
// preset.url이 설정된 경우 해당 URL에서 프리셋을 로드하여 기본값으로 사용합니다.
// 우선순위: 프로젝트 설정 > 프리셋 > 전역 설정
// 파일이 없으면 기본 설정을 반환.
func Load(cfgPath string) (*Config, error) {
	globalCfg := loadGlobalConfig()

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			if globalCfg != nil {
				applyDefaults(globalCfg)
				return globalCfg, nil
			}
			cfg := &Config{}
			applyDefaults(cfg)
			return cfg, nil
		}
		return nil, err
	}

	// 구 버전 스키마 감지: 현재 스키마로 파싱 실패 시 자동 마이그레이션 시도.
	ver := schema.DetectVersion(data)
	if ver != schema.VersionCurrent && ver != schema.VersionUnknown {
		result, migErr := schema.Migrate(data)
		if migErr == nil {
			data = result.Data
		} else {
			logger.Warn("config auto-migration failed, proceeding with original",
				"path", cfgPath, "error", migErr)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, formatConfigError(cfgPath, err)
	}

	// 프리셋 로드: preset.url이 설정된 경우 URL에서 기본 설정을 가져옴.
	var presetCfg *Config
	if cfg.Preset.URL != "" {
		presetCfg, err = loadPresetConfig(&cfg.Preset)
		if err != nil {
			return nil, fmt.Errorf("preset url 로드 실패: %w", err)
		}
	}

	// 우선순위: 프로젝트 > 프리셋 > 전역
	if presetCfg != nil {
		if globalCfg != nil {
			merged := mergeConfigs(globalCfg, presetCfg)
			cfg = mergeConfigs(&merged, &cfg)
		} else {
			cfg = mergeConfigs(presetCfg, &cfg)
		}
	} else if globalCfg != nil {
		cfg = mergeConfigs(globalCfg, &cfg)
	}

	applyDefaults(&cfg)
	if err := resolveAllowedWords(&cfg); err != nil {
		return nil, err
	}
	for _, w := range Validate(&cfg, cfgPath) {
		logger.Warn(w)
	}
	return &cfg, nil
}

// loadPresetConfig: preset.url에서 설정을 가져와 파싱합니다.
// 버전 감지 및 마이그레이션을 적용한 후 Config로 반환합니다.
// 프리셋 안의 preset.url은 무시합니다 (중첩 프리셋 미지원).
func loadPresetConfig(preset *PresetConfig) (*Config, error) {
	var body []byte

	if cached, ok := loadCachedBytes(&preset.Cache, preset.URL); ok {
		body = cached
	} else {
		var err error
		body, err = fetchURL(preset.URL)
		if err != nil {
			return nil, err
		}
		saveCachedBytes(&preset.Cache, preset.URL, body)
	}

	// 버전 감지 및 마이그레이션
	ver := schema.DetectVersion(body)
	if ver != schema.VersionCurrent && ver != schema.VersionUnknown {
		result, migErr := schema.Migrate(body)
		if migErr == nil {
			body = result.Data
		} else {
			logger.Warn("preset config auto-migration failed, proceeding with original",
				"url", preset.URL, "error", migErr)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("preset YAML 파싱 실패: %w", err)
	}
	// 프리셋 안에 preset.url이 있으면 에러: 중첩/무한루프 방지.
	// preset은 한 단계만 허용합니다.
	if cfg.Preset.URL != "" {
		return nil, fmt.Errorf("preset은 중첩될 수 없습니다 (preset 안에 preset.url 사용 불가): %s", cfg.Preset.URL)
	}
	return &cfg, nil
}

// maxPresetSize: URL에서 가져오는 프리셋 파일의 최대 크기 (10MB).
const maxPresetSize = 10 * 1024 * 1024

// fetchURL: HTTP/HTTPS URL에서 raw 바이트를 가져옵니다.
func fetchURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxPresetSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxPresetSize {
		return nil, fmt.Errorf("preset url response exceeds 10MB limit")
	}
	return body, nil
}

// globalConfigEnvVar: 전역 설정 파일 경로를 강제 지정하는 환경 변수 이름.
const globalConfigEnvVar = "COMMIT_CHECKER_GLOBAL_CONFIG"

// GlobalConfigPath: 전역 설정 파일 경로를 결정합니다.
// 다음 우선순위로 첫 번째로 존재하는 파일의 경로와 true 를 반환합니다:
//  1. $COMMIT_CHECKER_GLOBAL_CONFIG 환경 변수
//     (설정 시 이 경로만 사용. 파일이 없으면 Warn 로그 후 전역 설정 없음으로 처리)
//  2. $XDG_CONFIG_HOME/commit-checker/config.yml (XDG_CONFIG_HOME 설정 시, 모든 플랫폼)
//  3. os.UserConfigDir()/commit-checker/config.yml
//     (Linux: ~/.config, macOS: ~/Library/Application Support, Windows: %AppData%)
//  4. ~/.commit-checker.yml (legacy — 하위 호환 유지)
//
// 존재하는 파일이 없으면 ("", false) 를 반환합니다.
func GlobalConfigPath() (path string, exists bool) {
	// 1. 환경 변수: 설정되어 있으면 다른 경로는 보지 않음.
	if envPath := os.Getenv(globalConfigEnvVar); envPath != "" {
		if globalFileExists(envPath) {
			return envPath, true
		}
		logger.Warn("COMMIT_CHECKER_GLOBAL_CONFIG points to a missing file, ignoring global config",
			"path", envPath)
		return "", false
	}

	var candidates []string
	// 2. XDG_CONFIG_HOME: 설정 시 모든 플랫폼에서 우선 확인.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "commit-checker", "config.yml"))
	}
	// 3. 플랫폼 표준 설정 디렉터리.
	if dir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(dir, "commit-checker", "config.yml"))
	}
	// 4. legacy 홈 디렉터리 경로 (하위 호환).
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".commit-checker.yml"))
	}
	for _, p := range candidates {
		if globalFileExists(p) {
			return p, true
		}
	}
	return "", false
}

// globalFileExists: 경로가 존재하는 일반 파일인지 확인.
// 경로는 사용자 본인이 환경 변수·표준 설정 디렉터리로 지정하는 로컬 설정 파일이므로
// path traversal 위협 모델에 해당하지 않음 (G304 전역 제외와 동일한 사유).
func globalFileExists(path string) bool {
	info, err := os.Stat(filepath.Clean(path)) // #nosec G703 -- 사용자 지정 로컬 설정 경로
	return err == nil && !info.IsDir()
}

// loadGlobalConfig: 전역 설정 파일을 로드합니다.
// 경로는 GlobalConfigPath 의 우선순위를 따르며, 파일이 없거나 오류가 발생하면 nil을 반환합니다.
// 프로젝트 설정(Load)과 동일하게 구버전 스키마 자동 마이그레이션을 적용합니다.
func loadGlobalConfig() *Config {
	globalPath, ok := GlobalConfigPath()
	if !ok {
		return nil // 전역 설정 없음 — 정상
	}
	data, err := os.ReadFile(globalPath)
	if err != nil {
		logger.Warn("global config read error, ignoring", "path", globalPath, "error", err)
		return nil
	}

	// 구 버전 스키마 감지: 현재 스키마로 파싱 실패 시 자동 마이그레이션 시도.
	ver := schema.DetectVersion(data)
	if ver != schema.VersionCurrent && ver != schema.VersionUnknown {
		result, migErr := schema.Migrate(data)
		if migErr == nil {
			data = result.Data
		} else {
			logger.Warn("global config auto-migration failed, proceeding with original",
				"path", globalPath, "error", migErr)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Warn("global config parse error, ignoring", "path", globalPath, "error", err)
		return nil
	}
	return &cfg
}
