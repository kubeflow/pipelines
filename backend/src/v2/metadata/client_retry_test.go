// Copyright 2026 The Kubeflow Authors
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

package metadata

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsReadOnlyMLMDMethod(t *testing.T) {
	tests := []struct {
		fullMethod string
		want       bool
	}{
		{"/ml_metadata.MetadataStoreService/GetArtifactsByID", true},
		{"/ml_metadata.MetadataStoreService/GetContextByTypeAndName", true},
		{"/ml_metadata.MetadataStoreService/GetLineageSubgraph", true},
		{"/ml_metadata.MetadataStoreService/PutExecution", false},
		{"/ml_metadata.MetadataStoreService/PutArtifactType", false},
		{"/ml_metadata.MetadataStoreService/PutContexts", false},
	}
	for _, test := range tests {
		t.Run(test.fullMethod, func(t *testing.T) {
			assert.Equal(t, test.want, isReadOnlyMLMDMethod(test.fullMethod))
		})
	}
}

func TestMethodAwareRetryInterceptor(t *testing.T) {
	const maxRetries = 3
	readMethod := "/ml_metadata.MetadataStoreService/GetArtifactsByID"
	writeMethod := "/ml_metadata.MetadataStoreService/PutExecution"

	tests := []struct {
		name         string
		method       string
		code         codes.Code
		wantAttempts int
	}{
		{"read retries Internal", readMethod, codes.Internal, maxRetries},
		{"read retries Unknown", readMethod, codes.Unknown, maxRetries},
		// The retry middleware treats DeadlineExceeded/Canceled as caller
		// context errors and never retries them, even if listed in WithCodes.
		{"read does not retry DeadlineExceeded", readMethod, codes.DeadlineExceeded, 1},
		{"read retries Unavailable", readMethod, codes.Unavailable, maxRetries},
		{"read retries Aborted", readMethod, codes.Aborted, maxRetries},
		{"read does not retry NotFound", readMethod, codes.NotFound, 1},
		{"write retries Unavailable", writeMethod, codes.Unavailable, maxRetries},
		{"write retries Aborted", writeMethod, codes.Aborted, maxRetries},
		{"write does not retry Internal", writeMethod, codes.Internal, 1},
		{"write does not retry Unknown", writeMethod, codes.Unknown, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				attempts++
				return status.Error(test.code, "injected failure")
			}
			interceptor := newMethodAwareRetryInterceptor(maxRetries, time.Millisecond)
			err := interceptor(context.Background(), test.method, nil, nil, nil, invoker)
			require.Error(t, err)
			assert.Equal(t, test.code, status.Code(err))
			assert.Equal(t, test.wantAttempts, attempts)
		})
	}

	t.Run("success does not retry", func(t *testing.T) {
		attempts := 0
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			attempts++
			return nil
		}
		interceptor := newMethodAwareRetryInterceptor(maxRetries, time.Millisecond)
		require.NoError(t, interceptor(context.Background(), readMethod, nil, nil, nil, invoker))
		assert.Equal(t, 1, attempts)
	})
}

func TestPositiveIntFromEnv(t *testing.T) {
	const envName = "TEST_POSITIVE_INT_FROM_ENV"

	t.Run("unset returns not ok", func(t *testing.T) {
		_, ok, err := positiveIntFromEnv(envName)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("valid value is returned", func(t *testing.T) {
		t.Setenv(envName, "42")
		value, ok, err := positiveIntFromEnv(envName)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, 42, value)
	})

	t.Run("non-integer errors", func(t *testing.T) {
		t.Setenv(envName, "not-a-number")
		_, _, err := positiveIntFromEnv(envName)
		require.Error(t, err)
	})

	t.Run("zero errors", func(t *testing.T) {
		t.Setenv(envName, "0")
		_, _, err := positiveIntFromEnv(envName)
		require.Error(t, err)
	})

	t.Run("negative errors", func(t *testing.T) {
		t.Setenv(envName, "-5")
		_, _, err := positiveIntFromEnv(envName)
		require.Error(t, err)
	})
}
