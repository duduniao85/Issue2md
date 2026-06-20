// graphql_test.go 验证 GraphQL 数据源（Discussion + 嵌套 replies + cursor 分页），见 spec.md §4.3。httptest（宪法 2.3）。
package issue2md

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// gqlDiscussion 构造一个 discussion GraphQL 响应体；commentsJSON 为 comments 节点的原始 JSON。
func gqlDiscussion(commentsJSON, pageInfoJSON string) string {
	return `{"data":{"repository":{"discussion":{` +
		`"title":"DT","body":"DB","url":"https://github.com/o/r/discussions/7","number":7,` +
		`"author":{"login":"alice"},"createdAt":"2024-01-01T00:00:00Z",` +
		`"reactions":{"totalCount":5},` +
		`"labels":{"nodes":[{"name":"q"}]},` +
		`"comments":{` + commentsJSON + `,"pageInfo":` + pageInfoJSON + `}}}}}`
}

func TestFetchDiscussion_NestedReplies(t *testing.T) {
	// 3 层嵌套：comment c1 → reply r1 → reply rr1
	comments := `"nodes":[{"author":{"login":"u1"},"body":"c1","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":1},"replies":{"nodes":[{"author":{"login":"u2"},"body":"r1","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":0},"replies":{"nodes":[{"author":{"login":"u3"},"body":"rr1","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":0}}]}}]}}]`
	body := gqlDiscussion(comments, `{"hasNextPage":false,"endCursor":null}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	doc, err := fetchDiscussion(context.Background(), c, Ref{KindDiscussion, "o", "r", 7})
	if err != nil {
		t.Fatalf("fetchDiscussion: %v", err)
	}
	if doc.Kind != KindDiscussion {
		t.Errorf("Kind = %v, want KindDiscussion", doc.Kind)
	}
	if doc.Title != "DT" || doc.Author != "alice" || doc.Body != "DB" {
		t.Errorf("meta = %+v, want DT/alice/DB", doc)
	}
	if doc.Reactions.TotalCount != 5 {
		t.Errorf("Reactions.TotalCount = %d, want 5", doc.Reactions.TotalCount)
	}
	if len(doc.Labels) != 1 || doc.Labels[0] != "q" {
		t.Errorf("Labels = %v, want [q]", doc.Labels)
	}
	if len(doc.Comments) != 1 {
		t.Fatalf("len(Comments) = %d, want 1", len(doc.Comments))
	}
	c1 := doc.Comments[0]
	if c1.Body != "c1" || c1.Author != "u1" {
		t.Errorf("c1 = %+v, want c1/u1", c1)
	}
	if len(c1.Replies) != 1 || c1.Replies[0].Body != "r1" {
		t.Fatalf("c1.Replies = %+v, want r1", c1.Replies)
	}
	r1 := c1.Replies[0]
	if len(r1.Replies) != 1 || r1.Replies[0].Body != "rr1" || r1.Replies[0].Author != "u3" {
		t.Errorf("r1.Replies = %+v, want rr1/u3", r1.Replies)
	}
}

func TestFetchDiscussion_CursorPagination(t *testing.T) {
	page1Comments := `"nodes":[{"author":{"login":"u1"},"body":"c1","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":0}},{"author":{"login":"u2"},"body":"c2","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":0}}]`
	page2Comments := `"nodes":[{"author":{"login":"u3"},"body":"c3","createdAt":"2024-01-01T00:00:00Z","reactions":{"totalCount":0}}]`
	page1 := gqlDiscussion(page1Comments, `{"hasNextPage":true,"endCursor":"X"}`)
	page2 := gqlDiscussion(page2Comments, `{"hasNextPage":false,"endCursor":null}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Variables struct {
				Cursor string `json:"cursor"`
			} `json:"variables"`
		}
		_ = json.Unmarshal(raw, &req)
		if req.Variables.Cursor == "X" {
			io.WriteString(w, page2)
			return
		}
		io.WriteString(w, page1)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	doc, err := fetchDiscussion(context.Background(), c, Ref{KindDiscussion, "o", "r", 7})
	if err != nil {
		t.Fatalf("fetchDiscussion: %v", err)
	}
	if len(doc.Comments) != 3 {
		t.Fatalf("len(Comments) = %d, want 3 (cursor 分页全量)", len(doc.Comments))
	}
	if doc.Comments[0].Body != "c1" || doc.Comments[2].Body != "c3" {
		t.Errorf("Comments = %v %v, want c1..c3", doc.Comments[0].Body, doc.Comments[2].Body)
	}
}

func TestFetchDiscussion_NotFound(t *testing.T) {
	// discussion 为 null（不存在）
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":{"repository":{"discussion":null}}}`)
	}))
	defer srv.Close()

	c := newHTTPClient(Options{BaseURL: srv.URL})
	_, err := fetchDiscussion(context.Background(), c, Ref{KindDiscussion, "o", "r", 7})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("fetchDiscussion(null) err = %v, want errors.Is(ErrNotFound)", err)
	}
}
