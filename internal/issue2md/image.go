// image.go 处理图片：扫描 markdown、下载去重、改写链接，见 spec.md §4.5/§9.11-9.14。
// 下载失败仅记入 warnings，不中断（部分成功）；stdout 模式由 Convert 跳过本步骤。
package issue2md

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var imageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// localizeImages 扫描 md 中的 Markdown 图片，下载到 imageDir 并改写链接为 linkPrefix+name。
// 返回改写后的 md 与 warnings（下载失败的）；同名 URL 仅下载一次。
func localizeImages(ctx context.Context, c *httpClient, md, imageDir, linkPrefix string) (string, []string, error) {
	if imageRe.MatchString(md) {
		if err := os.MkdirAll(imageDir, 0o755); err != nil {
			return md, nil, &Error{Kind: KindIO, Op: "localize images",
				Message: "create image dir: " + imageDir, Cause: err}
		}
	}
	var warnings []string
	seen := map[string]string{} // url -> local name
	seq := 0
	result := imageRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := imageRe.FindStringSubmatch(match)
		alt, imgURL := sub[1], sub[2]
		if name, ok := seen[imgURL]; ok {
			return "![" + alt + "](" + linkPrefix + name + ")"
		}
		data, ext, err := downloadImage(ctx, c, imgURL)
		if err != nil {
			warnings = append(warnings, "image download failed: "+imgURL+": "+err.Error())
			return match // 保留原远程链接
		}
		seq++
		name := strconv.Itoa(seq) + "-" + hashURL(imgURL) + ext
		if err := os.WriteFile(filepath.Join(imageDir, name), data, 0o644); err != nil {
			warnings = append(warnings, "image write failed: "+name+": "+err.Error())
			return match
		}
		seen[imgURL] = name
		return "![" + alt + "](" + linkPrefix + name + ")"
	})
	return result, warnings, nil
}

// downloadImage 直接 GET 绝对图片 URL（host 非 api.github.com），返回 body + 扩展名。
func downloadImage(ctx context.Context, c *httpClient, url string) ([]byte, string, error) {
	resp, err := c.send(ctx, requestSpec{method: http.MethodGet, url: url})
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return data, inferExt(url, resp.Header.Get("Content-Type")), nil
}

// inferExt 按优先级推断扩展名：URL 扩展名 > Content-Type > .bin。
func inferExt(urlStr, contentType string) string {
	lower := strings.ToLower(urlStr)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(lower, ext) {
			return ext
		}
	}
	switch strings.ToLower(contentType) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	}
	return ".bin"
}

// hashURL 返回 URL 的 sha256 前 8 位 hex（命名去重）。
func hashURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:])[:8]
}
