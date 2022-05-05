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
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

func DecodeHTTPResponse(response *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	switch encoding := response.Header.Get("Content-Encoding"); encoding {
	case "gzip":
		var err error

		reader, err = gzip.NewReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("could not decompress response data: %w", err)
		}

		defer reader.Close()

	case "":
		reader = response.Body

	default:
		return nil, fmt.Errorf("unsupported encoding %q", encoding)
	}

	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read response data: %w", err)
	}

	return contents, nil
}
