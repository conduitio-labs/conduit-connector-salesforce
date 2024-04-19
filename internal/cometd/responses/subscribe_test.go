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

package responses

import (
	"fmt"
	"testing"

	"github.com/jaswdr/faker"
	"github.com/stretchr/testify/require"
)

func TestSubscribeResponse_GetSubscriptions(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("panics when unsupported type is provided", func(t *testing.T) {
		subscription := fakerInstance.Int()

		response := SubscribeResponse{
			Subscription: subscription,
		}

		require.PanicsWithError(t, fmt.Sprintf("unexpected subscriptions data: %d", subscription), func() {
			response.GetSubscriptions()
		})
	})

	t.Run("slice with one element is returned when Subscription is string", func(t *testing.T) {
		subscription := fakerInstance.Lorem().Sentence(3)

		response := SubscribeResponse{
			Subscription: subscription,
		}

		subscriptions := response.GetSubscriptions()

		require.Len(t, subscriptions, 1)
		require.Equal(t, subscriptions[0], subscription)
	})

	t.Run("slice with all elements is returned when Subscription is slice", func(t *testing.T) {
		subscription := []string{
			fakerInstance.Lorem().Sentence(3),
			fakerInstance.Lorem().Sentence(4),
			fakerInstance.Lorem().Sentence(5),
		}

		response := SubscribeResponse{
			Subscription: subscription,
		}

		subscriptions := response.GetSubscriptions()

		require.Len(t, subscriptions, 3)
		require.Equal(t, subscriptions[0], subscription[0])
		require.Equal(t, subscriptions[1], subscription[1])
		require.Equal(t, subscriptions[2], subscription[2])
	})
}
