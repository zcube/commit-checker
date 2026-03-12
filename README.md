[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

Git 커밋 메시지와 소스 코드의 정책을 자동으로 검사하는 CLI 도구입니다.
[lefthook](https://github.com/evilmartians/lefthook) / husky 등 Git 훅 매니저와 함께 사용합니다.

## 기능

| 검사 항목 | 설명 |
|---|---|
| **주석 언어** | 지정된 언어(한국어/영어/일본어/중국어)로 작성된 주석인지 검사 |
| **허용 단어 사전** | 기술 용어·고유명사를 허용 단어로 등록하여 오탐 방지 |
| **Co-authored-by** | AI 기여 표시 트레일러 차단 (이메일 허용 목록 지원) |
| **비표준 유니코드 공백** | NBSP, EM SPACE, ZWSP, BiDi 제어문자 등 차단 |
| **모호한 유니코드 문자** | 키릴 А ↔ 라틴 A 등 시각적 혼동 문자 차단 |
| **파일 유니코드 검사** | 소스/마크다운 파일 내용에서 비가시·모호한 유니코드 문자 검사 |
| **잘못된 UTF-8** | 잘못된 바이트 시퀀스 차단 |
| **이모지 금지** | 커밋 메시지 및 주석에서 이모지 사용 차단 (선택적) |
| **바이너리 파일 감지** | 컴파일된 실행파일 등 바이너리 파일 커밋 차단 |
| **인코딩 검사** | UTF-8이 아닌 파일 커밋 차단 (chardet 기반) |
| **데이터 파일 린트** | YAML, JSON (JSON5 지원), XML 구문 검사 |
| **EditorConfig** | .editorconfig 규칙 준수 여부 검사 |
| **Conventional Commits** | 커밋 메시지 형식 강제 (선택적) |
| **리포지터리 분석** | 개발 언어 감지 및 린트 설정 누락 경고 |
| **자동 수정 (fix)** | 유니코드/인코딩 위반 사항을 git history에서 일괄 수정 |
| **진행 표시기** | bubbletea 기반 TUI 스피너 (TTY 감지, 비TTY 시 텍스트 폴백) |

## 설치

### Homebrew (macOS / Linux)

```bash
brew install zcube/tap/commit-checker
```

### go install

```bash
go install github.com/zcube/commit-checker@latest
```

Go 1.22 이상이 필요합니다. 설치 후 `commit-checker version` 으로 확인합니다.

### 바이너리 직접 다운로드

[GitHub Releases](https://github.com/zcube/commit-checker/releases) 페이지에서 플랫폼에 맞는 파일을 다운로드합니다.

```bash
# Linux (amd64)
curl -L https://github.com/zcube/commit-checker/releases/latest/download/commit-checker_linux_amd64.tar.gz | tar xz
sudo mv commit-checker /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/zcube/commit-checker/releases/latest/download/commit-checker_darwin_arm64.tar.gz | tar xz
sudo mv commit-checker /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/zcube/commit-checker/releases/latest/download/commit-checker_darwin_amd64.tar.gz | tar xz
sudo mv commit-checker /usr/local/bin/
```

### Docker

```bash
docker pull ghcr.io/zcube/commit-checker:latest

# staged diff 검사
docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff

# 커밋 메시지 검사
docker run --rm -v "$(pwd):/repo" -w /repo \
  ghcr.io/zcube/commit-checker msg /repo/.git/COMMIT_EDITMSG
```

## Git 훅 연동 (lefthook)

### 1. lefthook 설치

```bash
# macOS
brew install lefthook

# npm
npm install --save-dev lefthook

# go install
go install github.com/evilmartians/lefthook@latest
```

### 2. commit-checker 설치

```bash
go install github.com/zcube/commit-checker@latest
```

### 3. lefthook.yml 추가

프로젝트 루트에 `lefthook.yml` 을 생성합니다:

```yaml
pre-commit:
  commands:
    comment-language:
      run: commit-checker diff

commit-msg:
  commands:
    message-policy:
      run: commit-checker msg {1}
```

### 4. 훅 설치

```bash
lefthook install
```

이후 `git commit` 시 자동으로 검사가 실행됩니다.

### husky (Node.js 프로젝트)

```bash
npx husky init
```

`.husky/pre-commit`:
```bash
#!/bin/sh
commit-checker diff
```

`.husky/commit-msg`:
```bash
#!/bin/sh
commit-checker msg "$1"
```

## 설정

프로젝트 루트에 `.commit-checker.yml` 을 생성합니다.
`commit-checker init` 으로 기본 설정 파일을 자동 생성할 수 있습니다.
VS Code를 사용하면 `.commit-checker.schema.json` 스키마로 자동완성이 제공됩니다.

```yaml
# yaml-language-server: $schema=./.commit-checker.schema.json

comment_language:
  enabled: true
  required_language: korean   # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff            # diff | full
  no_emoji: false             # true이면 주석에서 이모지 금지
  extensions:
    - .go
    - .ts
    - .py

  # 허용 단어: 언어 검사에서 무시할 영어 단어 목록
  allowed_words:
    - TypeScript
    - JavaScript
    - API
  # allowed_words_file: .commit-checker-words.txt
  # allowed_words_url: https://example.com/allowed-words.txt
  # allowed_words_cache:
  #   enabled: true
  #   ttl: 24h

binary_file:
  enabled: true
  # ignore_files:
  #   - "**/*.png"

lint:
  enabled: true
  yaml:
    enabled: true
  json:
    enabled: true
    # allow_json5: true       # JSON5 주석/trailing comma 허용
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # 파일 내 비가시 유니코드 문자 검사
  # no_ambiguous_chars: true   # 파일 내 ASCII 혼동 유니코드 문자 검사

editorconfig:
  enabled: true
  # ignore_files:
  #   - "vendor/**"

commit_message:
  # enabled: true  # false이면 모든 커밋 메시지 검사 비활성화
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false             # true이면 커밋 메시지에서 이모지 금지
  locale: ko
  conventional_commit:
    enabled: false
  language_check:
    enabled: false
    required_language: korean
```

설정 파일이 없으면 기본값이 적용됩니다.

### 허용 단어 사전

기술 용어나 고유명사를 언어 검사에서 제외할 수 있습니다:

```yaml
comment_language:
  # 인라인 목록
  allowed_words:
    - TypeScript
    - JavaScript
    - API
    - URL

  # 로컬 파일 (한 줄에 하나, # 주석 지원)
  allowed_words_file: .commit-checker-words.txt

  # URL (형식 동일, HTTP/HTTPS)
  allowed_words_url: https://example.com/allowed-words.txt

  # URL 캐싱 (선택적)
  allowed_words_cache:
    enabled: true
    ttl: 24h                  # 캐시 유효 기간
    # dir: ~/.cache/commit-checker  # 캐시 디렉터리 (기본값)
```

세 가지 소스(인라인, 파일, URL)는 병합되어 적용됩니다.

### 파일별 언어 규칙

i18n/locale 파일 등 예외 경로를 지정할 수 있습니다:

```yaml
comment_language:
  required_language: korean
  file_languages:
    - pattern: "locales/**"
      language: any
    - pattern: "i18n/**"
      language: english
    - pattern: "locale/ja/**"
      language: ja
```

### 소스 내 디렉티브

파일 또는 구간 단위로 언어 규칙을 재정의합니다:

```go
// commit-checker:ignore
// This English comment is intentional (next comment only)

// commit-checker:file-lang=english  <- 파일 전체에 적용

// commit-checker:disable:lang=english
// This block is intentionally in English
// commit-checker:enable
```

지원 디렉티브:

| 디렉티브 | 설명 |
|---|---|
| `commit-checker:ignore` | 바로 다음 주석 1개만 검사 제외 |
| `commit-checker:disable` | 이 줄부터 검사 비활성화 |
| `commit-checker:disable:lang=<L>` | 비활성화하고 해당 구간은 언어 L로 검사 |
| `commit-checker:enable` | 검사 재활성화 |
| `commit-checker:lang=<L>` | 이 줄부터 필요 언어를 L로 전환 |
| `commit-checker:file-lang=<L>` | 파일 전체의 필요 언어를 L로 설정 |

`<L>` 값: `korean` `english` `japanese` `chinese` `any` (또는 `ko` `en` `ja` `zh`)

## 커맨드

```
commit-checker init          기본 설정 파일(.commit-checker.yml) 생성
commit-checker diff          staged diff의 주석/인코딩/린트/바이너리/유니코드 검사
commit-checker run           추적된 전체 파일의 정책 준수 검사
commit-checker msg <file>    커밋 메시지 파일 검사
commit-checker fix           git history 자동 수정 (dry-run 지원)
commit-checker analyze       리포지터리 분석 (언어 감지, 린트 설정 확인)
commit-checker version       버전 정보 출력
```

### init 커맨드

```bash
# 기본 설정 파일 생성 (시스템 로케일 자동 감지)
commit-checker init

# 특정 로케일로 생성
commit-checker init --lang en

# 기존 파일 덮어쓰기
commit-checker init --force
```

### run 커맨드

```bash
# 추적된 전체 파일 검사 (staged 상태 무관)
commit-checker run
```

`diff` 와 달리 스테이지 여부에 관계없이 `git ls-files` 로 추적된 모든 파일을 검사합니다.

### fix 커맨드

```bash
# 수정 내용 미리 보기
commit-checker fix --dry-run

# 마지막 5개 커밋 수정
commit-checker fix --range HEAD~5..HEAD

# 내 커밋만 수정
commit-checker fix --mine --dry-run
```

### analyze 커맨드

```bash
# 현재 리포지터리 분석
commit-checker analyze
```

개발 언어를 감지하고, 해당 언어에 대한 린트 설정 파일(`.golangci.yml`, `.eslintrc.*`, `pyproject.toml` 등)이
없으면 경고합니다. `.editorconfig`, `.gitattributes`, `.gitignore` 존재 여부도 확인합니다.

## 지원 언어

| 언어 | 확장자 |
|---|---|
| Go | `.go` |
| TypeScript | `.ts` `.tsx` |
| JavaScript | `.js` `.jsx` `.mjs` `.cjs` |
| Java | `.java` |
| Kotlin | `.kt` `.kts` |
| Python | `.py` |
| C / C++ | `.c` `.h` `.cpp` `.cc` `.hpp` |
| C# | `.cs` |
| Swift | `.swift` |
| Rust | `.rs` |
| Dockerfile | `Dockerfile` `Dockerfile.*` `*.dockerfile` |
| Markdown | `.md` `.markdown` |

## i18n 지원

CLI 출력은 다음 언어를 지원합니다:

- 한국어 (ko) - 기본
- English (en)
- 日本語 (ja)
- 中文 (zh)

환경 변수 `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` 또는 설정 파일의 `locale` 값으로 선택합니다.

## 라이선스

MIT
