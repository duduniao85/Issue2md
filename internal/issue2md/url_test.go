// url_test.go 验证 URL 解析，见 spec.md §4.1 / §9.1-9.4。表格驱动（宪法 2.2）。
package issue2md

import (
	"errors"
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantRef Ref
		wantErr error // 非 nil 时期望 errors.Is 命中
	}{
		// 合法
		{"issue", "https://github.com/owner/repo/issues/123", Ref{KindIssue, "owner", "repo", 123}, nil},
		{"pull", "https://github.com/owner/repo/pull/42", Ref{KindPull, "owner", "repo", 42}, nil},
		{"discussion", "https://github.com/owner/repo/discussions/7", Ref{KindDiscussion, "owner", "repo", 7}, nil},
		{"带锚点", "https://github.com/o/r/issues/1#issuecomment-99", Ref{KindIssue, "o", "r", 1}, nil},
		{"带查询", "https://github.com/o/r/issues/1?foo=bar", Ref{KindIssue, "o", "r", 1}, nil},
		{"末尾斜杠", "https://github.com/o/r/issues/1/", Ref{KindIssue, "o", "r", 1}, nil},
		{"http 协议", "http://github.com/o/r/issues/1", Ref{KindIssue, "o", "r", 1}, nil},
		{"大写域名", "https://GITHUB.com/o/r/issues/1", Ref{KindIssue, "o", "r", 1}, nil},
		{"owner/repo 含 . 和 -", "https://github.com/my-org/my.repo/issues/5", Ref{KindIssue, "my-org", "my.repo", 5}, nil},

		// 非法（期望 ErrInvalidURL）
		{"非 github 域名", "https://gitlab.com/o/r/issues/1", Ref{}, ErrInvalidURL},
		{"kind 错误", "https://github.com/o/r/wiki/1", Ref{}, ErrInvalidURL},
		{"number 非整数", "https://github.com/o/r/issues/abc", Ref{}, ErrInvalidURL},
		{"number 为 0", "https://github.com/o/r/issues/0", Ref{}, ErrInvalidURL},
		{"缺失 number", "https://github.com/o/r/issues/", Ref{}, ErrInvalidURL},
		{"路径不足", "https://github.com/o/r", Ref{}, ErrInvalidURL},
		{"空字符串", "", Ref{}, ErrInvalidURL},
		{"非 URL", "not a url", Ref{}, ErrInvalidURL},
		{"非 http(s) 协议", "ftp://github.com/o/r/issues/1", Ref{}, ErrInvalidURL},
		{"owner 含非法字符", "https://github.com/bad~name/repo/issues/1", Ref{}, ErrInvalidURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.src)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("parseURL(%q) err = nil, want non-nil", tt.src)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("parseURL(%q) err = %v, want errors.Is(%v)", tt.src, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseURL(%q) unexpected err: %v", tt.src, err)
			}
			if got != tt.wantRef {
				t.Errorf("parseURL(%q) = %+v, want %+v", tt.src, got, tt.wantRef)
			}
		})
	}
}
