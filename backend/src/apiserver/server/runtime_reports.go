// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"google.golang.org/grpc/metadata"
)

func taskReportIdentity(ctx context.Context) (*model.TaskReportIdentity, error) {
	headers, _ := metadata.FromIncomingContext(ctx)
	tasks, attempts := headers.Get(apiclient.KFPTaskIDHeader), headers.Get(apiclient.KFPTaskAttemptIDHeader)
	if len(tasks) == 0 && len(attempts) == 0 {
		return nil, nil
	}
	if len(tasks) != 1 || len(attempts) != 1 || strings.TrimSpace(tasks[0]) == "" || strings.TrimSpace(attempts[0]) == "" {
		return nil, util.NewInvalidInputError("Launcher reports require exactly one nonempty kfp-task-id and kfp-task-attempt-id header")
	}
	return &model.TaskReportIdentity{TaskUUID: tasks[0], AttemptUUID: attempts[0]}, nil
}
