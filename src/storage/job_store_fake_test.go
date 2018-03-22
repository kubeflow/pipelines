// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFakePersistentJobStoreCreateJob(t *testing.T) {
	pipelineID := uint(12345)
	workflow := &v1alpha1.Workflow{}
	workflow.Name = "WORKFLOW_NAME"
	jobStore := NewFakePersistentJobStore()
	jobStore.CreateJob(pipelineID, workflow)

	// Testing that the workflow is in the store.
	result, err := jobStore.GetJob(pipelineID, workflow.Name)
	assert.Equal(t, result.Workflow.Name, workflow.Name, "Unexpected job name. Expect %v. Got %v.",
		workflow.Name, result.Workflow.Name)

	// Testing that the store does not return any workflow for a different pipeline name.
  otherPipelineID := uint(55555)
	result, err = jobStore.GetJob(otherPipelineID, workflow.Name)
	assert.NotNil(t, err, "The error should not be null.")

	// Testing that the store does not return any workflow for a different workflow name.
	otherWorkflowName := "OTHER_WORKFLOW_NAME"
	result, err = jobStore.GetJob(pipelineID, otherWorkflowName)
	assert.NotNil(t, err, "The error should not be null.")

	// Testing the count of workflows for the pipeline.
	pipelineCount := jobStore.GetJobCountForPipeline(pipelineID)
	assert.Equal(t, pipelineCount, 1, "There should be a single workflow for pipeline %v.",
		pipelineID)

	// Testing the count of workflows for another pipeline.
	pipelineCount = jobStore.GetJobCountForPipeline(otherPipelineID)
	assert.Equal(t, pipelineCount, 0, "There should be no workflow for pipeline %v.",
		otherPipelineID)
}

func TestGetJobCountForPipeline(t *testing.T) {
	pipeline1 := uint(11111)
	pipeline2 := uint(22222)
	pipeline3 := uint(33333)
	workflow1 := &v1alpha1.Workflow{}
	workflow1.Name = "WORKFLOW_NAME_1"
	workflow2 := &v1alpha1.Workflow{}
	workflow2.Name = "WORKFLOW_NAME_2"
	workflow3 := &v1alpha1.Workflow{}
	workflow3.Name = "WORKFLOW_NAME_3"

	jobStore := NewFakePersistentJobStore()
	jobStore.CreateJob(pipeline1, workflow1)
	jobStore.CreateJob(pipeline1, workflow2)
	jobStore.CreateJob(pipeline2, workflow3)

	pipelineCount := jobStore.GetJobCountForPipeline(pipeline1)
	assert.Equal(t, pipelineCount, 2,
		"Wrong number of jobs for pipeline %v. Expect %v. Got %v.",
		2, pipelineCount)

	pipelineCount = jobStore.GetJobCountForPipeline(pipeline2)
	assert.Equal(t, pipelineCount, 1,
		"Wrong number of jobs for pipeline %v. Expect %v. Got %v.",
		1, pipelineCount)

	pipelineCount = jobStore.GetJobCountForPipeline(pipeline3)
	assert.Equal(t, pipelineCount, 0,
		"Wrong number of jobs for pipeline %v. Expect %v. Got %v.",
		0, pipelineCount)
}