// Copyright 2026 The Kubeflow Authors
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

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestGetPipelineByName_SQLNamespaceIsolation(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com",
	}))
	for _, apiVersion := range []string{"v1beta1", "v2beta1"} {
		t.Run(apiVersion, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				multiUser     bool
				includeShared bool
				unauthorized  bool
				namespace     string
				wantNamespace string
				wantCode      codes.Code
			}{
				{name: "omitted_namespace_cannot_read_private", multiUser: true, wantCode: codes.NotFound},
				{name: "shared_not_shadowed_by_newer_private", multiUser: true, includeShared: true},
				{name: "explicit_authorized_namespace", multiUser: true, includeShared: true, namespace: "tenant-a", wantNamespace: "tenant-a"},
				{name: "explicit_unauthorized_namespace", multiUser: true, includeShared: true, unauthorized: true, namespace: "tenant-a", wantCode: codes.PermissionDenied},
				{name: "shared_read_requires_no_namespace_permission", multiUser: true, includeShared: true, unauthorized: true},
				{name: "single_user_resolves_shared_namespace", includeShared: true, namespace: "tenant-a"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					previousMultiUser := viper.Get(common.MultiUserMode)
					previousRequireNamespace := viper.Get(common.RequireNamespaceForPipelines)
					t.Cleanup(func() {
						viper.Set(common.MultiUserMode, previousMultiUser)
						viper.Set(common.RequireNamespaceForPipelines, previousRequireNamespace)
					})
					viper.Set(common.MultiUserMode, tc.multiUser)
					viper.Set(common.RequireNamespaceForPipelines, false)

					clientManager := resource.NewFakeClientManagerOrFatalV2()
					t.Cleanup(func() { clientManager.Close() })
					if tc.unauthorized {
						clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
					}
					pipelineStore := clientManager.PipelineStore()
					ids := make(map[string]string)
					// The fake clock advances on each creation, making tenant pipelines
					// and their versions newer than the same-name shared pipeline.
					for _, namespace := range []string{"", "tenant-a", "tenant-b"} {
						if namespace == "" && !tc.includeShared {
							continue
						}
						pipeline, version, err := pipelineStore.CreatePipelineAndPipelineVersion(
							&model.Pipeline{Name: "same-name", DisplayName: "same-name", Namespace: namespace},
							&model.PipelineVersion{Name: "version", DisplayName: "version"},
						)
						require.NoError(t, err)
						require.NoError(t, pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineReady))
						require.NoError(t, pipelineStore.UpdatePipelineVersionStatus(version.UUID, model.PipelineVersionReady))
						ids[namespace] = pipeline.UUID
					}

					resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
					var gotID string
					var err error
					if apiVersion == "v1beta1" {
						pipeline, getErr := createPipelineServerV1(resourceManager, nil).GetPipelineByNameV1(ctx,
							&apiv1beta1.GetPipelineByNameRequest{Name: "same-name", Namespace: tc.namespace})
						gotID, err = pipeline.GetId(), getErr
					} else {
						pipeline, getErr := createPipelineServer(resourceManager, nil).GetPipelineByName(ctx,
							&apiv2beta1.GetPipelineByNameRequest{Name: "same-name", Namespace: tc.namespace})
						gotID, err = pipeline.GetPipelineId(), getErr
					}
					if tc.wantCode != codes.OK {
						require.Error(t, err)
						var userError *util.UserError
						require.ErrorAs(t, err, &userError)
						assert.Equal(t, tc.wantCode, userError.ExternalStatusCode())
						assert.Empty(t, gotID)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, ids[tc.wantNamespace], gotID)
				})
			}
		})
	}
}
