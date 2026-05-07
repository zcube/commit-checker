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
| **データファイルlint** | YAML、JSON（JSON5対応）、XML構文検査 |
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
```

### Docker

```bash
docker pull ghcr.io/zcube/commit-checker:latest

docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff
```

## Gitフック連携（lefthook）

### 1. lefthookのインストール

```bash
brew install lefthook                        # macOS
npm install --save-dev lefthook              # npm
go install github.com/evilmartians/lefthook@latest  # go
```

### 2. lefthook.yml の作成

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

### 3. フックのインストール

```bash
lefthook install
```

以降、`git commit` 時に自動的に検査が実行されます。

## 設定

プロジェクトルートに `.commit-checker.yml` を作成します。
`commit-checker init` でデフォルト設定ファイルを自動生成できます。

```yaml
comment_language:
  enabled: true
  required_language: japanese  # korean | english | japanese | chinese | any
  min_length: 5
  no_emoji: false              # true でコメントの絵文字を禁止

  # 許可単語: 言語検査で無視する英語単語
  allowed_words:
    - TypeScript
    - API
  # allowed_words_file: .commit-checker-words.txt
  # allowed_words_url: https://example.com/allowed-words.txt
  # allowed_words_cache:
  #   enabled: true
  #   ttl: 24h

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # ファイル内の不可視Unicode文字を検査
  # no_ambiguous_chars: true   # ファイル内のASCII紛らわしいUnicode文字を検査

commit_message:
  # enabled: true  # false ですべてのコミットメッセージ検査を無効化
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false              # true でコミットメッセージの絵文字を禁止
  locale: ja

binary_file:
  enabled: true
  # default_policy: block        # block | allow | lfs (既定: block)
  # rules:                       # 拡張子別ポリシー (最初にマッチした規則が適用)
  #   - extensions: [.psd, .ai]
  #     policy: lfs
  # 内蔵画像拡張子 (.png .jpg .jpeg .gif .webp .bmp .ico .tiff .tif .heic .heif .avif)
  # は規則に該当しない場合は自動で allow が適用されます。

append_only:                    # オプション — DB マイグレーション等
  enabled: false
  # paths:
  #   - "migrations/**"
  # filename_order: numeric が既定。順序検査を無効にするには "none"。

cache_dir:                      # node_modules, dist, build, target などへのコミットをブロック
  enabled: true
  # ignore_dirs:
  #   - vendor                   # Go vendor 等を意図的にコミットする場合
```

### バイナリファイルポリシー

拡張子別に 3 種のポリシーを指定できます:

| ポリシー | 動作 |
|---|---|
| `block` | 拒否（画像以外の既定） |
| `allow` | 許可 |
| `lfs` | git LFS で追跡されている場合のみ許可 |

内蔵画像拡張子は規則に該当しない場合 `allow` が適用されます。
優先順位: `rules` > 内蔵画像 (allow) > `default_policy` > `block`。

### append-only パス

DB マイグレーションファイルなど、一度コミットされた内容を変更してはいけないパスを指定します。

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
  # filename_order: numeric (既定) で新ファイル名は既存の最大より後に来る必要があります
  # ("9 < 10" の自然数順)。"none" で無効化。
```

### ビルド成果物・キャッシュディレクトリ

`node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` 等を
親ディレクトリのインジケータ (`go.mod`, `package.json`, `Cargo.toml` 等) で検証します。

```yaml
cache_dir:
  enabled: true
  ignore_dirs:
    - vendor
```

#### clean コマンド

キャッシュ/ビルドディレクトリ内の未追跡ファイルを削除します。**git 追跡ファイルは絶対に削除しません**。

```bash
commit-checker clean         # 検出のみ表示 (dry-run)
commit-checker clean --yes   # 未追跡ファイルを実際に削除
```

### ソース内ディレクティブ

| ディレクティブ | 説明 |
|---|---|
| `commit-checker:ignore` | 次のコメント1つだけ検査をスキップ |
| `commit-checker:disable` | この行から検査を無効化 |
| `commit-checker:disable:lang=<L>` | 無効化しこの区間は言語Lで検査 |
| `commit-checker:enable` | 検査を再有効化 |
| `commit-checker:lang=<L>` | この行から必要な言語を切り替え |
| `commit-checker:file-lang=<L>` | ファイル全体の必要な言語を設定 |

## コマンド

```
commit-checker init          デフォルト設定ファイルの生成
commit-checker diff          ステージされたdiffの検査
commit-checker run           追跡中の全ファイルを検査
commit-checker msg <file>    コミットメッセージの検査
commit-checker fix           git履歴の自動修正（--dry-run対応）
commit-checker migrate       設定ファイルを最新スキーマに移行
commit-checker analyze       リポジトリ分析
commit-checker clean         キャッシュ/ビルドディレクトリの未追跡ファイル整理
commit-checker version       バージョン情報
```

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

環境変数 `COMMIT_CHECKER_LANG` または設定ファイルの `locale` で選択できます。

## ライセンス

MIT
