// Copyright 2018 Google LLC
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
	"fmt"
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type FakeBadObjectStore struct{}

func (m *FakeBadObjectStore) AddFile(template []byte, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) DeleteFile(filePath string) error {
	return errors.New("Not implemented.")
}

func (m *FakeBadObjectStore) GetFile(filePath string) ([]byte, error) {
	return []byte(""), nil
}

func (m *FakeBadObjectStore) AddAsYamlFile(o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetFromYamlFile(o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

var testWorkflow = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1"},
	Spec:       v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}}},
	Status:     v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeRunning},
})

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Pipeline) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	p, err := manager.CreatePipeline("p1", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, p
}

func initWithExperiment(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	experiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(experiment)
	assert.Nil(t, err)
	return store, manager, experiment
}

func initWithExperimentAndPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	experiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(experiment)
	assert.Nil(t, err)
	pipeline, err := manager.CreatePipeline("p1", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, experiment, pipeline
}

// Util function to create an initial state with pipeline uploaded
func initWithJob(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &api.Job{
		Name:         "j1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	j, err := manager.CreateJob(job)
	assert.Nil(t, err)

	return store, manager, j
}

func initWithOneTimeRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.NodeFailed
	updatedWorkflow.Status.Nodes = map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	manager.ReportWorkflowResource(updatedWorkflow)
	return store, manager, runDetail
}

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:   name,
		Status: model.PipelineReady,
		DefaultVersion: &model.PipelineVersion{
			Name:   name + "_version",
			Status: model.PipelineVersionReady,
		}}
}

func TestCreatePipeline(t *testing.T) {
	store, _, pipeline := initWithPipeline(t)
	defer store.Close()
	pipelineExpected := &model.Pipeline{
		UUID:             DefaultFakeUUID,
		CreatedAtInSec:   1,
		Name:             "p1",
		Parameters:       "[{\"name\":\"param1\"}]",
		Status:           model.PipelineReady,
		DefaultVersionId: DefaultFakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "p1",
			Parameters:     "[{\"name\":\"param1\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
		}}
	assert.Equal(t, pipelineExpected, pipeline)
}

func TestCreatePipeline_ComplexPipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	createdPipeline, err := manager.CreatePipeline("pipeline1", "", []byte(strings.TrimSpace(
		complexPipeline)))
	assert.Nil(t, err)
	_, err = manager.GetPipeline(createdPipeline.UUID)
	assert.Nil(t, err)
}

func TestCreatePipeline_GetParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePipeline("pipeline1", "", []byte("I am invalid yaml"))
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to parse the parameter")
}

func TestCreatePipeline_StorePipelineMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePipeline("pipeline1", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to start a transaction to create a new pipeline")
}

func TestCreatePipeline_CreatePipelineFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	_, err := manager.CreatePipeline("pipeline1", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
	// Verify there is a pipeline in DB with status PipelineCreating.
	pipeline, err := manager.pipelineStore.GetPipelineWithStatus(DefaultFakeUUID, model.PipelineCreating)
	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
}

func TestGetPipelineTemplate(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetPipelineTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, []byte(testWorkflow.ToStringForStore()), actualTemplate)
}

func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.CreatePipelinePath(fmt.Sprint(1)))
	manager := NewResourceManager(store)
	_, err := manager.GetPipelineTemplate("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetPipelineTemplate_PipelineFileNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("pipeline1"))
	manager := NewResourceManager(store)
	_, err := manager.GetPipelineTemplate(pipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "object not found")
}

func TestCreateRun_ThroughPipelineID(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	experiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(experiment)
	assert.Nil(t, err)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: util.StringPointer("world")}}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = defaultPipelineRunnerServiceAccount

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			DisplayName:    "run1",
			Name:           "workflow-name",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 3,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				PipelineId:           p.UUID,
				PipelineName:         "p1",
				WorkflowSpecManifest: testWorkflow.ToStringForStore(),
				Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
					ResourceType:  common.Run,
					ReferenceUUID: experiment.UUID,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.workflowClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpec(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: util.StringPointer("world")}}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = defaultPipelineRunnerServiceAccount
	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			DisplayName:    "run1",
			Name:           "workflow-name",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 2,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: testWorkflow.ToStringForStore(),
				Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
					ResourceType:  common.Run,
					ReferenceUUID: DefaultFakeUUID,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.workflowClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_NoExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "No experiment",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		// No experiment
		ResourceReferences: []*api.ResourceReference{},
	}
	runDetail, err := manager.CreateRun(apiRun)
	assert.Nil(t, err)
	expectedRunDetail := []*model.ResourceReference{{
		ResourceUUID: "123e4567-e89b-12d3-a456-426655440000",
		ResourceType: common.Run,
		// Experiment is now set
		ReferenceUUID: DefaultFakeUUID,
		ReferenceName: "Default",
		ReferenceType: common.Experiment,
		Relationship:  common.Owner,
	}}
	assert.Equal(t, expectedRunDetail, runDetail.Run.ResourceReferences, "The CreateRun return has unexpected value.")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail.Run.ResourceReferences, "CreateRun stored invalid data in database")
}

func TestCreateRun_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch workflow spec")
}

func TestCreateRun_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string("I am invalid"),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to unmarshal workflow spec manifest")
}

func TestCreateRun_OverrideParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param2", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unrecognized input parameter")
}

func TestCreateRun_CreateWorkflowError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	manager.workflowClient = &FakeBadWorkflowClient{}
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a workflow")
}

func TestCreateRun_StoreRunMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.DB().Close()
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.DeleteRun(runDetail.UUID)
	assert.Nil(t, err)

	_, err = manager.GetRun(runDetail.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteRun("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_CrdFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	manager.workflowClient = &FakeBadWorkflowClient{}
	err := manager.DeleteRun(runDetail.UUID)
	//assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	//assert.Contains(t, err.Error(), "some error")
	// TODO(IronPan) This should return error if swf CRD doesn't cascade delete runs.
	assert.Nil(t, err)
}

func TestDeleteRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteRun(runDetail.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)

	_, err = manager.GetExperiment(experiment.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_ClearsDefaultExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	// Set default experiment ID. This is not normally done manually
	err := manager.SetDefaultExperimentId(experiment.UUID)
	assert.Nil(t, err)
	// Verify that default experiment ID is set
	defaultExperimentId, err := manager.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, experiment.UUID, defaultExperimentId)

	err = manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)

	_, err = manager.GetExperiment(experiment.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")

	// Verify that default experiment ID has been cleared
	defaultExperimentId, err = manager.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "", defaultExperimentId)
}

func TestDeleteExperiment_ExperimentNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteExperiment("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_CrdFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	manager.workflowClient = &FakeBadWorkflowClient{}
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)
}

func TestDeleteExperiment_DbFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestTerminateRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	err := manager.TerminateRun(runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, "Terminating", actualRunDetail.Conditions)

	isTerminated, err := store.workflowClientFake.isTerminated(runDetail.Run.Name)
	assert.Nil(t, err)
	assert.True(t, isTerminated)
}

func TestTerminateRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.TerminateRun("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestTerminateRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.TerminateRun(runDetail.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestRetryRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, actualRunDetail.WorkflowRuntimeManifest, "Failed")

	err = manager.RetryRun(runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, actualRunDetail.WorkflowRuntimeManifest, "Running")
}

func TestRetryRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.RetryRun("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryRun_FailedDeletePods(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.podClient = FakeBadPodClient{}
	err := manager.RetryRun(runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestRetryRun_UpdateAndCreateFailed(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.workflowClient = &FakeBadWorkflowClient{}
	err := manager.RetryRun(runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create or update the run")
}

func TestCreateJob_ThroughWorkflowSpec(t *testing.T) {
	store, _, job := initWithJob(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "default",
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123",
				ResourceType:  common.Job,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Equal(t, expectedJob, job)
}

func TestCreateJob_ThroughPipelineID(t *testing.T) {
	store, manager, pipeline := initWithPipeline(t)
	defer store.Close()
	experiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(experiment)
	job := &api.Job{
		Name:    "j1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	newJob, err := manager.CreateJob(job)
	expectedJob := &model.Job{
		UUID:           "123",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "default",
		Enabled:        true,
		CreatedAtInSec: 3,
		UpdatedAtInSec: 3,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipeline.UUID,
			PipelineName:         "p1",
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123",
				ResourceType:  common.Job,
				ReferenceUUID: experiment.UUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestCreateJob_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	job := &api.Job{
		Name:    "pp 1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateJob(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch workflow spec")
}

func TestCreateJob_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	job := &api.Job{
		Name:    "pp 1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string("I am invalid"),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateJob(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to unmarshal workflow spec manifest")
}

func TestCreateJob_ExtraInputParameterError(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	job := &api.Job{
		Name:    "pp 1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: []*api.Parameter{
				{Name: "param2", Value: "world"},
			},
		},
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unrecognized input parameter: param2")
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	manager.scheduledWorkflowClient = &FakeBadScheduledWorkflowClient{}
	job := &api.Job{
		Name:         "pp1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineId: p.UUID},
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to create a scheduled workflow")
}

func TestEnableJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.EnableJob(job.UUID, false)
	job, err = manager.GetJob(job.UUID)
	expectedJob := &model.Job{
		UUID:           "123",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "default",
		Enabled:        false,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123",
				ResourceType:  common.Job,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, job)
}

func TestEnableJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.EnableJob("1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CrdFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.scheduledWorkflowClient = &FakeBadScheduledWorkflowClient{}
	err := manager.EnableJob(job.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestEnableJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.DB().Close()
	err := manager.EnableJob(job.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(job.UUID)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 123 not found")
}

func TestDeleteJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteJob("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestDeleteJob_CrdFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	manager.scheduledWorkflowClient = &FakeBadScheduledWorkflowClient{}
	err := manager.DeleteJob(job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestDeleteJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteJob(job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestReportWorkflowResource_ScheduledWorkflowIDEmpty_Success(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			UID:    types.UID(run.UUID),
			Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeRunning},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)
	runDetail, err := manager.GetRun(run.UUID)
	assert.Nil(t, err)
	expectedRun := model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "run1",
		Name:           "workflow-name",
		StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
		CreatedAtInSec: 2,
		Conditions:     "Running",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Run,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Equal(t, expectedRun, runDetail.Run)
}

func TestReportWorkflowResource_ScheduledWorkflowIDNotEmpty_Success(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "WORKFLOW_1",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "WORKFLOW_1"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(job.UUID),
			}},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
		},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "WORKFLOW_1",
			DisplayName:      "MY_NAME",
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Name:             "MY_NAME",
			Namespace:        "MY_NAMESPACE",
			CreatedAtInSec:   11,
			ScheduledAtInSec: 0,
			FinishedAtInSec:  0,
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: workflow.GetWorkflowSpec().ToStringForStore(),
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: job.UUID,
					ReferenceName: job.Name,
					ReferenceType: common.Job,
					Relationship:  common.Creator,
				},
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: DefaultFakeUUID,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestReportWorkflowResource_ScheduledWorkflowIDNotEmpty_NoExperiment_Success(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	job := &api.Job{
		Name:         "j1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()},
		// no experiment reference
	}
	newJob, err := manager.CreateJob(job)

	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "WORKFLOW_1",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "WORKFLOW_1"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(newJob.UUID),
			}},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
		},
	})

	err = manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "WORKFLOW_1",
			DisplayName:      "MY_NAME",
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Name:             "MY_NAME",
			Namespace:        "MY_NAMESPACE",
			CreatedAtInSec:   11,
			ScheduledAtInSec: 0,
			FinishedAtInSec:  0,
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: workflow.GetWorkflowSpec().ToStringForStore(),
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: newJob.UUID,
					ReferenceName: newJob.Name,
					ReferenceType: common.Job,
					Relationship:  common.Creator,
				},
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: DefaultFakeUUID,
					ReferenceName: "Default",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestReportWorkflowResource_WorkflowCompleted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:   run.Name,
			UID:    types.UID(run.UUID),
			Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeFailed},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)

	wf, err := store.workflowClientFake.Get(run.Run.Name, v1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.Labels[util.LabelKeyWorkflowPersistedFinalState], "true")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:   run.Name,
			UID:    types.UID(run.UUID),
			Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeFailed},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_DeleteFailed(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	manager.workflowClient = &FakeBadWorkflowClient{}
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:   run.Name,
			UID:    types.UID(run.UUID),
			Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeFailed},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete workflow")
}

func TestReportScheduledWorkflowResource_Success(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(job.UUID),
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		Name:        "MY_NAME",
		DisplayName: "j1",
		Namespace:   "MY_NAMESPACE",
		Enabled:     false,
		UUID:        job.UUID,
		Conditions:  "NO_STATUS",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  job.UUID,
				ResourceType:  common.Job,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	assert.Equal(t, expectedJob, actualJob)
}

func TestReportScheduledWorkflowResource_Error(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create pipeline
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}})
	p, err := manager.CreatePipeline("1", "", []byte(workflow.ToStringForStore()))
	assert.Nil(t, err)

	// Create job
	job := &api.Job{
		Name:         "pp1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineId: p.UUID},
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	store.Close()

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newJob.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf)
	assert.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.(*util.UserError).String(), "database is closed")
}

func TestGetWorkflowSpecBytes_ByPipelineID(t *testing.T) {
	store, manager, pipeline := initWithPipeline(t)
	defer store.Close()
	spec := &api.PipelineSpec{
		PipelineId: pipeline.UUID,
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "world"},
		},
	}
	workflowBytes, err := manager.getWorkflowSpecBytes(spec)
	assert.Nil(t, err)
	var actualWorkflow v1alpha1.Workflow
	json.Unmarshal(workflowBytes, &actualWorkflow)
	assert.Equal(t, testWorkflow.Get(), &actualWorkflow)
}

func TestGetWorkflowSpecBytes_ByPipelineID_NotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	spec := &api.PipelineSpec{
		PipelineId: "1",
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "world"},
		},
	}
	_, err := manager.getWorkflowSpecBytes(spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetWorkflowSpecBytes_ByWorkflowManifest(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	spec := &api.PipelineSpec{
		WorkflowManifest: "some manifest",
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "world"},
		},
	}
	workflowBytes, err := manager.getWorkflowSpecBytes(spec)
	assert.Nil(t, err)
	assert.Equal(t, []byte("some manifest"), workflowBytes)
}

func TestGetWorkflowSpecBytes_MissingSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	spec := &api.PipelineSpec{
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "world"},
		},
	}
	_, err := manager.getWorkflowSpecBytes(spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please provide a valid pipeline spec")
}

func TestReadArtifact_Succeed(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	expectedContent := "test"
	filePath := "test/file.txt"
	store.ObjectStore().AddFile([]byte(expectedContent), filePath)

	// Create a scheduled run
	// job, _ := manager.CreateJob(&api.Job{
	// 	Name:       "pp1",
	// 	PipelineId: p.UUID,
	// 	Enabled:    true,
	// })
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "MY_NAME",
			Namespace:         "MY_NAMESPACE",
			UID:               "run-1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				"node-1": {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: "artifact-1",
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)

	artifactContent, err := manager.ReadArtifact("run-1", "node-1", "artifact-1")
	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(artifactContent))
}

func TestReadArtifact_WorkflowNoStatus_NotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "MY_NAME",
			Namespace:         "MY_NAMESPACE",
			UID:               "run-1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(job.UUID),
			}},
		}})
	err := manager.ReportWorkflowResource(workflow)
	assert.Nil(t, err)

	_, err = manager.ReadArtifact("run-1", "node-1", "artifact-1")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReadArtifact_NoRun_NotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.ReadArtifact("run-1", "node-1", "artifact-1")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestCreatePipelineVersion(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	p, err := manager.CreatePipeline(
		"p1", "", []byte(testWorkflow.ToStringForStore()))
	fmt.Printf("JING p %+v\n", p)
	assert.Nil(t, err)

	// store.pipelineStore.uuid =
	// 	util.NewFakeUUIDGeneratorOrFatal(fakeUUIDOne, nil)
	version, err := manager.CreatePipelineVersion(
		&api.PipelineVersion{
			Name: "p_v_1",
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   "p1",
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		},
		[]byte(testWorkflow.ToStringForStore()))

	defer store.Close()
	pipelineVersionExpected := &model.PipelineVersion{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 2,
		Name:           "p_v_1",
		Parameters:     "[{\"name\":\"param1\"}]",
		Status:         model.PipelineVersionReady,
		PipelineId:     DefaultFakeUUID,
	}
	assert.Equal(t, pipelineVersionExpected, version)
}

const (
	complexPipeline = `
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: tfmataxicabclassificationpipelineexample-
spec:
  arguments:
    parameters:
    - name: output
    - name: project
    - name: schema
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json
    - name: train
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv
    - name: evaluation
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv
    - name: preprocess-mode
      value: local
    - name: preprocess-module
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py
    - name: target
      value: tips
    - name: learning-rate
      value: '0.1'
    - name: hidden-layer-size
      value: '1500'
    - name: steps
      value: '3000'
    - name: workers
      value: '0'
    - name: pss
      value: '0'
    - name: predict-mode
      value: local
    - name: analyze-mode
      value: local
    - name: analyze-slice-column
      value: trip_start_hour
  entrypoint: tfmataxicabclassificationpipelineexample
  templates:
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/analysis'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.analyze-mode}}'
      - --slice-columns
      - '{{inputs.parameters.analyze-slice-column}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: output
      - name: project
      - name: schema
      - name: training-train
    name: analysis
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: analysis-analysis
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/predict'
      - --data
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --mode
      - '{{inputs.parameters.predict-mode}}'
      - --project
      - '{{inputs.parameters.project}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: predict-mode
      - name: project
      - name: schema
      - name: target
      - name: training-train
    name: prediction
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: prediction-predict
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --train
      - '{{inputs.parameters.train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/transformed'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.preprocess-mode}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tft
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: schema
      - name: train
    name: preprocess
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: preprocess-transformed
        valueFrom:
          path: /output.txt
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: analyze-mode
            value: '{{inputs.parameters.analyze-mode}}'
          - name: analyze-slice-column
            value: '{{inputs.parameters.analyze-slice-column}}'
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: analysis
        template: analysis
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: predict-mode
            value: '{{inputs.parameters.predict-mode}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: prediction
        template: prediction
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-mode
            value: '{{inputs.parameters.preprocess-mode}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: train
            value: '{{inputs.parameters.train}}'
        name: preprocess
        template: preprocess
      - arguments:
          parameters:
          - name: hidden-layer-size
            value: '{{inputs.parameters.hidden-layer-size}}'
          - name: learning-rate
            value: '{{inputs.parameters.learning-rate}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: preprocess-transformed
            value: '{{tasks.preprocess.outputs.parameters.preprocess-transformed}}'
          - name: pss
            value: '{{inputs.parameters.pss}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: steps
            value: '{{inputs.parameters.steps}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: workers
            value: '{{inputs.parameters.workers}}'
        dependencies:
        - preprocess
        name: training
        template: training
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: predict-mode
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: train
      - name: workers
    name: tfmataxicabclassificationpipelineexample
  - container:
      args:
      - --job-dir
      - '{{inputs.parameters.output}}/{{workflow.name}}/train'
      - --transformed-data-dir
      - '{{inputs.parameters.preprocess-transformed}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --learning-rate
      - '{{inputs.parameters.learning-rate}}'
      - --hidden-layer-size
      - '{{inputs.parameters.hidden-layer-size}}'
      - --steps
      - '{{inputs.parameters.steps}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --workers
      - '{{inputs.parameters.workers}}'
      - --pss
      - '{{inputs.parameters.pss}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      - --tfjob-timeout-minutes
      - '60'
      image: gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf
    inputs:
      parameters:
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: preprocess-module
      - name: preprocess-transformed
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: workers
    name: training
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: training-train
        valueFrom:
          path: /output.txt`
)
