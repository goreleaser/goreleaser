// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// TopicsList represents a list of repo's topics
type TopicsList struct {
	Topics []string `json:"topics"`
}

// ListRepoTopics list all repository's topics
func (c *Client) ListRepoTopics(user, repo string) (*TopicsList, error) {
	var list TopicsList
	return &list, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/topics", user, repo), nil, nil, &list)
}

// SetRepoTopics replaces the list of repo's topics
func (c *Client) SetRepoTopics(user, repo string, list TopicsList) error {
	body, err := json.Marshal(&list)
	if err != nil {
		return err
	}

	_, err = c.getResponse("PUT", fmt.Sprintf("/repos/%s/%s/topics", user, repo), jsonHeader, bytes.NewReader(body))
	return err
}

// AddRepoTopic adds a topic to a repo's topics list
func (c *Client) AddRepoTopic(user, repo, topic string) error {
	_, err := c.getResponse("PUT", fmt.Sprintf("/repos/%s/%s/topics/%s", user, repo, topic), nil, nil)
	return err
}

// DeleteRepoTopic deletes a topic from repo's topics list
func (c *Client) DeleteRepoTopic(user, repo, topic string) error {
	_, err := c.getResponse("DELETE", fmt.Sprintf("/repos/%s/%s/topics/%s", user, repo, topic), nil, nil)
	return err
}
