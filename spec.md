# issue2md 产品规格说明书 (spec.md)

| 字段 | 值 |
| --- | --- |
| 版本 | 0.1 (Draft) |
| 日期 | 2026-06-20 |
| 模块路径 | `github.com/duduniao85/issue2md` |
| Go 版本 | >= 1.24（当前 go.mod 为 1.26.4） |
| 宪法版本 | constitution.md v1.0 |
| 关联文档 | [`API-sketch.md`](./API-sketch.md)（暴露接口设计，核心设计的一部分） |

> 本文件是 issue2md 的**唯一权威需求来源**。所有实现、测试、范围决策均以本文件为准；
> 任何超出第 2.2 节「明确不做」的功能，必须先修订本 spec，不得擅自实现（对齐宪法 1.1 YAGNI）。

---

## 1. 产品概述

### 1.1 一句话定义
`issue2md` 是一个命令行工具：输入一个 github.com 的 Issue / Pull Request / Discussion 完整 URL，将其**元数据 + 正文 + 全部评论**（含图片）转换为一份带 YAML front matter 的本地 Markdown 文件，用于离线归档、知识沉淀与喂给 LLM。

### 1.2 目标用户
- 需要把 GitHub 讨论沉淀为本地文档的开发者。
- 希望把 Issue/PR/Discussion 内容喂给 AI（Claude 等）做总结、检索的开发者。

### 1.3 核心价值
- **零依赖**：仅用 Go 标准库（对齐宪法 1.2）。
- **可归档**：图片本地化，断链无忧。
- **可组合**：支持 stdout 输出，融入 Unix 管道。
- **可扩展**：核心逻辑与 CLI 解耦，未来可零改动复用于 Web 服务（见 §7.2）。

---

## 2. 范围 (Scope)

### 2.1 In Scope（MVP 必做）
1. 解析 github.com 的 Issue / PR / Discussion **完整 URL**。
2. 匿名访问公开仓库；环境变量 `GITHUB_TOKEN` 可选，用于私有仓库与提升速率。
3. 获取：元数据 + 正文 + **全部**评论（自动翻页）。
   - Issue/PR：REST API。
   - Discussion：GraphQL API，**递归拉取嵌套回复**。
4. 下载正文/评论中的图片到本地，并改写 Markdown 链接。
5. 输出带 YAML front matter 的 Markdown 文件；`-o -` 切到 stdout。
6. 清晰的错误分类与退出码。
7. **架构预留**：核心库暴露 HTTP 无关的 `Convert` 契约，并为未来 Web 服务保留顶级 `web/` 目录（见 §7.2）。

### 2.2 Out of Scope（明确不做 — 对齐宪法 1.1 YAGNI）
- ❌ GitHub Enterprise / 自建 GitHub（非 github.com 域名）。
- ❌ `owner/repo#N` 等简写格式（仅接受完整 URL）。
- ❌ Commit / Release / Gist / Wiki / Project 等非「讨论」类内容。
- ❌ PR 的 review comments、状态事件、CI 状态、commit 列表（仅取主讨论流评论）。
- ❌ 批量输入（多 URL / 文件列表）—— 交给 `xargs` / shell 循环。
- ❌ 自动重试 / 退避（快速失败，见 §6.1）。
- ❌ GitLab / Gitee 等第三方平台。
- ❌ 配置文件（`.issue2md.yaml`）；仅环境变量 + 命令行 flag。
- ❌ 交互式提示（如覆盖确认）—— 非交互 CLI，默认覆盖。
- ❌ 任何全局可变状态（对齐宪法 3.2）。
- ❌ Web 服务本身（本期仅预留 `web/` 目录与 `Convert` 共享契约，见 §7.2；未来再实现）。

---

## 3. 用户故事

| # | 角色 | 故事 | 验收要点 |
| --- | --- | --- | --- |
| US-1 | 开发者 | 我输入一个公开 Issue URL，得到一份含全部评论的本地 md | 文件生成、评论完整、图片已下载 |
| US-2 | 开发者 | 我设置 Token 后，能归档私有仓库的 PR | 带 Token 成功，且元数据含 PR 状态 |
| US-3 | 开发者 | 我把输出管道给 `grep`，快速检索内容 | `-o -` 输出到 stdout |
| US-4 | 开发者 | 我归档一个有 200+ 评论的 Discussion | 全量翻页 + 嵌套回复递归拉取，不截断 |
| US-5 | 开发者 | URL 无效 / 仓库私有但未带 Token | 得到清晰、可操作的错误信息与非零退出码 |

---

## 4. 功能需求

### 4.1 输入：URL 解析
- **仅接受**完整 `https://github.com/{owner}/{repo}/{kind}/{number}`：
  - `{kind}` ∈ {`issues`, `pull`, `discussions`}。
  - `{owner}`、`{repo}`：非空、仅允许 `[A-Za-z0-9._-]+`。
  - `{number}`：正整数。
- 协议：`http://` 与 `https://` 均接受（内部按 https 调用 API）。
- 域名必须为 `github.com`（大小写不敏感）；其余一律拒绝。
- 允许 URL 带末尾 `/`、锚点（`#...`）、查询串（`?...`）—— 解析时忽略。
- 非法 URL：返回**使用错误**（退出码 2），提示合法格式示例。

### 4.2 认证
- Token 来源优先级：`--token` flag > 环境变量 `GITHUB_TOKEN`。
- 无 Token：匿名调用（`Accept: application/vnd.github+json`）。
- 有 Token：附加请求头 `Authorization: Bearer <token>`。
- 无 Token 访问私有仓库 → GitHub 返回 404（见 §9.6），提示「该仓库可能私有，请设置 GITHUB_TOKEN」。
- Token 不落地、不打印、不写日志（安全：错误信息中必须脱敏）。

### 4.3 数据获取
**Issue / PR — REST API v3**（`https://api.github.com`）
- 主体：`GET /repos/{owner}/{repo}/issues/{number}`
  - 注：PR 在 issue 端点同样可取（`pull_request` 字段非空即为 PR）；PR 额外字段（merged / base / head）取 `GET /repos/{owner}/{repo}/pulls/{number}`。
- 评论：`GET /repos/{owner}/{repo}/issues/{number}/comments`，**自动翻页**（`per_page=100`，跟随 `Link` 头或递增 `page` 直到空）。

**Discussion — GraphQL API**（`https://api.github.com/graphql`）
- 使用 `repository.discussions(number:)` 查询，字段含 title/body/author/createdAt/labels/reactionCount，及 `comments(first:100)` 与每条评论的 `replies(first:100)`。
- **递归**拉取嵌套 replies，层数无上限（直到某层 replies 为空）。
- 翻页：使用 GraphQL cursor 分页（`pageInfo.endCursor` / `hasNextPage`）。

> 宪法 1.2 落实点：**不引入任何 GraphQL 客户端库**。GraphQL 请求用标准库 `net/http` 直接 POST JSON `{ "query": "...", "variables": {...} }`，响应用 `encoding/json` 解析。

### 4.4 内容深度（MVP 边界）
- **取**：title、url、type、repository(owner/repo)、number、author(login)、state、created_at、updated_at、labels[]、正文 body、**reactions 各类计数**（+1/-1/laugh/hooray/confused/heart/rocket/eyes/total，**不含用户明细**）、全部评论[]（每条含 author/created_at/body、reactions 计数）。
- PR 额外取：merged 状态、base 分支、head 分支。
- **不取**（§2.2）：reaction **用户明细**（即具体哪些用户 reacted）、review comments、状态事件时间线、关联 commit。

### 4.5 图片处理
- 扫描正文与每条评论的 body，匹配 Markdown 图片语法 `![alt](url)` 与裸图片 URL（`.png/.jpg/.jpeg/.gif/.webp/.svg`）。
- 对命中图片发起 GET 下载，存入 `{输出文件名去扩展名}.files/`，命名 `{seq}-{sha256前8位}.{原扩展名}`，**按 URL 去重**（同一 URL 只下载一次）。
- 将 Markdown 中的原 URL 改写为相对路径 `./{输出文件名}.files/{name}`。
- 无扩展名图片：按响应 `Content-Type` 推断扩展名；推断失败则用 `.bin`。
- 下载失败（超时/404/4xx）：**警告并保留原远程链接**，不中断整体转换（部分成功）。
- **stdout 模式特例**：当 `-o -` 时，**不下载图片、保留远程链接**（流式输出无法承载本地目录），并在 stderr 打印一条提示。

### 4.6 输出
- **默认输出文件**（当前工作目录）：`{owner}-{repo}-{type}-{number}.md`
  - `{type}`：`issue` / `pull` / `discussion`。
  - 示例：`golang-go-issue-123.md`。
- **图片目录**：与 md 同级，`{owner}-{repo}-{type}-{number}.files/`。
- `-o <path>`：指定输出文件路径（可为相对/绝对路径）；目录不存在则报错（不自动创建，保持行为可预测）。
- `-o -`：输出到 stdout。
- 文件已存在：**默认覆盖**（非交互，§2.2）。
- Markdown 结构：

  ```markdown
  ---
  title: "示例标题"
  url: "https://github.com/owner/repo/issues/123"
  type: "issue"
  repository: "owner/repo"
  number: 123
  author: "octocat"
  state: "open"
  created_at: "2024-01-01T00:00:00Z"
  updated_at: "2024-01-02T00:00:00Z"
  labels:
    - "bug"
  comments: 5
  source: "issue2md v0.1"
  fetched_at: "2026-06-20T12:00:00Z"
  ---

  # 示例标题

  <正文原文>

  ---

  ## 评论

  ### @reviewer1 · 2024-01-01T10:00:00Z

  <评论1原文>

  ### @reviewer2 · 2024-01-01T11:00:00Z

  <评论2原文>
  ```

  - Discussion 嵌套回复：用 `####` 层级与缩进表示父子关系。
  - PR：front matter 在 `source` 后输出 `pr:` 块（`merged`/`base`/`head`）。
  - **YAML 安全**：所有 front matter 字符串值用双引号包裹并对内部 `"`/`\` 转义；数组用 `- item`。
    > 宪法 1.2 落实点：**不引入 yaml 第三方库**，手写最小安全 YAML writer（字段集固定且简单）。

---

## 5. CLI 接口规范

### 5.1 用法
```
issue2md [flags] <github-url>
```

### 5.2 Flags
| Flag | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `-o` | string | 自动命名 | 输出文件路径；`-o -` 输出 stdout |
| `-token` | string | `$GITHUB_TOKEN` | GitHub Token |
| `-no-images` | bool | false | 不下载图片，保留远程链接 |
| `-timeout` | duration | `30s` | 单次 HTTP 请求超时 |
| `-baseurl` | string | `https://api.github.com` | GitHub API 根（高级/测试钩子） |
| `-v` | bool | — | 打印版本并退出 |
| `-h` | bool | — | 打印帮助并退出 |

> 用标准库 `flag` 包。位置参数有且仅有一个（URL）；多了/少了 → 退出码 2。

### 5.3 退出码
| 码 | 含义 | 示例 |
| --- | --- | --- |
| 0 | 成功 | — |
| 1 | 一般错误 | IO 失败、图片目录写入失败 |
| 2 | 使用错误 | 参数缺失、URL 非法 |
| 3 | GitHub API 错误 | 404 / 401 / 403(速率) / 5xx / 网络错误 |

### 5.4 示例
```bash
# 公开 Issue，默认输出文件
issue2md https://github.com/golang/go/issues/123

# 私有仓库 PR（依赖 $GITHUB_TOKEN）
issue2md https://github.com/acme/internal/pull/42

# 管道用法
issue2md -o - https://github.com/golang/go/issues/123 | grep -i bug

# 不下载图片
issue2md -no-images https://github.com/golang/go/discussions/7
```

---

## 6. 非功能需求

### 6.1 错误处理（对齐宪法 3.1，不可协商）
- 所有错误**显式处理**；传递时一律 `fmt.Errorf("...: %w", err)` 包装，保留因果链。
- 面向用户的错误信息：**一句可操作的话** + 退出码分类。
- **快速失败，不自动重试**（§2.2）：
  - 403 且 `X-RateLimit-Remaining: 0` → 提示「已达速率上限，请设置 GITHUB_TOKEN 或等待至 {reset 时间}」。
  - 401 → 提示「Token 无效」。
  - 404 → 提示「不存在，或为私有仓库需设置 GITHUB_TOKEN」。
  - 5xx / 网络错误 → 提示「GitHub 暂时不可用 / 网络问题，请稍后重试」。
- context 贯穿所有外部调用，受 `-timeout` 与调用方 ctx 控制。

### 6.2 依赖（对齐宪法 1.2 / 1.3）
- **零第三方依赖**：仅 `net/http`、`net/url`、`encoding/json`、`flag`、`os`、`fmt`、`context`、`time`、`regexp`、`crypto/sha256`、`encoding/hex`、`strings`、`strconv`、`path/filepath`、`io`。
- `go.mod` 不出现任何 `require`。

### 6.3 并发与性能
- 评论翻页可顺序拉取（MVP 不强制并发，保持简单）。
- 图片下载可串行（MVP）；如需优化，后续修订 spec。

---

## 7. 架构设计（对齐宪法 3.2：依赖注入、无全局变量）

```
cmd/
  issue2md/main.go            # CLI 入口：flag 解析 → 调用库 → 映射退出码
internal/issue2md/            # 核心库：本期实现，CLI 与未来 Web 共用
  types.go        # 核心数据结构：Document/Comment/Kind 等
  url.go          # URL 解析与校验
  errors.go       # 分类错误类型（UsageErr/APIErr/NotFound/RateLimit/...）
  github.go       # GitHub REST 客户端（注入 *http.Client + base URL + token）
  graphql.go      # GitHub GraphQL 客户端（同上）
  fetch.go        # 编排：按 Kind 调用对应客户端，翻页/递归拉取，组装 Document
  image.go        # 图片下载、去重命名、Markdown 链接改写
  yaml.go         # 最小安全 YAML front matter writer
  render.go       # Document → Markdown 字符串
  convert.go      # 顶层 Convert(ctx, url, opts) (result, error) 编排
  *_test.go       # 每个文件配套测试
web/                          # 未来 Web 服务（本期占位 README，不写实现）
  README.md       # 预留说明：复用 internal/issue2md.Convert，包一层 net/http handler
```

### 7.1 关键约定
- 所有外部依赖（`*http.Client`、时间源、文件系统写入）**通过函数参数/结构体字段注入**，便于测试与对齐宪法 3.2。
- 顶层入口 `Convert(ctx context.Context, src string, opts Options) (*Result, error)`：库的公共契约，CLI 只是薄封装。
- `Options` 结构体聚合所有可配置项（Token/Timeout/OutputPath/NoImages/...）。
- 所有公共类型与函数的完整签名以 [`API-sketch.md`](./API-sketch.md) 为准；本文仅作摘要引用。

### 7.2 Web 扩展点（预留，本期不实现）
本期仅交付 CLI；但架构上为未来 Web 服务预留：
- **共享契约**：`Convert(ctx, src, opts) (*Result, error)` 是 HTTP 无关的纯函数。未来 Web 的 handler 只需把 HTTP 参数映射为 `Options`、把 `Result` 序列化为 JSON，**零侵入复用核心库**，无需改动 `internal/issue2md`。
- **目录**：顶级 `web/` 与 `internal/` 同级，本期仅放 `README.md` 占位，说明未来 `server.go`（net/http handler 层）与 `api.go`（HTTP 契约 / DTO）的职责。
- **Makefile**：`make web` 为占位 target（本期提示「Web 服务尚在规划」，未来用于构建 web 二进制）。

> 这是「预留位置与契约，不实现功能」的 YAGNI 合规做法 —— 不提前编写 Web 代码，但保证其上线时是加法而非重构。待 Web 需求明确后，再为本期新增独立 spec 章节。

---

## 8. 测试策略（对齐宪法第二条 TDD 铁律）

### 8.1 TDD 循环（不可协商）
- 每个功能/Bug 修复从**一个失败测试**开始（Red → Green → Refactor）。

### 8.2 表格驱动（对齐宪法 2.2）
- 所有单元测试采用 table-driven：`[]struct{ name string; input X; want Y }`。
- 重点表格：URL 解析（合法/非法各分支）、YAML 转义（特殊字符）、图片链接改写、退出码映射。

### 8.3 真实依赖（对齐宪法 2.3：拒绝 Mock）
- GitHub 交互测试用标准库 `net/http/httptest` 起本地 HTTP 服务器：
  - 返回固定 JSON/GraphQL fixture。
  - 用**真实** `*http.Client` 打本地地址。
  - 这是真实的 HTTP 往返（非接口 mock），符合宪法 2.3 精神。
- 端到端集成测试：注入指向 httptest server 的 base URL，跑 `Convert`，断言生成的 md 文件内容 + 图片文件落地。

---

## 9. 边缘场景清单（验收必看）

| # | 场景 | 期望行为 |
| --- | --- | --- |
| 9.1 | 非 github.com 域名 | 退出码 2，提示仅支持 github.com |
| 9.2 | kind 非 issues/pull/discussions | 退出码 2，提示合法格式 |
| 9.3 | number 非正整数 / 缺失 | 退出码 2 |
| 9.4 | URL 带锚点 / 查询串 / 末尾斜杠 | 正常解析，忽略附加部分 |
| 9.5 | 公开仓库 + 无 Token | 正常工作（匿名） |
| 9.6 | 私有仓库 + 无 Token | GitHub 返回 404；提示「可能私有，请设置 GITHUB_TOKEN」 |
| 9.7 | Token 无效 (401) | 退出码 3，提示 Token 无效；**Token 不回显** |
| 9.8 | 速率上限 (403 + X-RateLimit-Remaining:0) | 退出码 3，提示等待至 reset 时间或设置 Token |
| 9.9 | 评论 > 100 条 | 自动翻页，全量获取 |
| 9.10 | Discussion 多层嵌套回复 | 递归拉取并按层级渲染 |
| 9.11 | 图片 URL 重复出现 | 仅下载一次，多处共享同一本地路径 |
| 9.12 | 某张图片下载失败 | 警告 + 保留原远程链接，不中断 |
| 9.13 | 图片无扩展名 | 按 Content-Type 推断，失败用 `.bin` |
| 9.14 | stdout 模式 (`-o -`) | 不下载图片，保留远程链接，stderr 提示 |
| 9.15 | 输出文件已存在 | 默认覆盖（非交互） |
| 9.16 | 输出目录不存在 | 退出码 1，提示目录不存在（不自动创建） |
| 9.17 | 正文/评论含代码围栏 ``` | 原样保留；front matter 用 `---` 闭合，正文不受影响 |
| 9.18 | 标题/正文含 `:` `"` 换行 等 | front matter 值双引号包裹并转义 |
| 9.19 | 网络超时 / DNS 失败 | 退出码 3，提示网络问题 |
| 9.20 | GitHub 5xx | 退出码 3，提示暂时不可用 |

---

## 10. 验收标准 (Acceptance Criteria)

- **AC-1**：`issue2md <公开issue url>` 在当前目录生成 `{owner}-{repo}-issue-{n}.md`，含 front matter、正文、全部评论；exit 0。
- **AC-2**：`GITHUB_TOKEN=... issue2md <私有pr url>` 成功，front matter 含 `type: pull` 且 state 正确；exit 0。
- **AC-3**：`issue2md -o - <url>` 输出到 stdout，图片保留远程链接，stderr 有提示；exit 0。
- **AC-4**：对 >100 评论的 Discussion，输出文件评论数 == API 实际评论数（全量）。
- **AC-5**：正文含图片时，生成 `{...}.files/` 目录，md 内图片链接已改写为相对路径，文件存在。
- **AC-6**：非法 URL → exit 2 且提示合法格式；私有仓库无 Token → exit 3 且提示设置 Token。
- **AC-7**：`go.mod` 无第三方依赖；`go vet ./...` 与 `go build ./...` 通过。
- **AC-8**：`make test` 全绿，覆盖 §9 全部边缘场景（表格驱动）。
- **AC-9**：错误链可被 `%w` 解包（`errors.Is/As` 可识别 NotFound/RateLimit/UsageErr）。
- **AC-10**：`web/` 目录存在且含 `README.md` 占位；`internal/issue2md.Convert` 签名稳定，未来 Web 可直接复用（本期不实现 Web）。

---

## 11. 交付物清单

1. `spec.md`（本文件）。
2. `API-sketch.md`：核心库与未来 Web 的接口设计草案（§7 引用，接口以它为准）。
3. `Makefile`：`make test`（`go test ./...`）、`make build`（构建 `bin/issue2md`）、`make run`（便捷运行）、`make web`（占位 target，本期提示「Web 服务尚在规划」，见 §7.2）。
4. `internal/issue2md/*.go` + 配套 `*_test.go`。
5. `cmd/issue2md/main.go`。
6. `web/README.md`：未来 Web 服务的预留说明（复用 `Convert`、handler/api 职责划分）。
7. `README.md`：安装、用法、Token 配置、退出码说明。

---

## 附录 A：决策溯源（共创结论快照）

| 维度 | 决策 | 依据 |
| --- | --- | --- |
| 内容范围 | Issue + PR + Discussion | 用户选定；Discussion 需 GraphQL |
| 认证 | 可选 Token（环境变量） | 兼顾匿名易用与私有能力 |
| 输出 | 文件 + stdout 双模式 | Unix 管道友好 |
| 内容深度 | 正文 + 全部评论 | 常见归档需求 |
| URL 平台 | 仅 github.com | YAGNI |
| Markdown 格式 | YAML front matter | 可被静态站点/LLM 消费 |
| 图片 | 下载到本地 | 归档断链无忧 |
| 批量 | 单次单个 URL | 职责单一，shell 补齐 |
| 重试 | 快速失败 | 简单、可预测 |
| 架构 | 可复用库 + CLI 薄封装 | 宪法 3.2，可测可注入 |
| 测试 | htttest 本地服务器 | 宪法 2.3，真实 HTTP 往返 |
| 输出结构 | 单 md + 同名 .files 目录 | 文件独立、清晰 |
| 依赖 | 零第三方 | 宪法 1.2 标准库优先 |
| Web 预留 | 独立顶级 `web/` 目录 + `Convert` 共享契约 | 用户：未来需 Web；YAGNI 预留位置不实现 |
