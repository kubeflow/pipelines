// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driverapi_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeArgsUnmarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        driverapi.RuntimeArgs
		wantErr     bool
	}{
		{"null", `null`, nil, false},
		{"empty", `""`, nil, false},
		{"settings", `"{\"setting\":\"value\"}"`, driverapi.RuntimeArgs{"setting": "value"}, false},
		{"object instead of string", `{}`, nil, true},
		{"invalid JSON in string", `"{"`, nil, true},
		{"nonstring value", `"{\"setting\":false}"`, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var args driverapi.RuntimeArgs
			err := json.Unmarshal([]byte(tc.input), &args)
			if tc.wantErr {
				require.Error(t, err)
				assert.NotNil(t, errors.Unwrap(err))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, args)
			}
		})
	}
}
