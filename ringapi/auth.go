package ringapi

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

var (
	oauthConfig = oauth2.Config{
		ClientID: "ring_official_android",
		Scopes:   []string{"client"},
		Endpoint: oauth2.Endpoint{
			TokenURL:  "https://oauth.ring.com/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
)

// TokenHandler implements routines for Fetch'ing and Store'ing
// the OAUTH2 Token from some persistence. This is especially useful
// when 2FA is in use.
type TokenHandler interface {
	FetchToken() *oauth2.Token
	StoreToken(*oauth2.Token)
}

// Authenticator implements routines for capturing info necessary
// to authenticate the user and authorize the token for use with this
// application.
type Authenticator interface {
	PromptCredentials() (string, string, error)
	Prompt2FACode() (string, error)
}

type interceptRoundTripper struct {
	code string
}

func (i *interceptRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Intercept the Token request to inject the 2FA code into the header.
	// This is how ring API does 2FA.
	req.Header.Add("2fa-support", "true")
	req.Header.Add("2fa-code", i.code)
	return http.DefaultTransport.RoundTrip(req)
}

// OpenAuthorizedSession creates a new AuthorizedSession instance by
// retreiving an OAUTH2 Token. This token might already exist, in which case
// the Authenticator is not needed. If the token does not exist and the
// Authenticator is nil (maybe because your application is not user-interactive)
// this function will fail.
func OpenAuthorizedSession(cfg ApiConfig, t TokenHandler, a Authenticator) (*AuthorizedSession, error) {

	if t == nil {
		return nil, fmt.Errorf("TokenHandler is required")
	}
	token := t.FetchToken()

	if token == nil {
		if a == nil {
			return nil, fmt.Errorf("No token found and no Authenticator was provided")
		}

		log.Printf("No previously stored token found. Will need to authorize")

		u, p, err := a.PromptCredentials()
		if err != nil {
			return nil, err
		}

		// Attempt to get an OAUTH token by password. This will typically fail
		// but send a prompt to the user's phone or whatever with a code to enter.
		token, err = oauthConfig.PasswordCredentialsToken(oauth2.NoContext, u, p)
		if err == nil {
			log.Printf("Password-only auth worked")
		} else {
			rErr, ok := err.(*oauth2.RetrieveError)
			if !ok {
				return nil, err
			}

			// 412 (Precondition Failed) is how ring indicates 2FA is in play
			if rErr.Response.StatusCode != 412 {
				return nil, err
			}

			log.Printf("Ring indicates 2FA code needed")

			// Prompt for that code and then we'll do it again.
			code, err := a.Prompt2FACode()
			if err != nil {
				return nil, fmt.Errorf("Failure prompting for 2FA: %v", err)
			}

			client := &http.Client{
				Transport: &interceptRoundTripper{
					code: code,
				},
			}
			ctx := context.WithValue(oauth2.NoContext, oauth2.HTTPClient, client)

			token, err = oauthConfig.PasswordCredentialsToken(ctx, u, p)
			if err != nil {
				return nil, err
			}

			// We got the token via 2FA
		}

		t.StoreToken(token)
	}

	log.Printf("Token acquired")

	session := &AuthorizedSession{
		client: oauthConfig.Client(oauth2.NoContext, token),
		config: cfg,
	}

	return session, nil
}
