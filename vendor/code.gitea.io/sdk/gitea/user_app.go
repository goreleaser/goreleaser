// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// BasicAuthEncode generate base64 of basic auth head
func BasicAuthEncode(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// AccessToken represents an API access token.
type AccessToken struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Token          string `json:"sha1"`
	TokenLastEight string `json:"token_last_eight"`
}

// ListAccessTokens lista all the access tokens of user
func (c *Client) ListAccessTokens(user, pass string) ([]*AccessToken, error) {
	tokens := make([]*AccessToken, 0, 10)
	return tokens, c.getParsedResponse("GET", fmt.Sprintf("/users/%s/tokens", user),
		http.Header{"Authorization": []string{"Basic " + BasicAuthEncode(user, pass)}}, nil, &tokens)
}

// CreateAccessTokenOption options when create access token
type CreateAccessTokenOption struct {
	Name string `json:"name"`
}

// CreateAccessToken create one access token with options
func (c *Client) CreateAccessToken(user, pass string, opt CreateAccessTokenOption) (*AccessToken, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	t := new(AccessToken)
	return t, c.getParsedResponse("POST", fmt.Sprintf("/users/%s/tokens", user),
		http.Header{
			"content-type":  []string{"application/json"},
			"Authorization": []string{"Basic " + BasicAuthEncode(user, pass)}},
		bytes.NewReader(body), t)
}

// DeleteAccessToken delete token with key id
func (c *Client) DeleteAccessToken(user string, keyID int64) error {
	_, err := c.getResponse("DELETE", fmt.Sprintf("/users/%s/tokens/%d", user, keyID),
		http.Header{"Authorization": []string{"Basic " + BasicAuthEncode(user, c.password)}}, nil)
	return err
}
