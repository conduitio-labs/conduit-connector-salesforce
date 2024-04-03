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
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"github.com/simpleforce/simpleforce"
)

const (
	defaultBatchDelay = time.Second * 5
	defaultBatchSize  = 1000
	keepAliveParam    = "CLIENT_SESSION_KEEP_ALIVE"
)

type Destination struct {
	sdk.UnimplementedDestination

	Config Config
	client *simpleforce.Client
}

type Config struct {
	Environment     string
	ClientID        string
	ClientSecret    string
	Username        string
	Password        string
	SecurityToken   string
	PushTopicsNames []string
	KeyField        string
	ObjectName string
	InstanceURL string
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
	// TODO: implement
	return nil
}

func (d *Destination) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Debug().Msg("Configuring Destination Connector.")

	err := sdk.Util.ParseConfig(cfg, &d.Config)
	if err != nil {
		return errors.Errorf("failed to parse destination config: %w", err)
	}

	return nil
}

// Open sets up the salesforce client by authenticating
func (d *Destination) Open(ctx context.Context) error {
	client := simpleforce.NewClient(d.Config.InstanceURL, d.Config.ClientID, simpleforce.DefaultAPIVersion)
	if client == nil {
		return errors.New("Unable to create Salesforce client")
	}

	err := client.LoginPassword(d.Config.Username, d.Config.Password, d.Config.SecurityToken)
	if err != nil {
		return errors.New("Unable to login to Salesforce")
	}

	d.client = client
	return nil
}

func (d *Destination) Write(ctx context.Context, records []sdk.Record) (int, error) {
	for _, r := range records {
		switch r.Operation {
		case sdk.OperationCreate, sdk.OperationUpdate:
			// upsert

			// detect if data is structured, or if it's JSON raw data.
			var data map[string]interface{}
			data, ok := r.Payload.After.(sdk.StructuredData)
			if !ok {
				rawData, ok := r.Payload.After.(sdk.RawData)
				if !ok {
					return 0, errors.New("cannot extract rawData from payload.after")
				}
				
				if err := json.Unmarshal(rawData.Bytes(), &data); err != nil {
					return 0, errors.New("cannot unmarshal JSON from payload.after")
				}
			}
			
			upsertObj := d.client.SObject(d.Config.ObjectName)
			for k, v := range data {
				upsertObj.Set(k, v)
			}

			upsertObj = upsertObj.Upsert()

			sdk.Logger(ctx).Debug().Msgf("upsert: %+w", upsertObj)
		case sdk.OperationDelete:
			// delete
			obj := d.client.SObject(d.Config.ObjectName)
			
			var keyData map[string]interface{}
			if err := json.Unmarshal(r.Key.Bytes(), &keyData); err != nil {
				return 0, errors.New("cannot unmarshal JSON from payload.after")
			}

			key, ok := keyData[d.Config.KeyField]
			if !ok {
				return 0, errors.New("could not find key field %s", d.Config.KeyField)
			}

			if err := obj.Delete(key.(string)); err != nil {
				return 0, errors.Errorf("failed to delete onj: %w", err)
			}
		}
	}

	return len(records), nil
}

func (d *Destination) Teardown(ctx context.Context) error {
	return errors.New("unimplemented")
}
