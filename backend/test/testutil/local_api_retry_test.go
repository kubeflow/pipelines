// Copyright 2026 The Kubeflow Authors
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

package testutil

import (
	"errors"
	"testing"
)

func TestIsRetriableAPITestError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "localhost timeout",
			err:  errors.New(`Get "http://localhost:8888/apis/v2beta1/runs/run-1": context deadline exceeded`),
			want: true,
		},
		{
			name: "backend dns timeout",
			err:  errors.New(`dial tcp: lookup mysql.kubeflow.svc.cluster.local on 10.96.0.10:53: read udp 10.244.0.14:55398->10.96.0.10:53: i/o timeout`),
			want: true,
		},
		{
			name: "non retryable",
			err:  errors.New("permission denied"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := IsRetriableAPITestError(testCase.err)
			if got != testCase.want {
				t.Fatalf("IsRetriableAPITestError() = %v, want %v", got, testCase.want)
			}
		})
	}
}
