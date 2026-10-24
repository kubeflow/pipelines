package poc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDockerConcurrentAttemptsIsolateFixedParameterAndCustomArtifactPaths(t *testing.T) {
	root := t.TempDir()
	var requests []TaskExecutionRequest
	for _, attempt := range []string{"first", "second"} {
		artifactRoot := filepath.Join(root, "run", "artifact-local", attempt)
		custom := "/tmp/custom/data.txt"
		preparation := &TaskPreparationRequest{
			Run: &model.Run{UUID: "run"}, ArtifactLocalRoot: artifactRoot,
			Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "alpine:3.23"},
			ExecutorInput: &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
				OutputFile: filepath.Join(artifactRoot, "minio", "bucket", "metadata.json"),
				Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"Output": {OutputFile: "/tmp/kfp/outputs/Output"}},
				Artifacts: map[string]*pipelinespec.ArtifactList{"dataset": {Artifacts: []*pipelinespec.RuntimeArtifact{{
					Uri: "minio://bucket/" + attempt + "/dataset", CustomPath: &custom,
				}}}},
			}},
		}
		require.NoError(t, PrepareDockerTask(context.Background(), preparation))
		request := TaskExecutionRequest{
			Identity:          TaskAttemptIdentity{RunID: "run", TaskID: attempt, AttemptID: attempt},
			ArtifactLocalRoot: artifactRoot, Container: preparation.Container, ExecutorInput: preparation.ExecutorInput,
		}
		mounts := dockerMounts(&request)
		assert.Contains(t, mounts, bindMount(filepath.Join(artifactRoot, "docker-tmp"), "/tmp"))
		assert.Contains(t, mounts, bindMount(artifactRoot, artifactRoot))
		for _, mount := range mounts {
			assert.NotEqual(t, "/tmp/kfp/outputs", strings.Split(mount, ":")[0])
			assert.NotEqual(t, "/tmp/kfp/outputs", strings.Split(mount, ":")[1])
		}
		assert.Equal(t, filepath.Join(artifactRoot, "docker-tmp", "kfp", "outputs", "Output"),
			request.ExecutorInput.GetOutputs().GetParameters()["Output"].GetOutputFile())
		artifact := request.ExecutorInput.GetOutputs().GetArtifacts()["dataset"].Artifacts[0]
		publicationPath, err := resolvedRuntimeArtifactLocalPath(artifactRoot, dockerTempMountSource(&request), artifact)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(artifactRoot, "docker-tmp", "custom", "data.txt"), publicationPath)
		assert.Equal(t, publicationPath, artifact.GetCustomPath())
		requests = append(requests, request)
	}
	assert.NotEqual(t, dockerTempMountSource(&requests[0]), dockerTempMountSource(&requests[1]))
	var writes sync.WaitGroup
	errors := make(chan error, len(requests)*2)
	for index := range requests {
		request := requests[index]
		writes.Add(1)
		go func() {
			defer writes.Done()
			value := []byte(request.Identity.AttemptID)
			errors <- os.WriteFile(request.ExecutorInput.GetOutputs().GetParameters()["Output"].GetOutputFile(), value, 0o600)
			errors <- os.WriteFile(request.ExecutorInput.GetOutputs().GetArtifacts()["dataset"].Artifacts[0].GetCustomPath(), value, 0o600)
		}()
	}
	writes.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	for _, request := range requests {
		values, err := collectOutputParameters(request.ExecutorInput, nil)
		require.NoError(t, err)
		assert.Equal(t, request.Identity.AttemptID, values["Output"].GetStringValue())
		artifact := request.ExecutorInput.GetOutputs().GetArtifacts()["dataset"].Artifacts[0]
		data, err := os.ReadFile(artifact.GetCustomPath())
		require.NoError(t, err)
		assert.Equal(t, request.Identity.AttemptID, string(data))
	}
}

func TestDockerPreparationRetryPreservesInputBlobURI(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	require.NoError(t, os.WriteFile(blocked, []byte("not a directory"), 0o600))
	store := component.NewMockObjectStoreClient()
	store.DownloadFunc = func(_ context.Context, uri, path, key string) error {
		assert.Equal(t, "minio://bucket/producer/dataset", uri)
		return os.WriteFile(path, []byte("downloaded"), 0o600)
	}
	preparation := &TaskPreparationRequest{
		Run: &model.Run{UUID: "run"}, ArtifactLocalRoot: root, ObjectStore: store,
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "alpine:3.23"},
		ExecutorInput: &pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{Artifacts: map[string]*pipelinespec.ArtifactList{
				"dataset": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: "minio://bucket/producer/dataset", CustomPath: ptr("/producer/local/path")}}},
			}},
			Outputs: &pipelinespec.ExecutorInput_Outputs{Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
				"Output": {OutputFile: filepath.Join(blocked, "Output")},
			}},
		},
	}
	require.Error(t, PrepareDockerTask(context.Background(), preparation), "fail after downloading, while preparing output directories")
	artifact := preparation.ExecutorInput.GetInputs().GetArtifacts()["dataset"].Artifacts[0]
	assert.Equal(t, "minio://bucket/producer/dataset", artifact.GetUri(), "a failed preparation must retain the durable source URI")
	require.NoError(t, os.Remove(blocked))
	require.NoError(t, PrepareDockerTask(context.Background(), preparation))
	assert.Equal(t, "minio://bucket/producer/dataset", artifact.GetUri())
	assert.Equal(t, filepath.Join(root, "minio", "bucket", "producer", "dataset"), artifact.GetCustomPath())
	assert.Len(t, store.DownloadCalls, 1, "same-attempt retry may reuse already staged bytes without changing source identity")
	_, args, err := compileDockerCommandAndArgs(preparation.ExecutorInput, []string{"python"}, []string{"{{$}}"})
	require.NoError(t, err)
	compiled := &pipelinespec.ExecutorInput{}
	require.NoError(t, protojson.Unmarshal([]byte(args[0]), compiled))
	assert.Equal(t, artifact.GetCustomPath(), compiled.GetInputs().GetArtifacts()["dataset"].Artifacts[0].GetUri())
	assert.Equal(t, "minio://bucket/producer/dataset", artifact.GetUri(), "legacy compilation adaptation must not mutate persisted request data")
}

func TestDockerOutputStagingRequiresAttemptRoot(t *testing.T) {
	preparation := &TaskPreparationRequest{
		Run: &model.Run{UUID: "run"}, Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "alpine:3.23"},
		ExecutorInput: &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
			Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"Output": {OutputFile: "/tmp/kfp/outputs/Output"}},
		}},
	}
	require.ErrorContains(t, PrepareDockerTask(context.Background(), preparation), "attempt-local artifact root")
}

func TestDockerInputArtifactsUseConsumerStagingAndPreserveWorkspacePaths(t *testing.T) {
	root := t.TempDir()
	producerPath := filepath.Join(root, "producer", "dataset")
	require.NoError(t, os.MkdirAll(filepath.Dir(producerPath), 0o755))
	require.NoError(t, os.WriteFile(producerPath, []byte("producer-local stale bytes"), 0o600))
	workspacePath := component.WorkspaceMountPath + "/shared/dataset"
	workspaceRoot := filepath.Join(root, "workspace")
	sharedFile := filepath.Join(workspaceRoot, "shared", "dataset")
	require.NoError(t, os.MkdirAll(filepath.Dir(sharedFile), 0o755))
	require.NoError(t, os.WriteFile(sharedFile, []byte("shared workspace bytes"), 0o600))
	workspaceHandle := &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath, Metadata: map[string]string{"path": workspaceRoot}}
	input := &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"dataset": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: "minio://bucket/producer/dataset", CustomPath: &producerPath}}},
			"workspace": {Artifacts: []*pipelinespec.RuntimeArtifact{{
				Uri: workspacePath, CustomPath: &workspacePath,
				Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{"_kfp_workspace": structpb.NewBoolValue(true)}},
			}}},
		},
	}}
	for _, attempt := range []string{"consumer-one", "consumer-two"} {
		artifactRoot := filepath.Join(root, attempt)
		local := filepath.Join(artifactRoot, "minio", "bucket", "producer", "dataset")
		store := component.NewMockObjectStoreClient()
		store.DownloadFunc = func(_ context.Context, uri, path, key string) error {
			assert.Equal(t, "minio://bucket/producer/dataset", uri)
			assert.Equal(t, "dataset", key)
			assert.Equal(t, local, path)
			return os.WriteFile(path, []byte("published "+attempt), 0o600)
		}
		preparation := &TaskPreparationRequest{
			Run: &model.Run{UUID: "run"}, ArtifactLocalRoot: artifactRoot, ObjectStore: store,
			Container:     &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "alpine:3.23"},
			ExecutorInput: proto.Clone(input).(*pipelinespec.ExecutorInput),
			Workspace:     workspaceHandle,
		}
		require.NoError(t, PrepareDockerTask(context.Background(), preparation))
		assert.Len(t, store.DownloadCalls, 1, "workspace bytes must never be downloaded or producer-local bytes reused")
		artifact := preparation.ExecutorInput.GetInputs().GetArtifacts()["dataset"].Artifacts[0]
		assert.Equal(t, local, artifact.GetCustomPath())
		assert.Equal(t, "minio://bucket/producer/dataset", artifact.GetUri())
		bytes, err := os.ReadFile(local)
		require.NoError(t, err)
		assert.Equal(t, "published "+attempt, string(bytes))
		workspace := preparation.ExecutorInput.GetInputs().GetArtifacts()["workspace"].Artifacts[0]
		assert.Equal(t, workspacePath, workspace.GetCustomPath())
		assert.Equal(t, workspacePath, workspace.GetUri())
		mounts := dockerMounts(&TaskExecutionRequest{ArtifactLocalRoot: artifactRoot, ExecutorInput: preparation.ExecutorInput, Workspace: workspaceHandle})
		assert.Contains(t, mounts, bindMount(workspaceRoot, component.WorkspaceMountPath))
		for _, mount := range mounts {
			assert.False(t, strings.HasPrefix(mount, filepath.Dir(producerPath)+":"))
		}
	}
	original, err := os.ReadFile(producerPath)
	require.NoError(t, err)
	assert.Equal(t, "producer-local stale bytes", string(original))
	shared, err := os.ReadFile(sharedFile)
	require.NoError(t, err)
	assert.Equal(t, "shared workspace bytes", string(shared))
}
