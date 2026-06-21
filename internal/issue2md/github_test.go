// github_test.go 验证 REST 数据源（Issue/PR + 评论分页），见 spec.md §4.3。httptest（宪法 2.3）。
package issue2md

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const restReactionsZero = `"reactions":{"total_count":0,"+1":0,"-1":0,"laugh":0,"hooray":0,"confused":0,"heart":0,"rocket":0,"eyes":0}`

func TestFetchIssue(t *testing.T) {
	const issueJSON = `{"title":"T","body":"B","state":"open","html_url":"https://github.com/o/r/issues/1","number":1,"user":{"login":"alice"},"labels":[{"name":"bug"},{"name":"help"}],` +
		`"reactions":{"total_count":3,"+1":2,"-1":0,"laugh":1,"hooray":0,"confused":0,"heart":0,"rocket":0,"eyes":0},"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/comments") {
			io.WriteString(w, `[]`)
			return
		}
		io.WriteString(w, issueJSON)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	doc, err := fetchIssue(context.Background(), c, Ref{KindIssue, "o", "r", 1})
	if err != nil {
		t.Fatalf("fetchIssue: %v", err)
	}
	if doc.Kind != KindIssue {
		t.Errorf("Kind = %v, want KindIssue", doc.Kind)
	}
	if doc.Title != "T" {
		t.Errorf("Title = %q, want T", doc.Title)
	}
	if doc.Author != "alice" {
		t.Errorf("Author = %q, want alice", doc.Author)
	}
	if doc.State != "open" {
		t.Errorf("State = %q, want open", doc.State)
	}
	if doc.Body != "B" {
		t.Errorf("Body = %q, want B", doc.Body)
	}
	if doc.Number != 1 {
		t.Errorf("Number = %d, want 1", doc.Number)
	}
	if len(doc.Labels) != 2 || doc.Labels[0] != "bug" || doc.Labels[1] != "help" {
		t.Errorf("Labels = %v, want [bug help]", doc.Labels)
	}
	if doc.Reactions.TotalCount != 3 || doc.Reactions.PlusOne != 2 || doc.Reactions.Laugh != 1 {
		t.Errorf("Reactions = %+v, want total=3 +1=2 laugh=1", doc.Reactions)
	}
	if doc.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestFetchPull(t *testing.T) {
	const issueJSON = `{"title":"PR","body":"PB","state":"open","html_url":"https://github.com/o/r/pull/2","number":2,"user":{"login":"bob"},"labels":[],` +
		restReactionsZero + `,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`
	const pullJSON = `{"merged":true,"base":{"ref":"main"},"head":{"ref":"feature"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments"):
			io.WriteString(w, `[]`)
		case strings.Contains(r.URL.Path, "/pulls/"):
			io.WriteString(w, pullJSON)
		default:
			io.WriteString(w, issueJSON)
		}
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	doc, err := fetchPull(context.Background(), c, Ref{KindPull, "o", "r", 2})
	if err != nil {
		t.Fatalf("fetchPull: %v", err)
	}
	if doc.Kind != KindPull {
		t.Errorf("Kind = %v, want KindPull", doc.Kind)
	}
	if doc.PR == nil {
		t.Fatal("PR is nil")
	}
	if !doc.PR.Merged {
		t.Error("PR.Merged = false, want true")
	}
	if doc.PR.Base != "main" {
		t.Errorf("PR.Base = %q, want main", doc.PR.Base)
	}
	if doc.PR.Head != "feature" {
		t.Errorf("PR.Head = %q, want feature", doc.PR.Head)
	}
}

func TestListComments_Pagination(t *testing.T) {
	page1 := `[{"body":"c1","user":{"login":"u1"},` + restReactionsZero + `,"created_at":"2024-01-01T00:00:00Z"},` +
		`{"body":"c2","user":{"login":"u2"},` + restReactionsZero + `,"created_at":"2024-01-01T00:00:00Z"}]`
	page2 := `[{"body":"c3","user":{"login":"u3"},` + restReactionsZero + `,"created_at":"2024-01-01T00:00:00Z"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			io.WriteString(w, page2)
			return
		}
		w.Header().Set("Link", `<`+r.URL.Path+`?page=2>; rel="next"`)
		io.WriteString(w, page1)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	comments, err := listComments(context.Background(), c, Ref{KindIssue, "o", "r", 1})
	if err != nil {
		t.Fatalf("listComments: %v", err)
	}
	if len(comments) != 3 {
		t.Fatalf("len(comments) = %d, want 3", len(comments))
	}
	if comments[0].Body != "c1" || comments[0].Author != "u1" {
		t.Errorf("comments[0] = %+v, want c1/u1", comments[0])
	}
	if comments[2].Body != "c3" {
		t.Errorf("comments[2] = %+v, want c3", comments[2])
	}
}

func TestFetchIssue_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := newHTTPClient(Options{BaseURL: srv.URL})
	_, err := fetchIssue(context.Background(), c, Ref{KindIssue, "o", "r", 1})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("fetchIssue(404) err = %v, want errors.Is(ErrNotFound)", err)
	}
}

// TestListComments_MalformedNextLink 验证：Link 头里 rel="next" 的 URL 无法解析时，
// 不再静默 break 丢失后续评论，而是显式返回错误（宪法 §3.1 显式处理）。
func TestListComments_MalformedNextLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 非法 URL（无效百分号转义）使 url.Parse 失败
		w.Header().Set("Link", `<http://a/%zz>; rel="next"`)
		io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	_, err := listComments(context.Background(), c, Ref{KindIssue, "o", "r", 1})
	if err == nil {
		t.Fatal("listComments(非法 next link) err = nil, want non-nil")
	}
}
