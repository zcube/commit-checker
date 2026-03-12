package cmd_test

// End-to-end tests that compile the binary and run it against real git
// repositories created in temp directories.
//
// The binary is built once in TestMain and shared across all tests.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary into a temp directory.
	tmpDir, err := os.MkdirTemp("", "commit-checker-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	bin := filepath.Join(tmpDir, "commit-checker")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	// Build from the module root (one level up from cmd/).
	moduleRoot := moduleRootDir()
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, out)
		os.Exit(1)
	}
	binaryPath = bin

	os.Exit(m.Run())
}

// moduleRootDir returns the absolute path to the module root (parent of cmd/).
func moduleRootDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(file))
}

// ---- helpers ----------------------------------------------------------------

type testRepo struct {
	dir string
	t   *testing.T
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	dir := t.TempDir()
	r := &testRepo{dir: dir, t: t}
	r.git("init")
	r.git("config", "user.email", "test@commit-checker.test")
	r.git("config", "user.name", "E2E Test")
	return r
}

func (r *testRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	// Disable GPG signing for test repos (global config may require signing).
	cmd.Env = append(os.Environ(), "GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=commit.gpgsign",
		"GIT_CONFIG_VALUE_0=false")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func (r *testRepo) write(relPath, content string) {
	r.t.Helper()
	full := filepath.Join(r.dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *testRepo) stage(relPath, content string) {
	r.t.Helper()
	r.write(relPath, content)
	r.git("add", relPath)
}

func (r *testRepo) commit(msg string) {
	r.t.Helper()
	r.git("commit", "-m", msg)
}

func (r *testRepo) writeConfig(content string) {
	r.t.Helper()
	r.write(".commit-checker.yml", content)
}

// run executes the commit-checker binary in the repo directory.
// Returns (stdout+stderr, exit_code).
func (r *testRepo) run(args ...string) (string, int) {
	r.t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = r.dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	code := 0
	if err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			r.t.Fatalf("run error: %v", err)
		}
	}
	return buf.String(), code
}

// ---- commit-checker diff tests ----------------------------------------------

func TestE2E_Diff_KoreanComment_Exit0(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// 한국어 주석입니다
func main() {}
`)
	_, code := r.run("diff")
	if code != 0 {
		t.Errorf("expected exit 0 for Korean comment, got %d", code)
	}
}

func TestE2E_Diff_EnglishComment_Exit1(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// This comment is written in English only
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("expected exit 1 for English comment, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_NoStagedFiles_Exit0(t *testing.T) {
	r := newTestRepo(t)
	// No staged files
	_, code := r.run("diff")
	if code != 0 {
		t.Errorf("expected exit 0 with no staged files, got %d", code)
	}
}

func TestE2E_Diff_UnsupportedExtension_Exit0(t *testing.T) {
	r := newTestRepo(t)
	r.stage("notes.txt", "This is all English content in a plain text file.\n")
	_, code := r.run("diff")
	if code != 0 {
		t.Errorf("txt should be skipped, expected exit 0, got %d", code)
	}
}

func TestE2E_Diff_Disabled_Exit0(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  enabled: false\n")
	r.stage("main.go", `package main

// This English comment should be ignored when disabled
func main() {}
`)
	_, code := r.run("diff")
	if code != 0 {
		t.Errorf("disabled check should exit 0, got %d", code)
	}
}

func TestE2E_Diff_LocaleKo_EnglishFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: ko\n")
	r.stage("main.go", `package main

// English comment should fail with locale=ko
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("locale=ko should reject English comments, expected exit 1, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleEn_EnglishPasses(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: en\n")
	r.stage("main.go", `package main

// This English comment should pass with locale=en
func main() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locale=en should accept English comments, expected exit 0, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_IgnoreFiles(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  ignore_files:\n    - \"generated/**\"\n")
	r.stage("generated/gen.go", `package gen

// This English comment is in an ignored path
func Gen() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("ignored file should not fail, expected exit 0, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_Languages_GoOnly(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  languages:\n    - go\n")
	// Stage TypeScript (should be ignored) and Go (should be checked)
	r.stage("app.ts", "// This English TypeScript comment should be ignored\nconst x = 1;\n")
	r.stage("main.go", `package main

// 한국어 주석 — 통과
func main() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("only go files checked; Korean Go comment should pass, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleJa_JapanesePasses(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: ja\n")
	r.stage("main.go", `package main

// これはひらがなとカタカナのコメントです
func main() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locale=ja should accept Japanese kana comments, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleJa_KoreanFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: ja\n")
	r.stage("main.go", `package main

// 이것은 한국어 주석입니다
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("locale=ja should reject Korean comments, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleJa_EnglishFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: ja\n")
	r.stage("main.go", `package main

// This is an English comment
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("locale=ja should reject English comments, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleZh_ChinesePasses(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: zh\n")
	r.stage("main.go", `package main

// 这是一个处理用户数据的函数
func main() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locale=zh should accept Chinese comments, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleZh_KoreanFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: zh\n")
	r.stage("main.go", `package main

// 이것은 한국어 주석입니다
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("locale=zh should reject Korean comments, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_LocaleZh_KanaOnlyFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: zh\n")
	// kana-only text contains no CJK characters
	r.stage("main.go", `package main

// これはひらがなとカタカナのコメントです
func main() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("locale=zh should reject kana-only Japanese (no CJK), got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_Python_Japanese_Locale(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: ja\n")
	r.stage("utils.py", `# ユーザーデータを処理する関数
def process():
    pass
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("Japanese Python comment should pass locale=ja, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_TypeScript_Chinese_Locale(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig("comment_language:\n  locale: zh\n")
	r.stage("service.ts", `// 处理用户请求的服务函数
const handler = () => {};
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("Chinese TypeScript comment should pass locale=zh, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Diff_OutputContainsFilePath(t *testing.T) {
	r := newTestRepo(t)
	r.stage("service/handler.go", `package handler

// This English comment should produce an error with the file path
func Handle() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strContains(out, "handler.go") {
		t.Errorf("output should mention the file path, got:\n%s", out)
	}
}

// ---- file_languages E2E tests -----------------------------------------------

func TestE2E_FileLanguages_AnyAllowsEnglish(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`comment_language:
  required_language: korean
  file_languages:
    - pattern: "locales/**"
      language: any
`)
	r.stage("locales/en.go", `package locales

// This is an English comment in a locale file
const Hello = "Hello"
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locales/** any: English should pass, exit %d\n%s", code, out)
	}
}

func TestE2E_FileLanguages_EnglishOverride(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`comment_language:
  required_language: korean
  file_languages:
    - pattern: "i18n/**"
      language: english
`)
	r.stage("i18n/messages.go", `package i18n

// All comments in this file are English
const Key = "value"
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("i18n/** english: English should pass, exit %d\n%s", code, out)
	}
}

func TestE2E_FileLanguages_Japanese_Override(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`comment_language:
  required_language: korean
  file_languages:
    - pattern: "locale/ja/**"
      language: ja
`)
	r.stage("locale/ja/msg.go", `package ja

// これは日本語の翻訳ファイルです
const Hello = "こんにちは"
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locale/ja/** ja: Japanese should pass, exit %d\n%s", code, out)
	}
}

func TestE2E_FileLanguages_Chinese_Override(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`comment_language:
  required_language: korean
  file_languages:
    - pattern: "locale/zh/**"
      language: zh
`)
	r.stage("locale/zh/msg.go", `package zh

// 这是中文翻译文件的注释
const Hello = "你好"
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("locale/zh/** zh: Chinese should pass, exit %d\n%s", code, out)
	}
}

func TestE2E_FileLanguages_MultiRule_NonMatchFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`comment_language:
  required_language: korean
  file_languages:
    - pattern: "locales/**"
      language: any
`)
	// This file does NOT match locales/** so Korean is required → English fails
	r.stage("service/handler.go", `package handler

// This English comment is not in an overridden path
func Handle() {}
`)
	out, code := r.run("diff")
	if code != 1 {
		t.Errorf("non-matching file should use Korean default and fail, exit %d\n%s", code, out)
	}
}

// ---- inline directive E2E tests ---------------------------------------------

func TestE2E_Directive_Ignore(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:ignore
// This English comment should be ignored
func f() {}

// 이 한국어 주석은 통과합니다
func g() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf(":ignore should suppress the next comment, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_Disable_Enable(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:disable
// This entire region is in English and is exempted
// More English here
// commit-checker:enable

// 한국어로 돌아옴 — 정상 체크
func g() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf(":disable/:enable region should pass, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_Disable_WithLang_English(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:disable:lang=english
// This section uses English intentionally
// commit-checker:enable

// 한국어 정상
func g() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf(":disable:lang=english should accept English, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_FileLang_English(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:file-lang=english

// This entire file uses English comments
// All comments here are intentionally English
func process() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("file-lang=english should accept English, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_FileLang_Japanese(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:file-lang=ja

// これはひらがなとカタカナのコメントです
// ユーザー処理のロジック
func process() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("file-lang=ja should accept Japanese, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_FileLang_Chinese(t *testing.T) {
	r := newTestRepo(t)
	r.stage("main.go", `package main

// commit-checker:file-lang=zh

// 这是一个中文注释的示例内容
func process() {}
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("file-lang=zh should accept Chinese, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_Python_Disable(t *testing.T) {
	r := newTestRepo(t)
	r.stage("utils.py", `# commit-checker:disable
# This English Python comment is exempted
# commit-checker:enable

# 한국어 주석은 통과
def process():
    pass
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("Python :disable/:enable should work, exit %d\n%s", code, out)
	}
}

func TestE2E_Directive_TypeScript_FileLang(t *testing.T) {
	r := newTestRepo(t)
	r.stage("i18n.ts", `// commit-checker:file-lang=en
// This TypeScript file has English comments
const msg = "hello";
`)
	out, code := r.run("diff")
	if code != 0 {
		t.Errorf("TypeScript file-lang=en should accept English, exit %d\n%s", code, out)
	}
}

// ---- commit-checker msg tests -----------------------------------------------

func TestE2E_Msg_Clean_Exit0(t *testing.T) {
	r := newTestRepo(t)
	msgFile := writeMsgFile(t, "feat: 새로운 기능 추가\n\n정상적인 커밋 메시지입니다.\n")
	_, code := r.run("msg", msgFile)
	if code != 0 {
		t.Errorf("clean message expected exit 0, got %d", code)
	}
}

func TestE2E_Msg_CoAuthor_AI_Exit1(t *testing.T) {
	r := newTestRepo(t)
	msgFile := writeMsgFile(t, "feat: add feature\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")
	out, code := r.run("msg", msgFile)
	if code != 1 {
		t.Errorf("AI co-author should cause exit 1, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_CoAuthor_Human_Exit0(t *testing.T) {
	r := newTestRepo(t)
	msgFile := writeMsgFile(t, "feat: add feature\n\nCo-authored-by: Alice <alice@myteam.com>\n")
	out, code := r.run("msg", msgFile)
	if code != 0 {
		t.Errorf("human co-author should exit 0, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_InvisibleChar_Exit1(t *testing.T) {
	r := newTestRepo(t)
	// U+00A0 NO-BREAK SPACE
	msgFile := writeMsgFile(t, "feat: hello\u00A0world\n")
	out, code := r.run("msg", msgFile)
	if code != 1 {
		t.Errorf("NBSP should cause exit 1, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_AmbiguousChar_Exit1(t *testing.T) {
	r := newTestRepo(t)
	// U+0410 Cyrillic А
	msgFile := writeMsgFile(t, "feat: \u0410mbiguous character\n")
	out, code := r.run("msg", msgFile)
	if code != 1 {
		t.Errorf("ambiguous char should cause exit 1, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_BOM_Exit0(t *testing.T) {
	r := newTestRepo(t)
	// BOM (U+FEFF) must be allowed
	msgFile := writeMsgFile(t, "\uFEFFfeat: commit with BOM\n")
	out, code := r.run("msg", msgFile)
	if code != 0 {
		t.Errorf("BOM should be allowed, expected exit 0, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_BadRune_Exit1(t *testing.T) {
	r := newTestRepo(t)
	msgFile := writeMsgFile(t, "feat: bad\x80rune\n")
	out, code := r.run("msg", msgFile)
	if code != 1 {
		t.Errorf("bad UTF-8 should cause exit 1, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_LanguageCheck_Enabled_KoreanFails(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`commit_message:
  no_ai_coauthor: false
  no_unicode_spaces: false
  no_ambiguous_chars: false
  no_bad_runes: false
  language_check:
    enabled: true
    required_language: korean
`)
	msgFile := writeMsgFile(t, "add new feature for user authentication\n")
	out, code := r.run("--config", filepath.Join(r.dir, ".commit-checker.yml"), "msg", msgFile)
	if code != 1 {
		t.Errorf("English commit message should fail Korean language check, got exit %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_LanguageCheck_MergeSkipped(t *testing.T) {
	r := newTestRepo(t)
	r.writeConfig(`commit_message:
  no_ai_coauthor: false
  no_unicode_spaces: false
  no_ambiguous_chars: false
  no_bad_runes: false
  language_check:
    enabled: true
    required_language: korean
    skip_prefixes:
      - "Merge"
`)
	msgFile := writeMsgFile(t, "Merge branch 'feature/xyz' into main\n")
	out, code := r.run("--config", filepath.Join(r.dir, ".commit-checker.yml"), "msg", msgFile)
	if code != 0 {
		t.Errorf("Merge commit should be skipped, expected exit 0, got %d\noutput: %s", code, out)
	}
}

func TestE2E_Msg_MultipleViolations_AllReported(t *testing.T) {
	r := newTestRepo(t)
	// Both co-author AND invisible char
	msgFile := writeMsgFile(t, "feat: hello\u00A0world\n\nCo-authored-by: Copilot <github-copilot[bot]@users.noreply.github.com>\n")
	out, code := r.run("msg", msgFile)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strContains(out, "co-author") {
		t.Errorf("co-author error not reported:\n%s", out)
	}
	if !strContains(out, "invisible") {
		t.Errorf("invisible char error not reported:\n%s", out)
	}
}

// ---- commit-checker fix --dry-run tests -------------------------------------

func TestE2E_Fix_DryRun_ShowsCoAuthor(t *testing.T) {
	r := newTestRepo(t)
	// Seed commit so HEAD~1 exists
	r.stage("init.go", "package main\n")
	r.commit("chore: 초기화")
	// Violating commit
	r.write("init.go", "package main\nfunc f(){}\n")
	r.git("add", "init.go")
	r.commit("feat: 초기 커밋\n\nCo-authored-by: Claude <noreply@anthropic.com>")

	out, code := r.run("fix", "--range", "HEAD~1..HEAD", "--dry-run")
	if code != 0 {
		t.Fatalf("fix --dry-run should exit 0, got %d\noutput: %s", code, out)
	}
	if !strContains(out, "co-author") {
		t.Errorf("dry-run should report co-author fix:\n%s", out)
	}
}

func TestE2E_Fix_DryRun_CleanHistory(t *testing.T) {
	r := newTestRepo(t)
	// Seed commit so HEAD~1 exists
	r.stage("init.go", "package main\n")
	r.commit("chore: 초기화")
	// Clean commit
	r.write("init.go", "package main\nfunc f(){}\n")
	r.git("add", "init.go")
	r.commit("feat: 정상 커밋 메시지")

	out, code := r.run("fix", "--range", "HEAD~1..HEAD", "--dry-run")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\noutput: %s", code, out)
	}
	if !strContains(out, "no auto-fixable") {
		t.Errorf("should report no violations for clean commit:\n%s", out)
	}
}

func TestE2E_Fix_DryRun_AmbiguousChar(t *testing.T) {
	r := newTestRepo(t)
	// Seed commit so HEAD~1 exists
	r.stage("init.go", "package main\n")
	r.commit("chore: 초기화")
	// Violating commit: U+0410 Cyrillic А in commit message
	r.write("init.go", "package main\nfunc f(){}\n")
	r.git("add", "init.go")
	r.commit("feat: \u0410dd ambiguous char")

	out, code := r.run("fix", "--range", "HEAD~1..HEAD", "--dry-run")
	if code != 0 {
		t.Fatalf("fix --dry-run should exit 0, got %d\noutput: %s", code, out)
	}
	if !strContains(out, "ambiguous") {
		t.Errorf("dry-run should report ambiguous char fix:\n%s", out)
	}
}

// ---- helpers ----------------------------------------------------------------

func writeMsgFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "COMMIT_EDITMSG*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func strContains(s, sub string) bool {
	return len(s) >= len(sub) && findStr(s, sub)
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
