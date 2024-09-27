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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"reflect"
	"testing"

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
	type fields struct {
		scheme     string
		bucketName string
		prefix     string
	}

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

func Test_GetMinioDefaultEndpoint(t *testing.T) {
	defer func() {
		os.Unsetenv("MINIO_SERVICE_SERVICE_HOST")
		os.Unsetenv("MINIO_SERVICE_SERVICE_PORT")
	}()
	tests := []struct {
		name                string
		minioServiceHostEnv string
		minioServicePortEnv string
		want                string
	}{
		{
			name:                "In full Kubeflow, KFP multi-user mode on",
			minioServiceHostEnv: "",
			minioServicePortEnv: "",
			want:                "minio-service.kubeflow:9000",
		},
		{
			name:                "In KFP standalone without multi-user mode",
			minioServiceHostEnv: "1.2.3.4",
			minioServicePortEnv: "4321",
			want:                "1.2.3.4:4321",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.minioServiceHostEnv != "" {
				os.Setenv("MINIO_SERVICE_SERVICE_HOST", tt.minioServiceHostEnv)
			} else {
				os.Unsetenv("MINIO_SERVICE_SERVICE_HOST")
			}
			if tt.minioServicePortEnv != "" {
				os.Setenv("MINIO_SERVICE_SERVICE_PORT", tt.minioServicePortEnv)
			} else {
				os.Unsetenv("MINIO_SERVICE_SERVICE_PORT")
			}
			got := MinioDefaultEndpoint()
			if got != tt.want {
				t.Errorf(
					"MinioDefaultEndpoint() = %q, want %q\nwhen MINIO_SERVICE_SERVICE_HOST=%q MINIO_SERVICE_SERVICE_PORT=%q",
					got, tt.want, tt.minioServiceHostEnv, tt.minioServicePortEnv,
				)
			}
		})
	}
}

func Test_createS3BucketSession(t *testing.T) {
	tt := []struct {
		msg            string
		ns             string
		sessionInfo    *SessionInfo
		sessionSecret  *corev1.Secret
		expectedConfig *aws.Config
		wantErr        bool
		errorMsg       string
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
			expectedConfig: &aws.Config{
				Credentials:      credentials.NewStaticCredentials("accessKey", "secretKey", ""),
				Region:           aws.String("us-east-1"),
				Endpoint:         aws.String("s3.amazonaws.com"),
				DisableSSL:       aws.Bool(false),
				S3ForcePathStyle: aws.Bool(true),
			},
		},
		{
			msg:            "Bucket with no session",
			ns:             "testnamespace",
			sessionInfo:    nil,
			sessionSecret:  nil,
			expectedConfig: nil,
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
			sessionSecret:  nil,
			expectedConfig: nil,
			wantErr:        true,
			errorMsg:       "secrets \"does-not-exist\" not found",
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
			expectedConfig: nil,
			wantErr:        true,
			errorMsg:       "could not find specified keys",
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
				fmt.Printf(testersecret.Namespace)
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

			if test.expectedConfig != nil {
				// confirm config is populated with values from the session
				expectedSess, err := session.NewSession(test.expectedConfig)
				assert.Nil(t, err)
				assert.Equal(t, expectedSess.Config.Region, actualSession.Config.Region)
				assert.Equal(t, expectedSess.Config.Credentials, actualSession.Config.Credentials)
				assert.Equal(t, expectedSess.Config.DisableSSL, actualSession.Config.DisableSSL)
				assert.Equal(t, expectedSess.Config.S3ForcePathStyle, actualSession.Config.S3ForcePathStyle)
			} else {
				assert.Nil(t, actualSession)
			}
		})
	}
}
