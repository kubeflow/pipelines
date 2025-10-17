// Copyright 2025 The Kubeflow Authors
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

package proto_tests

import (
	"fmt"
	"path/filepath"
	"testing"

	specPB "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

// This is the commit that contains the proto generated files
// that were used to generate the test data.
const commit = "1791485"

func generatePath(path string) string {
	return filepath.Join(fmt.Sprintf("generated-%s", commit), path)
}

func TestRuns(t *testing.T) {
	testOBJ(t, caseOpts[*pb.Run]{
		message:          completedRun,
		expectedPBPath:   generatePath("run_completed.pb"),
		expectedJSONPath: generatePath("run_completed.json"),
	})

	testOBJ(t, caseOpts[*pb.Run]{
		message:          completedRunWithPipelineSpec,
		expectedPBPath:   generatePath("run_completed_with_spec.pb"),
		expectedJSONPath: generatePath("run_completed_with_spec.json"),
	})

	testOBJ(t, caseOpts[*pb.Run]{
		message:          failedRun,
		expectedPBPath:   generatePath("run_failed.pb"),
		expectedJSONPath: generatePath("run_failed.json"),
	})
}

func TestPipelines(t *testing.T) {
	testOBJ(t, caseOpts[*pb.Pipeline]{
		message:          pipeline,
		expectedPBPath:   generatePath("pipeline.pb"),
		expectedJSONPath: generatePath("pipeline.json"),
	})
	testOBJ(t, caseOpts[*pb.PipelineVersion]{
		message:          pipelineVersion,
		expectedPBPath:   generatePath("pipeline_version.pb"),
		expectedJSONPath: generatePath("pipeline_version.json"),
	})
}

func TestExperiments(t *testing.T) {
	testOBJ(t, caseOpts[*pb.Experiment]{
		message:          experiment,
		expectedPBPath:   generatePath("experiment.pb"),
		expectedJSONPath: generatePath("experiment.json"),
	})
}

func TestVisualization(t *testing.T) {
	testOBJ(t, caseOpts[*pb.Visualization]{
		message:          visualization,
		expectedPBPath:   generatePath("visualization.pb"),
		expectedJSONPath: generatePath("visualization.json"),
	})
}

func TestRecurringRun(t *testing.T) {
	testOBJ(t, caseOpts[*pb.RecurringRun]{
		message:          recurringRun,
		expectedPBPath:   generatePath("recurring_run.pb"),
		expectedJSONPath: generatePath("recurring_run.json"),
	})
}

func TestPipelineSpec(t *testing.T) {
	testOBJ(t, caseOpts[*specPB.PipelineSpec]{
		message:          pipelineSpec,
		expectedPBPath:   generatePath("pipeline_spec.pb"),
		expectedJSONPath: generatePath("pipeline_spec.json"),
	})
}

func TestPlatformSpec(t *testing.T) {
	testOBJ(t, caseOpts[*specPB.PlatformSpec]{
		message:          platformSpec,
		expectedPBPath:   generatePath("platform_spec.pb"),
		expectedJSONPath: generatePath("platform_spec.json"),
	})
}
