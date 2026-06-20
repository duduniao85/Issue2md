# web/ — 未来 Web 服务（预留，本期未实现）

> 对应 spec.md §7.2 与 API-sketch.md §B。本目录当前仅占位，不含任何 Go 代码。

## 状态
issue2md 本期为**纯 CLI**。本目录为未来 Web 服务预留位置
（YAGNI：预留位置与契约，不实现功能）。

## 设计原则
Web 服务是 `internal/issue2md` 之上的**薄层**，全部委托核心库 `Convert`：
- HTTP 请求 → 构造 `Options`；
- `*Result` → 序列化为 JSON 响应；
- `*Error` → 映射为 HTTP 状态码。

handler 内无业务逻辑，核心库零改动即可上线（加法而非重构）。

## 预留文件（Web 落地时新增）
- `server.go` — `net/http` handler 层与路由（宪法 1.2 标准库）。
- `api.go` — HTTP API 契约 / DTO（`ConvertRequest` / `ConvertResponse`，见 API-sketch.md §B.3）。

## 端点（sketch）
- `GET  /api/v1/health`  — 存活检查
- `POST /api/v1/convert` — 转换一个 URL

详见 API-sketch.md §B。待 Web 需求明确后，将以独立 spec 章节定稿。
