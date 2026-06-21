// source.go 定义默认数据源类型 githubSource 与构造函数 NewSource。
//
// 当前为具体类型而非接口：仅一个实现，且 Convert 不接受外部注入、测试用 httptest
// 打真实实现（宪法 §1.3 反过度工程）。待确有第二数据源（如 GitLab）时再提取接口。
// Fetch 分发逻辑与各 Kind 的获取函数位于 fetch.go。
package issue2md

// githubSource 是默认数据源：封装 REST(Issue/PR) + GraphQL(Discussion) 共用的 httpClient。
type githubSource struct {
	c *httpClient
}

// NewSource 构造默认数据源。
func NewSource(opts Options) *githubSource {
	return &githubSource{c: newHTTPClient(opts)}
}
