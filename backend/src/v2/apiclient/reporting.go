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

package apiclient

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Runtime launchers carry the same identity on every task and artifact RPC.
const (
	KFPTaskIDEnvVar        = "KFP_TASK_ID"
	KFPTaskAttemptIDEnvVar = "KFP_TASK_ATTEMPT_ID"
	KFPTaskIDHeader        = "kfp-task-id"
	KFPTaskAttemptIDHeader = "kfp-task-attempt-id"
)

func taskReportInterceptor(cfg *Config) (grpc.UnaryClientInterceptor, error) {
	taskID, attemptID := cfg.TaskID, cfg.TaskAttemptID
	if taskID == "" {
		taskID = os.Getenv(KFPTaskIDEnvVar)
	}
	if attemptID == "" {
		attemptID = os.Getenv(KFPTaskAttemptIDEnvVar)
	}
	// Argo launchers need no runtime identity. A task ID alone does not opt in.
	if attemptID == "" {
		return nil, nil
	}
	if taskID == "" {
		return nil, fmt.Errorf("%s requires %s for attempt-fenced launcher reports", KFPTaskAttemptIDEnvVar, KFPTaskIDEnvVar)
	}
	return func(ctx context.Context, method string, request, response any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		headers, _ := metadata.FromOutgoingContext(ctx)
		headers = headers.Copy()
		headers.Set(KFPTaskIDHeader, taskID)
		headers.Set(KFPTaskAttemptIDHeader, attemptID)
		return invoke(metadata.NewOutgoingContext(ctx, headers), method, request, response, conn, options...)
	}, nil
}
