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
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func initEnvVars() {
	viper.Set(common.PodNamespace, "ns1")
}

type FakeBadObjectStore struct{}

func (m *FakeBadObjectStore) GetPipelineKey(pipelineID string) string {
	return pipelineID
}

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
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
	Spec: v1alpha1.WorkflowSpec{
		Entrypoint: "testy",
		Templates: []v1alpha1.Template{v1alpha1.Template{
			Name: "testy",
			Container: &corev1.Container{
				Image:   "docker/whalesay",
				Command: []string{"cowsay"},
				Args:    []string{"hello world"},
			},
		}},
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}}},
	Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
})

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Pipeline) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	p, err := manager.CreatePipeline("p1", "", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, p
}

func initWithExperiment(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return store, manager, experiment
}

func initWithExperimentAndPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	pipeline, err := manager.CreatePipeline("p1", "", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, experiment, pipeline
}

func initWithExperimentAndPipelineAndRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.RunDetail) {
	store, manager, exp, pipeline := initWithExperimentAndPipeline(t)
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

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
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
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, exp, pipeline, runDetail
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
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

// Util function to create an initial state with pipeline uploaded
func initWithJobV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &api.Job{
		Name:         "j1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineManifest: v2SpecHelloWorld},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	j, err := manager.CreateJob(context.Background(), job)
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
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeRunV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithPatchedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "{{kfp-default-bucket}}"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
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
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.Nodes = map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunCompressed(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
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
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	nodes := map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	nodeData, err := json.Marshal(nodes)
	assert.Nil(t, err)
	updatedWorkflow.Status.CompressedNodes = file.CompressEncodeString(string(nodeData))
	err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunOffloaded(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
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
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.OffloadNodeStatusVersion = "offload-hash"
	err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
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
	tt := []struct {
		msg            string
		name           string // optional
		description    string // optional
		template       string // pipeline template
		badObjectStore bool   // optional, object requests always fail
		badDB          bool   // optional, DB request always fail
		// The following are expected results.
		model *model.Pipeline // optional, expected pipeline model when success
		// To verify an error, set the errorCode and
		// optionally set errorMsg and errorIs based on the test's needs.
		errorCode codes.Code
		errorMsg  string // error message
		errorIs   error  // verify a wrapped error is specific instance
	}{
		{
			msg:         "HappyCase",
			template:    testWorkflow.ToStringForStore(),
			name:        "p_v",
			description: "test",
			model: &model.Pipeline{
				Name:        "p_v",
				Parameters:  "[{\"name\":\"param1\"}]",
				Description: "test",
			},
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			name:     "complex",
			model: &model.Pipeline{
				Name:       "complex",
				Parameters: "[{\"name\":\"output\"},{\"name\":\"project\"},{\"name\":\"schema\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json\"},{\"name\":\"train\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv\"},{\"name\":\"evaluation\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv\"},{\"name\":\"preprocess-mode\",\"value\":\"local\"},{\"name\":\"preprocess-module\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py\"},{\"name\":\"target\",\"value\":\"tips\"},{\"name\":\"learning-rate\",\"value\":\"0.1\"},{\"name\":\"hidden-layer-size\",\"value\":\"1500\"},{\"name\":\"steps\",\"value\":\"3000\"},{\"name\":\"workers\",\"value\":\"0\"},{\"name\":\"pss\",\"value\":\"0\"},{\"name\":\"predict-mode\",\"value\":\"local\"},{\"name\":\"analyze-mode\",\"value\":\"local\"},{\"name\":\"analyze-slice-column\",\"value\":\"trip_start_hour\"}]",
			},
		},
		{
			msg:            "BadObjectStore",
			badObjectStore: true,
			template:       testWorkflow.ToStringForStore(),
			errorCode:      codes.Internal,
			errorMsg:       "bad object store",
			// We previously verified that the failed pipeline version
			// in DB is in status PipelineVersionCreating by faking
			// the UUID generator, so that we know the created version
			// UUID in advance.
			// We cannot verify it using public APIs,
			// because the API does not expose them unless we know its UUID, but we
			// cannot know its UUID when create version request failed.
			// TODO: do we really need to verify this status? or should
			// the create version request return a UUID when the
			// pipeline version fails in PipelineVersionCreating state.
		},
		{
			msg:       "InvalidTemplate",
			template:  "I am invalid yaml",
			errorCode: codes.InvalidArgument,
			errorIs:   template.ErrorInvalidPipelineSpec,
		},
		{
			msg:       "BadDB",
			template:  testWorkflow.ToStringForStore(),
			badDB:     true,
			errorCode: codes.Internal,
			errorMsg:  "database is closed",
		},
		{
			msg:      "V2PipelineSpec",
			template: v2SpecHelloWorld,
			name:     "v2spec",
			model: &model.Pipeline{
				Name: "v2spec",
				// TODO(v2): when parameter extraction is implemented, this won't be empty.
				Parameters: "[]",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			// setup
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store)
			if test.badObjectStore {
				manager.objectStore = &FakeBadObjectStore{}
			}
			if test.badDB {
				store.DB().Close()
			}

			// start test
			if test.name == "" {
				test.name = "my_pipeline_name"
			}
			pipeline, err := manager.CreatePipeline(
				test.name,
				test.description,
				"",
				// Do not upload test.template here, because pipeline API is out of test scope.
				[]byte(test.template),
			)

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				if test.errorIs != nil {
					assert.ErrorIs(t, err, test.errorIs)
				}
				return
			}
			require.Nil(t, err)

			test.model.CreatedAtInSec = 1
			test.model.Status = "READY"
			test.model.UUID = pipeline.UUID
			test.model.DefaultVersionId = pipeline.DefaultVersion.UUID
			test.model.DefaultVersion = &model.PipelineVersion{
				UUID:           pipeline.DefaultVersion.UUID,
				Name:           test.model.Name,
				CreatedAtInSec: 1,
				Parameters:     test.model.Parameters,
				PipelineId:     pipeline.UUID,
				Status:         model.PipelineVersionStatus(pipeline.Status),
			}
			assert.Equal(t, test.model, pipeline)
		})
	}
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
	store.objectStore.AddFile(template, store.objectStore.GetPipelineKey(fmt.Sprint(1)))
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

// TODO: use table driven test to test CreateRun api
func TestCreateRun_ThroughPipelineID(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_run",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   p.UUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	}, []byte(testWorkflow.ToStringForStore()), true)
	assert.Nil(t, err)

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
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
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}
	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: experiment.UUID,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 4,
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
				{
					ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
					ResourceType:  common.Run,
					ReferenceUUID: version.UUID,
					ReferenceName: version.Name,
					ReferenceType: common.PipelineVersion,
					Relationship:  common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRunV2(t)
	expectedExperimentUUID := runDetail.ExperimentUUID

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: expectedExperimentUUID,
			DisplayName:    "run1",
			Name:           "hello-world-0",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 2,
			PipelineSpec: model.PipelineSpec{
				PipelineSpecManifest: v2SpecHelloWorld,
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
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}


func TestCreateRun_ThroughWorkflowSpec(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	expectedExperimentUUID := runDetail.ExperimentUUID
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: expectedExperimentUUID,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
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
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecWithPatch(t *testing.T) {
	viper.Set(common.HasDefaultBucketEnvVar, "true")
	viper.Set(common.ProjectIDEnvVar, "test-project-id")
	viper.Set(common.DefaultBucketNameEnvVar, "test-default-bucket")
	store, manager, runDetail := initWithPatchedRun(t)
	expectedExperimentUUID := runDetail.ExperimentUUID
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("test-default-bucket")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: expectedExperimentUUID,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 2,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: testWorkflow.ToStringForStore(),
				Parameters:           "[{\"name\":\"param1\",\"value\":\"test-default-bucket\"}]",
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
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughPipelineVersion(t *testing.T) {
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
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

	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: version.UUID},
				Relationship: api.Relationship_CREATOR,
			},
		},
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: experiment.UUID,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "sa1",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 4,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
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
				{
					ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
					ResourceType:  common.Run,
					ReferenceUUID: version.UUID,
					ReferenceName: "version_for_run",
					ReferenceType: common.PipelineVersion,
					Relationship:  common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
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

	apiRun := &api.Run{
		Name: "run1",
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
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: version.UUID},
				Relationship: api.Relationship_CREATOR,
			},
		},
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "123e4567-e89b-12d3-a456-426655440000",
			ExperimentUUID: experiment.UUID,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "sa1",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 4,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				PipelineId:           pipeline.UUID,
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
				{
					ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
					ResourceType:  common.Run,
					ReferenceUUID: version.UUID,
					ReferenceName: "version_for_run",
					ReferenceType: common.PipelineVersion,
					Relationship:  common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, runDetail, "The CreateRun return has unexpected value.")
	assert.Equal(t, 1, store.ArgoClientFake.GetWorkflowCount(), "Workflow CRD is not created.")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail, runDetail, "CreateRun stored invalid data in database")
}

func TestCreateRun_NoExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	experimentID, err := manager.CreateDefaultExperiment()
	experiment, err := manager.GetExperiment(experimentID)
	assert.Equal(t, experiment.Name, "Default")

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
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
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
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch manifest bytes")
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
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRun_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: "null", // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
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
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unrecognized input parameter")
}

func TestCreateRun_CreateWorkflowError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	manager.argoClient = client.NewFakeArgoClientWithBadWorkflow()
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
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
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	_, err = manager.GetRun(runDetail.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_CrdFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	manager.argoClient = client.NewFakeArgoClientWithBadWorkflow()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	//assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	//assert.Contains(t, err.Error(), "some error")
	// TODO(IronPan) This should return error if swf CRD doesn't cascade delete runs.
	assert.Nil(t, err)
}

func TestDeleteRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
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

	manager.argoClient = client.NewFakeArgoClientWithBadWorkflow()
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

	err := manager.TerminateRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, "Terminating", actualRunDetail.Conditions)

	isTerminated, err := store.ArgoClientFake.IsTerminated(runDetail.Run.Name)
	assert.Nil(t, err)
	assert.True(t, isTerminated)
}

func TestTerminateRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.TerminateRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestTerminateRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.TerminateRun(context.Background(), runDetail.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestRetryRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, actualRunDetail.WorkflowRuntimeManifest, "Failed")

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, actualRunDetail.WorkflowRuntimeManifest, "Running")
}

func TestRetryRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.RetryRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryRun_FailedDeletePods(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestRetryRun_FailedDeletePodsCompressed(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRunCompressed(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestRetryRun_FailedOffloadNodeStatus(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRunOffloaded(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Cannot retry workflow with offloaded node status")
}

func TestRetryRun_UpdateAndCreateFailed(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.argoClient = client.NewFakeArgoClientWithBadWorkflow()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create or update the run")
}

// TODO Use table driven to write UT to test CreateJob
func TestCreateJob_ThroughWorkflowSpec(t *testing.T) {
	store, _, job := initWithJob(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
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

func TestCreateJob_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Equal(t, expectedJob, job)
	fetchedJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, fetchedJob, "CreateJob stored invalid data in database")
}

func TestCreateJob_ThroughPipelineID(t *testing.T) {
	store, manager, pipeline := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
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

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
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

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 4,
		UpdatedAtInSec: 4,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipeline.UUID,
			PipelineName:         "p1",
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: experiment.UUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: version.UUID,
				ReferenceName: version.Name,
				ReferenceType: common.PipelineVersion,
				Relationship:  common.Creator,
			},
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestCreateJob_ThroughPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_job",
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

	job := &api.Job{
		Name:    "j1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: version.UUID},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 4,
		UpdatedAtInSec: 4,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: experiment.UUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: version.UUID,
				ReferenceName: "version_for_job",
				ReferenceType: common.PipelineVersion,
				Relationship:  common.Creator,
			},
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestCreateJob_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	version, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_job",
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
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: version.UUID},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 4,
		UpdatedAtInSec: 4,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			PipelineName:         "p1",
			PipelineId:           pipeline.UUID,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: experiment.UUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: version.UUID,
				ReferenceName: "version_for_job",
				ReferenceType: common.PipelineVersion,
				Relationship:  common.Creator,
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
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch manifest bytes")
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
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateJob_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	job := &api.Job{
		Name:    "pp 1",
		Enabled: true,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string("null"), // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
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
	_, err := manager.CreateJob(context.Background(), job)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unrecognized input parameter: param2")
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	job := &api.Job{
		Name:         "pp1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineId: p.UUID},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to create a scheduled workflow")
}

func TestEnableJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.EnableJob(context.Background(), job.UUID, false)
	job, err = manager.GetJob(job.UUID)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
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
	err := manager.EnableJob(context.Background(), "1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.EnableJob(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestEnableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, &v1.DeleteOptions{})
	// When swf CR is missing, enabling the job needs to fail.
	err := manager.EnableJob(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed")
	assert.Contains(t, err.Error(), "not found")
}

func TestDisableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	require.Equal(t, job.Enabled, true)

	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, &v1.DeleteOptions{})
	err := manager.EnableJob(context.Background(), job.UUID, false)
	require.Nil(t, err, "Disabling the job should succeed even when the custom resource is missing.")
	job, err = manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.Equal(t, job.Enabled, false)
}

func TestEnableJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.DB().Close()
	err := manager.EnableJob(context.Background(), job.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteJob(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestDeleteJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Delete job CR failed: some error")
}

func TestDeleteJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, &v1.DeleteOptions{})

	// Now deleting job should still succeed when the swf CR is already deleted.
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Nil(t, err)

	// And verify Job has been deleted from DB too.
	_, err = manager.GetJob(job.UUID)
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestReportWorkflowResource_ScheduledWorkflowIDEmpty_Success(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	expectedExperimentUUID := run.ExperimentUUID
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			Namespace: "ns1",
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)
	runDetail, err := manager.GetRun(run.UUID)
	assert.Nil(t, err)
	expectedRun := model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentUUID: expectedExperimentUUID,
		DisplayName:    "run1",
		Name:           "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
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
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "WORKFLOW_1",
			ExperimentUUID:   DefaultFakeUUID,
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
					ReferenceUUID: DefaultFakeUUID,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: job.UUID,
					ReferenceName: job.Name,
					ReferenceType: common.Job,
					Relationship:  common.Creator,
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
	newJob, err := manager.CreateJob(context.Background(), job)

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

	err = manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "WORKFLOW_1",
			ExperimentUUID:   DefaultFakeUUID,
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
					ReferenceUUID: DefaultFakeUUID,
					ReferenceName: "Default",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
				{
					ResourceUUID:  "WORKFLOW_1",
					ResourceType:  common.Run,
					ReferenceUUID: newJob.UUID,
					ReferenceName: newJob.Name,
					ReferenceType: common.Job,
					Relationship:  common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestReportWorkflowResource_WorkflowMissingRunID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: run.Name,
		},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow[workflow-name] missing the Run ID label")
}

func TestReportWorkflowResource_RunNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	ctx := context.Background()
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "obsolete",
			Namespace: "kubeflow",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-id-not-exist"},
		},
	})
	store.ArgoClient().Workflow("kubeflow").Create(ctx, workflow.Workflow, v1.CreateOptions{})
	err := manager.ReportWorkflowResource(ctx, workflow)
	require.NotNil(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Run run-id-not-exist not found")
}

func TestReportWorkflowResource_WorkflowCompleted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := "kubeflow"
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.Name,
			Namespace: namespace,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	wf, err := store.ArgoClientFake.Workflow(namespace).Get(context.Background(), run.Run.Name, v1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.Labels[util.LabelKeyWorkflowPersistedFinalState], "true")
}

func TestReportWorkflowResource_WorkflowCompleted_WorkflowNotFound(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "non-existent-workflow",
			Namespace: "kubeflow",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NotNil(t, err)
	assert.Equalf(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), "Expected not found error, but got %s", err.Error())
	assert.Contains(t, err.Error(), "Failed to add PersistedFinalState label")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.Name,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_WorkflowNotFound(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "non-existent-workflow",
			Namespace: "kubeflow",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NotNil(t, err)
	assert.Equalf(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), "Expected not found error, but got %s", err.Error())
	assert.Contains(t, err.Error(), "Failed to delete the completed workflow")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_DeleteFailed(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	manager.argoClient = client.NewFakeArgoClientWithBadWorkflow()
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.Name,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
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
		Name:           "MY_NAME",
		DisplayName:    "j1",
		Namespace:      "MY_NAMESPACE",
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           job.UUID,
		Conditions:     "NO_STATUS",
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
	p, err := manager.CreatePipeline("1", "", "", []byte(workflow.ToStringForStore()))
	assert.Nil(t, err)

	// Create job
	job := &api.Job{
		Name:         "pp1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineId: p.UUID},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
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
	workflowBytes, err := manager.getWorkflowSpecBytesFromPipelineSpec(spec)
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
	_, err := manager.getWorkflowSpecBytesFromPipelineSpec(spec)
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
	err := manager.ReportWorkflowResource(context.Background(), workflow)
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
	err := manager.ReportWorkflowResource(context.Background(), workflow)
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

const (
	v2compatPipeline = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: two-step-pipeline-
  annotations:
    pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
    pipelines.kubeflow.org/pipeline_compilation_time: '2021-07-14T06:59:20.208189'
    pipelines.kubeflow.org/pipeline_spec: '{"inputs": [{"default": "", "name": "pipeline-root"},
      {"default": "pipeline/two_step_pipeline", "name": "pipeline-name"}], "name":
      "two_step_pipeline"}'
    pipelines.kubeflow.org/v2_pipeline: "true"
  labels:
    pipelines.kubeflow.org/v2_pipeline: "true"
    pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
spec:
  entrypoint: two-step-pipeline
  templates:
  - name: preprocess
    container:
      args:
      - sh
      - -ec
      - |
        program_path=$(mktemp)
        printf "%s" "$0" > "$program_path"
        python3 -u "$program_path" "$@"
      - |
        def _make_parent_dirs_and_return_path(file_path: str):
            import os
            os.makedirs(os.path.dirname(file_path), exist_ok=True)
            return file_path

        def preprocess(
            uri, some_int, output_parameter_one,
            output_dataset_one
        ):
            '''Dummy Preprocess Step.'''
            with open(output_dataset_one, 'w') as f:
                f.write('Output dataset')
            with open(output_parameter_one, 'w') as f:
                f.write("{}".format(1234))

        import argparse
        _parser = argparse.ArgumentParser(prog='Preprocess', description='Dummy Preprocess Step.')
        _parser.add_argument("--uri", dest="uri", type=str, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--some-int", dest="some_int", type=int, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--output-parameter-one", dest="output_parameter_one", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--output-dataset-one", dest="output_dataset_one", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parsed_args = vars(_parser.parse_args())

        _outputs = preprocess(**_parsed_args)
      - --uri
      - '{{$.inputs.parameters[''uri'']}}'
      - --some-int
      - '{{$.inputs.parameters[''some_int'']}}'
      - --output-parameter-one
      - '{{$.outputs.parameters[''output_parameter_one''].output_file}}'
      - --output-dataset-one
      - '{{$.outputs.artifacts[''output_dataset_one''].path}}'
      command: [/kfp-launcher/launch, --mlmd_server_address, $(METADATA_GRPC_SERVICE_HOST),
        --mlmd_server_port, $(METADATA_GRPC_SERVICE_PORT), --runtime_info_json, $(KFP_V2_RUNTIME_INFO),
        --container_image, $(KFP_V2_IMAGE), --task_name, preprocess, --pipeline_name,
        '{{inputs.parameters.pipeline-name}}', --pipeline_run_id, $(WORKFLOW_ID),
        --pipeline_task_id, $(KFP_POD_NAME), --pipeline_root, '{{inputs.parameters.pipeline-root}}',
        --, some_int=12, uri=uri-to-import, --]
      env:
      - name: KFP_POD_NAME
        valueFrom:
          fieldRef: {fieldPath: metadata.name}
      - name: KFP_NAMESPACE
        valueFrom:
          fieldRef: {fieldPath: metadata.namespace}
      - name: WORKFLOW_ID
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''workflows.argoproj.io/workflow'']'}
      - name: ENABLE_CACHING
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''pipelines.kubeflow.org/enable_caching'']'}
      - {name: KFP_V2_IMAGE, value: 'python:3.9'}
      - {name: KFP_V2_RUNTIME_INFO, value: '{"inputParameters": {"some_int": {"type":
          "INT"}, "uri": {"type": "STRING"}}, "inputArtifacts": {}, "outputParameters":
          {"output_parameter_one": {"type": "INT", "path": "/tmp/outputs/output_parameter_one/data"}},
          "outputArtifacts": {"output_dataset_one": {"schemaTitle": "system.Dataset",
          "instanceSchema": "", "metadataPath": "/tmp/outputs/output_dataset_one/data"}}}'}
      envFrom:
      - configMapRef: {name: metadata-grpc-configmap, optional: true}
      image: python:3.9
      volumeMounts:
      - {mountPath: /kfp-launcher, name: kfp-launcher}
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
    outputs:
      parameters:
      - name: preprocess-output_parameter_one
        valueFrom: {path: /tmp/outputs/output_parameter_one/data}
      artifacts:
      - {name: preprocess-output_dataset_one, path: /tmp/outputs/output_dataset_one/data}
      - {name: preprocess-output_parameter_one, path: /tmp/outputs/output_parameter_one/data}
    metadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/component_ref: '{}'
        pipelines.kubeflow.org/arguments.parameters: '{"some_int": "12", "uri": "uri-to-import"}'
      labels:
        pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
        pipelines.kubeflow.org/pipeline-sdk-type: kfp
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/enable_caching: "true"
    initContainers:
    - command: [/bin/mount_launcher.sh]
      image: gcr.io/ml-pipeline/kfp-launcher:1.6.4
      name: kfp-launcher
      mirrorVolumeMounts: true
    volumes:
    - {name: kfp-launcher}
  - name: train-op
    container:
      args:
      - sh
      - -ec
      - |
        program_path=$(mktemp)
        printf "%s" "$0" > "$program_path"
        python3 -u "$program_path" "$@"
      - |
        def _make_parent_dirs_and_return_path(file_path: str):
            import os
            os.makedirs(os.path.dirname(file_path), exist_ok=True)
            return file_path

        def train_op(
            dataset,
            model,
            num_steps = 100
        ):
            '''Dummy Training Step.'''

            with open(dataset, 'r') as input_file:
                input_string = input_file.read()
                with open(model, 'w') as output_file:
                    for i in range(num_steps):
                        output_file.write(
                            "Step {}\n{}\n=====\n".format(i, input_string)
                        )

        import argparse
        _parser = argparse.ArgumentParser(prog='Train op', description='Dummy Training Step.')
        _parser.add_argument("--dataset", dest="dataset", type=str, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--num-steps", dest="num_steps", type=int, required=False, default=argparse.SUPPRESS)
        _parser.add_argument("--model", dest="model", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parsed_args = vars(_parser.parse_args())

        _outputs = train_op(**_parsed_args)
      - --dataset
      - '{{$.inputs.artifacts[''dataset''].path}}'
      - --num-steps
      - '{{$.inputs.parameters[''num_steps'']}}'
      - --model
      - '{{$.outputs.artifacts[''model''].path}}'
      command: [/kfp-launcher/launch, --mlmd_server_address, $(METADATA_GRPC_SERVICE_HOST),
        --mlmd_server_port, $(METADATA_GRPC_SERVICE_PORT), --runtime_info_json, $(KFP_V2_RUNTIME_INFO),
        --container_image, $(KFP_V2_IMAGE), --task_name, train-op, --pipeline_name,
        '{{inputs.parameters.pipeline-name}}', --pipeline_run_id, $(WORKFLOW_ID),
        --pipeline_task_id, $(KFP_POD_NAME), --pipeline_root, '{{inputs.parameters.pipeline-root}}',
        --, 'num_steps={{inputs.parameters.preprocess-output_parameter_one}}', --]
      env:
      - name: KFP_POD_NAME
        valueFrom:
          fieldRef: {fieldPath: metadata.name}
      - name: KFP_NAMESPACE
        valueFrom:
          fieldRef: {fieldPath: metadata.namespace}
      - name: WORKFLOW_ID
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''workflows.argoproj.io/workflow'']'}
      - name: ENABLE_CACHING
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''pipelines.kubeflow.org/enable_caching'']'}
      - {name: KFP_V2_IMAGE, value: 'python:3.7'}
      - {name: KFP_V2_RUNTIME_INFO, value: '{"inputParameters": {"num_steps": {"type":
          "INT"}}, "inputArtifacts": {"dataset": {"metadataPath": "/tmp/inputs/dataset/data",
          "schemaTitle": "system.Dataset", "instanceSchema": ""}}, "outputParameters":
          {}, "outputArtifacts": {"model": {"schemaTitle": "system.Model", "instanceSchema":
          "", "metadataPath": "/tmp/outputs/model/data"}}}'}
      envFrom:
      - configMapRef: {name: metadata-grpc-configmap, optional: true}
      image: python:3.7
      volumeMounts:
      - {mountPath: /kfp-launcher, name: kfp-launcher}
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
      - {name: preprocess-output_parameter_one}
      artifacts:
      - {name: preprocess-output_dataset_one, path: /tmp/inputs/dataset/data}
    outputs:
      artifacts:
      - {name: train-op-model, path: /tmp/outputs/model/data}
    metadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/component_ref: '{}'
        pipelines.kubeflow.org/arguments.parameters: '{"num_steps": "{{inputs.parameters.preprocess-output_parameter_one}}"}'
      labels:
        pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
        pipelines.kubeflow.org/pipeline-sdk-type: kfp
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/enable_caching: "true"
    initContainers:
    - command: [/bin/mount_launcher.sh]
      image: gcr.io/ml-pipeline/kfp-launcher:1.6.4
      name: kfp-launcher
      mirrorVolumeMounts: true
    volumes:
    - {name: kfp-launcher}
  - name: two-step-pipeline
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
    dag:
      tasks:
      - name: preprocess
        template: preprocess
        arguments:
          parameters:
          - {name: pipeline-name, value: '{{inputs.parameters.pipeline-name}}'}
          - {name: pipeline-root, value: '{{inputs.parameters.pipeline-root}}'}
      - name: train-op
        template: train-op
        dependencies: [preprocess]
        arguments:
          parameters:
          - {name: pipeline-name, value: '{{inputs.parameters.pipeline-name}}'}
          - {name: pipeline-root, value: '{{inputs.parameters.pipeline-root}}'}
          - {name: preprocess-output_parameter_one, value: '{{tasks.preprocess.outputs.parameters.preprocess-output_parameter_one}}'}
          artifacts:
          - {name: preprocess-output_dataset_one, from: '{{tasks.preprocess.outputs.artifacts.preprocess-output_dataset_one}}'}
  arguments:
    parameters:
    - {name: pipeline-root, value: ''}
    - {name: pipeline-name, value: pipeline/two_step_pipeline}
  serviceAccountName: pipeline-runner
`

	complexPipeline = `
# Copyright 2018 The Kubeflow Authors
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

func TestCreatePipelineVersion(t *testing.T) {
	tt := []struct {
		msg            string
		template       string               // pipeline template
		version        *api.PipelineVersion // optional
		badObjectStore bool                 // optional, object requests always fail
		badDB          bool                 // optional, DB request always fail
		// The following are expected results.
		model *model.PipelineVersion // optional, expected version model when success
		// To verify an error, set the errorCode and
		// optionally set errorMsg and errorIs based on the test's needs.
		errorCode codes.Code
		errorMsg  string // error message
		errorIs   error  // verify a wrapped error is specific instance
	}{
		{
			msg:      "HappyCase",
			template: testWorkflow.ToStringForStore(),
			version: &api.PipelineVersion{
				Name:        "p_v",
				Description: "test",
			},
			model: &model.PipelineVersion{
				Name:        "p_v",
				Parameters:  "[{\"name\":\"param1\"}]",
				Description: "test",
			},
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			version: &api.PipelineVersion{
				Name: "complex",
			},
			model: &model.PipelineVersion{
				Name:       "complex",
				Parameters: "[{\"name\":\"output\"},{\"name\":\"project\"},{\"name\":\"schema\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json\"},{\"name\":\"train\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv\"},{\"name\":\"evaluation\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv\"},{\"name\":\"preprocess-mode\",\"value\":\"local\"},{\"name\":\"preprocess-module\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py\"},{\"name\":\"target\",\"value\":\"tips\"},{\"name\":\"learning-rate\",\"value\":\"0.1\"},{\"name\":\"hidden-layer-size\",\"value\":\"1500\"},{\"name\":\"steps\",\"value\":\"3000\"},{\"name\":\"workers\",\"value\":\"0\"},{\"name\":\"pss\",\"value\":\"0\"},{\"name\":\"predict-mode\",\"value\":\"local\"},{\"name\":\"analyze-mode\",\"value\":\"local\"},{\"name\":\"analyze-slice-column\",\"value\":\"trip_start_hour\"}]",
			},
		},
		{
			msg:            "BadObjectStore",
			badObjectStore: true,
			template:       testWorkflow.ToStringForStore(),
			errorCode:      codes.Internal,
			errorMsg:       "bad object store",
			// We previously verified that the failed pipeline version
			// in DB is in status PipelineVersionCreating by faking
			// the UUID generator, so that we know the created version
			// UUID in advance.
			// We cannot verify it using public APIs,
			// because the API does not expose them unless we know its UUID, but we
			// cannot know its UUID when create version request failed.
			// TODO: do we really need to verify this status? or should
			// the create version request return a UUID when the
			// pipeline version fails in PipelineVersionCreating state.
		},
		{
			msg:       "InvalidTemplate",
			template:  "I am invalid yaml",
			errorCode: codes.InvalidArgument,
			errorIs:   template.ErrorInvalidPipelineSpec,
		},
		{
			msg:       "BadDB",
			template:  testWorkflow.ToStringForStore(),
			badDB:     true,
			errorCode: codes.Internal,
			errorMsg:  "database is closed",
		},
		{
			msg:      "V2PipelineSpec",
			template: v2SpecHelloWorld,
			version: &api.PipelineVersion{
				Name: "v2spec",
			},
			model: &model.PipelineVersion{
				Name: "v2spec",
				// TODO(v2): when parameter extraction is implemented, this won't be empty.
				Parameters: "[]",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store)

			// Create a pipeline before versions.
			pipeline, err := manager.CreatePipeline(
				"my_pipeline",
				"",
				"",
				// Do not upload test.template here, because pipeline API is out of test scope.
				[]byte(testWorkflow.ToStringForStore()),
			)
			require.Nil(t, err)

			// Override bad dependencies after create pipeline request succeeds.
			if test.badObjectStore {
				manager.objectStore = &FakeBadObjectStore{}
			}
			if test.badDB {
				store.DB().Close()
			}
			// Create a version under the above pipeline.
			if test.version == nil {
				test.version = &api.PipelineVersion{Name: "my_pipeline_version_name"}
			}
			test.version.ResourceReferences = []*api.ResourceReference{{
				Key: &api.ResourceKey{
					Id:   pipeline.UUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			}}
			version, err := manager.CreatePipelineVersion(test.version,
				[]byte(test.template), true)
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				if test.errorIs != nil {
					assert.ErrorIs(t, err, test.errorIs)
				}
				return
			}
			require.Nil(t, err)

			version.UUID = ""
			test.model.PipelineId = pipeline.UUID
			test.model.Status = model.PipelineVersionReady
			test.model.CreatedAtInSec = 2
			assert.Equal(t, test.model, version)
		})
	}
}

func TestCreatePipelineOrVersion_V2PipelineName(t *testing.T) {
	tests := []struct {
		// inputs
		name      string
		namespace string
		template  string // template to upload
		// expected
		pipelineName string
	}{
		{name: "v2-compat", namespace: "", pipelineName: "pipeline/v2-compat"},
		{name: "pipe3", namespace: "", pipelineName: "pipeline/pipe3"},
		{name: "pipeline2", namespace: "kubeflow", pipelineName: "namespace/kubeflow/pipeline/pipeline2"},
		{name: "abcd", namespace: "user", pipelineName: "namespace/user/pipeline/abcd"},
		{name: "v2-spec1", namespace: "", template: v2SpecHelloWorld, pipelineName: "pipeline/v2-spec1"},
		{name: "v2-spec2", namespace: "user", template: v2SpecHelloWorld, pipelineName: "namespace/user/pipeline/v2-spec2"},
	}
	for _, test := range tests {
		testClone := test
		testClone.template = "" // template is too long for the message
		t.Run(fmt.Sprintf("%+v", testClone), func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store)

			if test.template == "" {
				test.template = strings.TrimSpace(v2compatPipeline)
			}

			// Verify v2 pipeline name of CreatePipeline template.
			createdPipeline, err := manager.CreatePipeline(test.name, "", test.namespace, []byte(test.template))
			require.Nil(t, err)
			bytes, err := manager.GetPipelineTemplate(createdPipeline.UUID)
			require.Nil(t, err)
			tmpl, err := template.New(bytes)
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())

			// Verify v2 pipeline name of CreatePipelineVersion template.
			version, err := manager.CreatePipelineVersion(
				&api.PipelineVersion{
					Name: "pipeline_version",
					ResourceReferences: []*api.ResourceReference{{
						Key: &api.ResourceKey{
							Id:   createdPipeline.UUID,
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					}},
				},
				[]byte(test.template), true)
			require.Nil(t, err)
			bytes, err = manager.GetPipelineVersionTemplate(version.UUID)
			require.Nil(t, err)
			tmpl, err = template.New(bytes)
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())
		})
	}
}

func TestDeletePipelineVersion(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create a pipeline.
	_, err := manager.CreatePipeline("pipeline", "", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Nil(t, err)

	// Create a version under the above pipeline.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(
		&api.PipelineVersion{
			Name: "pipeline_version",
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   DefaultFakeUUID,
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		},
		[]byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"), true)

	// Delete the above pipeline_version.
	err = manager.DeletePipelineVersion(FakeUUIDOne)
	assert.Nil(t, err)

	// Verify the version doesn't exist.
	_, err = manager.GetPipelineVersion(FakeUUIDOne)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineVersion_FileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create a pipeline.
	_, err := manager.CreatePipeline("pipeline", "", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Nil(t, err)

	// Create a version under the above pipeline.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(
		&api.PipelineVersion{
			Name: "pipeline_version",
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   DefaultFakeUUID,
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		},
		[]byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"), true)

	// Switch to a bad object store
	manager.objectStore = &FakeBadObjectStore{}

	// Delete the above pipeline_version.
	err = manager.DeletePipelineVersion(FakeUUIDOne)
	assert.NotNil(t, err)

	// Verify the version in deleting status.
	version, err := manager.pipelineStore.GetPipelineVersionWithStatus(FakeUUIDOne, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	assert.NotNil(t, version)
}

func TestCreateDefaultExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	experimentID, err := manager.CreateDefaultExperiment()
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateDefaultExperiment_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	experimentID, err := manager.CreateDefaultExperiment()
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateTask(t *testing.T) {
	_, manager, _, _, runDetail := initWithExperimentAndPipelineAndRun(t)
	task := &api.Task{
		Namespace:       "",
		PipelineName:    "pipeline/my-pipeline",
		RunId:           runDetail.UUID,
		MlmdExecutionID: "1",
		CreatedAt:       &timestamp.Timestamp{Seconds: 1462875553},
		FinishedAt:      &timestamp.Timestamp{Seconds: 1462875663},
		Fingerprint:     "123",
	}

	expectedTask := &model.Task{
		UUID:              DefaultFakeUUID,
		Namespace:         "",
		PipelineName:      "pipeline/my-pipeline",
		RunUUID:           runDetail.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1462875553,
		FinishedTimestamp: 1462875663,
		Fingerprint:       "123",
	}
	createdTask, err := manager.CreateTask(context.Background(), task)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, createdTask, "The CreateTask return has unexpected value.")

	// Verify the T in DB is in status PipelineVersionCreating.
	storedTask, err := manager.taskStore.GetTask(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, storedTask, "The StoredTask return has unexpected value.")
}

var v2SpecHelloWorld = `
{
  "components": {
    "comp-hello-world": {
      "executorLabel": "exec-hello-world",
      "inputDefinitions": {
	"parameters": {
	  "text": {
	    "type": "STRING"
	  }
	}
      }
    }
  },
  "deploymentSpec": {
    "executors": {
      "exec-hello-world": {
	"container": {
	  "args": [
	    "--text",
	    "{{$.inputs.parameters['text']}}"
	  ],
	  "command": [
	    "sh",
	    "-ec",
	    "program_path=$(mktemp)\nprintf \"%s\" \"$0\" > \"$program_path\"\npython3 -u \"$program_path\" \"$@\"\n",
	    "def hello_world(text):\n    print(text)\n    return text\n\nimport argparse\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\n_parser.add_argument(\"--text\", dest=\"text\", type=str, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = hello_world(**_parsed_args)\n"
	  ],
	  "image": "python:3.7"
	}
      }
    }
  },
  "pipelineInfo": {
    "name": "hello-world"
  },
  "root": {
    "dag": {
      "tasks": {
	"hello-world": {
	  "cachingOptions": {
	    "enableCache": true
	  },
	  "componentRef": {
	    "name": "comp-hello-world"
	  },
	  "inputs": {
	    "parameters": {
	      "text": {
		"componentInputParameter": "text"
	      }
	    }
	  },
	  "taskInfo": {
	    "name": "hello-world"
	  }
	}
      }
    },
    "inputDefinitions": {
      "parameters": {
	"text": {
	  "type": "STRING"
	}
      }
    }
  },
  "schemaVersion": "2.0.0",
  "sdkVersion": "kfp-1.6.5"
}
`
