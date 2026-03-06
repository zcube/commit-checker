[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

A CLI tool that automatically enforces policies on Git commit messages and source code.
Works with [lefthook](https://github.com/evilmartians/lefthook), husky, or any Git hook manager.

## Features

| Check | Description |
|---|---|
| **Comment language** | Verify comments are written in the required language (Korean/English/Japanese/Chinese) |
| **Co-authored-by** | Block AI co-author trailers (with email allow-list support) |
| **Unicode spaces** | Block invisible/non-standard Unicode space characters (NBSP, ZWSP, BiDi, etc.) |
| **Ambiguous chars** | Block Unicode characters that look like ASCII (e.g., Cyrillic A vs Latin A) |
| **Invalid UTF-8** | Block invalid byte sequences |
| **Emoji ban** | Block emojis in commit messages and comments (optional) |
| **Binary file detection** | Block compiled executables and binary files from being committed |
| **Encoding check** | Block non-UTF-8 encoded files (chardet-based) |
| **Data file lint** | YAML, JSON (with JSON5 support), XML syntax validation |
| **EditorConfig** | Validate files against .editorconfig rules |
| **Conventional Commits** | Enforce commit message format (optional) |
| **Repository analysis** | Detect development languages and warn about missing lint configs |
| **Auto-fix** | Batch-fix unicode/encoding violations across git history |

## Installation

### go install (recommended)

```bash
go install github.com/zcube/commit-checker@latest
```

Requires Go 1.22+. Verify with `commit-checker version`.

### Binary download

Download from [GitHub Releases](https://github.com/zcube/commit-checker/releases):

```bash
# Linux (amd64)
curl -L https://github.com/zcube/commit-checker/releases/latest/download/commit-checker_linux_amd64.tar.gz | tar xz
sudo mv commit-checker /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/zcube/commit-checker/releases/latest/download/commit-checker_darwin_arm64.tar.gz | tar xz
sudo mv commit-checker /usr/local/bin/
```

### Docker

```bash
docker pull ghcr.io/zcube/commit-checker:latest

# Check staged diff
docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff

# Check commit message
docker run --rm -v "$(pwd):/repo" -w /repo \
  ghcr.io/zcube/commit-checker msg /repo/.git/COMMIT_EDITMSG
```

## Git Hook Integration (lefthook)

### 1. Install lefthook

```bash
brew install lefthook        # macOS
npm install --save-dev lefthook  # npm
go install github.com/evilmartians/lefthook@latest  # go
```

### 2. Install commit-checker

```bash
go install github.com/zcube/commit-checker@latest
```

### 3. Create lefthook.yml

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

### 4. Install hooks

```bash
lefthook install
```

Checks run automatically on every `git commit`.

### husky (Node.js projects)

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

## Configuration

Create `.commit-checker.yml` in your project root.
Use `.commit-checker.schema.json` for IDE autocompletion in VS Code.

```yaml
# yaml-language-server: $schema=./.commit-checker.schema.json

comment_language:
  enabled: true
  required_language: english   # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff             # diff | full
  no_emoji: false              # true to ban emojis in comments

binary_file:
  enabled: true

lint:
  enabled: true
  json:
    allow_json5: false         # true to allow JSON5 comments/trailing commas

encoding:
  enabled: true
  require_utf8: true

editorconfig:
  enabled: true

commit_message:
  no_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false              # true to ban emojis in commit messages
  locale: en
```

Defaults apply when the config file is absent.

### Per-file language rules

```yaml
comment_language:
  required_language: english
  file_languages:
    - pattern: "locales/**"
      language: any
    - pattern: "i18n/**"
      language: any
```

### In-source directives

| Directive | Description |
|---|---|
| `commit-checker:ignore` | Skip the next comment only |
| `commit-checker:disable` | Disable checking from this line |
| `commit-checker:disable:lang=<L>` | Disable and use language L for this region |
| `commit-checker:enable` | Re-enable checking |
| `commit-checker:lang=<L>` | Switch required language from this point |
| `commit-checker:file-lang=<L>` | Set required language for the entire file |

`<L>` values: `korean` `english` `japanese` `chinese` `any` (or `ko` `en` `ja` `zh`)

## Commands

```
commit-checker diff          Check staged diff (comments/encoding/lint/binary)
commit-checker msg <file>    Check commit message file
commit-checker fix           Auto-fix git history (supports --dry-run)
commit-checker analyze       Analyze repository (language detection, lint config check)
commit-checker version       Print version info
```

### fix command

```bash
commit-checker fix --dry-run              # Preview changes
commit-checker fix --range HEAD~5..HEAD   # Fix last 5 commits
commit-checker fix --mine --dry-run       # Fix only my commits
```

### analyze command

```bash
commit-checker analyze
```

Detects development languages and warns when lint config files (`.golangci.yml`, `.eslintrc.*`, `pyproject.toml`, etc.)
are missing. Also checks for `.editorconfig`, `.gitattributes`, `.gitignore`.

## Supported Languages

| Language | Extensions |
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

## i18n Support

CLI output is available in:

- Korean (ko) - default
- English (en)
- Japanese (ja)
- Chinese (zh)

Set via `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` environment variables, or the `locale` config field.

## License

MIT
