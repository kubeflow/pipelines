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

package constants

// The split balances average per-spec durations from twelve recent green
// E2E Critical lanes. New pipelines default to shard B so they remain covered.
var e2eCriticalShardAPipelines = map[string]struct{}{
	"artifact_crust.yaml":                         {},
	"collected_parameters.yaml":                   {},
	"component_with_optional_inputs.yaml":         {},
	"container_component_with_no_inputs.yaml":     {},
	"missing_kubernetes_optional_inputs.yaml":     {},
	"multiple_artifacts_namedtuple.yaml":          {},
	"notebook_component_simple.yaml":              {},
	"parallel_for_after_dependency.yaml":          {},
	"pipeline_with_artifact_custom_path.yaml":     {},
	"pipeline_with_artifact_upload_download.yaml": {},
	"pipeline_with_importer_workspace.yaml":       {},
	"pipeline_with_input_status_state.yaml":       {},
	"pipeline_with_pod_metadata.yaml":             {},
	"pipeline_with_retry.yaml":                    {},
	"pipeline_with_submit_request.yaml":           {},
	"producer_consumer_param_pipeline.yaml":       {},
	"pythonic_artifacts_test_pipeline.yaml":       {},
}

// E2eCriticalShardForPipeline returns the CI shard label for a critical
// pipeline file.
func E2eCriticalShardForPipeline(pipelineFile string) string {
	if _, ok := e2eCriticalShardAPipelines[pipelineFile]; ok {
		return E2eCriticalShardA
	}
	return E2eCriticalShardB
}
