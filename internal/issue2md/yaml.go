// yaml.go 手写最小安全 YAML front matter writer，见 spec.md §4.6 / §6.2。
// 不引入 yaml 第三方库（宪法 1.2）：字符串值双引号包裹并转义，数组用 "- item"。
package issue2md

import (
	"strconv"
	"strings"
	"time"
)

// frontMatter 是 front matter 字段集（render.go 从 Document 构造）。
type frontMatter struct {
	Title      string
	URL        string
	Type       string
	Repository string
	Number     int
	Author     string
	State      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Labels     []string
	Comments   int
	Source     string
	FetchedAt  time.Time
	PR         *PRInfo // 非 nil（PR）时渲染 pr: 块
}

// quoteYAML 将字符串用双引号包裹并转义特殊字符（YAML 双引号字符串）。
func quoteYAML(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// writeFrontMatter 生成 YAML front matter 块（以 --- 包裹）。
func writeFrontMatter(m frontMatter) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + quoteYAML(m.Title) + "\n")
	b.WriteString("url: " + quoteYAML(m.URL) + "\n")
	b.WriteString("type: " + quoteYAML(m.Type) + "\n")
	b.WriteString("repository: " + quoteYAML(m.Repository) + "\n")
	b.WriteString("number: " + strconv.Itoa(m.Number) + "\n")
	b.WriteString("author: " + quoteYAML(m.Author) + "\n")
	b.WriteString("state: " + quoteYAML(m.State) + "\n")
	b.WriteString("created_at: " + quoteYAML(m.CreatedAt.Format(time.RFC3339)) + "\n")
	b.WriteString("updated_at: " + quoteYAML(m.UpdatedAt.Format(time.RFC3339)) + "\n")
	if len(m.Labels) > 0 {
		b.WriteString("labels:\n")
		for _, l := range m.Labels {
			b.WriteString("  - " + quoteYAML(l) + "\n")
		}
	} else {
		b.WriteString("labels: []\n")
	}
	b.WriteString("comments: " + strconv.Itoa(m.Comments) + "\n")
	b.WriteString("source: " + quoteYAML(m.Source) + "\n")
	if m.PR != nil {
		b.WriteString("pr:\n")
		b.WriteString("  merged: " + strconv.FormatBool(m.PR.Merged) + "\n")
		b.WriteString("  base: " + quoteYAML(m.PR.Base) + "\n")
		b.WriteString("  head: " + quoteYAML(m.PR.Head) + "\n")
	}
	b.WriteString("fetched_at: " + quoteYAML(m.FetchedAt.Format(time.RFC3339)) + "\n")
	b.WriteString("---\n")
	return b.String()
}
