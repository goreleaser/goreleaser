package packslip

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/context"
	packslip "github.com/jdx/packslip/go"
)

const sigstoreAudience = "sigstore"

// oidcToken returns the OIDC token that Sigstore's Fulcio exchanges for a
// signing certificate: SIGSTORE_ID_TOKEN when set, otherwise the ambient
// GitHub Actions token.
func oidcToken(ctx *context.Context) (string, error) {
	if token := os.Getenv("SIGSTORE_ID_TOKEN"); token != "" {
		return token, nil
	}
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if requestURL == "" || requestToken == "" {
		return "", errors.New("no OIDC token found: set SIGSTORE_ID_TOKEN, or grant the GitHub Actions job `id-token: write` permission")
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("invalid ACTIONS_ID_TOKEN_REQUEST_URL: %w", err)
	}
	q := u.Query()
	q.Set("audience", sigstoreAudience)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not request GitHub Actions OIDC token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not request GitHub Actions OIDC token: %s", resp.Status)
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", fmt.Errorf("could not read GitHub Actions OIDC token: %w", err)
	}
	if body.Value == "" {
		return "", errors.New("GitHub Actions returned an empty OIDC token")
	}
	return body.Value, nil
}

// identityFromToken derives the identity Fulcio will put in the signing
// certificate from the token's claims. The token is not verified here: Fulcio
// does that, and packslip rejects a certificate that does not match this
// identity before anything is logged.
func identityFromToken(token string) (packslip.Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return packslip.Identity{}, errors.New("the OIDC token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return packslip.Identity{}, fmt.Errorf("could not decode the OIDC token: %w", err)
	}
	var claims struct {
		Issuer         string `json:"iss"`
		Email          string `json:"email"`
		EmailVerified  bool   `json:"email_verified"`
		JobWorkflowRef string `json:"job_workflow_ref"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return packslip.Identity{}, fmt.Errorf("could not decode the OIDC token: %w", err)
	}
	identity := packslip.Identity{Scheme: "sigstore-oidc", Issuer: claims.Issuer}
	switch {
	case claims.JobWorkflowRef != "":
		identity.KeyID = "https://github.com/" + claims.JobWorkflowRef
	case claims.Email != "" && claims.EmailVerified:
		identity.KeyID = claims.Email
	default:
		return packslip.Identity{}, errors.New("unsupported OIDC token: packslip signing supports GitHub Actions workflows and verified-email identities")
	}
	if identity.Issuer == "" {
		return packslip.Identity{}, errors.New("the OIDC token has no issuer")
	}
	return identity, nil
}
