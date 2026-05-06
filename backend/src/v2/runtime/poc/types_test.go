package poc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type recordingPodClient struct {
	apiserverclient.FakePodClient
	pods        map[string]*corev1.Pod
	createCount int
}

func (c *recordingPodClient) Create(_ context.Context, pod *corev1.Pod, _ metav1.CreateOptions) (*corev1.Pod, error) {
	c.createCount++
	created := pod.DeepCopy()
	created.UID = "pod-uid-123"
	created.Status.Phase = corev1.PodSucceeded
	c.pods[created.Name] = created
	return created, nil
}

func (c *recordingPodClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Pod, error) {
	return c.pods[name], nil
}

type recordingCoreClient struct {
	podClient typedcorev1.PodInterface
}

func (c *recordingCoreClient) PodClient(namespace string) typedcorev1.PodInterface {
	return c.podClient
}

func (c *recordingCoreClient) GetClientSet() kubernetes.Interface {
	return nil
}

func TestKubernetesExecutorCreatesAndReattachesPods(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	podClient := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: podClient}}

	recordedHandles := 0
	task := &model.Task{
		UUID:  "task-12345678",
		Name:  "hello-world",
		State: model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
	}
	request := &TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
		},
		Task:          task,
		ComponentSpec: &pipelinespec.ComponentSpec{},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image:   "python:3.11",
			Command: []string{"python"},
			Args:    []string{"-c", "print('hello')"},
		},
		ExecutorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{},
		},
		PipelineName:    "hello-pipeline",
		TaskDisplayName: "hello-world",
	}
	request.HandleRecorder = func(pods model.JSONSlice, metadata model.JSONData) error {
		recordedHandles++
		task.Pods = pods
		task.StatusMetadata = metadata
		return nil
	}

	result, err := executor.ExecuteTask(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, podClient.createCount)
	assert.Equal(t, 1, recordedHandles)

	reattachResult, err := executor.ExecuteTask(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, reattachResult)
	assert.Equal(t, 1, podClient.createCount, "reattach should reuse the existing pod handle")
}

func TestEnsureContainerWritableDirMakesDirectoryWorldWritable(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "nested", "outputs")

	err := ensureContainerWritableDir(target)
	require.NoError(t, err)

	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0o777), info.Mode().Perm())
}

func TestResetContainerWritableOutputPathRemovesStaleFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "outputs", "result.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("stale"), 0o600))

	err := resetContainerWritableOutputPath(target)
	require.NoError(t, err)

	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr))

	dirInfo, err := os.Stat(filepath.Dir(target))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o777), dirInfo.Mode().Perm())
}

func TestDockerMountsIncludesCustomArtifactOutputDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	customPath := filepath.Join(t.TempDir(), "custom-artifacts", "out_dataset")
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	mounts := dockerMounts(&TaskExecutionRequest{
		ArtifactLocalRoot: artifactRoot,
		ExecutorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"out_dataset": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							CustomPath: &customPath,
						}},
					},
				},
			},
		},
	})

	tempMountSource := dockerTempMountSourceForArtifactRoot(artifactRoot)
	expectedMount := bindMount(
		filepath.Join(tempMountSource, strings.TrimPrefix(filepath.Dir(customPath), filepath.Clean(os.TempDir())+string(filepath.Separator))),
		filepath.Join(tempMountSource, strings.TrimPrefix(filepath.Dir(customPath), filepath.Clean(os.TempDir())+string(filepath.Separator))),
	)
	assert.Contains(t, mounts, expectedMount)
}

func TestDockerMountsIncludesDedicatedTempDirForRuntimeCustomPaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	mounts := dockerMounts(&TaskExecutionRequest{
		Run: &model.Run{UUID: "run-123"},
	})
	assert.Contains(
		t,
		mounts,
		bindMount(filepath.Join(homeDir, ".cache", "kfp-v2-runtime", "run-123", "docker-tmp"), os.TempDir()),
	)
}

func TestDockerMountsIncludesLocalSDKRootWhenConfigured(t *testing.T) {
	localSDKRoot := t.TempDir()
	t.Setenv(localSDKRootEnvVar, localSDKRoot)

	mounts := dockerMounts(&TaskExecutionRequest{})

	assert.Contains(t, mounts, bindMount(localSDKRoot, localSDKMountPath))
}

func TestDockerMountsIncludesFileSchemeArtifactRoot(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	mounts := dockerMounts(&TaskExecutionRequest{
		ArtifactLocalRoot: artifactRoot,
	})

	assert.Contains(t, mounts, bindMount(filepath.Join(artifactRoot, "file"), "/file"))
}

func TestResolvedRuntimeArtifactLocalPathUsesArtifactRootForURIArtifacts(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	localPath, err := resolvedRuntimeArtifactLocalPath(artifactRoot, "", &pipelinespec.RuntimeArtifact{
		Uri: "minio://bucket/path/to/artifact",
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Join(artifactRoot, "minio", "bucket", "path", "to", "artifact"),
		localPath,
	)
}

func TestResolvedRuntimeArtifactLocalPathUsesArtifactRootForFileURIArtifacts(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	localPath, err := resolvedRuntimeArtifactLocalPath(artifactRoot, "", &pipelinespec.RuntimeArtifact{
		Uri: "file:///tmp/kfp-artifacts/path/to/artifact",
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Join(artifactRoot, "file", "tmp", "kfp-artifacts", "path", "to", "artifact"),
		localPath,
	)
}

func TestResolvedRuntimeArtifactLocalPathInsideArtifactRootEnvDoesNotDeadlock(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	var path string
	err := withArtifactLocalRootEnv(artifactRoot, func() error {
		resolvedPath, resolveErr := resolvedRuntimeArtifactLocalPath(artifactRoot, "", &pipelinespec.RuntimeArtifact{
			Uri: "minio://bucket/path/to/artifact",
		})
		if resolveErr != nil {
			return resolveErr
		}
		path = resolvedPath
		return nil
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Join(artifactRoot, "minio", "bucket", "path", "to", "artifact"),
		path,
	)
}

func TestResolvedRuntimeArtifactLocalPathMapsCustomTmpPathToHostTempMount(t *testing.T) {
	tempMountSource := filepath.Join(t.TempDir(), "docker-tmp")
	localPath, err := resolvedRuntimeArtifactLocalPath("", tempMountSource, &pipelinespec.RuntimeArtifact{
		CustomPath: ptr("/tmp/out_dataset"),
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempMountSource, "out_dataset"), localPath)
}

func TestResolvedRuntimeArtifactLocalPathKeepsArtifactRootLocalPath(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	localPath, err := resolvedRuntimeArtifactLocalPath(
		artifactRoot,
		filepath.Join(t.TempDir(), "docker-tmp"),
		&pipelinespec.RuntimeArtifact{CustomPath: ptr(filepath.Join(artifactRoot, "minio", "bucket", "artifact"))},
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(artifactRoot, "minio", "bucket", "artifact"), localPath)
}

func TestNormalizeInputArtifactPathsForDockerUsesCustomPathAsURI(t *testing.T) {
	customPath := "/tmp/out_dataset"
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"in_dataset": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Uri:        "minio://mlpipeline/path/to/artifact",
						CustomPath: &customPath,
					}},
				},
			},
		},
	}

	normalizeInputArtifactPathsForDocker(executorInput)

	artifact := executorInput.GetInputs().GetArtifacts()["in_dataset"].Artifacts[0]
	assert.Equal(t, customPath, artifact.GetUri())
	assert.Equal(t, customPath, artifact.GetCustomPath())
}

func TestNormalizeOutputArtifactPathsForDockerSetsCustomPathFromURI(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"out_dataset": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Uri: "file:///tmp/kfp-artifacts/path/to/artifact",
					}},
				},
			},
		},
	}

	normalizeOutputArtifactPathsForDocker(executorInput, artifactRoot, filepath.Join(t.TempDir(), "docker-tmp"))

	artifact := executorInput.GetOutputs().GetArtifacts()["out_dataset"].Artifacts[0]
	assert.Equal(
		t,
		filepath.Join(artifactRoot, "file", "tmp", "kfp-artifacts", "path", "to", "artifact"),
		artifact.GetCustomPath(),
	)
}

func TestArtifactLocalRootFromOutputFile(t *testing.T) {
	outputFile := "/tmp/root/minio/bucket/path/output_metadata.json"
	assert.Equal(t, "/tmp/root", artifactLocalRootFromOutputFile(outputFile))
	assert.Equal(t, "/tmp/root", artifactLocalRootFromOutputFile("/tmp/root/file/tmp/kfp-artifacts/output_metadata.json"))
	assert.Equal(t, "", artifactLocalRootFromOutputFile("/tmp/output_metadata.json"))
}

func TestDockerTempMountSourcePrefersArtifactRoot(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	assert.Equal(
		t,
		filepath.Join(homeDir, ".cache", "kfp-v2-runtime", "run-123", "docker-tmp"),
		dockerTempMountSource(&TaskExecutionRequest{Run: &model.Run{UUID: "run-123"}}),
	)
}

func ptr(value string) *string {
	return &value
}
