// Copyright © 2022 Meroxa, Inc.
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

package pubsub

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"
)

// Fetches system certs and returns them if possible. If unable to fetch system certs then an empty cert pool is returned instead.
func getCerts() *x509.CertPool {
	if certs, err := x509.SystemCertPool(); err == nil {
		return certs
	}

	return x509.NewCertPool()
}

// parseUnionFields parses the schema JSON to identify avro union fields.
func parseUnionFields(_ context.Context, schemaJSON string) (map[string]struct{}, error) {
	var (
		schema map[string]interface{}
		fields []interface{}
		ok     bool
	)

	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	unionFields := make(map[string]struct{})
	if fields, ok = schema["fields"].([]interface{}); !ok {
		return nil, fmt.Errorf("failed to parse fields from topic schema")
	}
	for _, field := range fields {
		f := field.(map[string]interface{})
		fieldType := f["type"]
		if types, ok := fieldType.([]interface{}); ok && len(types) > 1 {
			unionFields[f["name"].(string)] = struct{}{}
		}
	}
	return unionFields, nil
}

// flattenUnionFields flattens union fields decoded from Avro.
func flattenUnionFields(_ context.Context, data map[string]interface{}, unionFields map[string]struct{}) map[string]interface{} {
	flatData := make(map[string]interface{})
	for key, value := range data {
		if _, ok := unionFields[key]; ok { // Check if this field is a union
			if valueMap, ok := value.(map[string]interface{}); ok && len(valueMap) == 1 {
				for _, actualValue := range valueMap {
					flatData[key] = actualValue
					break
				}
			} else {
				flatData[key] = value
			}
		} else {
			flatData[key] = value
		}
	}

	return flatData
}

// checks connection error.
func connErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "is unavailable")
}

func invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}
