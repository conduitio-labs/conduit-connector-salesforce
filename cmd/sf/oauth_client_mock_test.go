package main

import (
	"context"

	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth/response"
)

type oAuthClientMock struct {
}

func (c *oAuthClientMock) Authenticate(_ context.Context) (response.TokenResponse, error) {
	return response.TokenResponse{
		AccessToken: "access-token",
		InstanceURL: "hxxp://instance.url",
		ID:          "1",
		IssuedAt:    "2",
		Signature:   "3",
	}, nil
}
