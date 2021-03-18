// Copyright 2021 Google LLC
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
package component

import (
	"reflect"
	"testing"

	_ "gocloud.dev/blob/gcsblob"
)

func Test_parseCloudBucket(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *bucketConfig
		wantErr bool
	}{
		{
			name: "Parses GCS - Just the bucket",
			path: "gs://my-bucket",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Just the bucket with trailing slash",
			path: "gs://my-bucket/",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix",
			path: "gs://my-bucket/my-path",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with prefix and trailing slash",
			path: "gs://my-bucket/my-path/",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "my-path/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix",
			path: "gs://my-bucket/my-path/123",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "my-path/123/",
			},
			wantErr: false,
		},
		{
			name: "Parses GCS - Bucket with multiple path components in prefix and trailing slash",
			path: "gs://my-bucket/my-path/123/",
			want: &bucketConfig{
				scheme:     "gs://",
				bucketName: "my-bucket",
				prefix:     "my-path/123/",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBucketConfig(tt.path)
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

func Test_bucketConfig_keyFromURI(t *testing.T) {
	type fields struct {
		scheme     string
		bucketName string
		prefix     string
	}

	tests := []struct {
		name         string
		bucketConfig *bucketConfig
		uri          string
		want         string
		wantErr      bool
	}{
		{
			name:         "Bucket with empty prefix",
			bucketConfig: &bucketConfig{scheme: "gs://", bucketName: "my-bucket", prefix: ""},
			uri:          "gs://my-bucket/path1/path2",
			want:         "path1/path2",
			wantErr:      false,
		},
		{
			name:         "Bucket with non-empty prefix ",
			bucketConfig: &bucketConfig{scheme: "gs://", bucketName: "my-bucket", prefix: "path0/"},
			uri:          "gs://my-bucket/path0/path1/path2",
			want:         "path1/path2",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.bucketConfig.keyFromURI(tt.uri)
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
