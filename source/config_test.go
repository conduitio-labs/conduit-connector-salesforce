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
	"testing"

	"github.com/jaswdr/faker"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("fails when Environment is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			"nonExistentKey": "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyEnvironment))
	})

	t.Run("fails when Client ID is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			ConfigKeyEnvironment: fakerInstance.Lorem().Word(),
			"nonExistentKey":     "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyClientID))
	})

	t.Run("fails when Client Secret is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			ConfigKeyEnvironment: fakerInstance.Lorem().Word(),
			ConfigKeyClientID:    fakerInstance.RandomStringWithLength(32),
			"nonExistentKey":     "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyClientSecret))
	})

	t.Run("fails when Username is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			ConfigKeyEnvironment:  fakerInstance.Lorem().Word(),
			ConfigKeyClientID:     fakerInstance.RandomStringWithLength(32),
			ConfigKeyClientSecret: fakerInstance.RandomStringWithLength(32),
			"nonExistentKey":      "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyUsername))
	})

	t.Run("fails when Password is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			ConfigKeyEnvironment:  fakerInstance.Lorem().Word(),
			ConfigKeyClientID:     fakerInstance.RandomStringWithLength(32),
			ConfigKeyClientSecret: fakerInstance.RandomStringWithLength(32),
			ConfigKeyUsername:     fakerInstance.Lorem().Sentence(6),
			"nonExistentKey":      "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyPassword))
	})

	t.Run("fails when Push Topic Name is empty", func(t *testing.T) {
		_, err := ParseConfig(map[string]string{
			ConfigKeyEnvironment:  fakerInstance.Lorem().Word(),
			ConfigKeyClientID:     fakerInstance.RandomStringWithLength(32),
			ConfigKeyClientSecret: fakerInstance.RandomStringWithLength(32),
			ConfigKeyUsername:     fakerInstance.Lorem().Sentence(6),
			ConfigKeyPassword:     fakerInstance.Lorem().Sentence(6),
			"nonExistentKey":      "value",
		})

		require.EqualError(t, err, fmt.Sprintf("%q config value must be set", ConfigKeyPushTopicName))
	})

	t.Run("returns config when all required config values were provided", func(t *testing.T) {
		cfgRaw := map[string]string{
			ConfigKeyEnvironment:   fakerInstance.Lorem().Word(),
			ConfigKeyClientID:      fakerInstance.RandomStringWithLength(32),
			ConfigKeyClientSecret:  fakerInstance.RandomStringWithLength(32),
			ConfigKeyUsername:      fakerInstance.Lorem().Sentence(6),
			ConfigKeyPassword:      fakerInstance.Lorem().Sentence(6),
			ConfigKeyPushTopicName: fakerInstance.Lorem().Word(),
			"nonExistentKey":       "value",
		}

		config, err := ParseConfig(cfgRaw)

		require.NoError(t, err)
		require.Equal(t, cfgRaw[ConfigKeyEnvironment], config.Environment)
		require.Equal(t, cfgRaw[ConfigKeyClientID], config.ClientID)
		require.Equal(t, cfgRaw[ConfigKeyClientSecret], config.ClientSecret)
		require.Equal(t, cfgRaw[ConfigKeyUsername], config.Username)
		require.Equal(t, cfgRaw[ConfigKeyPassword], config.Password)
		require.Equal(t, cfgRaw[ConfigKeyPushTopicName], config.PushTopicName)
		require.Empty(t, "", config.SecurityToken)
		require.Empty(t, "", config.KeyField)
	})

	t.Run("returns config when all config values were provided", func(t *testing.T) {
		cfgRaw := map[string]string{
			ConfigKeyEnvironment:   fakerInstance.Lorem().Word(),
			ConfigKeyClientID:      fakerInstance.RandomStringWithLength(32),
			ConfigKeyClientSecret:  fakerInstance.RandomStringWithLength(32),
			ConfigKeyUsername:      fakerInstance.Lorem().Sentence(6),
			ConfigKeyPassword:      fakerInstance.Lorem().Sentence(6),
			ConfigKeyPushTopicName: fakerInstance.Lorem().Word(),
			ConfigKeySecurityToken: fakerInstance.RandomStringWithLength(32),
			ConfigKeyKeyField:      fakerInstance.Lorem().Word(),
			"nonExistentKey":       "value",
		}

		config, err := ParseConfig(cfgRaw)

		require.NoError(t, err)
		require.Equal(t, cfgRaw[ConfigKeyEnvironment], config.Environment)
		require.Equal(t, cfgRaw[ConfigKeyClientID], config.ClientID)
		require.Equal(t, cfgRaw[ConfigKeyClientSecret], config.ClientSecret)
		require.Equal(t, cfgRaw[ConfigKeyUsername], config.Username)
		require.Equal(t, cfgRaw[ConfigKeyPassword], config.Password)
		require.Equal(t, cfgRaw[ConfigKeyPushTopicName], config.PushTopicName)
		require.Equal(t, cfgRaw[ConfigKeySecurityToken], config.SecurityToken)
		require.Equal(t, cfgRaw[ConfigKeyKeyField], config.KeyField)
	})
}
