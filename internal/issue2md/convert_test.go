// convert_test.go 验证 Convert 顶层编排（端到端），见 plan.md §6.1 / spec §9。httptest（宪法 2.3）。
package issue2md

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// issueJSON 构造 issue 主体；withImg 控制正文是否含图片 URL（仅用于不下载图片的场景）。
func issueJSON(withImg bool) string {
	body := "CB"
	if withImg {
		body = "CB ![a](http://example.invalid/img.png)"
	}
	return `{"title":"CT","body":"` + body + `","state":"open","html_url":"https://github.com/o/r/issues/1","number":1,"user":{"login":"alice"},"labels":[],` +
		restReactionsZero + `,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`
}

func newIssueServer(t *testing.T, withImg bool, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != 0 && status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/comments") {
			io.WriteString(w, `[]`)
			return
		}
		io.WriteString(w, issueJSON(withImg))
	}))
}

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		ref          Ref
		outputPath   string
		wantPath     string
		wantImageDir string
	}{
		{"自动命名", Ref{KindIssue, "o", "r", 1}, "", "o-r-issue-1.md", "o-r-issue-1.files"},
		{"显式路径", Ref{KindPull, "o", "r", 2}, "/tmp/x.md", "/tmp/x.md", "/tmp/x.files"},
		{"stdout", Ref{KindIssue, "o", "r", 1}, "-", "-", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, imageDir, _ := resolveOutputPath(tt.ref, Options{OutputPath: tt.outputPath})
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			if imageDir != tt.wantImageDir {
				t.Errorf("imageDir = %q, want %q", imageDir, tt.wantImageDir)
			}
		})
	}
}

func TestConvert_IssueToFile(t *testing.T) {
	srv := newIssueServer(t, false, 0) // 正文无图片，避免真实下载
	defer srv.Close()

	tmp := t.TempDir()
	out := filepath.Join(tmp, "result.md")
	res, err := Convert(context.Background(), "https://github.com/o/r/issues/1", Options{BaseURL: srv.URL, OutputPath: out})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if res.OutputPath != out {
		t.Errorf("OutputPath = %q, want %q", res.OutputPath, out)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `title: "CT"`) {
		t.Errorf("文件应含 front matter\n%s", s)
	}
	if !strings.Contains(s, "# CT") {
		t.Errorf("文件应含标题\n%s", s)
	}
}

func TestConvert_Stdout(t *testing.T) {
	srv := newIssueServer(t, true, 0) // 正文含图片
	defer srv.Close()

	res, err := Convert(context.Background(), "https://github.com/o/r/issues/1", Options{BaseURL: srv.URL, OutputPath: "-"})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if res.OutputPath != "-" {
		t.Errorf("OutputPath = %q, want -", res.OutputPath)
	}
	if res.ImageDir != "" {
		t.Errorf("stdout 模式 ImageDir 应为空，got %q", res.ImageDir)
	}
	if !strings.Contains(res.Markdown, "example.invalid") {
		t.Errorf("stdout 应保留原远程图片链接（不下载）")
	}
}

func TestConvert_NoImagesToFile(t *testing.T) {
	srv := newIssueServer(t, true, 0) // 正文含图片
	defer srv.Close()

	tmp := t.TempDir()
	out := filepath.Join(tmp, "r.md")
	_, err := Convert(context.Background(), "https://github.com/o/r/issues/1", Options{BaseURL: srv.URL, OutputPath: out, NoImages: true})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "example.invalid") {
		t.Errorf("NoImages 应保留原远程链接")
	}
	if _, err := os.Stat(filepath.Join(tmp, "r.files")); !os.IsNotExist(err) {
		t.Errorf("NoImages 不应创建图片目录")
	}
}

func TestConvert_InvalidURL(t *testing.T) {
	_, err := Convert(context.Background(), "not a url", Options{})
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("Convert(非法URL) err = %v, want errors.Is(ErrInvalidURL)", err)
	}
}

func TestConvert_NotFound(t *testing.T) {
	srv := newIssueServer(t, false, http.StatusNotFound)
	defer srv.Close()
	_, err := Convert(context.Background(), "https://github.com/o/r/issues/1", Options{BaseURL: srv.URL})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Convert(404) err = %v, want errors.Is(ErrNotFound)", err)
	}
}

func TestConvert_DirNotExist(t *testing.T) {
	srv := newIssueServer(t, false, 0)
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "nodir", "result.md") // nodir 不存在
	_, err := Convert(context.Background(), "https://github.com/o/r/issues/1", Options{BaseURL: srv.URL, OutputPath: out})
	if !errors.Is(err, ErrIO) {
		t.Errorf("Convert(目录不存在) err = %v, want errors.Is(ErrIO)", err)
	}
}
