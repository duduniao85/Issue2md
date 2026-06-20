// main_test.go 验证 CLI 行为（run 返回退出码），见 spec.md §5。
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	if code := run([]string{"-v"}); code != 0 {
		t.Errorf("run(-v) = %d, want 0", code)
	}
}

func TestRun_NoArgs(t *testing.T) {
	if code := run([]string{}); code != 2 {
		t.Errorf("run() = %d, want 2", code)
	}
}

func TestRun_InvalidURL(t *testing.T) {
	if code := run([]string{"not-a-url"}); code != 2 {
		t.Errorf("run(not-a-url) = %d, want 2", code)
	}
}

func TestRun_Convert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/comments") {
			io.WriteString(w, `[]`)
			return
		}
		io.WriteString(w, `{"title":"T","body":"B","state":"open","html_url":"u","number":1,"user":{"login":"a"},"labels":[],"reactions":{"total_count":0,"+1":0,"-1":0,"laugh":0,"hooray":0,"confused":0,"heart":0,"rocket":0,"eyes":0},"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "r.md")
	code := run([]string{"-baseurl", srv.URL, "-o", out, "https://github.com/o/r/issues/1"})
	if code != 0 {
		t.Fatalf("run = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "# T") {
		t.Errorf("文件应含标题 # T\n%s", string(data))
	}
}
