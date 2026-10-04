// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int {
	return &v
}

func TestS3ProvideSessionInfoMaxRetries(t *testing.T) {
	const path = "s3://team-bucket/team-a/model"

	fromEnv := &S3Credentials{FromEnv: true}

	tt := []struct {
		name               string
		defaultMaxRetries  *int
		overrideMaxRetries *int
		overrideBucketName string
		overrideKeyPrefix  string
		expectedMaxRetries string
	}{
		{
			name:               "default only, matching override does not set maxRetries",
			defaultMaxRetries:  intPtr(3),
			overrideMaxRetries: nil,
			overrideBucketName: "team-bucket",
			overrideKeyPrefix:  "team-a",
			expectedMaxRetries: "3",
		},
		{
			name:               "override only, default omits maxRetries",
			defaultMaxRetries:  nil,
			overrideMaxRetries: intPtr(9),
			overrideBucketName: "team-bucket",
			overrideKeyPrefix:  "team-a",
			expectedMaxRetries: "9",
		},
		{
			name:               "override wins when default and override differ",
			defaultMaxRetries:  intPtr(3),
			overrideMaxRetries: intPtr(9),
			overrideBucketName: "team-bucket",
			overrideKeyPrefix:  "team-a",
			expectedMaxRetries: "9",
		},
		{
			name:               "no maxRetries configured falls back to the documented default",
			defaultMaxRetries:  nil,
			overrideMaxRetries: nil,
			overrideBucketName: "team-bucket",
			overrideKeyPrefix:  "team-a",
			expectedMaxRetries: "5",
		},
		{
			name:               "override for another bucket leaves the default in place",
			defaultMaxRetries:  intPtr(3),
			overrideMaxRetries: intPtr(9),
			overrideBucketName: "other-bucket",
			overrideKeyPrefix:  "team-a",
			expectedMaxRetries: "3",
		},
		{
			name:               "override for another key prefix leaves the default in place",
			defaultMaxRetries:  intPtr(3),
			overrideMaxRetries: intPtr(9),
			overrideBucketName: "team-bucket",
			overrideKeyPrefix:  "team-b",
			expectedMaxRetries: "3",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			providerConfig := S3ProviderConfig{
				Default: &S3ProviderDefault{
					Endpoint:    "s3.amazonaws.com",
					Region:      "us-east-1",
					Credentials: fromEnv,
					MaxRetries:  tc.defaultMaxRetries,
				},
				Overrides: []S3Override{
					{
						BucketName:  tc.overrideBucketName,
						KeyPrefix:   tc.overrideKeyPrefix,
						Credentials: fromEnv,
						MaxRetries:  tc.overrideMaxRetries,
					},
				},
			}

			sessionInfo, err := providerConfig.ProvideSessionInfo(path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedMaxRetries, sessionInfo.Params["maxRetries"])
		})
	}
}

func TestS3ProvideSessionInfoOverrideKeepsRemainingParams(t *testing.T) {
	providerConfig := S3ProviderConfig{
		Default: &S3ProviderDefault{
			Endpoint:    "s3.amazonaws.com",
			Region:      "us-east-1",
			Credentials: &S3Credentials{FromEnv: true},
			MaxRetries:  intPtr(3),
		},
		Overrides: []S3Override{
			{
				BucketName:  "team-bucket",
				KeyPrefix:   "team-a",
				Endpoint:    "minio.team-a:9000",
				Region:      "us-west-2",
				Credentials: &S3Credentials{FromEnv: true},
				MaxRetries:  intPtr(9),
			},
		},
	}

	sessionInfo, err := providerConfig.ProvideSessionInfo("s3://team-bucket/team-a/model")
	require.NoError(t, err)
	assert.Equal(t, objectstore.SessionInfo{
		Provider: "s3",
		Params: map[string]string{
			"endpoint":       "minio.team-a:9000",
			"region":         "us-west-2",
			"disableSSL":     "false",
			"forcePathStyle": "true",
			"maxRetries":     "9",
			"fromEnv":        "true",
		},
	}, sessionInfo)
}
