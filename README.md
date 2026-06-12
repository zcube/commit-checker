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
| **바이너리 파일 정책** | 확장자별 block / allow / lfs 정책 (이미지 기본 허가, git LFS 검증 지원) |
| **인코딩 검사** | UTF-8이 아닌 파일 커밋 차단 (chardet 기반) |
| **데이터 파일 린트** | YAML, JSON (JSON5/JSONC 지원), XML 구문 검사 |
| **EditorConfig** | .editorconfig 규칙 준수 여부 검사 |
| **Conventional Commits** | 커밋 메시지 형식 강제 (선택적) |
| **append-only 경로** | 지정 경로에서 파일 삭제·내용 수정·중간 삽입 차단 (DB 마이그레이션 등) |
| **빌드 산출물·캐시 디렉터리** | node_modules, dist, build, target, __pycache__, .venv 등의 커밋 차단 (부모 인디케이터 검증 기반) |
| **clean 명령** | 미추적 캐시/빌드 파일 정리 (git 추적 파일은 보존) |
| **리포지터리 분석** | 개발 언어 감지 및 린트 설정 누락 경고 |
| **자동 수정 (fix)** | 유니코드/인코딩 위반 사항을 git history에서 일괄 수정 |
| **설정 마이그레이션** | 구 버전 설정 파일을 자동 감지하여 최신 스키마로 변환 |
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
    commit-checker:
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

### 선택 훅 (필요한 것만 추가)

아래 블록들은 서로 독립적이며 필요한 것만 `lefthook.yml` 에 덧붙이면 됩니다.

#### 자동 수정 반영 (fix)

`fix` 는 수정한 파일을 스스로 `git add` 까지 수행하므로 포매팅(이모지 제거·NBSP 정리 등)이 검사 전에 반영됩니다. 기본 설정의 `pre-commit` 블록을 아래처럼 바꿉니다 (lefthook 은 커맨드를 이름순으로 실행하므로 `auto-fix` 가 `commit-checker` 보다 먼저 실행됩니다):

```yaml
pre-commit:
  commands:
    auto-fix:
      run: commit-checker fix
      stage_fixed: true
    commit-checker:
      run: commit-checker diff
```

#### merge 우회 방지 (pre-merge-commit)

merge 커밋은 pre-commit 훅을 타지 않으므로, feature 브랜치의 위반이 merge 로 유입될 수 있습니다. 이를 막으려면 pre-merge-commit 에도 검사를 등록합니다:

```yaml
pre-merge-commit:
  commands:
    commit-checker:
      run: commit-checker diff
```

#### 커밋 메시지 정책 힌트 (prepare-commit-msg)

커밋 메시지 에디터 하단에 활성 정책 힌트를 `#` 주석으로 표시합니다 (-m/merge/squash/amend 시 무동작, `#` 줄은 커밋 시 git 이 제거). 반드시 `{0}` 을 사용하세요 — `{1}` `{2}` `{3}` 은 인자가 없을 때 리터럴로 남아 깨집니다 (lefthook 2.1.9 실측):

```yaml
prepare-commit-msg:
  commands:
    policy-hint:
      run: commit-checker prepare-msg {0}
```

#### push 전 커밋 메시지 검사 (pre-push)

```yaml
pre-push:
  commands:
    check-commits:
      run: commit-checker push
```

### 5. 기존 파일 전체 검사 (초기 도입 시)

commit-checker 를 기존 리포지터리에 도입하면 훅 설치 이전 커밋의 파일은 검사되지 않습니다.
도입 시점에 한 번 전체 파일을 검사하려면 `run` 커맨드를 사용합니다:

```bash
commit-checker run
```

`git ls-files` 로 추적되는 모든 파일을 staged 여부와 관계없이 검사합니다.
위반 항목을 자동으로 수정하려면 `fix` 커맨드를 함께 사용합니다:

```bash
# 수정 내용 미리 보기
commit-checker fix --dry-run

# 실제 수정 적용
commit-checker fix
```

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

### Git 2.54+ 설정 기반 훅 (훅 매니저 없이)

Git 2.54부터는 lefthook 같은 훅 매니저 없이 git 설정만으로 commit-checker를 연동할 수 있습니다.

```bash
# 기본: 스테이지된 변경 검사 (pre-commit)
git config set hook.commit-checker-diff.command "commit-checker diff"
git config set --append hook.commit-checker-diff.event pre-commit

# 기본: 커밋 메시지 검사 (commit-msg) — 메시지 파일 경로는 git이 자동 전달
git config set hook.commit-checker-msg.command "commit-checker msg"
git config set --append hook.commit-checker-msg.event commit-msg

# ── 이하 선택: 필요한 것만 등록 ──

# 선택: push 전 커밋 메시지 검사 (pre-push)
git config set hook.commit-checker-push.command "commit-checker push"
git config set --append hook.commit-checker-push.event pre-push

# 선택: merge 커밋 검사 (pre-merge-commit) — merge 커밋은 pre-commit 훅을 타지 않음
git config set hook.commit-checker-merge.command "commit-checker diff"
git config set --append hook.commit-checker-merge.event pre-merge-commit

# 선택: 커밋 메시지 에디터에 정책 힌트 표시 (prepare-commit-msg) — 인자는 git이 자동 전달
git config set hook.commit-checker-prepare.command "commit-checker prepare-msg"
git config set --append hook.commit-checker-prepare.event prepare-commit-msg
```

- `--global` 을 붙이면 모든 리포지터리에 일괄 적용됩니다 (개인 전역 정책에 유용).
- 등록 확인: `git hook list pre-commit`
- 같은 이벤트의 훅 여러 개는 설정 순서대로 실행되고, 기존 `.git/hooks/` 스크립트(lefthook 등)는 마지막에 실행되므로 공존할 수 있습니다.
- 주의: `.git/config` 는 커밋되지 않으므로 팀 전체 강제에는 lefthook 같은 매니저가 여전히 적합합니다. 설정 기반 훅은 개인 설정·전역 정책에 알맞습니다.

### 그 밖의 훅 연동

#### git am 워크플로

`git am` 으로 패치 메일을 적용하는 워크플로에서도 동일한 정책을 적용할 수 있습니다:

```bash
# 패치의 커밋 메시지 검사 — 메시지 파일 경로는 git이 자동 전달
git config set hook.commit-checker-am-msg.command "commit-checker msg"
git config set --append hook.commit-checker-am-msg.event applypatch-msg

# 적용된 패치 내용 검사 — pre-applypatch 시점에는 패치가 staged 상태
git config set hook.commit-checker-am-diff.command "commit-checker diff"
git config set --append hook.commit-checker-am-diff.event pre-applypatch
```

#### 서버 측 강제 (update 훅)

서버(bare 리포지터리)의 `update` 훅은 ref 당 `<refname> <old> <new>` 인자를 받습니다.
`push --range` 를 사용하면 클라이언트에 훅을 설치하지 않아도 서버에서 커밋 메시지 정책을 강제할 수 있습니다:

```bash
#!/bin/sh
# hooks/update — 인자: <refname> <old> <new>
exec commit-checker push --range "$2..$3"
```

신규 브랜치(old 가 모두 0)는 경고를 출력하고 검사를 건너뜁니다.

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
    - .tf

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
  # default_policy: block       # block | allow | lfs (기본 block)
  # rules:                      # 확장자별 정책 규칙
  #   - extensions: [.psd, .ai]
  #     policy: lfs              # PSD 등은 LFS 추적 시에만 허용
  #   - extensions: [.mp4, .mov]
  #     policy: lfs
  # ignore_files:
  #   - "**/*.png"

lint:
  enabled: true
  yaml:
    enabled: true
    # comment_filter: true    # 파일 내 skip-lint 주석으로 검사 제외 허용
  json:
    enabled: true
    # allow_json5: true       # JSON5 주석/trailing comma 허용
    # comment_filter: true    # .json을 JSONC 모드로 검사 (주석 제거 후 strict JSON)
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

append_only:
  enabled: false
  # paths:
  #   - "migrations/**"
  #   - "db/migrations/**"

# protected_paths:
#   enabled: true
#   paths:
#     - "legacy/**"            # 매칭 경로의 모든 staged 변경(추가·수정·삭제) 차단

cache_dir:
  enabled: true                # 기본 활성화
  # ignore_dirs:
  #   - vendor                 # vendor 디렉터리를 의도적으로 커밋하는 Go 프로젝트 등

# guide:
#   enabled: false             # 위반 시 개선 가이드 출력 비활성화 (기본 활성)
```

설정 파일이 없으면 기본값이 적용됩니다.

### 바이너리 파일 정책

확장자별로 세 가지 정책을 지정할 수 있습니다:

| 정책 | 동작 |
|---|---|
| `block` | 차단 (기본) |
| `allow` | 허가 |
| `lfs` | git LFS 로 추적되는 경우만 허가 (`.gitattributes` 의 `filter=lfs` 확인) |

내장 이미지 확장자(`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.ico`, `.tiff`,
`.tif`, `.heic`, `.heif`, `.avif`)는 별도 규칙이 없으면 **자동으로 `allow`** 입니다.

```yaml
binary_file:
  enabled: true
  default_policy: block          # 매칭되지 않은 바이너리: 기본 block
  rules:
    # 이미지를 LFS 로 강제하고 싶을 때:
    - extensions: [.png, .jpg, .jpeg, .gif, .webp]
      policy: lfs
    # PSD/AI 같은 디자인 원본:
    - extensions: [.psd, .ai, .sketch]
      policy: lfs
    # 동영상:
    - extensions: [.mp4, .mov, .webm]
      policy: lfs
  ignore_files:
    - "assets/icons/**"          # 정책 검사 자체를 건너뜀
```

우선순위: `rules` 매칭 > 내장 이미지(`allow`) > `default_policy` (없으면 `block`).

### 데이터 파일 린트

YAML / JSON / XML 파일의 구문을 검사합니다.
`.jsonc` 확장자 파일은 설정과 무관하게 항상 JSON5 모드(`//` 주석, trailing comma 허용)로 검사합니다.

```yaml
lint:
  enabled: true
  yaml:
    enabled: true
    comment_filter: true     # 파일 내 skip-lint 주석 지원
  json:
    enabled: true
    # allow_json5: true      # JSON5 주석/trailing comma 허용
    comment_filter: true     # .json 파일을 JSONC 모드로 검사
  xml:
    enabled: true
```

- `json.comment_filter: true` — `.json` 파일에서 `//`, `/* */` 주석을 제거한 뒤 strict JSON으로 검사합니다 (trailing comma 불허).
- `yaml.comment_filter: true` — 파일 안에 `# commit-checker: skip-lint` 주석이 있으면 해당 파일의 검사를 비활성화합니다.

### append-only 경로

DB 마이그레이션 파일 등 한 번 커밋된 내용을 변경해서는 안 되는 경로를 지정합니다.
위반 시 에러만 발생하며 데이터는 보존됩니다.

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
    - "db/migrations/**"
  # filename_order: none   # 기본값은 numeric. 순서 검사를 끄려면 none 지정
```

허용되는 변경:
- 새 파일 추가 (기존 파일보다 뒤 이름만 허용, `filename_order: none` 으로 비활성화 가능)
- 기존 파일 끝에 내용 추가

차단되는 변경:
- 파일 삭제
- 기존 줄 수정·삭제
- 파일 중간에 내용 삽입
- 기존 파일보다 앞이나 같은 이름의 새 파일 추가 (`filename_order: none` 시 허용)

파일 이름 순서는 자연수 정렬 기준으로 `9 < 10` 으로 처리합니다 (기본값).

### protected_paths (보호 경로)

glob 패턴에 매칭되는 경로의 모든 staged 변경(추가·수정·삭제)을 차단합니다.
append_only 가 파일 끝에 내용 추가를 허용하는 것과 달리, protected_paths 는 어떤 변경도 허용하지 않는 완전 동결 정책입니다.

```yaml
protected_paths:
  enabled: true
  paths:
    - "legacy/**"
```

| 검사 | 허용되는 변경 |
|---|---|
| `append_only` | 새 파일 추가, 기존 파일 끝에 내용 추가 |
| `protected_paths` | 없음 (완전 동결) |

`exceptions.global_ignore` 에 매칭되는 파일은 검사에서 제외됩니다.

### 빌드 산출물·캐시 디렉터리 검사

`node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` 등의 빌드 산출물 또는 캐시 디렉터리가
git에 커밋되거나 스테이지되는 것을 차단합니다.

**부모 디렉터리 인디케이터 기반 검증**으로 false positive를 줄입니다:

| 디렉터리 | 인디케이터 |
|---|---|
| `node_modules` | 부모에 `package.json` / lockfile |
| `dist` | 부모에 `package.json` / `go.mod` / `Cargo.toml` |
| `build` | 부모에 `package.json` / `Cargo.toml` / `build.gradle` / `pubspec.yaml` / `CMakeLists.txt` 또는 자체에 `CMakeCache.txt` |
| `target` | 부모에 `Cargo.toml` / `pom.xml` / `build.sbt` |
| `vendor` | 부모에 `go.mod` / `Cargo.toml` / `Gemfile` 등 |
| `__pycache__` | 부모에 `.py` 파일 |
| `.venv` 등 | 자체에 `pyvenv.cfg` (이름 무관) |

지원 디렉터리: `node_modules`, `dist`, `out`, `build`, `target`, `vendor`,
`.gradle`, `.next`, `.nuxt`, `.output`, `.svelte-kit`, `.yarn`, `.bun`,
`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.turbo`,
`.parcel-cache`, `.venv` (+pyvenv 가상환경), `.tox`, `.nox`, `.embuild`, `.dart_tool`.

```yaml
cache_dir:
  enabled: true               # 기본 활성화
  ignore_dirs:                # 의도적으로 커밋하는 디렉터리
    - vendor                  # 예: Go vendor 디렉터리
```

#### clean 명령

캐시/빌드 디렉터리 안의 미추적 파일을 정리합니다. **git 추적 파일은 절대 삭제하지 않습니다**
(`git ls-files --others` 기반).

```bash
# 발견 항목만 표시 (dry-run)
commit-checker clean

# 미추적 파일 실제 삭제
commit-checker clean --yes
```

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

### 개선 가이드

검사 실패 시 위반 목록과 요약 줄 뒤에 **실패한 카테고리별 수정 가이드**를 카테고리당 1회 출력합니다.
가이드는 AI 에이전트가 출력을 읽고 바로 실행할 수 있는 명령형 수정 지시입니다:

```
config/bad.json:3: invalid character '}' looking for beginning of value

개선 가이드 (AI 에이전트: 아래 지시에 따라 위 위반 사항을 수정하세요):
  [lint] 보고된 파일:라인의 구문 오류를 수정하세요. 주석이 필요한 JSON 파일은 .jsonc 확장자 사용 또는 lint.json.comment_filter: true 설정을 고려하세요.
```

기본 활성화되어 있으며 설정으로 끌 수 있습니다:

```yaml
guide:
  enabled: false
```

전역 플래그 `--no-guide` 를 사용하면 설정과 무관하게 비활성화됩니다.
`--format json` 출력에는 `"guides": {"<category>": "<text>"}` 필드로 포함되며, 비활성화 시 필드가 생략됩니다.

## 커맨드

```
commit-checker init          기본 설정 파일(.commit-checker.yml) 생성
commit-checker diff          staged diff의 주석/인코딩/린트/바이너리/유니코드 검사
commit-checker run           추적된 전체 파일의 정책 준수 검사
commit-checker msg <file>    커밋 메시지 파일 검사
commit-checker prepare-msg   prepare-commit-msg 훅용: 에디터에 활성 정책 힌트 표시
commit-checker fix           git history 자동 수정 (dry-run 지원)
commit-checker migrate       설정 파일을 최신 스키마로 마이그레이션
commit-checker analyze       리포지터리 분석 (언어 감지, 린트 설정 확인)
commit-checker clean         캐시/빌드 디렉터리 미추적 파일 정리
commit-checker version       버전 정보 출력
```

### diff 커맨드 (CI 친화적 from..to 비교)

`git diff` 와 호환되는 인자 형식을 그대로 받습니다. 인자가 없으면 기존처럼
스테이지된 변경(HEAD ↔ index)을 검사합니다.

```bash
commit-checker diff                      # 기본: 스테이지 (pre-commit)
commit-checker diff --staged             # 명시적 (--cached 동의어)
commit-checker diff HEAD                 # HEAD ↔ working tree (uncommitted 전체)
commit-checker diff origin/main          # origin/main ↔ working tree
commit-checker diff A B                  # A ↔ B
commit-checker diff A..B                 # A ↔ B (range 표기)
commit-checker diff A...B                # merge-base(A,B) ↔ B
```

`--only` 플래그로 지정한 검사만 실행할 수 있습니다 (`run` 도 동일하게 지원).
설정에서 `enabled: false` 인 검사도 `--only` 로 지정하면 강제 실행됩니다.

```bash
commit-checker diff --only comment_language   # 주석 언어만 검사
commit-checker diff --only lint,encoding      # 복수 카테고리 지정
```

- run·diff 공통 카테고리: `binary` `encoding` `unicode` `lint` `editorconfig` `comment_language` `cache_dir`
- diff 전용 카테고리: `custom_rules` `append_only` `protected_paths`

CI 예시 (GitHub Actions, GitLab CI 등):

```yaml
# GitHub Actions 의 PR 검사
- run: commit-checker diff ${{ github.event.pull_request.base.sha }}..HEAD

# GitLab CI MR 검사
- commit-checker diff ${CI_MERGE_REQUEST_DIFF_BASE_SHA}..HEAD
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

# 특정 검사만 실행 (--only)
commit-checker run --only lint
```

`diff` 와 달리 스테이지 여부에 관계없이 `git ls-files` 로 추적된 모든 파일을 검사합니다.

### prepare-msg 커맨드

`prepare-commit-msg` 훅용 커맨드입니다. 커밋 메시지 에디터에 활성화된 정책 힌트를 `#` 주석으로 표시합니다.
git이 커밋 시 `#` 줄을 제거하므로 힌트는 메시지에 남지 않습니다.
`-m`/merge/squash/amend 커밋에서는 아무 동작도 하지 않습니다.

```bash
# git이 전달하는 인자를 그대로 받습니다: <파일> [source] [sha]
commit-checker prepare-msg .git/COMMIT_EDITMSG
```

### fix 커맨드

```bash
# 수정 내용 미리 보기
commit-checker fix --dry-run

# 마지막 5개 커밋 수정
commit-checker fix --range HEAD~5..HEAD

# 내 커밋만 수정
commit-checker fix --mine --dry-run
```

### migrate 커맨드

```bash
# 설정 파일 스키마 버전 감지 및 최신으로 마이그레이션
commit-checker migrate

# 변경 사항 미리 보기 (파일 수정 없음)
commit-checker migrate --dry-run
```

구 버전 설정 파일(예: `no_coauthor` → `no_ai_coauthor`)을 자동으로 최신 스키마로 변환합니다.
주석과 서식이 보존됩니다.

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
| HCL (Terraform) | `.hcl` `.tf` `.tfvars` |

## i18n 지원

CLI 출력은 다음 언어를 지원합니다:

- 한국어 (ko) - 기본
- English (en)
- 日本語 (ja)
- 中文 (zh)

환경 변수 `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` 또는 설정 파일의 `locale` 값으로 선택합니다.

## 라이선스

MIT
