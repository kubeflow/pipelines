// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderPolicyConformance runs the same test cases the frontend runs
// in provider-policy.test.ts, from a single shared fixture, so the Go and
// TypeScript implementations of the artifact-URI trust decision can't
// silently drift apart. See kubeflow/pipelines#14046.
type conformanceQuery struct {
	FromEnv    string `json:"fromEnv"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
	DisableSSL string `json:"disableSSL"`
}

type conformanceExpect struct {
	Allowed           bool   `json:"allowed"`
	RejectionContains string `json:"rejectionContains"`
	Endpoint          string `json:"endpoint"`
	DisableSSL        string `json:"disableSSL"`
	EffectiveIsNull   bool   `json:"effectiveIsNull"`
}

type conformanceCase struct {
	Name        string            `json:"name"`
	Provider    string            `json:"provider"`
	AdminConfig json.RawMessage   `json:"adminConfig"`
	Bucket      string            `json:"bucket"`
	KeyPrefix   string            `json:"keyPrefix"`
	Query       *conformanceQuery `json:"query"`
	Expect      conformanceExpect `json:"expect"`
}

func buildConformanceURI(provider, bucket, keyPrefix string, query *conformanceQuery) string {
	uri := provider + "://" + bucket + "/" + keyPrefix
	if query == nil {
		return uri
	}
	values := url.Values{}
	if query.FromEnv != "" {
		values.Set("fromEnv", query.FromEnv)
	}
	if query.Endpoint != "" {
		values.Set("endpoint", query.Endpoint)
	}
	if query.Region != "" {
		values.Set("region", query.Region)
	}
	if query.DisableSSL != "" {
		values.Set("disableSSL", query.DisableSSL)
	}
	if len(values) > 0 {
		uri += "?" + values.Encode()
	}
	return uri
}

func isAdminConfigPresent(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func TestProviderPolicyConformance(t *testing.T) {
	data, err := os.ReadFile("../../../../test/conformance/artifact-provider-policy/cases.json")
	require.NoError(t, err)

	var cases []conformanceCase
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			uri := buildConformanceURI(tc.Provider, tc.Bucket, tc.KeyPrefix, tc.Query)

			var provideErr error
			var params map[string]string

			switch tc.Provider {
			case "s3":
				var cfg S3ProviderConfig
				if isAdminConfigPresent(tc.AdminConfig) {
					require.NoError(t, json.Unmarshal(tc.AdminConfig, &cfg))
				}
				info, e := cfg.ProvideSessionInfo(uri)
				provideErr = e
				params = info.Params
			case "minio":
				var cfg MinioProviderConfig
				if isAdminConfigPresent(tc.AdminConfig) {
					require.NoError(t, json.Unmarshal(tc.AdminConfig, &cfg))
				}
				info, e := cfg.ProvideSessionInfo(uri)
				provideErr = e
				params = info.Params
			default:
				t.Fatalf("unknown provider %q", tc.Provider)
			}

			if !tc.Expect.Allowed {
				require.Error(t, provideErr)
				if tc.Expect.RejectionContains != "" {
					assert.Contains(t, provideErr.Error(), tc.Expect.RejectionContains)
				}
				return
			}
			require.NoError(t, provideErr)

			if tc.Expect.EffectiveIsNull {
				// Go has no separate "null" concept: bare fromEnv with no
				// structured settings is the equivalent "nothing admin-driven
				// to apply" case.
				assert.Equal(t, "true", params["fromEnv"])
				return
			}

			// On the unmanaged-but-allowed path, Go's SessionInfo carries only
			// fromEnv -- the endpoint is resolved later, directly from the raw
			// URI query, by objectstore.OpenBucket's gocloud URL opener, not
			// from SessionInfo. Only compare endpoint/disableSSL when Go
			// actually populated them (the admin-config-driven cases).
			if tc.Expect.Endpoint != "" {
				if actual, ok := params["endpoint"]; ok {
					assert.Equal(t, tc.Expect.Endpoint, actual)
				}
			}
			if tc.Expect.DisableSSL != "" {
				if actual, ok := params["disableSSL"]; ok {
					assert.Equal(t, tc.Expect.DisableSSL, actual)
				}
			}
		})
	}
}
