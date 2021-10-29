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

package resource

import (
	"github.com/ghodss/yaml"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"testing"
)



func TestRetryWorkflowWith(t *testing.T) {
	wf := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-05-26T09:14:07Z"
  generateName: resubmit-
  generation: 1
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: resubmit-hl9ft
  namespace: kubeflow
  resourceVersion: "13488984"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/kubeflow/workflows/resubmit-hl9ft
  uid: 4628dce4-b4f5-11e9-b75e-42010a8001b8
spec:
  arguments: {}
  entrypoint: rand-fail-dag
  templates:
  - dag:
      tasks:
      - arguments: {}
        name: A
        template: random-fail
      - arguments: {}
        dependencies:
        - A
        name: B
        template: random-fail
      - arguments: {}
        dependencies:
        - B
        name: C
        template: random-fail
    inputs: {}
    metadata: {}
    name: rand-fail-dag
    outputs: {}
  - container:
      args:
      - import random; import sys; exit_code = random.choice([0, 0, 1]); print('exiting
        with code {}'.format(exit_code)); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: random-fail
    outputs: {}
status:
  finishedAt: "2021-05-26T09:14:29Z"
  nodes:
    resubmit-hl9ft:
      children:
      - resubmit-hl9ft-3929423573
      displayName: resubmit-hl9ft
      finishedAt: "2021-05-26T09:14:29Z"
      id: resubmit-hl9ft
      name: resubmit-hl9ft
      phase: Failed
      startedAt: "2021-05-26T09:14:07Z"
      templateName: rand-fail-dag
      type: DAG
    resubmit-hl9ft-3879090716:
      boundaryID: resubmit-hl9ft
      children:
      - resubmit-hl9ft-3895868335
      displayName: B
      finishedAt: "2021-05-26T09:14:23Z"
      id: resubmit-hl9ft-3879090716
      message: failed with exit code 1
      name: resubmit-hl9ft.B
      phase: Failed
      startedAt: "2021-05-26T09:14:19Z"
      templateName: random-fail
      type: Pod
    resubmit-hl9ft-3895868335:
      boundaryID: resubmit-hl9ft
      displayName: C
      finishedAt: "2021-05-26T09:14:29Z"
      id: resubmit-hl9ft-3895868335
      message: 'omitted: depends condition not met'
      name: resubmit-hl9ft.C
      phase: Omitted
      startedAt: "2021-05-26T09:14:29Z"
      templateName: random-fail
      type: Skipped
    resubmit-hl9ft-3929423573:
      boundaryID: resubmit-hl9ft
      children:
      - resubmit-hl9ft-3879090716
      displayName: A
      finishedAt: "2021-05-26T09:14:11Z"
      id: resubmit-hl9ft-3929423573
      name: resubmit-hl9ft.A
      phase: Succeeded
      startedAt: "2021-05-26T09:14:07Z"
      templateName: random-fail
      type: Pod
  phase: Failed
  startedAt: "2021-05-26T09:14:07Z"
`

	var workflow util.Workflow
	err := yaml.Unmarshal([]byte(wf), &workflow)
	assert.Nil(t, err)
	newWf, nodes, err := formulateRetryWorkflow(&workflow)

	newWfString, err := yaml.Marshal(newWf)
	assert.Nil(t, err)
	assert.Equal(t, []string{"resubmit-hl9ft-3879090716"}, nodes)

	expectedNewWfString :=
		`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-05-26T09:14:07Z"
  generateName: resubmit-
  generation: 1
  labels:
    workflows.argoproj.io/phase: Running
  name: resubmit-hl9ft
  namespace: kubeflow
  resourceVersion: "13488984"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/kubeflow/workflows/resubmit-hl9ft
  uid: 4628dce4-b4f5-11e9-b75e-42010a8001b8
spec:
  arguments: {}
  entrypoint: rand-fail-dag
  templates:
  - dag:
      tasks:
      - arguments: {}
        name: A
        template: random-fail
      - arguments: {}
        dependencies:
        - A
        name: B
        template: random-fail
      - arguments: {}
        dependencies:
        - B
        name: C
        template: random-fail
    inputs: {}
    metadata: {}
    name: rand-fail-dag
    outputs: {}
  - container:
      args:
      - import random; import sys; exit_code = random.choice([0, 0, 1]); print('exiting
        with code {}'.format(exit_code)); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: random-fail
    outputs: {}
status:
  finishedAt: null
  nodes:
    resubmit-hl9ft:
      children:
      - resubmit-hl9ft-3929423573
      displayName: resubmit-hl9ft
      finishedAt: null
      id: resubmit-hl9ft
      name: resubmit-hl9ft
      phase: Running
      startedAt: "2021-05-26T09:14:07Z"
      templateName: rand-fail-dag
      type: DAG
    resubmit-hl9ft-3929423573:
      boundaryID: resubmit-hl9ft
      children:
      - resubmit-hl9ft-3879090716
      displayName: A
      finishedAt: "2021-05-26T09:14:11Z"
      id: resubmit-hl9ft-3929423573
      name: resubmit-hl9ft.A
      phase: Succeeded
      startedAt: "2021-05-26T09:14:07Z"
      templateName: random-fail
      type: Pod
  phase: Running
  startedAt: "2021-05-26T09:14:07Z"
`

	assert.Equal(t, expectedNewWfString, string(newWfString))
}

func TestConvertPipelineIdToDefaultPipelineVersion(t *testing.T) {
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_run",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   pipeline.UUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	}, []byte(testWorkflow.ToStringForStore()), true)
	assert.Nil(t, err)

	// Create a run of the latest pipeline version, but by specifying the pipeline id.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	expectedApiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: FakeUUIDOne},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	err = convertPipelineIdToDefaultPipelineVersion(apiRun.PipelineSpec, &apiRun.ResourceReferences, manager)
	assert.Nil(t, err)
	assert.Equal(t, expectedApiRun, apiRun)
}

// No conversion if a pipeline version already exists in resource references.
func TestConvertPipelineIdToDefaultPipelineVersion_NoOp(t *testing.T) {
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()

	// Create a new pipeline version with UUID being FakeUUID.
	oldVersionId := pipeline.DefaultVersionId
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_run",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   pipeline.UUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	}, []byte(testWorkflow.ToStringForStore()), true)
	assert.Nil(t, err)
	// FakeUUID is the new default version's id.
	assert.NotEqual(t, oldVersionId, FakeUUIDOne)

	// Create a run by specifying both the old pipeline version and the pipeline.
	// As a result, the old version will be used and the pipeline id will be ignored.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: oldVersionId},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	expectedApiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: oldVersionId},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	err = convertPipelineIdToDefaultPipelineVersion(apiRun.PipelineSpec, &apiRun.ResourceReferences, manager)
	assert.Nil(t, err)
	assert.Equal(t, expectedApiRun, apiRun)
}
