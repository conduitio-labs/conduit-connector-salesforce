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
