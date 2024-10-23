package pubsub

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"strings"

	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/pkg/errors"
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
		return nil, errors.Errorf("failed to parse schema: %s", err)
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

// checks connection error.
func connErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "is unavailable")
}

func invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}

func validateAndPreparePayload(dataMap opencdc.StructuredData, avroSchema string) (map[string]interface{}, error) {
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
		switch t := fieldType.(type) {
		case []interface{}:
			for _, unionType := range t {
				if unionType.(string) != "null" {
					avroRecord[fieldName] = map[string]interface{}{
						unionType.(string): value,
					}
				}
			}
		default:
			avroRecord[fieldName] = value
		}
	}

	return avroRecord, nil
}

func extractPayload(op opencdc.Operation, payload opencdc.Change) (opencdc.StructuredData, error) {
	var sdkData opencdc.Data
	if op == opencdc.OperationDelete {
		sdkData = payload.Before
	} else {
		sdkData = payload.After
	}

	dataStruct, okStruct := sdkData.(opencdc.StructuredData)
	dataRaw, okRaw := sdkData.(opencdc.RawData)

	if okStruct {
		return dataStruct, nil
	} else if okRaw {
		data := make(opencdc.StructuredData)
		if err := json.Unmarshal(dataRaw, &payload); err != nil {
			return nil, errors.Errorf("cannot unmarshal raw data payload into structured (%T): %s", sdkData, err)
		}

		return data, nil
	}

	return nil, errors.Errorf("cannot find data in payload (%T)", sdkData)
}
