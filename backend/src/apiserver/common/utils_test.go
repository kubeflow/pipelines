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
	"reflect"
	"testing"
)

func TestParseResourceIdsFromFullName(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			"namespace-experiment-run",
			args{"namespaces/default/experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-experiment-job-run",
			args{"namespaces/default/experiments/Default/jobs/j1/runs/run-one"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "run-one",
				"RecurringRunId":    "j1",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"empty-namespace-experiment-run",
			args{"namespaces//experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"experiment-run",
			args{"experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-pipeline-version",
			args{"namespaces/default/pipelines/p1/versions/pv1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunId":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-experiment-job-exec-artifact",
			args{"namespaces/default/experiments/Default/jobs/j2/executions/e1/artifacts/a1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"everything",
			args{"namespaces/default/experiments/Default/pipelines/p1/versions/pv1/jobs/j2/runs/r1/executions/e1/artifacts/a1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunId":             "r1",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"empty",
			args{""},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"everything starts and ends with /",
			args{"///namespaces/default/experiments/Default/pipelines/p1/versions/pv1/jobs/j2/runs/r1/executions/e1/artifacts/a1//"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunId":             "r1",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"wrong keys",
			args{"///foo/bar/fiat/lux"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunId":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseResourceIdsFromFullName(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseResourceIdsFromFullName() = %v, want %v", got, tt.want)
			}
		})
	}
}
