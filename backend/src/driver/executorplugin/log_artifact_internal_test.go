// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	"gocloud.dev/blob/memblob"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUploadDriverLogArtifact(t *testing.T) {
	for _, tc := range []struct {
		name        string
		execution   *driver.Execution
		prefix, key string
		register    bool
	}{
		{"container", &driver.Execution{TaskID: "task-id"}, "prefix", "task/prefix/driver-logs", true},
		{"dag", &driver.Execution{TaskID: "task-id"}, "", "driver/log-id-logs", true},
		{"no execution", nil, "", "driver/log-id-logs", false},
		{"no task id", &driver.Execution{}, "prefix", "task/prefix/driver-logs", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewSimpleClientset()
			api := &logMetadataAPI{task: &gc.PipelineTask{}}
			path := filepath.Join(t.TempDir(), "driver.log")
			require.NoError(t, os.WriteFile(path, []byte("driver log"), 0600))
			log := &driverLogArtifactContext{
				Execution: tc.execution, Task: "task", OutputPathPrefix: tc.prefix, LocalPath: path, LogID: "log-id",
				Namespace: "ns", PipelineRoot: "s3://bucket/root", StoreSessionInfo: `{"provider":"s3"}`, RunID: "run-id", KFPAPI: api,
			}
			bucket := memblob.OpenBucket(nil)
			uploader := logUploader{
				openBucket: func(_ context.Context, gotClient kubernetes.Interface, namespace string, cfg *objectstore.Config, session *objectstore.SessionInfo) (*blob.Bucket, error) {
					assert.Same(t, client, gotClient)
					assert.Equal(t, "ns", namespace)
					assert.Equal(t, "bucket", cfg.BucketName)
					assert.Equal(t, "s3", session.Provider)
					return bucket, nil
				},
				uploadBlob: func(ctx context.Context, bucket *blob.Bucket, path, key string) error {
					assert.Equal(t, tc.key, key)
					require.NoError(t, objectstore.UploadBlob(ctx, bucket, path, key))
					contents, err := bucket.ReadAll(ctx, key)
					require.NoError(t, err)
					assert.Equal(t, "driver log", string(contents))
					return nil
				},
			}
			require.NoError(t, uploader.uploadDriverLogArtifact(ctx, log, client))
			if tc.register {
				require.NotNil(t, api.update)
				assert.Equal(t, "run-id", api.update.RunId)
				assert.Equal(t, "task-id", api.update.TaskId)
				assert.Equal(t, "s3://bucket/root/"+tc.key, api.update.Task.StatusMetadata.CustomProperties["driver_logs_uri"].GetStringValue())
				assert.Equal(t, log.StoreSessionInfo, api.update.Task.StatusMetadata.CustomProperties["store_session_info"].GetStringValue())
			} else {
				assert.Nil(t, api.update)
			}
			_, err := bucket.ReadAll(ctx, tc.key)
			require.Error(t, err, "bucket must be closed after publication")
		})
	}
}

func TestUploadDriverLogArtifactErrors(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		mutate        func(*driverLogArtifactContext, *logMetadataAPI, *logUploader)
		cause         error
	}{
		{"session", "session info", func(log *driverLogArtifactContext, _ *logMetadataAPI, _ *logUploader) { log.StoreSessionInfo = "{" }, nil},
		{"bucket config", "bucket config", func(log *driverLogArtifactContext, _ *logMetadataAPI, _ *logUploader) { log.PipelineRoot = "invalid" }, nil},
		{"open", "open bucket", func(_ *driverLogArtifactContext, _ *logMetadataAPI, u *logUploader) {
			u.openBucket = func(context.Context, kubernetes.Interface, string, *objectstore.Config, *objectstore.SessionInfo) (*blob.Bucket, error) {
				return nil, assert.AnError
			}
		}, assert.AnError},
		{"upload", "upload log", func(_ *driverLogArtifactContext, _ *logMetadataAPI, u *logUploader) {
			u.uploadBlob = func(context.Context, *blob.Bucket, string, string) error { return assert.AnError }
		}, assert.AnError},
		{"get task", "read task", func(_ *driverLogArtifactContext, api *logMetadataAPI, _ *logUploader) { api.getErr = assert.AnError }, assert.AnError},
		{"update task", "register driver log", func(_ *driverLogArtifactContext, api *logMetadataAPI, _ *logUploader) { api.updateErr = assert.AnError }, assert.AnError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &logMetadataAPI{task: &gc.PipelineTask{}}
			log := &driverLogArtifactContext{PipelineRoot: "s3://bucket", StoreSessionInfo: "{}", Execution: &driver.Execution{TaskID: "task-id"}, KFPAPI: api}
			var bucket *blob.Bucket
			uploader := logUploader{
				openBucket: func(context.Context, kubernetes.Interface, string, *objectstore.Config, *objectstore.SessionInfo) (*blob.Bucket, error) {
					bucket = memblob.OpenBucket(nil)
					return bucket, nil
				},
				uploadBlob: func(context.Context, *blob.Bucket, string, string) error { return nil },
			}
			tc.mutate(log, api, &uploader)
			err := uploader.uploadDriverLogArtifact(context.Background(), log, fake.NewSimpleClientset())
			require.ErrorContains(t, err, tc.message)
			if tc.cause != nil {
				assert.ErrorIs(t, err, tc.cause)
			}
			if tc.name != "update task" {
				assert.Nil(t, api.update, "must not register an unsuccessful upload")
			}
			if bucket != nil {
				_, err = bucket.ReadAll(context.Background(), "anything")
				require.ErrorContains(t, err, "closed")
			}
		})
	}
	uploader := logUploader{} // neither case may touch object storage
	require.Error(t, uploader.uploadDriverLogArtifact(context.Background(), nil, nil))
	require.NoError(t, uploader.uploadDriverLogArtifact(context.Background(), &driverLogArtifactContext{}, nil))
}
