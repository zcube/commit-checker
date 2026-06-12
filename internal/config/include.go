// include.go: git 의 [includeIf "gitdir:..."] 와 유사한 조건부 설정 포함(include) 처리.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zcube/commit-checker/internal/config/schema"
	"github.com/zcube/commit-checker/internal/logger"
	"github.com/zcube/commit-checker/internal/pathutil"
	"gopkg.in/yaml.v3"
)

// resolveIncludes: cfg.Include 규칙을 해석하여 include 파일들을 베이스로 깐 병합 결과를 반환합니다.
// 전역 설정과 프로젝트 설정 양쪽에서 로드 직후 동일하게 호출됩니다.
//
// 병합 의미론:
//   - include 는 베이스 제공 역할: 본문(cfg) 값 > include 값.
//   - 여러 include 간에는 나중 항목 > 앞 항목.
//   - 처리 순서: include 들을 순서대로 병합해 베이스를 만들고, 그 위에 본문을 병합.
//   - 전체 우선순위는 기존 유지: 프로젝트(include 처리됨) > preset > 전역(include 처리됨).
//
// 안전장치:
//   - gitdir 가 비어있으면 항상 포함, 지정 시 현재 작업 디렉터리(리포 루트)가 패턴과 매칭될 때만 포함.
//   - 누락 파일은 Warn 후 건너뜀.
//   - 포함된 파일 안의 include 는 무시 (중첩 금지) + Warn.
func resolveIncludes(cfg *Config, cfgPath string) *Config {
	if len(cfg.Include) == 0 {
		return cfg
	}
	// 훅은 리포 루트에서 실행되므로 현재 작업 디렉터리를 gitdir 비교 대상으로 사용.
	workDir, err := os.Getwd()
	if err != nil {
		logger.Warn("include: 작업 디렉터리 확인 실패, gitdir 조건 include 를 건너뜀",
			"config", cfgPath, "error", err)
		workDir = ""
	}

	var base *Config
	for _, rule := range cfg.Include {
		if rule.Path == "" {
			logger.Warn("include: path 가 비어있는 항목을 건너뜀", "config", cfgPath)
			continue
		}
		if rule.Gitdir != "" && (workDir == "" || !gitdirMatch(rule.Gitdir, workDir)) {
			continue // gitdir 조건 비매칭 — 포함하지 않음
		}
		inc := loadIncludeFile(rule.Path, cfgPath)
		if inc == nil {
			continue
		}
		if base == nil {
			base = inc
		} else {
			// 나중 include 가 앞 include 보다 우선
			merged := mergeConfigs(base, inc)
			base = &merged
		}
	}
	if base == nil {
		return cfg
	}
	// 본문이 include 베이스보다 우선
	merged := mergeConfigs(base, cfg)
	return &merged
}

// loadIncludeFile: include 대상 설정 파일을 읽어 Config 로 파싱합니다.
// 프로젝트/전역 설정과 동일하게 구버전 스키마 자동 마이그레이션을 적용하며,
// 읽기·파싱 실패 시 Warn 후 nil 을 반환합니다 (Load 전체는 실패하지 않음).
func loadIncludeFile(path, cfgPath string) *Config {
	resolved := resolveIncludePath(path, cfgPath)
	// 경로는 사용자 본인이 로컬 설정 파일에 직접 적는 값이므로
	// path traversal 위협 모델에 해당하지 않음 (G304 전역 제외와 동일한 사유).
	data, err := os.ReadFile(resolved) // #nosec G703 G304 -- 사용자 지정 로컬 설정 경로
	if err != nil {
		logger.Warn("include: 파일을 읽을 수 없어 건너뜀", "path", resolved, "error", err)
		return nil
	}

	// 구 버전 스키마 감지: 현재 스키마로 파싱 실패 시 자동 마이그레이션 시도.
	ver := schema.DetectVersion(data)
	if ver != schema.VersionCurrent && ver != schema.VersionUnknown {
		result, migErr := schema.Migrate(data)
		if migErr == nil {
			data = result.Data
		} else {
			logger.Warn("include config auto-migration failed, proceeding with original",
				"path", resolved, "error", migErr)
		}
	}

	var inc Config
	if err := yaml.Unmarshal(data, &inc); err != nil {
		logger.Warn("include: YAML 파싱 실패로 건너뜀", "path", resolved, "error", err)
		return nil
	}
	// 중첩 include 금지: 포함된 파일 안의 include 는 무시.
	if len(inc.Include) > 0 {
		logger.Warn("include: 중첩 include 는 지원하지 않아 무시함", "path", resolved)
		inc.Include = nil
	}
	return &inc
}

// resolveIncludePath: include path 의 '~' 를 홈 디렉터리로 확장하고,
// 상대 경로는 include 를 선언한 설정 파일 기준으로 해석합니다 (git include 와 동일).
func resolveIncludePath(path, cfgPath string) string {
	p := expandTilde(path)
	if !filepath.IsAbs(p) {
		p = filepath.Join(filepath.Dir(cfgPath), p)
	}
	return filepath.Clean(p)
}

// gitdirMatch: git 의 includeIf "gitdir:" 의미론으로 workDir 가 패턴과 매칭되는지 확인합니다.
//   - '~' 는 홈 디렉터리로 확장.
//   - 패턴이 '/' 로 끝나면 '**' 를 덧붙여 해당 디렉터리와 하위 전체를 매칭 (git 과 동일).
//   - macOS 의 /tmp → /private/tmp 류 불일치를 막기 위해 양쪽에 심볼릭 링크 해석을 적용.
func gitdirMatch(pattern, workDir string) bool {
	dirSuffix := strings.HasSuffix(pattern, "/")
	p := filepath.ToSlash(expandTilde(pattern))
	if dirSuffix {
		p = strings.TrimSuffix(p, "/") + "/**"
	}
	p = resolveSymlinkPrefix(p)

	wd := workDir
	if r, err := filepath.EvalSymlinks(wd); err == nil {
		wd = r
	}
	return pathutil.MatchPath(filepath.ToSlash(wd), p)
}

// resolveSymlinkPrefix: 패턴에서 glob 메타문자(*, ?, []) 이전까지의 리터럴 디렉터리
// prefix 에 filepath.EvalSymlinks 를 적용합니다. prefix 가 존재하지 않거나 해석에
// 실패하면 패턴을 그대로 반환합니다 (best-effort).
func resolveSymlinkPrefix(pattern string) string {
	segs := strings.Split(pattern, "/")
	lit := 0
	for ; lit < len(segs); lit++ {
		if strings.ContainsAny(segs[lit], "*?[") {
			break
		}
	}
	if lit == 0 {
		return pattern
	}
	prefix := strings.Join(segs[:lit], "/")
	if prefix == "" {
		// 패턴이 "/**" 처럼 루트 직후 glob 으로 시작하는 경우
		return pattern
	}
	resolved, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		return pattern
	}
	rest := segs[lit:]
	if len(rest) == 0 {
		return filepath.ToSlash(resolved)
	}
	return filepath.ToSlash(resolved) + "/" + strings.Join(rest, "/")
}

// expandTilde: 경로 선두의 '~' 또는 '~/' 를 사용자 홈 디렉터리로 확장합니다.
// 홈 디렉터리를 알 수 없으면 원본을 그대로 반환합니다.
func expandTilde(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
