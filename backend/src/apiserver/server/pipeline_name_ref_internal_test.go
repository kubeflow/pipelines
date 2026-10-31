// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pipelineNameRefManager(t *testing.T, sarClient client.SubjectAccessReviewInterface) *resource.ResourceManager {
	t.Helper()
	initEnvVars()
	clients, err := resource.NewFakeClientManager(util.NewFakeTimeForEpoch(), util.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	if sarClient != nil {
		// The resource manager captures this client, so set it before construction.
		clients.SubjectAccessReviewClientFake = sarClient
	}
	return resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
}

func createNamedPipeline(t *testing.T, manager *resource.ResourceManager, namespace, name string, versions ...string) (*model.Pipeline, []*model.PipelineVersion) {
	t.Helper()
	pipeline, err := manager.CreatePipeline(&model.Pipeline{Name: name, Namespace: namespace})
	require.NoError(t, err)
	var pipelineVersions []*model.PipelineVersion
	for _, version := range versions {
		pipelineVersion, err := manager.CreatePipelineVersion(&model.PipelineVersion{
			Name: version, PipelineId: pipeline.UUID, PipelineSpec: model.LargeText(testIRPipeline),
		})
		require.NoError(t, err)
		pipelineVersions = append(pipelineVersions, pipelineVersion)
	}
	return pipeline, pipelineVersions
}

func pipelineNameRef(pipelineName, pipelineVersionName string) *apiv2.PipelineVersionReference {
	return &apiv2.PipelineVersionReference{PipelineName: pipelineName, PipelineVersionName: pipelineVersionName}
}

func runByName(namespace string, ref *apiv2.PipelineVersionReference) *apiv2.Run {
	return &apiv2.Run{
		DisplayName:    "run1",
		Namespace:      namespace,
		PipelineSource: &apiv2.Run_PipelineVersionReference{PipelineVersionReference: ref},
	}
}

func TestCreateRun_ByPipelineName_UsesDefaultVersion(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, _ := createNamedPipeline(t, manager, "", "training", "v1", "v2")
	defaultVersion, err := manager.GetDefaultPipelineVersion(pipeline.UUID)
	require.NoError(t, err)

	run, err := createRunServer(manager).CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByName("", pipelineNameRef("training", "")),
	})
	require.NoError(t, err)
	assert.Equal(t, pipeline.UUID, run.GetPipelineVersionReference().GetPipelineId())
	assert.Equal(t, defaultVersion.UUID, run.GetPipelineVersionReference().GetPipelineVersionId())
}

func TestCreateRun_ByPipelineAndVersionName(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1", "v2")
	defaultVersion, err := manager.GetDefaultPipelineVersion(pipeline.UUID)
	require.NoError(t, err)
	// Name a version other than the default, so a resolution that ignored the
	// version name could not pass.
	named := versions[0]
	if named.UUID == defaultVersion.UUID {
		named = versions[1]
	}

	run, err := createRunServer(manager).CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByName("", pipelineNameRef("training", named.Name)),
	})
	require.NoError(t, err)
	assert.Equal(t, pipeline.UUID, run.GetPipelineVersionReference().GetPipelineId())
	assert.Equal(t, named.UUID, run.GetPipelineVersionReference().GetPipelineVersionId())
}

func TestCreateRun_ByPipelineName_InvalidReferences(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, _ := createNamedPipeline(t, manager, "", "training", "v1")

	tests := []struct {
		name string
		ref  *apiv2.PipelineVersionReference
		code codes.Code
	}{
		{"version name without pipeline name", pipelineNameRef("", "v1"), codes.InvalidArgument},
		{"pipeline name and pipeline ID", &apiv2.PipelineVersionReference{PipelineName: "training", PipelineId: pipeline.UUID}, codes.InvalidArgument},
		{"unknown pipeline", pipelineNameRef("missing", ""), codes.NotFound},
		{"unknown version", pipelineNameRef("training", "missing"), codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createRunServer(manager).CreateRun(context.Background(), &apiv2.CreateRunRequest{
				Run: runByName("", tt.ref),
			})
			require.Error(t, err)
			assert.True(t, util.IsUserErrorCodeMatch(err, tt.code), "got %v", err)
		})
	}
}

func TestCreateRecurringRun_ByPipelineAndVersionName(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1", "v2")

	recurringRun, err := createJobServer(manager).CreateRecurringRun(context.Background(), &apiv2.CreateRecurringRunRequest{
		RecurringRun: &apiv2.RecurringRun{
			DisplayName:    "recurring1",
			Mode:           apiv2.RecurringRun_DISABLE,
			MaxConcurrency: 1,
			PipelineSource: &apiv2.RecurringRun_PipelineVersionReference{
				PipelineVersionReference: pipelineNameRef("training", versions[0].Name),
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, pipeline.UUID, recurringRun.GetPipelineVersionReference().GetPipelineId())
	assert.Equal(t, versions[0].UUID, recurringRun.GetPipelineVersionReference().GetPipelineVersionId())
}

func TestCreateRun_ByPipelineName_ScopedToRunNamespace(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	manager := pipelineNameRefManager(t, nil)
	createNamedPipeline(t, manager, "ns2", "training", "v1")

	// The only pipeline with this name lives in another namespace.
	_, err := createRunServer(manager).CreateRun(defaultOnlyGrantContext(), &apiv2.CreateRunRequest{
		Run: runByName("ns1", pipelineNameRef("training", "")),
	})
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "got %v", err)

	// With a same-named pipeline in both namespaces, the run's namespace wins.
	own, _ := createNamedPipeline(t, manager, "ns1", "training", "v1")
	run, err := createRunServer(manager).CreateRun(defaultOnlyGrantContext(), &apiv2.CreateRunRequest{
		Run: runByName("ns1", pipelineNameRef("training", "")),
	})
	require.NoError(t, err)
	assert.Equal(t, own.UUID, run.GetPipelineVersionReference().GetPipelineId())
}

// Denies reading pipelines and allows everything else.
type denyPipelineReadSARClient struct{}

func (denyPipelineReadSARClient) Create(_ context.Context, sar *authzv1.SubjectAccessReview, _ v1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	ra := sar.Spec.ResourceAttributes
	allowed := ra == nil || ra.Resource != common.RbacResourceTypePipelines || ra.Verb != common.RbacResourceVerbGet
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
}

func TestCreateRun_ByPipelineName_AuthorizesBeforeLookup(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	manager := pipelineNameRefManager(t, denyPipelineReadSARClient{})
	createNamedPipeline(t, manager, "ns1", "training", "v1")

	// An existing and a missing name must fail the same way, so the lookup
	// cannot reveal which pipelines exist to a caller who cannot read them.
	for _, name := range []string{"training", "missing"} {
		_, err := createRunServer(manager).CreateRun(defaultOnlyGrantContext(), &apiv2.CreateRunRequest{
			Run: runByName("ns1", pipelineNameRef(name, "")),
		})
		require.Error(t, err)
		assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "%s: got %v", name, err)
	}
}
