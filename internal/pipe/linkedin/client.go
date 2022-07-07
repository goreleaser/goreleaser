package linkedin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/oauth2"
)

type oauthClientConfig struct {
	Context     *context.Context
	AccessToken string
}

type client struct {
	client  *http.Client
	baseURL string
}

type postShareText struct {
	Text string `json:"text"`
}

type postShareRequest struct {
	Text  postShareText `json:"text"`
	Owner string        `json:"owner"`
}

func createLinkedInClient(cfg oauthClientConfig) (client, error) {
	if cfg.Context == nil {
		return client{}, fmt.Errorf("context is nil")
	}

	if cfg.AccessToken == "" {
		return client{}, fmt.Errorf("empty access token")
	}

	config := oauth2.Config{}

	c := config.Client(cfg.Context, &oauth2.Token{
		AccessToken: cfg.AccessToken,
	})

	if c == nil {
		return client{}, fmt.Errorf("client is nil")
	}

	return client{
		client:  c,
		baseURL: "https://api.linkedin.com",
	}, nil
}

// getProfileID returns the Current Member's ID
// POST Share API requires a Profile ID in the 'owner' field
// Format must be in: 'urn:li:person:PROFILE_ID'
// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api#retrieve-current-members-profile
func (c client) getProfileID() (string, error) {
	resp, err := c.client.Get(c.baseURL + "/v2/me")
	if err != nil {
		return "", fmt.Errorf("could not GET /v2/me: %w", err)
	}

	value, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(value, &result)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal: %w", err)
	}

	if v, ok := result["id"]; ok {
		return v.(string), nil
	}

	return "", fmt.Errorf("could not find 'id' in result: %w", err)
}

func (c client) Share(message string) (string, error) {
	// To get Owner of the share, we need to get profile id
	profileID, err := c.getProfileID()
	if err != nil {
		return "", fmt.Errorf("could not get profile id: %w", err)
	}

	req := postShareRequest{
		Text: postShareText{
			Text: message,
		},
		// Person or Organization URN
		// Owner of the share. Required on create.
		// https://docs.microsoft.com/en-us/linkedin/marketing/integrations/community-management/shares/share-api?tabs=http#schema
		Owner: fmt.Sprintf("urn:li:person:%s", profileID),
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("could not marshal request: %w", err)
	}

	// Filling only required 'owner' and 'text' field is OK
	// https://docs.microsoft.com/en-us/linkedin/marketing/integrations/community-management/shares/share-api?tabs=http#sample-request-3
	resp, err := c.client.Post(c.baseURL+"/v2/shares", "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return "", fmt.Errorf("could not POST /v2/shares: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read from body: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal: %w", err)
	}

	// Activity URN
	// URN of the activity associated with this share. Activities act as a wrapper around
	// shares and articles to represent content in the LinkedIn feed. Read only.
	if v, ok := result["activity"]; ok {
		return fmt.Sprintf("https://www.linkedin.com/feed/update/%s", v.(string)), nil
	}

	return "", fmt.Errorf("could not find 'activity' in result: %w", err)
}
