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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matryer/is"
	"google.golang.org/grpc/metadata"
)

func Test_Authorize(t *testing.T) {
	tests := []struct {
		name    string
		auth    func(t *testing.T) authorizer
		wantErr error
	}{
		{
			name: "failed to obtain token",
			auth: func(t *testing.T) authorizer {
				t.Helper()
				is := is.New(t)

				mux := http.NewServeMux()
				mux.HandleFunc(loginEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, err := w.Write([]byte(`{"error":"failed","error_description":"badly"}`))
					is.NoErr(err)
				})
				mux.HandleFunc(userInfoEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
				u := newAuthServer(t, mux)

				auther, err := newAuthorizer("client123", "secret123", u)
				is.NoErr(err)
				return auther
			},
			wantErr: errors.New(`failed to login "client123": oauth req failed: failed: badly`),
		},
		{
			name: "failed to decode token",
			auth: func(t *testing.T) authorizer {
				t.Helper()
				is := is.New(t)

				mux := http.NewServeMux()
				mux.HandleFunc(loginEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`garble`))
					is.NoErr(err)
				})
				mux.HandleFunc(userInfoEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
				u := newAuthServer(t, mux)

				auther, err := newAuthorizer("client123", "secret123", u)
				is.NoErr(err)
				return auther
			},
			wantErr: errors.New(`failed to login "client123": error decoding token "client123" response: invalid character 'g' looking for beginning of value`),
		},
		{
			name: "failed to obtain user info",
			auth: func(t *testing.T) authorizer {
				t.Helper()
				is := is.New(t)

				mux := http.NewServeMux()
				mux.HandleFunc(loginEndpoint, func(w http.ResponseWriter, r *http.Request) {
					data, _ := io.ReadAll(r.Body)
					is.Equal(
						string(data),
						"client_id=client123&client_secret=secret123&grant_type=client_credentials",
					)
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(&token{
						ID:          "test123",
						AccessToken: "access-token-123",
						InstanceURL: "foobar.sfdc.cloud",
					})
					is.NoErr(err)
				})
				mux.HandleFunc(userInfoEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					_, err := w.Write([]byte(`{"error":"failed","error_description":"badly"}`))
					is.NoErr(err)
				})
				u := newAuthServer(t, mux)

				auther, err := newAuthorizer("client123", "secret123", u)
				is.NoErr(err)
				return auther
			},
			wantErr: errors.New(`failed to retrieve user info "client123": userinfo req failed: failed: badly`),
		},
		{
			name: "failed to decode user info",
			auth: func(t *testing.T) authorizer {
				t.Helper()
				is := is.New(t)

				mux := http.NewServeMux()
				mux.HandleFunc(loginEndpoint, func(w http.ResponseWriter, r *http.Request) {
					data, _ := io.ReadAll(r.Body)
					is.Equal(
						string(data),
						"client_id=client123&client_secret=secret123&grant_type=client_credentials",
					)
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(&token{
						ID:          "test123",
						AccessToken: "access-token-123",
						InstanceURL: "foobar.sfdc.cloud",
					})
					is.NoErr(err)
				})
				mux.HandleFunc(userInfoEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`garble`))
					is.NoErr(err)
				})
				u := newAuthServer(t, mux)

				auther, err := newAuthorizer("client123", "secret123", u)
				is.NoErr(err)
				return auther
			},
			wantErr: errors.New(`failed to retrieve user info "client123": failed to decode user info "client123": invalid character 'g' looking for beginning of value`),
		},
		{
			name: "success",
			auth: func(t *testing.T) authorizer {
				t.Helper()
				is := is.New(t)

				mux := http.NewServeMux()
				mux.HandleFunc(loginEndpoint, func(w http.ResponseWriter, r *http.Request) {
					data, _ := io.ReadAll(r.Body)
					is.Equal(
						string(data),
						"client_id=client123&client_secret=secret123&grant_type=client_credentials",
					)
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(&token{
						ID:          "test123",
						AccessToken: "access-token-123",
						InstanceURL: "foobar.sfdc.cloud",
					})
					is.NoErr(err)
				})
				mux.HandleFunc(userInfoEndpoint, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(&user{
						UserID: "foo@bar.com",
						OrgID:  "0xdeadbeef",
					})
					is.NoErr(err)
				})
				u := newAuthServer(t, mux)

				auther, err := newAuthorizer("client123", "secret123", u)
				is.NoErr(err)
				return auther
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			auth := tc.auth(t)

			err := auth.Authorize(context.TODO())
			if tc.wantErr != nil {
				is.True(err != nil)
				is.Equal(err.Error(), tc.wantErr.Error())
			} else {
				is.NoErr(err)

				md, ok := metadata.FromOutgoingContext(auth.Context(context.TODO()))
				is.True(ok)
				is.Equal(md, metadata.Pairs(
					"accesstoken", "access-token-123",
					"instanceurl", "foobar.sfdc.cloud",
					"tenantid", "0xdeadbeef",
				))
			}
		})
	}
}

func Test_Context(t *testing.T) {
	t.Run("returns metadata with auth", func(t *testing.T) {
		is := is.New(t)
		ctx := context.TODO()
		expectCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("foo", "bar"))

		auth := oauth{
			authdata: metadata.Pairs("foo", "bar"),
		}
		is.Equal(auth.Context(ctx), expectCtx)
	})

	t.Run("panics when authdata is missing", func(t *testing.T) {
		is := is.New(t)
		defer func() {
			is.Equal(recover(), "need to authorize first")
		}()

		auth := oauth{}
		auth.Context(context.TODO())
	})
}

func Test_apiErr_Decode(t *testing.T) {
	is := is.New(t)

	r := strings.NewReader("foobar")
	err := (&oauth{}).apiErr(r)
	var expectErr apiError
	is.True(!errors.As(err, &expectErr))
	is.True(strings.HasPrefix(err.Error(), "failed to decode error: invalid character"))
}

func newAuthServer(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(func() { srv.Close() })
	return srv.URL
}
