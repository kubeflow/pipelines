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
	"testing"

	"github.com/kubeflow/pipelines/v2/objectstore"
)

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
