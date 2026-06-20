// Command issue2md 将 GitHub Issue/PR/Discussion 转换为带 YAML front matter 的本地 Markdown。
//
// 本入口为薄封装：仅做 flag 解析、参数校验与退出码映射；核心逻辑在 internal/issue2md。
// CLI 接口见 spec.md §5，退出码映射见 API-sketch.md §A.6。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/duduniao85/issue2md/internal/issue2md"
)

// version 写入 -v 输出与 front matter source（与 render.go sourceVersion 一致）。
const version = "issue2md v0.1"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run 解析 args、执行一次转换，返回进程退出码。拆出以便测试（不调 os.Exit）。
func run(args []string) int {
	fs := flag.NewFlagSet("issue2md", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: issue2md [flags] <github-url>")
		fmt.Fprintln(os.Stderr, "flags:")
		fs.PrintDefaults()
	}
	output := fs.String("o", "", "输出文件路径；-o - 输出到 stdout")
	token := fs.String("token", "", "GitHub Token（默认读 $GITHUB_TOKEN）")
	noImages := fs.Bool("no-images", false, "不下载图片，保留远程链接")
	timeout := fs.Duration("timeout", 30*time.Second, "单次 HTTP 请求超时")
	baseURL := fs.String("baseurl", "", "GitHub API 根（高级/测试，默认 https://api.github.com）")
	showVersion := fs.Bool("v", false, "打印版本并退出")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println(version)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	opts := issue2md.Options{
		Token:      firstNonEmpty(*token, os.Getenv("GITHUB_TOKEN")),
		OutputPath: *output,
		NoImages:   *noImages,
		Timeout:    *timeout,
		BaseURL:    *baseURL,
	}
	res, err := issue2md.Convert(context.Background(), fs.Arg(0), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "issue2md:", err)
		return exitCode(err)
	}
	if opts.OutputPath == "-" {
		fmt.Print(res.Markdown)
	}
	return 0
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// exitCode 将 *issue2md.Error.Kind 映射到进程退出码（API-sketch §A.6）。
func exitCode(err error) int {
	var e *issue2md.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case issue2md.KindUsage:
			return 2
		case issue2md.KindIO:
			return 1
		default:
			return 3
		}
	}
	return 1
}
