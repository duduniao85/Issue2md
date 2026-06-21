// httpc_test.go 验证共用 HTTP 基建，见 plan.md §3.3/§6.3。表格驱动 + httptest（宪法 2.2/2.3）。
package issue2md

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	c := newHTTPClient(Options{Token: "abc", BaseURL: "http://example.test"})
	if c.base != "http://example.test" {
		t.Errorf("base = %q, want http://example.test", c.base)
	}
	if c.token != "abc" {
		t.Errorf("token = %q, want abc", c.token)
	}
	// 默认 base 与 client
	c2 := newHTTPClient(Options{})
	if c2.base != "https://api.github.com" {
		t.Errorf("default base = %q, want https://api.github.com", c2.base)
	}
	if c2.httpClient == nil {
		t.Error("httpClient should be non-nil by default")
	}
}

func TestBuildRequest_Headers(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantAuth    string
		wantAuthSet bool
	}{
		{"with token", "tok123", "Bearer tok123", true},
		{"no token", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newHTTPClient(Options{Token: tt.token, BaseURL: "http://example.test"})
			req, err := c.buildRequest(context.Background(), requestSpec{
				method: http.MethodGet,
				url:    "http://example.test/repos/o/r/issues/1",
			})
			if err != nil {
				t.Fatalf("buildRequest err: %v", err)
			}
			if got := req.Header.Get("Accept"); got != "application/vnd.github+json" {
				t.Errorf("Accept = %q, want application/vnd.github+json", got)
			}
			if got := req.Header.Get("User-Agent"); got == "" {
				t.Error("User-Agent should be set")
			}
			gotAuth := req.Header.Get("Authorization")
			if tt.wantAuthSet && gotAuth != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", gotAuth, tt.wantAuth)
			}
			if !tt.wantAuthSet && gotAuth != "" {
				t.Errorf("Authorization = %q, want empty", gotAuth)
			}
		})
	}
}

func TestCheckStatus(t *testing.T) {
	const resetEpoch = "1700000000"
	tests := []struct {
		name       string
		statusCode int
		rateRemain string // X-RateLimit-Remaining
		wantKind   ErrorKind
		wantNil    bool
		wantReset  bool
	}{
		{"200 ok", 200, "", 0, true, false},
		{"299 ok", 299, "", 0, true, false},
		{"404", 404, "", KindNotFound, false, false},
		{"401", 401, "", KindUnauthorized, false, false},
		{"403 rate limit", 403, "0", KindRateLimited, false, true},
		{"403 forbidden", 403, "100", KindUnauthorized, false, false},
		{"500", 500, "", KindServer, false, false},
		{"503", 503, "", KindServer, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			if tt.rateRemain != "" {
				resp.Header.Set("X-RateLimit-Remaining", tt.rateRemain)
				resp.Header.Set("X-RateLimit-Reset", resetEpoch)
			}
			err := checkStatus(resp)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("checkStatus(%d) err = %v, want nil", tt.statusCode, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkStatus(%d) err = nil, want non-nil", tt.statusCode)
			}
			e, ok := err.(*Error)
			if !ok {
				t.Fatalf("checkStatus(%d) err type %T, want *Error", tt.statusCode, err)
			}
			if e.Kind != tt.wantKind {
				t.Errorf("checkStatus(%d) Kind = %v, want %v", tt.statusCode, e.Kind, tt.wantKind)
			}
			if tt.wantReset && e.ResetAt.IsZero() {
				t.Errorf("checkStatus(%d) ResetAt is zero, want set", tt.statusCode)
			}
		})
	}
}

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name string
		link string
		want string
	}{
		{"has next", `<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=1>; rel="prev"`, "https://api.github.com/x?page=2"},
		{"only next", `<https://api.github.com/y?page=5>; rel="next"`, "https://api.github.com/y?page=5"},
		{"no next", `<https://api.github.com/z?page=1>; rel="prev"`, ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNextLink(tt.link); got != tt.want {
				t.Errorf("parseNextLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDo_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Close() // 关闭制造网络错误

	c := newHTTPClient(Options{BaseURL: base})
	_, err := c.do(context.Background(), http.MethodGet, "/repos/o/r/issues/1")
	if err == nil {
		t.Fatal("do() err = nil, want non-nil (network error)")
	}
	if !errors.Is(err, ErrNetwork) {
		t.Errorf("do() err = %v, want errors.Is(ErrNetwork)", err)
	}
}

func TestDo_CheckStatus_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL, Token: "t"})
	resp, err := c.do(context.Background(), http.MethodGet, "/repos/o/r/issues/1")
	if err != nil {
		t.Fatalf("do() unexpected err: %v", err)
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); !errors.Is(err, ErrNotFound) {
		t.Errorf("checkStatus(404) = %v, want errors.Is(ErrNotFound)", err)
	}
}

// TestDo_Timeout 验证 Options.Timeout 生效（零值默认 30s；此处显式设短超时）。
// 服务端故意延迟响应，超时后请求应映射为 KindNetwork（宪法 §3.1：文档承诺须与实现一致）。
func TestDo_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond) // 远大于客户端超时
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL, Timeout: 50 * time.Millisecond})
	_, err := c.do(context.Background(), http.MethodGet, "/repos/o/r/issues/1")
	if err == nil {
		t.Fatal("do() err = nil, want non-nil (timeout)")
	}
	if !errors.Is(err, ErrNetwork) {
		t.Errorf("do() err = %v, want errors.Is(ErrNetwork)", err)
	}
}
