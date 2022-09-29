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

package cometd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/cometd/requests"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/cometd/responses"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/utils"
	"golang.org/x/net/publicsuffix"
)

type Client interface {
	Handshake(ctx context.Context) (responses.SuccessfulHandshakeResponse, error)
	Connect(ctx context.Context) (responses.ConnectResponse, error)
	SubscribeToPushTopic(ctx context.Context, pushTopic string) (responses.SubscribeResponse, error)
	UnsubscribeToPushTopic(ctx context.Context, pushTopic string) (responses.UnsubscribeResponse, error)
	Disconnect(ctx context.Context) (responses.DisconnectResponse, error)
}

func NewDefaultClient(baseURL, accessToken string) (Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	return &DefaultClient{
		baseURL:        baseURL,
		accessToken:    accessToken,
		longPollClient: &http.Client{Jar: jar},
	}, nil
}

type DefaultClient struct {
	baseURL        string
	accessToken    string
	clientID       string
	longPollClient *http.Client
}

// Handshake performs a handshake request.
// See: https://docs.cometd.org/current7/reference/#_bayeux_meta_handshake
func (s *DefaultClient) Handshake(ctx context.Context) (responses.SuccessfulHandshakeResponse, error) {
	// Prepare and send request
	responseData, err := s.httpPost(ctx, requests.HandshakeRequest{})
	if err != nil {
		return responses.SuccessfulHandshakeResponse{}, err
	}

	// Assume the handshake was successful
	var successfulResponses []responses.SuccessfulHandshakeResponse
	if err := json.Unmarshal(responseData, &successfulResponses); err == nil && len(successfulResponses) > 0 {
		s.clientID = successfulResponses[0].ClientID

		return successfulResponses[0], nil
	}

	// Assume the handshake was not successful
	var unsuccessfulResponses []responses.UnsuccessfulHandshakeResponseError
	if err := json.Unmarshal(responseData, &unsuccessfulResponses); err != nil {
		return responses.SuccessfulHandshakeResponse{}, fmt.Errorf("unable to process handshake response: %w", err)
	} else if len(unsuccessfulResponses) == 0 {
		return responses.SuccessfulHandshakeResponse{}, fmt.Errorf("unable to process handshake response: unexpected response")
	}

	return responses.SuccessfulHandshakeResponse{}, unsuccessfulResponses[0]
}

// Connect performs a connect request.
// See: https://docs.cometd.org/current7/reference/#_bayeux_meta_connect
func (s *DefaultClient) Connect(ctx context.Context) (responses.ConnectResponse, error) {
	// Prepare and send request
	responseData, err := s.httpPost(ctx, requests.ConnectRequest{
		ClientID: s.clientID,
	})
	if err != nil {
		return responses.ConnectResponse{}, fmt.Errorf("unable to perform connect request: %w", err)
	}

	// The response may contain response status alone or with additional response details.
	// Cannot be unmarshalled at once
	var successfulResponses []json.RawMessage
	if err := json.Unmarshal(responseData, &successfulResponses); err != nil {
		return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: %w", err)
	} else if len(successfulResponses) == 0 {
		return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: empty response")
	}

	connectResponse := responses.ConnectResponse{
		Events: make([]responses.ConnectResponseEvent, 0),
	}

	for _, response := range successfulResponses {
		// Unmarshal to determine the type
		var item map[string]interface{}
		if err := json.Unmarshal(response, &item); err != nil {
			return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: %w", err)
		}

		// Unmarshal event data
		if _, ok := item["data"].(map[string]interface{}); ok {
			var event responses.ConnectResponseEvent
			if err := json.Unmarshal(response, &event); err != nil {
				return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: %w", err)
			}

			connectResponse.Events = append(connectResponse.Events, event)

			continue
		}

		// Unmarshal response data
		if connectResponse.ClientID != "" {
			return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: multiple responses returned by the server")
		}

		if _, ok := item["successful"].(bool); ok {
			if err := json.Unmarshal(response, &connectResponse); err != nil {
				return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: %w", err)
			}

			continue
		}

		return responses.ConnectResponse{}, fmt.Errorf("unable to process connect response: unsupported repsonse data")
	}

	return connectResponse, nil
}

// SubscribeToPushTopic performs a subscribe to topic request.
// See: https://docs.cometd.org/current7/reference/#_bayeux_meta_subscribe
func (s *DefaultClient) SubscribeToPushTopic(ctx context.Context, pushTopic string) (responses.SubscribeResponse, error) {
	// Prepare and send request
	responseData, err := s.httpPost(ctx, requests.SubscribePushTopicRequest{
		ClientID:  s.clientID,
		PushTopic: pushTopic,
	})
	if err != nil {
		return responses.SubscribeResponse{}, fmt.Errorf("unable to perform subscribe request: %w", err)
	}

	// Assume is subscribed
	var successfulResponses []responses.SubscribeResponse
	if err := json.Unmarshal(responseData, &successfulResponses); err != nil {
		return responses.SubscribeResponse{}, fmt.Errorf("unable to process subscribe response: %w", err)
	} else if len(successfulResponses) == 0 {
		return responses.SubscribeResponse{}, fmt.Errorf("unable to process subscribe response: empty response")
	}

	return successfulResponses[0], nil
}

// UnsubscribeToPushTopic performs a unsubscribe from topic request.
// See: https://docs.cometd.org/current7/reference/#_bayeux_meta_unsubscribe
func (s *DefaultClient) UnsubscribeToPushTopic(ctx context.Context, pushTopic string) (responses.UnsubscribeResponse, error) {
	// Prepare and send request
	responseData, err := s.httpPost(ctx, requests.UnsubscribePushTopicRequest{
		ClientID:  s.clientID,
		PushTopic: pushTopic,
	})
	if err != nil {
		return responses.UnsubscribeResponse{}, fmt.Errorf("unable to perform unsubscribe request: %w", err)
	}

	// Assert is unsubscribed
	var successfulResponses []responses.UnsubscribeResponse
	if err := json.Unmarshal(responseData, &successfulResponses); err != nil {
		return responses.UnsubscribeResponse{}, fmt.Errorf("unable to process unsubscribe response: %w", err)
	} else if len(successfulResponses) == 0 {
		return responses.UnsubscribeResponse{}, fmt.Errorf("unable to process unsubscribe response: empty response")
	}

	return successfulResponses[0], nil
}

// Disconnect performs a disconnect request.
// See: https://docs.cometd.org/current7/reference/#_bayeux_meta_disconnect
func (s *DefaultClient) Disconnect(ctx context.Context) (responses.DisconnectResponse, error) {
	// Prepare and send request
	responseData, err := s.httpPost(ctx, requests.DisconnectRequest{
		ClientID: s.clientID,
	})
	if err != nil {
		return responses.DisconnectResponse{}, fmt.Errorf("unable to perform disconnect request: %w", err)
	}

	// Assert is disconnected
	var successfulResponses []responses.DisconnectResponse
	if err := json.Unmarshal(responseData, &successfulResponses); err != nil {
		return responses.DisconnectResponse{}, fmt.Errorf("unable to process disconnect response: %w", err)
	} else if len(successfulResponses) == 0 {
		return responses.DisconnectResponse{}, fmt.Errorf("unable to process disconnect response: empty response")
	}

	return successfulResponses[0], nil
}

// httpPost sends a POST request to the CometD server
func (s *DefaultClient) httpPost(ctx context.Context, payload requests.Request) ([]byte, error) {
	// Prepare the payload
	requestData, err := payload.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	buff.Write(requestData)

	// Prepare the Request
	request, err := http.NewRequestWithContext(
		ctx,
		"POST",
		s.baseURL,
		&buff,
	)
	if err != nil {
		return nil, err
	}

	// Add headers
	request.Header.Set("Authorization", fmt.Sprintf("OAuth %s", s.accessToken))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "gzip;q=1.0, *;q=0.1")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ConduitIO/Salesforce-v0.1.0")

	// Execute the Request
	resp, err := s.longPollClient.Do(request)
	if err != nil {
		return nil, err
	}

	// Read the body
	respBytes, err := utils.DecodeHTTPResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("could not read response data: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("could not read response data: %w", err)
	}

	return respBytes, nil
}
