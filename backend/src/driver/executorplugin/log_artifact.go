// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes"
)

type logUploader struct {
	openBucket func(context.Context, kubernetes.Interface, string, *objectstore.Config, *objectstore.SessionInfo) (*blob.Bucket, error)
	uploadBlob func(context.Context, *blob.Bucket, string, string) error
}

type driverLogArtifactContext struct {
	Execution        *driver.Execution
	Task             string
	LocalPath        string
	OutputPathPrefix string
	Namespace        string
	PipelineRoot     string
	StoreSessionInfo string
	LogID            string
	RunID            string
	KFPAPI           kfpapi.API
}

func (u logUploader) uploadDriverLogArtifact(ctx context.Context, logContext *driverLogArtifactContext, k8sClient kubernetes.Interface) error {
	if logContext == nil {
		return fmt.Errorf("driver log context is missing; initialize it before publishing logs")
	}
	if logContext.PipelineRoot != "" {
		var session objectstore.SessionInfo
		if err := json.Unmarshal([]byte(logContext.StoreSessionInfo), &session); err != nil {
			return fmt.Errorf("failed to get session info from store: %w", err)
		}
		bucketConfig, err := objectstore.ParseBucketPathToConfig(logContext.PipelineRoot)
		if err != nil {
			return fmt.Errorf("failed to parse bucket config: %w", err)
		}
		bucket, err := u.openBucket(ctx, k8sClient, logContext.Namespace, bucketConfig, &session)
		if err != nil {
			return fmt.Errorf("failed to open bucket: %w", err)
		}
		defer bucket.Close()
		key := fmt.Sprintf("driver/%s-logs", logContext.LogID)
		if logContext.Execution != nil && logContext.OutputPathPrefix != "" {
			key = fmt.Sprintf("%s/%s/driver-logs", logContext.Task, logContext.OutputPathPrefix)
		}
		glog.Infof("Uploading log key: %s ...", key)
		err = u.uploadBlob(ctx, bucket, logContext.LocalPath, key)
		if err != nil {
			return fmt.Errorf("failed to upload log: %w", err)
		}
		if logContext.Execution != nil && logContext.Execution.TaskID != "" && logContext.KFPAPI != nil {
			uri := util.GenerateOutputURI(logContext.PipelineRoot, []string{key}, false)
			return registerDriverLog(ctx, logContext.KFPAPI, logContext.RunID, logContext.Execution.TaskID, logContext.Namespace, uri, logContext.StoreSessionInfo)
		}
	}
	return nil
}

// registerDriverLog establishes Artifact API ownership before exposing the URI,
// preserving task state and existing status metadata.
func registerDriverLog(ctx context.Context, api kfpapi.API, runID, taskID, namespace, uri, session string) error {
	task, err := api.GetTask(ctx, &go_client.GetTaskRequest{RunId: runID, TaskId: taskID})
	if err != nil {
		return fmt.Errorf("failed to read task for driver log: %w", err)
	}
	request := &go_client.CreateArtifactRequest{
		RunId: runID, TaskId: taskID, ProducerKey: "driver-logs", ReuseIfExists: true,
		Artifact: &go_client.Artifact{
			Name: "driver-logs", Type: go_client.Artifact_Artifact, Uri: &uri, Namespace: namespace,
			Metadata: map[string]*structpb.Value{"store_session_info": structpb.NewStringValue(session)},
		},
	}
	if attributes := task.GetTypeAttributes(); attributes != nil {
		request.IterationIndex = attributes.IterationIndex
	}
	if _, err := api.CreateArtifact(ctx, request); err != nil {
		return fmt.Errorf("failed to register driver log ownership; check artifact creation permissions: %w", err)
	}
	metadata := &go_client.PipelineTask_StatusMetadata{}
	if task.GetStatusMetadata() != nil {
		metadata = proto.Clone(task.GetStatusMetadata()).(*go_client.PipelineTask_StatusMetadata)
	}
	if metadata.CustomProperties == nil {
		metadata.CustomProperties = map[string]*structpb.Value{}
	}
	metadata.CustomProperties["driver_logs_uri"] = structpb.NewStringValue(uri)
	metadata.CustomProperties["store_session_info"] = structpb.NewStringValue(session)
	_, err = api.UpdateTask(ctx, &go_client.UpdateTaskRequest{
		RunId: runID, TaskId: taskID,
		Task: &go_client.PipelineTask{RunId: runID, TaskId: taskID, StatusMetadata: metadata},
	})
	if err != nil {
		return fmt.Errorf("failed to register driver log: %w", err)
	}
	return nil
}
