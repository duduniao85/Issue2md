# issue2md API 设计草案 (API-sketch.md)

| 字段 | 值 |
| --- | --- |
| 版本 | 0.2 (Draft) |
| 日期 | 2026-06-20 |
| 关联 | [`spec.md`](./spec.md)（§7 架构引用本文）、[`constitution.md`](./constitution.md) |
| 状态 | A 节签名冻结（v0.2 增 `Source`/`Ref`/`Reactions`）；B 节方向性 sketch，待 Web 需求落地后定稿 |

> 本文是 issue2md **暴露接口**的权威设计，是核心设计的一部分。约定为 **sketch 级**：
> 给出类型签名、字段语义、不变量与错误契约；**不含实现逻辑**。任何签名级变更须先修订本文。

---

## 0. 设计原则（对齐宪法）

1. **最小公共面**：`internal/issue2md` 对外可见符号**仅本文 A 节列出**；其余一律小写未导出。
2. **值对象优先于 interface**（宪法 1.3 反过度工程）：核心数据用 `struct`，不为抽象而抽象。**例外**：`Source` 接口因存在 REST/GraphQL 双实现与测试注入需求，允许引入；全库仅此一个接口（见 §A.8）。
3. **依赖注入用具体类型**：注入 `*http.Client` 与 base URL（便于 `httptest`，无需 mock interface，契合宪法 2.3）。
4. **错误用类型 + 哨兵**：完整支持 `errors.Is` / `errors.As`（满足 spec AC-9）。
5. **向后兼容**：公共结构仅追加字段、不改签名；破坏性变更须升 spec 与本文版本。

---

## A. 核心库 API（`internal/issue2md`）— 本期实现

### A.1 导出符号清单
| 类别 | 符号 |
| --- | --- |
| 入口函数 | `Convert`, `Render` |
| 配置/结果 | `Options`, `Result` |
| 数据模型 | `Kind`, `Ref`, `Document`, `Comment`, `Reactions`, `PRInfo` |
| 数据源 | `Source`, `NewSource` |
| 错误 | `Error`, `ErrorKind`, `ErrInvalidURL`, `ErrNotFound`, `ErrUnauthorized`, `ErrRateLimited`, `ErrServerUnavailable`, `ErrNetwork`, `ErrIO` |

### A.2 顶层入口

```go
// Convert 解析 src（github.com 完整 URL），抓取内容、下载图片（除非禁用）、
// 渲染并写出 Markdown。成功返回 *Result；失败返回可被 errors.Is/As 识别的 *Error。
// ctx 控制整体生命周期与超时；opts.Timeout 作用于单次 HTTP 请求。
func Convert(ctx context.Context, src string, opts Options) (*Result, error)
```
**不变量**：
- 成功路径：`Result.Document != nil` 且 `Result.Markdown != ""`。
- `OutputPath == "-"` 时：不写盘、不下载图片，`Result.OutputPath == "-"`、`ImageDir == ""`。
- 任一图片下载失败仅记入 `Warnings`，**不**使 `Convert` 返回错误（部分成功，spec §4.5/9.12）。
- 速率限制 / 404 / 401 / 5xx / 网络错误 / URL 非法 / 写盘失败 → 返回对应 `*Error`（**不重试**，spec §6.1）。

### A.3 配置 `Options`

```go
type Options struct {
    // Token 为 GitHub PAT；空则匿名。调用方负责优先级合并（--token flag > $GITHUB_TOKEN）。
    Token string
    // OutputPath：
    //   ""  → 自动命名 {owner}-{repo}-{type}-{number}.md，写入当前工作目录
    //   "-" → 输出到 stdout（此时强制不下载图片）
    //   其他 → 写入该路径；目录须已存在，否则返回 KindIO 错误（spec 9.16，不自动创建）
    OutputPath string
    // NoImages 为 true 时跳过图片下载、保留远程链接（即使 OutputPath 非 "-"）。
    NoImages bool
    // Timeout 单次 HTTP 请求超时；零值默认 30s。
    Timeout time.Duration
    // HTTPClient 注入真实 *http.Client（便于测试）；nil 时库内构造默认 client。
    HTTPClient *http.Client
    // BaseURL 为 GitHub API 根，默认 "https://api.github.com"；测试可指向 httptest server。
    BaseURL string
}
```

### A.4 结果 `Result`

```go
type Result struct {
    Document  *Document // 成功路径非 nil
    Markdown  string    // 完整渲染结果（含 front matter）
    OutputPath string   // 实际写入路径；stdout 模式为 "-"
    ImageDir  string    // 图片目录路径；未下载图片时为 ""
    Warnings  []string  // 非致命警告（如个别图片下载失败）
}
```

### A.5 数据模型

```go
// Ref 定位一个 GitHub 讨论资源（URL 解析产物，Source 入参）。
type Ref struct {
    Kind   Kind
    Owner  string
    Repo   string
    Number int
}

type Kind string

const (
    KindIssue      Kind = "issue"
    KindPull       Kind = "pull"
    KindDiscussion Kind = "discussion"
)

type Document struct {
    Kind       Kind
    Title      string
    URL        string
    Repository string    // "owner/repo"
    Number     int
    Author     string
    State      string    // open | closed | merged（仅 PR）
    CreatedAt  time.Time
    UpdatedAt  time.Time
    Labels     []string
    Body       string    // 正文原始 markdown
    Reactions  Reactions // 各类计数（不含用户明细）
    Comments   []Comment
    PR         *PRInfo   // 非 PR 时为 nil
    FetchedAt  time.Time
}

type Comment struct {
    Author    string
    CreatedAt time.Time
    Body      string
    Reactions Reactions // 各类计数（不含用户明细）
    Replies   []Comment // 仅 Discussion 嵌套；Issue/PR 为 nil
}

type PRInfo struct {
    Merged bool
    Base   string // 目标分支
    Head   string // 源分支
}

// Reactions 聚合表态计数（不含用户明细，对齐 spec §4.4）。
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
```

### A.6 错误模型（满足 AC-9）

```go
type ErrorKind int

const (
    KindUsage ErrorKind = iota // 使用错误：参数缺失 / URL 非法
    KindIO                     // 本地 IO：写文件 / 目录不存在
    KindNotFound               // 404：不存在或私有未授权
    KindUnauthorized           // 401：Token 无效
    KindRateLimited            // 403 速率上限
    KindServer                 // 5xx
    KindNetwork                // 网络错误 / DNS / 超时
)

type Error struct {
    Kind    ErrorKind
    Op      string    // 触发位置，如 "parse url" / "fetch issue" / "write file"
    Message string    // 面向用户的一句话（Token 须脱敏）
    ResetAt time.Time // 仅 Kind==KindRateLimited 有意义
    Cause   error     // 底层错误，经 Unwrap 暴露
}

func (e *Error) Error() string  // 形如 "parse url: invalid github url: <cause>"
func (e *Error) Unwrap() error  // 返回 e.Cause（支持 errors.Is 链）
func (e *Error) Is(target error) bool // 按 Kind 命中对应哨兵

// 哨兵：支持 errors.Is(err, issue2md.ErrRateLimited) 等
var (
    ErrInvalidURL        = errors.New("invalid github url")
    ErrNotFound          = errors.New("not found or private")
    ErrUnauthorized      = errors.New("unauthorized: invalid token")
    ErrRateLimited       = errors.New("rate limited")
    ErrServerUnavailable = errors.New("github server error")
    ErrNetwork           = errors.New("network error")
    ErrIO                = errors.New("local io error")
)
```

**ErrorKind → CLI 退出码**（呼应 spec §5.3）

| ErrorKind | 哨兵 | 退出码 |
| --- | --- | --- |
| KindUsage | `ErrInvalidURL` | 2 |
| KindIO | `ErrIO` | 1 |
| KindNotFound | `ErrNotFound` | 3 |
| KindUnauthorized | `ErrUnauthorized` | 3 |
| KindRateLimited | `ErrRateLimited` | 3 |
| KindServer | `ErrServerUnavailable` | 3 |
| KindNetwork | `ErrNetwork` | 3 |

**使用示例**

```go
res, err := issue2md.Convert(ctx, url, opts)
switch {
case err == nil:
    // ok
case errors.Is(err, issue2md.ErrInvalidURL):
    os.Exit(2) // 用法错误
case errors.Is(err, issue2md.ErrRateLimited):
    var e *issue2md.Error
    _ = errors.As(err, &e)
    fmt.Fprintf(os.Stderr, "速率上限，重置于 %v", e.ResetAt)
    os.Exit(3)
}
```

### A.7 渲染（可独立复用）

```go
// Render 将 Document 渲染为带 YAML front matter 的 Markdown 字符串。
// 纯函数（不触网、不碰磁盘），Convert 内部调用它；未来 Web 也可直接复用。
func Render(doc *Document) (string, error)
```
> 图片链接改写发生在 `Convert` 的写出阶段（依赖 `ImageDir`），而非 `Render` 内——故 `Render` 输出保留**原始**远程图片链接；本地化由 `Convert` 在落盘前完成。这一边界保证 `Render` 无副作用、可被 Web 安全复用。

### A.8 数据源 `Source`

```go
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
> `Source` 是全库**唯一**接口：REST（Issue/PR）与 GraphQL（Discussion）双实现，且为未来扩展（如 GitLab）与 httptest 注入预留。调用方（`fetch.go`）只依赖此接口，不见 REST/GraphQL 细节（依赖倒置，契合宪法 1.3 的「必要抽象」，非教条回避接口）。

---

## B. 未来 Web API（预留 sketch — 本期不实现）

> 对应 spec §7.2。Web 上线时是**加法**：核心库零改动，仅新增 `web/` 下的 handler。

### B.1 设计原则
handler 是**薄层**，全部委托 `Convert`：
- HTTP 请求 → 构造 `Options`；
- `*Result` → 序列化为 HTTP 响应；
- `*Error` → 映射为 HTTP 状态码 + JSON 错误体。
handler 内**无**业务逻辑、**无**重复抓取/渲染代码。

### B.2 端点总览
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/health` | 存活检查 |
| POST | `/api/v1/convert` | 转换一个 URL |

### B.3 `POST /api/v1/convert`

**请求 DTO**

```go
type ConvertRequest struct {
    URL      string `json:"url"`                 // 必填，github.com 完整 URL
    Token    string `json:"token,omitempty"`     // 可选；亦可走 Authorization 头
    NoImages bool   `json:"no_images,omitempty"`
    // 无 output 字段：Web 始终在响应体返回内容，不写服务器磁盘
}
```

**响应 DTO（200）**

```go
type ConvertResponse struct {
    Markdown string   `json:"markdown"`
    Document Meta     `json:"document"`
    Warnings []string `json:"warnings,omitempty"`
}

type Meta struct {
    Kind       string `json:"kind"`
    Title      string `json:"title"`
    URL        string `json:"url"`
    Repository string `json:"repository"`
    Number     int    `json:"number"`
    Author     string `json:"author"`
    State      string `json:"state"`
    Comments   int    `json:"comments"`
}
```

**错误响应**：`{"error": "<message>", "kind": "<ErrorKind>"}`，HTTP 状态码按 ErrorKind 映射：

| ErrorKind | HTTP 状态码 |
| --- | --- |
| KindUsage | 400 |
| KindUnauthorized | 401 |
| KindNotFound | 404 |
| KindRateLimited | 429 |
| KindServer | 502 |
| KindNetwork | 502 |
| KindIO | 500（Web 不写盘，理论不应发生） |

### B.4 handler 伪代码（复用 `Convert`）

```go
func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
    var req ConvertRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, issue2md.ErrInvalidURL); return
    }
    opts := issue2md.Options{
        OutputPath: "-",                          // Web 取内容，不写盘
        Token:      firstNonEmpty(req.Token, tokenFromHeader(r)),
        NoImages:   req.NoImages,
    }
    res, err := issue2md.Convert(r.Context(), req.URL, opts)
    if err != nil {
        writeError(w, statusFor(err), err); return
    }
    json.NewEncoder(w).Encode(ConvertResponse{
        Markdown: res.Markdown, Warnings: res.Warnings, Document: metaOf(res.Document),
    })
}
```

### B.5 稳定性
- 路径前缀 `/api/v1/`，便于未来不兼容演进。
- B 节为**方向性 sketch**；Web 实现前将以独立 spec 章节定稿 DTO 与鉴权细节（如是否支持服务端配置默认 Token、速率保护等）。

---

## C. 版本与稳定性策略

- **核心库 v0（A 节）**：符号签名冻结；仅允许追加字段。破坏性变更须升 spec + 本文版本。
- **Web API（B 节）**：未实现，方向性 sketch；落地时定稿。
- 所有公共错误均经 `fmt.Errorf("...: %w", err)` 链式包装（宪法 3.1），保证 `errors.Is/As` 全链可达。
