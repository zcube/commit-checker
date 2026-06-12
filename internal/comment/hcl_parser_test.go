package comment_test

import (
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

// parseHCL 은 HCL 소스를 파싱해 종류별로 분류한 결과를 반환합니다.
func parseHCL(t *testing.T, src string) parseResult {
	t.Helper()
	p := &comment.HCLParser{}
	all, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("파싱 오류: %v", err)
	}
	return classify(all)
}

func TestHCLParser_HashLineComment(t *testing.T) {
	src := `# 해시 줄 주석입니다
resource "aws_instance" "web" {
  ami = "ami-12345" # 인라인 해시 주석
}
`
	r := parseHCL(t, src)

	if len(r.comments) != 2 {
		t.Fatalf("expected 2 comments, got %d: %+v", len(r.comments), r.comments)
	}
	if r.comments[0].Text != "해시 줄 주석입니다" || r.comments[0].Line != 1 {
		t.Errorf("첫 번째 주석 불일치: %+v", r.comments[0])
	}
	if r.comments[1].Text != "인라인 해시 주석" || r.comments[1].Line != 3 {
		t.Errorf("두 번째 주석 불일치: %+v", r.comments[1])
	}
}

func TestHCLParser_SlashLineComment(t *testing.T) {
	src := `// 슬래시 줄 주석입니다
variable "name" {
  default = "value" // 인라인 슬래시 주석
}
`
	r := parseHCL(t, src)

	if len(r.comments) != 2 {
		t.Fatalf("expected 2 comments, got %d: %+v", len(r.comments), r.comments)
	}
	if r.comments[0].Text != "슬래시 줄 주석입니다" || r.comments[0].Line != 1 {
		t.Errorf("첫 번째 주석 불일치: %+v", r.comments[0])
	}
	if r.comments[1].Text != "인라인 슬래시 주석" || r.comments[1].Line != 3 {
		t.Errorf("두 번째 주석 불일치: %+v", r.comments[1])
	}
}

func TestHCLParser_BlockComment(t *testing.T) {
	src := `/* 블록 주석
   두 번째 줄 */
locals {
  x = 1
}
`
	r := parseHCL(t, src)

	if len(r.comments) != 1 {
		t.Fatalf("expected 1 comment, got %d: %+v", len(r.comments), r.comments)
	}
	c := r.comments[0]
	if !c.IsBlock {
		t.Errorf("블록 주석이어야 함: %+v", c)
	}
	if c.Line != 1 || c.EndLine != 2 {
		t.Errorf("블록 주석 줄 번호 불일치 (want 1-2): %+v", c)
	}
	if !strings.Contains(c.Text, "블록 주석") || !strings.Contains(c.Text, "두 번째 줄") {
		t.Errorf("블록 주석 텍스트 불일치: %q", c.Text)
	}
}

func TestHCLParser_CommentMarkersInsideString(t *testing.T) {
	// 문자열 안의 # // /* 는 주석으로 오인되지 않아야 함
	src := `locals {
  a = "url#fragment"
  b = "https://example.com"
  c = "glob/*.txt"
}
`
	r := parseHCL(t, src)

	if len(r.comments) != 0 {
		t.Errorf("문자열 내 주석 기호가 주석으로 오인됨: %+v", r.comments)
	}
	wantStrings := []string{"url#fragment", "https://example.com", "glob/*.txt"}
	if len(r.strings) != len(wantStrings) {
		t.Fatalf("expected %d strings, got %d: %+v", len(wantStrings), len(r.strings), r.strings)
	}
	for i, want := range wantStrings {
		if r.strings[i].Text != want {
			t.Errorf("strings[%d] = %q, want %q", i, r.strings[i].Text, want)
		}
	}
}

func TestHCLParser_InterpolationNestedQuotes(t *testing.T) {
	// 인터폴레이션 내부의 중첩 따옴표가 외부 문자열을 조기 종료시키지 않아야 함
	src := `locals {
  v = "x-${var.a == "b" ? 1 : 2}-y" # 뒤쪽 주석
}
`
	r := parseHCL(t, src)

	if len(r.strings) != 1 {
		t.Fatalf("expected 1 string, got %d: %+v", len(r.strings), r.strings)
	}
	want := `x-${var.a == "b" ? 1 : 2}-y`
	if r.strings[0].Text != want {
		t.Errorf("string = %q, want %q", r.strings[0].Text, want)
	}
	if len(r.comments) != 1 || r.comments[0].Text != "뒤쪽 주석" || r.comments[0].Line != 2 {
		t.Errorf("인터폴레이션 뒤 주석 불일치: %+v", r.comments)
	}
}

func TestHCLParser_InterpolationNestedBraces(t *testing.T) {
	// 인터폴레이션 안의 중첩 중괄호({ } 객체)와 중첩 ${} 추적
	src := `locals {
  v = "a-${jsonencode({ key = "v-${var.x}" })}-z"
}
`
	r := parseHCL(t, src)

	if len(r.strings) != 1 {
		t.Fatalf("expected 1 string, got %d: %+v", len(r.strings), r.strings)
	}
	want := `a-${jsonencode({ key = "v-${var.x}" })}-z`
	if r.strings[0].Text != want {
		t.Errorf("string = %q, want %q", r.strings[0].Text, want)
	}
}

func TestHCLParser_DollarLiteralEscape(t *testing.T) {
	// $${ 와 %%{ 는 인터폴레이션이 아닌 리터럴 이스케이프
	src := `locals {
  a = "literal $${not_interp} text"
  b = "literal %%{not_directive} text"
}
`
	r := parseHCL(t, src)

	if len(r.strings) != 2 {
		t.Fatalf("expected 2 strings, got %d: %+v", len(r.strings), r.strings)
	}
	if r.strings[0].Text != "literal ${not_interp} text" {
		t.Errorf("strings[0] = %q", r.strings[0].Text)
	}
	if r.strings[1].Text != "literal %{not_directive} text" {
		t.Errorf("strings[1] = %q", r.strings[1].Text)
	}
}

func TestHCLParser_TemplateDirective(t *testing.T) {
	// %{ ... } 템플릿 디렉티브도 인터폴레이션과 동일하게 추적
	src := `locals {
  v = "%{ if var.env == "prod" }production%{ else }dev%{ endif }"
}
`
	r := parseHCL(t, src)

	if len(r.strings) != 1 {
		t.Fatalf("expected 1 string, got %d: %+v", len(r.strings), r.strings)
	}
	want := `%{ if var.env == "prod" }production%{ else }dev%{ endif }`
	if r.strings[0].Text != want {
		t.Errorf("string = %q, want %q", r.strings[0].Text, want)
	}
}

func TestHCLParser_Heredoc(t *testing.T) {
	src := `resource "aws_instance" "web" {
  user_data = <<EOF
#!/bin/bash
# this hash is not a comment
echo "hello"
EOF
}
`
	r := parseHCL(t, src)

	// heredoc 본문의 # 줄이 주석으로 오인되지 않아야 함
	if len(r.comments) != 0 {
		t.Errorf("heredoc 본문이 주석으로 오인됨: %+v", r.comments)
	}

	// 문자열: "aws_instance", "web", heredoc 본문 — 3개
	if len(r.strings) != 3 {
		t.Fatalf("expected 3 strings, got %d: %+v", len(r.strings), r.strings)
	}
	h := r.strings[2]
	wantBody := "#!/bin/bash\n# this hash is not a comment\necho \"hello\""
	if h.Text != wantBody {
		t.Errorf("heredoc 본문 = %q, want %q", h.Text, wantBody)
	}
	if h.Line != 2 || h.EndLine != 6 {
		t.Errorf("heredoc 줄 번호 불일치 (want 2-6): Line=%d EndLine=%d", h.Line, h.EndLine)
	}
}

func TestHCLParser_HeredocIndented(t *testing.T) {
	// <<- 형식: 닫는 라벨 앞에 공백 허용
	src := `locals {
  doc = <<-EOT
    line one
    line two
  EOT
}
`
	r := parseHCL(t, src)

	if len(r.strings) != 1 {
		t.Fatalf("expected 1 string, got %d: %+v", len(r.strings), r.strings)
	}
	h := r.strings[0]
	if !strings.Contains(h.Text, "line one") || !strings.Contains(h.Text, "line two") {
		t.Errorf("heredoc 본문 불일치: %q", h.Text)
	}
	if strings.Contains(h.Text, "EOT") {
		t.Errorf("닫는 라벨이 본문에 포함됨: %q", h.Text)
	}
	if h.Line != 2 || h.EndLine != 5 {
		t.Errorf("heredoc 줄 번호 불일치 (want 2-5): Line=%d EndLine=%d", h.Line, h.EndLine)
	}
}

func TestHCLParser_HeredocPlainAllowsIndentedLabel(t *testing.T) {
	// hashicorp/hcl v2 hclsyntax 토크나이저는 << 형식에서도 닫는 라벨 앞 공백을 허용
	// (업스트림 스캐너가 라벨 줄을 TrimSpace 후 비교 — 실제 Terraform 동작과 동일).
	// 종전 수제 파서는 << 형식에서 들여쓰기된 라벨을 무시했으나 이는 비표준 동작이었음.
	src := `locals {
  doc = <<EOT
body
  EOT
EOT
}
# 뒤따르는 주석
`
	r := parseHCL(t, src)

	if len(r.strings) != 1 {
		t.Fatalf("expected 1 string, got %d: %+v", len(r.strings), r.strings)
	}
	h := r.strings[0]
	// 들여쓰기된 "  EOT" (4번째 줄)에서 heredoc 이 닫히므로 본문은 "body" 뿐
	if h.Text != "body" {
		t.Errorf("heredoc 본문 = %q, want %q", h.Text, "body")
	}
	if h.EndLine != 4 {
		t.Errorf("heredoc EndLine = %d, want 4", h.EndLine)
	}
	if len(r.comments) != 1 || r.comments[0].Line != 7 {
		t.Errorf("heredoc 이후 주석 줄 번호 불일치: %+v", r.comments)
	}
}

func TestHCLParser_LineNumbers(t *testing.T) {
	src := `# 첫 줄 주석
/* 블록
   주석 */
locals {
  a = "문자열" # 다섯째 줄 주석
}
// 일곱째 줄 주석
`
	r := parseHCL(t, src)

	if len(r.comments) != 4 {
		t.Fatalf("expected 4 comments, got %d: %+v", len(r.comments), r.comments)
	}
	wantLines := []struct{ line, endLine int }{
		{1, 1}, {2, 3}, {5, 5}, {7, 7},
	}
	for i, w := range wantLines {
		if r.comments[i].Line != w.line || r.comments[i].EndLine != w.endLine {
			t.Errorf("comments[%d] 줄 번호 = %d-%d, want %d-%d (%q)",
				i, r.comments[i].Line, r.comments[i].EndLine, w.line, w.endLine, r.comments[i].Text)
		}
	}
	if len(r.strings) != 1 || r.strings[0].Line != 5 {
		t.Errorf("문자열 줄 번호 불일치: %+v", r.strings)
	}
}

func TestHCLParser_NoTrailingNewline(t *testing.T) {
	// 파일이 개행 없이 끝나는 경우
	src := `x = 1 # 마지막 주석`
	r := parseHCL(t, src)

	if len(r.comments) != 1 || r.comments[0].Text != "마지막 주석" {
		t.Errorf("개행 없는 마지막 주석 불일치: %+v", r.comments)
	}
}

func TestHCLParser_Testdata(t *testing.T) {
	src := loadTestdata(t, "sample.tf")
	r := parseHCL(t, src)

	assertHasComment(t, r, "웹 서버 인스턴스 정의")
	assertHasComment(t, r, "초기화 스크립트")
	assertHasComment(t, r, "블록 주석")

	// heredoc 본문과 문자열은 주석으로 추출되지 않아야 함
	for _, c := range r.comments {
		if strings.Contains(c.Text, "this hash inside heredoc") {
			t.Errorf("heredoc 본문이 주석으로 추출됨: %+v", c)
		}
	}

	// HCL 에는 import 컨텍스트가 없으므로 KindImport 0개
	if len(r.imports) != 0 {
		t.Errorf("HCL 에서 KindImport 가 생성되지 않아야 함, 실제: %d개", len(r.imports))
	}
}
