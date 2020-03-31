// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"fmt"
)

// GetIssueSubscribers get list of users who subscribed on an issue
func (c *Client) GetIssueSubscribers(owner, repo string, index int64) ([]*User, error) {
	if err := c.CheckServerVersionConstraint(">=1.11.0"); err != nil {
		return nil, err
	}
	subscribers := make([]*User, 0, 10)
	return subscribers, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/issues/%d/subscriptions", owner, repo, index), nil, nil, &subscribers)
}

// AddIssueSubscription Subscribe user to issue
func (c *Client) AddIssueSubscription(owner, repo string, index int64, user string) error {
	if err := c.CheckServerVersionConstraint(">=1.11.0"); err != nil {
		return err
	}
	_, err := c.getResponse("PUT", fmt.Sprintf("/repos/%s/%s/issues/%d/subscriptions/%s", owner, repo, index, user), nil, nil)
	return err
}

// DeleteIssueSubscription unsubscribe user from issue
func (c *Client) DeleteIssueSubscription(owner, repo string, index int64, user string) error {
	if err := c.CheckServerVersionConstraint(">=1.11.0"); err != nil {
		return err
	}
	_, err := c.getResponse("DELETE", fmt.Sprintf("/repos/%s/%s/issues/%d/subscriptions/%s", owner, repo, index, user), nil, nil)
	return err
}

// IssueSubscribe subscribe current user to an issue
func (c *Client) IssueSubscribe(owner, repo string, index int64) error {
	u, err := c.GetMyUserInfo()
	if err != nil {
		return err
	}
	return c.AddIssueSubscription(owner, repo, index, u.UserName)
}

// IssueUnSubscribe unsubscribe current user from an issue
func (c *Client) IssueUnSubscribe(owner, repo string, index int64) error {
	u, err := c.GetMyUserInfo()
	if err != nil {
		return err
	}
	return c.DeleteIssueSubscription(owner, repo, index, u.UserName)
}
