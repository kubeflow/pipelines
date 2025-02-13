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

package server

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	invalidPipelineVersionId = "not_exist_pipeline_version"
	DefaultFakeUUID          = "123e4567-e89b-12d3-a456-426655440000"
	NonDefaultFakeUUID       = "123e4567-e89b-12d3-a456-426655441000"
	FakeUUIDOne              = "123e4567-e89b-12d3-a456-426655440001"
	DefaultFakeIdOne         = "123e4567-e89b-12d3-a456-426655440000"
	DefaultFakeIdTwo         = "123e4567-e89b-12d3-a456-426655440001"
	DefaultFakeIdThree       = "123e4567-e89b-12d3-a456-426655440002"
	DefaultFakeIdFour        = "123e4567-e89b-12d3-a456-426655440003"
	DefaultFakeIdFive        = "123e4567-e89b-12d3-a456-426655440004"
	DefaultFakeIdSix         = "123e4567-e89b-12d3-a456-426655440005"
	DefaultFakeIdSeven       = "123e4567-e89b-12d3-a456-426655440006"
	DefaultFakeIdEight       = "123e4567-e89b-12d3-a456-426655440007"
)

var testWorkflowPatch = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow2"},
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
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}, {Name: "param2"}}},
	},
})

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

var testWorkflow2 = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name2", UID: "workflow2", Namespace: "ns2"},
	Spec: v1alpha1.WorkflowSpec{
		Entrypoint: "testy2",
		Templates: []v1alpha1.Template{{
			Name: "testy2",
			Container: &corev1.Container{
				Image:   "docker/whalesay2",
				Command: []string{"cowsay2"},
				Args:    []string{"hello world2"},
			},
		}},
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}},
	},
	Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
})

var validReference = []*apiv1beta1.ResourceReference{
	{
		Key: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT, Id: DefaultFakeUUID,
		},
		Relationship: apiv1beta1.Relationship_OWNER,
	},
}

var validReferencesOfExperimentAndPipelineVersion = []*apiv1beta1.ResourceReference{
	{
		Key: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   DefaultFakeUUID,
		},
		Relationship: apiv1beta1.Relationship_OWNER,
	},
	{
		Key: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_PIPELINE_VERSION,
			Id:   DefaultFakeUUID,
		},
		Relationship: apiv1beta1.Relationship_CREATOR,
	},
}

var referencesOfInvalidPipelineVersion = []*apiv1beta1.ResourceReference{
	{
		Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: invalidPipelineVersionId},
		Relationship: apiv1beta1.Relationship_CREATOR,
	},
}

// This automatically runs before all the tests.
func initEnvVars() {
	viper.Set(common.PodNamespace, "ns1")
}

func initWithExperiment(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	var apiExperiment *apiv1beta1.Experiment
	if common.IsMultiUserMode() {
		apiExperiment = &apiv1beta1.Experiment{
			Name: "exp1",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		}
	} else {
		apiExperiment = &apiv1beta1.Experiment{
			Name: "exp1",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		}
	}
	modelExperiment, err := toModelExperiment(apiExperiment)
	assert.Nil(t, err)
	experiment, err := resourceManager.CreateExperiment(modelExperiment)
	assert.Nil(t, err)
	return clientManager, resourceManager, experiment
}

func initWithExperiment_SubjectAccessReview_Unauthorized(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &apiv1beta1.Experiment{Name: "exp1"}
	if common.IsMultiUserMode() {
		apiExperiment = &apiv1beta1.Experiment{
			Name: "exp1",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		}
	}
	modelExperiment, err := toModelExperiment(apiExperiment)
	modelExperiment.Namespace = resourceManager.ReplaceNamespace(modelExperiment.Namespace)
	assert.Nil(t, err)
	experiment, err := resourceManager.CreateExperiment(modelExperiment)
	assert.Nil(t, err)
	return clientManager, resourceManager, experiment
}

func initWithExperimentAndPipelineVersion(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Experiment, *model.PipelineVersion) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	// Create an experiment.
	apiExperiment := &apiv1beta1.Experiment{Name: "exp1"}
	modelExperiment, err := toModelExperiment(apiExperiment)
	assert.Nil(t, err)
	experiment, err := resourceManager.CreateExperiment(modelExperiment)
	assert.Nil(t, err)

	// Create a pipeline and then a pipeline version.
	p, err := resourceManager.CreatePipeline(
		&model.Pipeline{
			Name:        "p1",
			Description: "",
			Namespace:   "",
		},
	)
	assert.Nil(t, err)
	pv, err := resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:         "p1",
			PipelineId:   p.UUID,
			PipelineSpec: testWorkflow.ToStringForStore(),
		},
	)
	assert.Nil(t, err)
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(NonDefaultFakeUUID, nil))
	resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version",
			PipelineId: DefaultFakeUUID,
		},
	)
	return clientManager, resourceManager, experiment, pv
}

func initWithExperimentsAndTwoPipelineVersions(t *testing.T) *resource.FakeClientManager {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	// Create an experiment.
	apiExperiment := &apiv1beta1.Experiment{Name: "exp1"}
	modelExperiment, err := toModelExperiment(apiExperiment)
	assert.Nil(t, err)
	_, err = resourceManager.CreateExperiment(modelExperiment)
	assert.Nil(t, err)

	// Create a pipeline and then a pipeline version.
	p, err := resourceManager.CreatePipeline(
		&model.Pipeline{
			Name:        "pipeline",
			Description: "",
			Namespace:   "",
		},
	)
	assert.Nil(t, err)
	_, err = resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:         "pipeline",
			PipelineId:   p.UUID,
			PipelineSpec: "apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		},
	)
	assert.Nil(t, err)
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655441001", nil))
	resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	_, err = resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version",
			PipelineId: DefaultFakeUUID,
		},
	)
	assert.Nil(t, err)
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(NonDefaultFakeUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	// Create another pipeline and then pipeline version.
	p1, err := resourceManager.CreatePipeline(
		&model.Pipeline{
			Name:        "anpther-pipeline",
			Description: "",
			Namespace:   "",
		},
	)

	assert.Nil(t, err)
	_, err = resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:         "anpther-pipeline",
			PipelineId:   p1.UUID,
			Description:  "",
			PipelineSpec: "apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		},
	)
	assert.Nil(t, err)

	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655441002", nil))
	resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	_, err = resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "another_pipeline_version",
			PipelineId: NonDefaultFakeUUID,
		},
	)
	assert.Nil(t, err)
	return clientManager
}

func initWithOneTimeRun(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Run) {
	clientManager, manager, exp := initWithExperiment(t)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}
	apiRun := &apiv1beta1.Run{
		Name: "run1",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: exp.Namespace},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	modelRun, err := toModelRun(apiRun)
	assert.Nil(t, err)
	runDetail, err := manager.CreateRun(ctx, modelRun)
	assert.Nil(t, err)
	return clientManager, manager, runDetail
}

func AssertUserError(t *testing.T, err error, expectedCode codes.Code) {
	userError, ok := err.(*util.UserError)
	assert.True(t, ok)
	assert.Equal(t, expectedCode, userError.ExternalStatusCode())
}

func getPermissionDeniedError(userIdentity string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	return util.NewPermissionDeniedError(
		errors.New("Unauthorized access"),
		"User '%s' is not authorized with reason: %s (request: %+v)",
		userIdentity,
		"this is not allowed",
		resourceAttributes,
	)
}

func wrapFailedAuthzApiResourcesError(err error) error {
	return util.Wrap(err, "Failed to authorize with API")
}

func wrapFailedAuthzRequestError(err error) error {
	return util.Wrap(err, "Failed to authorize the request")
}
