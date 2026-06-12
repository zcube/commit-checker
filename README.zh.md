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
| **数据文件lint** | YAML、JSON（支持JSON5/JSONC）、XML语法验证 |
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

从 [GitHub Releases](https://github.com/zcube/commit-checker/releases) 下载适合您平台的文件：

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

# 检查 staged diff
docker run --rm -v "$(pwd):/repo" -w /repo ghcr.io/zcube/commit-checker diff

# 检查提交消息
docker run --rm -v "$(pwd):/repo" -w /repo \
  ghcr.io/zcube/commit-checker msg /repo/.git/COMMIT_EDITMSG
```

## Git钩子集成（lefthook）

### 1. 安装lefthook

```bash
# macOS
brew install lefthook

# npm
npm install --save-dev lefthook

# go install
go install github.com/evilmartians/lefthook@latest
```

### 2. 安装commit-checker

```bash
go install github.com/zcube/commit-checker@latest
```

### 3. 创建 lefthook.yml

在项目根目录创建 `lefthook.yml`：

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

### 4. 安装钩子

```bash
lefthook install
```

之后每次 `git commit` 时会自动执行检查。

### 可选钩子（按需添加）

以下各代码块彼此独立，只需将需要的部分追加到 `lefthook.yml` 即可。

#### 自动修复（fix）

`fix` 会自行对修改过的文件执行 `git add` 重新暂存，因此格式修复（去除表情符号、清理 NBSP 等）会在检查前生效。请将基础配置中的 `pre-commit` 块替换为以下内容（lefthook 按名称顺序执行命令，因此 `auto-fix` 会先于 `commit-checker` 运行）：

```yaml
pre-commit:
  commands:
    auto-fix:
      run: commit-checker fix
      stage_fixed: true
    commit-checker:
      run: commit-checker diff
```

#### 防止 merge 绕过（pre-merge-commit）

merge 提交不会触发 pre-commit 钩子，feature 分支的违规可能通过 merge 流入。为防止这种情况，请同时在 pre-merge-commit 上注册检查：

```yaml
pre-merge-commit:
  commands:
    commit-checker:
      run: commit-checker diff
```

#### 提交信息策略提示（prepare-commit-msg）

在提交信息编辑器底部以 `#` 注释显示当前生效的策略提示（-m/merge/squash/amend 时不执行任何操作；`#` 行会在提交时被 git 移除）。请务必使用 `{0}` — `{1}` `{2}` `{3}` 在没有参数时会以字面形式残留并导致钩子损坏（lefthook 2.1.9 实测）：

```yaml
prepare-commit-msg:
  commands:
    policy-hint:
      run: commit-checker prepare-msg {0}
```

#### push 前检查提交信息（pre-push）

```yaml
pre-push:
  commands:
    check-commits:
      run: commit-checker push
```

### 5. 检查现有全部文件（初次引入时）

将 commit-checker 引入现有仓库时，钩子安装之前提交的文件不会被检查。
如需在引入时对全部文件做一次检查，请使用 `run` 命令：

```bash
commit-checker run
```

无论是否暂存，都会检查 `git ls-files` 跟踪的所有文件。
如需自动修复违规项，请配合使用 `fix` 命令：

```bash
# 预览修复内容
commit-checker fix --dry-run

# 实际应用修复
commit-checker fix
```

### husky（Node.js项目）

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

### Git 2.54+ 基于配置的钩子（无需钩子管理器）

从 Git 2.54 开始，无需 lefthook 等钩子管理器，仅通过 git 配置即可集成 commit-checker。

```bash
# 基础：检查暂存的变更（pre-commit）
git config set hook.commit-checker-diff.command "commit-checker diff"
git config set --append hook.commit-checker-diff.event pre-commit

# 基础：检查提交信息（commit-msg）— 信息文件路径由 git 自动传递
git config set hook.commit-checker-msg.command "commit-checker msg"
git config set --append hook.commit-checker-msg.event commit-msg

# ── 以下为可选：按需注册 ──

# 可选：push 前检查提交信息（pre-push）
git config set hook.commit-checker-push.command "commit-checker push"
git config set --append hook.commit-checker-push.event pre-push

# 可选：检查 merge 提交（pre-merge-commit）— merge 提交不会触发 pre-commit 钩子
git config set hook.commit-checker-merge.command "commit-checker diff"
git config set --append hook.commit-checker-merge.event pre-merge-commit

# 可选：在提交信息编辑器中显示策略提示（prepare-commit-msg）— 参数由 git 自动传递
git config set hook.commit-checker-prepare.command "commit-checker prepare-msg"
git config set --append hook.commit-checker-prepare.event prepare-commit-msg
```

- 加上 `--global` 可一次性应用到所有仓库（适合个人全局策略）。包含全局配置文件、按目录策略、按仓库 opt-out 的完整指南请参阅 [全局安装](#全局安装) 一节。
- 确认注册：`git hook list pre-commit`
- 同一事件的多个钩子按配置顺序执行，现有的 `.git/hooks/` 脚本（如 lefthook）最后执行，因此可以共存。
- 注意：`.git/config` 不会被提交，团队级强制仍更适合使用 lefthook 等管理器。基于配置的钩子适合个人配置和全局策略。

### 其他钩子集成

#### git am 工作流

在使用 `git am` 应用补丁邮件的工作流中也可以应用相同的策略：

```bash
# 检查补丁的提交信息 — 信息文件路径由 git 自动传递
git config set hook.commit-checker-am-msg.command "commit-checker msg"
git config set --append hook.commit-checker-am-msg.event applypatch-msg

# 检查已应用的补丁内容 — pre-applypatch 时补丁处于 staged 状态
git config set hook.commit-checker-am-diff.command "commit-checker diff"
git config set --append hook.commit-checker-am-diff.event pre-applypatch
```

#### 服务器端强制（update 钩子）

服务器（bare 仓库）的 `update` 钩子按 ref 接收 `<refname> <old> <new>` 参数。
使用 `push --range`，无需在客户端安装任何钩子即可在服务器端强制执行提交信息策略：

```bash
#!/bin/sh
# hooks/update — 参数: <refname> <old> <new>
exec commit-checker push --range "$2..$3"
```

新分支（old 全为 0）会输出警告并跳过检查。

## 全局安装

只需注册一次钩子和配置，commit-checker 即可在所有仓库中运行。

### 全局钩子 + 全局配置

```bash
# 注册全局钩子（Git 2.54+）
git config set --global hook.commit-checker-diff.command "commit-checker diff"
git config set --global --append hook.commit-checker-diff.event pre-commit
git config set --global hook.commit-checker-msg.command "commit-checker msg"
git config set --global --append hook.commit-checker-msg.event commit-msg
```

全局配置文件按以下顺序查找，使用**第一个存在的文件**：

| 顺序 | 位置 |
|---|---|
| 1 | `$COMMIT_CHECKER_GLOBAL_CONFIG` 环境变量（显式指定，文件不存在时警告并忽略） |
| 2 | `$XDG_CONFIG_HOME/commit-checker/config.yml` |
| 3 | OS 标准配置目录 — Linux `~/.config/commit-checker/config.yml`、macOS `~/Library/Application Support/commit-checker/config.yml`、Windows `%AppData%\commit-checker\config.yml` |
| 4 | `~/.commit-checker.yml`（legacy，向后兼容） |

```yaml
# ~/.config/commit-checker/config.yml — 简单的全局配置示例
comment_language:
  locale: zh
commit_message:
  no_ai_coauthor: true
  locale: zh
```

### 按目录的策略（gitdir include）

支持与 git 的 `[includeIf "gitdir:..."]` 对应的条件 include。
可以在全局配置的同一处，为公司仓库（`~/work/`）和个人仓库管理不同的策略：

```yaml
# ~/.config/commit-checker/config.yml
include:
  - path: ~/.config/commit-checker/base.yml   # 无条件 → 始终包含（通用共享基础）
  - path: ~/.config/commit-checker/work.yml
    gitdir: ~/work/                            # 仅 ~/work/ 下的仓库
comment_language:
  locale: zh
```

- 优先级：正文 > 后面的 include > 前面的 include — include 提供基础值，正文覆盖它。
- `gitdir`：`~` 会展开为主目录，以 `/` 结尾时匹配整个子目录树（与 git 相同）。
- 全局配置和项目配置中均可使用。include 的嵌套会被忽略，远程 preset 中的 include 出于安全考虑也会被忽略。

### 按仓库的控制（override·opt-out·opt-in）

**override** — 仓库中存在 `.commit-checker.yml` 时，全局配置会被完全忽略，
只应用仓库配置（无论团队成员的全局配置如何，都保证执行相同的检查）。
全局配置仅在仓库没有配置文件时使用。
如需部分 override，请在仓库配置中直接声明指向共享基础文件的 [gitdir include](#按目录的策略gitdir-include)。

**opt-out** — 要在特定仓库中关闭所有检查，只需在项目 `.commit-checker.yml` 中添加一行：

```yaml
enabled: false
```

**opt-in 运营** — 将全局钩子的 command 注册为 `commit-checker diff --require-config`，
则只检查存在项目配置文件的仓库。在没有配置文件的仓库中
不执行任何操作并以退出码 0 结束：

```bash
git config set --global hook.commit-checker-diff.command "commit-checker diff --require-config"
git config set --global hook.commit-checker-msg.command "commit-checker msg --require-config"
```

## 配置

在项目根目录创建 `.commit-checker.yml`。
运行 `commit-checker init` 可自动生成默认配置文件。
使用 VS Code 时可通过 `.commit-checker.schema.json` 架构获得自动补全。

```yaml
# yaml-language-server: $schema=./.commit-checker.schema.json

comment_language:
  enabled: true
  required_language: chinese  # korean | english | japanese | chinese | any
  min_length: 5
  check_mode: diff            # diff | full
  no_emoji: false             # true 禁止注释中的表情符号
  extensions:
    - .go
    - .ts
    - .py
    - .tf

  # 允许词: 语言检查中忽略的英语单词列表
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
  # default_policy: block       # block | allow | lfs (默认: block)
  # rules:                      # 按扩展名的策略规则
  #   - extensions: [.psd, .ai]
  #     policy: lfs              # PSD 等仅在 LFS 追踪时允许
  #   - extensions: [.mp4, .mov]
  #     policy: lfs
  # ignore_files:
  #   - "**/*.png"

lint:
  enabled: true
  yaml:
    enabled: true
    # comment_filter: true    # 可通过文件内 skip-lint 注释排除检查
  json:
    enabled: true
    # allow_json5: true       # 允许 JSON5 注释/尾随逗号
    # comment_filter: true    # 以 JSONC 模式检查 .json（去除注释后 strict JSON）
  xml:
    enabled: true

encoding:
  enabled: true
  require_utf8: true
  # no_invisible_chars: true   # 检测文件中的不可见Unicode字符
  # no_ambiguous_chars: true   # 检测文件中与ASCII易混淆的Unicode字符

editorconfig:
  enabled: true
  # ignore_files:
  #   - "vendor/**"

commit_message:
  # enabled: true  # false 禁用所有提交消息检查
  no_ai_coauthor: true
  no_unicode_spaces: true
  no_ambiguous_chars: true
  no_bad_runes: true
  no_emoji: false             # true 禁止提交消息中的表情符号
  locale: zh
  conventional_commit:
    enabled: false
  language_check:
    enabled: false
    required_language: chinese

append_only:
  enabled: false
  # paths:
  #   - "migrations/**"
  #   - "db/migrations/**"

# protected_paths:
#   enabled: true
#   paths:
#     - "legacy/**"            # 阻止匹配路径下的所有 staged 变更（添加/修改/删除）

cache_dir:
  enabled: true                # 默认启用
  # ignore_dirs:
  #   - vendor                 # 有意提交 vendor 目录的 Go 项目等

# guide:
#   enabled: false             # 禁用违规时的改进指南输出（默认启用）
```

没有配置文件时将应用默认值。

### 二进制文件策略

可按扩展名指定三种策略：

| 策略 | 行为 |
|---|---|
| `block` | 拒绝（默认） |
| `allow` | 允许 |
| `lfs` | 仅在 git LFS 追踪时允许（检查 `.gitattributes` 中的 `filter=lfs`） |

内置图片扩展名（`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.ico`, `.tiff`,
`.tif`, `.heic`, `.heif`, `.avif`）在没有单独规则时**自动应用 `allow`**。

```yaml
binary_file:
  enabled: true
  default_policy: block          # 未匹配的二进制文件: 默认 block
  rules:
    # 希望强制图片使用 LFS 时:
    - extensions: [.png, .jpg, .jpeg, .gif, .webp]
      policy: lfs
    # PSD/AI 等设计原始文件:
    - extensions: [.psd, .ai, .sketch]
      policy: lfs
    # 视频:
    - extensions: [.mp4, .mov, .webm]
      policy: lfs
  ignore_files:
    - "assets/icons/**"          # 完全跳过策略检查
```

优先级: `rules` 匹配 > 内置图片（`allow`）> `default_policy`（未指定则为 `block`）。

### 数据文件lint

检查 YAML / JSON / XML 文件的语法。
`.jsonc` 扩展名的文件无论配置如何，始终以 JSON5 模式（允许 `//` 注释、尾随逗号）检查。

```yaml
lint:
  enabled: true
  yaml:
    enabled: true
    comment_filter: true     # 支持文件内 skip-lint 注释
  json:
    enabled: true
    # allow_json5: true      # 允许 JSON5 注释/尾随逗号
    comment_filter: true     # 以 JSONC 模式检查 .json 文件
  xml:
    enabled: true
```

- `json.comment_filter: true` — 去除 `.json` 文件中的 `//`、`/* */` 注释后，按 strict JSON 检查（不允许尾随逗号）。
- `yaml.comment_filter: true` — 文件内存在 `# commit-checker: skip-lint` 注释时，禁用对该文件的检查。

### append-only 路径

为 DB 迁移文件等"一旦提交不可修改"的路径指定规则。
违规时仅报错，数据会被保留。

```yaml
append_only:
  enabled: true
  paths:
    - "migrations/**"
    - "db/migrations/**"
  # filename_order: none   # 默认为 numeric。设为 none 可禁用顺序检查
```

允许的更改：
- 添加新文件（仅允许排在现有文件之后的名称，可用 `filename_order: none` 禁用）
- 在现有文件末尾追加内容

阻止的更改：
- 删除文件
- 修改或删除现有行
- 在文件中间插入内容
- 添加排在现有文件之前或同名的新文件（`filename_order: none` 时允许）

文件名顺序按自然数排序处理，即 `9 < 10`（默认）。

### protected_paths（保护路径）

阻止 glob 模式匹配路径下的所有 staged 变更（添加、修改、删除）。
与允许在文件末尾追加内容的 append_only 不同，protected_paths 是不允许任何变更的完全冻结策略。

```yaml
protected_paths:
  enabled: true
  paths:
    - "legacy/**"
```

| 检查 | 允许的变更 |
|---|---|
| `append_only` | 添加新文件、在现有文件末尾追加内容 |
| `protected_paths` | 无（完全冻结） |

匹配 `exceptions.global_ignore` 的文件会从检查中排除。

### 构建产物·缓存目录检查

阻止将 `node_modules`, `dist`, `build`, `target`, `__pycache__`, `.venv` 等
构建产物或缓存目录提交或暂存到 git。

通过**基于父目录指示器的验证**减少误报：

| 目录 | 指示器 |
|---|---|
| `node_modules` | 父目录中有 `package.json` / lockfile |
| `dist` | 父目录中有 `package.json` / `go.mod` / `Cargo.toml` |
| `build` | 父目录中有 `package.json` / `Cargo.toml` / `build.gradle` / `pubspec.yaml` / `CMakeLists.txt`，或自身包含 `CMakeCache.txt` |
| `target` | 父目录中有 `Cargo.toml` / `pom.xml` / `build.sbt` |
| `vendor` | 父目录中有 `go.mod` / `Cargo.toml` / `Gemfile` 等 |
| `__pycache__` | 父目录中有 `.py` 文件 |
| `.venv` 等 | 自身包含 `pyvenv.cfg`（与名称无关） |

支持的目录: `node_modules`, `dist`, `out`, `build`, `target`, `vendor`,
`.gradle`, `.next`, `.nuxt`, `.output`, `.svelte-kit`, `.yarn`, `.bun`,
`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.turbo`,
`.parcel-cache`, `.venv`（+pyvenv 虚拟环境）, `.tox`, `.nox`, `.embuild`, `.dart_tool`。

```yaml
cache_dir:
  enabled: true               # 默认启用
  ignore_dirs:                # 有意提交的目录
    - vendor                  # 例: Go vendor 目录
```

#### clean 命令

清理缓存/构建目录中的未追踪文件。**git 追踪的文件绝不会被删除**
（基于 `git ls-files --others`）。

```bash
# 仅显示发现项 (dry-run)
commit-checker clean

# 实际删除未追踪文件
commit-checker clean --yes
```

### 允许词典

可将技术术语和专有名词从语言检查中排除：

```yaml
comment_language:
  # 内联列表
  allowed_words:
    - TypeScript
    - JavaScript
    - API
    - URL

  # 本地文件（每行一个单词，支持 # 注释）
  allowed_words_file: .commit-checker-words.txt

  # URL（格式相同，HTTP/HTTPS）
  allowed_words_url: https://example.com/allowed-words.txt

  # URL 缓存（可选）
  allowed_words_cache:
    enabled: true
    ttl: 24h                  # 缓存有效期
    # dir: ~/.cache/commit-checker  # 缓存目录（默认值）
```

三种来源（内联、文件、URL）会合并后生效。

### 按文件的语言规则

可为 i18n/locale 文件等指定例外路径：

```yaml
comment_language:
  required_language: chinese
  file_languages:
    - pattern: "locales/**"
      language: any
    - pattern: "i18n/**"
      language: english
    - pattern: "locale/ja/**"
      language: ja
```

### 源码内指令

以文件或区间为单位覆盖语言规则：

```go
// commit-checker:ignore
// This English comment is intentional (next comment only)

// commit-checker:file-lang=english  <- 应用于整个文件

// commit-checker:disable:lang=english
// This block is intentionally in English
// commit-checker:enable
```

支持的指令：

| 指令 | 说明 |
|---|---|
| `commit-checker:ignore` | 仅跳过下一个注释的检查 |
| `commit-checker:disable` | 从此行开始禁用检查 |
| `commit-checker:disable:lang=<L>` | 禁用并在此区间使用语言L检查 |
| `commit-checker:enable` | 重新启用检查 |
| `commit-checker:lang=<L>` | 从此行开始将所需语言切换为L |
| `commit-checker:file-lang=<L>` | 将整个文件的所需语言设为L |

`<L>` 的取值: `korean` `english` `japanese` `chinese` `any`（或 `ko` `en` `ja` `zh`）

### 改进指南

检查失败时，会在违规列表和摘要行之后，按**失败的类别**各输出一次修复指南。
指南是 AI 代理读取输出后可立即执行的命令式修复指示:

```
config/bad.json:3: invalid character '}' looking for beginning of value

改进指南（AI 代理：请按照以下说明修复上述违规项）：
  [lint] 修复报告的文件:行中的语法错误。需要注释的 JSON 文件请考虑使用 .jsonc 扩展名或设置 lint.json.comment_filter: true。
```

默认启用，可通过配置关闭:

```yaml
guide:
  enabled: false
```

使用全局标志 `--no-guide` 可不受配置影响直接禁用。
`--format json` 的输出中会以 `"guides": {"<category>": "<text>"}` 字段包含指南，禁用时该字段会被省略。

## 命令

```
commit-checker init          生成默认配置文件（.commit-checker.yml）
commit-checker diff          检查 staged diff 的注释/编码/lint/二进制/Unicode
commit-checker run           检查所有已跟踪文件的策略合规性
commit-checker msg <file>    检查提交消息文件
commit-checker prepare-msg   用于 prepare-commit-msg 钩子: 在编辑器中显示生效的策略提示
commit-checker fix           自动修复git历史（支持 dry-run）
commit-checker migrate       将配置文件迁移到最新架构
commit-checker analyze       仓库分析（语言检测、lint配置确认）
commit-checker clean         清理缓存/构建目录的未追踪文件
commit-checker version       输出版本信息
```

### diff 命令（CI 友好的 from..to 比较）

直接接受与 `git diff` 兼容的参数形式。无参数时与以往相同，
检查已暂存的更改（HEAD ↔ index）。

```bash
commit-checker diff                      # 默认: 暂存 (pre-commit)
commit-checker diff --staged             # 显式 (--cached 同义)
commit-checker diff HEAD                 # HEAD ↔ working tree（全部未提交内容）
commit-checker diff origin/main          # origin/main ↔ working tree
commit-checker diff A B                  # A ↔ B
commit-checker diff A..B                 # A ↔ B (range 表示法)
commit-checker diff A...B                # merge-base(A,B) ↔ B
```

使用 `--only` 标志可以只运行指定的检查（`run` 同样支持）。
即使配置中为 `enabled: false` 的检查，通过 `--only` 指定时也会强制执行。

```bash
commit-checker diff --only comment_language   # 仅检查注释语言
commit-checker diff --only lint,encoding      # 指定多个类别
```

- run·diff 共用类别: `binary` `encoding` `unicode` `lint` `editorconfig` `comment_language` `cache_dir`
- diff 专用类别: `custom_rules` `append_only` `protected_paths`

CI 示例（GitHub Actions、GitLab CI 等）：

```yaml
# GitHub Actions 的 PR 检查
- run: commit-checker diff ${{ github.event.pull_request.base.sha }}..HEAD

# GitLab CI 的 MR 检查
- commit-checker diff ${CI_MERGE_REQUEST_DIFF_BASE_SHA}..HEAD
```

### init 命令

```bash
# 生成默认配置文件（自动检测系统区域设置）
commit-checker init

# 以指定区域设置生成
commit-checker init --lang en

# 覆盖现有文件
commit-checker init --force
```

### run 命令

```bash
# 检查所有已跟踪文件（与暂存状态无关）
commit-checker run

# 仅运行特定检查（--only）
commit-checker run --only lint
```

与 `diff` 不同，无论是否暂存，都会检查 `git ls-files` 跟踪的所有文件。

### prepare-msg 命令

用于 `prepare-commit-msg` 钩子的命令。在提交信息编辑器中以 `#` 注释显示当前生效的策略提示。
git 在提交时会去除 `#` 行，因此提示不会留在提交信息中。
对 `-m`/merge/squash/amend 提交不执行任何操作。

```bash
# 按原样接收 git 传递的参数: <文件> [source] [sha]
commit-checker prepare-msg .git/COMMIT_EDITMSG
```

### fix 命令

```bash
# 预览修复内容
commit-checker fix --dry-run

# 修复最近5个提交
commit-checker fix --range HEAD~5..HEAD

# 仅修复我的提交
commit-checker fix --mine --dry-run
```

### migrate 命令

```bash
# 检测配置文件架构版本并迁移到最新版
commit-checker migrate

# 预览更改（不修改文件）
commit-checker migrate --dry-run
```

自动将旧版配置文件（如 `no_coauthor` → `no_ai_coauthor`）转换为最新架构。
注释和格式会被保留。

### analyze 命令

```bash
# 分析当前仓库
commit-checker analyze
```

检测开发语言，并在缺少对应语言的lint配置文件（`.golangci.yml`, `.eslintrc.*`, `pyproject.toml` 等）时
发出警告。同时确认 `.editorconfig`, `.gitattributes`, `.gitignore` 是否存在。

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
| HCL (Terraform) | `.hcl` `.tf` `.tfvars` |

## i18n支持

CLI输出支持以下语言：

- 韩语 (ko) - 默认
- English (en)
- 日本語 (ja)
- 中文 (zh)

通过环境变量 `COMMIT_CHECKER_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG` 或配置文件的 `locale` 值选择。

## 许可证

MIT
