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

package util

import "testing"

func TestGenerateOutputURI(t *testing.T) {
	const (
		pipelineName      = "my-pipeline-name"
		runID             = "my-run-id"
		pipelineRoot      = "minio://mlpipeline/v2/artifacts"
		pipelineRootQuery = "?query=string&another=query"
	)

	tests := []struct {
		name                string
		pipelineRoot        string
		paths               []string
		preserveQueryString bool
		want                string
	}{
		{
			name:                "plain pipeline root without preserved query",
			pipelineRoot:        pipelineRoot,
			paths:               []string{pipelineName, runID},
			preserveQueryString: false,
			want:                "minio://mlpipeline/v2/artifacts/my-pipeline-name/my-run-id",
		},
		{
			name:                "plain pipeline root with preserved query",
			pipelineRoot:        pipelineRoot,
			paths:               []string{pipelineName, runID},
			preserveQueryString: true,
			want:                "minio://mlpipeline/v2/artifacts/my-pipeline-name/my-run-id",
		},
		{
			name:                "pipeline root with query without preservation",
			pipelineRoot:        pipelineRoot + pipelineRootQuery,
			paths:               []string{pipelineName, runID},
			preserveQueryString: false,
			want:                "minio://mlpipeline/v2/artifacts/my-pipeline-name/my-run-id",
		},
		{
			name:                "pipeline root with query with preservation",
			pipelineRoot:        pipelineRoot + pipelineRootQuery,
			paths:               []string{pipelineName, runID},
			preserveQueryString: true,
			want:                "minio://mlpipeline/v2/artifacts/my-pipeline-name/my-run-id?query=string&another=query",
		},
		{
			name:                "trailing slash is normalized",
			pipelineRoot:        pipelineRoot + "/",
			paths:               []string{pipelineName, runID},
			preserveQueryString: false,
			want:                "minio://mlpipeline/v2/artifacts/my-pipeline-name/my-run-id",
		},
		{
			name:                "multiple query separators are left in place",
			pipelineRoot:        pipelineRoot + "?query=string?another=query",
			paths:               []string{pipelineName, runID},
			preserveQueryString: true,
			want:                "minio://mlpipeline/v2/artifacts?query=string?another=query/my-pipeline-name/my-run-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := GenerateOutputURI(test.pipelineRoot, test.paths, test.preserveQueryString)
			if got != test.want {
				t.Fatalf("GenerateOutputURI(%q, %v, %t) = %q, want %q", test.pipelineRoot, test.paths, test.preserveQueryString, got, test.want)
			}
		})
	}
}
