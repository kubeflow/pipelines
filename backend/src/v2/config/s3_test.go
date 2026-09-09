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

func boolPtr(v bool) *bool {
	return &v
}

// TestS3ProvideSessionInfoOverrideWinsOverQuery is the regression case for
// the override-bypass bug: a query string on the artifact URI must not be
// able to override an admin-configured Override/Default for that bucket.
func TestS3ProvideSessionInfoOverrideWinsOverQuery(t *testing.T) {
	providerConfig := S3ProviderConfig{
		Default: &S3ProviderDefault{
			Endpoint:    "s3.amazonaws.com",
			Region:      "us-east-1",
			Credentials: &S3Credentials{FromEnv: true},
		},
		Overrides: []S3Override{
			{
				BucketName:  "team-bucket",
				KeyPrefix:   "team-a",
				Endpoint:    "minio.team-a:9000",
				Region:      "us-west-2",
				Credentials: &S3Credentials{FromEnv: true},
			},
		},
	}

	// A tenant-supplied endpoint/disableSSL/region on the URI must be
	// ignored: the matching override's settings must win.
	sessionInfo, err := providerConfig.ProvideSessionInfo(
		"s3://team-bucket/team-a/model?endpoint=attacker.example&region=custom&disableSSL=true")
	require.NoError(t, err)
	assert.Equal(t, "minio.team-a:9000", sessionInfo.Params["endpoint"])
	assert.Equal(t, "us-west-2", sessionInfo.Params["region"])
	assert.NotEqual(t, "attacker.example", sessionInfo.Params["endpoint"])
}

// TestS3ProvideSessionInfoDefaultWinsOverQueryWithoutMatchingOverride covers
// a bucket with no matching Override but a provider-wide Default: the
// Default is still authoritative admin configuration and must not be
// bypassed by a query string either.
func TestS3ProvideSessionInfoDefaultWinsOverQueryWithoutMatchingOverride(t *testing.T) {
	providerConfig := S3ProviderConfig{
		Default: &S3ProviderDefault{
			Endpoint:    "s3.company.example",
			Region:      "us-east-1",
			Credentials: &S3Credentials{FromEnv: true},
		},
	}

	sessionInfo, err := providerConfig.ProvideSessionInfo(
		"s3://some-other-bucket/path/model?endpoint=attacker.example")
	require.NoError(t, err)
	assert.Equal(t, "s3.company.example", sessionInfo.Params["endpoint"])
}

func TestS3ProvideSessionInfoUnmanagedProviderQueries(t *testing.T) {
	const uri = "s3://unconfigured-bucket/path/model?endpoint=example.com"

	t.Run("allowed by default when unset", func(t *testing.T) {
		providerConfig := S3ProviderConfig{}
		sessionInfo, err := providerConfig.ProvideSessionInfo(uri)
		require.NoError(t, err)
		assert.Equal(t, objectstore.SessionInfo{
			Provider: "s3",
			Params:   map[string]string{"fromEnv": "true"},
		}, sessionInfo)
	})

	t.Run("allowed when explicitly enabled", func(t *testing.T) {
		providerConfig := S3ProviderConfig{AllowUnmanagedProviderQueries: boolPtr(true)}
		_, err := providerConfig.ProvideSessionInfo(uri)
		require.NoError(t, err)
	})

	t.Run("rejected when explicitly disabled", func(t *testing.T) {
		providerConfig := S3ProviderConfig{AllowUnmanagedProviderQueries: boolPtr(false)}
		_, err := providerConfig.ProvideSessionInfo(uri)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "allowUnmanagedProviderQueries")
	})

	t.Run("query-free URI is unaffected by the gate even when disabled", func(t *testing.T) {
		providerConfig := S3ProviderConfig{AllowUnmanagedProviderQueries: boolPtr(false)}
		sessionInfo, err := providerConfig.ProvideSessionInfo("s3://unconfigured-bucket/path/model")
		require.NoError(t, err)
		assert.Equal(t, objectstore.SessionInfo{
			Provider: "s3",
			Params:   map[string]string{"fromEnv": "true"},
		}, sessionInfo)
	})
}

func TestS3ProvideSessionInfoUnmanagedQueryGuardrails(t *testing.T) {
	tt := []struct {
		name        string
		uri         string
		wantErr     bool
		errContains string
	}{
		{
			name: "public endpoint is allowed",
			uri:  "s3://unconfigured-bucket/path/model?endpoint=s3.us-west-2.amazonaws.com",
		},
		{
			name:        "loopback IP endpoint is rejected",
			uri:         "s3://unconfigured-bucket/path/model?endpoint=127.0.0.1:9999",
			wantErr:     true,
			errContains: "loopback/link-local/private",
		},
		{
			name:        "link-local IP endpoint is rejected",
			uri:         "s3://unconfigured-bucket/path/model?endpoint=169.254.169.254",
			wantErr:     true,
			errContains: "loopback/link-local/private",
		},
		{
			name:        "RFC1918 private IP endpoint is rejected",
			uri:         "s3://unconfigured-bucket/path/model?endpoint=10.0.0.5:9000",
			wantErr:     true,
			errContains: "loopback/link-local/private",
		},
		{
			name:        "disableSSL is rejected on the unmanaged path",
			uri:         "s3://unconfigured-bucket/path/model?endpoint=s3.us-west-2.amazonaws.com&disableSSL=true",
			wantErr:     true,
			errContains: "disableSSL",
		},
		{
			name: "disableSSL=false is allowed",
			uri:  "s3://unconfigured-bucket/path/model?endpoint=s3.us-west-2.amazonaws.com&disableSSL=false",
		},
		{
			name: "hostname endpoint is allowed without a DNS lookup",
			uri:  "s3://unconfigured-bucket/path/model?endpoint=internal.svc.cluster.local",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			providerConfig := S3ProviderConfig{}
			_, err := providerConfig.ProvideSessionInfo(tc.uri)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// A matching admin Override still wins even when its own endpoint would
// otherwise look "unsafe" -- the guardrails apply only to the unmanaged
// path, never to admin-configured settings, matching the in-cluster
// MinIO/cluster-DNS deployment that's the common case.
func TestS3ProvideSessionInfoGuardrailsDoNotApplyToAdminConfig(t *testing.T) {
	providerConfig := S3ProviderConfig{
		Default: &S3ProviderDefault{
			Endpoint:    "seaweedfs.kubeflow:9000",
			Credentials: &S3Credentials{FromEnv: true},
			DisableSSL:  boolPtr(true),
		},
	}

	sessionInfo, err := providerConfig.ProvideSessionInfo("s3://team-bucket/team-a/model")
	require.NoError(t, err)
	assert.Equal(t, "seaweedfs.kubeflow:9000", sessionInfo.Params["endpoint"])
	assert.Equal(t, "true", sessionInfo.Params["disableSSL"])
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
