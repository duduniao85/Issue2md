// httpc.go 提供共用 HTTP 基建，见 plan.md §3.3/§6.3。
// REST(github.go) 与 GraphQL(graphql.go) 共用此基建（路线 B：风格统一）。
package issue2md

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// httpClient 封装 GitHub API 的 HTTP 调用基建。依赖注入 *http.Client 与 base（宪法 3.2/2.3）。
type httpClient struct {
	base       string // API 根，默认 https://api.github.com
	token      string // 可选 PAT
	httpClient *http.Client
	userAgent  string
}

// newHTTPClient 依 Options 构造，填充默认值。
func newHTTPClient(opts Options) *httpClient {
	c := opts.HTTPClient
	if c == nil {
		c = &http.Client{}
	}
	base := opts.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	return &httpClient{base: base, token: opts.Token, httpClient: c, userAgent: "issue2md"}
}

// buildRequest 构造带标准头的请求（Authorization/Accept/User-Agent）。
func (c *httpClient) buildRequest(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

// do 发送请求；网络错误映射为 KindNetwork。不检查状态码（由 checkStatus 负责）。
func (c *httpClient) do(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := c.buildRequest(ctx, method, path)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Kind: KindNetwork, Op: "github api", Message: "network error", Cause: err}
	}
	return resp, nil
}

// checkStatus 将响应状态码映射到 *Error；2xx 返回 nil。
func checkStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return &Error{Kind: KindNotFound, Op: "github api", Message: "not found or private"}
	case resp.StatusCode == http.StatusUnauthorized:
		return &Error{Kind: KindUnauthorized, Op: "github api", Message: "invalid token"}
	case resp.StatusCode == http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return &Error{Kind: KindRateLimited, Op: "github api",
				Message: "rate limit exceeded", ResetAt: parseRateLimitReset(resp.Header.Get("X-RateLimit-Reset"))}
		}
		return &Error{Kind: KindUnauthorized, Op: "github api", Message: "forbidden"}
	case resp.StatusCode >= 500:
		return &Error{Kind: KindServer, Op: "github api", Message: "github server error"}
	default:
		return &Error{Kind: KindServer, Op: "github api",
			Message: fmt.Sprintf("unexpected status: %d", resp.StatusCode)}
	}
}

// parseNextLink 解析 Link 头中 rel="next" 的 URL；无则返回空。
func parseNextLink(link string) string {
	for _, part := range strings.Split(link, ",") {
		sec := strings.Split(strings.TrimSpace(part), ";")
		if len(sec) < 2 {
			continue
		}
		u := strings.Trim(strings.TrimSpace(sec[0]), "<>")
		for _, attr := range sec[1:] {
			if strings.Contains(strings.TrimSpace(attr), `rel="next"`) {
				return u
			}
		}
	}
	return ""
}

// parseRateLimitReset 解析 X-RateLimit-Reset（Unix 秒）为 time.Time；失败返回零值。
func parseRateLimitReset(s string) time.Time {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0)
}
