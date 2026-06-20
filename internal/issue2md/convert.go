// convert.go 定义顶层入口 Convert 与配置/结果类型，见 API-sketch.md §A.2-A.4 / plan.md §6.1。
package issue2md

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options 聚合 Convert 的全部可配置项。
type Options struct {
	Token      string
	OutputPath string        // "" 自动命名 / "-" stdout / 其他 写入该路径
	NoImages   bool          // true 跳过图片下载，保留远程链接
	Timeout    time.Duration // 单次 HTTP 请求超时；零值默认 30s
	HTTPClient *http.Client  // 注入真实 client（便于测试）；nil 时库内构造
	BaseURL    string        // GitHub API 根，默认 https://api.github.com
}

// Result 是 Convert 的成功返回。
type Result struct {
	Document   *Document
	Markdown   string
	OutputPath string
	ImageDir   string
	Warnings   []string
}

// Convert 解析 src（github.com 完整 URL），抓取内容、下载图片（除非禁用）、
// 渲染并写出 Markdown。失败返回可被 errors.Is/As 识别的 *Error。
func Convert(ctx context.Context, src string, opts Options) (*Result, error) {
	ref, err := parseURL(src)
	if err != nil {
		return nil, err
	}
	source := NewSource(opts)
	doc, err := source.Fetch(ctx, ref)
	if err != nil {
		return nil, err
	}
	doc.FetchedAt = time.Now()

	md, err := Render(doc)
	if err != nil {
		return nil, err
	}

	path, imageDir, linkPrefix := resolveOutputPath(ref, opts)

	// stdout：不写盘、不下载图片
	if path == "-" {
		return &Result{Document: doc, Markdown: md, OutputPath: "-"}, nil
	}

	// 写文件：输出目录须已存在（spec §9.16，不自动创建）
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			return nil, &Error{Kind: KindIO, Op: "convert", Message: "output dir not exist: " + dir}
		}
	}

	var warnings []string
	if !opts.NoImages {
		client := newHTTPClient(opts)
		md, warnings, err = localizeImages(ctx, client, md, imageDir, linkPrefix)
		if err != nil {
			return nil, err
		}
	}

	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return nil, &Error{Kind: KindIO, Op: "convert", Message: "write file failed: " + path, Cause: err}
	}

	res := &Result{Document: doc, Markdown: md, OutputPath: path, Warnings: warnings}
	if !opts.NoImages {
		res.ImageDir = imageDir
	}
	return res, nil
}

// resolveOutputPath 决定输出文件路径、图片目录与 md 内链接前缀。
func resolveOutputPath(ref Ref, opts Options) (path, imageDir, linkPrefix string) {
	switch {
	case opts.OutputPath == "-":
		return "-", "", ""
	case opts.OutputPath == "":
		base := fmt.Sprintf("%s-%s-%s-%d", ref.Owner, ref.Repo, ref.Kind, ref.Number)
		dir := base + ".files"
		return base + ".md", dir, "./" + dir + "/"
	default:
		base := strings.TrimSuffix(opts.OutputPath, filepath.Ext(opts.OutputPath))
		dir := base + ".files"
		return opts.OutputPath, dir, "./" + filepath.Base(dir) + "/"
	}
}
