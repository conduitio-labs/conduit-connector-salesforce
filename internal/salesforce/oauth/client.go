package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/conduitio/conduit-connector-salesforce/internal/salesforce/oauth/response"
	"github.com/conduitio/conduit-connector-salesforce/internal/utils"
)

type Environment = string

const (
	EnvironmentSandbox Environment = "sandbox"

	grantType    = "password"
	loginURI     = "https://login.salesforce.com/services/oauth2/token"
	testLoginURI = "https://test.salesforce.com/services/oauth2/token"
)

func NewClient(
	environment Environment,
	clientId string,
	clientSecret string,
	username string,
	password string,
	securityToken string,
) *Client {
	return &Client{
		environment:   environment,
		clientId:      clientId,
		clientSecret:  clientSecret,
		username:      username,
		password:      password,
		securityToken: securityToken,
	}
}

type Client struct {
	environment   Environment
	clientId      string
	clientSecret  string
	username      string
	password      string
	securityToken string
}

func (a *Client) Authenticate() (response.TokenResponse, error) {
	payload := url.Values{
		"grant_type":    {grantType},
		"client_id":     {a.clientId},
		"client_secret": {a.clientSecret},
		"username":      {a.username},
		"password":      {fmt.Sprintf("%v%v", a.password, a.securityToken)},
	}

	// Build Uri
	uri := loginURI
	if EnvironmentSandbox == a.environment {
		uri = testLoginURI
	}

	// Build Body
	body := strings.NewReader(payload.Encode())

	// Build Request
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("failed to prepare authentication request: %w", err)
	}

	// Add Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip;q=1.0, *;q=0.1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "ConduitIO/Salesforce-v0.1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("failed to send authentication request: %w", err)
	}

	respBytes, err := utils.DecodeHttpResponse(resp)
	if err != nil {
		return response.TokenResponse{}, fmt.Errorf("could not read response data: %w", err)
	}

	fmt.Println("OAuth response", string(respBytes))

	// Attempt to parse successful response
	var token response.TokenResponse
	if err := json.Unmarshal(respBytes, &token); err == nil {
		return token, nil
	}

	// Attempt to parse response as a force.com api error
	authFailureResponse := response.FailureResponse{}

	if err := json.Unmarshal(respBytes, &authFailureResponse); err != nil {
		return response.TokenResponse{}, fmt.Errorf("unable to process authentication response: %w", err)
	}

	return response.TokenResponse{}, authFailureResponse
}
