// httpc.go 提供共用 HTTP 基建，见 plan.md §3.3/§6.3。
// REST(github.go) 与 GraphQL(graphql.go) 共用此基建（路线 B：风格统一）。
package issue2md

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultHTTPTimeout 是 Options.Timeout 为零值时的回退（spec：单次请求默认 30s）。
const defaultHTTPTimeout = 30 * time.Second

// httpClient 封装 GitHub API 的 HTTP 调用基建。依赖注入 *http.Client 与 base（宪法 3.2/2.3）。
type httpClient struct {
	base       string // API 根，默认 https://api.github.com
	token      string // 可选 PAT
	httpClient *http.Client
	userAgent  string
}

// newHTTPClient 依 Options 构造，填充默认值；Timeout 零值回退 30s（宪法 §3.1：文档承诺须与实现一致）。
func newHTTPClient(opts Options) *httpClient {
	c := opts.HTTPClient
	if c == nil {
		c = &http.Client{}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	if c.Timeout == 0 { // 仅当调用方未显式设定时补默认
		c.Timeout = timeout
	}
	base := opts.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	return &httpClient{base: base, token: opts.Token, httpClient: c, userAgent: "issue2md"}
}

// requestSpec 描述一次 HTTP 请求：method、完整 URL、可选 body 与额外头。
// 标准头（Authorization/Accept/User-Agent）由 buildRequest 统一注入，避免散落多处。
type requestSpec struct {
	method  string
	url     string            // 完整 URL（base+path 或绝对图片地址）
	body    io.Reader         // nil 表示无 body
	headers map[string]string // 额外头（如 Content-Type: application/json）
}

// buildRequest 构造带标准头（Authorization/Accept/User-Agent）+ 额外头的请求。
func (c *httpClient) buildRequest(ctx context.Context, spec requestSpec) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, spec.method, spec.url, spec.body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	for k, v := range spec.headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// send 构造并发送请求；网络错误统一映射为 KindNetwork。不检查状态码（由 checkStatus 负责）。
func (c *httpClient) send(ctx context.Context, spec requestSpec) (*http.Response, error) {
	req, err := c.buildRequest(ctx, spec)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Kind: KindNetwork, Op: "github api", Message: "network error", Cause: err}
	}
	return resp, nil
}

// do 是 send 的便捷封装：用 base+path 构造请求（REST 读取）。
func (c *httpClient) do(ctx context.Context, method, path string) (*http.Response, error) {
	return c.send(ctx, requestSpec{method: method, url: c.base + path})
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
