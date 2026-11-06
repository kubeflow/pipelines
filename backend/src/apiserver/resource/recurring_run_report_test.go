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

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Model the API server's generation changes without relying on informer timing.
type generationSwfClient struct{ client.SwfClientInterface }

func (c generationSwfClient) ScheduledWorkflow(namespace string) swfclient.ScheduledWorkflowInterface {
	return generationScheduledWorkflowClient{c.SwfClientInterface.ScheduledWorkflow(namespace)}
}

type generationScheduledWorkflowClient struct {
	swfclient.ScheduledWorkflowInterface
}

func (c generationScheduledWorkflowClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, subresources ...string) (*swfapi.ScheduledWorkflow, error) {
	current, err := c.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	patched := current.DeepCopy()
	if err := json.Unmarshal(data, patched); err != nil {
		return nil, err
	}
	if patched.Spec.Enabled != current.Spec.Enabled {
		patched.Generation++
	}
	return c.Update(ctx, patched)
}

func TestReportScheduledWorkflowResource_PreservesNewerMode(t *testing.T) {
	originalMode := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, "false")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, originalMode) })
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled-%v", enabled), func(t *testing.T) {
			clients, manager, job := initWithJob(t)
			defer clients.Close()
			manager.swfClient = generationSwfClient{manager.swfClient}
			ctx := context.Background()
			swfClient := manager.getScheduledWorkflowClient(job.Namespace)
			current, err := swfClient.Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			current = current.DeepCopy()
			current.Generation = 1
			current.Spec.Enabled = !enabled
			_, err = swfClient.Update(ctx, current)
			require.NoError(t, err)
			// Simulate the persistence agent holding an informer snapshot.
			stale := util.NewScheduledWorkflow(current.DeepCopy())
			require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, enabled))
			require.NoError(t, manager.ReportScheduledWorkflowResource(stale))
			stored, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, enabled, stored.Enabled, "an old report must not undo the successful API request")

			// A later direct standalone edit still synchronizes its specification.
			edited := current.DeepCopy()
			edited.Generation = 3
			edited.Spec.Enabled = !enabled
			require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(edited)))
			stored, err = manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, !enabled, stored.Enabled)
		})
	}
}

func TestReportScheduledWorkflowResource_AfterDeletedResourceDisable(t *testing.T) {
	clients, manager, job := initWithJob(t)
	defer clients.Close()
	ctx := context.Background()
	swfClient := manager.getScheduledWorkflowClient(job.Namespace)
	current, err := swfClient.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	stale := current.DeepCopy()
	stale.Generation = 10
	require.True(t, stale.Spec.Enabled)
	require.NoError(t, swfClient.Delete(ctx, job.K8SName, &metav1.DeleteOptions{}))
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
	require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(stale)))
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
}

func TestChangeJobMode_RejectsDifferentPatchIdentity(t *testing.T) {
	clients, manager, job := initWithJob(t)
	defer clients.Close()
	current, err := manager.getScheduledWorkflowClient(job.Namespace).Get(context.Background(), job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	// The fake deliberately does not enforce the patch's UID precondition.
	current.UID = "recreated-resource"
	require.Error(t, manager.ChangeJobMode(context.Background(), job.UUID, false))
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.True(t, stored.Enabled)
}

type emptyPatchSwfClient struct{ client.SwfClientInterface }

func (c emptyPatchSwfClient) ScheduledWorkflow(namespace string) swfclient.ScheduledWorkflowInterface {
	return emptyPatchScheduledWorkflowClient{c.SwfClientInterface.ScheduledWorkflow(namespace)}
}

type emptyPatchScheduledWorkflowClient struct {
	swfclient.ScheduledWorkflowInterface
}

func (c emptyPatchScheduledWorkflowClient) Patch(context.Context, string, types.PatchType, []byte, ...string) (*swfapi.ScheduledWorkflow, error) {
	return nil, nil
}

func TestChangeJobMode_RejectsEmptyPatchResponse(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			clients, manager, job := initWithJob(t)
			defer clients.Close()
			manager.swfClient = emptyPatchSwfClient{manager.swfClient}
			require.ErrorContains(t, manager.ChangeJobMode(context.Background(), job.UUID, enabled), "empty ScheduledWorkflow patch response")
			stored, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, job.Enabled, stored.Enabled)
		})
	}
}
