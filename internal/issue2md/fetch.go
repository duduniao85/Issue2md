// fetch.go 编排数据获取，见 plan.md §6.3。
// 持有 httpc 客户端，按 Ref.Kind 分发到 REST(Issue/PR) 或 GraphQL(Discussion)。
package issue2md

import "context"

// githubSource 是 Source 的默认实现。
type githubSource struct {
	c *httpClient
}

// NewSource 构造默认数据源（封装 REST + GraphQL 共用的 httpc 客户端）。
func NewSource(opts Options) Source {
	return &githubSource{c: newHTTPClient(opts)}
}

// Fetch 按 ref.Kind 分发：Issue/PR→REST，Discussion→GraphQL。
func (s *githubSource) Fetch(ctx context.Context, ref Ref) (*Document, error) {
	switch ref.Kind {
	case KindIssue:
		return fetchIssue(ctx, s.c, ref)
	case KindPull:
		return fetchPull(ctx, s.c, ref)
	case KindDiscussion:
		return fetchDiscussion(ctx, s.c, ref)
	}
	return nil, &Error{Kind: KindUsage, Op: "fetch", Message: "unsupported kind: " + string(ref.Kind)}
}
