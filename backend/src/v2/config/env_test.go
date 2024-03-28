// Copyright 2024 The Kubeflow Authors
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
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getDefaultMinioSessionInfo(t *testing.T) {
	actualDefaultSession := getDefaultMinioSessionInfo()
	expectedDefaultSession := objectstore.SessionInfo{
		Region:       "minio",
		Endpoint:     "minio-service.kubeflow:9000",
		DisableSSL:   true,
		SecretName:   "mlpipeline-minio-artifact",
		AccessKeyKey: "accesskey",
		SecretKeyKey: "secretkey",
	}
	assert.Equal(t, actualDefaultSession, expectedDefaultSession)
}

func TestGetBucketSessionInfo(t *testing.T) {
	tt := []struct {
		msg                 string
		config              Config
		expectedSessionInfo objectstore.SessionInfo
		pipelineroot        string
		// optional
		shouldError bool
		// optional
		errorMsg string
	}{
		{
			msg:                 "invalid_unsupported_object_store_protocol",
			pipelineroot:        "unsupported://my-bucket/v2/artifacts",
			config:              Config{},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unsupported Cloud bucket",
		},
		{
			msg:                 "invalid_unsupported_pipeline_root_format",
			pipelineroot:        "minio.unsupported.format",
			config:              Config{},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unrecognized pipeline root format",
		},
		{
			msg:          "valid_no_providers",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config:       Config{},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "minio",
				Endpoint:     "minio-service.kubeflow:9000",
				DisableSSL:   true,
				SecretName:   "mlpipeline-minio-artifact",
				AccessKeyKey: "accesskey",
				SecretKeyKey: "secretkey",
			},
		},
		{
			msg:          "invalid_one_empty_minio_provider",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
minio: {}
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "Invalid provider config",
		},
		{
			msg:          "invalid_one_empty_minio_provider_no_authconfigs",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  authConfigs: []
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "Invalid provider config",
		},
		{
			msg:          "invalid_one_minio_provider_endpoint_only",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  endpoint: some-endpoint.com
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "Invalid provider config",
		},
		{
			msg:          "valid_one_minio_provider",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  endpoint: some-endpoint.com
  region: minio
  authConfigs: []
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "minio",
				Endpoint:     "some-endpoint.com",
				DisableSSL:   false,
				SecretName:   "mlpipeline-minio-artifact",
				AccessKeyKey: "accesskey",
				SecretKeyKey: "secretkey",
			},
		},
		{
			msg:          "valid_pick_matching_provider",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
gcs:
  endpoint: storage.cloud.google.com
  region: gcs
  authConfigs: []
minio:
  endpoint: some-endpoint.com
  region: minio
  authConfigs: []
aws:
  endpoint: s3.amazonaws.com
  region: s3
  authConfigs: []
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "minio",
				Endpoint:     "some-endpoint.com",
				DisableSSL:   false,
				SecretName:   "mlpipeline-minio-artifact",
				AccessKeyKey: "accesskey",
				SecretKeyKey: "secretkey",
			},
		},
		{
			msg:          "invalid_non_minio_should_require_secret",
			pipelineroot: "s3://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
s3:
  endpoint: s3.amazonaws.com
  region: us-east-1
  authConfigs: []
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "Invalid provider config",
		},
		{
			msg:          "invalid_matching_prefix_should_require_secretref",
			pipelineroot: "minio://my-bucket/v2/artifacts/pick/this",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  endpoint: minio.endpoint.com
  region: minio
  authConfigs:
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/skip/this
      secretRef:
        secretName: minio-skip-this-secret
        accessKeyKey: minio_skip_this_access_key
        secretKeyKey: minio_skip_this_secret_key
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/pick/this
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "Invalid provider config",
		},
		{
			msg:          "valid_s3_with_secret",
			pipelineroot: "s3://my-bucket/v2/artifacts",
			config: Config{
				data: map[string]string{
					"providers": `
s3:
  endpoint: s3.amazonaws.com
  region: us-east-1
  defaultProviderSecretRef:
    secretName: "s3-provider-secret"
    accessKeyKey: "different_access_key"
    secretKeyKey: "different_secret_key"
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "us-east-1",
				Endpoint:     "s3.amazonaws.com",
				DisableSSL:   false,
				SecretName:   "s3-provider-secret",
				SecretKeyKey: "different_secret_key",
				AccessKeyKey: "different_access_key",
			},
		},
		{
			msg:          "valid_minio_first_matching_auth_config",
			pipelineroot: "minio://my-bucket/v2/artifacts/pick/this",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  endpoint: minio.endpoint.com
  region: minio
  defaultProviderSecretRef:
      secretName: minio-default-provider-secret
      accessKeyKey: minio_default_different_access_key
      secretKeyKey: minio_default_different_secret_key
  authConfigs:
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/skip/this
      secretRef:
        secretName: minio-skip-this-secret
        accessKeyKey: minio_skip_this_access_key
        secretKeyKey: minio_skip_this_secret_key
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/pick/this
      secretRef:
        secretName: minio-pick-this-secret
        accessKeyKey: minio_pick_this_access_key
        secretKeyKey: minio_pick_this_secret_key
    - bucketName: my-bucket
      keyPrefix: v2/artifacts
      secretRef:
        secretName: minio-not-reached-secret
        accessKeyKey: minio_not_reached_access_key
        secretKeyKey: minio_not_reached_secret_key
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "minio",
				Endpoint:     "minio.endpoint.com",
				DisableSSL:   false,
				SecretName:   "minio-pick-this-secret",
				SecretKeyKey: "minio_pick_this_secret_key",
				AccessKeyKey: "minio_pick_this_access_key",
			},
		},
		{
			msg:          "valid_gs_first_matching_auth_config",
			pipelineroot: "gs://my-bucket/v2/artifacts/pick/this",
			config: Config{
				data: map[string]string{
					"providers": `
minio:
  endpoint: minio.endpoint.com
  region: minio
  defaultProviderSecretRef:
      secretName: minio-default-provider-secret
      accessKeyKey: minio_default_different_access_key
      secretKeyKey: minio_default_different_secret_key
  authConfigs:
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/skip/this
      secretRef:
        secretName: minio-skip-this-secret
        accessKeyKey: minio_skip_this_access_key
        secretKeyKey: minio_skip_this_secret_key
gcs: 
  endpoint: storage.googleapis.com
  region: gcs
  defaultProviderSecretRef:
    secretName: gcs-default-provider-secret
    accessKeyKey: gcs_default_different_access_key
    secretKeyKey: gcs_default_different_secret_key
  authConfigs:
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/pick/this
      secretRef:
        secretName: gcs-pick-this-secret
        accessKeyKey: gcs_pick_this_access_key
        secretKeyKey: gcs_pick_this_secret_key
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "gcs",
				Endpoint:     "storage.googleapis.com",
				DisableSSL:   false,
				SecretName:   "gcs-pick-this-secret",
				SecretKeyKey: "gcs_pick_this_secret_key",
				AccessKeyKey: "gcs_pick_this_access_key",
			},
		},
		{
			msg:          "valid_pick_default_when_no_matching_prefix",
			pipelineroot: "gs://my-bucket/v2/artifacts/pick/default",
			config: Config{
				data: map[string]string{
					"providers": `
gcs: 
  endpoint: storage.googleapis.com
  region: gcs
  defaultProviderSecretRef:
    secretName: gcs-default-provider-secret
    accessKeyKey: gcs_default_different_access_key
    secretKeyKey: gcs_default_different_secret_key
  authConfigs:
    - bucketName: my-bucket
      keyPrefix: v2/artifacts/skip/this
      secretRef:
        secretName: gcs-skip-this-secret
        accessKeyKey: gcs_skip_this_access_key
        secretKeyKey: gcs_skip_this_secret_key
`,
				},
			},
			expectedSessionInfo: objectstore.SessionInfo{
				Region:       "gcs",
				Endpoint:     "storage.googleapis.com",
				DisableSSL:   false,
				SecretName:   "gcs-default-provider-secret",
				SecretKeyKey: "gcs_default_different_secret_key",
				AccessKeyKey: "gcs_default_different_access_key",
			},
		},
	}

	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			actualSession, err := test.config.GetBucketSessionInfo(test.pipelineroot)
			if test.shouldError {
				assert.Error(t, err)
				if err != nil && test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedSessionInfo, actualSession)
		})
	}
}
