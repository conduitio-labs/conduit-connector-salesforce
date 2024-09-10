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

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/salesforce/oauth/response"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/utils"
)

type Environment = string

const (
	EnvironmentSandbox Environment = "sandbox"

	grantType    = "password"
	loginURI     = "https://login.salesforce.com/services/oauth2/token"
	testLoginURI = "https://test.salesforce.com/services/oauth2/token"
)

// skip go:generate moq -out client_moq_test.go . Client.
type Client interface {
	Authenticate(ctx context.Context) (response.TokenResponse, error)
}

func NewDefaultClient(
	environment Environment,
	clientID string,
	clientSecret string,
	username string,
	password string,
	securityToken string,
) Client {
	return &DefaultClient{
		httpClient:    http.DefaultClient,
		environment:   environment,
		clientID:      clientID,
		clientSecret:  clientSecret,
		username:      username,
		password:      password,
		securityToken: securityToken,
	}
}

type DefaultClient struct {
	httpClient    httpClient
	environment   Environment
	clientID      string
	clientSecret  string
	username      string
	password      string
	securityToken string
}

// skip go:generate moq -out http_client_moq_test.go . httpClient.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Authenticate attempts to authenticate the client with given credentials.
func (a *DefaultClient) Authenticate(ctx context.Context) (response.TokenResponse, error) {
	// Prepare request payload
	payload := url.Values{
		"grant_type":    {grantType},
		"client_id":     {a.clientID},
		"client_secret": {a.clientSecret},
		"username":      {a.username},
		"password":      {fmt.Sprintf("%v%v", a.password, a.securityToken)},
	}

	// Build URI
	uri := loginURI
	if EnvironmentSandbox == a.environment {
		uri = testLoginURI
	}

	// Build Body
	body := strings.NewReader(payload.Encode())

	// Build Request
	req, err := http.NewRequestWithContext(ctx, "POST", uri, body)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("failed to prepare authentication request: %w", err)
	}

	// Add Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip;q=1.0, *;q=0.1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "ConduitIO/Salesforce-v0.1.0")

	// Execute Request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("failed to send authentication request: %w", err)
	}

	// Read the body
	respBytes, err := utils.DecodeHTTPResponse(resp)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("could not read response data: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return response.TokenResponse{}, fmt.Errorf("could not read response data: %w", err)
	}

	// Attempt to parse successful response
	var token response.TokenResponse
	if err := json.Unmarshal(respBytes, &token); err == nil && token.AccessToken != "" && token.InstanceURL != "" {
		return token, nil
	}

	// Attempt to parse failure response
	authFailureResponse := response.FailureResponseError{}
	if err := json.Unmarshal(respBytes, &authFailureResponse); err != nil {
		return response.TokenResponse{}, fmt.Errorf("unable to process authentication response: %w", err)
	}

	return response.TokenResponse{}, authFailureResponse
}
