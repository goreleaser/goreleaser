// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea // import "code.gitea.io/sdk/gitea"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Attachment a generic attachment
type Attachment struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	DownloadCount int64     `json:"download_count"`
	Created       time.Time `json:"created_at"`
	UUID          string    `json:"uuid"`
	DownloadURL   string    `json:"browser_download_url"`
}

// ListReleaseAttachments list release's attachments
func (c *Client) ListReleaseAttachments(user, repo string, release int64) ([]*Attachment, error) {
	attachments := make([]*Attachment, 0, 10)
	err := c.getParsedResponse("GET",
		fmt.Sprintf("/repos/%s/%s/releases/%d/assets", user, repo, release),
		nil, nil, &attachments)
	return attachments, err
}

// GetReleaseAttachment returns the requested attachment
func (c *Client) GetReleaseAttachment(user, repo string, release int64, id int64) (*Attachment, error) {
	a := new(Attachment)
	err := c.getParsedResponse("GET",
		fmt.Sprintf("/repos/%s/%s/releases/%d/assets/%d", user, repo, release, id),
		nil, nil, &a)
	return a, err
}

// CreateReleaseAttachment creates an attachment for the given release
func (c *Client) CreateReleaseAttachment(user, repo string, release int64, file io.Reader, filename string) (*Attachment, error) {
	// Write file to body
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(part, file); err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}

	// Send request
	attachment := new(Attachment)
	err = c.getParsedResponse("POST",
		fmt.Sprintf("/repos/%s/%s/releases/%d/assets", user, repo, release),
		http.Header{"Content-Type": {writer.FormDataContentType()}}, body, &attachment)
	return attachment, err
}

// EditAttachmentOptions options for editing attachments
type EditAttachmentOptions struct {
	Name string `json:"name"`
}

// EditReleaseAttachment updates the given attachment with the given options
func (c *Client) EditReleaseAttachment(user, repo string, release int64, attachment int64, form EditAttachmentOptions) (*Attachment, error) {
	body, err := json.Marshal(&form)
	if err != nil {
		return nil, err
	}
	attach := new(Attachment)
	return attach, c.getParsedResponse("PATCH", fmt.Sprintf("/repos/%s/%s/releases/%d/assets/%d", user, repo, release, attachment), jsonHeader, bytes.NewReader(body), attach)
}

// DeleteReleaseAttachment deletes the given attachment including the uploaded file
func (c *Client) DeleteReleaseAttachment(user, repo string, release int64, id int64) error {
	_, err := c.getResponse("DELETE", fmt.Sprintf("/repos/%s/%s/releases/%d/assets/%d", user, repo, release, id), nil, nil)
	return err
}
