// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type reportValidationClient struct {
	client.SwfClientInterface
	get func(context.Context, string) (*swfapi.ScheduledWorkflow, error)
}

func (c *reportValidationClient) ScheduledWorkflow(namespace string) swfclient.ScheduledWorkflowInterface {
	return &reportValidationWorkflowClient{ScheduledWorkflowInterface: c.SwfClientInterface.ScheduledWorkflow(namespace), get: c.get}
}

type reportValidationWorkflowClient struct {
	swfclient.ScheduledWorkflowInterface
	get func(context.Context, string) (*swfapi.ScheduledWorkflow, error)
}

func (c *reportValidationWorkflowClient) Get(ctx context.Context, name string, _ metav1.GetOptions) (*swfapi.ScheduledWorkflow, error) {
	return c.get(ctx, name)
}

func TestReportScheduledWorkflowResource_StaleReportPreservesAPIMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disable"
		if enabled {
			name = "enable"
		}
		t.Run(name, func(t *testing.T) {
			store, manager, job := initWithJob(t)
			defer store.Close()
			require.NoError(t, manager.ChangeJobMode(context.Background(), job.UUID, !enabled))
			swfClient := manager.getScheduledWorkflowClient(job.Namespace)
			old, err := swfClient.Get(context.Background(), job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			stale := old.DeepCopy()
			stale.Generation = 1
			stale.Spec.Enabled = !enabled
			require.NoError(t, manager.ChangeJobMode(context.Background(), job.UUID, enabled))
			live, err := swfClient.Get(context.Background(), job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			live = live.DeepCopy()
			live.Spec.Enabled = enabled
			live.Generation = 2 // The fake client does not advance CR generations.
			manager.swfClient = &reportValidationClient{SwfClientInterface: manager.swfClient, get: func(ctx context.Context, name string) (*swfapi.ScheduledWorkflow, error) {
				_, bounded := ctx.Deadline()
				require.True(t, bounded, "the live read must not hold a row lock indefinitely")
				require.Equal(t, job.K8SName, name)
				return live.DeepCopy(), nil
			}}
			require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(stale)))
			stored, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, enabled, stored.Enabled, "a stale report must not reverse a completed API change")
		})
	}
}

func TestReportScheduledWorkflowResource_CurrentGenerationAcceptsDirectEdit(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	client := manager.getScheduledWorkflowClient(job.Namespace)
	live, err := client.Get(context.Background(), job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	live = live.DeepCopy()
	live.Generation = 2
	live.Spec.Enabled = false
	_, err = client.Update(context.Background(), live)
	require.NoError(t, err)
	require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(live.DeepCopy())))
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, stored.Enabled, "current direct Kubernetes edits must still sync to the API")
}

func TestReportScheduledWorkflowResource_LiveReadFailsClosed(t *testing.T) {
	for _, scenario := range []string{"unavailable", "deleted", "replaced", "wrong-namespace", "nil"} {
		t.Run(scenario, func(t *testing.T) {
			store, manager, job := initWithJob(t)
			defer store.Close()
			client := manager.getScheduledWorkflowClient(job.Namespace)
			live, err := client.Get(context.Background(), job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			report := live.DeepCopy()
			report.Spec.Enabled = false
			before, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			if scenario == "deleted" {
				require.NoError(t, client.Delete(context.Background(), job.K8SName, &metav1.DeleteOptions{}))
			} else {
				manager.swfClient = &reportValidationClient{SwfClientInterface: manager.swfClient, get: func(ctx context.Context, _ string) (*swfapi.ScheduledWorkflow, error) {
					_, bounded := ctx.Deadline()
					require.True(t, bounded)
					current := live.DeepCopy()
					switch scenario {
					case "unavailable":
						return nil, errors.New("Kubernetes unavailable")
					case "replaced":
						current.UID = "another-job"
					case "wrong-namespace":
						current.Namespace = "another-namespace"
					case "nil":
						return nil, nil
					}
					return current, nil
				}}
			}
			require.Error(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(report)))
			after, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}
