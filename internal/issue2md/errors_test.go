// errors_test.go 验证错误模型行为，见 API-sketch.md §A.6。
// 表格驱动（宪法 2.2）；覆盖 Error()/Unwrap()/Is() 与 ResetAt 携带。
package issue2md

import (
	"errors"
	"testing"
	"time"
)

func TestError_Text(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{"op+message", &Error{Op: "parse url", Message: "invalid"}, "parse url: invalid"},
		{"message only", &Error{Message: "bad"}, "bad"},
		{"empty", &Error{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	inner := errors.New("root cause")
	wrapped := &Error{Kind: KindNetwork, Op: "fetch", Message: "timeout", Cause: inner}
	if !errors.Is(wrapped, inner) {
		t.Errorf("errors.Is(wrapped, inner) = false, want true（Unwrap 链应可达底层）")
	}
	if u := wrapped.Unwrap(); u != inner {
		t.Errorf("Unwrap() = %v, want %v", u, inner)
	}
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		name   string
		kind   ErrorKind
		target error
		want   bool
	}{
		// 正向：每个 Kind 命中其对应哨兵（API-sketch §A.6 表）
		{"usage→ErrInvalidURL", KindUsage, ErrInvalidURL, true},
		{"io→ErrIO", KindIO, ErrIO, true},
		{"notfound→ErrNotFound", KindNotFound, ErrNotFound, true},
		{"unauthorized→ErrUnauthorized", KindUnauthorized, ErrUnauthorized, true},
		{"ratelimited→ErrRateLimited", KindRateLimited, ErrRateLimited, true},
		{"server→ErrServerUnavailable", KindServer, ErrServerUnavailable, true},
		{"network→ErrNetwork", KindNetwork, ErrNetwork, true},

		// 交叉否定：不同 Kind 不命中其他哨兵
		{"notfound↮ErrRateLimited", KindNotFound, ErrRateLimited, false},
		{"ratelimited↮ErrNotFound", KindRateLimited, ErrNotFound, false},
		{"usage↮ErrIO", KindUsage, ErrIO, false},

		// 非哨兵目标
		{"network↮nil", KindNetwork, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Error{Kind: tt.kind}
			if got := errors.Is(e, tt.target); got != tt.want {
				t.Errorf("errors.Is(Kind=%v, target=%v) = %v, want %v", tt.kind, tt.target, got, tt.want)
			}
		})
	}
}

// TestError_ResetAt 验证 RateLimited 错误携带重置时间（供 CLI 提示）。
func TestError_ResetAt(t *testing.T) {
	reset := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	e := &Error{Kind: KindRateLimited, ResetAt: reset}
	if !e.ResetAt.Equal(reset) {
		t.Errorf("ResetAt = %v, want %v", e.ResetAt, reset)
	}
}
