// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"fmt"
	"time"
)

// PayloadUser represents the author or committer of a commit
type PayloadUser struct {
	// Full name of the commit author
	Name     string `json:"name"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

// FIXME: consider using same format as API when commits API are added.
//        applies to PayloadCommit and PayloadCommitVerification

// PayloadCommit represents a commit
type PayloadCommit struct {
	// sha1 hash of the commit
	ID           string                     `json:"id"`
	Message      string                     `json:"message"`
	URL          string                     `json:"url"`
	Author       *PayloadUser               `json:"author"`
	Committer    *PayloadUser               `json:"committer"`
	Verification *PayloadCommitVerification `json:"verification"`
	Timestamp    time.Time                  `json:"timestamp"`
	Added        []string                   `json:"added"`
	Removed      []string                   `json:"removed"`
	Modified     []string                   `json:"modified"`
}

// PayloadCommitVerification represents the GPG verification of a commit
type PayloadCommitVerification struct {
	Verified  bool   `json:"verified"`
	Reason    string `json:"reason"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
}

// Branch represents a repository branch
type Branch struct {
	Name   string         `json:"name"`
	Commit *PayloadCommit `json:"commit"`
}

// ListRepoBranches list all the branches of one repository
func (c *Client) ListRepoBranches(user, repo string) ([]*Branch, error) {
	branches := make([]*Branch, 0, 10)
	return branches, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/branches", user, repo), nil, nil, &branches)
}

// GetRepoBranch get one branch's information of one repository
func (c *Client) GetRepoBranch(user, repo, branch string) (*Branch, error) {
	b := new(Branch)
	return b, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/branches/%s", user, repo, branch), nil, nil, &b)
}
