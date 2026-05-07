[한국어](./README.md) | [English](./README.en.md) | [日本語](./README.ja.md) | [中文](./README.zh.md)

# commit-checker

自动检查Git提交消息和源代码策略的CLI工具。
与 [lefthook](https://github.com/evilmartians/lefthook) / husky 等Git钩子管理器配合使用。

## 功能

| 检查项 | 说明 |
|---|---|
| **注释语言** | 检查注释是否使用指定语言（韩语/英语/日语/中文）编写 |
| **允许词典** | 注册技术术语和专有名词以防止误报 |
| **Co-authored-by** | 阻止AI共同作者尾部标记（支持邮箱白名单） |
| **Unicode空格** | 阻止不可见/非标准Unicode空白字符（NBSP、ZWSP、BiDi等） |
| **易混淆字符** | 阻止与ASCII字符相似的Unicode字符（如西里尔字母A vs 拉丁字母A） |
| **文件Unicode检查** | 检测源代码/Markdown文件中的不可见和易混淆Unicode字符 |
| **无效UTF-8** | 阻止无效的字节序列 |
| **表情符号禁止** | 阻止在提交消息和注释中使用表情符号（可选） |
| **二进制文件策略** | 按扩展名 block / allow / lfs 策略（图片默认允许，支持 git LFS 验证） |
| **编码检查** | 阻止提交非UTF-8编码的文件（基于chardet） |
| **数据文件lint** | YAML、JSON（支持JSON5）、XML语法验证 |
| **EditorConfig** | 验证文件是否符合.editorconfig规则 |
| **约定式提交** | 强制执行提交消息格式（可选） |
| **append-only路径** | 禁止在指定路径中删除文件、修改内容或中间插入（如DB迁移文件） |
| **缓存/构建目录** | 阻止 `node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` 等的提交（基于父目录指示器验证） |
| **clean 命令** | 清理缓存/构建目录中的未追踪文件（追踪文件保留） |
| **仓库分析** | 检测开发语言并警告缺失的lint配置 |
| **自动修复（fix）** | 在git历史中批量修复unicode/编码违规 |
| **配置迁移** | 自动检测旧版配置文件并迁移到最新架构 |
| **进度指示器** | bubbletea TUI旋转器（TTY感知，非TTY时纯文本回退） |

## 安装

### Homebrew (macOS / Linux)

```bash
brew install zcube/tap/commit-checker
```

### go install

```bash
go install github.com/zcube/commit-checker@latest
```

需要Go 1.22+。使用 `commit-checker version` 验证安装。

### 二进制下载

从 [GitHub Releases](https://github.com/zcube/commit-checker/releases) 下载：

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

## Git钩子集成（lefthook）

### 1. 安装lefthook

```bash
brew install lefthook                        # macOS
npm install --save-dev lefthook              # npm
go install github.com/evilmartians/lefthook@latest  # go
```

### 2. 创建 lefthook.yml

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

### 3. 安装钩子

```bash
lefthook install
```

之后每次 `git commit` 时会自动执行检查。

## 配置

在项目根目录创建 `.commit-checker.yml`。
运行 `commit-checker init` 可自动生成默认配置文件。

```yaml
comment_language:
  enabled: true
  required_language: chinese   # korean | english | japanese | chinese | any
  min_length: 5
  no_emoji: false              # true 禁止注释中的表情符号

  # 允许词: 语言检查中忽略的英语单词
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
  # no_invisible_chars: true   # 检测文件中的不可见Unicode字符
  # no_ambiguous_chars: true   # 检测文件中与ASCII易混淆的Unicode字符

commit_message:
  # enabled: true  # false 禁用所有提交消息检查
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false              # true 禁止提交消息中的表情符号
  locale: zh

binary_file:
  enabled: true
  # default_policy: block        # block | allow | lfs (默认: block)
  # rules:                       # 按扩展名的策略 (首个匹配规则生效)
  #   - extensions: [.psd, .ai]
  #     policy: lfs
  # 内置图片扩展名 (.png .jpg .jpeg .gif .webp .bmp .ico .tiff .tif .heic .heif .avif)
  # 在没有规则匹配时自动应用 allow。

append_only:                    # 可选 — DB 迁移等
  enabled: false
  # paths:
  #   - "migrations/**"
  # filename_order: numeric (默认)。设为 "none" 可禁用顺序检查。

cache_dir:                      # 阻止 node_modules, dist, build, target 等目录的提交
  enabled: true
  # ignore_dirs:
  #   - vendor                   # Go vendor 等有意提交的目录
```

### 二进制文件策略

按扩展名指定三种策略:

| 策略 | 行为 |
|---|---|
| `block` | 拒绝（非图片默认） |
| `allow` | 允许 |
| `lfs` | 仅在 git LFS 追踪时允许 |

内置图片扩展名在没有规则匹配时自动应用 `allow`。
优先级: `rules` > 内置图片 (allow) > `default_policy` > `block`。

### append-only 路径

为 DB 迁移等"一旦提交不可修改"的路径指定规则。

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
  # filename_order: numeric (默认): 新文件名按自然数排序必须在现有文件之后 ("9 < 10")
  # 设为 "none" 可禁用。
```

### 构建产物·缓存目录

通过父目录指示器（`go.mod`, `package.json`, `Cargo.toml` 等）验证
`node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` 等。

```yaml
cache_dir:
  enabled: true
  ignore_dirs:
    - vendor
```

#### clean 命令

清理缓存/构建目录中的未追踪文件。**git 追踪的文件绝不会被删除**。

```bash
commit-checker clean         # 仅显示发现项 (dry-run)
commit-checker clean --yes   # 实际删除未追踪文件
```

### 源码内指令

| 指令 | 说明 |
|---|---|
| `commit-checker:ignore` | 仅跳过下一个注释的检查 |
| `commit-checker:disable` | 从此行开始禁用检查 |
| `commit-checker:disable:lang=<L>` | 禁用并在此区间使用语言L检查 |
| `commit-checker:enable` | 重新启用检查 |
| `commit-checker:lang=<L>` | 从此行开始切换所需语言 |
| `commit-checker:file-lang=<L>` | 设置整个文件的所需语言 |

## 命令

```
commit-checker init          生成默认配置文件
commit-checker diff          检查暂存的diff
commit-checker run           检查所有已跟踪文件
commit-checker msg <file>    检查提交消息
commit-checker fix           自动修复git历史（支持 --dry-run）
commit-checker migrate       将配置文件迁移到最新架构
commit-checker analyze       仓库分析
commit-checker clean         清理缓存/构建目录的未追踪文件
commit-checker version       版本信息
```

## 支持的语言

| 语言 | 扩展名 |
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

## i18n支持

CLI输出支持以下语言：

- 韩语 (ko) - 默认
- English (en)
- 日本語 (ja)
- 中文 (zh)

通过环境变量 `COMMIT_CHECKER_LANG` 或配置文件的 `locale` 字段选择。

## 许可证

MIT
