// render.go 将 Document 渲染为带 YAML front matter 的 Markdown，见 spec.md §4.6 / API-sketch.md §A.7。
// Render 为纯函数（不触网、不碰盘）；图片本地化在 Convert 落盘阶段完成。
package issue2md

import (
	"strings"
	"time"
)

// sourceVersion 写入 front matter 的 source 字段。
const sourceVersion = "issue2md v0.1"

// Render 将 Document 渲染为 Markdown 字符串。
func Render(doc *Document) (string, error) {
	fm := frontMatter{
		Title:      doc.Title,
		URL:        doc.URL,
		Type:       string(doc.Kind),
		Repository: doc.Repository,
		Number:     doc.Number,
		Author:     doc.Author,
		State:      doc.State,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
		Labels:     doc.Labels,
		Comments:   len(doc.Comments),
		Source:     sourceVersion,
		FetchedAt:  doc.FetchedAt,
		PR:         doc.PR,
	}
	var b strings.Builder
	b.WriteString(writeFrontMatter(fm))
	b.WriteString("\n# " + doc.Title + "\n\n")
	b.WriteString(doc.Body)
	b.WriteString("\n")
	if len(doc.Comments) > 0 {
		b.WriteString("\n---\n\n## 评论\n\n")
		for _, c := range doc.Comments {
			writeComment(&b, c, 3)
		}
	}
	return b.String(), nil
}

// writeComment 递归写评论；level 为标题层级（顶层 ###，嵌套 ####+）。
func writeComment(b *strings.Builder, c Comment, level int) {
	prefix := strings.Repeat("#", level)
	b.WriteString(prefix + " @" + c.Author + " · " + c.CreatedAt.Format(time.RFC3339) + "\n\n")
	b.WriteString(c.Body)
	b.WriteString("\n")
	for _, r := range c.Replies {
		writeComment(b, r, level+1)
	}
}
