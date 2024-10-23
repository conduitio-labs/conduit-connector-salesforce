// Copyright © 2022 Meroxa, Inc. and Miquido
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

package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type authenticator interface {
	Login() (*LoginResponse, error)
	UserInfo(string) (*UserInfoResponse, error)
}

const (
	loginEndpoint    = "/services/oauth2/token"
	userInfoEndpoint = "/services/oauth2/userinfo"
)

var OAuthDialTimeout = 5 * time.Second

type LoginResponse struct {
	AccessToken      string `json:"access_token"`
	InstanceURL      string `json:"instance_url"`
	ID               string `json:"id"`
	TokenType        string `json:"token_type"`
	IssuedAt         string `json:"issued_at"`
	Signature        string `json:"signature"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (l LoginResponse) Err() error {
	return fmt.Errorf("%s: %s", l.Error, l.ErrorDescription)
}

type UserInfoResponse struct {
	UserID           string `json:"user_id"`
	OrganizationID   string `json:"organization_id"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (u UserInfoResponse) Err() error {
	return fmt.Errorf("%s: %s", u.Error, u.ErrorDescription)
}

type Credentials struct {
	ClientID, ClientSecret string
	OAuthEndpoint          *url.URL
}

func NewCredentials(clientID, secret, endpoint string) (Credentials, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to parse oauth endpoint url: %w", err)
	}

	return Credentials{
		ClientID:      clientID,
		ClientSecret:  secret,
		OAuthEndpoint: u,
	}, nil
}

type oauth struct {
	Credentials
}

func (a oauth) Login() (*LoginResponse, error) {
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	body.Set("client_id", a.ClientID)
	body.Set("client_secret", a.ClientSecret)

	ctx, cancelFn := context.WithTimeout(context.Background(), OAuthDialTimeout)
	defer cancelFn()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.loginURL(),
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make login req: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http oauth request failed: %w", err)
	}
	defer httpResp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("error decoding login response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected oauth login (status: %d) response: %w", httpResp.StatusCode, loginResp.Err(),
		)
	}

	return &loginResp, nil
}

func (a oauth) UserInfo(accessToken string) (*UserInfoResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), OAuthDialTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.userInfoURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make userinfo req: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting user info response - %s", err)
	}

	defer httpResp.Body.Close()

	var userInfoResp UserInfoResponse
	err = json.NewDecoder(httpResp.Body).Decode(&userInfoResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding user info - %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected oauth userInfo (status: %d) response: %w", httpResp.StatusCode, userInfoResp.Err(),
		)
	}

	return &userInfoResp, nil
}

func (a oauth) loginURL() string {
	return a.OAuthEndpoint.JoinPath(loginEndpoint).String()
}

func (a oauth) userInfoURL() string {
	return a.OAuthEndpoint.JoinPath(userInfoEndpoint).String()
}
