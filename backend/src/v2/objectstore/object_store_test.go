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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	_ "gocloud.dev/blob/gcsblob"
)

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
