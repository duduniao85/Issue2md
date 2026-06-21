// fetch_test.go 验证 Source 调度（NewSource + Fetch 按 Kind 分发），见 plan.md §6.3。httptest（宪法 2.3）。
package issue2md

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewSource(t *testing.T) {
	src := NewSource(Options{BaseURL: "http://x"})
	if src == nil {
		t.Fatal("NewSource returned nil")
	}
}

func TestSourceFetch_IssueDispatch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/comments") {
			io.WriteString(w, `[]`)
			return
		}
		io.WriteString(w, `{"title":"T","body":"","state":"open","html_url":"u","number":1,"user":{"login":"a"},"labels":[],`+
			restReactionsZero+`,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	src := NewSource(Options{BaseURL: srv.URL})
	doc, err := src.Fetch(context.Background(), Ref{KindIssue, "o", "r", 1})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if doc.Kind != KindIssue {
		t.Errorf("Kind = %v, want KindIssue", doc.Kind)
	}
	if !strings.Contains(gotPath, "/issues/") {
		t.Errorf("dispatched path = %q, want contains /issues/ (REST)", gotPath)
	}
}

func TestSourceFetch_DiscussionDispatch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, gqlDiscussion(`"nodes":[]`, `{"hasNextPage":false,"endCursor":null}`))
	}))
	defer srv.Close()

	src := NewSource(Options{BaseURL: srv.URL})
	doc, err := src.Fetch(context.Background(), Ref{KindDiscussion, "o", "r", 7})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if doc.Kind != KindDiscussion {
		t.Errorf("Kind = %v, want KindDiscussion", doc.Kind)
	}
	if gotPath != "/graphql" {
		t.Errorf("dispatched path = %q, want /graphql", gotPath)
	}
}

// TestSourceFetch_PullDispatch 验证 KindPull 分发命中 REST /pulls/ 端点（补充分发覆盖）。
func TestSourceFetch_PullDispatch(t *testing.T) {
	var sawPulls bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments"):
			io.WriteString(w, `[]`)
		case strings.Contains(r.URL.Path, "/pulls/"):
			sawPulls = true
			io.WriteString(w, `{"merged":false,"base":{"ref":"main"},"head":{"ref":"dev"}}`)
		default:
			io.WriteString(w, `{"title":"PT","body":"","state":"open","html_url":"u","number":2,"user":{"login":"a"},"labels":[],`+
				restReactionsZero+`,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
		}
	}))
	defer srv.Close()

	src := NewSource(Options{BaseURL: srv.URL})
	doc, err := src.Fetch(context.Background(), Ref{KindPull, "o", "r", 2})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if doc.Kind != KindPull {
		t.Errorf("Kind = %v, want KindPull", doc.Kind)
	}
	if !sawPulls {
		t.Error("dispatched path 缺少 /pulls/（PR 未走 REST pulls 端点）")
	}
	if doc.PR == nil || doc.PR.Base != "main" {
		t.Errorf("PR = %+v, want Base=main", doc.PR)
	}
}
