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
	"encoding/json"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReportWorkflowV1(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := &ReportServerV1{
		BaseReportServer: &BaseReportServer{
			resourceManager: resourceManager,
		},
	}

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "default",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
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
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := reportServer.ReportWorkflowV1(nil, &api.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})
	assert.Nil(t, err)
	run, err = resourceManager.GetRun(run.UUID)
	assert.Nil(t, err)
	assert.NotNil(t, run)
}

func TestReportWorkflowV1_ValidationFailed(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServer(resourceManager)

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			UID:       types.UID(run.UUID),
		},
	})

	_, err := reportServer.ReportWorkflow(nil, &apiv2.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestReportWorkflow(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServer(resourceManager)

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "default",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
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
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := reportServer.ReportWorkflow(nil, &apiv2.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})
	assert.Nil(t, err)
	run, err = resourceManager.GetRun(run.UUID)
	assert.Nil(t, err)
	assert.NotNil(t, run)
}

func TestReportWorkflow_RejectsStaleTerminalReportWithoutPersistingTasks(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	ctx := context.Background()

	workflowClient := clientManager.ExecClientFake.Execution("ns1")
	liveWorkflow, err := workflowClient.Get(ctx, run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.SetVersion("terminal-version")
	staleWorkflow := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	staleWorkflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	staleWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	_, err = workflowClient.Update(ctx, staleWorkflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	_, err = resourceManager.ReportWorkflowResource(ctx, staleWorkflow)
	require.NoError(t, err)
	_, _, _, claimGeneration, err := clientManager.RunStore().ClaimRunForRetry(run.UUID, 0, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), claimGeneration)

	staleWorkflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"node-1": {
			ID:          "node-1",
			Name:        run.K8SName + ".task",
			DisplayName: "task",
			Type:        v1alpha1.NodeTypePod,
			Phase:       v1alpha1.NodeFailed,
		},
	}
	_, err = NewReportServer(resourceManager).ReportWorkflow(ctx, &apiv2.ReportWorkflowRequest{
		Workflow: staleWorkflow.ToStringForStore(),
	})
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable))
	assert.Contains(t, err.Error(), "Skipping stale workflow report")

	var taskCount int
	require.NoError(t, clientManager.DB().QueryRow(
		"SELECT COUNT(*) FROM tasks WHERE RunUUID = ?", run.UUID,
	).Scan(&taskCount))
	assert.Zero(t, taskCount)
}

func TestReportWorkflow_ValidationFailed(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServerV1(resourceManager)

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			UID:       types.UID(run.UUID),
		},
	})

	_, err := reportServer.ReportWorkflowV1(nil, &api.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestValidateReportWorkflowRequest(t *testing.T) {
	// Name
	workflow := &v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}
	marshalledWorkflow, _ := json.Marshal(workflow)
	expectedExecSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(string(marshalledWorkflow)))
	generatedWorkflow, err := validateReportWorkflowRequest(string(marshalledWorkflow))
	assert.Nil(t, err)
	assert.Equal(t, &expectedExecSpec, generatedWorkflow)
}

func TestValidateReportWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := validateReportWorkflowRequest("WRONG WORKFLOW")
	assert.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportWorkflowRequest_MissingField(t *testing.T) {
	// Name
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	_, err := validateReportWorkflowRequest(workflow.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	workflow = util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})

	_, err = validateReportWorkflowRequest(workflow.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	workflow = util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})

	_, err = validateReportWorkflowRequest(workflow.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestValidateReportScheduledWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := validateReportScheduledWorkflowRequest("WRONG_SCHEDULED_WORKFLOW")
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportScheduledWorkflowRequest_MissingField(t *testing.T) {
	// Name
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})

	_, err := validateReportScheduledWorkflowRequest(swf.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
		},
	})

	_, err = validateReportScheduledWorkflowRequest(swf.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
		},
	})

	_, err = validateReportScheduledWorkflowRequest(swf.ToStringForStore())
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestReportScheduledWorkflowV1_InvalidManifest(t *testing.T) {
	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServerV1(resourceManager)

	_, err := reportServer.ReportScheduledWorkflowV1(context.Background(), &api.ReportScheduledWorkflowRequest{
		ScheduledWorkflow: "INVALID_JSON",
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestReportScheduledWorkflowV1_MissingFields(t *testing.T) {
	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServerV1(resourceManager)

	// Missing name
	scheduledWorkflow := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})
	_, err := reportServer.ReportScheduledWorkflowV1(context.Background(), &api.ReportScheduledWorkflowRequest{
		ScheduledWorkflow: scheduledWorkflow.ToStringForStore(),
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestReportScheduledWorkflow_InvalidManifest(t *testing.T) {
	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServer(resourceManager)

	_, err := reportServer.ReportScheduledWorkflow(context.Background(), &apiv2.ReportScheduledWorkflowRequest{
		ScheduledWorkflow: "INVALID_JSON",
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Could not unmarshal")
}
