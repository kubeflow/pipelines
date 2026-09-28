// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"errors"
	"strconv"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestPipelineSizeLimits(t *testing.T) {
	names := []string{MaxPipelineUploadBytesEnv, MaxPipelineSpecBytesEnv, MaxPipelineUpdateBodyBytesEnv}
	for _, name := range names {
		t.Setenv(name, "")
	}
	limits, err := GetPipelineSizeLimits()
	require.NoError(t, err)
	require.Equal(t, PipelineSizeLimits{32 << 20, 32 << 20, 32 << 20}, limits)
	for i, name := range names {
		t.Run(name, func(t *testing.T) {
			for _, value := range []string{"1", "67108864", strconv.Itoa(MaximumPipelineSizeBytes)} {
				t.Run(value, func(t *testing.T) {
					t.Setenv(name, value)
					got, err := GetPipelineSizeLimits()
					require.NoError(t, err)
					want := []int{32 << 20, 32 << 20, 32 << 20}
					want[i], _ = strconv.Atoi(value)
					require.Equal(t, PipelineSizeLimits{want[0], want[1], want[2]}, got)
				})
			}
			for _, value := range []string{"0", "-1", "32MiB", "1.5", "secret-invalid-value", "9223372036854775808", strconv.Itoa(MaximumPipelineSizeBytes + 1)} {
				t.Run(value, func(t *testing.T) {
					t.Setenv(name, value)
					_, err := GetPipelineSizeLimits()
					require.ErrorContains(t, err, name)
					require.ErrorContains(t, err, "unset it")
					require.NotContains(t, err.Error(), "secret-invalid-value")
				})
			}
		})
	}
}

func TestSizeLimitErrorIsSafeAndInvalidArgument(t *testing.T) {
	err := NewSizeLimitError("pipeline_upload", 32<<20, MaxPipelineUploadBytesEnv)
	var limitErr *SizeLimitError
	require.True(t, errors.As(err, &limitErr))
	require.Contains(t, limitErr.Error(), "33554432 bytes (32.00 MiB)")
	require.Contains(t, limitErr.Error(), MaxPipelineUploadBytesEnv)
	require.Contains(t, limitErr.Error(), "container image or object store")
	var userErr *util.UserError
	require.True(t, errors.As(err, &userErr))
	require.Equal(t, codes.InvalidArgument, userErr.ExternalStatusCode())
}
