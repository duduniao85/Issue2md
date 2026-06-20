// source.go 定义数据源接口 Source，见 API-sketch.md §A.8。
//
// Source 是全库唯一接口：统一 REST（Issue/PR）与 GraphQL（Discussion）数据源，
// 便于未来扩展（如 GitLab）与 httptest 注入（宪法 1.3「必要抽象」，宪法 2.3「真实依赖」）。
// 默认实现 githubSource 与构造函数 NewSource 在 fetch.go（Phase 2 落地）。
package issue2md

import "context"

// Source 抽象「获取一个 GitHub 讨论资源」的数据源。
type Source interface {
	// Fetch 按 ref 抓取并组装为 Document；含分页与嵌套递归。
	Fetch(ctx context.Context, ref Ref) (*Document, error)
}
