//
// Copyright 2020, Eric Stevens
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"fmt"
	"time"
)

// GroupHook represents a GitLab group hook.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/groups.html#list-group-hooks
type GroupHook struct {
	ID                       int        `json:"id"`
	URL                      string     `json:"url"`
	GroupID                  int        `json:"group_id"`
	PushEvents               bool       `json:"push_events"`
	IssuesEvents             bool       `json:"issues_events"`
	ConfidentialIssuesEvents bool       `json:"confidential_issues_events"`
	MergeRequestsEvents      bool       `json:"merge_requests_events"`
	TagPushEvents            bool       `json:"tag_push_events"`
	NoteEvents               bool       `json:"note_events"`
	JobEvents                bool       `json:"job_events"`
	PipelineEvents           bool       `json:"pipeline_events"`
	WikiPageEvents           bool       `json:"wiki_page_events"`
	EnableSslVerification    bool       `json:"enable_ssl_verification"`
	CreatedAt                *time.Time `json:"created_at"`
}

// ListGroupHooks gets a list of group hooks.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/groups.html#list-group-hooks
func (s *GroupsService) ListGroupHooks(gid interface{}) ([]*GroupHook, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/hooks", pathEscape(group))

	req, err := s.client.NewRequest("GET", u, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	var gh []*GroupHook
	resp, err := s.client.Do(req, &gh)
	if err != nil {
		return nil, resp, err
	}

	return gh, resp, err
}
