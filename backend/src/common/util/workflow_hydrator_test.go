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

package util

import (
	"context"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseArgoPersistConfig(t *testing.T) {
	persist, err := ParseArgoPersistConfig([]byte(`
archive: true
nodeStatusOffLoad: true
clusterName: default
postgresql:
  host: postgres.example.invalid
  port: 5432
  database: argo
  tableName: argo_workflows
`))
	require.NoError(t, err)
	require.NotNil(t, persist)
	assert.True(t, persist.NodeStatusOffload)
	assert.Equal(t, "default", persist.GetClusterName())
	require.NotNil(t, persist.PostgreSQL)
	assert.Equal(t, "argo", persist.PostgreSQL.Database)
	assert.Equal(t, "argo_workflows", persist.PostgreSQL.TableName)
}

func TestParseArgoPersistConfig_Empty(t *testing.T) {
	_, err := ParseArgoPersistConfig(nil)
	require.Error(t, err)
}

func TestArgoPersistSecretNames(t *testing.T) {
	persist, err := ParseArgoPersistConfig([]byte(`
nodeStatusOffLoad: true
postgresql:
  host: postgres.example.invalid
  database: argo
  userNameSecret:
    name: pg-user
    key: username
  passwordSecret:
    name: pg-pass
    key: password
`))
	require.NoError(t, err)
	assert.Equal(t, []string{"pg-pass", "pg-user"}, ArgoPersistSecretNames(persist))
	assert.Nil(t, ArgoPersistSecretNames(nil))
}

func TestWorkflow_HydrateAndRetryOffloadedNodes(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	nodes := workflowapi.Nodes{
		"ok": {
			ID:    "ok",
			Name:  "my-wf",
			Phase: workflowapi.NodeSucceeded,
			Type:  workflowapi.NodeTypePod,
		},
		"fail": {
			ID:    "fail",
			Name:  "my-wf.step2",
			Phase: workflowapi.NodeFailed,
			Type:  workflowapi.NodeTypePod,
		},
	}
	repo.Put("wf-uid", "offload-hash", nodes)
	SetWorkflowHydratorForTest(t, NewMemoryWorkflowHydrator(repo))

	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-wf",
			UID:  "wf-uid",
			Labels: map[string]string{
				"workflows.argoproj.io/completed":   "true",
				LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: workflowapi.WorkflowStatus{
			Phase:                    workflowapi.WorkflowFailed,
			OffloadNodeStatusVersion: "offload-hash",
		},
	})

	require.Error(t, workflow.CanRetry())
	require.NoError(t, workflow.Hydrate(context.Background()))
	require.NoError(t, workflow.CanRetry())
	assert.Equal(t, workflowapi.NodeFailed, workflow.Status.Nodes["fail"].Phase)

	retryExec, podsToDelete, err := workflow.GenerateRetryExecution()
	require.NoError(t, err)
	assert.NotEmpty(t, podsToDelete)

	retryWorkflow := retryExec.(*Workflow)
	assert.Equal(t, workflowapi.WorkflowRunning, retryWorkflow.Status.Phase)
	_, succeededExists := retryWorkflow.Status.Nodes["ok"]
	_, failedExists := retryWorkflow.Status.Nodes["fail"]
	assert.True(t, succeededExists)
	assert.False(t, failedExists)
	require.NoError(t, retryExec.Dehydrate(context.Background()))
}
