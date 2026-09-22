package forgejo

import (
	"net/url"

	"acahti/internal/page"
)

type ActivityComment struct {
	HTMLURL        string `json:"html_url"`
	IssueURL       string `json:"issue_url"`
	PullRequestURL string `json:"pull_request_url"`
}

type Activity struct {
	ID      int64            `json:"id"`
	OpType  string           `json:"op_type"`
	Content string           `json:"content"`
	RefName string           `json:"ref_name"`
	Created string           `json:"created"`
	Repo    *Repo            `json:"repo"`
	Comment *ActivityComment `json:"comment,omitempty"`
}

func (c *Client) ListUserActivityFeeds(login, date string, q page.Query) (page.Result[Activity], error) {
	if !c.Ready() || login == "" {
		return page.Of([]Activity{}, q, false), nil
	}
	extra := url.Values{}
	extra.Set("only-performed-by", "true")
	if date != "" {
		extra.Set("date", date)
	}
	return listPage[Activity](c, "/api/v1/users/"+url.PathEscape(login)+"/activities/feeds", q, extra, login)
}
