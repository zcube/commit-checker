package comment_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

// testdataPath 는 testdata 디렉터리 경로를 반환합니다.
func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

// loadTestdata 는 testdata 파일을 읽어 내용을 반환합니다.
func loadTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("testdata 파일 읽기 실패 %s: %v", name, err)
	}
	return string(data)
}

// parseResult 는 파싱 결과를 종류별로 분류합니다.
type parseResult struct {
	comments []comment.Comment
	strings  []comment.Comment
	imports  []comment.Comment
}

func classify(all []comment.Comment) parseResult {
	var r parseResult
	for _, c := range all {
		switch c.Kind {
		case comment.KindComment:
			r.comments = append(r.comments, c)
		case comment.KindString:
			r.strings = append(r.strings, c)
		case comment.KindImport:
			r.imports = append(r.imports, c)
		}
	}
	return r
}

// assertImportNotInStrings 는 import/include 경로가 KindString으로 추출되지 않았는지 확인합니다.
func assertImportNotInStrings(t *testing.T, r parseResult, importPaths ...string) {
	t.Helper()
	for _, imp := range importPaths {
		for _, s := range r.strings {
			if s.Text == imp {
				t.Errorf("import 경로 %q 가 KindString으로 추출됨 — KindImport 이어야 함 (line %d)", imp, s.Line)
			}
		}
	}
}

// assertHasImport 는 특정 경로가 KindImport 로 추출되었는지 확인합니다.
func assertHasImport(t *testing.T, r parseResult, path string) {
	t.Helper()
	for _, imp := range r.imports {
		if imp.Text == path {
			return
		}
	}
	t.Errorf("KindImport 에서 %q 를 찾을 수 없음. 실제 imports: %v", path, r.imports)
}

// assertHasComment 는 특정 텍스트를 포함하는 KindComment 가 있는지 확인합니다.
func assertHasComment(t *testing.T, r parseResult, substr string) {
	t.Helper()
	for _, c := range r.comments {
		if containsText(c.Text, substr) {
			return
		}
	}
	t.Errorf("KindComment 에서 %q 를 포함하는 항목을 찾을 수 없음", substr)
}

// ---- Go ---------------------------------------------------------------

func TestGoParser_Testdata(t *testing.T) {
	src := loadTestdata(t, "sample.go")
	p := &comment.GoParser{}
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	// 주석이 추출되어야 함
	assertHasComment(t, r, "패키지 수준 상수")
	assertHasComment(t, r, "입력 값 유효성 검사")
	assertHasComment(t, r, "블록 주석")

	// import 경로가 KindImport 여야 함
	assertHasImport(t, r, "fmt")
	assertHasImport(t, r, "strings")
	assertHasImport(t, r, "github.com/example/somepackage")

	// import 경로가 KindString 으로 누출되지 않아야 함
	assertImportNotInStrings(t, r, "fmt", "strings", "github.com/example/somepackage")
}

// ---- TypeScript -------------------------------------------------------

func TestCStyleParser_Testdata_TypeScript(t *testing.T) {
	src := loadTestdata(t, "sample.ts")
	p := comment.NewCStyleParser([]string{".ts"}, true)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "사용자 정보를 나타내는 인터페이스")
	assertHasComment(t, r, "API에서 사용자 데이터를")

	// import 경로가 KindString 으로 누출되지 않아야 함
	assertImportNotInStrings(t, r, "react", "axios", "reflect-metadata")

	// from 'react' 는 KindImport 여야 함
	assertHasImport(t, r, "react")
	assertHasImport(t, r, "axios")
	assertHasImport(t, r, "reflect-metadata")
}

// ---- JavaScript -------------------------------------------------------

func TestCStyleParser_Testdata_JavaScript(t *testing.T) {
	src := loadTestdata(t, "sample.js")
	p := comment.NewCStyleParser([]string{".js"}, true)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "설정 파일을 불러오는 함수")
	assertHasComment(t, r, "파일 내용을 읽어서")

	assertImportNotInStrings(t, r, "path", "dotenv/config")
	assertHasImport(t, r, "path")
	assertHasImport(t, r, "dotenv/config")
}

// ---- Java -------------------------------------------------------------

func TestCStyleParser_Testdata_Java(t *testing.T) {
	src := loadTestdata(t, "sample.java")
	p := comment.NewCStyleParser([]string{".java"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "사용자 서비스 클래스")
	assertHasComment(t, r, "입력 유효성 검사 후 추가")

	// Java import 는 문자열 리터럴이 없으므로 KindImport 는 0개여야 함
	if len(r.imports) != 0 {
		t.Errorf("Java import 문에서 KindImport 가 생성되지 않아야 함, 실제: %d개", len(r.imports))
	}
}

// ---- Kotlin -----------------------------------------------------------

func TestCStyleParser_Testdata_Kotlin(t *testing.T) {
	src := loadTestdata(t, "sample.kt")
	p := comment.NewCStyleParser([]string{".kt"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "아이템 저장소 인터페이스")
	assertHasComment(t, r, "데이터베이스 기반 아이템 저장소")
}

// ---- Python -----------------------------------------------------------

func TestPythonParser_Testdata(t *testing.T) {
	src := loadTestdata(t, "sample.py")
	p := &comment.PythonParser{}
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "설정 파일 기본 경로")
	assertHasComment(t, r, "경로가 지정되지 않으면")

	// Python import 는 문자열이 없으므로 KindImport 0개
	if len(r.imports) != 0 {
		t.Errorf("Python import 에서 KindImport 가 생성되지 않아야 함, 실제: %d개", len(r.imports))
	}
}

// ---- C ----------------------------------------------------------------

func TestCStyleParser_Testdata_C(t *testing.T) {
	src := loadTestdata(t, "sample.c")
	p := comment.NewCStyleParser([]string{".c"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "프로그램의 진입점")
	assertHasComment(t, r, "인자 수 확인")

	// #include "utils.h" 와 "config.h" 는 KindImport 여야 함
	assertHasImport(t, r, "utils.h")
	assertHasImport(t, r, "config.h")
	assertImportNotInStrings(t, r, "utils.h", "config.h")
}

// ---- C++ --------------------------------------------------------------

func TestCStyleParser_Testdata_CPP(t *testing.T) {
	src := loadTestdata(t, "sample.cpp")
	p := comment.NewCStyleParser([]string{".cpp"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "데이터 처리기 클래스")
	assertHasComment(t, r, "생성자")

	assertHasImport(t, r, "processor.hpp")
	assertImportNotInStrings(t, r, "processor.hpp")
}

// ---- C# ---------------------------------------------------------------

func TestCStyleParser_Testdata_CSharp(t *testing.T) {
	src := loadTestdata(t, "sample.cs")
	p := comment.NewCStyleParser([]string{".cs"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "주문 처리 서비스")
	assertHasComment(t, r, "주문 저장소")
}

// ---- Swift ------------------------------------------------------------

func TestCStyleParser_Testdata_Swift(t *testing.T) {
	src := loadTestdata(t, "sample.swift")
	p := comment.NewCStyleParser([]string{".swift"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "네트워크 요청을 관리하는 클래스")
	assertHasComment(t, r, "공유 인스턴스")
}

// ---- Rust -------------------------------------------------------------

func TestCStyleParser_Testdata_Rust(t *testing.T) {
	src := loadTestdata(t, "sample.rs")
	p := comment.NewCStyleParser([]string{".rs"}, false)
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	assertHasComment(t, r, "설정 구조체")
	assertHasComment(t, r, "파일에서 설정을 불러옵니다")
}

// ---- Dockerfile -------------------------------------------------------

func TestDockerfileParser_Testdata(t *testing.T) {
	src := loadTestdata(t, "Dockerfile")
	p := &comment.DockerfileParser{}
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	if len(r.comments) == 0 {
		t.Fatal("Dockerfile 에서 주석을 추출하지 못함")
	}
	assertHasComment(t, r, "빌드 스테이지")
	assertHasComment(t, r, "의존성 파일을 먼저 복사")
	assertHasComment(t, r, "실행 스테이지")

	// Dockerfile 에는 문자열 리터럴이 없어야 함
	if len(r.strings) != 0 {
		t.Errorf("Dockerfile 에서 KindString 이 추출됨: %v", r.strings)
	}
}

// ---- Markdown ---------------------------------------------------------

func TestMarkdownParser_Testdata(t *testing.T) {
	src := loadTestdata(t, "sample.md")
	p := &comment.MarkdownParser{}
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	r := classify(all)

	if len(r.comments) == 0 {
		t.Fatal("Markdown 에서 텍스트를 추출하지 못함")
	}
	assertHasComment(t, r, "프로젝트 소개")
	assertHasComment(t, r, "설치 방법")
	assertHasComment(t, r, "기여 방법")

	// 코드 블록 내부(go install ..., yaml 설정 등)는 추출되지 않아야 함
	for _, c := range r.comments {
		if containsText(c.Text, "go install") {
			t.Errorf("코드 블록 내용이 추출됨: %q (line %d)", c.Text, c.Line)
		}
		if containsText(c.Text, "enabled: true") {
			t.Errorf("코드 블록 내용이 추출됨: %q (line %d)", c.Text, c.Line)
		}
	}
}

// ---- GetParser + HasExtension 통합 테스트 ----------------------------

func TestGetParser_AllLanguages(t *testing.T) {
	cases := []struct {
		filename string
		wantNil  bool
	}{
		{"main.go", false},
		{"app.ts", false},
		{"app.tsx", false},
		{"index.js", false},
		{"index.jsx", false},
		{"Main.java", false},
		{"Main.kt", false},
		{"script.py", false},
		{"main.c", false},
		{"main.cpp", false},
		{"Program.cs", false},
		{"App.swift", false},
		{"lib.rs", false},
		{"Dockerfile", false},
		{"Dockerfile.prod", false},
		{"Dockerfile.dev", false},
		{"app.dockerfile", false},
		{"README.md", false},
		{"docs.markdown", false},
		{"unknown.xyz", true},
	}

	for _, tc := range cases {
		p := comment.GetParser(tc.filename)
		if tc.wantNil && p != nil {
			t.Errorf("GetParser(%q): 파서가 없어야 하지만 %T 반환됨", tc.filename, p)
		}
		if !tc.wantNil && p == nil {
			t.Errorf("GetParser(%q): 파서를 반환해야 하지만 nil 반환됨", tc.filename)
		}
	}
}
