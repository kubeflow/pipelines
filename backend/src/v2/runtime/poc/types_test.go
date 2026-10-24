package poc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type recordingPodClient struct {
	apiserverclient.FakePodClient
	pods        map[string]*corev1.Pod
	createCount int
	deleteCount int
	getErr      error
}

func (c *recordingPodClient) Create(_ context.Context, pod *corev1.Pod, _ metav1.CreateOptions) (*corev1.Pod, error) {
	if c.pods[pod.Name] != nil {
		return nil, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "pods"}, pod.Name)
	}
	c.createCount++
	created := pod.DeepCopy()
	created.UID = "pod-uid-123"
	created.Status.Phase = corev1.PodSucceeded
	c.pods[created.Name] = created
	return created, nil
}

func (c *recordingPodClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Pod, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	pod := c.pods[name]
	if pod == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
	}
	return pod.DeepCopy(), nil
}

func (c *recordingPodClient) Delete(_ context.Context, name string, options metav1.DeleteOptions) error {
	pod := c.pods[name]
	if pod == nil {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
	}
	if options.Preconditions == nil || options.Preconditions.UID == nil || *options.Preconditions.UID != pod.UID {
		return errors.New("delete must be UID guarded")
	}
	c.deleteCount++
	delete(c.pods, name)
	return nil
}

type recordingCoreClient struct {
	podClient typedcorev1.PodInterface
	clientSet kubernetes.Interface
}

func (c *recordingCoreClient) PodClient(namespace string) typedcorev1.PodInterface {
	if c.podClient == nil && c.clientSet != nil {
		return c.clientSet.CoreV1().Pods(namespace)
	}
	return c.podClient
}

func (c *recordingCoreClient) GetClientSet() kubernetes.Interface {
	return c.clientSet
}

func TestTaskExecutionRequestDurableJSON(t *testing.T) {
	request := TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt", Namespace: "ns"},
		Namespace: "ns", TaskName: "hello",
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python"},
		ExecutorInput: &pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{ParameterValues: map[string]*structpb.Value{
				"number": structpb.NewNumberValue(3.5), "string": structpb.NewStringValue("hello"), "bool": structpb.NewBoolValue(true),
			}},
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"result": {OutputFile: "output.txt"}},
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{
						Uri: "file:///executor-logs",
						Type: &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{
							InstanceSchema: "title: system.Artifact\ntype: object\n",
						}, SchemaVersion: "0.0.1"},
					}}},
				},
			},
		},
		OutputParameterTypes: map[string]pipelinespec.ParameterType_ParameterTypeEnum{"result": pipelinespec.ParameterType_STRING},
		Workspace:            &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath, Metadata: map[string]string{"path": "workspace"}},
		ArtifactLocalRoot:    "artifacts",
		Pod:                  &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}},
	}
	raw, err := json.Marshal(request)
	require.NoError(t, err)
	decoded := TaskExecutionRequest{}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.True(t, proto.Equal(request.Container, decoded.Container))
	assert.True(t, proto.Equal(request.ExecutorInput, decoded.ExecutorInput))
	decoded.Container, decoded.ExecutorInput = request.Container, request.ExecutorInput
	assert.Equal(t, request, decoded)
	assert.NotContains(t, string(raw), "HandleRecorder")
	assert.NotContains(t, string(raw), "ObjectStore")
}

func TestKubernetesExecutorRecoveryAndExplicitCancellation(t *testing.T) {
	ctx := context.Background()
	client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}, namespace: "ns"}
	identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"}
	request := TaskExecutionRequest{Identity: identity, Namespace: "ns", Pod: &corev1.Pod{}}
	reference, err := executor.Submit(ctx, request)
	require.NoError(t, err)
	recovery, err := executor.Recover(ctx, identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
	assert.Equal(t, TaskOutcomeSuccess, recovery.Result.Outcome)
	assert.Equal(t, reference, *recovery.Reference)

	pod := client.pods[attemptWorkloadName(identity)]
	pod.Status.Phase = corev1.PodRunning
	recovery, err = executor.Recover(ctx, identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryExisting, recovery.State)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	result, err := executor.Wait(canceled, reference)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, result.Outcome, "an interrupted observer must not become a workload failure")
	assert.Zero(t, client.deleteCount)
	require.NoError(t, executor.Cancel(ctx, reference))
	require.NoError(t, executor.Cancel(ctx, reference))
	assert.Equal(t, 1, client.deleteCount)
	recovery, err = executor.Recover(ctx, identity, &reference)
	require.NoError(t, err)
	assert.Equal(t, RecoveryStopped, recovery.State)
}

func TestKubernetesExecutorRecoversInAttemptNamespaceWithoutReference(t *testing.T) {
	identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"}
	name := attemptWorkloadName(identity)
	client := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "execution-one", UID: "uid-one", Labels: attemptLabels(identity),
		}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "execution-two", UID: "uid-two", Labels: attemptLabels(identity),
		}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
	)
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{clientSet: client}, namespace: "worker-namespace"}
	for _, namespace := range []string{"execution-one", "execution-two"} {
		t.Run(namespace, func(t *testing.T) {
			attempt := identity
			require.NoError(t, json.Unmarshal([]byte(`{"namespace":"`+namespace+`"}`), &attempt))
			recovery, err := executor.Recover(context.Background(), attempt, nil)
			require.NoError(t, err)
			require.Equal(t, RecoveryCompleted, recovery.State)
			require.NotNil(t, recovery.Reference)
			assert.Equal(t, namespace, recovery.Reference.Data["namespace"])
			assert.Equal(t, attempt, recovery.Reference.Identity)
			assert.Equal(t, TaskOutcomeSuccess, recovery.Result.Outcome)
		})
	}
}

func TestKubernetesExecutorRejectsInconsistentAttemptNamespaces(t *testing.T) {
	client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
	identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt", Namespace: "execution"}
	_, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "different", Pod: &corev1.Pod{}})
	require.ErrorContains(t, err, "attempt namespace")
	assert.Zero(t, client.createCount)
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "execution", Pod: &corev1.Pod{}})
	require.NoError(t, err)
	reference.Data["namespace"] = "different"
	_, err = executor.Wait(context.Background(), reference)
	require.ErrorContains(t, err, "attempt namespace")
	require.ErrorContains(t, executor.Cancel(context.Background(), reference), "attempt namespace")
	recovery, err := executor.Recover(context.Background(), identity, &reference)
	require.ErrorContains(t, err, "attempt namespace")
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Zero(t, client.deleteCount)
}

func TestKubernetesExecutorIdentityAndUIDGuards(t *testing.T) {
	client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}, namespace: "ns"}
	identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "ns", Pod: &corev1.Pod{}})
	require.NoError(t, err)
	pod := client.pods[attemptWorkloadName(identity)]
	pod.UID = "replacement"
	_, err = executor.Wait(context.Background(), reference)
	require.ErrorContains(t, err, "UID")
	require.ErrorContains(t, executor.Cancel(context.Background(), reference), "UID")
	recovery, err := executor.Recover(context.Background(), identity, &reference)
	require.Error(t, err)
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Zero(t, client.deleteCount)
	pod.Labels[attemptIDLabel] = "another-attempt"
	_, err = executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "ns", Pod: &corev1.Pod{}})
	require.ErrorContains(t, err, "identity mismatch")
	client.getErr = errors.New("API unavailable")
	recovery, err = executor.Recover(context.Background(), identity, nil)
	require.Error(t, err)
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Equal(t, 1, client.createCount)
}

func TestKubernetesExecutorClassifiesTerminalFailure(t *testing.T) {
	for _, test := range []struct {
		reason    string
		outcome   TaskOutcome
		transient bool
	}{
		{"Error", TaskOutcomeFailure, false},
		{"Evicted", TaskOutcomeError, true},
	} {
		t.Run(test.reason, func(t *testing.T) {
			client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
			executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
			identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"}
			reference, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "ns", Pod: &corev1.Pod{}})
			require.NoError(t, err)
			pod := client.pods[attemptWorkloadName(identity)]
			pod.Status = corev1.PodStatus{Phase: corev1.PodFailed, Reason: test.reason, Message: "failed"}
			result, err := executor.Wait(context.Background(), reference)
			require.NoError(t, err)
			assert.Equal(t, test.outcome, result.Outcome)
			assert.Equal(t, test.transient, result.Failure.Transient)
			assert.Len(t, client.pods, 1, "terminal pods must remain available for launcher result recovery")
		})
	}

}

func TestKubernetesExecutorDoesNotClassifyPublicationAsComputationFailure(t *testing.T) {
	for _, phase := range []string{component.RuntimePhasePublishing, component.RuntimePhasePublicationFailed, component.RuntimePhasePublished} {
		t.Run(phase, func(t *testing.T) {
			client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
			executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}, namespace: "ns"}
			identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"}
			reference, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: identity, Namespace: "ns", Pod: &corev1.Pod{}})
			require.NoError(t, err)
			report := &component.RuntimeLauncherReport{Version: 1, AttemptID: identity.AttemptID, ComputationCompleted: true,
				Phase: phase, PublicationAttempts: 3, PublicationCompleted: phase == component.RuntimePhasePublished}
			raw, err := json.Marshal(report)
			require.NoError(t, err)
			pod := client.pods[attemptWorkloadName(identity)]
			pod.Status.Phase = corev1.PodFailed
			if report.PublicationCompleted {
				pod.Status.Phase = corev1.PodSucceeded
			}
			pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "main", State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Message: string(raw)},
			}}}
			result, err := executor.Wait(context.Background(), reference)
			require.NoError(t, err)
			assert.True(t, result.ComputationCompleted)
			assert.Equal(t, report.PublicationCompleted, result.PublicationCompleted)
			if report.PublicationCompleted {
				assert.Equal(t, TaskOutcomeSuccess, result.Outcome)
			} else {
				assert.Equal(t, TaskOutcomeError, result.Outcome)
				assert.Equal(t, "publication", result.Failure.Class)
			}
			recovery, err := executor.Recover(context.Background(), identity, nil)
			require.NoError(t, err)
			assert.Equal(t, RecoveryCompleted, recovery.State)
			assert.True(t, recovery.Result.ComputationCompleted)
			assert.Equal(t, result.Outcome, recovery.Result.Outcome)
			assert.Equal(t, 1, client.createCount)
			assert.Zero(t, client.deleteCount)
		})
	}
}

func TestKubernetesExecutorSeparatesAttemptsWithoutDeletingResults(t *testing.T) {
	client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}, namespace: "ns"}
	firstIdentity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "first"}
	secondIdentity := firstIdentity
	secondIdentity.AttemptID = "second"
	first, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: firstIdentity, Namespace: "ns", Pod: &corev1.Pod{}})
	require.NoError(t, err)
	second, err := executor.Submit(context.Background(), TaskExecutionRequest{Identity: secondIdentity, Namespace: "ns", Pod: &corev1.Pod{}})
	require.NoError(t, err)
	assert.NotEqual(t, first.Data["podName"], second.Data["podName"])
	assert.Len(t, client.pods, 2)
	assert.Zero(t, client.deleteCount)
	recovery, err := executor.Recover(context.Background(), firstIdentity, &first)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
}

func TestKubernetesExecutorCreatesAndReattachesPods(t *testing.T) {
	proxy.InitializeConfig("http://proxy.example:3128", "http://secure-proxy.example:3128", "custom.internal")
	t.Cleanup(proxy.InitializeConfigWithEmptyForTests)
	podClient := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: podClient}}
	task := &model.Task{UUID: "task-12345678", Name: "hello-world", State: model.TaskStatus(apiv2.PipelineTask_RUNNING)}
	preparation := &TaskPreparationRequest{
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
	pod, err := buildCoordinatorExecutorPod(preparation)
	require.NoError(t, err)
	request := TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run-id", TaskID: task.UUID, AttemptID: "attempt-one"},
		Namespace: "ns1", TaskName: task.Name, Pod: pod,
	}
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err)
	assert.Equal(t, TaskOutcomeSuccess, result.Outcome)
	assert.Equal(t, 1, podClient.createCount)
	for _, pod := range podClient.pods {
		assert.Equal(t, request.Identity.RunID, pod.Labels[util.LabelKeyWorkflowRunId])
		assert.Equal(t, request.Identity.AttemptID, pod.Labels[attemptIDLabel])
		for _, expected := range proxy.GetConfig().GetEnvVars() {
			assert.Contains(t, pod.Spec.Containers[0].Env, expected)
		}
	}

	reattachReference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, reference, reattachReference)
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

func TestDockerAttemptTempMountSourcesAreIndependent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	first := filepath.Join(root, "run", "artifact-local", "attempt-one")
	second := filepath.Join(root, "run", "artifact-local", "attempt-two")
	assert.NotEqual(t, dockerTempMountSourceForArtifactRoot(first), dockerTempMountSourceForArtifactRoot(second),
		"independent attempts must never share the host source mounted as container /tmp")
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
	assert.Contains(t, mounts, bindMount(tempMountSource, dockerContainerTempRoot))
	assert.Contains(t, mounts, bindMount(artifactRoot, artifactRoot))
	localPath, err := resolvedRuntimeArtifactLocalPath(artifactRoot, tempMountSource, &pipelinespec.RuntimeArtifact{CustomPath: &customPath})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempMountSource, strings.TrimPrefix(customPath, filepath.Clean(os.TempDir())+string(filepath.Separator))), localPath)
	assert.NotContains(t, mounts, bindMount(filepath.Dir(customPath), filepath.Dir(customPath)))
}

func TestDockerMountsIncludesDedicatedTempDirForRuntimeCustomPaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	request := &TaskExecutionRequest{
		Identity: TaskAttemptIdentity{RunID: "run-123"},
	}
	mounts := dockerMounts(request)
	assert.Contains(
		t,
		mounts,
		bindMount(filepath.Join(homeDir, ".cache", "kfp-v2-runtime", attemptWorkloadName(request.Identity), "docker-tmp"), dockerContainerTempRoot),
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
		CustomPath: ptr(filepath.Join(os.TempDir(), "out_dataset")),
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

func TestCompileDockerCommandAndArgsLocalizesArtifactURIsForDocker(t *testing.T) {
	artifactRoot := filepath.Join(t.TempDir(), "artifact-root")
	customPath := filepath.Join(artifactRoot, "file", "tmp", "kfp-artifacts", "path", "to", "artifact")
	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"out_dataset": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Uri:        "file:///tmp/kfp-artifacts/path/to/artifact",
						CustomPath: ptr(customPath),
					}},
				},
			},
		},
	}

	compiledCommand, compiledArgs, err := compileDockerCommandAndArgs(
		executorInput,
		[]string{"python"},
		[]string{"--executor_input", "{{$}}", "--output-uri", "{{$.outputs.artifacts['out_dataset'].uri}}"},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"python"}, compiledCommand)
	require.Len(t, compiledArgs, 4)
	assert.Equal(t, customPath, compiledArgs[3])

	localizedExecutorInput := &pipelinespec.ExecutorInput{}
	require.NoError(t, protojson.Unmarshal([]byte(compiledArgs[1]), localizedExecutorInput))
	outputArtifact := localizedExecutorInput.GetOutputs().GetArtifacts()["out_dataset"].Artifacts[0]
	assert.Equal(t, customPath, outputArtifact.GetUri())
	assert.Equal(t, customPath, outputArtifact.GetCustomPath())

	originalArtifact := executorInput.GetOutputs().GetArtifacts()["out_dataset"].Artifacts[0]
	assert.Equal(t, "file:///tmp/kfp-artifacts/path/to/artifact", originalArtifact.GetUri())
	assert.Equal(t, customPath, originalArtifact.GetCustomPath())
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
	root := filepath.Join(t.TempDir(), "attempt", "artifact-root")
	assert.Equal(
		t,
		filepath.Join(root, "docker-tmp"),
		dockerTempMountSource(&TaskExecutionRequest{Identity: TaskAttemptIdentity{RunID: "run-123"}, ArtifactLocalRoot: root}),
	)
}

func ptr(value string) *string {
	return &value
}
