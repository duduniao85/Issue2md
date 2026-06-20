// errors.go 定义核心库的错误模型，见 API-sketch.md §A.6。
// 支持 errors.Is（按哨兵）与 errors.As（取 *Error 细节）双识别（spec AC-9）。
package issue2md

import (
	"errors"
	"time"
)

// ErrorKind 分类错误，决定 CLI 退出码与未来 HTTP 状态码。
type ErrorKind int

const (
	KindUsage        ErrorKind = iota // 使用错误：参数缺失 / URL 非法
	KindIO                            // 本地 IO：写文件 / 目录不存在
	KindNotFound                      // 404：不存在或私有未授权
	KindUnauthorized                  // 401：Token 无效
	KindRateLimited                   // 403 速率上限
	KindServer                        // 5xx
	KindNetwork                       // 网络错误 / DNS / 超时
)

// Error 是核心库返回的错误类型。
type Error struct {
	Kind    ErrorKind
	Op      string    // 触发位置，如 "parse url" / "fetch issue" / "write file"
	Message string    // 面向用户的一句话（Token 须脱敏）
	ResetAt time.Time // 仅 Kind==KindRateLimited 有意义
	Cause   error     // 底层错误，经 Unwrap 暴露
}

// Error 返回基础错误文案。
func (e *Error) Error() string {
	if e.Op != "" {
		return e.Op + ": " + e.Message
	}
	return e.Message
}

// Unwrap 暴露底层 Cause，支持 errors.Is 链（宪法 3.1 %w 链式包装）。
func (e *Error) Unwrap() error { return e.Cause }

// Is 按 Kind 命中对应哨兵，使 errors.Is(err, issue2md.ErrRateLimited) 等可用。
func (e *Error) Is(target error) bool {
	switch e.Kind {
	case KindUsage:
		return target == ErrInvalidURL
	case KindIO:
		return target == ErrIO
	case KindNotFound:
		return target == ErrNotFound
	case KindUnauthorized:
		return target == ErrUnauthorized
	case KindRateLimited:
		return target == ErrRateLimited
	case KindServer:
		return target == ErrServerUnavailable
	case KindNetwork:
		return target == ErrNetwork
	}
	return false
}

// 哨兵错误：调用方可用 errors.Is 判别类别。
var (
	ErrInvalidURL        = errors.New("invalid github url")
	ErrNotFound          = errors.New("not found or private")
	ErrUnauthorized      = errors.New("unauthorized: invalid token")
	ErrRateLimited       = errors.New("rate limited")
	ErrServerUnavailable = errors.New("github server error")
	ErrNetwork           = errors.New("network error")
	ErrIO                = errors.New("local io error")
)
