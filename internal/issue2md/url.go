// url.go 解析与校验 github.com 完整 URL，见 spec.md §4.1 / §9.1-9.4。
package issue2md

import (
	"net/url"
	"strconv"
	"strings"
)

// parseURL 解析 github.com 完整 URL，返回定位 Ref。
//
// 仅接受 {http,https}://github.com/{owner}/{repo}/{issues|pull|discussions}/{number}；
// owner/repo 仅允许 [A-Za-z0-9._-]+；number 为正整数；
// 忽略锚点/查询串/末尾斜杠。非法返回 KindUsage 错误（errors.Is 命中 ErrInvalidURL）。
func parseURL(src string) (Ref, error) {
	u, err := url.Parse(src)
	if err != nil || u.Host == "" {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "invalid url: " + src}
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "only github.com is supported, got: " + u.Host}
	}
	switch scheme := strings.ToLower(u.Scheme); scheme {
	case "http", "https":
		// ok
	default:
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "only http/https is supported, got: " + u.Scheme}
	}

	// 路径分段：/{owner}/{repo}/{kind}/{number}
	seg := splitPath(u.Path)
	if len(seg) != 4 {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url",
			Message: "expected /{owner}/{repo}/{issues|pull|discussions}/{number}"}
	}
	owner, repo, kindStr, numStr := seg[0], seg[1], seg[2], seg[3]

	if !validName(owner) || !validName(repo) {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "invalid owner or repo: " + owner + "/" + repo}
	}
	kind, ok := parseKind(kindStr)
	if !ok {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "unsupported kind: " + kindStr}
	}
	number, err := strconv.Atoi(numStr)
	if err != nil || number <= 0 {
		return Ref{}, &Error{Kind: KindUsage, Op: "parse url", Message: "invalid number: " + numStr}
	}
	return Ref{Kind: kind, Owner: owner, Repo: repo, Number: number}, nil
}

// parseKind 将 URL 路径段映射到 Kind。
func parseKind(s string) (Kind, bool) {
	switch s {
	case "issues":
		return KindIssue, true
	case "pull":
		return KindPull, true
	case "discussions":
		return KindDiscussion, true
	}
	return "", false
}

// splitPath 将 "/a/b/c/d" 切为 ["a","b","c","d"]，丢弃空段。
func splitPath(p string) []string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	out := parts[:0]
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// validName 校验 owner/repo：非空且仅 [A-Za-z0-9._-]（spec §4.1）。
func validName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
