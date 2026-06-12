[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

Gitコミットメッセージとソースコードのポリシーを自動的に検査するCLIツールです。
[lefthook](https://github.com/evilmartians/lefthook) / husky などのGitフックマネージャーと一緒に使用します。

## 機能

| 検査項目 | 説明 |
|---|---|
| **コメント言語** | 指定された言語（韓国語/英語/日本語/中国語）でコメントが書かれているか検査 |
| **許可単語辞書** | 技術用語・固有名詞を許可単語として登録し、誤検出を防止 |
| **Co-authored-by** | AI共著者トレーラーのブロック（メール許可リスト対応） |
| **Unicode空白** | NBSP、EM SPACE、ZWSP、BiDi制御文字などの非標準空白をブロック |
| **紛らわしい文字** | ASCII文字に似たUnicode文字をブロック（例：キリル文字のA vs ラテン文字のA） |
| **ファイルUnicode検査** | ソース/マークダウンファイル内の不可視・紛らわしいUnicode文字を検出 |
| **不正なUTF-8** | 無効なバイトシーケンスをブロック |
| **絵文字禁止** | コミットメッセージやコメントでの絵文字使用をブロック（オプション） |
| **バイナリファイルポリシー** | 拡張子別 block / allow / lfs ポリシー（画像は既定で許可、git LFS 検証対応） |
| **エンコーディング検査** | UTF-8以外のファイルのコミットをブロック（chardetベース） |
| **データファイルlint** | YAML、JSON（JSON5/JSONC対応）、XML構文検査 |
| **EditorConfig** | .editorconfigルールへの準拠を検査 |
| **Conventional Commits** | コミットメッセージ形式の強制（オプション） |
| **append-onlyパス** | 指定パスでのファイル削除・内容変更・中間挿入を禁止（DBマイグレーション等） |
| **キャッシュ/ビルドディレクトリ** | `node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` などへのコミットをブロック（親インジケータ検証） |
| **clean コマンド** | キャッシュ/ビルドディレクトリ内の未追跡ファイルを整理（追跡ファイルは保護） |
| **リポジトリ分析** | 開発言語の検出とlint設定の欠落警告 |
| **自動修正（fix）** | Unicode/エンコーディング違反をgit履歴で一括修正 |
| **設定マイグレーション** | 旧バージョンの設定ファイルを自動検出し、最新スキーマに変換 |
| **進捗表示** | bubbletea TUIスピナー（TTY検出、非TTY時テキストフォールバック） |

## インストール

### Homebrew (macOS / Linux)

```bash
brew install zcube/tap/commit-checker
```

### go install

```bash
go install github.com/zcube/commit-checker@latest
```

Go 1.22以上が必要です。`commit-checker version` で確認してください。

### バイナリダウンロード

[GitHub Releases](https://github.com/zcube/commit-checker/releases)からプラットフォームに合ったファイルをダウンロードします。

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

# staged diff の検査
docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff

# コミットメッセージの検査
docker run --rm -v "$(pwd):/repo" -w /repo \
  ghcr.io/zcube/commit-checker msg /repo/.git/COMMIT_EDITMSG
```

## Gitフック連携（lefthook）

### 1. lefthookのインストール

```bash
# macOS
brew install lefthook

# npm
npm install --save-dev lefthook

# go install
go install github.com/evilmartians/lefthook@latest
```

### 2. commit-checkerのインストール

```bash
go install github.com/zcube/commit-checker@latest
```

### 3. lefthook.yml の作成

プロジェクトルートに `lefthook.yml` を作成します:

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

### 4. フックのインストール

```bash
lefthook install
```

以降、`git commit` 時に自動的に検査が実行されます。

### 5. 既存ファイルの全体検査（初期導入時）

commit-checker を既存のリポジトリに導入した場合、フックインストール以前のコミットのファイルは検査されません。
導入時に一度全ファイルを検査するには `run` コマンドを使用します:

```bash
commit-checker run
```

`git ls-files` で追跡されているすべてのファイルを、staged かどうかに関係なく検査します。
違反項目を自動的に修正するには `fix` コマンドを併用します:

```bash
# 修正内容のプレビュー
commit-checker fix --dry-run

# 実際に修正を適用
commit-checker fix
```

### husky（Node.jsプロジェクト）

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

## 設定

プロジェクトルートに `.commit-checker.yml` を作成します。
`commit-checker init` でデフォルト設定ファイルを自動生成できます。
VS Code を使用すると `.commit-checker.schema.json` スキーマによる自動補完が利用できます。

```yaml
# yaml-language-server: $schema=./.commit-checker.schema.json

comment_language:
  enabled: true
  required_language: japanese # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff            # diff | full
  no_emoji: false             # true でコメントの絵文字を禁止
  extensions:
    - .go
    - .ts
    - .py

  # 許可単語: 言語検査で無視する英語単語のリスト
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
  # default_policy: block       # block | allow | lfs (既定: block)
  # rules:                      # 拡張子別のポリシー規則
  #   - extensions: [.psd, .ai]
  #     policy: lfs              # PSD などは LFS 追跡時のみ許可
  #   - extensions: [.mp4, .mov]
  #     policy: lfs
  # ignore_files:
  #   - "**/*.png"

lint:
  enabled: true
  yaml:
    enabled: true
    # comment_filter: true    # ファイル内の skip-lint コメントで検査を除外可能
  json:
    enabled: true
    # allow_json5: true       # JSON5 のコメント/末尾カンマを許可
    # comment_filter: true    # .json を JSONC モードで検査（コメント除去後 strict JSON）
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # ファイル内の不可視Unicode文字を検査
  # no_ambiguous_chars: true   # ファイル内のASCII紛らわしいUnicode文字を検査

editorconfig:
  enabled: true
  # ignore_files:
  #   - "vendor/**"

commit_message:
  # enabled: true  # false ですべてのコミットメッセージ検査を無効化
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false             # true でコミットメッセージの絵文字を禁止
  locale: ja
  conventional_commit:
    enabled: false
  language_check:
    enabled: false
    required_language: japanese

append_only:
  enabled: false
  # paths:
  #   - "migrations/**"
  #   - "db/migrations/**"

cache_dir:
  enabled: true                # 既定で有効
  # ignore_dirs:
  #   - vendor                 # vendor ディレクトリを意図的にコミットする Go プロジェクト等
```

設定ファイルがない場合はデフォルト値が適用されます。

### バイナリファイルポリシー

拡張子別に3種類のポリシーを指定できます:

| ポリシー | 動作 |
|---|---|
| `block` | ブロック（既定） |
| `allow` | 許可 |
| `lfs` | git LFS で追跡されている場合のみ許可（`.gitattributes` の `filter=lfs` を確認） |

内蔵画像拡張子（`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.ico`, `.tiff`,
`.tif`, `.heic`, `.heif`, `.avif`）は、別途規則がなければ**自動的に `allow`** になります。

```yaml
binary_file:
  enabled: true
  default_policy: block          # マッチしないバイナリ: 既定で block
  rules:
    # 画像を LFS に強制したい場合:
    - extensions: [.png, .jpg, .jpeg, .gif, .webp]
      policy: lfs
    # PSD/AI のようなデザイン原本:
    - extensions: [.psd, .ai, .sketch]
      policy: lfs
    # 動画:
    - extensions: [.mp4, .mov, .webm]
      policy: lfs
  ignore_files:
    - "assets/icons/**"          # ポリシー検査自体をスキップ
```

優先順位: `rules` のマッチ > 内蔵画像（`allow`）> `default_policy`（未指定なら `block`）。

### データファイルlint

YAML / JSON / XML ファイルの構文を検査します。
`.jsonc` 拡張子のファイルは、設定に関係なく常に JSON5 モード（`//` コメント、末尾カンマ許可）で検査します。

```yaml
lint:
  enabled: true
  yaml:
    enabled: true
    comment_filter: true     # ファイル内の skip-lint コメントに対応
  json:
    enabled: true
    # allow_json5: true      # JSON5 のコメント/末尾カンマを許可
    comment_filter: true     # .json ファイルを JSONC モードで検査
  xml:
    enabled: true
```

- `json.comment_filter: true` — `.json` ファイルから `//`, `/* */` コメントを除去した後、strict JSON として検査します（末尾カンマは不許可）。
- `yaml.comment_filter: true` — ファイル内に `# commit-checker: skip-lint` コメントがあると、そのファイルの検査を無効化します。

### append-only パス

DBマイグレーションファイルなど、一度コミットされた内容を変更してはいけないパスを指定します。
違反時はエラーのみ発生し、データは保持されます。

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
    - "db/migrations/**"
  # filename_order: none   # 既定は numeric。順序検査を無効にするには none を指定
```

許可される変更:
- 新規ファイルの追加（既存ファイルより後の名前のみ許可、`filename_order: none` で無効化可能）
- 既存ファイル末尾への内容追加

ブロックされる変更:
- ファイルの削除
- 既存行の変更・削除
- ファイル中間への内容挿入
- 既存ファイルより前または同名の新規ファイル追加（`filename_order: none` の場合は許可）

ファイル名の順序は自然数ソート基準で `9 < 10` として扱われます（既定）。

### ビルド成果物・キャッシュディレクトリ検査

`node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` などのビルド成果物またはキャッシュディレクトリが
gitにコミットまたはステージされるのをブロックします。

**親ディレクトリのインジケータに基づく検証**で誤検出を減らします:

| ディレクトリ | インジケータ |
|---|---|
| `node_modules` | 親に `package.json` / lockfile |
| `dist` | 親に `package.json` / `go.mod` / `Cargo.toml` |
| `build` | 親に `package.json` / `Cargo.toml` / `build.gradle` / `pubspec.yaml` / `CMakeLists.txt`、または自身に `CMakeCache.txt` |
| `target` | 親に `Cargo.toml` / `pom.xml` / `build.sbt` |
| `vendor` | 親に `go.mod` / `Cargo.toml` / `Gemfile` など |
| `__pycache__` | 親に `.py` ファイル |
| `.venv` など | 自身に `pyvenv.cfg`（名前は不問） |

対応ディレクトリ: `node_modules`, `dist`, `out`, `build`, `target`, `vendor`,
`.gradle`, `.next`, `.nuxt`, `.output`, `.svelte-kit`, `.yarn`, `.bun`,
`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.turbo`,
`.parcel-cache`, `.venv`（+pyvenv 仮想環境）, `.tox`, `.nox`, `.embuild`, `.dart_tool`。

```yaml
cache_dir:
  enabled: true               # 既定で有効
  ignore_dirs:                # 意図的にコミットするディレクトリ
    - vendor                  # 例: Go vendor ディレクトリ
```

#### clean コマンド

キャッシュ/ビルドディレクトリ内の未追跡ファイルを整理します。**git 追跡ファイルは絶対に削除しません**
（`git ls-files --others` ベース）。

```bash
# 検出項目のみ表示 (dry-run)
commit-checker clean

# 未追跡ファイルを実際に削除
commit-checker clean --yes
```

### 許可単語辞書

技術用語や固有名詞を言語検査から除外できます:

```yaml
comment_language:
  # インラインリスト
  allowed_words:
    - TypeScript
    - JavaScript
    - API
    - URL

  # ローカルファイル（1行に1単語、# コメント対応）
  allowed_words_file: .commit-checker-words.txt

  # URL（同じ形式、HTTP/HTTPS）
  allowed_words_url: https://example.com/allowed-words.txt

  # URLキャッシュ（オプション）
  allowed_words_cache:
    enabled: true
    ttl: 24h                  # キャッシュの有効期間
    # dir: ~/.cache/commit-checker  # キャッシュディレクトリ（既定値）
```

3つのソース（インライン、ファイル、URL）はマージされて適用されます。

### ファイル別言語ルール

i18n/locale ファイルなどの例外パスを指定できます:

```yaml
comment_language:
  required_language: japanese
  file_languages:
    - pattern: "locales/**"
      language: any
    - pattern: "i18n/**"
      language: english
    - pattern: "locale/ja/**"
      language: ja
```

### ソース内ディレクティブ

ファイルまたは区間単位で言語ルールを上書きします:

```go
// commit-checker:ignore
// This English comment is intentional (next comment only)

// commit-checker:file-lang=english  <- ファイル全体に適用

// commit-checker:disable:lang=english
// This block is intentionally in English
// commit-checker:enable
```

対応ディレクティブ:

| ディレクティブ | 説明 |
|---|---|
| `commit-checker:ignore` | 次のコメント1つだけ検査をスキップ |
| `commit-checker:disable` | この行から検査を無効化 |
| `commit-checker:disable:lang=<L>` | 無効化しこの区間は言語Lで検査 |
| `commit-checker:enable` | 検査を再有効化 |
| `commit-checker:lang=<L>` | この行から必要な言語をLに切り替え |
| `commit-checker:file-lang=<L>` | ファイル全体の必要な言語をLに設定 |

`<L>` の値: `korean` `english` `japanese` `chinese` `any`（または `ko` `en` `ja` `zh`）

## コマンド

```
commit-checker init          デフォルト設定ファイル（.commit-checker.yml）の生成
commit-checker diff          staged diff のコメント/エンコーディング/lint/バイナリ/Unicode 検査
commit-checker run           追跡中の全ファイルのポリシー準拠検査
commit-checker msg <file>    コミットメッセージファイルの検査
commit-checker fix           git履歴の自動修正（dry-run対応）
commit-checker migrate       設定ファイルを最新スキーマに移行
commit-checker analyze       リポジトリ分析（言語検出、lint設定の確認）
commit-checker clean         キャッシュ/ビルドディレクトリの未追跡ファイル整理
commit-checker version       バージョン情報の出力
```

### diff コマンド（CI向け from..to 比較）

`git diff` と互換の引数形式をそのまま受け付けます。引数がない場合は従来どおり
ステージ済みの変更（HEAD ↔ index）を検査します。

```bash
commit-checker diff                      # 既定: ステージ済み (pre-commit)
commit-checker diff --staged             # 明示的 (--cached と同義)
commit-checker diff HEAD                 # HEAD ↔ working tree（未コミット全体）
commit-checker diff origin/main          # origin/main ↔ working tree
commit-checker diff A B                  # A ↔ B
commit-checker diff A..B                 # A ↔ B (range 表記)
commit-checker diff A...B                # merge-base(A,B) ↔ B
```

CIの例（GitHub Actions、GitLab CI など）:

```yaml
# GitHub Actions の PR 検査
- run: commit-checker diff ${{ github.event.pull_request.base.sha }}..HEAD

# GitLab CI の MR 検査
- commit-checker diff ${CI_MERGE_REQUEST_DIFF_BASE_SHA}..HEAD
```

### init コマンド

```bash
# デフォルト設定ファイルの生成（システムロケールを自動検出）
commit-checker init

# 特定のロケールで生成
commit-checker init --lang en

# 既存ファイルの上書き
commit-checker init --force
```

### run コマンド

```bash
# 追跡中の全ファイルを検査（staged 状態に関係なく）
commit-checker run
```

`diff` と異なり、ステージ状態に関係なく `git ls-files` で追跡されているすべてのファイルを検査します。

### fix コマンド

```bash
# 修正内容のプレビュー
commit-checker fix --dry-run

# 直近5コミットの修正
commit-checker fix --range HEAD~5..HEAD

# 自分のコミットのみ修正
commit-checker fix --mine --dry-run
```

### migrate コマンド

```bash
# 設定ファイルのスキーマバージョンを検出し最新に移行
commit-checker migrate

# 変更内容のプレビュー（ファイルは変更しない）
commit-checker migrate --dry-run
```

旧バージョンの設定ファイル（例: `no_coauthor` → `no_ai_coauthor`）を自動的に最新スキーマへ変換します。
コメントと書式は保持されます。

### analyze コマンド

```bash
# 現在のリポジトリを分析
commit-checker analyze
```

開発言語を検出し、その言語向けのlint設定ファイル（`.golangci.yml`, `.eslintrc.*`, `pyproject.toml` など）が
ない場合は警告します。`.editorconfig`, `.gitattributes`, `.gitignore` の有無も確認します。

## 対応言語

| 言語 | 拡張子 |
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

## i18n対応

CLI出力は以下の言語に対応しています：

- 韓国語 (ko) - デフォルト
- English (en)
- 日本語 (ja)
- 中文 (zh)

環境変数 `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` または設定ファイルの `locale` 値で選択します。

## ライセンス

MIT
