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
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
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

func intPtr(i int64) *int64 {
	return &i
}

func strPtr(i string) *string {
	return &i
}

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
	return errors.New("Not implemented")
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

func (m *FakeBadObjectStore) GetSignedUrl(bucketConfig *objectstore.Config, secret *corev1.Secret, expirySeconds time.Duration, artifactURI string) (string, error) {
	return "", util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetObjectSize(bucketConfig *objectstore.Config, secret *corev1.Secret, artifactURI string) (int64, error) {
	return 0, util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func createPipelineV1(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:   name,
		Status: model.PipelineReady,
	}
}

func createPipeline(name string, description string, namespace string) *model.Pipeline {
	return &model.Pipeline{
		Name:        name,
		Description: description,
		Status:      model.PipelineReady,
		Namespace:   namespace,
	}
}

func createPipelineVersion(pipelineId string, name string, description string, url string, pipelineSpec string, pipelineSpecURI string, namespace string) *model.PipelineVersion {
	if namespace == "" {
		namespace = "default"
	}
	paramsJSON := "[{\"name\":\"param1\"}]"
	spec := pipelineSpec
	tmpl, err := template.New([]byte(pipelineSpec))
	if err != nil {
		spec = pipelineSpec
	} else {
		paramsJSON, _ = tmpl.ParametersJSON()
		spec = string(tmpl.Bytes())
	}
	return &model.PipelineVersion{
		Name:            name,
		Parameters:      paramsJSON,
		PipelineId:      pipelineId,
		CodeSourceUrl:   url,
		Description:     description,
		Status:          model.PipelineVersionReady,
		PipelineSpec:    spec,
		PipelineSpecURI: pipelineSpecURI,
	}
}

var testWorkflow = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
	Spec: v1alpha1.WorkflowSpec{
		Entrypoint: "testy",
		Templates: []v1alpha1.Template{{
			Name: "testy",
			Container: &corev1.Container{
				Image:   "docker/whalesay",
				Command: []string{"cowsay"},
				Args:    []string{"hello world"},
			},
		}},
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}},
	},
	Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
})

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Pipeline, *model.PipelineVersion) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	p1 := createPipeline("p1", "", "ns1")
	p, _ := manager.CreatePipeline(p1)
	pv1 := createPipelineVersion(
		p.UUID,
		"p1/v1",
		"v1",
		"url://namespaces/ns1/pipelines/p1/versions/v1",
		testWorkflow.ToStringForStore(),
		"uri://namespaces/ns1/pipelines/p1/versions/v1/p1v1.yaml",
		"ns1",
	)
	pv, err := manager.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	return store, manager, p, pv
}

func initWithExperiment(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1", Namespace: "ns1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return store, manager, experiment
}

func initWithExperimentAndPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.PipelineVersion) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	p1 := createPipeline("p1", "", "ns1")
	p, _ := manager.CreatePipeline(p1)
	pv1 := createPipelineVersion(
		p.UUID,
		"p1/v1",
		"v1",
		"url://namespaces/ns1/pipelines/p1/versions/v1",
		testWorkflow.ToStringForStore(),
		"uri://namespaces/ns1/pipelines/p1/versions/v1/p1v1.yaml",
		"ns1",
	)
	pv, err := manager.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	return store, manager, experiment, p, pv
}

func initWithExperimentAndPipelineAndRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.PipelineVersion, *model.Run) {
	store, manager, exp, pipeline, version := initWithExperimentAndPipeline(t)
	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: exp.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	run, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, exp, pipeline, version, run
}

// Util function to create an initial state with pipeline uploaded
func initWithJob(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ExperimentId: exp.UUID,
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

// Util function to create an initial state with pipeline uploaded
func initWithJobV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		ExperimentId: exp.UUID,
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

func initWithOneTimeRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeRunV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithPatchedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),

			Parameters: "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.Nodes = map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunCompressed(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
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
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunOffloaded(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.OffloadNodeStatusVersion = "offload-hash"
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

// Tests CreatePipeline and CreatePipelineVersion
func TestCreatePipeline(t *testing.T) {
	tt := []struct {
		msg            string
		name           string // optional
		description    string // optional
		template       string // pipeline template
		badObjectStore bool   // optional, object requests always fail
		badDB          bool   // optional, DB request always fail
		// The following are expected results.
		model        *model.Pipeline        // optional, expected pipeline model when success
		modelVersion *model.PipelineVersion // optional, expected pipeline model when success
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
			model:       createPipeline("p_v", "test", "user1"),
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			name:     "complex",
			model:    createPipeline("complex", "", "user1"),
		},
		{
			msg:            "BadObjectStore",
			badObjectStore: true,
			template:       testWorkflow.ToStringForStore(),
			errorCode:      codes.Internal,
			errorMsg:       "bad object store",
			model:          createPipeline("BadOS", "", "user1"),
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
			model:     createPipeline("InvalidYAML", "", "user1"),
			errorCode: codes.InvalidArgument,
			errorIs:   template.ErrorInvalidPipelineSpec,
		},
		{
			msg:       "BadDB",
			template:  testWorkflow.ToStringForStore(),
			badDB:     true,
			errorCode: codes.Internal,
			errorMsg:  "database is closed",
			model:     createPipeline("BadDB", "", "user1"),
		},
		{
			msg:      "V2PipelineSpec",
			template: v2SpecHelloWorld,
			name:     "v2spec",
			model:    createPipeline("v2 spec", "", "user1"),
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			var pipelineVersion, pv *model.PipelineVersion
			// setup
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
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
				test.model,
			)
			if err == nil {
				pv = createPipelineVersion(
					pipeline.UUID,
					pipeline.Name,
					pipeline.Description,
					fmt.Sprintf("url://%v", pipeline.Name),
					test.template,
					fmt.Sprintf("uri://pipelines/%v/versions/v1/spec.yaml", pipeline.Name),
					pipeline.Namespace,
				)
				pipelineVersion, err = manager.CreatePipelineVersion(
					pv,
				)
			}

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
			assert.Equal(t, test.model, pipeline)

			pv.UUID = pipelineVersion.UUID
			pv.PipelineId = pipelineVersion.PipelineId
			pv.CreatedAtInSec = 2
			pv.Status = "READY"
			pv.Parameters = pipelineVersion.Parameters
			assert.Equal(t, pv, pipelineVersion)
		})
	}
}

// Tests CreatePipelineVersion
func TestCreatePipelineVersion(t *testing.T) {
	initEnvVars()
	tt := []struct {
		msg            string
		template       string                 // pipeline template
		version        *model.PipelineVersion // optional
		badObjectStore bool                   // optional, object requests always fail
		badDB          bool                   // optional, DB request always fail
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
			version: &model.PipelineVersion{
				Name:        "p_v",
				Description: "test",
			},
			model: &model.PipelineVersion{
				Name:         "p_v",
				Parameters:   "[{\"name\":\"param1\"}]",
				Description:  "test",
				PipelineSpec: testWorkflow.ToStringForStore(),
			},
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			version: &model.PipelineVersion{
				Name: "complex",
			},
			model: &model.PipelineVersion{
				Name:         "complex",
				Parameters:   "[{\"name\":\"output\"},{\"name\":\"project\"},{\"name\":\"schema\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json\"},{\"name\":\"train\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv\"},{\"name\":\"evaluation\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv\"},{\"name\":\"preprocess-mode\",\"value\":\"local\"},{\"name\":\"preprocess-module\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py\"},{\"name\":\"target\",\"value\":\"tips\"},{\"name\":\"learning-rate\",\"value\":\"0.1\"},{\"name\":\"hidden-layer-size\",\"value\":\"1500\"},{\"name\":\"steps\",\"value\":\"3000\"},{\"name\":\"workers\",\"value\":\"0\"},{\"name\":\"pss\",\"value\":\"0\"},{\"name\":\"predict-mode\",\"value\":\"local\"},{\"name\":\"analyze-mode\",\"value\":\"local\"},{\"name\":\"analyze-slice-column\",\"value\":\"trip_start_hour\"}]",
				PipelineSpec: complexPipeline,
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
			version: &model.PipelineVersion{
				Name: "v2spec",
			},
			model: &model.PipelineVersion{
				Name: "v2spec",
				// TODO(v2): when parameter extraction is implemented, this won't be empty.
				Parameters:   "[{\"name\":\"param1\"}]",
				PipelineSpec: testWorkflow.ToStringForStore(),
			},
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

			// Create a pipeline before versions.
			p0 := createPipelineV1("my_pipeline")
			pv0 := createPipelineVersion(
				"",
				"my_pipeline",
				"",
				"",
				testWorkflow.ToStringForStore(),
				"",
				"",
			)
			pipeline, err := manager.CreatePipeline(p0)
			require.Nil(t, err)
			pv0.PipelineId = pipeline.UUID
			_, err = manager.CreatePipelineVersion(pv0)
			require.Nil(t, err)

			// Override bad dependencies after create pipeline request succeeds.
			if test.badObjectStore {
				manager.objectStore = &FakeBadObjectStore{}
			}
			if test.badDB {
				store.DB().Close()
			}
			// Create a version under the above pipeline.
			var pv *model.PipelineVersion
			if test.model == nil {
				pv = createPipelineVersion(
					pipeline.UUID,
					"my_pipeline_version_name",
					"",
					"",
					test.template,
					"",
					"",
				)
			} else {
				pv = test.model
				pv.PipelineId = pipeline.UUID
			}
			version, err := manager.CreatePipelineVersion(pv)
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
			test.model.CreatedAtInSec = 3
			test.model.PipelineSpec = version.PipelineSpec
			assert.Equal(t, test.model, version)
		})
	}
}

// Tests CreatePipelineVersion, GetPipelineVersionTemplate and GetPipelineLatestTemplate
func TestCreatePipelineOrVersion_V2PipelineName(t *testing.T) {
	initEnvVars()
	tests := []struct {
		// inputs
		name      string
		namespace string
		template  string // template to upload
		// expected
		pipelineName string
	}{
		{name: "v2-compat", namespace: "", pipelineName: "two-step-pipeline"},
		{name: "pipe3", namespace: "", pipelineName: "two-step-pipeline"},
		{name: "pipeline2", namespace: "kubeflow", pipelineName: "two-step-pipeline"},
		{name: "abcd", namespace: "user", pipelineName: "two-step-pipeline"},
		{name: "v2-spec1", namespace: "", template: v2SpecHelloWorld, pipelineName: "hello-world"},
		{name: "v2-spec2", namespace: "user", template: v2SpecHelloWorld, pipelineName: "hello-world"},
	}
	for _, test := range tests {
		testClone := test
		testClone.template = "" // template is too long for the message
		t.Run(fmt.Sprintf("%+v", testClone), func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

			if test.template == "" {
				test.template = strings.TrimSpace(v2compatPipeline)
			}

			// Verify v2 pipeline name of CreatePipeline template.
			p := createPipeline(
				test.name,
				"",
				test.namespace,
			)
			createdPipeline, err := manager.CreatePipeline(p)
			require.Nil(t, err)

			// Verify v2 pipeline name of CreatePipelineVersion template.
			pv := createPipelineVersion(
				createdPipeline.UUID,
				"pipeline_version",
				"",
				"",
				test.template,
				"",
				"",
			)
			if pv.PipelineSpec == "" {
				pv.PipelineSpec = v2compatPipeline
			}
			version, err := manager.CreatePipelineVersion(pv)
			require.Nil(t, err)
			bytes, err := manager.GetPipelineVersionTemplate(version.UUID)
			require.Nil(t, err)
			tmpl, err := template.New(bytes)
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())

			bytes, err = manager.GetPipelineLatestTemplate(createdPipeline.UUID)
			require.Nil(t, err)
			tmpl, err = template.New(bytes)
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())
		})
	}
}

func TestResourceManager_CreatePipelineAndPipelineVersion(t *testing.T) {
	tests := []struct {
		name         string
		p            *model.Pipeline
		pv           *model.PipelineVersion
		wantPipeline *model.Pipeline
		wantVersion  *model.PipelineVersion
		wantErr      bool
		errorMsg     string
	}{
		{
			"Valid - pipeline v2",
			&model.Pipeline{
				Name:        "pipeline v2",
				Description: "pipeline two",
				Namespace:   "user1",
			},
			&model.PipelineVersion{
				Name:            "pipeline v2 version 1",
				Description:     "pipeline v2 version description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 1,
				Name:           "pipeline v2",
				Description:    "pipeline two",
				Namespace:      "user1",
				Status:         model.PipelineReady,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  2,
				Name:            "pipeline v2 version 1",
				Description:     "pipeline v2 version description",
				PipelineId:      DefaultFakePipelineIdTwo,
				Status:          model.PipelineVersionReady,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
				Parameters:      "[]",
			},
			false,
			"",
		},
		{
			"Valid - pipeline v1",
			&model.Pipeline{
				Name:        "pipeline v1",
				Description: "pipeline one",
				Parameters:  `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
			},
			&model.PipelineVersion{
				Name:            "pipeline v1 version 1",
				Description:     "pipeline v1 version description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    complexPipeline,
				PipelineSpecURI: "pipeline_version_one.yaml",
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 1,
				Name:           "pipeline v1",
				Description:    "pipeline one",
				Parameters:     `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
				Status:         model.PipelineReady,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  2,
				PipelineId:      DefaultFakePipelineIdTwo,
				Name:            "pipeline v1 version 1",
				Description:     "pipeline v1 version description",
				Status:          model.PipelineVersionReady,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    complexPipeline,
				PipelineSpecURI: "pipeline_version_one.yaml",
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
			assert.True(t, ok)
			pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))

			gotPipeline, gotVersion, err := manager.CreatePipelineAndPipelineVersion(tt.p, tt.pv)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, gotPipeline)
				assert.Nil(t, gotVersion)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.wantPipeline, gotPipeline)
				tt.wantVersion.PipelineSpec = gotVersion.PipelineSpec
				tt.wantVersion.Parameters = gotVersion.Parameters
				assert.Equal(t, tt.wantVersion, gotVersion)
			}
		})
	}
}

// Tests GetPipelineByNameAndNamespace
func TestGetPipelineByNameAndNamespace(t *testing.T) {
	tt := []struct {
		msg          string
		pipelineName string
		namespace    string
		badDB        bool
		errorCode    codes.Code
		errMsg       string
	}{
		{
			msg:          "OK",
			pipelineName: "p1",
			namespace:    "ns1",
			errorCode:    codes.OK,
		},
		{
			msg:          "NotFount",
			pipelineName: "doesNotExists",
			namespace:    "ns1",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "SharedPipelineNotFound",
			pipelineName: "p1",
			namespace:    "wrongNamespace",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "BadDB",
			pipelineName: "p1",
			namespace:    "ns1",
			badDB:        true,
			errorCode:    codes.Internal,
			errMsg:       "database is closed",
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store, manager, p, _ := initWithPipeline(t)
			if test.badDB {
				store.Close()
			}

			result, err := manager.GetPipelineByNameAndNamespace(
				test.pipelineName,
				test.namespace,
			)

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
				return
			}
			require.Nil(t, err)
			assert.Equal(t, result, p)
		})
	}
}

// Tests GetPipelineByNameAndNamespaceV1
func TestGetPipelineByNameAndNamespaceV1(t *testing.T) {
	tt := []struct {
		msg          string
		pipelineName string
		namespace    string
		badDB        bool
		errorCode    codes.Code
		errMsg       string
	}{
		{
			msg:          "OK",
			pipelineName: "p1",
			namespace:    "ns1",
			errorCode:    codes.OK,
		},
		{
			msg:          "NotFount",
			pipelineName: "doesNotExists",
			namespace:    "ns1",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "SharedPipelineNotFound",
			pipelineName: "p1",
			namespace:    "wrongNamespace",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "BadDB",
			pipelineName: "p1",
			namespace:    "ns1",
			badDB:        true,
			errorCode:    codes.Internal,
			errMsg:       "database is closed",
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store, manager, p, pv := initWithPipeline(t)
			if test.badDB {
				store.Close()
			}

			resp, respv, err := manager.GetPipelineByNameAndNamespaceV1(
				test.pipelineName,
				test.namespace,
			)

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
				return
			}
			require.Nil(t, err)
			assert.Equal(t, p, resp)
			assert.Equal(t, pv, respv)
		})
	}
}

// Tests GetPipelineLatestTemplate (from PipelineSpec)
func TestGetLatestPipelineVersion(t *testing.T) {
	store, manager, p, pv := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetLatestPipelineVersion(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, pv, actualTemplate)

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	assert.True(t, ok)
	pv2 := createPipelineVersion(
		p.UUID,
		"new version",
		"new version desc",
		"url://pipelines/p1/versions/v2",
		testWorkflow.ToStringForStore(),
		"uri://pipelines/p1/versions/v2/spec.yaml",
		p.Namespace,
	)
	pv2expected, _ := manager.CreatePipelineVersion(pv2)
	pv2.UUID = pv2expected.UUID
	pv2.CreatedAtInSec = pv2expected.CreatedAtInSec
	pv2.Status = model.PipelineVersionReady
	actualTemplate2, err := manager.GetLatestPipelineVersion(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, pv2, actualTemplate2)
}

// Tests GetPipelineLatestTemplate (from PipelineSpec)
func TestGetPipelineTemplate(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, []byte(testWorkflow.ToStringForStore()), actualTemplate)
}

// Tests GetPipelineLatestTemplate (from PipelineSpecURI)
func TestGetPipelineTemplate_FromPipelineURI(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	manager.objectStore.AddFile([]byte(testWorkflow.ToStringForStore()), p.UUID)
	pv := &model.PipelineVersion{
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: p.UUID,
	}
	_, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (from PipelineVersionId)
func TestGetPipelineTemplate_FromPipelineVersionId(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	pv := &model.PipelineVersion{
		UUID:            "1000",
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: p.UUID,
	}

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	assert.True(t, ok)

	manager.objectStore.AddFile([]byte(testWorkflow.ToStringForStore()), manager.objectStore.GetPipelineKey("1000"))
	pv2, _ := manager.CreatePipelineVersion(pv)
	assert.NotEqual(t, p.UUID, pv2.UUID)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (from PipelineId)
func TestGetPipelineTemplate_FromPipelineId(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	pv := &model.PipelineVersion{
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: p.UUID,
	}

	manager.objectStore.AddFile([]byte(testWorkflow.ToStringForStore()), manager.objectStore.GetPipelineKey(p.UUID))

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv2, _ := manager.CreatePipelineVersion(pv)
	assert.NotEqual(t, p.UUID, pv2.UUID)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (NotFound)
func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.objectStore.AddFile(template, store.objectStore.GetPipelineKey(fmt.Sprint(1)))
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	_, err := manager.GetPipelineLatestTemplate("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

// Tests GetPipelineLatestTemplate (pipelineSpec NotFound)
func TestGetPipelineTemplate_PipelineFileNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipelineV1("pipeline1"))
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	_, err := manager.GetPipelineLatestTemplate(pipeline.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// Tests ListPipelines
func TestListPipelines(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	assert.Nil(t, err)

	_, nTotal, _, err := manager.ListPipelines(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelines(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests ListPipelinesV1
func TestListPipelinesV1(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	assert.Nil(t, err)

	_, _, nTotal, _, err := manager.ListPipelinesV1(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID)
	assert.Nil(t, err)

	_, _, nTotal, _, err = manager.ListPipelinesV1(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests ListPipelineVersions
func TestListPipelineVersions(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew1.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "", nil)
	assert.Nil(t, err)

	_, nTotal, _, err := manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete a pipeline version
	err = manager.DeletePipelineVersion(FakeUUIDOne)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests UpdatePipelineStatus
func TestUpdatePipelineStatus(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)
	p2 := createPipelineV1(
		"pipeline2",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew2.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	p1retrieved, err := manager.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineReady, p1retrieved.Status)

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineCreating)
	assert.Nil(t, err)
	_, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineDeleting)
	assert.Nil(t, err)
	_, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineReady)
	assert.Nil(t, err)
	p1retrieved, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineReady, p1retrieved.Status)
}

// Tests UpdatePipelineVersionStatus
func TestUpdatePipelineVersionStatus(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)
	p2 := createPipelineV1(
		"pipeline2",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew2.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	p1retrieved, err := manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineVersionReady, p1retrieved.Status)

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionCreating)
	assert.Nil(t, err)
	_, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	_, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionReady)
	assert.Nil(t, err)
	p1retrieved, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineVersionReady, p1retrieved.Status)
}

func TestDeletePipelineVersion(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p := createPipelineV1(
		"pipeline",
	)
	pnew, err := manager.CreatePipeline(p)
	assert.Nil(t, err)
	// Create a version under the above pipeline.
	pv := createPipelineVersion(
		pnew.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a version under the above pipeline.
	pv2 := createPipelineVersion(
		pnew.UUID,
		"pipeline_version",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pnew2, err := manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	// Delete the above pipeline_version.
	err = manager.DeletePipelineVersion(pnew2.UUID)
	assert.Nil(t, err)

	// Verify the version doesn't exist.
	_, err = manager.GetPipelineVersion(FakeUUIDOne)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Verify the first version exists.
	_, err = manager.GetPipelineVersion(DefaultFakeUUID)
	assert.Nil(t, err)

	// Verify the latest version
	pvLatestTeplate, err := manager.GetPipelineLatestTemplate(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"creationTimestamp\":null},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}", string(pvLatestTeplate))
}

// Tests DeletePipelineVersion (NotFound)
func TestDeletePipelineVersion_FileError(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p := createPipelineV1(
		"pipeline",
	)
	pnew, err := manager.CreatePipeline(p)
	assert.Nil(t, err)
	// Create a version under the above pipeline.
	pv := createPipelineVersion(
		pnew.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager.CreatePipelineVersion(pv)

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

// Tests DeletePipeline
func TestDeletePipeline(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID)
	assert.Nil(t, err)

	// Verify the pipeline doesn't exist.
	_, err = manager.GetPipeline(FakeUUIDOne)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Verify the first pipeline exists.
	_, err = manager.GetPipeline(DefaultFakeUUID)
	assert.Nil(t, err)

	// Must fail due to active pipeline versions
	err = manager.DeletePipeline(pnew1.UUID)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("as it has existing pipeline versions (e.g. %v)", FakeUUIDOne))
}

// TODO: use table driven test to test CreateRun api
func TestCreateRun_ThroughPipelineID(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(p.UUID, "version_for_run", "", "", testWorkflow.ToStringForStore(), "", "")
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.ObjectMeta.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}
	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId:    version.UUID,
			PipelineId:           p.UUID,
			PipelineName:         "version_for_run",
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRunV2(t)
	expectedExperimentUUID := runDetail.ExperimentId

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "hello-world-0",
		ServiceAccount: "pipeline-runner",
		Namespace:      runDetail.Namespace,
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	expectedRunDetail.PipelineSpec.PipelineSpecManifest = runDetail.PipelineSpec.PipelineSpecManifest
	expectedRunDetail.RunDetails.PipelineRuntimeManifest = runDetail.RunDetails.PipelineRuntimeManifest
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpec(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	expectedExperimentUUID := runDetail.ExperimentId
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			State:            "PENDING",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecWithPatch(t *testing.T) {
	viper.Set(common.HasDefaultBucketEnvVar, "true")
	viper.Set(common.ProjectIDEnvVar, "test-project-id")
	viper.Set(common.DefaultBucketNameEnvVar, "test-default-bucket")
	store, manager, runDetail := initWithPatchedRun(t)
	expectedExperimentUUID := runDetail.ExperimentId
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("test-default-bucket")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
		},
	}
	expectedRunDetail.PipelineSpec.PipelineName = runDetail.PipelineSpec.PipelineName
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecSameManifest(t *testing.T) {
	viper.Set(common.HasDefaultBucketEnvVar, "true")
	viper.Set(common.ProjectIDEnvVar, "test-project-id")
	viper.Set(common.DefaultBucketNameEnvVar, "test-default-bucket")
	_, manager, runDetail := initWithPatchedRun(t)

	manager.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore, _ := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))

	newRun, err := manager.CreateRun(
		context.Background(),
		&model.Run{
			DisplayName: "run1",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: testWorkflow.ToStringForStore(),
				Parameters:           "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
			},
			ExperimentId: runDetail.ExperimentId,
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, "run1", newRun.DisplayName)
	assert.Empty(t, newRun.PipelineId)
	assert.Empty(t, newRun.PipelineVersionId)
	assert.NotEqual(t, runDetail.WorkflowRuntimeManifest, newRun.WorkflowRuntimeManifest)
	assert.Equal(t, runDetail.WorkflowSpecManifest, newRun.WorkflowSpecManifest)
	assert.Empty(t, newRun.PipelineSpecManifest)
}

func TestCreateRun_ThroughPipelineVersion(t *testing.T) {
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_run",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
			PipelineVersionId: version.UUID,
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "sa1",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId:    version.UUID,
			PipelineId:           version.PipelineId,
			PipelineName:         version.Name,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_run",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:        pipeline.UUID,
			PipelineVersionId: version.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "sa1",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipeline.UUID,
			PipelineVersionId:    version.UUID,
			PipelineName:         version.Name,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch a template with an empty pipeline spec manifest")
}

func TestCreateRun_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: string("I am invalid"),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRun_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "null", // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRun_OverrideParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unrecognized input parameter")
}

func TestCreateRun_CreateWorkflowError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a workflow")
}

func TestCreateRun_StoreRunMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	store.DB().Close()
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
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
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.DeleteRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_CrdFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	// assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	// assert.Contains(t, err.Error(), "some error")
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
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Experiment id cannot be equal to the default id")
}

func TestDeleteExperiment_ExperimentNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.DeleteExperiment("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_CrdFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
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

	isTerminated, err := store.ExecClientFake.IsTerminated(runDetail.K8SName)
	assert.Nil(t, err)
	assert.True(t, isTerminated)
}

func TestTerminateRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
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
	assert.Equal(t, actualRunDetail.RunDetails.State, model.RuntimeStateRunning)
}

func TestRetryRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
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

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error updating and creating a workflow")
}

func TestUnarchiveRun_OK(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.UnarchiveRun(runDetail.UUID)
	assert.Nil(t, err)
}

func TestUnarchiveRun_Failed_ExperimentArchived(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.ArchiveExperiment(context.Background(), runDetail.ExperimentId)
	assert.Nil(t, err)
	err = manager.UnarchiveRun(runDetail.UUID)
	assert.NotNil(t, err)
	assert.Equal(t, codes.FailedPrecondition, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unarchive the experiment first to allow")
}

func TestUnarchiveRun_Failed_ResourceNotFound(t *testing.T) {
	store, manager, _ := initWithExperiment(t)
	defer store.Close()
	err := manager.UnarchiveRun(FakeUUIDOne)
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// TODO Use table driven to write UT to test CreateJob
func TestCreateJob_ThroughWorkflowSpec(t *testing.T) {
	store, _, job := initWithJob(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		ExperimentId:   DefaultFakeUUID,
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
	}
	expectedJob.PipelineSpec.PipelineName = job.PipelineSpec.PipelineName
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
}

func TestCreateJob_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		ExperimentId:   DefaultFakeUUID,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
	}
	expectedJob.PipelineSpec.PipelineName = job.PipelineSpec.PipelineName
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
	fetchedJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), fetchedJob.ToV1(), "CreateJob stored invalid data in database")
}

func TestCreateJob_ThroughPipelineID(t *testing.T) {
	store, manager, pipeline, _ := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, _ := manager.CreateExperiment(apiExperiment)
	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_job",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 5,
		UpdatedAtInSec: 5,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipeline.UUID,
			PipelineName:         version.Name,
			PipelineVersionId:    version.UUID,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_ThroughPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_job",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId: version.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 5,
		UpdatedAtInSec: 5,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           version.PipelineId,
			PipelineName:         version.Name,
			PipelineVersionId:    version.UUID,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_job",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,

		PipelineSpec: model.PipelineSpec{
			PipelineId:        pipeline.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
			PipelineVersionId: version.UUID,
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 5,
		UpdatedAtInSec: 5,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   experiment.UUID,

		PipelineSpec: model.PipelineSpec{
			PipelineName:         version.Name,
			PipelineId:           pipeline.UUID,
			PipelineVersionId:    version.UUID,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		DisplayName:  "pp 1",
		Enabled:      true,
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			Parameters: "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch a template with an empty pipeline spec manifest")
}

func TestCreateJob_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: string("I am invalid"),
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateJob_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: string("null"), // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateJob_ExtraInputParameterError(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unrecognized input parameter: param2")
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{PipelineId: p.UUID},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a recurring run during scheduling a workflow")
}

func TestEnableJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.ChangeJobMode(context.Background(), job.UUID, false)
	job, err := manager.GetJob(job.UUID)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   DefaultFakeUUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           job.PipelineSpec.PipelineId,
			PipelineName:         job.PipelineSpec.PipelineName,
			PipelineVersionId:    job.PipelineSpec.PipelineVersionId,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
}

func TestEnableJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.ChangeJobMode(context.Background(), "1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.ChangeJobMode(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
}

func TestEnableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})
	// When swf CR is missing, enabling the job needs to fail.
	err := manager.ChangeJobMode(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
	assert.Contains(t, err.Error(), "not found")
}

func TestDisableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	require.Equal(t, job.Enabled, true)

	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})
	err := manager.ChangeJobMode(context.Background(), job.UUID, false)
	require.Nil(t, err, "Disabling the job should succeed even when the custom resource is missing")
	job, err = manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.Equal(t, job.Enabled, false)
}

func TestEnableJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.DB().Close()
	err := manager.ChangeJobMode(context.Background(), job.UUID, false)
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
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
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
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
}

func TestDeleteJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})

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
	expectedExperimentUUID := run.ExperimentId
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)
	run, err = manager.GetRun(run.UUID)
	assert.Nil(t, err)
	expectedRun := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Running",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
				{
					UpdateTimeInSec: 4,
					State:           model.RuntimeStateRunning,
				},
			},
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	expectedRun.PipelineSpec.PipelineName = run.PipelineSpec.PipelineName
	expectedRun.RunDetails.WorkflowRuntimeManifest = run.RunDetails.WorkflowRuntimeManifest
	assert.Equal(t, expectedRun.ToV1(), run.ToV1())
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.Run{
		UUID:           "WORKFLOW_1",
		ExperimentId:   job.ExperimentId,
		DisplayName:    "MY_NAME",
		StorageState:   model.StorageStateAvailable,
		K8SName:        "MY_NAME",
		Namespace:      job.Namespace,
		RecurringRunId: job.UUID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: workflow.GetExecutionSpec().ToStringForStore(),
			PipelineSpecManifest: job.PipelineSpec.PipelineSpecManifest,
			PipelineId:           job.PipelineSpec.PipelineId,
			PipelineName:         job.PipelineSpec.PipelineName,
			PipelineVersionId:    job.PipelineSpec.PipelineVersionId,
		},
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
			CreatedAtInSec:          11,
			ScheduledAtInSec:        11,
			FinishedAtInSec:         0,
			Conditions:              "Error",
			State:                   model.RuntimeStateUnspecified,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStateUnspecified,
				},
			},
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1())
}

func TestReportWorkflowResource_WorkflowMissingRunID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: run.K8SName,
		},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow[workflow-name] missing the Run ID label")
}

func TestReportWorkflowResource_RunNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "obsolete",
			Namespace: "kubeflow",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-id-not-exist"},
		},
	})
	store.ExecClient().Execution("kubeflow").Create(ctx, workflow, v1.CreateOptions{})
	_, err := manager.ReportWorkflowResource(ctx, workflow)
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
			Name:      run.K8SName,
			Namespace: namespace,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	wf, err := store.ExecClientFake.Execution(namespace).Get(context.Background(), run.K8SName, v1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowPersistedFinalState], "true")
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
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
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NotNil(t, err)
	assert.Equalf(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), "Expected not found error, but got %s", err.Error())
	assert.Contains(t, err.Error(), "Failed to delete the completed workflow")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_DeleteFailed(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
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
		K8SName:        "MY_NAME",
		DisplayName:    "j1",
		Namespace:      actualJob.Namespace,
		ExperimentId:   actualJob.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
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
			PipelineSpecManifest: actualJob.PipelineSpec.PipelineSpecManifest,
			PipelineName:         actualJob.PipelineSpec.PipelineName,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReportScheduledWorkflowResource_Success_withParamsV1(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(job.UUID),
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{
						Name:  "param_v1",
						Value: "value_v1",
					},
				},
			},
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		K8SName:        "MY_NAME",
		DisplayName:    "j1",
		Namespace:      actualJob.Namespace,
		ExperimentId:   actualJob.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			Parameters:           `[{"name":"param_v1","value":"value_v1"}]`,
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
			PipelineSpecManifest: actualJob.PipelineSpec.PipelineSpecManifest,
			PipelineName:         actualJob.PipelineSpec.PipelineName,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReportScheduledWorkflowResource_Success_withRuntimeParamsV2(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "updated_name",
			Namespace: "ns1",
			UID:       types.UID(job.UUID),
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{
						Name:  "param1",
						Value: "\"world-updated\"",
					},
				},
			},
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		K8SName:        "updated_name",
		DisplayName:    "j1",
		Namespace:      "ns1",
		ExperimentId:   job.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: v2SpecHelloWorld,
			PipelineName:         actualJob.PipelineSpec.PipelineName,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `{"param1":"world-updated"}`,
				PipelineRoot: "job-1-root",
			},
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReconcileSwfCrs(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()

	fetchedJob, err := manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.NotNil(t, fetchedJob)

	swfClient := store.SwfClient().ScheduledWorkflow("ns1")

	options := v1.GetOptions{}
	ctx := context.Background()

	swf, err := swfClient.Get(ctx, "job-", options)
	require.Nil(t, err)

	// emulates an invalid/outdated spec
	swf.Spec.Workflow.Spec = nil
	swf, err = swfClient.Update(ctx, swf)
	require.Nil(t, swf.Spec.Workflow.Spec)

	err = manager.ReconcileSwfCrs(ctx)
	require.Nil(t, err)

	swf, err = swfClient.Get(ctx, "job-", options)
	require.Nil(t, err)
	require.NotNil(t, swf.Spec.Workflow.Spec)
}

func TestReportScheduledWorkflowResource_Error(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.CreateDefaultExperiment("")
	// Create pipeline
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
	})
	p := createPipelineV1("1")
	pipeline, err := manager.CreatePipeline(p)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pipeline.UUID,
		"1",
		"",
		"",
		workflow.ToStringForStore(),
		"",
		pipeline.Namespace,
	)
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	// Create job
	job := &model.Job{
		K8SName:      "pp1",
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID},
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

func TestReadArtifact_Succeed(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	expectedContent := "test"
	filePath := "test/file.txt"
	store.ObjectStore().AddFile([]byte(expectedContent), filePath)

	// Create a scheduled run
	// job, _ := manager.CreateJob(model.Job{
	// 	Name:       "pp1",
	// 	PipelineId: p.UUID,
	// 	Enabled:    true,
	// })
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
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
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
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
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	_, err = manager.ReadArtifact("run-1", "node-1", "artifact-1")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReadArtifact_NoRun_NotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

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
    - {name: pipeline-name, value: two-step-pipeline}
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

func TestCreateDefaultExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experimentID, err := manager.CreateDefaultExperiment("")
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateDefaultExperiment_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experimentID, err := manager.CreateDefaultExperiment("multi-user")
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "multi-user",
		StorageState:   "AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateTask(t *testing.T) {
	_, manager, _, _, _, runDetail := initWithExperimentAndPipelineAndRun(t)
	task := &model.Task{
		Namespace:         "",
		PipelineName:      "pipeline/my-pipeline",
		RunId:             runDetail.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1462875553,
		FinishedTimestamp: 1462875663,
		Fingerprint:       "123",
	}

	expectedTask := &model.Task{
		UUID:              DefaultFakeUUID,
		PipelineName:      "pipeline/my-pipeline",
		RunId:             runDetail.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1462875553,
		FinishedTimestamp: 1462875663,
		Fingerprint:       "123",
	}
	createdTask, err := manager.CreateTask(task)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, createdTask, "The CreateTask return has unexpected value")

	// Verify the T in DB is in status PipelineVersionCreating.
	storedTask, err := manager.taskStore.GetTask(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, storedTask, "The StoredTask return has unexpected value")
}

func TestBackwardsCompatibilityForSessionInfo(t *testing.T) {
	_, manager, _, _, _, _ := initWithExperimentAndPipelineAndRun(t)

	// First Artifact has assigned a bucket_session_info
	artifact1 := &ml_metadata.Artifact{
		Id:  intPtr(0),
		Uri: strPtr("s3://test-bucket/pipeline/some-pipeline-id/task/key0"),
	}

	config1, _, err := manager.GetArtifactSessionInfo(context.Background(), artifact1)

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, config1)

	// Second Artifact has assigned a store_session_info
	artifact2 := &ml_metadata.Artifact{
		Id:  intPtr(1),
		Uri: strPtr("s3://test-bucket/pipeline/some-pipeline-id/task/key1"),
	}

	// Call the function
	config2, _, err := manager.GetArtifactSessionInfo(context.Background(), artifact2)

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, config2)
}

var v2SpecHelloWorld = `
components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        text:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - "--text"
        - "{{$.inputs.parameters['text']}}"
        command:
        - sh
        - "-ec"
        - |
          program_path=$(mktemp)
          printf "%s" "$0" > "$program_path"
          python3 -u "$program_path" "$@"
        - |
          def hello_world(text):
              print(text)
              return text

          import argparse
          _parser = argparse.ArgumentParser(prog='Hello world', description='')
          _parser.add_argument("--text", dest="text", type=str, required=True, default=argparse.SUPPRESS)
          _parsed_args = vars(_parser.parse_args())

          _outputs = hello_world(**_parsed_args)
        image: python:3.7
pipelineInfo:
  name: hello-world
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            text:
              componentInputParameter: text
        taskInfo:
          name: hello-world
  inputDefinitions:
    parameters:
      text:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

var v2SpecHelloWorldMutated = `
components:
  comp-hello-world:
    executorLabel: exec-hello-world
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        image: python:3.7
pipelineInfo:
  name: pipelines/p1/versions/v1
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        taskInfo:
          name: hello-world
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`
