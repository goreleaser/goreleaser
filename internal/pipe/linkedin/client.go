package linkedin

import (
	"bytes"
	stdctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"golang.org/x/oauth2"
)

var ErrLinkedinForbidden = errors.New("forbidden. please check your permissions")

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

type profileResponse struct {
	ID  string `json:"id"`
	Sub string `json:"sub"`
}

type shareResponse struct {
	Activity string `json:"activity"`
}

func createLinkedInClient(cfg oauthClientConfig) (client, error) {
	if cfg.Context == nil {
		return client{}, errors.New("context is nil")
	}

	if cfg.AccessToken == "" {
		return client{}, errors.New("empty access token")
	}

	config := oauth2.Config{}

	c := config.Client(cfg.Context, &oauth2.Token{
		AccessToken: cfg.AccessToken,
	})

	if c == nil {
		return client{}, errors.New("client is nil")
	}

	return client{
		client:  c,
		baseURL: "https://api.linkedin.com",
	}, nil
}

// getProfileIDLegacy returns the Current Member's ID
// it's legacy because it uses deprecated v2/me endpoint, that requires old permissions such as r_liteprofile
// POST Share API requires a Profile ID in the 'owner' field
// Format must be in: 'urn:li:person:PROFILE_ID'
// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api#retrieve-current-members-profile
func (c client) getProfileIDLegacy(ctx stdctx.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/me", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("could not create GET request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not GET /v2/me: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return "", ErrLinkedinForbidden
	}

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return "", retryx.HTTP(gerrors.Wrap(
			fmt.Errorf("GET /v2/me returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			gerrors.WithOutput(string(body)),
		), resp)
	}

	value, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	var result profileResponse
	if err := json.Unmarshal(value, &result); err != nil {
		return "", fmt.Errorf("could not unmarshal: %w", err)
	}

	if result.ID == "" {
		return "", errors.New("could not find 'id' in result")
	}
	return result.ID, nil
}

// getProfileSub returns the Current Member's sub (formally ID) - requires 'profile' permission
// POST Share API requires a Profile ID in the 'owner' field
// Format must be in: 'urn:li:person:PROFILE_SUB'
// https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin-v2#api-request-to-retreive-member-details
func (c client) getProfileSub(ctx stdctx.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/userinfo", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("could not create GET request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not GET /v2/userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return "", ErrLinkedinForbidden
	}

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return "", retryx.HTTP(gerrors.Wrap(
			fmt.Errorf("GET /v2/userinfo returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			gerrors.WithOutput(string(body)),
		), resp)
	}

	value, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	var result profileResponse
	if err := json.Unmarshal(value, &result); err != nil {
		return "", fmt.Errorf("could not unmarshal: %w", err)
	}

	if result.Sub == "" {
		return "", errors.New("could not find 'sub' in result")
	}
	return result.Sub, nil
}

// Person or Organization URN - urn:li:person:PROFILE_IDENTIFIER
// Owner of the share. Required on create.
// tries to get the profile sub (formally id) first, if it fails, it tries to get the profile id (legacy)
// https://docs.microsoft.com/en-us/linkedin/marketing/integrations/community-management/shares/share-api?tabs=http#schema
func (c client) getProfileURN(ctx stdctx.Context) (string, error) {
	// To build the URN, we need to get the profile sub (formally id)
	profileSub, err := c.getProfileSub(ctx)
	if err != nil {
		if !errors.Is(err, ErrLinkedinForbidden) {
			return "", fmt.Errorf("could not get profile sub: %w", err)
		}

		log.Debug("could not get linkedin profile sub due to permission, getting profile id (legacy)")

		profileSub, err = c.getProfileIDLegacy(ctx)
		if err != nil {
			return "", fmt.Errorf("could not get profile id: %w", err)
		}
	}

	return fmt.Sprintf("urn:li:person:%s", profileSub), nil
}

func (c client) Share(ctx stdctx.Context, message string) (string, error) {
	// To get Owner of the share, we need to get the profile URN
	profileURN, err := c.getProfileURN(ctx)
	if err != nil {
		return "", fmt.Errorf("could not get profile URN: %w", err)
	}

	reqBody := postShareRequest{
		Text: postShareText{
			Text: message,
		},
		Owner: profileURN,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("could not marshal request: %w", err)
	}

	// Filling only required 'owner' and 'text' field is OK
	// https://docs.microsoft.com/en-us/linkedin/marketing/integrations/community-management/shares/share-api?tabs=http#sample-request-3
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/shares", bytes.NewReader(reqBodyBytes))
	if err != nil {
		return "", fmt.Errorf("could not create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not POST /v2/shares: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return "", retryx.HTTP(gerrors.Wrap(
			fmt.Errorf("POST /v2/shares returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			gerrors.WithOutput(string(body)),
		), resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read from body: %w", err)
	}

	var result shareResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("could not unmarshal: %w", err)
	}

	// Activity URN
	// URN of the activity associated with this share. Activities act as a wrapper around
	// shares and articles to represent content in the LinkedIn feed. Read only.
	if result.Activity == "" {
		return "", errors.New("could not find 'activity' in result")
	}
	return fmt.Sprintf("https://www.linkedin.com/feed/update/%s", result.Activity), nil
}
