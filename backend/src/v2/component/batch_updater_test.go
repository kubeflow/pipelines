// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
)

func TestBatchUpdater_FlushDeduplicatesArtifactTasks(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	updater := NewBatchUpdater()

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_OUTPUT,
	}

	updater.QueueArtifactTask(artifactTask)
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_OUTPUT,
	})

	require.NoError(t, updater.Flush(context.Background(), mockAPI))

	resp, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, resp.ArtifactTasks, 1)
}
