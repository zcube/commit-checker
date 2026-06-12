package config_test

// 조건부 설정 포함(include) 동작 검증.
//
// 병합 의미론:
//   - include 는 베이스 제공: 본문 값 > 나중 include > 앞 include
//   - 프로젝트 설정이 존재하면 프로젝트(include 처리됨) > preset 만 적용되고 전역은 완전 무시
//   - 프로젝트 설정이 없으면 전역(include 처리됨) > 전역의 preset 이 적용
//
// gitdir 매칭 (git includeIf 시맨틱):
//   - '~' 는 홈 디렉터리로 확장
//   - 패턴이 '/' 로 끝나면 '**' 가 덧붙어 해당 디렉터리와 하위 전체를 매칭
//   - 비교 대상은 현재 작업 디렉터리 (훅은 리포 루트에서 실행됨)

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

// writeConfigAt: 지정한 디렉터리에 .commit-checker.yml 을 기록하고 경로를 반환.
func writeConfigAt(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".commit-checker.yml")
	writeFileAt(t, path, content)
	return path
}

func TestInclude_무조건include_베이스병합(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	// gitdir 없는 include 는 항상 포함 (범용 공유 설정)
	writeFileAt(t, filepath.Join(dir, "base.yml"), "comment_language:\n  min_length: 7\n")
	path := writeConfigAt(t, dir, fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(dir, "base.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("무조건 include 의 min_length 가 적용되어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
}

func TestInclude_본문이include보다우선(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	writeFileAt(t, filepath.Join(dir, "base.yml"), "comment_language:\n  min_length: 7\n  locale: en\n")
	path := writeConfigAt(t, dir, fmt.Sprintf(`
include:
  - path: %s
comment_language:
  min_length: 3
`, filepath.Join(dir, "base.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 3 {
		t.Errorf("본문 값이 include 보다 우선해야 함: got %d, want 3", cfg.CommentLanguage.MinLength)
	}
	// 본문에 없는 값은 include 베이스에서 채워짐
	if cfg.CommentLanguage.GetLocale() != "english" {
		t.Errorf("본문에 없는 locale 은 include 값을 사용해야 함: got %q", cfg.CommentLanguage.GetLocale())
	}
}

func TestInclude_여러include_나중항목우선(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	writeFileAt(t, filepath.Join(dir, "first.yml"), "comment_language:\n  min_length: 2\n  allowed_words: [FirstWord]\n")
	writeFileAt(t, filepath.Join(dir, "second.yml"), "comment_language:\n  min_length: 4\n  allowed_words: [SecondWord]\n")
	path := writeConfigAt(t, dir, fmt.Sprintf(`
include:
  - path: %s
  - path: %s
`, filepath.Join(dir, "first.yml"), filepath.Join(dir, "second.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 4 {
		t.Errorf("나중 include 가 앞 include 보다 우선해야 함: got %d, want 4", cfg.CommentLanguage.MinLength)
	}
	// 목록 필드는 합쳐짐 (앞 include 먼저)
	words := cfg.CommentLanguage.AllowedWords
	if len(words) < 2 || words[0] != "FirstWord" || words[1] != "SecondWord" {
		t.Errorf("include 간 목록 병합 실패: %v", words)
	}
}

func TestInclude_Gitdir매칭시에만포함(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()
	workRepo := filepath.Join(dir, "work", "repo")
	otherRepo := filepath.Join(dir, "other", "repo")
	for _, d := range []string{workRepo, otherRepo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFileAt(t, filepath.Join(dir, "work.yml"), "comment_language:\n  min_length: 7\n")
	content := fmt.Sprintf(`
include:
  - path: %s
    gitdir: %s/
`, filepath.Join(dir, "work.yml"), filepath.Join(dir, "work"))

	// 매칭: ~work 디렉터리 아래 리포에서 실행
	t.Chdir(workRepo)
	cfg, err := config.Load(writeConfigAt(t, workRepo, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("gitdir 매칭 시 include 가 적용되어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}

	// 비매칭: 다른 디렉터리 리포에서는 포함되지 않음 (기본값 5)
	t.Chdir(otherRepo)
	cfg2, err := config.Load(writeConfigAt(t, otherRepo, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg2.CommentLanguage.MinLength != 5 {
		t.Errorf("gitdir 비매칭 시 include 는 무시되어야 함: got %d, want 5(기본값)", cfg2.CommentLanguage.MinLength)
	}
}

func TestInclude_Gitdir_슬래시끝_하위디렉터리매칭(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()
	// '/' 로 끝나는 패턴은 '**' 가 덧붙어 깊은 하위 디렉터리까지 매칭 (git 과 동일)
	deepRepo := filepath.Join(dir, "work", "team", "sub", "repo")
	if err := os.MkdirAll(deepRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(dir, "work.yml"), "comment_language:\n  min_length: 7\n")

	t.Chdir(deepRepo)
	cfg, err := config.Load(writeConfigAt(t, deepRepo, fmt.Sprintf(`
include:
  - path: %s
    gitdir: %s/
`, filepath.Join(dir, "work.yml"), filepath.Join(dir, "work"))))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("'/' 끝 패턴은 깊은 하위 디렉터리도 매칭해야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
}

func TestInclude_틸드확장(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	// path 와 gitdir 모두 '~' 가 홈 디렉터리로 확장됨
	workRepo := filepath.Join(tmpHome, "work", "repo")
	if err := os.MkdirAll(workRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(tmpHome, "inc.yml"), "comment_language:\n  min_length: 7\n")

	t.Chdir(workRepo)
	cfg, err := config.Load(writeConfigAt(t, workRepo, `
include:
  - path: ~/inc.yml
    gitdir: ~/work/
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("~ 확장된 path/gitdir 가 동작해야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
}

func TestInclude_상대경로는설정파일기준(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()
	other := t.TempDir()

	writeFileAt(t, filepath.Join(dir, "extra.yml"), "comment_language:\n  min_length: 7\n")
	path := writeConfigAt(t, dir, "include:\n  - path: extra.yml\n")

	// 작업 디렉터리가 달라도 include 를 선언한 설정 파일 기준으로 해석됨
	t.Chdir(other)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("상대 경로 include 는 설정 파일 기준이어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
}

func TestInclude_누락파일은건너뜀(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	writeFileAt(t, filepath.Join(dir, "exists.yml"), "comment_language:\n  min_length: 7\n")
	path := writeConfigAt(t, dir, fmt.Sprintf(`
include:
  - path: %s
  - path: %s
`, filepath.Join(dir, "no-such-file.yml"), filepath.Join(dir, "exists.yml")))

	// 누락 파일은 Warn 후 건너뛰고 Load 는 성공해야 함
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("누락 include 가 있어도 Load 는 성공해야 함: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("존재하는 include 는 적용되어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
}

func TestInclude_중첩include무시(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	// nested.yml 은 inc.yml 안에서 include 되지만 중첩 금지로 무시되어야 함
	writeFileAt(t, filepath.Join(dir, "nested.yml"), "comment_language:\n  locale: en\n")
	writeFileAt(t, filepath.Join(dir, "inc.yml"), fmt.Sprintf(`
include:
  - path: %s
comment_language:
  min_length: 7
`, filepath.Join(dir, "nested.yml")))
	path := writeConfigAt(t, dir, fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(dir, "inc.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("1단계 include 본문은 적용되어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
	if cfg.CommentLanguage.GetLocale() != "korean" {
		t.Errorf("중첩 include 는 무시되어야 함 (locale 기본값 유지): got %q", cfg.CommentLanguage.GetLocale())
	}
}

func TestInclude_구버전스키마자동마이그레이션(t *testing.T) {
	isolateGlobalPaths(t)
	dir := t.TempDir()

	// v1.0.x 의 no_coauthor 필드: 마이그레이션되면 no_ai_coauthor: false 가 됨
	writeFileAt(t, filepath.Join(dir, "old.yml"), "commit_message:\n  no_coauthor: false\n")
	path := writeConfigAt(t, dir, fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(dir, "old.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("include 파일의 no_coauthor: false 가 no_ai_coauthor 로 마이그레이션되어야 함")
	}
}

func TestInclude_전역include_프로젝트존재시무시(t *testing.T) {
	isolateGlobalPaths(t)
	incDir := t.TempDir()

	// 전역 설정의 include (무조건) — min_length 7, locale en
	writeFileAt(t, filepath.Join(incDir, "global-base.yml"), "comment_language:\n  min_length: 7\n  locale: en\n")
	writeXDGGlobal(t, fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(incDir, "global-base.yml")))

	// 프로젝트 설정이 존재하면 전역(과 전역의 include)은 완전히 무시됨
	path := writeConfig(t, "comment_language:\n  min_length: 3\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 3 {
		t.Errorf("프로젝트 본문 값이 적용되어야 함: got %d, want 3", cfg.CommentLanguage.MinLength)
	}
	if cfg.CommentLanguage.GetLocale() != "korean" {
		t.Errorf("전역 include 의 locale 은 무시되어야 함 (기본 korean): got %q", cfg.CommentLanguage.GetLocale())
	}
}

func TestInclude_프로젝트존재시_프로젝트include만동작(t *testing.T) {
	isolateGlobalPaths(t)
	incDir := t.TempDir()

	writeFileAt(t, filepath.Join(incDir, "global-inc.yml"), "comment_language:\n  allowed_words: [GlobalIncWord]\n")
	writeFileAt(t, filepath.Join(incDir, "project-inc.yml"), "comment_language:\n  allowed_words: [ProjectIncWord]\n")
	writeXDGGlobal(t, fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(incDir, "global-inc.yml")))

	path := writeConfig(t, fmt.Sprintf(`
include:
  - path: %s
comment_language:
  allowed_words:
    - ProjectWord
`, filepath.Join(incDir, "project-inc.yml")))

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	found := map[string]bool{}
	for _, w := range cfg.CommentLanguage.AllowedWords {
		found[w] = true
	}
	// 프로젝트가 선언한 include 와 본문 목록은 병합됨
	for _, want := range []string{"ProjectIncWord", "ProjectWord"} {
		if !found[want] {
			t.Errorf("프로젝트 include + 본문 목록이 병합되어야 함: %q 누락 (got %v)",
				want, cfg.CommentLanguage.AllowedWords)
		}
	}
	// 전역 include 의 목록은 섞이면 안 됨 (전역 완전 무시)
	if found["GlobalIncWord"] {
		t.Errorf("프로젝트 설정 존재 시 전역 include 의 목록이 섞이면 안 됨: %v", cfg.CommentLanguage.AllowedWords)
	}
}

func TestInclude_프리셋의include는무시(t *testing.T) {
	isolateGlobalPaths(t)
	incDir := t.TempDir()

	// 원격 프리셋이 로컬 파일을 끌어오지 못해야 함 (보안)
	writeFileAt(t, filepath.Join(incDir, "local.yml"), "comment_language:\n  min_length: 9\n")
	presetYAML := fmt.Sprintf("include:\n  - path: %s\n", filepath.Join(incDir, "local.yml"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(presetYAML))
	}))
	defer srv.Close()

	path := writeConfig(t, fmt.Sprintf("preset:\n  url: %s\n", srv.URL))
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 5 {
		t.Errorf("preset 안의 include 는 무시되어야 함: got %d, want 5(기본값)", cfg.CommentLanguage.MinLength)
	}
}

// TestInclude_사용자시나리오: 전역 config.yml 이 base.yml(항상) + work.yml(gitdir 조건)을
// include — work 디렉터리 리포와 일반 리포에서 서로 다른 정책이 적용되는지 확인.
func TestInclude_사용자시나리오_디렉터리별정책(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	cfgDir := filepath.Join(tmpHome, ".config", "commit-checker")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	// 범용 공유 설정 (항상 포함)
	writeFileAt(t, filepath.Join(cfgDir, "base.yml"), `
comment_language:
  min_length: 7
  allowed_words:
    - BaseWord
`)
	// work 디렉터리 전용 정책
	writeFileAt(t, filepath.Join(cfgDir, "work.yml"), `
comment_language:
  locale: en
  allowed_words:
    - WorkWord
`)
	// 전역 설정: 무조건 include + gitdir 조건 include
	writeFileAt(t, filepath.Join(cfgDir, "config.yml"), `
include:
  - path: ~/.config/commit-checker/base.yml
  - path: ~/.config/commit-checker/work.yml
    gitdir: ~/work/
`)

	workRepo := filepath.Join(tmpHome, "work", "repo")
	homeRepo := filepath.Join(tmpHome, "personal", "repo")
	for _, d := range []string{workRepo, homeRepo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// ~/work/ 아래 리포: base + work 정책 모두 적용
	t.Chdir(workRepo)
	cfg, err := config.Load(filepath.Join(workRepo, ".commit-checker.yml")) // 프로젝트 설정 없음
	if err != nil {
		t.Fatalf("Load(work): %v", err)
	}
	if cfg.CommentLanguage.GetLocale() != "english" {
		t.Errorf("work 리포는 work.yml 의 locale: en 이 적용되어야 함: got %q", cfg.CommentLanguage.GetLocale())
	}
	if cfg.CommentLanguage.MinLength != 7 {
		t.Errorf("work 리포에도 base.yml 의 min_length 가 적용되어야 함: got %d, want 7", cfg.CommentLanguage.MinLength)
	}
	words := map[string]bool{}
	for _, w := range cfg.CommentLanguage.AllowedWords {
		words[w] = true
	}
	if !words["BaseWord"] || !words["WorkWord"] {
		t.Errorf("work 리포는 base+work allowed_words 가 모두 병합되어야 함: %v", cfg.CommentLanguage.AllowedWords)
	}

	// 일반 리포: base 정책만 적용 (work.yml 비매칭)
	t.Chdir(homeRepo)
	cfg2, err := config.Load(filepath.Join(homeRepo, ".commit-checker.yml")) // 프로젝트 설정 없음
	if err != nil {
		t.Fatalf("Load(personal): %v", err)
	}
	if cfg2.CommentLanguage.GetLocale() != "korean" {
		t.Errorf("일반 리포는 work.yml 이 적용되지 않아야 함 (기본 korean): got %q", cfg2.CommentLanguage.GetLocale())
	}
	if cfg2.CommentLanguage.MinLength != 7 {
		t.Errorf("일반 리포에도 base.yml 은 적용되어야 함: got %d, want 7", cfg2.CommentLanguage.MinLength)
	}
	for _, w := range cfg2.CommentLanguage.AllowedWords {
		if w == "WorkWord" {
			t.Errorf("일반 리포에 work.yml 의 allowed_words 가 섞이면 안 됨: %v", cfg2.CommentLanguage.AllowedWords)
		}
	}
}
