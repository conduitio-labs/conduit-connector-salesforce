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
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

type UserInfoResponse struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
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

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code returned on OAuth authentication call: %v", httpResp.StatusCode)
	}

	var loginResponse LoginResponse
	err = json.NewDecoder(httpResp.Body).Decode(&loginResponse)
	if err != nil {
		return nil, err
	}

	return &loginResponse, nil
}

func UserInfo(oauthEndpoint, accessToken string) (*UserInfoResponse, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), OAuthDialTimeout)
	defer cancelFn()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthEndpoint+userInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code returned on OAuth user info call: %v", httpResp.StatusCode)
	}

	var userInfoResponse UserInfoResponse
	err = json.NewDecoder(httpResp.Body).Decode(&userInfoResponse)
	if err != nil {
		return nil, err
	}

	return &userInfoResponse, nil
}
