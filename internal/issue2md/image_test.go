// image_test.go 验证图片处理（下载/去重/改写链接），见 spec.md §4.5/§9.11-9.14。httptest + t.TempDir（宪法 2.3）。
package issue2md

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	return len(entries)
}

func TestLocalizeImages_Basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	md := "intro\n![alt](" + srv.URL + "/img.png)\nend"
	dir := t.TempDir()
	out, warnings, err := localizeImages(context.Background(), c, md, dir, "./x.files/")
	if err != nil {
		t.Fatalf("localizeImages: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want empty", warnings)
	}
	if strings.Contains(out, srv.URL) {
		t.Error("原 URL 应被改写掉")
	}
	if !strings.Contains(out, "./x.files/") {
		t.Errorf("应含本地前缀 ./x.files/，got:\n%s", out)
	}
	if n := countFiles(t, dir); n != 1 {
		t.Errorf("下载文件数 = %d, want 1", n)
	}
}

func TestLocalizeImages_Dedup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	url := srv.URL + "/img.png"
	md := "![](" + url + ") ![](" + url + ")"
	dir := t.TempDir()
	out, _, err := localizeImages(context.Background(), c, md, dir, "./x.files/")
	if err != nil {
		t.Fatalf("localizeImages: %v", err)
	}
	if n := countFiles(t, dir); n != 1 {
		t.Errorf("重复 URL 应只下载 1 次，got %d", n)
	}
	if cnt := strings.Count(out, "./x.files/"); cnt != 2 {
		t.Errorf("两处链接都应改写为本地，got %d 处，out:\n%s", cnt, out)
	}
}

func TestLocalizeImages_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	url := srv.URL + "/missing.png"
	md := "![](" + url + ")"
	dir := t.TempDir()
	out, warnings, err := localizeImages(context.Background(), c, md, dir, "./x.files/")
	if err != nil {
		t.Fatalf("下载失败不应使 localizeImages 报错（部分成功），got: %v", err)
	}
	if !strings.Contains(out, url) {
		t.Error("下载失败应保留原远程链接")
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %v, want 1", warnings)
	}
	if n := countFiles(t, dir); n != 0 {
		t.Errorf("失败不应落盘文件，got %d", n)
	}
}

func TestLocalizeImages_ContentTypeExt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	md := "![](" + srv.URL + "/noext" + ")" // URL 无扩展名
	dir := t.TempDir()
	_, _, err := localizeImages(context.Background(), c, md, dir, "./x.files/")
	if err != nil {
		t.Fatalf("localizeImages: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".png") {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("无扩展名应按 Content-Type 推断为 .png，got %v", names)
	}
}
