// Copyright © 2024 Meroxa, Inc.
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

package eventbus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-errors/errors"
	"google.golang.org/grpc/metadata"
)

const (
	tokenHeader      = "accesstoken"
	instanceHeader   = "instanceurl"
	tenantHeader     = "tenantid"
	loginEndpoint    = "/services/oauth2/token"
	userInfoEndpoint = "/services/oauth2/userinfo"
	oauthDialTimeout = 5 * time.Second
)

type authorizer interface {
	// Authorize retrieves new access token.
	Authorize(context.Context) error
	// Context returns context containing authorization credentials.
	Context(context.Context) context.Context
}

type oauth struct {
	// oauth2 settings
	clientID      string
	clientSecret  string
	oauthEndpoint *url.URL

	// protects access to authdata
	rw sync.RWMutex
	// authorized grpc metadata
	authdata metadata.MD
}

type token struct {
	ID          string `json:"id"`
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

type user struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"organization_id"`
}

type apiError struct {
	Err  string `json:"error"`
	Desc string `json:"error_description"`
}

// Error returns a string with the error and description.
func (e apiError) Error() string {
	return e.Err + ": " + e.Desc
}

func newAuthorizer(clientID, secret, endpoint string) (*oauth, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Errorf("failed to parse oauth endpoint url: %w", err)
	}

	return &oauth{
		clientID:      clientID,
		clientSecret:  secret,
		oauthEndpoint: u,
		authdata:      nil,
	}, nil
}

// Authorize retrives an oauth token and information about the associated user.
// Returns error when:
// * Failure to obtain token
// * Failure to obtain user data associated with the token.
func (a *oauth) Authorize(ctx context.Context) error {
	a.rw.Lock()
	defer a.rw.Unlock()

	token, err := a.obtainToken(ctx)
	if err != nil {
		return errors.Errorf("failed to login %q: %w", a.clientID, err)
	}

	user, err := a.obtainUser(ctx, token.AccessToken)
	if err != nil {
		return errors.Errorf("failed to retrieve user info %q: %w", a.clientID, err)
	}

	a.authdata = metadata.Pairs(
		tokenHeader, token.AccessToken,
		instanceHeader, token.InstanceURL,
		tenantHeader, user.OrgID,
	)

	return nil
}

// obtainToken request new oauth token from the salesforce endpoint.
// Returns error when the request fails.
func (a *oauth) obtainToken(ctx context.Context) (*token, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthDialTimeout)
	defer cancel()

	v := url.Values{}
	v.Set("grant_type", "client_credentials")
	v.Set("client_id", a.clientID)
	v.Set("client_secret", a.clientSecret)

	body := strings.NewReader(v.Encode())
	reqURL := a.oauthEndpoint.JoinPath(loginEndpoint).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return nil, errors.Errorf("failed to create token req: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("http oauth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("oauth req failed: %w", a.apiErr(resp.Body))
	}

	var t token
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, errors.Errorf("error decoding token %q response: %w", a.clientID, err)
	}

	return &t, nil
}

// userInfo retrives the organization and user ID associated with the token.
// Returns error when the request fails.
func (a *oauth) obtainUser(ctx context.Context, token string) (*user, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthDialTimeout)
	defer cancel()

	reqURL := a.oauthEndpoint.JoinPath(userInfoEndpoint).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Errorf("failed to make userinfo req: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("http userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("userinfo req failed: %w", a.apiErr(resp.Body))
	}

	var u user
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, errors.Errorf("failed to decode user info %q: %w", a.clientID, err)
	}

	return &u, nil
}

// apiErr extracts an error from the body reader.
// Returns the extracted error.
func (a *oauth) apiErr(r io.Reader) error {
	var decodedErr apiError
	if err := json.NewDecoder(r).Decode(&decodedErr); err != nil {
		return errors.Errorf("failed to decode error: %w", err)
	}

	return decodedErr
}

// Context returns new gRPC metadata context containing authentication data.
func (a *oauth) Context(ctx context.Context) context.Context {
	if a.authdata == nil {
		panic("need to authorize first")
	}

	a.rw.RLock()
	defer a.rw.RUnlock()

	return metadata.NewOutgoingContext(ctx, a.authdata)
}
