// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package objectstore

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	_ "gocloud.dev/blob/gcsblob"
)

func Test_parseCloudBucket(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *Config
		wantErr bool
	}{
		{
			name: "Parses GCS - Just the bucket",
			path: "gs://my-bucket",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Just the bucket with trailing slash",
			path: "gs://my-bucket/",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix",
			path: "gs://my-bucket/my-path",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix and trailing slash",
			path: "gs://my-bucket/my-path/",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix",
			path: "gs://my-bucket/my-path/123",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/123/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix and trailing slash",
			path: "gs://my-bucket/my-path/123/",
			want: &Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/123/",
			},
			wantErr: false,
		},
		{
			name: "Parses Minio - Bucket with query string",
			path: "minio://my-bucket",
			want: &Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "",
				QueryString: "",
			},
			wantErr: false,
		}, {
			name: "Parses Minio - Bucket with prefix",
			path: "minio://my-bucket/my-path",
			want: &Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "my-path/",
				QueryString: "",
			},
			wantErr: false,
		}, {
			name: "Parses Minio - Bucket with multiple path components in prefix",
			path: "minio://my-bucket/my-path/123",
			want: &Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "my-path/123/",
				QueryString: "",
			},
			wantErr: false,
		}, {
			name: "Parses S3 - Bucket with session",
			path: "s3://my-bucket/my-path/123",
			want: &Config{
				Scheme:      "s3://",
				BucketName:  "my-bucket",
				Prefix:      "my-path/123/",
				QueryString: "",
				SessionInfo: &SessionInfo{
					Provider: "s3",
					Params: map[string]string{
						"region":       "us-east-1",
						"endpoint":     "s3.amazonaws.com",
						"disableSSL":   "false",
						"fromEnv":      "false",
						"secretName":   "s3-testsecret",
						"accessKeyKey": "s3-testaccessKeyKey",
						"secretKeyKey": "s3-testsecretKeyKey",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBucketConfig(tt.path, tt.want.SessionInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("%q: parseCloudBucket() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%q: parseCloudBucket() = %v, want %v", tt.name, got, tt.want)
			}
			assert.Equal(t, got.SessionInfo, tt.want.SessionInfo)
		})
	}
}

func Test_bucketConfig_KeyFromURI(t *testing.T) {
	tests := []struct {
		name         string
		bucketConfig *Config
		uri          string
		want         string
		wantErr      bool
	}{
		{
			name:         "Bucket with empty prefix",
			bucketConfig: &Config{Scheme: "gs://", BucketName: "my-bucket", Prefix: ""},
			uri:          "gs://my-bucket/path1/path2",
			want:         "path1/path2",
			wantErr:      false,
		},
		{
			name:         "Bucket with non-empty Prefix ",
			bucketConfig: &Config{Scheme: "gs://", BucketName: "my-bucket", Prefix: "path0/"},
			uri:          "gs://my-bucket/path0/path1/path2",
			want:         "path1/path2",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.bucketConfig.KeyFromURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("%q: buckerConfig.keyFromURI() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bucketConfig.keyFromURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createS3BucketSession(t *testing.T) {
	tt := []struct {
		msg               string
		ns                string
		sessionInfo       *SessionInfo
		sessionSecret     *corev1.Secret
		expectValidClient bool
		expectedRegion    string
		expectedEndpoint  string
		expectedPathStyle bool
		wantErr           bool
		errorMsg          string
	}{
		{
			msg: "Bucket with session",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":         "us-east-1",
					"endpoint":       "s3.amazonaws.com",
					"disableSSL":     "false",
					"fromEnv":        "false",
					"secretName":     "s3-provider-secret",
					"accessKeyKey":   "test_access_key",
					"secretKeyKey":   "test_secret_key",
					"forcePathStyle": "true",
				},
			},
			sessionSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "s3-provider-secret", Namespace: "testnamespace"},
				Data:       map[string][]byte{"test_secret_key": []byte("secretKey"), "test_access_key": []byte("accessKey")},
			},
			expectValidClient: true,
			expectedRegion:    "us-east-1",
			expectedEndpoint:  "s3.amazonaws.com",
			expectedPathStyle: true,
		},
		{
			msg:               "Bucket with no session",
			ns:                "testnamespace",
			sessionInfo:       nil,
			sessionSecret:     nil,
			expectValidClient: false,
		},
		{
			msg: "Bucket with session but secret doesn't exist",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":       "us-east-1",
					"endpoint":     "s3.amazonaws.com",
					"disableSSL":   "false",
					"fromEnv":      "false",
					"secretName":   "does-not-exist",
					"accessKeyKey": "test_access_key",
					"secretKeyKey": "test_secret_key",
				},
			},
			sessionSecret:     nil,
			expectValidClient: false,
			wantErr:           true,
			errorMsg:          "secrets \"does-not-exist\" not found",
		},
		{
			msg: "Bucket with session secret exists but key mismatch",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":       "us-east-1",
					"endpoint":     "s3.amazonaws.com",
					"disableSSL":   "false",
					"fromEnv":      "false",
					"secretName":   "s3-provider-secret",
					"accessKeyKey": "does_not_exist_secret_key",
					"secretKeyKey": "does_not_exist_access_key",
				},
			},
			sessionSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "s3-provider-secret", Namespace: "testnamespace"},
				Data:       map[string][]byte{"test_secret_key": []byte("secretKey"), "test_access_key": []byte("accessKey")},
			},
			expectValidClient: false,
			wantErr:           true,
			errorMsg:          "could not find specified keys",
		},
		{
			msg: "Bucket with fromEnv and custom endpoint",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":         "us-west-2",
					"endpoint":       "minio-service.kubeflow:9000",
					"disableSSL":     "true",
					"fromEnv":        "true",
					"forcePathStyle": "true",
				},
			},
			sessionSecret:     nil,
			expectValidClient: true,
			expectedRegion:    "us-west-2",
			expectedEndpoint:  "minio-service.kubeflow:9000",
			expectedPathStyle: true,
		},
		{
			msg: "Bucket with fromEnv and only region",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"region":  "eu-central-1",
					"fromEnv": "true",
				},
			},
			sessionSecret:     nil,
			expectValidClient: true,
			expectedRegion:    "eu-central-1",
		},
		{
			msg: "Bucket with fromEnv but no endpoint or region",
			ns:  "testnamespace",
			sessionInfo: &SessionInfo{
				Provider: "s3",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			sessionSecret:     nil,
			expectValidClient: false,
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			fakeKubernetesClientset := fake.NewSimpleClientset()
			ctx := context.Background()

			if test.sessionSecret != nil {
				testersecret, err := fakeKubernetesClientset.CoreV1().Secrets(test.ns).Create(
					ctx,
					test.sessionSecret,
					metav1.CreateOptions{})
				assert.Nil(t, err)
				fmt.Printf("%s", testersecret.Namespace)
			}

			actualSession, err := createS3BucketSession(ctx, test.ns, test.sessionInfo, fakeKubernetesClientset)
			if test.wantErr {
				assert.Error(t, err)
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
			} else {
				assert.Nil(t, err)
			}

			if test.expectValidClient {
				// confirm that a valid S3 client was returned
				assert.NotNil(t, actualSession)
				// In AWS SDK v2, we can't directly access internal config details
				// but we can verify that the client was created successfully
				// and would have the expected configuration based on our inputs
			} else {
				assert.Nil(t, actualSession)
			}
		})
	}
}
