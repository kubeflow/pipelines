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

// Package driverlogs publishes a driver invocation's log file as a run artifact.
package driverlogs

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

// Uploader stores driver logs through object-store hooks, which tests replace.
type Uploader struct {
	OpenBucket func(context.Context, kubernetes.Interface, string, *objectstore.Config, *objectstore.SessionInfo) (*blob.Bucket, error)
	UploadBlob func(context.Context, *blob.Bucket, string, string) error
}

// ArtifactContext carries what one invocation needs to publish its log.
// URI and Registered are filled in by the upload.
type ArtifactContext struct {
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
	// URI is the uploaded log's location; empty until the upload succeeds.
	URI string
	// Registered reports whether URI was linked to the task for the UI.
	Registered bool
}

// Upload stores the log and, when the execution has a task, links it for the UI.
func (u Uploader) Upload(ctx context.Context, logContext *ArtifactContext, k8sClient kubernetes.Interface) error {
	if logContext == nil {
		return fmt.Errorf("driver log context is missing; initialize it before publishing logs")
	}
	if logContext.PipelineRoot == "" {
		return nil
	}
	var session objectstore.SessionInfo
	if err := json.Unmarshal([]byte(logContext.StoreSessionInfo), &session); err != nil {
		return fmt.Errorf("failed to parse store session info for driver logs; check the kfp-launcher providers config: %w", err)
	}
	bucketConfig, err := objectstore.ParseBucketPathToConfig(logContext.PipelineRoot)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline root %q as a bucket config; use a <scheme>://bucket/path root: %w", logContext.PipelineRoot, err)
	}
	bucket, err := u.OpenBucket(ctx, k8sClient, logContext.Namespace, bucketConfig, &session)
	if err != nil {
		return fmt.Errorf("failed to open bucket for driver logs; check the object store credentials: %w", err)
	}
	defer bucket.Close()
	key := fmt.Sprintf("driver/%s-logs", logContext.LogID)
	if logContext.Execution != nil && logContext.OutputPathPrefix != "" {
		key = fmt.Sprintf("%s/%s/driver-logs", logContext.Task, logContext.OutputPathPrefix)
	}
	glog.Infof("Uploading log key: %s ...", key)
	if err := u.UploadBlob(ctx, bucket, logContext.LocalPath, key); err != nil {
		return fmt.Errorf("failed to upload driver log %q; check the object store write permissions: %w", key, err)
	}
	logContext.URI = util.GenerateOutputURI(logContext.PipelineRoot, []string{key}, false)
	if logContext.Execution == nil || logContext.Execution.TaskID == "" || logContext.KFPAPI == nil {
		return nil
	}
	if err := registerDriverLog(ctx, logContext.KFPAPI, logContext.RunID, logContext.Execution.TaskID, logContext.Namespace, logContext.URI, logContext.StoreSessionInfo); err != nil {
		return err
	}
	logContext.Registered = true
	return nil
}

// registerDriverLog establishes Artifact API ownership before exposing the URI,
// preserving task state and existing status metadata.
func registerDriverLog(ctx context.Context, api kfpapi.API, runID, taskID, namespace, uri, session string) error {
	task, err := api.GetTask(ctx, &go_client.GetTaskRequest{RunId: runID, TaskId: taskID})
	if err != nil {
		return fmt.Errorf("failed to read task %q for driver log; check the KFP API connection: %w", taskID, err)
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
		return fmt.Errorf("failed to register driver log on task %q; check task update permissions: %w", taskID, err)
	}
	return nil
}
