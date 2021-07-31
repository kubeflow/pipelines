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

package objectstore_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/kubeflow/pipelines/v2/objectstore"
	_ "gocloud.dev/blob/gcsblob"
)

func Test_parseCloudBucket(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *objectstore.Config
		wantErr bool
	}{
		{
			name: "Parses GCS - Just the bucket",
			path: "gs://my-bucket",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Just the bucket with trailing slash",
			path: "gs://my-bucket/",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix",
			path: "gs://my-bucket/my-path",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix and trailing slash",
			path: "gs://my-bucket/my-path/",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix",
			path: "gs://my-bucket/my-path/123",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/123/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix and trailing slash",
			path: "gs://my-bucket/my-path/123/",
			want: &objectstore.Config{
				Scheme:     "gs://",
				BucketName: "my-bucket",
				Prefix:     "my-path/123/",
			},
			wantErr: false,
		},
		{
			name: "Parses Minio - Bucket with query string",
			path: "minio://my-bucket",
			want: &objectstore.Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "",
				QueryString: "",
			},
			wantErr: false,
		}, {
			name: "Parses Minio - Bucket with prefix",
			path: "minio://my-bucket/my-path",
			want: &objectstore.Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "my-path/",
				QueryString: "",
			},
			wantErr: false,
		}, {
			name: "Parses Minio - Bucket with multiple path components in prefix",
			path: "minio://my-bucket/my-path/123",
			want: &objectstore.Config{
				Scheme:      "minio://",
				BucketName:  "my-bucket",
				Prefix:      "my-path/123/",
				QueryString: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := objectstore.ParseBucketConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("%q: parseCloudBucket() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%q: parseCloudBucket() = %v, want %v", tt.name, got, tt.want)
			}
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
		bucketConfig *objectstore.Config
		uri          string
		want         string
		wantErr      bool
	}{
		{
			name:         "Bucket with empty prefix",
			bucketConfig: &objectstore.Config{Scheme: "gs://", BucketName: "my-bucket", Prefix: ""},
			uri:          "gs://my-bucket/path1/path2",
			want:         "path1/path2",
			wantErr:      false,
		},
		{
			name:         "Bucket with non-empty Prefix ",
			bucketConfig: &objectstore.Config{Scheme: "gs://", BucketName: "my-bucket", Prefix: "path0/"},
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
			got := objectstore.MinioDefaultEndpoint()
			if got != tt.want {
				t.Errorf(
					"MinioDefaultEndpoint() = %q, want %q\nwhen MINIO_SERVICE_SERVICE_HOST=%q MINIO_SERVICE_SERVICE_PORT=%q",
					got, tt.want, tt.minioServiceHostEnv, tt.minioServicePortEnv,
				)
			}
		})
	}
}
