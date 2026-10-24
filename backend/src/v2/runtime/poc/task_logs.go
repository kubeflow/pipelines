package poc

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
)

type taskLogWriter struct {
	io.Writer
	written bool
}

func (w *taskLogWriter) Write(data []byte) (int, error) {
	count, err := w.Writer.Write(data)
	w.written = w.written || count > 0
	return count, err
}

// ReadTaskLog resolves a run-owned task's active or published log source.
func (c *Coordinator) ReadTaskLog(ctx context.Context, run *model.Run, taskID string, follow bool, dst io.Writer) error {
	if run == nil || run.UUID == "" || taskID == "" {
		return util.NewInvalidInputError("Run and task identities are required for task logs")
	}
	if c.taskStore == nil {
		return fmt.Errorf("task store is not configured for log retrieval")
	}
	task, err := c.taskStore.GetTask(taskID)
	if err != nil {
		return err
	}
	if task.RunUUID != run.UUID || task.Namespace != run.Namespace {
		return util.NewInvalidInputError("Task %s does not belong to run %s", taskID, run.UUID)
	}
	if run.ExecutorType == nil {
		return util.NewInvalidInputError("Run %s has no persisted executor", run.UUID)
	}
	executor, ok := c.executors[*run.ExecutorType]
	if !ok {
		return util.NewInvalidInputError("Executor %s is unavailable for run %s", *run.ExecutorType, run.UUID)
	}
	reader, ok := executor.(ExecutionLogReader)
	if !ok {
		return util.NewInvalidInputError("Executor %s does not provide task logs", executor.Type())
	}
	writer := &taskLogWriter{Writer: dst}
	var backendErr error
	if c.runtimeStore != nil {
		attempt, err := c.runtimeStore.Get(taskID)
		if err != nil && !runtimeTaskNotFound(err) {
			return err
		}
		if err == nil {
			reference, err := taskLogReference(ctx, executor, attempt)
			if err != nil {
				backendErr = err
			} else if reference != nil {
				if err := reader.ReadLogs(ctx, *reference, follow, writer); err == nil {
					return nil
				} else if writer.written || ctx.Err() != nil {
					return err
				} else {
					backendErr = err
				}
			}
		}
	}
	if available, err := c.readPublishedTaskLogs(ctx, run, task, writer); available || err != nil {
		return err
	}
	if backendErr != nil {
		return backendErr
	}
	if executor.Type() == ExecutorKubernetes {
		reference, err := publicPodLogReference(run, task)
		if err != nil {
			return err
		}
		return reader.ReadLogs(ctx, reference, follow, writer)
	}
	return util.NewResourceNotFoundError("available logs for task", taskID)
}

func taskLogReference(ctx context.Context, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask) (*ExecutionReference, error) {
	if len(attempt.ExecutionReference) > 0 {
		var reference ExecutionReference
		if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
			return nil, err
		}
		return &reference, nil
	}
	if attempt.State == model.RuntimeTaskSubmitting || attempt.State == model.RuntimeTaskRunning {
		var request TaskExecutionRequest
		if err := fromJSONData(attempt.ExecutionRequest, &request); err != nil {
			return nil, err
		}
		reference, _, err := recoverAttempt(ctx, executor, request.Identity, nil)
		if err != nil {
			return nil, err
		}
		if reference.Executor != "" {
			return &reference, nil
		}
	}
	return nil, nil
}

func publicPodLogReference(run *model.Run, task *model.Task) (ExecutionReference, error) {
	pods, err := model.JSONSliceToProtoSlice(task.Pods, func() *apiv2.PipelineTask_TaskPod {
		return &apiv2.PipelineTask_TaskPod{}
	})
	if err != nil {
		return ExecutionReference{}, err
	}
	properties, _ := task.StatusMetadata["customProperties"].(map[string]interface{})
	attemptID, _ := properties["attemptID"].(string)
	for _, pod := range pods {
		if pod.GetType() != apiv2.PipelineTask_EXECUTOR || pod.GetName() == "" || pod.GetUid() == "" || attemptID == "" {
			continue
		}
		return ExecutionReference{
			Executor: ExecutorKubernetes, Version: 1,
			Identity: TaskAttemptIdentity{RunID: run.UUID, TaskID: task.UUID, AttemptID: attemptID, Namespace: run.Namespace},
			Data:     model.JSONData{"namespace": run.Namespace, "podName": pod.GetName(), "podUID": pod.GetUid()},
		}, nil
	}
	return ExecutionReference{}, util.NewResourceNotFoundError("available logs for task", task.UUID)
}

func (c *Coordinator) readPublishedTaskLogs(ctx context.Context, run *model.Run, task *model.Task, dst io.Writer) (bool, error) {
	if c.artifactTaskStore == nil || c.artifactStore == nil {
		return false, nil
	}
	output := model.IOType(apiv2.IOType_OUTPUT)
	links, _, _, err := c.artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
		{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: task.UUID}},
		{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
	}, &output, list.EmptyOptions())
	if err != nil {
		return false, err
	}
	var latest *model.Artifact
	for _, link := range links {
		if link.ArtifactKey != "executor-logs" {
			continue
		}
		artifact, err := c.artifactStore.GetArtifact(link.ArtifactID)
		if err != nil {
			return false, err
		}
		if artifact.Namespace != run.Namespace {
			return false, util.NewInvalidInputError("Log artifact does not belong to run namespace %s", run.Namespace)
		}
		if latest == nil || artifact.CreatedAtInSec > latest.CreatedAtInSec {
			latest = artifact
		}
	}
	if latest == nil || latest.URI == nil {
		return false, nil
	}
	if err := validateTaskLogURI(*latest.URI); err != nil {
		return true, err
	}
	uri, err := url.Parse(*latest.URI)
	if err != nil {
		return true, err
	}
	if uri.Scheme == "file" {
		return true, readFilesystemTaskLog(ctx, uri, dst)
	}
	directory, err := os.MkdirTemp("", "kfp-task-logs-")
	if err != nil {
		return true, err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "logs")
	objectStore, err := c.newObjectStoreClient(ctx, run.Namespace)
	if err != nil {
		return true, err
	}
	if err := objectStore.DownloadArtifact(ctx, *latest.URI, path, "executor-logs"); err != nil {
		return true, err
	}
	file, err := os.Open(path)
	if err != nil {
		return true, err
	}
	defer file.Close()
	_, err = io.Copy(dst, file)
	return true, err
}

func validateTaskLogURI(raw string) error {
	uri, err := url.Parse(raw)
	if err != nil || uri.RawQuery != "" || uri.Fragment != "" {
		return util.NewInvalidInputError("Task log artifact URI is invalid")
	}
	switch uri.Scheme {
	case "minio", "s3", "gs":
		return nil
	case "file":
		_, _, err := filesystemTaskLogLocation(uri)
		return err
	default:
		return fmt.Errorf("unsupported task log artifact scheme %q", uri.Scheme)
	}
}

func filesystemTaskLogLocation(uri *url.URL) (string, string, error) {
	root, err := url.Parse(common.GetStringConfigWithDefault("ObjectStoreConfig.BucketURL", ""))
	if err != nil || root.Scheme != "file" || root.Path == "" || uri.Host != root.Host {
		return "", "", util.NewInvalidInputError("Filesystem task logs require the configured file artifact root")
	}
	rootPath, err := filepath.EvalSymlinks(root.Path)
	if err != nil {
		return "", "", err
	}
	logPath, err := filepath.EvalSymlinks(uri.Path)
	if err != nil {
		return "", "", err
	}
	relative, err := filepath.Rel(rootPath, logPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", util.NewInvalidInputError("Task log artifact is outside the configured artifact root")
	}
	info, err := os.Stat(logPath)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", util.NewInvalidInputError("Task log artifact is not a regular file")
	}
	root.Path, root.RawPath = rootPath, ""
	return root.String(), filepath.ToSlash(relative), nil
}

func readFilesystemTaskLog(ctx context.Context, uri *url.URL, dst io.Writer) error {
	root, key, err := filesystemTaskLogLocation(uri)
	if err != nil {
		return err
	}
	bucket, err := blob.OpenBucket(ctx, root)
	if err != nil {
		return err
	}
	defer bucket.Close()
	reader, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(dst, reader)
	return err
}
