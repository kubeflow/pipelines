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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasStructuredS3Settings(t *testing.T) {
	assert.False(t, HasStructuredS3Settings(map[string]string{}))

	assert.False(t, HasStructuredS3Settings(map[string]string{
		S3ParamFromEnv:      "true",
		S3ParamSecretName:   "secret",
		S3ParamAccessKeyKey: "access",
		S3ParamSecretKeyKey: "key",
	}))

	assert.True(t, HasStructuredS3Settings(map[string]string{
		S3ParamFromEnv: "true",
		S3ParamRegion:  "us-east-1",
	}))

	for _, key := range []string{
		S3ParamRegion,
		S3ParamEndpoint,
		S3ParamDisableSSL,
		S3ParamForcePathStyle,
		S3ParamMaxRetries,
	} {
		t.Run(key, func(t *testing.T) {
			assert.True(t, HasStructuredS3Settings(map[string]string{key: "x"}))
		})
	}
}
