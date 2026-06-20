// yaml_test.go 验证 YAML front matter writer，见 spec.md §4.6。表格驱动（宪法 2.2）。
package issue2md

import (
	"strings"
	"testing"
	"time"
)

func TestQuoteYAML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", `"hello"`},
		{"含双引号", `a"b`, `"a\"b"`},
		{"含反斜杠", `a\b`, `"a\\b"`},
		{"含换行", "a\nb", `"a\nb"`},
		{"含冒号", "with: colon", `"with: colon"`},
		{"空", "", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteYAML(tt.in); got != tt.want {
				t.Errorf("quoteYAML(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWriteFrontMatter(t *testing.T) {
	m := frontMatter{
		Title:      "Bug: 崩溃",
		URL:        "https://github.com/o/r/issues/1",
		Type:       "issue",
		Repository: "o/r",
		Number:     1,
		Author:     "alice",
		State:      "open",
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Labels:     []string{"bug", "help wanted"},
		Comments:   5,
		Source:     "issue2md v0.1",
		FetchedAt:  time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	out := writeFrontMatter(m)

	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("应以 --- 开头，got prefix %q", out[:min(len(out), 10)])
	}
	if !strings.HasSuffix(out, "---\n") {
		t.Errorf("应以 --- 结尾")
	}
	mustContain := []string{
		`title: "Bug: 崩溃"`,
		`type: "issue"`,
		`number: 1`,
		`author: "alice"`,
		`state: "open"`,
		`comments: 5`,
		`  - "bug"`,
		`  - "help wanted"`,
		`created_at: "2024-01-01T00:00:00Z"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("输出缺失 %q\n--- 完整输出 ---\n%s", s, out)
		}
	}
}

func TestWriteFrontMatter_NoLabels(t *testing.T) {
	m := frontMatter{Title: "x", Labels: nil}
	out := writeFrontMatter(m)
	if !strings.Contains(out, "labels: []") {
		t.Errorf("无标签应为 labels: []，输出:\n%s", out)
	}
}

func TestWriteFrontMatter_PR(t *testing.T) {
	m := frontMatter{
		Title: "PR",
		Type:  "pull",
		PR:    &PRInfo{Merged: true, Base: "main", Head: "feature"},
	}
	out := writeFrontMatter(m)
	for _, s := range []string{
		"pr:",
		"  merged: true",
		`  base: "main"`,
		`  head: "feature"`,
	} {
		if !strings.Contains(out, s) {
			t.Errorf("输出缺失 %q\n%s", s, out)
		}
	}
}

func TestWriteFrontMatter_NoPR(t *testing.T) {
	m := frontMatter{Title: "x", Type: "issue"} // PR == nil
	out := writeFrontMatter(m)
	if strings.Contains(out, "pr:") {
		t.Errorf("非 PR 不应输出 pr: 块\n%s", out)
	}
}
