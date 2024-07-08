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

package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"github.com/meroxa/simpleforce"
)

const (
	defaultBatchDelay = time.Second * 5
	defaultBatchSize  = 1000
)

type Destination struct {
	sdk.UnimplementedDestination

	Config Config
	client *simpleforce.Client
}

// TODO: move to config file.
const (
	ConfigKeyEnvironment   = "environment"
	ConfigKeyClientID      = "clientId"
	ConfigKeyClientSecret  = "clientSecret"
	ConfigKeyUsername      = "username"
	ConfigKeyPassword      = "password"
	ConfigKeySecurityToken = "securityToken"
	ConfigKeyKeyField      = "keyField"
	ConfigKeyObjectName    = "objectName"
	ConfigKeyInstanceURL   = "instanceURL"
	ConfigKeyHardDelete    = "hardDelete"
)

type Config struct {
	Environment     string
	ClientID        string
	ClientSecret    string
	Username        string
	Password        string
	SecurityToken   string
	PushTopicsNames []string
	KeyField        string
	ObjectName      string
	InstanceURL     string
	HardDelete      bool
}

// NewDestination creates the Destination and wraps it in the default middleware.
func NewDestination() sdk.Destination {
	// This is needed to override the default batch size and delay defaults for this destination connector.
	middlewares := sdk.DefaultDestinationMiddleware()
	for i, m := range middlewares {
		switch dest := m.(type) {
		case sdk.DestinationWithBatch:
			dest.DefaultBatchDelay = defaultBatchDelay
			dest.DefaultBatchSize = defaultBatchSize
			middlewares[i] = dest
		default:
		}
	}

	return sdk.DestinationWithMiddleware(&Destination{}, middlewares...)
}

func (d *Destination) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		ConfigKeyEnvironment: {
			Default:     "",
			Required:    true,
			Description: "Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com -> `MyDomainName`) or `sandbox` for test environment.",
		},
		ConfigKeyClientID: {
			Default:     "",
			Required:    true,
			Description: "OAuth Client ID (Consumer Key).",
		},
		ConfigKeyClientSecret: {
			Default:     "",
			Required:    true,
			Description: "OAuth Client Secret (Consumer Secret).",
		},
		ConfigKeyUsername: {
			Default:     "",
			Required:    true,
			Description: "Username.",
		},
		ConfigKeyPassword: {
			Default:     "",
			Required:    true,
			Description: "Password.",
		},
		ConfigKeySecurityToken: {
			Default:     "",
			Required:    false,
			Description: "Security token as described here: https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5.",
		},
		ConfigKeyObjectName: {
			Default:     "",
			Required:    true,
			Description: "The name of the salesforce object used. Example: Order__c",
		},
		ConfigKeyInstanceURL: {
			Default:     "",
			Required:    true,
			Description: "URL for the salesforce instance. Example: Order__c",
		},
		ConfigKeyKeyField: {
			Default:     "Id",
			Required:    false,
			Description: "The name of the field that should be used as a Payload's Key. Empty value will set it to `nil`.",
		},
		ConfigKeyHardDelete: {
			Default:     "false",
			Required:    false,
			Description: "`true` will turn on hard-deletes, removing objects via the DELETE method on Salesforce API. `false` will perform soft-deletes with an updated_at timestamp.",
		},
	}
}

func (d *Destination) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Destination Connector.")

	err := sdk.Util.ParseConfig(cfg, &d.Config)
	if err != nil {
		return errors.Errorf("failed to parse destination config: %w", err)
	}

	return nil
}

// Open sets up the salesforce client by authenticating.
func (d *Destination) Open(ctx context.Context) error {
	client := simpleforce.NewClient(d.Config.InstanceURL, d.Config.ClientID, simpleforce.DefaultAPIVersion)
	if client == nil {
		return errors.New("Unable to create Salesforce client")
	}

	d.client = client

	if err := d.login(ctx); err != nil {
		return errors.Errorf("Unable to login to Salesforce: %w", err)
	}
	return nil
}

func (d *Destination) Write(ctx context.Context, records []sdk.Record) (int, error) {
	for _, r := range records {
		var keyData, payloadData map[string]interface{}

		if err := unmarshal(r.Key, &keyData); err != nil {
			return 0, errors.Errorf("cannot extract data from key: %w", err)
		}

		sfObj := d.client.SObject(d.Config.ObjectName)

		// set ExternalIDField & ExternalID
		keyStr := fmt.Sprint(keyData[d.Config.KeyField])

		idField := strings.ToUpper(d.Config.KeyField)
		if !strings.HasSuffix(idField, "__c") {
			idField = fmt.Sprintf("%s__c", idField)
		}

		switch r.Operation {
		case sdk.OperationUpdate, sdk.OperationDelete:
			q := fmt.Sprintf(
				"SELECT FIELDS(ALL) FROM %s WHERE %s = '%s' LIMIT 1",
				d.Config.ObjectName,
				idField,
				keyStr,
			)
			result, err := d.client.Query(q)
			if err != nil {
				return 0, errors.Errorf("failed to get object from salesforce: %w", err)
			}

			if len(result.Records) != 1 {
				return 0, errors.Errorf("unexpected number of matched records from salesforce: %w", err)
			}

			// set salesforce ID, needed for updates & deletes.
			sfObj = sfObj.Set("Id", result.Records[0].ID())
		}

		sfObj = sfObj.Set("ExternalIDField", idField)
		sfObj = sfObj.Set(idField, keyStr)

		switch r.Operation {
		case sdk.OperationSnapshot, sdk.OperationCreate, sdk.OperationUpdate:
			if err := unmarshal(r.Payload.After, &payloadData); err != nil {
				return 0, errors.Errorf("cannot extract data from payload.After: %w", err)
			}

			// Set data fields
			for k, v := range payloadData {
				// we already set the key above
				if k == d.Config.KeyField {
					continue
				}
				key := k
				if !strings.HasSuffix(key, "__c") {
					key = fmt.Sprintf("%s__c", key)
				}
				sfObj = sfObj.Set(key, v)
			}
		}

		switch r.Operation {
		case sdk.OperationSnapshot, sdk.OperationCreate, sdk.OperationUpdate:
			sfObj = sfObj.Set("updated_at__c", time.Now().UTC().Format("2006/01/02 03:04:05"))
			if err := d.handleSobjectErr(ctx, sfObj.Upsert); err != nil {
				return 0, err
			}
		case sdk.OperationDelete:
			if d.Config.HardDelete {
				if err := sfObj.Delete(); err != nil {
					return 0, errors.Errorf("error deleting record: %w", err)
				}
			} else {
				sfObj = sfObj.Set("deleted_at__c", time.Now().UTC().Format("2006/01/02 03:04:05"))
				if err := d.handleSobjectErr(ctx, sfObj.Upsert); err != nil {
					return 0, err
				}
			}
		}
	}

	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	// TODO: implement
	return nil
}

func unmarshal(d sdk.Data, m *map[string]interface{}) error {
	// detect if data is already structured, or if it's JSON raw data.
	data, ok := d.(sdk.StructuredData)
	if ok {
		*m = data
		return nil
	}

	return json.Unmarshal(d.Bytes(), m)
}

func (d *Destination) login(ctx context.Context) error {
	if err := d.client.LoginPassword(d.Config.Username, d.Config.Password, d.Config.SecurityToken); err != nil {
		return err
	}
	sdk.Logger(ctx).Info().Msg("Logged into Salesforce succcessfully.")
	return nil
}

func (d *Destination) handleSobjectErr(ctx context.Context, f func() error) error {
	if err := f(); err != nil {
		if strings.Contains(err.Error(), "INVALID_SESSION_ID") {
			sdk.Logger(ctx).Info().Msg("error: INVALID_SESSION_ID, attempting to log in again.")
			// login again
			if loginErr := d.login(ctx); loginErr != nil {
				return loginErr
			}

			// retry again, and return error if it fails back-to-back
			return f()
		}
		sdk.Logger(ctx).Err(err)
		return err
	}
	return nil
}
