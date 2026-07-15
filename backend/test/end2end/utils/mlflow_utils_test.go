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

package utils

import (
	"errors"
	"testing"
)

func TestIsRetriableRetryRunError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "localhost eof",
			err:  errors.New(`Post "http://localhost:8888/apis/v2beta1/runs/run-1:retry": EOF`),
			want: true,
		},
		{
			name: "workflow modified conflict",
			err: errors.New(
				`Failed to retry a run: InternalServerError: Failed to retry run due to error updating and creating a workflow. ` +
					`Update error: Operation cannot be fulfilled on workflows.argoproj.io "fail-pipeline": ` +
					`the object has been modified; please apply your changes to the latest version and try again`,
			),
			want: true,
		},
		{
			name: "workflow name already exists",
			err: errors.New(
				`workflows.argoproj.io "fail-pipeline" already exists, the server was not able to generate a unique name for the object`,
			),
			want: true,
		},
		{
			name: "non retryable error",
			err:  errors.New("permission denied"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := isRetriableRetryRunError(testCase.err)
			if got != testCase.want {
				t.Fatalf("isRetriableRetryRunError() = %v, want %v", got, testCase.want)
			}
		})
	}
}
