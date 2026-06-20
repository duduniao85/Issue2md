// Package issue2md 提供 issue2md 的核心转换能力：将 GitHub Issue/PR/Discussion
// 转换为带 YAML front matter 的本地 Markdown。
//
// 本包是 CLI（cmd/issue2md）与未来 Web 服务（web/）共享的核心库，所有业务逻辑集中于此。
// 公共接口签名以 API-sketch.md 为准。
package issue2md

import "time"

// 本文件 (types.go) 定义核心数据模型，见 API-sketch.md §A.5。

// Ref 定位一个 GitHub 讨论资源（URL 解析产物，Source 入参）。
type Ref struct {
	Kind   Kind
	Owner  string
	Repo   string
	Number int
}

// Kind 标识 GitHub 内容类型。
type Kind string

const (
	KindIssue      Kind = "issue"
	KindPull       Kind = "pull"
	KindDiscussion Kind = "discussion"
)

// Document 是一次转换的结构化结果，承载元数据、正文与全部评论。
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
	Reactions  Reactions // 各类计数（不含用户明细）
	Comments   []Comment
	PR         *PRInfo // 非 PR 时为 nil
	FetchedAt  time.Time
}

// Comment 是一条评论；Discussion 的 Replies 承载嵌套回复。
type Comment struct {
	Author    string
	CreatedAt time.Time
	Body      string
	Reactions Reactions // 各类计数（不含用户明细）
	Replies   []Comment // 仅 Discussion 嵌套；Issue/PR 为 nil
}

// PRInfo 承载 PR 特有字段；仅 Document.Kind == KindPull 时非 nil。
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
