# issue2md 任务分解 (tasks.md)

> 编号：`specs/001-core-functionality`
> 关联：[`plan.md`](./plan.md) · [`spec.md`](../../spec.md) · [`API-sketch.md`](../../API-sketch.md) · [`constitution.md`](../../constitution.md)

| 字段 | 值 |
| --- | --- |
| 版本 | 0.1 |
| 日期 | 2026-06-20 |
| 作者 | 技术组长（Claude） |
| 目的 | 将 `plan.md` 分解为**原子化、可被 AI 直接执行**的任务，TDD 强制、依赖明确 |

---

## 0. 约定与图例

- **TDD 强制**（宪法第二条）：每个有行为的功能 = 一个**测试任务**(后缀 `a`，Red) + 一个**实现任务**(后缀 `b`，Green)。`a` 必须先于 `b`。
- **声明类例外**：纯类型/接口声明（`F1`/`F3`）无行为可测，不单列测试任务；由编译保证 + 消费方测试覆盖。这是 TDD 对"声明 vs 行为"的合理区分。
- **`[P]`**：该任务与同执行窗口内其他 `[P]` 任务**无相互依赖**，可并行。
- **依赖**：列前置任务 ID；阶段非绝对串行（如 Phase 3 的 `image` 依赖 Phase 2 的 `httpc`），以 ID 为准。
- **粒度**：每任务**只动一个主文件**（实现任务）或**只建一个测试文件**（测试任务）。
- **骨架现状**：`types.go`/`errors.go`/`convert.go`/`render.go`/`main.go` 已有占位；`url.go`/`github.go`/`graphql.go`/`fetch.go`/`image.go`/`yaml.go` 仅有注释；`source.go`/`httpc.go` 及多数 `*_test.go` 不存在。

### 0.1 依赖总览（关键路径）
```
Phase1:  F1 ──┬──► F3 ──────────────────────────────────┐ (Source 接口)
              │                                          │
         F2a ─► F2b ─────────────────────────────────────┤ (errors.Is)
                                                        │
Phase2:  G1a ─► G1b ────────────────────────────────────┐│ (url→Ref)
         G2a ─► G2b ──┬──► G3a ─► G3b ──┐               ││ (httpc 基建)
                       └──► G4a ─► G4b ──┴──► G5a ─► G5b ┘│ (REST/GraphQL/fetch)
                                                         │
Phase3:  C1a ─► C1b ──┬──► C2a ─► C2b ───────────────────┤ (yaml/render)
         C3a ─► C3b ──┘(依赖 G2b)                        │ (image)
                                                         │
Phase4:  A1a ─► A1b ─► A2a ─► A2b ─► A3 ─────────────────┘ (convert/CLI)
         (A1a 依赖 G1b,G5b,C2b,C3b)
```

### 0.2 任务清单速查
| ID | 文件 | 类型 | 依赖 | 阶段 |
| --- | --- | --- | --- | --- |
| F1 | types.go | 声明 | — | P1 |
| F2a | errors_test.go | 测试 | — | P1 |
| F2b | errors.go | 实现 | F2a | P1 |
| F3 | source.go | 声明 | F1 | P1 |
| G1a | url_test.go | 测试 | F1 | P2 |
| G1b | url.go | 实现 | G1a,F1 | P2 |
| G2a | httpc_test.go | 测试 | F2b | P2 |
| G2b | httpc.go | 实现 | G2a,F2b | P2 |
| G3a | github_test.go | 测试 | G2b,F1 | P2 |
| G3b | github.go | 实现 | G3a,G2b,F3 | P2 |
| G4a | graphql_test.go | 测试 | G2b,F1 | P2 |
| G4b | graphql.go | 实现 | G4a,G2b,F3 | P2 |
| G5a | fetch_test.go | 测试 | G3b,G4b,F3 | P2 |
| G5b | fetch.go | 实现 | G5a,G3b,G4b,F3 | P2 |
| C1a | yaml_test.go | 测试 | — | P3 |
| C1b | yaml.go | 实现 | C1a | P3 |
| C2a | render_test.go | 测试 | C1b,F1 | P3 |
| C2b | render.go | 实现 | C2a,C1b,F1 | P3 |
| C3a | image_test.go | 测试 | G2b | P3 |
| C3b | image.go | 实现 | C3a,G2b | P3 |
| A1a | convert_test.go | 测试 | G1b,G5b,C2b,C3b | P4 |
| A1b | convert.go | 实现 | A1a,G1b,G5b,C2b,C3b | P4 |
| A2a | cmd/issue2md/main_test.go | 测试 | A1b | P4 |
| A2b | cmd/issue2md/main.go | 实现 | A2a,A1b | P4 |
| A3 | README.md + 全量验证 | 集成 | A2b | P4 |

---

## Phase 1 — Foundation（数据结构定义）

### [P] F1 · types.go：补全核心数据模型
- **文件**：`internal/issue2md/types.go`（修改）
- **类型**：声明（TDD 例外：纯类型，由编译 + 消费方测试覆盖）
- **依赖**：无（骨架已有 `Kind`/`Document`/`Comment`/`PRInfo`）
- **参考**：API-sketch §A.5；spec §4.4
- **任务**：在现有骨架上新增 `Ref`、`Reactions` 两个 struct；给 `Document` 增加 `Reactions Reactions` 字段（置于 `Body` 与 `Comments` 之间），给 `Comment` 增加 `Reactions Reactions` 字段（置于 `Body` 与 `Replies` 之间）。字段名、类型、顺序严格对齐 API-sketch §A.5。
  - `Ref{ Kind; Owner; Repo; Number int }`
  - `Reactions{ TotalCount, PlusOne, MinusOne, Laugh, Hooray, Confused, Heart, Rocket, Eyes int }`
- **验收**：`go build ./...` 与 `go vet ./...` 通过。

### [P] F2a · errors_test.go：错误模型测试（Red）
- **文件**：`internal/issue2md/errors_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：无（`errors.go` 类型/哨兵已在骨架）
- **参考**：API-sketch §A.6；宪法 3.1
- **任务**：表格驱动测试 `*Error` 行为，覆盖：
  1. `Error()` 文本：`Op` 非空时为 `"op: message"`，空时为 `"message"`。
  2. `Unwrap()` 返回 `Cause`，且 `errors.Is(wrap(&Error{Cause: sentinel}), sentinel)` 为 true（链可达）。
  3. `Is(target)` 按 Kind 命中 7 个哨兵（API-sketch §A.6 表）：KindUsage↔ErrInvalidURL、KindIO↔ErrIO、KindNotFound↔ErrNotFound、KindUnauthorized↔ErrUnauthorized、KindRateLimited↔ErrRateLimited、KindServer↔ErrServerUnavailable、KindNetwork↔ErrNetwork。
  4. 交叉否定：`KindNotFound` 的 err 对 `ErrRateLimited` 应为 false。
- **验收**：`go test ./internal/issue2md/ -run Error` **失败**（Red）—— 当前 `Is()` 返回 `false`。

### F2b · errors.go：补全 `Is()` 实现（Green）
- **文件**：`internal/issue2md/errors.go`（修改）
- **类型**：实现（Green）
- **依赖**：F2a
- **参考**：API-sketch §A.6
- **任务**：实现 `func (e *Error) Is(target error) bool`：`switch e.Kind` 映射到对应哨兵，与 `target` 比较（`target == 哨兵`，因哨兵是 `errors.New` 产生的单例值）。覆盖 7 个映射。
- **验收**：F2a 测试转 Green；`go vet` 通过。

### F3 · source.go：定义 `Source` 接口
- **文件**：`internal/issue2md/source.go`（新建）
- **类型**：声明（TDD 例外：接口契约；实现与 `NewSource` 在 G5b）
- **依赖**：F1（`Ref`/`Document`）
- **参考**：API-sketch §A.8；plan §5.2
- **任务**：定义 `Source` 接口：`Fetch(ctx context.Context, ref Ref) (*Document, error)`。加包级注释说明「全库唯一接口，REST/GraphQL 双实现，默认实现 `githubSource` 在 fetch.go」。**不**在此实现 `NewSource`（留 G5b，因依赖 `githubSource`）。
- **验收**：`go build` 通过；签名与 API-sketch §A.8 一致。

---

## Phase 2 — GitHub Fetcher（API 交互逻辑，TDD）

> 本阶段 REST 与 GraphQL 均用标准库手写，共用 `httpc.go` 基建（路线 B）。所有外部交互测试用 `net/http/httptest` 起真实 HTTP server（宪法 2.3，拒绝 Mock）。

### [P] G1a · url_test.go：URL 解析测试（Red）
- **文件**：`internal/issue2md/url_test.go`（替换骨架 skip 占位）
- **类型**：测试（Red）
- **依赖**：F1（`Ref`/`Kind`）
- **参考**：spec §4.1、§9.1–9.4；plan §6.2
- **任务**：表格驱动测试 `parseURL`（待 G1b 实现），列：合法（issue/pull/discussions 各 1；带 `#anchor`/`?q=1`/末尾 `/`；`http://`/`https://`）→ 断言 `(Kind, Owner, Repo, Number)`；非法（非 github.com、kind 错、number 非正整数/缺失、路径段不足）→ 断言 `errors.Is(err, ErrInvalidURL)`。每行一个 `name`。
- **验收**：编译失败或 `go test -run ParseURL` 失败（Red）。

### G1b · url.go：`parseURL` 实现（Green）
- **文件**：`internal/issue2md/url.go`（实现，当前仅注释）
- **依赖**：G1a、F1
- **参考**：spec §4.1；plan §6.2
- **任务**：实现 `parseURL(src string) (Ref, error)`：`net/url` 解析；校验 `host == github.com`（大小写不敏感）；`path` 按 `/` 分段取 `{owner}/{repo}/{kind}/{number}`；`kind` 映射 `issues→KindIssue`、`pull→KindPull`、`discussions→KindDiscussion`；`number` 用 `strconv.Atoi` 且 >0；忽略 fragment/query/末尾斜杠。非法返回 `&Error{Kind: KindUsage, Op: "parse url", Message: ...}`（`errors.Is` 命中 `ErrInvalidURL`）。
- **验收**：G1a 测试 Green。

### [P] G2a · httpc_test.go：HTTP 基建测试（Red）
- **文件**：`internal/issue2md/httpc_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：F2b（错误类型，用于断言映射）
- **参考**：plan §3.3、§6.3；spec §6.1
- **任务**：用 `httptest.NewServer` 表格测试 `httpc.go`（待 G2b）各子功能：
  1. 请求头：有 Token→`Authorization: Bearer`；始终 `Accept: application/vnd.github+json` 与 `User-Agent`。
  2. 状态码→ErrorKind：404→`KindNotFound`、401→`KindUnauthorized`、403+`X-RateLimit-Remaining:0`→`KindRateLimited`（断言 `ResetAt` 来自 `X-RateLimit-Reset`）、500→`KindServer`。
  3. `Link` 头解析：`<url>; rel="next"` 正确提取下一页 URL；无 next 时返回空。
  4. 网络错误（关停 server）→`KindNetwork`。
- **验收**：Red。

### G2b · httpc.go：HTTP 基建实现（Green）
- **文件**：`internal/issue2md/httpc.go`（新建）
- **依赖**：G2a、F2b
- **参考**：plan §3.3、§6.3
- **任务**：实现共用 HTTP 基建（未导出）：`client` struct（持有注入的 `*http.Client`、`baseURL`、`token`、`timeout`）；`newClient(opts)` 构造；`(*client).do(ctx, method, path, query) (*http.Response, error)` 设头发请求；`(*client).check(resp) error` 状态码→`*Error` 映射 + 解析 `X-RateLimit-*`；`parseNextLink(header) string` 解析 `Link` rel=next。错误用 `fmt.Errorf("...: %w", err)` 包装。
- **验收**：G2a 测试 Green。

### G3a · github_test.go：REST 数据源测试（Red）
- **文件**：`internal/issue2md/github_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：G2b、F1
- **参考**：spec §4.3；plan §6.3
- **任务**：`httptest` + 表格测试 REST 获取与映射：
  1. Issue：`GET /repos/{o}/{r}/issues/{n}` fixture → `Document` 字段（Title/Author/State/Labels/Body/Reactions）正确。
  2. PR：额外 `GET /pulls/{n}` → `Document.PR.{Merged,Base,Head}` 正确。
  3. 评论分页：`ListIssueComments` 两页（`Link: rel="next"`），断言全量评论数与顺序；`per_page=100`。
  4. httptest 校验请求路径与头。
- **验收**：Red。

### G3b · github.go：REST 数据源实现（Green）
- **文件**：`internal/issue2md/github.go`（实现，当前仅注释）
- **依赖**：G3a、G2b、F3、F1
- **参考**：spec §4.3；plan §6.3
- **任务**：定义未导出 json struct（`restIssue`/`restComment`/`restReactions`/`restUser`/`restLabel`，带 tag）；实现 `fetchIssue`/`fetchPull`/`listComments(ctx, client, ref) ([]Comment, error)`，用 `httpc` 发请求、`parseNextLink` 翻页直到无 next；映射到 `Document`/`Comment`（含 `Reactions`、PR 字段）。
- **验收**：G3a 测试 Green。

### [P] G4a · graphql_test.go：GraphQL 数据源测试（Red）
- **文件**：`internal/issue2md/graphql_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：G2b、F1（与 G3a 并行）
- **参考**：spec §4.3、§9.10；plan §6.3
- **任务**：`httptest` 测试 GraphQL 获取：
  1. 单次 query 返回 discussion 主体 + 首层 comments。
  2. 嵌套 `replies` 多层（≥2 层）→ 断言 `Comment.Replies` 递归结构正确。
  3. cursor 分页：`pageInfo.hasNextPage=true` + `endCursor` → 续页，直至全量 comments。
  4. httptest 校验收到的 `query`/`variables` 字段。
- **验收**：Red。

### G4b · graphql.go：GraphQL 数据源实现（Green）
- **文件**：`internal/issue2md/graphql.go`（实现，当前仅注释）
- **依赖**：G4a、G2b、F3、F1
- **参考**：spec §4.3；plan §6.3、§1.3
- **任务**：定义未导出 json struct（`discussionResp` 嵌套 `repository.discussion` + `comments.nodes` + `replies.nodes` + `pageInfo`）；`const discussionQuery`（含变量 `$owner/$repo/$number/$commentsCursor`）；`fetchDiscussion(ctx, client, ref) (*Document, error)` 用 `httpc` POST `/graphql` body `{query,variables}`；递归收集 replies；cursor 翻页 comments；映射 `Document`（嵌套 `Replies`）。
- **验收**：G4a 测试 Green。

### G5a · fetch_test.go：Source 调度测试（Red）
- **文件**：`internal/issue2md/fetch_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：G3b、G4b、F3
- **参考**：plan §6.3；API-sketch §A.8
- **任务**：测试 `NewSource` 构造 + `githubSource.Fetch` 分发：`Ref{Kind:KindIssue}`→命中 REST 路径（httptest 校验 `/issues/`）；`KindDiscussion`→命中 GraphQL 路径（校验 `/graphql`）；错误透传（404→`ErrNotFound`）。
- **验收**：Red。

### G5b · fetch.go：`NewSource` + `Fetch` 实现（Green）
- **文件**：`internal/issue2md/fetch.go`（实现，当前仅注释）
- **依赖**：G5a、G3b、G4b、F3
- **参考**：plan §6.3、§5.2；API-sketch §A.8
- **任务**：定义未导出 `githubSource` struct（持有 `httpc` client）；`NewSource(opts Options) Source` 构造（实现 F3 声明的契约）；`(*githubSource).Fetch(ctx, ref) (*Document, error)` 按 `ref.Kind` 分发：`KindIssue`/`KindPull`→`github.go` 的 `fetchIssue`/`fetchPull`+`listComments`；`KindDiscussion`→`graphql.go` 的 `fetchDiscussion`；组装并返回 `*Document`（填 `FetchedAt`）。
- **验收**：G5a 测试 Green；`NewSource` 可被 `Convert` 调用。

---

## Phase 3 — Markdown Converter（转换逻辑，TDD）

### [P] C1a · yaml_test.go：YAML writer 测试（Red）
- **文件**：`internal/issue2md/yaml_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：无
- **参考**：spec §4.6；plan §6.5
- **任务**：表格驱动测试 YAML front matter writer：
  1. 字符串值双引号包裹；内部 `"`/`\` 转义；冒号、换行不破坏结构。
  2. 数组（labels）渲染为 `- "item"` 多行。
  3. front matter 以 `---` 开头与结尾。
  4. 给定一组字段（title/url/.../labels），断言整体输出字节精确匹配预期。
- **验收**：Red。

### C1b · yaml.go：YAML writer 实现（Green）
- **文件**：`internal/issue2md/yaml.go`（实现，当前仅注释）
- **依赖**：C1a
- **参考**：spec §4.6；plan §6.5；宪法 1.2（不引 yaml 库）
- **任务**：手写最小安全 writer：`quoteYAML(s) string`（双引号 + 转义 `"`/`\`/换行）；`writeFrontMatter(fields map[string]any) string`（标量双引号、`[]string` 渲染为 `-` 列表、`---` 包裹）。字段集固定（对齐 spec §4.6 front matter 示例）。
- **验收**：C1a 测试 Green。

### C2a · render_test.go：Render 测试（Red）
- **文件**：`internal/issue2md/render_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：C1b、F1
- **参考**：spec §4.6；API-sketch §A.7
- **任务**：表格驱动测试 `Render(doc)`：
  1. front matter（调 C1b）+ `# Title` + 正文 + `## 评论` 结构。
  2. Issue/PR 评论平铺为 `### @author · time`；Discussion 嵌套用 `####` 层级。
  3. 正文含代码围栏 ``` 原样保留。
  4. 构造 `Document` fixture（含 `Reactions`/嵌套 `Replies`），断言输出含正确片段。
- **验收**：Red（当前 `Render` 为 panic）。

### C2b · render.go：`Render` 实现（Green）
- **文件**：`internal/issue2md/render.go`（实现，当前 panic 占位）
- **依赖**：C2a、C1b、F1
- **参考**：spec §4.6；API-sketch §A.7
- **任务**：实现 `Render(doc *Document) (string, error)`：组装 front matter（C1b）+ `# {Title}` + body + `## 评论`（评论逐条 `### @{Author} · {CreatedAt}`，Discussion 递归 `Replies` 用 `####` 加层级）。**纯函数**，不改图片链接（边界见 API-sketch §A.7）。
- **验收**：C2a 测试 Green。

### [P] C3a · image_test.go：图片处理测试（Red）
- **文件**：`internal/issue2md/image_test.go`（新建）
- **类型**：测试（Red）
- **依赖**：G2b（用 httpc 下载）
- **参考**：spec §4.5、§9.11–9.14；plan §6.4
- **任务**：`httptest` 提供图片字节，表格测试：
  1. 扫描 `![alt](url)` 与裸图 URL。
  2. 同 URL 出现多次→仅下载一次、多处共享本地路径。
  3. 命名 `{seq}-{sha256前8}.{ext}`。
  4. 无扩展名→按响应 `Content-Type` 推断（image/png→.png）；失败→`.bin`。
  5. 某图下载失败→保留原远程链接 + 记入 warnings，不中断。
- **验收**：Red。

### C3b · image.go：图片处理实现（Green）
- **文件**：`internal/issue2md/image.go`（实现，当前仅注释）
- **依赖**：C3a、G2b
- **参考**：spec §4.5；plan §6.4
- **任务**：实现 `localizeImages(md string, imageDir string, c *httpc.Client) (string, []string, error)`：正则扫描图片 URL；按 URL 去重 map；`httpc` 下载；`crypto/sha256` 前 8 位命名；`Content-Type`→扩展名映射；改写 markdown 为相对路径 `./{base}.files/{name}`；失败保留原链接并收集 warning。
- **验收**：C3a 测试 Green。

---

## Phase 4 — CLI Assembly（命令行入口集成）

### A1a · convert_test.go：Convert 集成测试（Red）
- **文件**：`internal/issue2md/convert_test.go`（新建）
- **类型**：测试（Red，端到端）
- **依赖**：G1b、G5b、C2b、C3b
- **参考**：spec §6.1、§9、§10（AC-1~AC-9）；plan §6.1
- **任务**：用 `httptest` 模拟 GitHub，端到端测 `Convert`：
  1. 合法 Issue URL→生成 `{owner}-{repo}-issue-{n}.md` + `.files/`，内容含 front matter/正文/评论；exit 无关（库返回 nil err）。
  2. `OutputPath:"-"`→`Result.Markdown` 非空、`ImageDir==""`、不下载图片。
  3. `NoImages:true`→保留远程图片链接。
  4. 错误映射：非法 URL→`ErrInvalidURL`、404→`ErrNotFound`、速率→`ErrRateLimited`、输出目录不存在→`ErrIO`。
  5. 已存在输出文件→覆盖。
- **验收**：Red（当前 `Convert` 为 panic）。

### A1b · convert.go：`Convert` 编排实现（Green）
- **文件**：`internal/issue2md/convert.go`（实现，当前 panic 占位）
- **依赖**：A1a、G1b、G5b、C2b、C3b、F1
- **参考**：plan §6.1；API-sketch §A.2 不变量
- **任务**：实现 `Convert(ctx, src, opts) (*Result, error)` 编排：`parseURL`→`NewSource(opts)`→`Fetch`→`Render`→（若 `OutputPath=="-" || NoImages`）直接返回 `Result{Markdown}`；否则解析输出路径（目录须存在否则 `KindIO`）、`localizeImages`、写文件、返回 `Result`。所有错误 `fmt.Errorf("...: %w", err)` 包装。遵守 §A.2 不变量。
- **验收**：A1a 测试 Green。

### A2a · cmd/issue2md/main_test.go：CLI smoke 测试（Red）
- **文件**：`cmd/issue2md/main_test.go`（新建）
- **类型**：测试（Red，集成）
- **依赖**：A1b
- **参考**：spec §5
- **任务**：`go build` 测试二进制（`os/exec`），smoke：`-v`→打印版本 exit 0；无参数→exit 2；非法 URL→exit 2；合法 URL（指向 httptest）→exit 0 且生成文件。
- **验收**：Red（当前 main 打印"未实现" exit 2）。

### A2b · cmd/issue2md/main.go：CLI 接入实现（Green）
- **文件**：`cmd/issue2md/main.go`（修改，替换骨架"未实现"分支）
- **依赖**：A2a、A1b
- **参考**：spec §5；API-sketch §A.6
- **任务**：将骨架的 TODO 分支替换为：flag→`Options`（Token 优先级 `-token` > `$GITHUB_TOKEN`）；位置参数校验（恰好 1 个 URL，否则 exit 2）；调用 `issue2md.Convert`；按 `API-sketch §A.6` 把 `*issue2md.Error.Kind` 映射到退出码（Usage→2、IO→1、其余→3）；成功 exit 0。
- **验收**：A2a 测试 Green。

### A3 · README.md + 全量验证
- **文件**：`README.md`（修改）+ 全项目
- **类型**：集成 / 验证
- **依赖**：A2b
- **参考**：spec §10（AC-1~AC-10）
- **任务**：更新 README（用法/Token 配置/退出码）；运行 `make test`、`make vet`、`make build`、`gofmt -l .` 全绿；逐条核对 spec §10 验收标准（含 AC-10：`web/README.md` 占位与 `Convert` 契约稳定）。
- **验收**：`make test` 全绿；gofmt 空；AC-1~AC-10 达成。

---

## 附录：执行顺序建议（尊重依赖的关键路径）

1. **Phase 1**：`F1` ∥ `F2a` → `F2b` ∥ `F3`
2. **Phase 2**：`G1a` ∥ `G2a` → `G1b` ∥ `G2b` → `G3a` ∥ `G4a` → `G3b` ∥ `G4b` → `G5a` → `G5b`
3. **Phase 3**：`C1a` ∥ `C3a` → `C1b` ∥ `C3b` → `C2a` → `C2b`（注：`C3a` 需 `G2b` 先完成）
4. **Phase 4**：`A1a` → `A1b` → `A2a` → `A2b` → `A3`

> 关键路径（最长链，约 11 步）：
> `F2a → F2b → G2a → G2b → G3a → G3b → G5a → G5b → A1a → A1b → A2a → A2b → A3`

---

## 附录：与 spec §10 验收标准的映射
| AC | 由哪些任务保证 |
| --- | --- |
| AC-1（公开 Issue→md） | G1b+G3b+G5b+C2b+A1b |
| AC-2（私有 PR，Token） | G2b(auth)+G3b(PR)+A2b(Token) |
| AC-3（stdout） | A1b（OutputPath=="-" 分支） |
| AC-4（>100 评论全量） | G3b（Link 分页）/ G4b（cursor） |
| AC-5（图片本地化） | C3b+A1b |
| AC-6（非法/私有错误） | G1b(非法)+G2b(404)+A2b(退出码) |
| AC-7（零依赖+vet/build） | 全程零第三方；A3 验证 |
| AC-8（make test 全绿，§9 覆盖） | 全部 `*a` 测试任务 + A3 |
| AC-9（errors.Is/As） | F2a/F2b |
| AC-10（web 占位+Convert 稳定） | 骨架已就位；F3+A1b 保证签名 |
