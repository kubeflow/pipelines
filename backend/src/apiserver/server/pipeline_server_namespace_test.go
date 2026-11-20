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
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const installationNamespace = "kubeflow"

// k8sPipelineClient returns a Kubernetes-backed store holding one pipeline in
// the installation namespace and one in a tenant namespace, so a lookup that
// omits the namespace can be distinguished from one that names it.
func k8sPipelineClient(t *testing.T) (ctrlclient.Client, ctrlclient.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	installPipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "00000000-0000-0000-0000-00000000000a",
			Name:      "shared-name",
			Namespace: installationNamespace,
		},
	}
	tenantPipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "00000000-0000-0000-0000-00000000000b",
			Name:      "shared-name",
			Namespace: "tenant-a",
		},
	}

	c := ctrlfake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(installPipeline, tenantPipeline).
		Build()
	return c, c
}

func multiUserPipelineServer(t *testing.T, authorized bool) (*PipelineServer, func()) {
	t.Helper()
	initEnvVars()
	// initEnvVars pins POD_NAMESPACE to ns1. Point it back at the namespace the
	// fixture pipeline actually lives in, so an omitted-namespace lookup would
	// reach a real installation-namespace pipeline if the guard were removed,
	// rather than simply missing.
	viper.Set(common.PodNamespace, installationNamespace)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	if !authorized {
		clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	}
	k8sClient, k8sClientNoCache := k8sPipelineClient(t)
	clientManager.SetPipelineStore(storage.NewPipelineStoreKubernetes(k8sClient, k8sClientNoCache))

	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return createPipelineServer(resourceManager, nil), func() { clientManager.Close() }
}

func userContext() context.Context {
	md := metadata.New(map[string]string{
		common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com",
	})
	return metadata.NewIncomingContext(context.Background(), md)
}

// With REQUIRE_NAMESPACE_FOR_PIPELINES unset (the default), an omitted namespace
// clears ValidateNamespaceRequired and is authorized as the shared-read path. The
// store must then refuse to fall back to the installation namespace, otherwise a
// caller receives a pipeline in a namespace they were never authorized for.
func TestGetPipelineByName_MultiUser_OmittedNamespaceIsRejected(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.RequireNamespaceForPipelines, "false")
	defer viper.Set(common.RequireNamespaceForPipelines, "false")
	viper.Set("POD_NAMESPACE", installationNamespace)
	defer viper.Set("POD_NAMESPACE", "")

	pipelineServer, cleanup := multiUserPipelineServer(t, true)
	defer cleanup()

	pipeline, err := pipelineServer.GetPipelineByName(userContext(),
		&apiv2.GetPipelineByNameRequest{Name: "shared-name", Namespace: ""})

	require.Error(t, err, "an omitted namespace must not resolve to the installation namespace")
	assert.Nil(t, pipeline)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineByName_MultiUser_AuthorizedNamespaceSucceeds(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.RequireNamespaceForPipelines, "false")
	defer viper.Set(common.RequireNamespaceForPipelines, "false")
	viper.Set("POD_NAMESPACE", installationNamespace)
	defer viper.Set("POD_NAMESPACE", "")

	pipelineServer, cleanup := multiUserPipelineServer(t, true)
	defer cleanup()

	pipeline, err := pipelineServer.GetPipelineByName(userContext(),
		&apiv2.GetPipelineByNameRequest{Name: "shared-name", Namespace: "tenant-a"})

	require.NoError(t, err)
	require.NotNil(t, pipeline)
	assert.Equal(t, "shared-name", pipeline.GetDisplayName())
}

func TestGetPipelineByName_MultiUser_UnauthorizedNamespaceIsDenied(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.RequireNamespaceForPipelines, "false")
	defer viper.Set(common.RequireNamespaceForPipelines, "false")
	viper.Set("POD_NAMESPACE", installationNamespace)
	defer viper.Set("POD_NAMESPACE", "")

	pipelineServer, cleanup := multiUserPipelineServer(t, false)
	defer cleanup()

	_, err := pipelineServer.GetPipelineByName(userContext(),
		&apiv2.GetPipelineByNameRequest{Name: "shared-name", Namespace: "tenant-a"})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, err.(*util.UserError).ExternalStatusCode())
}

const (
	k8sParentPipelineA = "00000000-0000-0000-0000-0000000000a1"
	k8sParentPipelineB = "00000000-0000-0000-0000-0000000000b1"
	k8sParentVersionB  = "00000000-0000-0000-0000-0000000000b2"
)

// parentCheckPipelineServer serves two pipelines, A and B, in one namespace from
// the Kubernetes pipeline store, with one version owned by B. The store reads a
// version's parent from its owner reference, not from a database column.
func parentCheckPipelineServer(t *testing.T, authorized bool) (*PipelineServer, func()) {
	t.Helper()
	initEnvVars()
	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipelineA := &v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{UID: k8sParentPipelineA, Name: "pipeline-a", Namespace: "tenant-a"}}
	pipelineB := &v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{UID: k8sParentPipelineB, Name: "pipeline-b", Namespace: "tenant-a"}}
	versionB := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       k8sParentVersionB,
			Name:      "pipeline-b-v1",
			Namespace: "tenant-a",
			Labels:    map[string]string{"pipelines.kubeflow.org/pipeline-id": k8sParentPipelineB},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: v2beta1.GroupVersion.String(),
				Kind:       "Pipeline",
				Name:       "pipeline-b",
				UID:        k8sParentPipelineB,
			}},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineName: "pipeline-b",
			PipelineSpec: v2beta1.IRSpec{Value: map[string]interface{}{
				"pipelineInfo":  map[string]interface{}{"name": "pipeline-b"},
				"root":          map[string]interface{}{"dag": map[string]interface{}{"tasks": map[string]interface{}{}}},
				"schemaVersion": "2.1.0",
				"sdkVersion":    "kfp-2.13.0",
			}},
		},
	}
	k8sClient := ctrlfake.NewClientBuilder().WithScheme(scheme).WithObjects(pipelineA, pipelineB, versionB).Build()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	if !authorized {
		clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	}
	clientManager.SetPipelineStore(storage.NewPipelineStoreKubernetes(k8sClient, k8sClient))
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return createPipelineServer(resourceManager, nil), func() { clientManager.Close() }
}

func TestPipelineVersionParent_KubernetesStore_MismatchIsNotFound(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	pipelineServer, cleanup := parentCheckPipelineServer(t, true)
	defer cleanup()
	ctx := userContext()

	_, err := pipelineServer.GetPipelineVersion(ctx, &apiv2.GetPipelineVersionRequest{PipelineId: k8sParentPipelineA, PipelineVersionId: k8sParentVersionB})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	_, err = pipelineServer.UpdatePipelineVersion(ctx, &apiv2.UpdatePipelineVersionRequest{PipelineVersion: &apiv2.PipelineVersion{PipelineId: k8sParentPipelineA, PipelineVersionId: k8sParentVersionB, DisplayName: "renamed"}})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	_, err = pipelineServer.DeletePipelineVersion(ctx, &apiv2.DeletePipelineVersionRequest{PipelineId: k8sParentPipelineA, PipelineVersionId: k8sParentVersionB})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	version, err := pipelineServer.GetPipelineVersion(ctx, &apiv2.GetPipelineVersionRequest{PipelineId: k8sParentPipelineB, PipelineVersionId: k8sParentVersionB})
	require.NoError(t, err, "the version must still exist under its own pipeline")
	assert.Equal(t, k8sParentVersionB, version.GetPipelineVersionId())
	assert.NotEqual(t, "renamed", version.GetDisplayName())
}

// Authorization runs on the version's own namespace before the parent check, so
// a caller without access gets PermissionDenied and learns nothing about which
// pipeline the version belongs to.
func TestPipelineVersionParent_UnauthorizedCallerIsDeniedBeforeParentCheck(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	pipelineServer, cleanup := parentCheckPipelineServer(t, false)
	defer cleanup()
	ctx := userContext()

	for _, pipelineID := range []string{k8sParentPipelineA, k8sParentPipelineB} {
		_, err := pipelineServer.GetPipelineVersion(ctx, &apiv2.GetPipelineVersionRequest{PipelineId: pipelineID, PipelineVersionId: k8sParentVersionB})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, err.(*util.UserError).ExternalStatusCode(), "pipeline %s", pipelineID)

		_, err = pipelineServer.DeletePipelineVersion(ctx, &apiv2.DeletePipelineVersionRequest{PipelineId: pipelineID, PipelineVersionId: k8sParentVersionB})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, err.(*util.UserError).ExternalStatusCode(), "pipeline %s", pipelineID)
	}
}
