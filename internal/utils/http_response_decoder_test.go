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

package utils

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jaswdr/faker"
	"github.com/stretchr/testify/require"
)

func TestDecodeHTTPResponse(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("fails when content encoding is not supported", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Content-Encoding": {"some-unsupported-encoding"},
			},
		}

		payload, err := DecodeHTTPResponse(response)

		require.Nil(t, payload)
		require.EqualError(t, err, `unsupported encoding "some-unsupported-encoding"`)
	})

	t.Run("fails to read data when reader could not be initialized", func(t *testing.T) {
		errorText := fakerInstance.Lorem().Sentence(6)

		response := &http.Response{
			Body: io.NopCloser(&invalidReader{text: errorText}),
			Header: http.Header{
				"Content-Encoding": {""},
			},
		}

		payload, err := DecodeHTTPResponse(response)

		require.Nil(t, payload)
		require.EqualError(t, err, fmt.Sprintf("could not read response data: %s", errorText))
	})

	t.Run("fails to decompress data when reader could not be initialized", func(t *testing.T) {
		errorText := fakerInstance.Lorem().Sentence(6)

		response := &http.Response{
			Body: io.NopCloser(&invalidReader{text: errorText}),
			Header: http.Header{
				"Content-Encoding": {"gzip"},
			},
		}

		payload, err := DecodeHTTPResponse(response)

		require.Nil(t, payload)
		require.EqualError(t, err, fmt.Sprintf("could not decompress response data: %s", errorText))
	})

	t.Run("returns raw body when no encoding is specified", func(t *testing.T) {
		bodyValue := fakerInstance.Lorem().Bytes(16)

		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader(bodyValue)),
			Header: http.Header{
				"Content-Encoding": {""},
			},
		}

		payload, err := DecodeHTTPResponse(response)

		require.NoError(t, err)
		require.Equal(t, bodyValue, payload)
	})

	t.Run("decompresses and returns body when gzip encoding is specified", func(t *testing.T) {
		var buff bytes.Buffer
		bodyValue := fakerInstance.Lorem().Bytes(16)

		gz, err := gzip.NewWriterLevel(&buff, gzip.BestSpeed)
		require.NoError(t, err)

		_, err = gz.Write(bodyValue)
		require.NoError(t, err)

		require.NoError(t, gz.Close())

		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader(buff.Bytes())),
			Header: http.Header{
				"Content-Encoding": {"gzip"},
			},
		}

		payload, err := DecodeHTTPResponse(response)

		require.NoError(t, err)
		require.Equal(t, bodyValue, payload)
	})
}

type invalidReader struct {
	text string
}

func (r *invalidReader) Read(_ []byte) (int, error) {
	return 0, errors.New(r.text)
}
