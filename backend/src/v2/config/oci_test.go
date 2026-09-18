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

func boolPtr(v bool) *bool {
	return &v
}

func TestOCIProvideSessionInfo(t *testing.T) {
	const path = "oci://kfp-artifacts@mynamespace/team-a/model"

	defaultSecret := &S3Credentials{
		SecretRef: &S3SecretRef{
			SecretName:   "oci-customer-secret-key",
			AccessKeyKey: "accessKey",
			SecretKeyKey: "secretKey",
		},
	}
	overrideSecret := &S3Credentials{
		SecretRef: &S3SecretRef{
			SecretName:   "oci-team-a-key",
			AccessKeyKey: "accessKey",
			SecretKeyKey: "secretKey",
		},
	}
	fromEnv := &S3Credentials{FromEnv: true}

	tests := []struct {
		name     string
		config   OCIProviderConfig
		path     string
		expected objectstore.SessionInfo
		errorMsg string
	}{
		{
			name:   "no provider config uses environment credentials",
			config: OCIProviderConfig{},
			path:   path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params:   map[string]string{"fromEnv": "true"},
			},
		},
		{
			name: "query string forces environment credentials",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
			},
			path: "oci://kfp-artifacts@mynamespace/team-a/model?region=us-phoenix-1",
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params:   map[string]string{"fromEnv": "true"},
			},
		},
		{
			name: "default with secret credentials",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"forcePathStyle": "true",
					"maxRetries":     "5",
					"fromEnv":        "false",
					"secretName":     "oci-customer-secret-key",
					"accessKeyKey":   "accessKey",
					"secretKeyKey":   "secretKey",
				},
			},
		},
		{
			name: "default with endpoint override and tuning",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{
					Region:         "us-ashburn-1",
					Endpoint:       "https://objectstorage.internal.example:8443",
					ForcePathStyle: boolPtr(false),
					MaxRetries:     intPtr(9),
					Credentials:    fromEnv,
				},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"endpoint":       "https://objectstorage.internal.example:8443",
					"forcePathStyle": "false",
					"maxRetries":     "9",
					"fromEnv":        "true",
				},
			},
		},
		{
			name: "matching override replaces region and credentials",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{
					{BucketName: "kfp-artifacts", KeyPrefix: "team-a", Region: "us-phoenix-1", Credentials: overrideSecret},
				},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-phoenix-1",
					"forcePathStyle": "true",
					"maxRetries":     "5",
					"fromEnv":        "false",
					"secretName":     "oci-team-a-key",
					"accessKeyKey":   "accessKey",
					"secretKeyKey":   "secretKey",
				},
			},
		},
		{
			name: "override pinned to another namespace is skipped",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{
					{BucketName: "kfp-artifacts", Namespace: "othernamespace", KeyPrefix: "team-a", Credentials: overrideSecret},
				},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"forcePathStyle": "true",
					"maxRetries":     "5",
					"fromEnv":        "false",
					"secretName":     "oci-customer-secret-key",
					"accessKeyKey":   "accessKey",
					"secretKeyKey":   "secretKey",
				},
			},
		},
		{
			name: "override pinned to the same namespace matches",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{
					{BucketName: "kfp-artifacts", Namespace: "mynamespace", KeyPrefix: "team-a", Credentials: overrideSecret},
				},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"forcePathStyle": "true",
					"maxRetries":     "5",
					"fromEnv":        "false",
					"secretName":     "oci-team-a-key",
					"accessKeyKey":   "accessKey",
					"secretKeyKey":   "secretKey",
				},
			},
		},
		{
			name: "override with environment credentials drops the secret reference",
			config: OCIProviderConfig{
				Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{
					{
						BucketName:     "kfp-artifacts",
						KeyPrefix:      "team-a",
						Endpoint:       "https://objectstorage.internal.example:8443",
						ForcePathStyle: boolPtr(false),
						MaxRetries:     intPtr(2),
						Credentials:    fromEnv,
					},
				},
			},
			path: path,
			expected: objectstore.SessionInfo{
				Provider: "oci",
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"endpoint":       "https://objectstorage.internal.example:8443",
					"forcePathStyle": "false",
					"maxRetries":     "2",
					"fromEnv":        "true",
				},
			},
		},
		{
			name:     "missing default credentials",
			config:   OCIProviderConfig{Default: &OCIProviderDefault{Region: "us-ashburn-1"}},
			path:     path,
			errorMsg: "missing default credentials",
		},
		{
			name:     "missing default secretref",
			config:   OCIProviderConfig{Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: &S3Credentials{}}},
			path:     path,
			errorMsg: "missing default secretref",
		},
		{
			name: "missing override credentials",
			config: OCIProviderConfig{
				Default:   &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{{BucketName: "kfp-artifacts", KeyPrefix: "team-a"}},
			},
			path:     path,
			errorMsg: "missing override credentials",
		},
		{
			name: "missing override secretref",
			config: OCIProviderConfig{
				Default:   &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret},
				Overrides: []OCIOverride{{BucketName: "kfp-artifacts", KeyPrefix: "team-a", Credentials: &S3Credentials{}}},
			},
			path:     path,
			errorMsg: "missing override secretref",
		},
		{
			name:     "path without namespace is rejected",
			config:   OCIProviderConfig{Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: defaultSecret}},
			path:     "oci://kfp-artifacts/team-a/model",
			errorMsg: "oci://<bucket>@<namespace>/<prefix>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionInfo, err := tt.config.ProvideSessionInfo(tt.path)
			if tt.errorMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, objectstore.SessionInfo{}, sessionInfo)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, sessionInfo)
		})
	}
}

func TestOCIHasExplicitOverride(t *testing.T) {
	providerConfig := OCIProviderConfig{
		Default: &OCIProviderDefault{Region: "us-ashburn-1", Credentials: &S3Credentials{FromEnv: true}},
		Overrides: []OCIOverride{
			{BucketName: "allowlisted-bucket", Namespace: "mynamespace", KeyPrefix: "allowed/", Credentials: &S3Credentials{FromEnv: true}},
		},
	}

	hasOverride, err := providerConfig.HasExplicitOverride("oci://allowlisted-bucket@mynamespace/allowed/path")
	require.NoError(t, err)
	assert.True(t, hasOverride)

	hasOverride, err = providerConfig.HasExplicitOverride("oci://allowlisted-bucket@othernamespace/allowed/path")
	require.NoError(t, err)
	assert.False(t, hasOverride)

	hasOverride, err = providerConfig.HasExplicitOverride("oci://allowlisted-bucket@mynamespace/other/path")
	require.NoError(t, err)
	assert.False(t, hasOverride)

	_, err = providerConfig.HasExplicitOverride("oci://allowlisted-bucket/allowed/path")
	require.Error(t, err)
}
