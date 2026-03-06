[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

Gitコミットメッセージとソースコードのポリシーを自動的に検査するCLIツールです。
[lefthook](https://github.com/evilmartians/lefthook) / husky などのGitフックマネージャーと一緒に使用します。

## 機能

| 検査項目 | 説明 |
|---|---|
| **コメント言語** | 指定された言語（韓国語/英語/日本語/中国語）でコメントが書かれているか検査 |
| **Co-authored-by** | AI共著者トレーラーのブロック（メール許可リスト対応） |
| **Unicode空白** | NBSP、EM SPACE、ZWSP、BiDi制御文字などの非標準空白をブロック |
| **紛らわしい文字** | ASCII文字に似たUnicode文字をブロック（例：キリル文字のA vs ラテン文字のA） |
| **不正なUTF-8** | 無効なバイトシーケンスをブロック |
| **絵文字禁止** | コミットメッセージやコメントでの絵文字使用をブロック（オプション） |
| **バイナリファイル検出** | コンパイル済み実行ファイルなどのバイナリファイルのコミットをブロック |
| **エンコーディング検査** | UTF-8以外のファイルのコミットをブロック（chardetベース） |
| **データファイルlint** | YAML、JSON（JSON5対応）、XML構文検査 |
| **EditorConfig** | .editorconfigルールへの準拠を検査 |
| **Conventional Commits** | コミットメッセージ形式の強制（オプション） |
| **リポジトリ分析** | 開発言語の検出とlint設定の欠落警告 |
| **自動修正（fix）** | Unicode/エンコーディング違反をgit履歴で一括修正 |

## インストール

### go install（推奨）

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

```yaml
comment_language:
  enabled: true
  required_language: japanese  # korean | english | japanese | chinese | any
  min_length: 5
  no_emoji: false              # true でコメントの絵文字を禁止

commit_message:
  # enabled: true  # false ですべてのコミットメッセージ検査を無効化
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false              # true でコミットメッセージの絵文字を禁止
  locale: ja
```

### ソース内ディレクティブ

| ディレクティブ | 説明 |
|---|---|
| `commit-checker:ignore` | 次のコメント1つだけ検査をスキップ |
| `commit-checker:disable` | この行から検査を無効化 |
| `commit-checker:enable` | 検査を再有効化 |
| `commit-checker:lang=<L>` | この行から必要な言語を切り替え |
| `commit-checker:file-lang=<L>` | ファイル全体の必要な言語を設定 |

## コマンド

```
commit-checker diff          ステージされたdiffの検査
commit-checker msg <file>    コミットメッセージの検査
commit-checker fix           git履歴の自動修正（--dry-run対応）
commit-checker analyze       リポジトリ分析
commit-checker version       バージョン情報
```

## i18n対応

CLI出力は以下の言語に対応しています：

- 韓国語 (ko) - デフォルト
- English (en)
- 日本語 (ja)
- 中文 (zh)

環境変数 `COMMIT_CHECKER_LANG` または設定ファイルの `locale` で選択できます。

## ライセンス

MIT
