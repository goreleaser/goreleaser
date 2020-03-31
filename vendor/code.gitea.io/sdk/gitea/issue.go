// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// PullRequestMeta PR info if an issue is a PR
type PullRequestMeta struct {
	HasMerged bool       `json:"merged"`
	Merged    *time.Time `json:"merged_at"`
}

// Issue represents an issue in a repository
type Issue struct {
	ID               int64      `json:"id"`
	URL              string     `json:"url"`
	Index            int64      `json:"number"`
	Poster           *User      `json:"user"`
	OriginalAuthor   string     `json:"original_author"`
	OriginalAuthorID int64      `json:"original_author_id"`
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	Labels           []*Label   `json:"labels"`
	Milestone        *Milestone `json:"milestone"`
	Assignee         *User      `json:"assignee"`
	Assignees        []*User    `json:"assignees"`
	// Whether the issue is open or closed
	State       StateType        `json:"state"`
	Comments    int              `json:"comments"`
	Created     time.Time        `json:"created_at"`
	Updated     time.Time        `json:"updated_at"`
	Closed      *time.Time       `json:"closed_at"`
	Deadline    *time.Time       `json:"due_date"`
	PullRequest *PullRequestMeta `json:"pull_request"`
}

// ListIssueOption list issue options
type ListIssueOption struct {
	Page int
	// open, closed, all
	State   string
	Type    IssueType
	Labels  []string
	KeyWord string
}

// IssueType is issue a pull or only an issue
type IssueType string

const (
	// IssueTypeAll pr and issue
	IssueTypeAll IssueType = ""
	// IssueTypeIssue only issues
	IssueTypeIssue IssueType = "issues"
	// IssueTypePull only pulls
	IssueTypePull IssueType = "pulls"
)

// QueryEncode turns options into querystring argument
func (opt *ListIssueOption) QueryEncode() string {
	query := make(url.Values)
	if opt.Page > 0 {
		query.Add("page", fmt.Sprintf("%d", opt.Page))
	}
	if len(opt.State) > 0 {
		query.Add("state", opt.State)
	}

	if opt.Page > 0 {
		query.Add("page", fmt.Sprintf("%d", opt.Page))
	}
	if len(opt.State) > 0 {
		query.Add("state", opt.State)
	}
	if len(opt.Labels) > 0 {
		var lq string
		for _, l := range opt.Labels {
			if len(lq) > 0 {
				lq += ","
			}
			lq += l
		}
		query.Add("labels", lq)
	}
	if len(opt.KeyWord) > 0 {
		query.Add("q", opt.KeyWord)
	}
	query.Add("type", string(opt.Type))

	return query.Encode()
}

// ListIssues returns all issues assigned the authenticated user
func (c *Client) ListIssues(opt ListIssueOption) ([]*Issue, error) {
	link, _ := url.Parse("/repos/issues/search")
	issues := make([]*Issue, 0, 10)
	link.RawQuery = opt.QueryEncode()
	return issues, c.getParsedResponse("GET", link.String(), jsonHeader, nil, &issues)
}

// ListUserIssues returns all issues assigned to the authenticated user
func (c *Client) ListUserIssues(opt ListIssueOption) ([]*Issue, error) {
	// WARNING: "/user/issues" API is not implemented jet!
	allIssues, err := c.ListIssues(opt)
	if err != nil {
		return nil, err
	}
	user, err := c.GetMyUserInfo()
	if err != nil {
		return nil, err
	}
	// Workaround: client sort out non user related issues
	issues := make([]*Issue, 0, 10)
	for _, i := range allIssues {
		if i.ID == user.ID {
			issues = append(issues, i)
		}
	}
	return issues, nil
}

// ListRepoIssues returns all issues for a given repository
func (c *Client) ListRepoIssues(owner, repo string, opt ListIssueOption) ([]*Issue, error) {
	link, _ := url.Parse(fmt.Sprintf("/repos/%s/%s/issues", owner, repo))
	issues := make([]*Issue, 0, 10)
	link.RawQuery = opt.QueryEncode()
	return issues, c.getParsedResponse("GET", link.String(), jsonHeader, nil, &issues)
}

// GetIssue returns a single issue for a given repository
func (c *Client) GetIssue(owner, repo string, index int64) (*Issue, error) {
	issue := new(Issue)
	return issue, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index), nil, nil, issue)
}

// CreateIssueOption options to create one issue
type CreateIssueOption struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// username of assignee
	Assignee  string     `json:"assignee"`
	Assignees []string   `json:"assignees"`
	Deadline  *time.Time `json:"due_date"`
	// milestone id
	Milestone int64 `json:"milestone"`
	// list of label ids
	Labels []int64 `json:"labels"`
	Closed bool    `json:"closed"`
}

// CreateIssue create a new issue for a given repository
func (c *Client) CreateIssue(owner, repo string, opt CreateIssueOption) (*Issue, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	issue := new(Issue)
	return issue, c.getParsedResponse("POST", fmt.Sprintf("/repos/%s/%s/issues", owner, repo),
		jsonHeader, bytes.NewReader(body), issue)
}

// EditIssueOption options for editing an issue
type EditIssueOption struct {
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	Assignee  *string    `json:"assignee"`
	Assignees []string   `json:"assignees"`
	Milestone *int64     `json:"milestone"`
	State     *string    `json:"state"`
	Deadline  *time.Time `json:"due_date"`
}

// EditIssue modify an existing issue for a given repository
func (c *Client) EditIssue(owner, repo string, index int64, opt EditIssueOption) (*Issue, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	issue := new(Issue)
	return issue, c.getParsedResponse("PATCH", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index),
		jsonHeader, bytes.NewReader(body), issue)
}
