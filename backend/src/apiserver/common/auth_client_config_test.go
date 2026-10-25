// Copyright 2026 The Kubeflow Authors
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

package common

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationClientRateLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		values  map[string]interface{}
		env     map[string]string
		config  string
		qps     float64
		burst   int
		invalid string
	}{
		{name: "bounded defaults", qps: 40, burst: 80},
		{name: "legacy overrides", values: map[string]interface{}{"ClientQPS": 7.5, "ClientBurst": 15}, qps: 7.5, burst: 15},
		{name: "auth overrides", values: map[string]interface{}{"ClientQPS": 5, "ClientBurst": 10, "AuthClientQPS": 30, "AuthClientBurst": 60}, qps: 30, burst: 60},
		{name: "qps override only", values: map[string]interface{}{"ClientQPS": 5, "ClientBurst": 10, "AuthClientQPS": 30}, qps: 30, burst: 10},
		{name: "burst override only", values: map[string]interface{}{"ClientQPS": 5, "ClientBurst": 10, "AuthClientBurst": 60}, qps: 5, burst: 60},
		{name: "legacy qps only", values: map[string]interface{}{"ClientQPS": 7}, qps: 7, burst: 80},
		{name: "auth env", env: map[string]string{"AUTHCLIENTQPS": "12.5", "AUTHCLIENTBURST": "25"}, qps: 12.5, burst: 25},
		{name: "legacy env", env: map[string]string{"CLIENTQPS": "9", "CLIENTBURST": "18"}, qps: 9, burst: 18},
		{name: "config file", config: `{"AuthClientQPS":11,"AuthClientBurst":22}`, qps: 11, burst: 22},
		{name: "env over file", config: `{"AuthClientQPS":11,"AuthClientBurst":22}`, env: map[string]string{"AUTHCLIENTQPS": "13", "AUTHCLIENTBURST": "26"}, qps: 13, burst: 26},
		{name: "auth file over legacy env", config: `{"AuthClientQPS":11,"AuthClientBurst":22}`, env: map[string]string{"CLIENTQPS": "9", "CLIENTBURST": "18"}, qps: 11, burst: 22},
		{name: "empty auth qps env", env: map[string]string{"AUTHCLIENTQPS": ""}, invalid: "AuthClientQPS"},
		{name: "empty auth burst env", env: map[string]string{"AUTHCLIENTBURST": ""}, invalid: "AuthClientBurst"},
		{name: "empty legacy qps env", env: map[string]string{"CLIENTQPS": ""}, invalid: "ClientQPS"},
		{name: "empty legacy burst env", env: map[string]string{"CLIENTBURST": ""}, invalid: "ClientBurst"},
		{name: "independent legacy invalid", values: map[string]interface{}{"ClientQPS": -1, "ClientBurst": 0, "AuthClientQPS": 20, "AuthClientBurst": 40}, qps: 20, burst: 40},
		{name: "zero auth qps", values: map[string]interface{}{"AuthClientQPS": 0}, invalid: "AuthClientQPS"},
		{name: "negative auth qps", values: map[string]interface{}{"AuthClientQPS": -1}, invalid: "AuthClientQPS"},
		{name: "nan", values: map[string]interface{}{"AuthClientQPS": math.NaN()}, invalid: "AuthClientQPS"},
		{name: "infinite", values: map[string]interface{}{"AuthClientQPS": math.Inf(1)}, invalid: "AuthClientQPS"},
		{name: "float32 overflow", values: map[string]interface{}{"AuthClientQPS": "1e100"}, invalid: "AuthClientQPS"},
		{name: "float32 underflow", values: map[string]interface{}{"AuthClientQPS": "1e-100"}, invalid: "AuthClientQPS"},
		{name: "malformed qps", values: map[string]interface{}{"AuthClientQPS": "fast"}, invalid: "AuthClientQPS"},
		{name: "empty configured qps", values: map[string]interface{}{"AuthClientQPS": ""}, invalid: "AuthClientQPS"},
		{name: "zero burst", values: map[string]interface{}{"AuthClientBurst": 0}, invalid: "AuthClientBurst"},
		{name: "negative burst", values: map[string]interface{}{"AuthClientBurst": -1}, invalid: "AuthClientBurst"},
		{name: "fractional burst", values: map[string]interface{}{"AuthClientBurst": 1.5}, invalid: "AuthClientBurst"},
		{name: "burst overflow", values: map[string]interface{}{"AuthClientBurst": "999999999999999999999"}, invalid: "AuthClientBurst"},
		{name: "malformed burst", values: map[string]interface{}{"AuthClientBurst": "many"}, invalid: "AuthClientBurst"},
		{name: "invalid legacy qps", values: map[string]interface{}{"ClientQPS": -1}, invalid: "ClientQPS"},
		{name: "invalid legacy burst", values: map[string]interface{}{"ClientBurst": 0}, invalid: "ClientBurst"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			for _, key := range []string{"AUTHCLIENTQPS", "AUTHCLIENTBURST", "CLIENTQPS", "CLIENTBURST"} {
				t.Setenv(key, "")
				require.NoError(t, os.Unsetenv(key))
			}
			viper.AutomaticEnv()
			viper.AllowEmptyEnv(true)
			if tc.config != "" {
				viper.SetConfigType("json")
				require.NoError(t, viper.ReadConfig(strings.NewReader(tc.config)))
			}
			for key, value := range tc.values {
				viper.Set(key, value)
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			qps, burst, err := GetAuthorizationClientRateLimit()
			if tc.invalid != "" {
				require.ErrorContains(t, err, tc.invalid)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.qps, qps)
			require.Equal(t, tc.burst, burst)
			// Resolving review-specific capacity must not mutate general client settings.
			if tc.env["CLIENTQPS"] == "" {
				require.Equal(t, tc.values["ClientQPS"], viper.Get("ClientQPS"))
			}
		})
	}
}
