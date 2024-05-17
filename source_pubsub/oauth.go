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

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

type UserInfoResponse struct {
	UserID           string `json:"user_id"`
	OrganizationID   string `json:"organization_id"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type Credentials struct {
	ClientID, ClientSecret, OAuthEndpoint string
}

func Login(creds Credentials) (*LoginResponse, error) {
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	body.Set("client_id", creds.ClientID)
	body.Set("client_secret", creds.ClientSecret)

	ctx, cancelFn := context.WithTimeout(context.Background(), OAuthDialTimeout)
	defer cancelFn()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		creds.OAuthEndpoint+"/services/oauth2/token", strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	var loginResponse LoginResponse
	err = json.NewDecoder(httpResp.Body).Decode(&loginResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding login response - %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"non-200 status code returned on OAuth authentication call code=%v, error=%v, error_description=%v",
			httpResp.StatusCode,
			loginResponse.Error,
			loginResponse.ErrorDescription,
		)
	}

	return &loginResponse, nil
}

func UserInfo(oauthEndpoint, accessToken string) (*UserInfoResponse, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), OAuthDialTimeout)
	defer cancelFn()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthEndpoint+userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error making user info request - %s", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting user info response - %s", err)
	}

	defer httpResp.Body.Close()

	var userInfoResponse UserInfoResponse
	err = json.NewDecoder(httpResp.Body).Decode(&userInfoResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding user info - %s", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code returned on OAuth user info call: code=%v, error=%v, error_description=%v",
			httpResp.StatusCode,
			userInfoResponse.Error,
			userInfoResponse.ErrorDescription,
		)
	}

	return &userInfoResponse, nil
}
