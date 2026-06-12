[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

A CLI tool that automatically enforces policies on Git commit messages and source code.
Works with [lefthook](https://github.com/evilmartians/lefthook), husky, or any Git hook manager.

## Features

| Check | Description |
|---|---|
| **Comment language** | Verify comments are written in the required language (Korean/English/Japanese/Chinese) |
| **Allowed words** | Register technical terms and proper nouns to prevent false positives |
| **Co-authored-by** | Block AI co-author trailers (with email allow-list support) |
| **Unicode spaces** | Block invisible/non-standard Unicode space characters (NBSP, EM SPACE, ZWSP, BiDi, etc.) |
| **Ambiguous chars** | Block Unicode characters that look like ASCII (e.g., Cyrillic A vs Latin A) |
| **File Unicode check** | Detect invisible/ambiguous Unicode characters in source and markdown files |
| **Invalid UTF-8** | Block invalid byte sequences |
| **Emoji ban** | Block emojis in commit messages and comments (optional) |
| **Binary file policy** | Per-extension block / allow / lfs policy (images allowed by default, git-LFS verification) |
| **Encoding check** | Block non-UTF-8 encoded files (chardet-based) |
| **Data file lint** | YAML, JSON (with JSON5/JSONC support), XML syntax validation |
| **EditorConfig** | Validate files against .editorconfig rules |
| **Conventional Commits** | Enforce commit message format (optional) |
| **Append-only paths** | Block file deletion, content modification, and mid-file insertion (e.g. DB migrations) |
| **Cache / build dirs** | Block commits inside node_modules, dist, build, target, __pycache__, .venv, etc. (parent-indicator validation) |
| **clean command** | Remove untracked files inside cache/build dirs (tracked files preserved) |
| **Repository analysis** | Detect development languages and warn about missing lint configs |
| **Auto-fix** | Batch-fix unicode/encoding violations across git history |
| **Config migration** | Auto-detect old config versions and migrate to the latest schema |
| **Progress indicator** | Bubbletea TUI spinner (TTY-aware, plain text fallback) |

## Installation

### Homebrew (macOS / Linux)

```bash
brew install zcube/tap/commit-checker
```

### go install

```bash
go install github.com/zcube/commit-checker@latest
```

Requires Go 1.22+. Verify with `commit-checker version`.

### Binary download

Download the file for your platform from [GitHub Releases](https://github.com/zcube/commit-checker/releases).

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

# Check staged diff
docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff

# Check commit message
docker run --rm -v "$(pwd):/repo" -w /repo \
  ghcr.io/zcube/commit-checker msg /repo/.git/COMMIT_EDITMSG
```

## Git Hook Integration (lefthook)

### 1. Install lefthook

```bash
# macOS
brew install lefthook

# npm
npm install --save-dev lefthook

# go install
go install github.com/evilmartians/lefthook@latest
```

### 2. Install commit-checker

```bash
go install github.com/zcube/commit-checker@latest
```

### 3. Create lefthook.yml

Create `lefthook.yml` in your project root:

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

### 4. Install hooks

```bash
lefthook install
```

Checks run automatically on every `git commit`.

### 5. Check all existing files (initial adoption)

When commit-checker is adopted in an existing repository, files committed before the
hooks were installed are never checked. To check every file once at adoption time,
use the `run` command:

```bash
commit-checker run
```

It checks all files tracked by `git ls-files`, regardless of staged state.
To fix violations automatically, combine it with the `fix` command:

```bash
# Preview changes
commit-checker fix --dry-run

# Apply the fixes
commit-checker fix
```

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

### Git 2.54+ config-based hooks (no hook manager)

Starting with Git 2.54, you can integrate commit-checker with git configuration alone, without a hook manager like lefthook.

```bash
# Check staged changes (pre-commit)
git config set hook.commit-checker-diff.command "commit-checker diff"
git config set --append hook.commit-checker-diff.event pre-commit

# Check commit messages (commit-msg) — git passes the message file path automatically
git config set hook.commit-checker-msg.command "commit-checker msg"
git config set --append hook.commit-checker-msg.event commit-msg

# (Optional) check commit messages before push (pre-push)
git config set hook.commit-checker-push.command "commit-checker push"
git config set --append hook.commit-checker-push.event pre-push
```

- Add `--global` to apply the hooks to every repository at once (useful for a personal global policy).
- Verify registration: `git hook list pre-commit`
- Multiple hooks for the same event run in configuration order, and existing `.git/hooks/` scripts (e.g. lefthook) run last, so they can coexist.
- Note: `.git/config` is not committed, so a manager like lefthook is still the better fit for team-wide enforcement. Config-based hooks suit personal setups and global policies.

## Configuration

Create `.commit-checker.yml` in your project root.
Run `commit-checker init` to generate a default config file automatically.
Use `.commit-checker.schema.json` for IDE autocompletion in VS Code.

```yaml
# yaml-language-server: $schema=./.commit-checker.schema.json

comment_language:
  enabled: true
  required_language: english  # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff            # diff | full
  no_emoji: false             # true to ban emojis in comments
  extensions:
    - .go
    - .ts
    - .py
    - .tf

  # Allowed words: English terms to ignore during language detection
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
  # default_policy: block       # block | allow | lfs (default: block)
  # rules:                      # per-extension policy rules
  #   - extensions: [.psd, .ai]
  #     policy: lfs              # allow PSD etc. only when tracked by LFS
  #   - extensions: [.mp4, .mov]
  #     policy: lfs
  # ignore_files:
  #   - "**/*.png"

lint:
  enabled: true
  yaml:
    enabled: true
    # comment_filter: true    # opt out per file via in-file skip-lint comment
  json:
    enabled: true
    # allow_json5: true       # allow JSON5 comments/trailing commas
    # comment_filter: true    # check .json in JSONC mode (strict JSON after stripping comments)
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # Detect invisible Unicode chars in file content
  # no_ambiguous_chars: true   # Detect ASCII-confusable Unicode chars in file content

editorconfig:
  enabled: true
  # ignore_files:
  #   - "vendor/**"

commit_message:
  # enabled: true  # false to disable all commit message checks
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false             # true to ban emojis in commit messages
  locale: en
  conventional_commit:
    enabled: false
  language_check:
    enabled: false
    required_language: english

append_only:
  enabled: false
  # paths:
  #   - "migrations/**"
  #   - "db/migrations/**"

cache_dir:
  enabled: true                # enabled by default
  # ignore_dirs:
  #   - vendor                 # e.g. Go projects that intentionally commit vendor

# guide:
#   enabled: false             # disable the remediation guide output on violations (enabled by default)
```

Defaults apply when the config file is absent.

### Binary file policy

Three policies can be assigned per extension:

| Policy | Behaviour |
|---|---|
| `block` | Reject (default) |
| `allow` | Accept |
| `lfs` | Accept only when tracked by git LFS (checks `filter=lfs` in `.gitattributes`) |

Built-in image extensions (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.ico`, `.tiff`,
`.tif`, `.heic`, `.heif`, `.avif`) default to **`allow`** when no rule matches.

```yaml
binary_file:
  enabled: true
  default_policy: block          # unmatched binaries: block by default
  rules:
    # To force LFS for images:
    - extensions: [.png, .jpg, .jpeg, .gif, .webp]
      policy: lfs
    # Design source files like PSD/AI:
    - extensions: [.psd, .ai, .sketch]
      policy: lfs
    # Videos:
    - extensions: [.mp4, .mov, .webm]
      policy: lfs
  ignore_files:
    - "assets/icons/**"          # skip the policy check entirely
```

Resolution order: `rules` match > built-in image (`allow`) > `default_policy` (or `block`).

### Data file lint

Validates the syntax of YAML / JSON / XML files.
Files with the `.jsonc` extension are always checked in JSON5 mode
(`//` comments and trailing commas allowed), regardless of configuration.

```yaml
lint:
  enabled: true
  yaml:
    enabled: true
    comment_filter: true     # support in-file skip-lint comment
  json:
    enabled: true
    # allow_json5: true      # allow JSON5 comments/trailing commas
    comment_filter: true     # check .json files in JSONC mode
  xml:
    enabled: true
```

- `json.comment_filter: true` — strips `//` and `/* */` comments from `.json` files, then validates as strict JSON (trailing commas are not allowed).
- `yaml.comment_filter: true` — a `# commit-checker: skip-lint` comment anywhere in a file disables linting for that file.

### Append-only paths

Specify paths whose committed content must never change, such as DB migration files.
Violations only raise errors; data is preserved.

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
    - "db/migrations/**"
  # filename_order: none   # defaults to numeric; set to none to disable order check
```

Allowed changes:
- Adding new files (only names sorting after existing files; disable with `filename_order: none`)
- Appending content at the end of existing files

Blocked changes:
- Deleting files
- Modifying or deleting existing lines
- Inserting content in the middle of a file
- Adding new files that sort before existing files or share a name (allowed with `filename_order: none`)

File name order uses natural numeric sorting, so `9 < 10` (default).

### Build artifact / cache directories

Block build artifact or cache directories such as `node_modules`, `dist`, `build`,
`target`, `__pycache__`, `.venv` from being committed or staged in git.

**Parent-indicator validation** reduces false positives:

| Directory | Indicator |
|---|---|
| `node_modules` | parent has `package.json` / lockfile |
| `dist` | parent has `package.json` / `go.mod` / `Cargo.toml` |
| `build` | parent has `package.json` / `Cargo.toml` / `build.gradle` / `pubspec.yaml` / `CMakeLists.txt`, or `CMakeCache.txt` inside |
| `target` | parent has `Cargo.toml` / `pom.xml` / `build.sbt` |
| `vendor` | parent has `go.mod` / `Cargo.toml` / `Gemfile`, etc. |
| `__pycache__` | parent has `.py` files |
| `.venv` etc. | `pyvenv.cfg` inside (any name) |

Supported directories: `node_modules`, `dist`, `out`, `build`, `target`, `vendor`,
`.gradle`, `.next`, `.nuxt`, `.output`, `.svelte-kit`, `.yarn`, `.bun`,
`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.turbo`,
`.parcel-cache`, `.venv` (+pyvenv virtualenvs), `.tox`, `.nox`, `.embuild`, `.dart_tool`.

```yaml
cache_dir:
  enabled: true               # enabled by default
  ignore_dirs:                # directories committed intentionally
    - vendor                  # e.g. Go vendor directory
```

#### clean command

Remove untracked files inside cache/build directories. **Tracked files are
never deleted** (based on `git ls-files --others`).

```bash
# List found items only (dry-run)
commit-checker clean

# Actually delete untracked files
commit-checker clean --yes
```

### Allowed words dictionary

Exclude technical terms and proper nouns from language detection:

```yaml
comment_language:
  # Inline list
  allowed_words:
    - TypeScript
    - JavaScript
    - API
    - URL

  # Local file (one word per line, # comments supported)
  allowed_words_file: .commit-checker-words.txt

  # URL (same format, HTTP/HTTPS)
  allowed_words_url: https://example.com/allowed-words.txt

  # URL caching (optional)
  allowed_words_cache:
    enabled: true
    ttl: 24h                  # Cache TTL
    # dir: ~/.cache/commit-checker  # Cache directory (default)
```

All three sources (inline, file, URL) are merged.

### Per-file language rules

Specify exception paths such as i18n/locale files:

```yaml
comment_language:
  required_language: english
  file_languages:
    - pattern: "locales/**"
      language: any
    - pattern: "i18n/**"
      language: english
    - pattern: "locale/ja/**"
      language: ja
```

### In-source directives

Override language rules per file or per region:

```go
// commit-checker:ignore
// This English comment is intentional (next comment only)

// commit-checker:file-lang=english  <- applies to the whole file

// commit-checker:disable:lang=english
// This block is intentionally in English
// commit-checker:enable
```

Supported directives:

| Directive | Description |
|---|---|
| `commit-checker:ignore` | Skip the next comment only |
| `commit-checker:disable` | Disable checking from this line |
| `commit-checker:disable:lang=<L>` | Disable and use language L for this region |
| `commit-checker:enable` | Re-enable checking |
| `commit-checker:lang=<L>` | Switch required language from this point |
| `commit-checker:file-lang=<L>` | Set required language for the entire file |

`<L>` values: `korean` `english` `japanese` `chinese` `any` (or `ko` `en` `ja` `zh`)

### Remediation guide

When a check fails, a **per-category fix guide** is printed once per failed category, after the violation list and the summary line.
The guides are imperative fix instructions that AI agents can read from the output and act on immediately:

```
config/bad.json:3: invalid character '}' looking for beginning of value

Remediation guide (AI agents: follow the instructions below to fix the violations above):
  [lint] Fix the syntax errors at the reported file:line locations. For JSON files that need comments, consider using the .jsonc extension or setting lint.json.comment_filter: true.
```

Enabled by default; it can be turned off in the config:

```yaml
guide:
  enabled: false
```

The global `--no-guide` flag disables it regardless of the config.
With `--format json`, the guides are included as a `"guides": {"<category>": "<text>"}` field, which is omitted when disabled.

## Commands

```
commit-checker init          Generate default config file (.commit-checker.yml)
commit-checker diff          Check staged diff (comments/encoding/lint/binary/unicode)
commit-checker run           Check all tracked files for policy compliance
commit-checker msg <file>    Check commit message file
commit-checker fix           Auto-fix git history (supports --dry-run)
commit-checker migrate       Migrate config file to the latest schema
commit-checker analyze       Analyze repository (language detection, lint config check)
commit-checker clean         Remove untracked files inside cache/build directories
commit-checker version       Print version info
```

### diff command (CI-friendly `from..to`)

`commit-checker diff` accepts the same positional argument forms as `git diff`.
Without arguments, it checks staged changes (HEAD ↔ index) as before.

```bash
commit-checker diff                      # default: staged (pre-commit)
commit-checker diff --staged             # explicit (alias: --cached)
commit-checker diff HEAD                 # HEAD ↔ working tree (all uncommitted)
commit-checker diff origin/main          # origin/main ↔ working tree
commit-checker diff A B                  # A ↔ B
commit-checker diff A..B                 # A ↔ B (range)
commit-checker diff A...B                # merge-base(A,B) ↔ B
```

Typical CI usage (GitHub Actions, GitLab CI, etc.):

```yaml
# GitHub Actions: check the PR diff
- run: commit-checker diff ${{ github.event.pull_request.base.sha }}..HEAD

# GitLab CI: check the MR diff
- commit-checker diff ${CI_MERGE_REQUEST_DIFF_BASE_SHA}..HEAD
```

### init command

```bash
# Generate the default config file (auto-detects system locale)
commit-checker init

# Generate with a specific locale
commit-checker init --lang en

# Overwrite an existing file
commit-checker init --force
```

### run command

```bash
# Check all tracked files (regardless of staged state)
commit-checker run
```

Unlike `diff`, this checks all files tracked by `git ls-files`, regardless of staged state.

### fix command

```bash
# Preview changes
commit-checker fix --dry-run

# Fix the last 5 commits
commit-checker fix --range HEAD~5..HEAD

# Fix only my commits
commit-checker fix --mine --dry-run
```

### migrate command

```bash
# Detect schema version and migrate to latest
commit-checker migrate

# Preview changes without modifying the file
commit-checker migrate --dry-run
```

Automatically migrates old config files (e.g., `no_coauthor` → `no_ai_coauthor`) to the latest schema.
Comments and formatting are preserved.

### analyze command

```bash
# Analyze the current repository
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
| Dockerfile | `Dockerfile` `Dockerfile.*` `*.dockerfile` |
| Markdown | `.md` `.markdown` |
| HCL (Terraform) | `.hcl` `.tf` `.tfvars` |

## i18n Support

CLI output is available in:

- Korean (ko) - default
- English (en)
- Japanese (ja)
- Chinese (zh)

Set via `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` environment variables, or the `locale` config field.

## License

MIT
