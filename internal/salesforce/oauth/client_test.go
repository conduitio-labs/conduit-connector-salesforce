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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jaswdr/faker"
	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth/response"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("New Client can be created", func(t *testing.T) {
		var (
			environment   = fakerInstance.Lorem().Word()
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
		)

		client := NewDefaultClient(
			environment,
			clientID,
			clientSecret,
			username,
			password,
			securityToken,
		).(*DefaultClient)

		require.IsType(t, &DefaultClient{}, client)
		require.Same(t, http.DefaultClient, client.httpClient)
		require.Equal(t, environment, client.environment)
		require.Equal(t, clientID, client.clientID)
		require.Equal(t, clientSecret, client.clientSecret)
		require.Equal(t, username, client.username)
		require.Equal(t, password, client.password)
		require.Equal(t, securityToken, client.securityToken)
	})
}

func TestClient_Authenticate(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("Returns error when request fails", func(t *testing.T) {
		var (
			environment   = fakerInstance.Lorem().Word()
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				return nil, errors.New("request failed")
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		_, err := client.Authenticate(context.TODO())
		require.EqualError(t, err, "failed to send authentication request: request failed")

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Returns error when request returns unsupported encoding", func(t *testing.T) {
		var (
			environment     = fakerInstance.Lorem().Word()
			clientID        = fakerInstance.RandomStringWithLength(32)
			clientSecret    = fakerInstance.RandomStringWithLength(32)
			username        = fakerInstance.Lorem().Sentence(6)
			password        = fakerInstance.Lorem().Sentence(6)
			securityToken   = fakerInstance.RandomStringWithLength(32)
			contentEncoding = fakerInstance.Lorem().Sentence(2)
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				return &http.Response{
					Header: http.Header{
						"Content-Encoding": {contentEncoding},
					},
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		_, err := client.Authenticate(context.TODO())
		require.EqualError(t, err, fmt.Sprintf("could not read response data: unsupported encoding %q", contentEncoding))

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Returns error when request body could not be parsed as FailureResponseError", func(t *testing.T) {
		var (
			environment   = fakerInstance.Lorem().Word()
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				return &http.Response{
					Body: io.NopCloser(strings.NewReader("nil")),
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		_, err := client.Authenticate(context.TODO())
		require.EqualError(t, err, "unable to process authentication response: invalid character 'i' in literal null (expecting 'u')")

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Returns error when request body could not be parsed as FailureResponseError 2", func(t *testing.T) {
		var (
			environment      = fakerInstance.Lorem().Word()
			clientID         = fakerInstance.RandomStringWithLength(32)
			clientSecret     = fakerInstance.RandomStringWithLength(32)
			username         = fakerInstance.Lorem().Sentence(6)
			password         = fakerInstance.Lorem().Sentence(6)
			securityToken    = fakerInstance.RandomStringWithLength(32)
			closeErrorReason = fakerInstance.Lorem().Sentence(6)
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				return &http.Response{
					Body: NewFailedReadCloserMock(strings.NewReader("nil"), errors.New(closeErrorReason)),
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		_, err := client.Authenticate(context.TODO())
		require.EqualError(t, err, fmt.Sprintf("could not read response data: %s", closeErrorReason))

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Returns FailureResponseError when request body could be parsed", func(t *testing.T) {
		var (
			environment   = fakerInstance.Lorem().Word()
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
			responseError = response.FailureResponseError{
				ErrorName:        fakerInstance.Lorem().Sentence(3),
				ErrorDescription: fakerInstance.Lorem().Sentence(6),
			}
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				payload, err := json.Marshal(responseError)
				require.NoError(t, err)

				return &http.Response{
					Body: io.NopCloser(bytes.NewReader(payload)),
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		_, err := client.Authenticate(context.TODO())
		require.Equal(t, responseError, err)

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Returns TokenResponse when request was successful", func(t *testing.T) {
		var (
			environment   = fakerInstance.Lorem().Word()
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
			tokenResponse = response.TokenResponse{
				AccessToken: fakerInstance.UUID().V4(),
				InstanceURL: fakerInstance.Internet().URL(),
				ID:          fakerInstance.Numerify("#####"),
				IssuedAt:    fmt.Sprintf("%d", fakerInstance.UInt32()),
				Signature:   fakerInstance.Hash().SHA512(),
			}
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, loginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				payload, err := json.Marshal(tokenResponse)
				require.NoError(t, err)

				return &http.Response{
					Body: io.NopCloser(bytes.NewReader(payload)),
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		token, err := client.Authenticate(context.TODO())
		require.NoError(t, err)
		require.Equal(t, tokenResponse, token)

		require.Len(t, hcMock.DoCalls(), 1)
	})

	t.Run("Uses test login URI when environment is sandbox", func(t *testing.T) {
		var (
			environment   = EnvironmentSandbox
			clientID      = fakerInstance.RandomStringWithLength(32)
			clientSecret  = fakerInstance.RandomStringWithLength(32)
			username      = fakerInstance.Lorem().Sentence(6)
			password      = fakerInstance.Lorem().Sentence(6)
			securityToken = fakerInstance.RandomStringWithLength(32)
			tokenResponse = response.TokenResponse{
				AccessToken: fakerInstance.UUID().V4(),
				InstanceURL: fakerInstance.Internet().URL(),
				ID:          fakerInstance.Numerify("#####"),
				IssuedAt:    fmt.Sprintf("%d", fakerInstance.UInt32()),
				Signature:   fakerInstance.Hash().SHA512(),
			}
		)

		hcMock := httpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, testLoginURI, req.URL.String())

				body, err := req.GetBody()
				require.NoError(t, err)

				allBody, err := io.ReadAll(body)
				require.NoError(t, err)

				parsedQuery, err := url.ParseQuery(string(allBody))
				require.NoError(t, err)
				require.Equal(t, grantType, parsedQuery["grant_type"][0])
				require.Equal(t, clientID, parsedQuery["client_id"][0])
				require.Equal(t, clientSecret, parsedQuery["client_secret"][0])
				require.Equal(t, username, parsedQuery["username"][0])
				require.Equal(t, fmt.Sprintf("%s%s", password, securityToken), parsedQuery["password"][0])

				payload, err := json.Marshal(tokenResponse)
				require.NoError(t, err)

				return &http.Response{
					Body: io.NopCloser(bytes.NewReader(payload)),
				}, nil
			},
		}

		client := DefaultClient{
			httpClient:    &hcMock,
			environment:   environment,
			clientID:      clientID,
			clientSecret:  clientSecret,
			username:      username,
			password:      password,
			securityToken: securityToken,
		}

		token, err := client.Authenticate(context.TODO())
		require.NoError(t, err)
		require.Equal(t, tokenResponse, token)

		require.Len(t, hcMock.DoCalls(), 1)
	})
}
