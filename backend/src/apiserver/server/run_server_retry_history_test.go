// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestRetryLogHistoryUsesRunAuthorization(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, run := initWithOneTimeRunV2(t)
	defer clients.Close()
	task, err := clients.TaskStore().CreateTask(&model.Task{
		RunUUID: run.UUID, Namespace: run.Namespace, Name: "failed-task",
		State:          model.TaskStatus(apiv2.PipelineTask_FAILED),
		StatusMetadata: model.JSONData{"customProperties": map[string]interface{}{"driver_logs_uri": "gs://private/run/driver-logs"}},
	})
	require.NoError(t, err)
	_, err = clients.DB().Exec(`UPDATE run_details SET RetryGeneration = 1 WHERE UUID = ?`, run.UUID)
	require.NoError(t, err)
	require.NoError(t, clients.TaskStore().ResetTasksForRetry(run.UUID, 1, []string{task.UUID}))
	ctx := ctxWithUser()
	server := createRunServer(manager)
	got, err := server.GetTask(ctx, &apiv2.GetTaskRequest{RunId: run.UUID, TaskId: task.UUID})
	require.NoError(t, err)
	require.Len(t, got.GetLogHistory(), 1)
	require.Equal(t, "gs://private/run/driver-logs", got.GetLogHistory()[0].GetDriverLogsUri())
	require.Empty(t, got.GetStatusMetadata())

	// Knowing a task ID must not allow fetching its history through another run.
	got, err = server.GetTask(ctx, &apiv2.GetTaskRequest{RunId: "another-run", TaskId: task.UUID})
	require.Error(t, err)
	require.Nil(t, got)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server = createRunServer(manager)
	got, err = server.GetTask(ctx, &apiv2.GetTaskRequest{RunId: run.UUID, TaskId: task.UUID})
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "%v", err)
	require.Nil(t, got)
	listed, err := server.ListTasks(ctx, &apiv2.ListTasksRequest{RunId: run.UUID})
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "%v", err)
	require.Nil(t, listed)
}
