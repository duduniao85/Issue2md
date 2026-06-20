// graphql.go 封装 GitHub GraphQL 数据源（Discussion），见 spec.md §4.3。
// 用标准库 net/http 直接 POST GraphQL query（路线 B：不引第三方 GraphQL 客户端）。
package issue2md

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// discussionQuery 取 discussion 主体 + 首层 comments(cursor 分页) + 3 层嵌套 replies。
// MVP 固定 3 层内联（覆盖绝大多数 Discussion；更深可在后续迭代递归查询）。
const discussionQuery = `query($owner:String!, $repo:String!, $number:Int!, $cursor:String){
  repository(owner:$owner, name:$repo){
    discussion(number:$number){
      title body url number closed createdAt
      author{login}
      reactions{totalCount}
      labels(first:100){nodes{name}}
      comments(first:100, after:$cursor){
        nodes{
          author{login} body createdAt reactions{totalCount}
          replies(first:100){
            nodes{
              author{login} body createdAt reactions{totalCount}
              replies(first:100){
                nodes{author{login} body createdAt reactions{totalCount}}
              }
            }
          }
        }
        pageInfo{hasNextPage endCursor}
      }
    }
  }
}`

// --- GraphQL 响应 struct（未导出）---

type discussionResp struct {
	Data struct {
		Repository struct {
			Discussion graphqlDiscussion `json:"discussion"`
		} `json:"repository"`
	} `json:"data"`
}

type graphqlDiscussion struct {
	Title     string             `json:"title"`
	Body      string             `json:"body"`
	URL       string             `json:"url"`
	Number    int                `json:"number"`
	Closed    bool               `json:"closed"`
	CreatedAt time.Time          `json:"createdAt"`
	Author    graphqlActor       `json:"author"`
	Reactions graphqlReactions   `json:"reactions"`
	Labels    graphqlLabelConn   `json:"labels"`
	Comments  graphqlCommentConn `json:"comments"`
}

type graphqlActor struct {
	Login string `json:"login"`
}

type graphqlReactions struct {
	TotalCount int `json:"totalCount"`
}

type graphqlLabelConn struct {
	Nodes []graphqlLabel `json:"nodes"`
}

type graphqlLabel struct {
	Name string `json:"name"`
}

type graphqlCommentConn struct {
	Nodes    []graphqlComment `json:"nodes"`
	PageInfo graphqlPageInfo  `json:"pageInfo"`
}

type graphqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// graphqlComment 递归：Replies 承载嵌套回复。
type graphqlComment struct {
	Author    graphqlActor       `json:"author"`
	Body      string             `json:"body"`
	CreatedAt time.Time          `json:"createdAt"`
	Reactions graphqlReactions   `json:"reactions"`
	Replies   graphqlCommentConn `json:"replies"`
}

// fetchDiscussion 获取 Discussion（含嵌套 replies + 首层 comments cursor 分页），组装 Document。
func fetchDiscussion(ctx context.Context, c *httpClient, ref Ref) (*Document, error) {
	var doc *Document
	var cursor string
	for {
		body, err := postGraphQL(ctx, c, discussionQuery, map[string]any{
			"owner": ref.Owner, "repo": ref.Repo, "number": ref.Number, "cursor": cursor,
		})
		if err != nil {
			return nil, err
		}
		var dr discussionResp
		if err := json.Unmarshal(body, &dr); err != nil {
			return nil, &Error{Kind: KindServer, Op: "fetch discussion", Message: "decode failed", Cause: err}
		}
		d := dr.Data.Repository.Discussion
		if doc == nil {
			if d.URL == "" { // discussion: null（不存在）
				return nil, &Error{Kind: KindNotFound, Op: "fetch discussion", Message: "discussion not found"}
			}
			doc = discussionToDocument(&d, ref)
		} else {
			// 后续页只追加首层 comments，主体忽略
			doc.Comments = append(doc.Comments, gqlCommentsToComments(d.Comments.Nodes)...)
		}
		if !d.Comments.PageInfo.HasNextPage {
			break
		}
		cursor = d.Comments.PageInfo.EndCursor
	}
	return doc, nil
}

// postGraphQL POST /graphql，返回响应体；网络/状态码错误映射为 *Error。
func postGraphQL(ctx context.Context, c *httpClient, query string, vars map[string]any) ([]byte, error) {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, &Error{Kind: KindServer, Op: "graphql", Message: "encode failed", Cause: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/graphql", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Kind: KindNetwork, Op: "graphql", Message: "network error", Cause: err}
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

func discussionToDocument(d *graphqlDiscussion, ref Ref) *Document {
	labels := make([]string, 0, len(d.Labels.Nodes))
	for _, l := range d.Labels.Nodes {
		labels = append(labels, l.Name)
	}
	return &Document{
		Kind:       KindDiscussion,
		Title:      d.Title,
		URL:        d.URL,
		Repository: ref.Owner + "/" + ref.Repo,
		Number:     d.Number,
		Author:     d.Author.Login,
		State:      discussionState(d.Closed),
		CreatedAt:  d.CreatedAt,
		Labels:     labels,
		Body:       d.Body,
		Reactions:  Reactions{TotalCount: d.Reactions.TotalCount},
		Comments:   gqlCommentsToComments(d.Comments.Nodes),
	}
}

func discussionState(closed bool) string {
	if closed {
		return "closed"
	}
	return "open"
}

// gqlCommentsToComments 递归映射（含嵌套 Replies）。
func gqlCommentsToComments(nodes []graphqlComment) []Comment {
	out := make([]Comment, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, graphqlCommentToComment(n))
	}
	return out
}

func graphqlCommentToComment(g graphqlComment) Comment {
	return Comment{
		Author:    g.Author.Login,
		CreatedAt: g.CreatedAt,
		Body:      g.Body,
		Reactions: Reactions{TotalCount: g.Reactions.TotalCount},
		Replies:   gqlCommentsToComments(g.Replies.Nodes),
	}
}
