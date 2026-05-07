// Package cachedir 는 빌드 산출물·캐시 디렉터리를 식별하는 검증 로직을 제공합니다.
// clean-caches 프로젝트의 검증기(scan.go)에서 파일 단위로 복사하여 commit-checker
// 용도로 트림한 패키지입니다 (디스크 정리·git 헬퍼 등 정리 전용 코드는 제외).
//
// 주 용도: 스테이지된 또는 추적된 파일이 캐시/빌드 산출물 디렉터리 안에 있는지를
// 부모 디렉터리 인디케이터(go.mod, package.json, Cargo.toml 등) 기반으로 검증합니다.
package cachedir

import (
	"os"
	"path/filepath"
	"strings"
)

// ProtectedDirNames 는 절대 순회·수정해서는 안 되는 디렉터리 이름 집합입니다.
// 버전 관리 메타데이터를 보호하기 위함입니다.
var ProtectedDirNames = map[string]bool{
	".git": true,
	".svn": true,
	".hg":  true,
	".bzr": true,
}

// validator 는 후보 디렉터리 경로가 실제 빌드 산출물 디렉터리인지 확인하는 함수입니다.
type validator func(dirPath string) bool

// statAnyIn 은 base 안에 names 중 하나라도 wantDir 종류로 존재하는지 확인합니다.
func statAnyIn(base string, names []string, wantDir bool) bool {
	for _, name := range names {
		info, err := os.Stat(filepath.Join(base, name))
		if err == nil && info.IsDir() == wantDir {
			return true
		}
	}
	return false
}

// parentHasAny 는 부모 디렉터리에 names 중 하나가 일반 파일로 존재하면 통과하는 validator 를 반환합니다.
func parentHasAny(names ...string) validator {
	return func(dirPath string) bool {
		return statAnyIn(filepath.Dir(dirPath), names, false)
	}
}

// dirContainsAny 는 후보 디렉터리 자체가 names 중 하나를 일반 파일로 포함하면 통과하는 validator 를 반환합니다.
// CMake/Ninja out-of-source build 디렉터리 및 Python virtualenv (pyvenv.cfg) 식별에 사용됩니다.
func dirContainsAny(names ...string) validator {
	return func(dirPath string) bool {
		return statAnyIn(dirPath, names, false)
	}
}

// dirContainsDirAny 는 후보 디렉터리 자체가 names 중 하나를 하위 디렉터리로 포함하면 통과합니다.
// ESP-IDF .embuild ("espressif/" 포함) 같은 케이스를 식별합니다.
func dirContainsDirAny(names ...string) validator {
	return func(dirPath string) bool {
		return statAnyIn(dirPath, names, true)
	}
}

// parentHasPyFiles 는 __pycache__ 의 부모 디렉터리에 .py 소스 파일이 하나라도 있으면 통과합니다.
func parentHasPyFiles(dirPath string) bool {
	entries, err := os.ReadDir(filepath.Dir(dirPath))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
			return true
		}
	}
	return false
}

// 공유 파일 목록 상수 - 여러 dirValidators 항목이 참조합니다.
var (
	jsLockfiles = []string{
		"package.json", "package-lock.json",
		"yarn.lock", "pnpm-lock.yaml",
		"bun.lockb", "bun.lock",
	}
	gradleFiles = []string{
		"build.gradle", "build.gradle.kts",
		"settings.gradle", "settings.gradle.kts",
		"gradlew", "gradlew.bat",
	}
	nuxtConfigs = []string{"nuxt.config.js", "nuxt.config.ts", "nuxt.config.mjs"}
	nextConfigs = []string{"next.config.js", "next.config.ts", "next.config.mjs", "next.config.cjs"}
)

// dirValidators 는 후보 디렉터리 이름과 검증기 목록의 매핑입니다.
// 적어도 하나의 validator 가 true 를 반환해야 빌드 산출물로 인정됩니다.
//
// 설계 원칙: 부모 디렉터리의 잘 알려진 빌드 도구 인디케이터 파일을 우선 확인.
// 부모 인디케이터가 없는 경우(CMake out-of-source 등)에만 디렉터리 자체 내용을 검사합니다.
var dirValidators = map[string][]validator{

	// ── JavaScript / TypeScript 패키지 매니저 ──
	"node_modules": {
		parentHasAny(jsLockfiles...),
	},

	// ── 빌드 산출물: JavaScript / TypeScript ──
	"dist": {
		parentHasAny(jsLockfiles...),
		parentHasAny("go.mod"),
		parentHasAny("Cargo.toml"),
	},
	"out": {
		parentHasAny(nextConfigs...),
		parentHasAny("package.json"),
	},

	// ── 빌드 산출물: 다언어 ──
	"build": {
		parentHasAny("package.json"),
		parentHasAny("Cargo.toml"),
		parentHasAny(gradleFiles...),
		parentHasAny("pubspec.yaml"),
		parentHasAny("wails.json"),
		parentHasAny("CMakeLists.txt"),
		dirContainsAny("CMakeCache.txt", "build.ninja"),
	},
	"target": {
		parentHasAny("Cargo.toml"),
		parentHasAny("pom.xml"),
		parentHasAny("build.sbt"),
	},

	// ── 벤더링 ──
	"vendor": {
		parentHasAny("go.mod"),
		parentHasAny("Cargo.toml"),
		parentHasAny("composer.json"),
		parentHasAny("Gemfile"),
		parentHasAny("package.json"),
	},

	// ── Gradle 빌드 캐시 ──
	".gradle": {
		parentHasAny(gradleFiles...),
	},

	// ── Next.js 빌드 캐시 ──
	".next": {
		parentHasAny(nextConfigs...),
		parentHasAny("package.json"),
	},

	// ── Nuxt 빌드 캐시 ──
	".nuxt": {
		parentHasAny(nuxtConfigs...),
	},
	".output": {
		parentHasAny(nuxtConfigs...),
	},

	// ── SvelteKit 빌드 캐시 ──
	".svelte-kit": {
		parentHasAny("svelte.config.js", "svelte.config.ts", "svelte.config.cjs"),
	},

	// ── yarn berry 캐시 ──
	".yarn": {
		parentHasAny("yarn.lock"),
		parentHasAny("package.json"),
	},

	// ── bun 캐시 ──
	".bun": {
		parentHasAny("bun.lockb", "bun.lock"),
		parentHasAny("package.json"),
	},

	// ── Python 캐시 ──
	"__pycache__": {
		parentHasPyFiles,
		parentHasAny("pyproject.toml", "setup.py", "setup.cfg", "requirements.txt"),
	},
	".pytest_cache": {
		parentHasAny(
			"pytest.ini", "pyproject.toml", "setup.cfg",
			"tox.ini", "requirements.txt", "conftest.py",
		),
	},
	".mypy_cache": {
		parentHasAny("mypy.ini", ".mypy.ini", "pyproject.toml", "setup.cfg"),
	},
	".ruff_cache": {
		parentHasAny("ruff.toml", "pyproject.toml"),
	},

	// ── Turbo / Parcel 캐시 ──
	".turbo": {
		parentHasAny("turbo.json"),
		parentHasAny("package.json"),
	},
	".parcel-cache": {
		parentHasAny("package.json"),
	},

	// ── Python 가상환경 ──
	".venv": {
		dirContainsAny("pyvenv.cfg"),
	},
	".tox": {
		parentHasAny("tox.ini", "setup.cfg", "pyproject.toml"),
	},
	".nox": {
		parentHasAny("noxfile.py"),
	},

	// ── ESP-IDF 빌드 캐시 ──
	".embuild": {
		parentHasAny("sdkconfig.defaults", "sdkconfig", "idf_component.yml"),
		dirContainsDirAny("espressif"),
	},

	// ── Flutter / Dart 빌드 캐시 ──
	".dart_tool": {
		parentHasAny("pubspec.yaml"),
		dirContainsAny("package_config.json"),
	},
}

// IsKnownCacheDirName 은 name 이 dirValidators 에 등록된 캐시/빌드 디렉터리 이름이면 true 를 반환합니다.
// 검증기는 실행하지 않으며 단순 이름 일치만 확인합니다.
func IsKnownCacheDirName(name string) bool {
	_, ok := dirValidators[name]
	return ok
}

// IsCacheDir 은 dirPath 가 실제 캐시/빌드 디렉터리인지 확인합니다.
// 디렉터리 이름이 등록되어 있고 적어도 하나의 validator 가 통과해야 합니다.
// 이름은 등록되어 있지만 검증기가 실패하면(예: package.json 없는 임의의 build/) false 를 반환합니다.
func IsCacheDir(dirPath string) bool {
	name := filepath.Base(dirPath)
	validators, ok := dirValidators[name]
	if !ok {
		return false
	}
	for _, v := range validators {
		if v(dirPath) {
			return true
		}
	}
	return false
}

// IsPythonVirtualenv 는 dirPath 가 Python virtualenv (pyvenv.cfg 포함) 인지 확인합니다.
// .venv / venv / env / test_env 등 이름과 무관하게 동작합니다.
func IsPythonVirtualenv(dirPath string) bool {
	_, err := os.Stat(filepath.Join(dirPath, "pyvenv.cfg"))
	return err == nil
}

// FindCacheDirAncestor 는 filePath 의 조상 디렉터리 중 캐시/빌드 디렉터리가 있는지 검사합니다.
// 발견 시 해당 디렉터리 절대 경로와 true 를, 없으면 "" 와 false 를 반환합니다.
//
// repoRoot 보다 위로는 올라가지 않습니다. filePath 와 repoRoot 는 절대 경로 또는 동일 기준의 상대 경로여야 합니다.
func FindCacheDirAncestor(repoRoot, filePath string) (string, bool) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", false
	}
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return "", false
	}

	dir := filepath.Dir(absFile)
	for dir != absRoot && strings.HasPrefix(dir, absRoot+string(os.PathSeparator)) {

		if IsCacheDir(dir) || IsPythonVirtualenv(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// KnownCacheDirNames 는 등록된 모든 캐시/빌드 디렉터리 이름을 반환합니다 (이름 검사 전용).
func KnownCacheDirNames() []string {
	names := make([]string, 0, len(dirValidators))
	for n := range dirValidators {
		names = append(names, n)
	}
	return names
}
