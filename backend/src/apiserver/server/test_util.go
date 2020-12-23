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

package server

import (
	"context"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	invalidPipelineVersionId = "not_exist_pipeline_version"
)

var testWorkflow = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
	Spec:       v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}}},
})

var testWorkflow2 = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow2"},
	Spec:       v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}}},
})

var testWorkflowPatch = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow2"},
	Spec:       v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}, {Name: "param2"}}}},
})

var validReference = []*api.ResourceReference{
	{
		Key: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT, Id: resource.DefaultFakeUUID},
		Relationship: api.Relationship_OWNER,
	},
}

var validReferencesOfExperimentAndPipelineVersion = []*api.ResourceReference{
	{
		Key: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT,
			Id:   resource.DefaultFakeUUID,
		},
		Relationship: api.Relationship_OWNER,
	},
	{
		Key: &api.ResourceKey{
			Type: api.ResourceType_PIPELINE_VERSION,
			Id:   resource.DefaultFakeUUID,
		},
		Relationship: api.Relationship_CREATOR,
	},
}

var referencesOfExperimentAndInvalidPipelineVersion = []*api.ResourceReference{
	{
		Key: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT,
			Id:   resource.DefaultFakeUUID,
		},
		Relationship: api.Relationship_OWNER,
	},
	{
		Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: invalidPipelineVersionId},
		Relationship: api.Relationship_CREATOR,
	},
}

var referencesOfInvalidPipelineVersion = []*api.ResourceReference{
	{
		Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: invalidPipelineVersionId},
		Relationship: api.Relationship_CREATOR,
	},
}

// This automatically runs before all the tests.
func initEnvVars() {
	viper.Set(common.PodNamespace, "ns1")
}

func initWithExperiment(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	apiExperiment := &api.Experiment{Name: "exp1"}
	if common.IsMultiUserMode() {
		apiExperiment = &api.Experiment{
			Name: "exp1",
			ResourceReferences: []*api.ResourceReference{
				{
					Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
					Relationship: api.Relationship_OWNER,
				},
			},
		}
	}
	experiment, err := resourceManager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return clientManager, resourceManager, experiment
}

func initWithExperiment_SubjectAccessReview_Unauthorized(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager)
	apiExperiment := &api.Experiment{Name: "exp1"}
	if common.IsMultiUserMode() {
		apiExperiment = &api.Experiment{
			Name: "exp1",
			ResourceReferences: []*api.ResourceReference{
				{
					Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
					Relationship: api.Relationship_OWNER,
				},
			},
		}
	}
	experiment, err := resourceManager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return clientManager, resourceManager, experiment
}

func initWithExperimentAndPipelineVersion(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	// Create an experiment.
	apiExperiment := &api.Experiment{Name: "exp1"}
	experiment, err := resourceManager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)

	// Create a pipeline and then a pipeline version.
	_, err = resourceManager.CreatePipeline("pipeline", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Nil(t, err)
	_, err = resourceManager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "pipeline_version",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   resource.DefaultFakeUUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	},
		[]byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"), true)

	return clientManager, resourceManager, experiment
}

func initWithExperimentsAndTwoPipelineVersions(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	// Create an experiment.
	apiExperiment := &api.Experiment{Name: "exp1"}
	experiment, err := resourceManager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)

	// Create a pipeline and then a pipeline version.
	_, err = resourceManager.CreatePipeline("pipeline", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Nil(t, err)
	_, err = resourceManager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "pipeline_version",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   resource.DefaultFakeUUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	},
		[]byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"), true)

	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(resource.NonDefaultFakeUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	// Create another pipeline and then pipeline version.
	_, err = resourceManager.CreatePipeline("anpther-pipeline", "", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Nil(t, err)
	_, err =  resourceManager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "another_pipeline_version",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   resource.NonDefaultFakeUUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	},
		[]byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"), true)
	return clientManager, resourceManager, experiment
}

func initWithOneTimeRun(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.RunDetail) {
	clientManager, manager, exp := initWithExperiment(t)
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
	return clientManager, manager, runDetail
}

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Pipeline) {
	initEnvVars()
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(store)
	p, err := manager.CreatePipeline("p1", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, p
}

func AssertUserError(t *testing.T, err error, expectedCode codes.Code) {
	userError, ok := err.(*util.UserError)
	assert.True(t, ok)
	assert.Equal(t, expectedCode, userError.ExternalStatusCode())
}

func getPermissionDeniedError(ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	// Retrieve request details to compose the expected error
	userIdentity, _ := getUserIdentity(ctx)
	return util.NewPermissionDeniedError(
		errors.New("Unauthorized access"),
		"User '%s' is not authorized with reason: %s (request: %+v)",
		userIdentity,
		"this is not allowed",
		resourceAttributes,
	)
}

func wrapFailedAuthzApiResourcesError(err error) error {
	return util.Wrap(err, "Failed to authorize with API resource references")
}

func wrapFailedAuthzRequestError(err error) error {
	return util.Wrap(err, "Failed to authorize the request")
}
