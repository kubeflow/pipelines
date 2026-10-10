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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// Models an RBAC grant narrowed to resourceNames: ["Default"]: experiment
// creation is permitted only when the request names that experiment, exactly
// as Kubernetes evaluates resource-name matching.
type defaultOnlyExperimentSARClient struct{}

func (defaultOnlyExperimentSARClient) Create(_ context.Context, sar *authzv1.SubjectAccessReview, _ v1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	ra := sar.Spec.ResourceAttributes
	allowed := true
	if ra != nil && ra.Resource == common.RbacResourceTypeExperiments && ra.Verb == common.RbacResourceVerbCreate {
		allowed = ra.Name == "Default"
	}
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
}

func defaultOnlyGrantManager(t *testing.T) *resource.ResourceManager {
	t.Helper()
	initEnvVars()
	clients, err := resource.NewFakeClientManager(util.NewFakeTimeForEpoch(), util.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	clients.SubjectAccessReviewClientFake = defaultOnlyExperimentSARClient{}
	return resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
}

func defaultOnlyGrantContext() context.Context {
	md := metadata.New(map[string]string{
		common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com",
	})
	return metadata.NewIncomingContext(context.Background(), md)
}

func helloWorldSpec(t *testing.T) *structpb.Struct {
	t.Helper()
	spec := &structpb.Struct{}
	require.NoError(t, yaml.Unmarshal([]byte(v2SpecHelloWorld), spec))
	return spec
}

// A grant scoped to the default experiment's name must still allow the implicit
// creation, so the authorization request has to name the experiment the way the
// explicit endpoint does.
func TestCreateRun_DefaultExperimentGrantScopedToName(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	server := createRunServer(defaultOnlyGrantManager(t))

	got, err := server.CreateRun(defaultOnlyGrantContext(), &apiv2beta1.CreateRunRequest{
		Run: &apiv2beta1.Run{
			DisplayName:    "run1",
			Namespace:      "ns1",
			PipelineSource: &apiv2beta1.Run_PipelineSpec{PipelineSpec: helloWorldSpec(t)},
			RuntimeConfig: &apiv2beta1.RuntimeConfig{
				Parameters: map[string]*structpb.Value{"param1": structpb.NewStringValue("world")},
			},
		},
	})

	require.NoError(t, err, "a grant limited to resourceNames [Default] must permit the implicit create")
	assert.Equal(t, "ns1", got.GetNamespace())
	assert.NotEmpty(t, got.GetExperimentId())
}

func TestCreateRecurringRun_DefaultExperimentGrantScopedToName(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	server := createJobServer(defaultOnlyGrantManager(t))

	got, err := server.CreateRecurringRun(defaultOnlyGrantContext(), &apiv2beta1.CreateRecurringRunRequest{
		RecurringRun: &apiv2beta1.RecurringRun{
			DisplayName:    "recurring1",
			Namespace:      "ns1",
			Mode:           apiv2beta1.RecurringRun_DISABLE,
			MaxConcurrency: 1,
			PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: helloWorldSpec(t)},
			RuntimeConfig: &apiv2beta1.RuntimeConfig{
				Parameters: map[string]*structpb.Value{"param1": structpb.NewStringValue("world")},
			},
		},
	})

	require.NoError(t, err, "a grant limited to resourceNames [Default] must permit the implicit create")
	assert.Equal(t, "ns1", got.GetNamespace())
	assert.NotEmpty(t, got.GetExperimentId())
}
