// render_test.go 验证 Document → Markdown 渲染，见 spec.md §4.6 / API-sketch §A.7。
package issue2md

import (
	"strings"
	"testing"
	"time"
)

func TestRender_Issue(t *testing.T) {
	doc := &Document{
		Kind:       KindIssue,
		Title:      "Bug 标题",
		URL:        "https://github.com/o/r/issues/1",
		Repository: "o/r",
		Number:     1,
		Author:     "alice",
		State:      "open",
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Labels:     []string{"bug"},
		Body:       "这是正文",
		Comments: []Comment{
			{Author: "bob", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Body: "评论1"},
		},
		FetchedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	out, err := Render(doc)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	mustContain := []string{
		`title: "Bug 标题"`,
		`type: "issue"`,
		"# Bug 标题",
		"这是正文",
		"## 评论",
		"### @bob",
		"评论1",
		`source: "issue2md v0.1"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("输出缺失 %q\n--- 输出 ---\n%s", s, out)
		}
	}
}

func TestRender_DiscussionNested(t *testing.T) {
	doc := &Document{
		Kind:  KindDiscussion,
		Title: "D",
		Body:  "DB",
		Comments: []Comment{
			{Author: "u1", Body: "c1", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Replies: []Comment{
					{Author: "u2", Body: "r1", CreatedAt: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)},
				}},
		},
		FetchedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	out, err := Render(doc)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "### @u1") {
		t.Errorf("顶层评论应为 ### @u1\n%s", out)
	}
	if !strings.Contains(out, "#### @u2") {
		t.Errorf("嵌套回复应为 #### @u2\n%s", out)
	}
	if !strings.Contains(out, "r1") {
		t.Errorf("应含回复正文 r1\n%s", out)
	}
}

func TestRender_CodeFence(t *testing.T) {
	doc := &Document{
		Kind:      KindIssue,
		Title:     "T",
		Body:      "before\n```\ncode block\n```\nafter",
		FetchedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	out, err := Render(doc)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "```\ncode block\n```") {
		t.Errorf("代码围栏应原样保留\n%s", out)
	}
}

func TestRender_NoComments(t *testing.T) {
	doc := &Document{Kind: KindIssue, Title: "T", Body: "B", FetchedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)}
	out, err := Render(doc)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "## 评论") {
		t.Errorf("无评论不应输出 ## 评论\n%s", out)
	}
}

func TestRender_PR(t *testing.T) {
	doc := &Document{
		Kind:      KindPull,
		Title:     "PT",
		Body:      "PB",
		PR:        &PRInfo{Merged: true, Base: "main", Head: "feature"},
		FetchedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	out, err := Render(doc)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, s := range []string{
		`type: "pull"`,
		"pr:",
		"  merged: true",
		`  base: "main"`,
		`  head: "feature"`,
	} {
		if !strings.Contains(out, s) {
			t.Errorf("PR front matter 缺失 %q\n%s", s, out)
		}
	}
}
