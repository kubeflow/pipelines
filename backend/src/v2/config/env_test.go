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
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"os"
	"sigs.k8s.io/yaml"
	"testing"
)

type TestcaseData struct {
	Testcases []ProviderCase `json:"cases"`
}
type ProviderCase struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Test_getDefaultMinioSessionInfo(t *testing.T) {
	actualDefaultSession, err := getDefaultMinioSessionInfo()
	assert.Nil(t, err)
	expectedDefaultSession := objectstore.SessionInfo{
		Provider: "minio",
		Params: map[string]string{
			"region":       "minio",
			"endpoint":     "minio-service.kubeflow:9000",
			"disableSSL":   "true",
			"fromEnv":      "false",
			"secretName":   "mlpipeline-minio-artifact",
			"accessKeyKey": "accesskey",
			"secretKeyKey": "secretkey",
		},
	}
	assert.Equal(t, expectedDefaultSession, actualDefaultSession)
}

func TestGetBucketSessionInfo(t *testing.T) {

	providersDataFile, err := os.ReadFile("testdata/provider_cases.yaml")
	if os.IsNotExist(err) {
		panic(err)
	}

	var providersData TestcaseData
	err = yaml.Unmarshal(providersDataFile, &providersData)
	if err != nil {
		panic(err)
	}

	tt := []struct {
		msg                 string
		config              Config
		expectedSessionInfo objectstore.SessionInfo
		pipelineroot        string
		shouldError         bool
		errorMsg            string
		testDataCase        string
	}{
		{
			msg:                 "invalid - unsupported object store protocol",
			pipelineroot:        "unsupported://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unsupported Cloud bucket",
		},
		{
			msg:          "valid - only s3 pipelineroot no provider config",
			pipelineroot: "s3://my-bucket",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
		},
		{
			msg:                 "invalid - unsupported pipeline root format",
			pipelineroot:        "minio.unsupported.format",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unrecognized pipeline root format",
		},
		{
			msg:          "valid - no providers, should use minio default",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio",
					"endpoint":     "minio-service.kubeflow:9000",
					"disableSSL":   "true",
					"fromEnv":      "false",
					"secretName":   "mlpipeline-minio-artifact",
					"accessKeyKey": "accesskey",
					"secretKeyKey": "secretkey",
				},
			},
		},
		{
			msg:          "valid - no s3 provider match providers config",
			pipelineroot: "s3://my-bucket",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			testDataCase: "case0",
		},
		{
			msg:          "valid - no gcs provider match providers config",
			pipelineroot: "gs://my-bucket",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			testDataCase: "case0",
		},
		{
			msg:          "valid - no minio provider match providers config, use default minio config",
			pipelineroot: "minio://my-bucket",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio",
					"endpoint":     "minio-service.kubeflow:9000",
					"disableSSL":   "true",
					"fromEnv":      "false",
					"secretName":   "mlpipeline-minio-artifact",
					"accessKeyKey": "accesskey",
					"secretKeyKey": "secretkey",
				},
			},
			testDataCase: "case1",
		},
		{
			msg:          "valid - empty minio provider, use default minio config",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio",
					"endpoint":     "minio-service.kubeflow:9000",
					"disableSSL":   "true",
					"fromEnv":      "false",
					"secretName":   "mlpipeline-minio-artifact",
					"accessKeyKey": "accesskey",
					"secretKeyKey": "secretkey",
				},
			},
			testDataCase: "case1",
		},
		{
			msg:                 "invalid - empty minio provider no override",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case2",
		},
		{
			msg:                 "invalid - minio provider endpoint only",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case3",
		},
		{
			msg:                 "invalid - one minio provider no creds",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "missing default credentials",
			testDataCase:        "case4",
		},
		{
			msg:          "valid - minio provider with default only",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":         "minio",
					"endpoint":       "minio-endpoint-5.com",
					"disableSSL":     "true",
					"fromEnv":        "false",
					"secretName":     "test-secret-5",
					"accessKeyKey":   "test-accessKeyKey-5",
					"secretKeyKey":   "test-secretKeyKey-5",
					"forcePathStyle": "true",
				},
			},
			testDataCase: "case5",
		},
		{
			msg:          "valid - pick minio provider",
			pipelineroot: "minio://minio-bucket-a/some/minio/path/a",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":         "minio-a",
					"endpoint":       "minio-endpoint-6.com",
					"disableSSL":     "true",
					"fromEnv":        "false",
					"secretName":     "minio-test-secret-6-a",
					"accessKeyKey":   "minio-test-accessKeyKey-6-a",
					"secretKeyKey":   "minio-test-secretKeyKey-6-a",
					"forcePathStyle": "true",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:                 "invalid - s3 should require default creds",
			pipelineroot:        "s3://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "missing default credentials",
			testDataCase:        "case7",
		},
		{
			msg:                 "invalid - gs should require default creds",
			pipelineroot:        "gs://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "missing default credentials",
			testDataCase:        "case7",
		},
		{
			msg:                 "invalid - minio should require default creds",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "missing default credentials",
			testDataCase:        "case7",
		},
		{
			msg:                 "invalid - matching prefix override should require secretref, if fromEnv is false",
			pipelineroot:        "minio://minio-bucket-a/some/minio/path/a",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "missing override secretref",
			testDataCase:        "case8",
		},
		{
			msg:          "valid - matching prefix override should use creds from env, if fromEnv is true",
			pipelineroot: "minio://minio-bucket-a/some/minio/path/a",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":         "minio-a",
					"endpoint":       "minio-endpoint-9.com",
					"disableSSL":     "true",
					"fromEnv":        "true",
					"forcePathStyle": "true",
				},
			},
			testDataCase: "case9",
		},
		{
			msg:          "valid - matching prefix override should use secret creds, even if default uses FromEnv",
			pipelineroot: "minio://minio-bucket-a/some/minio/path/a",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":         "minio-a",
					"endpoint":       "minio-endpoint-10.com",
					"disableSSL":     "true",
					"fromEnv":        "false",
					"secretName":     "minio-test-secret-10",
					"accessKeyKey":   "minio-test-accessKeyKey-10",
					"secretKeyKey":   "minio-test-secretKeyKey-10",
					"forcePathStyle": "true",
				},
			},
			testDataCase: "case10",
		},
		{
			msg:          "valid - secret ref is not required for default s3 when fromEnv is true",
			pipelineroot: "minio://minio-bucket-a/some/minio/path/b",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":         "minio",
					"endpoint":       "minio-endpoint-10.com",
					"disableSSL":     "true",
					"fromEnv":        "true",
					"forcePathStyle": "true",
				},
			},
			testDataCase: "case10",
		},
		{
			msg:          "valid - match s3 default config when no override match exists",
			pipelineroot: "s3://s3-bucket/no/override/path",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":         "us-east-1",
					"endpoint":       "s3.amazonaws.com",
					"disableSSL":     "false",
					"fromEnv":        "false",
					"secretName":     "s3-testsecret-6",
					"accessKeyKey":   "s3-testaccessKeyKey-6",
					"secretKeyKey":   "s3-testsecretKeyKey-6",
					"forcePathStyle": "false",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:          "valid - override should match first subpath prefix in pipelineroot",
			pipelineroot: "s3://s3-bucket/some/s3/path/b",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":         "us-east-2",
					"endpoint":       "s3.us-east-2.amazonaws.com",
					"disableSSL":     "false",
					"fromEnv":        "false",
					"secretName":     "s3-test-secret-6-b",
					"accessKeyKey":   "s3-test-accessKeyKey-6-b",
					"secretKeyKey":   "s3-test-secretKeyKey-6-b",
					"forcePathStyle": "false",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:          "valid - test order, match first subpath prefix in pipelineroot, ignoring deeper path prefix further in list",
			pipelineroot: "s3://s3-bucket/some/s3/path/a/b",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":         "us-east-1",
					"endpoint":       "s3.amazonaws.com",
					"disableSSL":     "false",
					"fromEnv":        "false",
					"secretName":     "s3-test-secret-6-a",
					"accessKeyKey":   "s3-test-accessKeyKey-6-a",
					"secretKeyKey":   "s3-test-secretKeyKey-6-a",
					"forcePathStyle": "false",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:          "valid - first matching gs override",
			pipelineroot: "gs://gs-bucket-a/some/gs/path/1",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv":    "false",
					"secretName": "gs-test-secret-6-a",
					"tokenKey":   "gs-test-tokenKey-6-a",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:          "valid - pick default gs when no matching prefix",
			pipelineroot: "gs://path/does/not/exist/so/use/default",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv":    "false",
					"secretName": "gs-test-secret-6",
					"tokenKey":   "gs-test-tokenKey-6",
				},
			},
			testDataCase: "case6",
		},
		{
			msg:          "valid - gs secretref not required when default is set to env",
			pipelineroot: "gs://path/does/not/exist/so/use/default",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			testDataCase: "case11",
		},
		{
			msg:          "valid - gs secretref not required when matching override is set to env",
			pipelineroot: "gs://gs-bucket/some/gs/path/1/2",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			testDataCase: "case11",
		},
		{
			msg:          "valid - gs secretref is required when matching override is fromEnv:false",
			pipelineroot: "gs://gs-bucket/some/gs/path/1",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"fromEnv":    "false",
					"secretName": "gs-test-secret-11",
					"tokenKey":   "gs-test-tokenKey-11",
				},
			},
			testDataCase: "case11",
		},
	}

	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			config := Config{data: map[string]string{}}
			if test.testDataCase != "" {
				config.data["providers"] = fetchProviderFromData(providersData, test.testDataCase)
				if config.data["providers"] == "" {
					panic(fmt.Errorf("provider not found in testdata"))
				}
			}

			actualSession, err1 := config.GetStoreSessionInfo(test.pipelineroot)
			if test.shouldError {
				assert.Error(t, err1)
				if err1 != nil && test.errorMsg != "" {
					assert.Contains(t, err1.Error(), test.errorMsg)
				}
			} else {
				assert.Nil(t, err1)
			}

			assert.Equal(t, test.expectedSessionInfo, actualSession)
		})
	}
}

func Test_QueryParameters(t *testing.T) {
	providersDataFile, err := os.ReadFile("testdata/provider_cases.yaml")
	if os.IsNotExist(err) {
		panic(err)
	}

	var providersData TestcaseData
	err = yaml.Unmarshal(providersDataFile, &providersData)
	if err != nil {
		panic(err)
	}

	tt := []struct {
		msg                 string
		config              Config
		expectedSessionInfo objectstore.SessionInfo
		pipelineroot        string
		shouldError         bool
		errorMsg            string
		testDataCase        string
	}{
		{
			msg:          "valid - for s3 fetch fromEnv when when query parameters are present, and when no matching provider config is provided",
			pipelineroot: "s3://bucket_name/v2/artifacts/profile_name?region=bucket_region&endpoint=endpoint&disableSSL=not_use_ssl&s3ForcePathStyle=true",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			shouldError: false,
		},
		{
			msg:          "valid - for minio fetch fromEnv when when query parameters are present, and when no matching provider config is provided",
			pipelineroot: "minio://bucket_name/v2/artifacts/profile_name?region=bucket_region&endpoint=endpoint&disableSSL=not_use_ssl&s3ForcePathStyle=true",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			shouldError: false,
		},
		{
			msg:          "valid - for minio fetch fromEnv when when query parameters are present, and when matching provider config is provided",
			pipelineroot: "minio://bucket_name/v2/artifacts/profile_name?region=bucket_region&endpoint=endpoint&disableSSL=not_use_ssl&s3ForcePathStyle=true",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			shouldError:  false,
			testDataCase: "case12",
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			config := Config{data: map[string]string{}}
			if test.testDataCase != "" {
				config.data["providers"] = fetchProviderFromData(providersData, test.testDataCase)
				if config.data["providers"] == "" {
					panic(fmt.Errorf("provider not found in testdata"))
				}
			}
			actualSession, err1 := config.GetStoreSessionInfo(test.pipelineroot)
			if test.shouldError {
				assert.Error(t, err1)
				if err1 != nil && test.errorMsg != "" {
					assert.Contains(t, err1.Error(), test.errorMsg)
				}
			} else {
				assert.Nil(t, err1)
			}
			assert.Equal(t, test.expectedSessionInfo, actualSession)
		})
	}
}

func fetchProviderFromData(cases TestcaseData, name string) string {
	for _, c := range cases.Testcases {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
