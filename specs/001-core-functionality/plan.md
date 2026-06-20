# issue2md 技术实现方案 (plan.md)

> 编号：`specs/001-core-functionality`
> 关联：[`constitution.md`](../../constitution.md) · [`spec.md`](../../spec.md) · [`API-sketch.md`](../../API-sketch.md)

| 字段 | 值 |
| --- | --- |
| 版本 | 0.2 (Draft) |
| 日期 | 2026-06-20 |
| 作者 | 首席架构师（Claude） |
| 状态 | **路线 B 定稿**：D-1 已撤销（零依赖）；D-3/D-4 轻量文档同步 |

---

## 0. 摘要与关键决策声明（写给决策者）

### 0.1 一句话定位
本方案在已就位的 `spec.md` / `API-sketch.md` / 项目骨架之上，给出 `issue2md` 核心功能的**可落地实现路径**：包职责、数据结构、接口、流程、测试与里程碑。

### 0.2 偏离点状态
| # | 偏离点 | 状态 |
| --- | --- | --- |
| D-1 | 引入 `google/go-github` | ✅ **已撤销** — 采纳**路线 B（零依赖手写）**。无需修宪。 |
| D-2 | "go-github + GraphQL v4" | ✅ **技术纠正** — go-github 不支持 GraphQL；REST 与 GraphQL **统一用标准库 `net/http` 手写**。 |
| D-3 | `Reactions` 字段 | ⚠️ 轻量修订 — 纳入 `Document`/`Comment`（计数，非用户明细）；同步 `spec.md §4.4`。 |
| D-4 | `Source` 接口 | ⚠️ 轻量修订 — 引入**唯一**核心接口（统一 REST/GraphQL 数据源）；同步 `API-sketch.md §0`。 |

> **路线 B 的核心优势**：GraphQL 本就必须手写（go-github 无 GraphQL），故 REST 复用同一套手写 HTTP 基建，**风格统一、零额外依赖、边际成本最低**。详见 §1.3。

---

## 1. 技术上下文总结

### 1.1 技术选型
| 维度 | 选型 | 依据 |
| --- | --- | --- |
| 语言 | Go ≥ 1.21.0（项目 go.mod 为 1.26.4） | 用户指令 + 宪法 |
| Web 框架 | 仅标准库 `net/http` | 宪法 1.2；本期 Web 为预留（spec §7.2） |
| GitHub REST（Issue/PR） | **标准库 `net/http` + `encoding/json` 手写** | 路线 B（用户选定）；零依赖、与 GraphQL 统一 |
| GitHub GraphQL（Discussion） | **标准库 `net/http` 手写** query | go-github 无 GraphQL；与 REST 共用基建 |
| Markdown 输出 | 不使用第三方库（手写 front matter + 原样 body） | 用户指令；spec §4.6/§6.2 |
| YAML front matter | 手写最小安全 writer | spec §4.6（不引 yaml 库） |
| 数据存储 | 无数据库，API 实时获取 | 用户指令；spec §1 |

### 1.2 依赖清单（go.mod `require`）
**零第三方依赖**（与 spec §6.2 一致）。`go.mod` 无任何 `require`。REST、GraphQL、Web、渲染、图片、YAML 全部使用标准库：
`net/http`、`net/url`、`encoding/json`、`flag`、`os`、`fmt`、`context`、`time`、`regexp`、`crypto/sha256`、`encoding/hex`、`strings`、`strconv`、`path/filepath`、`io`。

### 1.3 关键技术澄清（D-2）与路线 B 的统一性
- `google/go-github` 是 **REST v3** 客户端，**不含 GraphQL**。原指令「go-github 结合 GraphQL v4」无法成立。
- Discussion **必须**走 GraphQL v4（`https://api.github.com/graphql`）。
- **路线 B 决策**：REST 与 GraphQL **共用一套手写 HTTP 基建**（`baseURL`、`Authorization` 头、`User-Agent`、错误解析、`X-RateLimit-*` 解析、`httptest` 注入点）。REST 只是"再加 2-3 个端点 + Link 头分页"的边际增量，与 GraphQL 风格完全一致——避免了"两套 HTTP 体系"的割裂。

---

## 2. 合宪性审查（逐条对照 `constitution.md`）

| 条款 | 符合性 | 审查结论 |
| --- | --- | --- |
| **1.1 YAGNI** | ⚠️ 轻量修订 | `Reactions` 计数扩展 spec §4.4（D-3）。其余严格按 spec，不做 Enterprise/批量/重试等。 |
| **1.2 标准库优先** | ✅ **完全符合** | 路线 B 零依赖，全部标准库（D-1 撤销后无需修宪）。 |
| **1.3 反过度工程** | ✅ 符合（附论证） | 仅引入 **1 个**必要接口 `Source`（§5.1 论证）。所有数据结构保持**值对象**，无继承、无接口嵌套。 |
| **2.1 TDD 循环** | ✅ 符合 | 每模块从失败测试起步，Red→Green→Refactor（§7、§8）。 |
| **2.2 表格驱动** | ✅ 符合 | 全部单测 table-driven（§7.2）。 |
| **2.3 拒绝 Mock** | ✅ 符合 | REST/GraphQL 测试均用 `httptest` 起真实 HTTP server，真实标准库 client 打本地地址（§7.3），属真实依赖往返。 |
| **3.1 错误处理** | ✅ 符合 | 一律 `fmt.Errorf("...: %w", err)`；`*Error` 支持 `errors.Is/As`（骨架 errors.go 已落地）。 |
| **3.2 无全局变量** | ✅ 符合 | 依赖经 `Options`/结构体注入；client 非全局（骨架 main.go 用 `const version`）。 |
| **治理（宪法最高）** | ✅ 符合 | D-1 撤销后，方案无需修宪。仅 D-3/D-4 为 spec/API-sketch 的**轻量同步修订**（附录 A）。 |

**审查结论**：路线 B 下方案**完全合宪**；仅剩两处文档级轻量同步（Reactions 入 spec §4.4、Source 入 API-sketch §0）。

---

## 3. 项目结构细化（基于现有骨架）

### 3.1 包结构与职责
```
cmd/issue2md/         CLI 入口（薄封装）：flag→Options→Convert→退出码
internal/issue2md/    核心库（唯一业务落点）
  types.go            领域模型：Document/Comment/Reactions/PRInfo/Kind/Ref      [契约·M1补全]
  errors.go           错误模型：ErrorKind/Error + 哨兵                          [契约·已落地]
  source.go           【新增】Source 接口 + Ref + NewSource（D-4）               [契约·M1新增]
  convert.go          Options/Result + 顶层 Convert() 编排                       [契约·已落地]
  url.go              URL 解析与校验（→ Ref）
  github.go           REST 数据源实现（标准库 net/http 手写）   [共用 HTTP 基建]
  graphql.go          GraphQL 数据源实现（标准库手写，Discussion）[共用 HTTP 基建]
  httpc.go            【新增】共用 HTTP 基建：client/auth/错误/速率头/Link 分页
  fetch.go            编排：按 Ref 选 REST/GraphQL，组装 Document
  image.go            图片扫描/下载/去重/链接改写
  yaml.go             最小安全 YAML front matter writer
  render.go           Document → Markdown（含 front matter）
  *_test.go           每文件配套 table-driven 测试
web/                  未来 Web 服务（本期占位 README）
```
> 与骨架差异：新增 `source.go`（D-4）、`httpc.go`（抽出 REST/GraphQL 共用的 HTTP 基建，避免重复）。`github.go` 保持原名，承载 REST（标准库手写）。

### 3.2 依赖关系（单向，无环）
```
cmd/issue2md ──► internal/issue2md (Convert/Options/Result/Error)
                        │
                        ├──► source.go   (Source 接口 + Ref)
                        │       ▲ 实现
                        ├──► github.go   (Source: REST，依赖 httpc.go)
                        ├──► graphql.go  (Source: GraphQL，依赖 httpc.go)
                        │       │
                        │       └──► httpc.go (共用 HTTP 基建)
                        │
                        ├──► url.go      (parseURL → Ref)
                        ├──► fetch.go    (调度 Source + 组装 Document)
                        ├──► image.go    (图片本地化)
                        ├──► render.go   (Document → md) ──► yaml.go
                        └──► convert.go  (顶层编排，调用以上全部)
```
- **方向**：`convert`→`fetch`/`image`/`render`/`url`；`fetch`→`source`（接口）；`github`/`graphql` 实现 `source` 并共享 `httpc`。**接口定义在调用方**（依赖倒置），`fetch` 只见 `Source`。
- **依赖隔离**：`github.go`/`graphql.go` 是仅有的发 HTTP 处，均经 `httpc.go`；其余包零 HTTP 感知。

### 3.3 职责边界
| 包/文件 | 职责 | 不做 |
| --- | --- | --- |
| `convert.go` | 顶层编排、IO 落盘、组装 Result | 不直接发 HTTP |
| `fetch.go` | 选数据源、组装 Document | 不渲染、不落盘 |
| `source.go` | 定义 `Source` 接口、`Ref`、`NewSource` | 不含实现 |
| `github.go` | REST 调用、REST→Document 映射、Link 分页 | 不处理 GraphQL |
| `graphql.go` | Discussion GraphQL 查询、嵌套递归、cursor 分页 | 不处理 REST |
| `httpc.go` | 共用 client/auth/错误解析/速率头 | 不含业务映射 |
| `url.go` | URL→Ref 校验 | 不发 HTTP |
| `image.go` | 图片下载/去重/改写链接 | 不改 Document 语义字段 |
| `render.go`/`yaml.go` | 纯渲染（无副作用） | 不触网/不落盘 |

---

## 4. 核心数据结构（模块间流转）

> 模块间流转的**核心 struct 是 `Document`**（即需求语境的「IssueData」统一形态，覆盖 Issue/PR/Discussion）。在骨架 `types.go` 基础上**新增 `Reactions`/`Ref`**（M1 落地）。

### 4.1 领域模型
```go
// internal/issue2md/types.go

// Ref 定位一个 GitHub 讨论资源（URL 解析产物，Source 入参）。
type Ref struct {
    Kind   Kind
    Owner  string
    Repo   string
    Number int
}

// Document 是模块间流转的核心数据结构（Issue/PR/Discussion 统一）。
type Document struct {
    Kind       Kind
    Title      string
    URL        string
    Repository string // "owner/repo"
    Number     int
    Author     string
    State      string // open | closed | merged（仅 PR）
    CreatedAt  time.Time
    UpdatedAt  time.Time
    Labels     []string
    Body       string    // 正文原始 markdown
    Reactions  Reactions // 【D-3·spec §4.4 修订】各类计数
    Comments   []Comment
    PR         *PRInfo   // 非 PR 时为 nil
    FetchedAt  time.Time
}

// Comment 承载评论；Discussion 的 Replies 承载嵌套。
type Comment struct {
    Author    string
    CreatedAt time.Time
    Body      string
    Reactions Reactions   // 【D-3·spec §4.4 修订】
    Replies   []Comment   // 仅 Discussion 嵌套；Issue/PR 为 nil
}

// Reactions 聚合表态计数（D-3；不含用户明细）。
type Reactions struct {
    TotalCount int
    PlusOne    int
    MinusOne   int
    Laugh      int
    Hooray     int
    Confused   int
    Heart      int
    Rocket     int
    Eyes       int
}

// PRInfo 承载 PR 特有字段。
type PRInfo struct {
    Merged bool
    Base   string // 目标分支
    Head   string // 源分支
}
```

### 4.2 配置与结果（已在骨架，沿用）
```go
type Options struct {
    Token      string
    OutputPath string        // "" 自动 / "-" stdout / 路径
    NoImages   bool
    Timeout    time.Duration
    HTTPClient *http.Client  // 注入：REST 与 GraphQL 共用，便于 httptest
    BaseURL    string        // API 根，测试指向 httptest
}
type Result struct {
    Document   *Document
    Markdown   string
    OutputPath string
    ImageDir   string
    Warnings   []string
}
```

### 4.3 字段来源映射（spec §4.4 ↔ API）
| Document 字段 | Issue/PR 来源（REST JSON） | Discussion 来源（GraphQL） |
| --- | --- | --- |
| Title | `issue.title` / `pull_request.title` | `discussion.title` |
| Author | `issue.user.login` | `discussion.author.login` |
| Body | `issue.body` | `discussion.body` |
| State | `issue.state`；PR `pull_request.merged` | `discussion.closed` → closed/open |
| Labels | `issue.labels[].name` | `discussion.labels.nodes[].name` |
| Reactions | `issue.reactions.{total_count,+1,...}` | `discussion.reactions.{totalCount,...}` |
| Comments | `GET issues/{n}/comments`（Link 分页） | `discussion.comments.nodes` + `replies.nodes`（递归） |
| PR.Merged/Base/Head | `pull_request.{merged,base.ref,head.ref}` | — |

### 4.4 原始响应 struct（不导出，定义于 github.go / graphql.go）
- REST：`restIssue` / `restComment` / `restReactions` / `restUser` 等，带 `json` tag。
- GraphQL：`discussionResp`（嵌套 `repository.discussion` + `comments.nodes` + `replies` 递归 + `pageInfo`）。

---

## 5. 接口设计（internal 对外暴露）

### 5.1 设计原则（对齐宪法 1.3）
- **值对象仍是主体**：`Document`/`Options`/`Result` 等皆为 struct，非接口。
- **只引入「有多实现需求」的接口**：`Source` 有 REST 与 GraphQL 两套实现，且便于未来扩展（GitLab）与 httptest 注入 → 引入接口是**真需求**，非教条抽象（论证 D-4 合规）。
- **接口定义在调用方**（`source.go`），实现可替换；`fetch.go` 依赖接口而非具体类型。

### 5.2 核心接口 `Source`
```go
// internal/issue2md/source.go

// Source 抽象「获取一个 GitHub 讨论资源」的数据源。
// 默认实现 githubSource 在内部按 Ref.Kind 分发到 REST 或 GraphQL（均标准库手写）。
type Source interface {
    // Fetch 按 ref 抓取并组装为 Document；含分页与嵌套递归。
    Fetch(ctx context.Context, ref Ref) (*Document, error)
}

// NewSource 构造默认数据源（封装 REST + GraphQL 共用的标准库 client）。
// opts.HTTPClient/BaseURL 同时作用于 REST 与 GraphQL，便于 httptest 注入。
func NewSource(opts Options) Source
```

### 5.3 接口 vs 具体类型的边界
| 关注点 | 形态 | 理由 |
| --- | --- | --- |
| 数据获取 | **接口 `Source`** | REST/GraphQL 双实现 + 未来扩展 |
| 渲染 | **函数 `Render(*Document)`** | 单一实现，无需接口（API-sketch §A.7 已定） |
| 图片处理 | **函数** | 单一实现 |
| 配置/结果/领域模型 | **struct** | 数据，非行为多态 |
| 错误 | **类型 `*Error` + 哨兵** | 已落地，支持 Is/As |

> 即：全库**仅 `Source` 一个接口**。这是 1.3「反过度工程」下的克制设计。

### 5.4 顶层 API（不变，沿用 API-sketch）
```go
func Convert(ctx context.Context, src string, opts Options) (*Result, error)
```
`Convert` 内部 `NewSource(opts)` 构造默认源；高级用法可未来在 `Options` 增 `Source` 注入字段（本期不做，YAGNI）。

---

## 6. 关键流程

### 6.1 Convert 主编排
```
Convert(ctx, src, opts)
  ├─ ref, err := parseURL(src)            // §6.2；err→KindUsage
  ├─ src_ := NewSource(opts)              // REST+GraphQL 标准库 client
  ├─ doc, err := src_.Fetch(ctx, ref)     // §6.3；err→KindNotFound/Unauthorized/RateLimited/...
  ├─ md,  err := Render(doc)              // 纯函数；保留原始远程图片链接
  ├─ if OutputPath == "-" || NoImages:    // §4.5 stdout/禁图特例
  │      return Result{Markdown:md, ...}  // 不下载图片
  ├─ md2, dir := localizeImages(doc, md, opts)  // §6.4 下载+改写；失败仅 Warnings
  ├─ write(OutputPath, md2)               // 目录须存在，否则 KindIO
  └─ return Result{...}
```

### 6.2 URL 解析（url.go）
- `parseURL` → `{Kind, Owner, Repo, Number, error}`。
- 仅 `github.com`，`{issues|pull|discussions}/{number}`；忽略锚点/查询/末尾斜杠。
- 非法 → `&Error{Kind: KindUsage, ...}`（CLI 退出码 2）。

### 6.3 数据获取（fetch.go + source.go + github.go/graphql.go）
- `githubSource.Fetch` 按 `ref.Kind` 分发：
  - `Issue`/`Pull` → `github.go`（标准库）：主体 `GET /repos/{o}/{r}/issues/{n}`；评论 `GET /repos/{o}/{r}/issues/{n}/comments`，解析 `Link` 头分页（`per_page=100`，`page` 递增直到空）。PR 额外 `GET /repos/{o}/{r}/pulls/{n}` 取 merged/base/head。
  - `Discussion` → `graphql.go`：单次 query 取主体 + 首层 comments；逐层递归 `replies`；cursor 分页取全量 comments。
- HTTP 基建（`httpc.go`）：统一注入 `*http.Client`/baseURL/Token、设置 `Authorization`/`Accept`/`User-Agent`、解析 `X-RateLimit-*` 与错误响应体。
- 错误映射：404→`KindNotFound`，401→`KindUnauthorized`，403+速率→`KindRateLimited`（填 `ResetAt`），5xx→`KindServer`，网络→`KindNetwork`。

### 6.4 图片处理（image.go）
- 正则扫描 `![alt](url)` 与裸图 URL；按 URL 去重；命名 `{seq}-{sha256前8}.{ext}`，存 `{输出名}.files/`。
- 失败 → `Warnings`，保留原链接（spec §9.12）。stdout/NoImages → 跳过（spec §9.14）。

### 6.5 渲染（render.go + yaml.go）
- `Render` 产出：YAML front matter（手写、双引号转义）+ `# 标题` + 正文 + `## 评论`（Discussion 用 `####` 层级）。
- 纯函数；图片改写不在此（见 API-sketch §A.7 边界）。

### 6.6 错误映射
| 触发 | ErrorKind | 哨兵 | CLI 退出码 |
| --- | --- | --- | --- |
| URL/参数非法 | KindUsage | ErrInvalidURL | 2 |
| 写盘/目录缺失 | KindIO | ErrIO | 1 |
| 404 | KindNotFound | ErrNotFound | 3 |
| 401 | KindUnauthorized | ErrUnauthorized | 3 |
| 403 速率 | KindRateLimited | ErrRateLimited | 3 |
| 5xx | KindServer | ErrServerUnavailable | 3 |
| 网络/超时 | KindNetwork | ErrNetwork | 3 |

---

## 7. 测试策略落地

### 7.1 TDD 顺序（Red→Green→Refactor，逐模块）
`url` → `errors`(补 Is) → `httpc`(基建) → `yaml` → `render` → `source`(REST httptest) → `source`(GraphQL httptest) → `fetch` → `image` → `convert`(集成) → `cmd/issue2md`。

### 7.2 表格驱动（宪法 2.2）
全部单测 `[]struct{name, in, want}`；重点：`parseURL` 合法/非法全覆盖 §9.1–9.4；`yaml` 转义特殊字符；错误→退出码映射；`Link` 头分页解析。

### 7.3 真实依赖（宪法 2.3，拒绝 Mock）
- **REST/GraphQL**：`httptest.NewServer` 返回 fixture；构造的标准库 client（`httpc`）`baseURL` 指向 httptest，用**真实** `*http.Client` 打本地 server。校验收到的 URL/Header/body。
- **图片**：httptest 提供图片字节，断言落盘文件。
- 不引入任何接口级 mock（testify/mock 等）。

---

## 8. 实现里程碑（对齐任务 #1–#9）
| 里程碑 | 对应任务 | 交付 |
| --- | --- | --- |
| M1 | #1 | url 解析 + types/errors/source 契约补全（**含 Reactions/Ref/Source/`Is`**）；新增 `httpc.go` 骨架 |
| M2 | #2 | yaml front matter writer（表格驱动） |
| M3 | #3 | github.go 标准库 REST（Issue/PR + Link 分页）+ httptest |
| M4 | #4 | graphql.go（Discussion + 嵌套递归）+ httptest |
| M5 | #5 | fetch.go（Source 调度 + Document 组装） |
| M6 | #6 | image.go（下载/去重/改写） |
| M7 | #7 | render.go（front matter + 评论层级） |
| M8 | #8 | convert.go 顶层编排 + 端到端集成测试 |
| M9 | #9 | cmd/issue2md 接入 + README + 全量 `make test/vet` |

---

## 9. 风险与对策
| 风险 | 对策 |
| --- | --- |
| 手写 REST 的 json tag / Link 头出错 | 表格驱动测试 + httptest 固定 fixture 覆盖；`httpc.go` 集中处理 |
| GraphQL schema 变更 | 查询写在 graphql.go 常量；httptest fixture 保护 |
| Discussion 深层嵌套性能 | 递归 + 顺序拉取（spec §6.3 允许）；超时由 ctx 控制 |
| Reactions 推高体积 | 仅取计数（非用户明细），可控 |
| 速率限制（匿名 60/h） | 快速失败 + 清晰提示（spec §6.1），不重试 |

---

## 附录 A：spec / API-sketch 轻量同步修订（待执行）

> D-1 撤销后，**无需修宪**（宪法第一条、spec §6.2 零依赖保持不变）。仅以下两处文档级同步：

**A.1 `spec.md` §4.4**（内容深度 — Reactions 计数，D-3）
> 「取」清单新增 `reactions 各类计数（不含用户明细）`；评论含 `reactions 计数`。
> 「不取」清单将「reaction 明细」明确为「reaction **用户**明细」。

**A.2 `API-sketch.md` §0 第 2 条**（Source 接口例外，D-4）
> 增补：「**例外**：`Source` 接口因存在 REST/GraphQL 双实现与测试注入需求，允许引入；全库仅此一个接口。」

**A.3 `API-sketch.md` §A**（数据结构/导出清单同步）
> `Document`/`Comment` 增 `Reactions`；新增 `Reactions`、`Ref` 类型；导出清单加 `Source`/`Ref`/`Reactions`；新增 `Source` 接口小节。

**A.4 骨架代码（M1 执行，非文档）**
> `types.go` 加 `Reactions`/`Ref`、`Document`/`Comment` 加字段；新增 `source.go`（`Source`+`NewSource`）、`httpc.go`（HTTP 基建）。

---

## 附录 B：决策溯源
| 维度 | 决策 | 来源 |
| --- | --- | --- |
| REST 客户端 | **标准库手写（路线 B）** | 用户选定；零依赖、与 GraphQL 统一 |
| GraphQL | 标准库手写（D-2） | 技术纠正；go-github 无 GraphQL |
| Reactions | 纳入计数（D-3） | 用户指令；同步 spec §4.4 |
| 接口 | 引入唯一 Source（D-4） | 用户指令；论证合规 |
| 共用 HTTP 基建 | 抽出 `httpc.go` | 路线 B：REST/GraphQL 统一 |
| 测试 | httptest + 真实 client | 宪法 2.3 |
