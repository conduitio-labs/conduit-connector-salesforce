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

package source

import (
	"fmt"
	"strings"
)

const (
	ConfigKeyEnvironment     = "environment"
	ConfigKeyClientID        = "clientId"
	ConfigKeyClientSecret    = "clientSecret"
	ConfigKeyUsername        = "username"
	ConfigKeyPassword        = "password"
	ConfigKeySecurityToken   = "securityToken"
	ConfigKeyPushTopicsNames = "pushTopicsNames"
	ConfigKeyKeyField        = "keyField"
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
}

func ParseConfig(cfgRaw map[string]string) (Config, error) {
	cfg := Config{
		Environment:     cfgRaw[ConfigKeyEnvironment],
		ClientID:        cfgRaw[ConfigKeyClientID],
		ClientSecret:    cfgRaw[ConfigKeyClientSecret],
		Username:        cfgRaw[ConfigKeyUsername],
		Password:        cfgRaw[ConfigKeyPassword],
		SecurityToken:   cfgRaw[ConfigKeySecurityToken],
		PushTopicsNames: make([]string, 0),
		KeyField:        cfgRaw[ConfigKeyKeyField],
	}
	if cfg.Environment == "" {
		return Config{}, requiredConfigErr(ConfigKeyEnvironment)
	}
	if cfg.ClientID == "" {
		return Config{}, requiredConfigErr(ConfigKeyClientID)
	}
	if cfg.ClientSecret == "" {
		return Config{}, requiredConfigErr(ConfigKeyClientSecret)
	}
	if cfg.Username == "" {
		return Config{}, requiredConfigErr(ConfigKeyUsername)
	}
	if cfg.Password == "" {
		return Config{}, requiredConfigErr(ConfigKeyPassword)
	}

	// Push Topics' Names
	registeredTopics := make(map[string]bool)

	for _, topicName := range strings.Split(cfgRaw[ConfigKeyPushTopicsNames], ",") {
		topicNameClear := strings.TrimSpace(topicName)
		if topicNameClear == "" {
			continue
		}

		if _, exists := registeredTopics[topicNameClear]; exists {
			continue
		}

		cfg.PushTopicsNames = append(cfg.PushTopicsNames, topicNameClear)
		registeredTopics[topicNameClear] = true
	}

	if len(cfg.PushTopicsNames) == 0 {
		return Config{}, requiredConfigErr(ConfigKeyPushTopicsNames)
	}

	return cfg, nil
}

func requiredConfigErr(name string) error {
	return fmt.Errorf("%q config value must be set", name)
}
