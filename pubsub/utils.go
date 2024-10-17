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
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	unionFields := make(map[string]struct{})
	fields := schema["fields"].([]interface{})
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

// checks connection error
func connErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "is unavailable")
}

func invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}

func validateAndPreparePayload(dataMap map[string]interface{}, avroSchema string) (map[string]interface{}, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(avroSchema), &schema); err != nil {
		return nil, err
	}

	avroRecord := make(map[string]interface{})
	fields := schema["fields"].([]interface{})

	for _, field := range fields {
		fieldMap := field.(map[string]interface{})
		fieldName := fieldMap["name"].(string)
		fieldType := fieldMap["type"]
		value, exists := dataMap[fieldName]
		if !exists {
			avroRecord[fieldName] = nil
			continue
		}
		if isUnionType(fieldType) {
			if value == nil {
				avroRecord[fieldName] = nil
			} else {
				avroRecord[fieldName] = map[string]interface{}{
					"string": value,
				}
			}
		} else {
			switch fieldType.(type) {
			case string:
				avroRecord[fieldName] = value
			case []interface{}:
				avroRecord[fieldName] = value
			default:
				avroRecord[fieldName] = value
			}
		}
	}

	return avroRecord, nil
}

func isUnionType(fieldType interface{}) bool {
	if types, ok := fieldType.([]interface{}); ok {
		return len(types) > 1
	}
	return false
}
