// github.go 封装 GitHub REST 数据源（Issue/PR），见 spec.md §4.3。基于 httpc 基建（路线 B）。
package issue2md

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// --- REST 原始响应 struct（未导出）---

type restUser struct {
	Login string `json:"login"`
}

type restLabel struct {
	Name string `json:"name"`
}

type restReactions struct {
	TotalCount int `json:"total_count"`
	PlusOne    int `json:"+1"`
	MinusOne   int `json:"-1"`
	Laugh      int `json:"laugh"`
	Hooray     int `json:"hooray"`
	Confused   int `json:"confused"`
	Heart      int `json:"heart"`
	Rocket     int `json:"rocket"`
	Eyes       int `json:"eyes"`
}

type restIssue struct {
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"`
	HTMLURL   string        `json:"html_url"`
	Number    int           `json:"number"`
	User      restUser      `json:"user"`
	Labels    []restLabel   `json:"labels"`
	Reactions restReactions `json:"reactions"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type restComment struct {
	Body      string        `json:"body"`
	User      restUser      `json:"user"`
	Reactions restReactions `json:"reactions"`
	CreatedAt time.Time     `json:"created_at"`
}

type restBranch struct {
	Ref string `json:"ref"`
}

type restPull struct {
	Merged bool       `json:"merged"`
	Base   restBranch `json:"base"`
	Head   restBranch `json:"head"`
}

// --- 获取与映射 ---

// fetchIssue 获取 Issue 主体 + 评论，组装 Document（KindIssue）。
func fetchIssue(ctx context.Context, c *httpClient, ref Ref) (*Document, error) {
	ri, err := getIssue(ctx, c, ref)
	if err != nil {
		return nil, err
	}
	comments, err := listComments(ctx, c, ref)
	if err != nil {
		return nil, err
	}
	return issueToDocument(ri, comments, ref), nil
}

// fetchPull 获取 PR 主体 + PR 字段 + 评论，组装 Document（KindPull）。
func fetchPull(ctx context.Context, c *httpClient, ref Ref) (*Document, error) {
	ri, err := getIssue(ctx, c, ref) // issue 端点同样可取 PR 主体
	if err != nil {
		return nil, err
	}
	rp, err := getPull(ctx, c, ref)
	if err != nil {
		return nil, err
	}
	comments, err := listComments(ctx, c, ref)
	if err != nil {
		return nil, err
	}
	doc := issueToDocument(ri, comments, ref)
	doc.Kind = KindPull
	doc.PR = &PRInfo{Merged: rp.Merged, Base: rp.Base.Ref, Head: rp.Head.Ref}
	return doc, nil
}

// getIssue GET /repos/{o}/{r}/issues/{n}。
func getIssue(ctx context.Context, c *httpClient, ref Ref) (*restIssue, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues/%d", ref.Owner, ref.Repo, ref.Number))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	var ri restIssue
	if err := json.NewDecoder(resp.Body).Decode(&ri); err != nil {
		return nil, &Error{Kind: KindServer, Op: "fetch issue", Message: "decode failed", Cause: err}
	}
	return &ri, nil
}

// getPull GET /repos/{o}/{r}/pulls/{n}。
func getPull(ctx context.Context, c *httpClient, ref Ref) (*restPull, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	var rp restPull
	if err := json.NewDecoder(resp.Body).Decode(&rp); err != nil {
		return nil, &Error{Kind: KindServer, Op: "fetch pull", Message: "decode failed", Cause: err}
	}
	return &rp, nil
}

// listComments 分页获取全部评论，跟随 Link rel=next。
func listComments(ctx context.Context, c *httpClient, ref Ref) ([]Comment, error) {
	listURL := fmt.Sprintf("/repos/%s/%s/issues/%d/comments?per_page=100", ref.Owner, ref.Repo, ref.Number)
	var all []Comment
	for {
		resp, err := c.do(ctx, http.MethodGet, listURL)
		if err != nil {
			return nil, err
		}
		if err := checkStatus(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		var rc []restComment
		decErr := json.NewDecoder(resp.Body).Decode(&rc)
		link := resp.Header.Get("Link")
		resp.Body.Close()
		if decErr != nil {
			return nil, &Error{Kind: KindServer, Op: "fetch comments", Message: "decode failed", Cause: decErr}
		}
		for _, x := range rc {
			all = append(all, restCommentToComment(x))
		}
		next := parseNextLink(link)
		if next == "" {
			break
		}
		u, err := url.Parse(next)
		if err != nil {
			break
		}
		listURL = u.RequestURI() // path?query，重新基于 baseURL 请求
	}
	return all, nil
}

// issueToDocument 将 restIssue + 评论映射为 Document。
func issueToDocument(ri *restIssue, comments []Comment, ref Ref) *Document {
	labels := make([]string, 0, len(ri.Labels))
	for _, l := range ri.Labels {
		labels = append(labels, l.Name)
	}
	return &Document{
		Kind:       ref.Kind,
		Title:      ri.Title,
		URL:        ri.HTMLURL,
		Repository: ref.Owner + "/" + ref.Repo,
		Number:     ri.Number,
		Author:     ri.User.Login,
		State:      ri.State,
		CreatedAt:  ri.CreatedAt,
		UpdatedAt:  ri.UpdatedAt,
		Labels:     labels,
		Body:       ri.Body,
		Reactions:  restReactionsToReactions(ri.Reactions),
		Comments:   comments,
	}
}

func restCommentToComment(rc restComment) Comment {
	return Comment{
		Author:    rc.User.Login,
		CreatedAt: rc.CreatedAt,
		Body:      rc.Body,
		Reactions: restReactionsToReactions(rc.Reactions),
	}
}

func restReactionsToReactions(r restReactions) Reactions {
	return Reactions{
		TotalCount: r.TotalCount,
		PlusOne:    r.PlusOne,
		MinusOne:   r.MinusOne,
		Laugh:      r.Laugh,
		Hooray:     r.Hooray,
		Confused:   r.Confused,
		Heart:      r.Heart,
		Rocket:     r.Rocket,
		Eyes:       r.Eyes,
	}
}
