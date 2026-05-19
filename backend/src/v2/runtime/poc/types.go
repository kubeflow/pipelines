package poc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	RuntimeModeCoordinator = "coordinator-poc"

	ExecutorDocker     = "docker"
	ExecutorKubernetes = "kubernetes"

	WorkspaceTypeHostPath = "host-path"
	WorkspaceTypePVC      = "pvc"

	runtimeManifestKind       = "KfpRuntimeManifest"
	runtimeManifestAPIVersion = "pipelines.kubeflow.org/v2alpha1"
	defaultWorkspaceRoot      = "/tmp/kfp-v2-runtime"
	defaultArtifactLocalRoot  = "/tmp/kfp-artifacts"
	localSDKRootEnvVar        = "KFP_V2_LOCAL_SDK_ROOT"
	localSDKMountPath         = "/kfp-local-sdk"
)

var artifactLocalPathEnvMu sync.Mutex

type WorkspaceHandle struct {
	Type     string            `json:"type,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type WorkloadExecutorPlugin interface {
	Type() string
	PrepareRun(ctx context.Context, run *model.Run, job *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error)
	ExecuteTask(ctx context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error)
}

type WorkspaceProvisionerPlugin interface {
	Type() string
	Provision(ctx context.Context, run *model.Run, job *pipelinespec.PipelineJob) (*WorkspaceHandle, error)
	Cleanup(ctx context.Context, run *model.Run, workspace *WorkspaceHandle) error
}

type RuntimeTask struct {
	TaskID       string   `json:"taskId,omitempty"`
	ParentTaskID string   `json:"parentTaskId,omitempty"`
	Name         string   `json:"name,omitempty"`
	DisplayName  string   `json:"displayName,omitempty"`
	Type         string   `json:"type,omitempty"`
	ScopePath    string   `json:"scopePath,omitempty"`
	DependsOn    []string `json:"dependsOn,omitempty"`
}

type RuntimeManifest struct {
	Kind         string            `json:"kind,omitempty"`
	APIVersion   string            `json:"apiVersion,omitempty"`
	Mode         string            `json:"mode,omitempty"`
	Executor     string            `json:"executor,omitempty"`
	ExecutorInfo map[string]string `json:"executorInfo,omitempty"`
	Workspace    *WorkspaceHandle  `json:"workspace,omitempty"`
	Tasks        []RuntimeTask     `json:"tasks,omitempty"`
}

type TaskExecutionRequest struct {
	Run                      *model.Run
	Task                     *model.Task
	Workspace                *WorkspaceHandle
	Container                *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	PipelineSpec             *structpb.Struct
	ExecutorInput            *pipelinespec.ExecutorInput
	OutputParameterTypes     map[string]pipelinespec.ParameterType_ParameterTypeEnum
	ComponentSpec            *pipelinespec.ComponentSpec
	TaskSpec                 *pipelinespec.PipelineTaskSpec
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig
	ResolverRunTasks         []*apiv2beta1.PipelineTaskDetail
	ResolverParentTask       *apiv2beta1.PipelineTaskDetail
	ParentTaskID             string
	PipelineName             string
	TaskDisplayName          string
	ArtifactLocalRoot        string
	MLPipelineTLSEnabled     bool
	ObjectStore              component.ObjectStoreClientInterface
	HandleRecorder           func(model.JSONSlice, model.JSONData) error
}

type TaskExecutionResult struct {
	OutputParameters map[string]*structpb.Value
	StatusMetadata   model.JSONData
}

type runtimeObjectStoreDeps struct {
	openedBucketCache map[string]*blob.Bucket
	launcherConfig    *config.Config
	k8sClient         kubernetes.Interface
	namespace         string
}

func Enabled() bool {
	return common.GetStringConfigWithDefault(common.V2RuntimeMode, "") == RuntimeModeCoordinator
}

func RequestedExecutor() string {
	executor := common.GetStringConfigWithDefault(common.V2RuntimeExecutor, "")
	switch executor {
	case ExecutorDocker, ExecutorKubernetes:
		return executor
	case "":
		if common.GetPodNamespace() != "" {
			return ExecutorKubernetes
		}
		return ExecutorDocker
	default:
		return executor
	}
}

func WorkspaceRoot() string {
	return common.GetStringConfigWithDefault(common.V2RuntimeWorkspaceRoot, defaultWorkspaceRoot)
}

func (m *RuntimeManifest) ToJSON() (string, error) {
	if m == nil {
		return "", nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ParseManifest(raw string) (*RuntimeManifest, error) {
	if raw == "" {
		return nil, nil
	}
	manifest := &RuntimeManifest{}
	if err := json.Unmarshal([]byte(raw), manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func IsManagedRun(run *model.Run) bool {
	if run == nil {
		return false
	}
	if run.PipelineRuntimeManifest == "" {
		return run.WorkflowRuntimeManifest == "" && run.K8SName != "" && run.K8SName == run.UUID
	}
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil {
		return run.WorkflowRuntimeManifest == "" && run.K8SName != "" && run.K8SName == run.UUID
	}
	return manifest != nil && manifest.Kind == runtimeManifestKind && manifest.Mode == RuntimeModeCoordinator
}

func NewRuntimeManifest(executor string) *RuntimeManifest {
	return &RuntimeManifest{
		Kind:       runtimeManifestKind,
		APIVersion: runtimeManifestAPIVersion,
		Mode:       RuntimeModeCoordinator,
		Executor:   executor,
	}
}

type dockerExecutor struct{}

var artifactSchemeRoots = []string{"minio", "gcs", "s3", "oci", "file"}

func (dockerExecutor) Type() string {
	return ExecutorDocker
}

func (dockerExecutor) PrepareRun(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error) {
	info := map[string]string{
		"launchMode": "local-docker",
	}
	if workspace != nil && workspace.Metadata != nil {
		if path := workspace.Metadata["path"]; path != "" {
			info["workspacePath"] = path
		}
	}
	if run != nil && run.Namespace != "" {
		info["namespace"] = run.Namespace
	}
	return info, nil
}

func (dockerExecutor) ExecuteTask(ctx context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	if request == nil || request.Container == nil {
		return nil, fmt.Errorf("container task execution request is incomplete")
	}
	image := request.Container.GetImage()
	if image == "" {
		return nil, fmt.Errorf("container image is required")
	}

	command, args := containerCommandAndArgs(request.Container)
	if err := withArtifactLocalRootEnv(request.ArtifactLocalRoot, func() error {
		if request.ObjectStore != nil {
			if err := downloadInputArtifacts(ctx, request.ObjectStore, request.ExecutorInput); err != nil {
				return err
			}
		}
		normalizeInputArtifactPathsForDocker(request.ExecutorInput)
		normalizeOutputArtifactPathsForDocker(
			request.ExecutorInput,
			request.ArtifactLocalRoot,
			dockerTempMountSource(request),
		)
		if err := prepareOutputArtifactFolders(request.ExecutorInput); err != nil {
			return err
		}
		if command != "" {
			compiledCommand, compiledArgs, err := compileDockerCommandAndArgs(request.ExecutorInput, []string{command}, args)
			if err != nil {
				return err
			}
			if len(compiledCommand) > 0 {
				command = compiledCommand[0]
			}
			args = compiledArgs
		}
		return nil
	}); err != nil {
		return nil, err
	}

	dockerArgs := []string{"run", "--rm"}
	if request.ArtifactLocalRoot != "" {
		dockerArgs = append(dockerArgs, "-e", "ARTIFACT_LOCAL_PATH="+request.ArtifactLocalRoot)
	}
	dockerArgs = append(dockerArgs, dockerProxyEnvArgs()...)
	for _, mount := range dockerMounts(request) {
		dockerArgs = append(dockerArgs, "-v", mount)
	}
	dockerArgs = append(dockerArgs, image)
	if command != "" {
		dockerArgs = append(dockerArgs, command)
	}
	dockerArgs = append(dockerArgs, args...)
	if request.HandleRecorder != nil {
		_ = request.HandleRecorder(nil, customStatusMetadata("", map[string]any{
			"executor":      "docker",
			"image":         image,
			"workspacePath": workspacePath(request.Workspace),
			"artifactRoot":  request.ArtifactLocalRoot,
			"dockerArgs":    strings.Join(dockerArgs, " "),
		}))
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker executor failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var outputParameters map[string]*structpb.Value
	if err := withArtifactLocalRootEnv(request.ArtifactLocalRoot, func() error {
		var collectErr error
		outputParameters, collectErr = collectOutputParameters(request.ExecutorInput, request.OutputParameterTypes)
		return collectErr
	}); err != nil {
		return nil, err
	}
	return &TaskExecutionResult{
		OutputParameters: outputParameters,
		StatusMetadata: customStatusMetadata("", map[string]any{
			"executor": "docker",
			"stdout":   stdout.String(),
			"stderr":   stderr.String(),
		}),
	}, nil
}

type kubernetesExecutor struct {
	coreClient apiserverclient.KubernetesCoreInterface
}

func (kubernetesExecutor) Type() string {
	return ExecutorKubernetes
}

func (kubernetesExecutor) PrepareRun(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error) {
	if run == nil || run.Namespace == "" {
		return nil, fmt.Errorf("kubernetes executor requires a namespace")
	}
	info := map[string]string{
		"launchMode": "kubernetes-pod",
		"namespace":  run.Namespace,
	}
	if workspace != nil && workspace.Metadata != nil {
		if pvc := workspace.Metadata["pvcName"]; pvc != "" {
			info["workspacePVC"] = pvc
		}
	}
	return info, nil
}

func (k kubernetesExecutor) ExecuteTask(ctx context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	if k.coreClient == nil {
		return nil, fmt.Errorf("kubernetes executor requires a kubernetes core client")
	}
	if request == nil || request.Task == nil || request.Run == nil || request.ComponentSpec == nil {
		return nil, fmt.Errorf("kubernetes executor request is incomplete")
	}
	if podName := existingExecutorPodName(request.Task); podName != "" {
		currentPod, err := k.coreClient.PodClient(request.Run.Namespace).Get(ctx, podName, metav1.GetOptions{})
		if err == nil && currentPod != nil {
			if isTerminalTaskState(request.Task.State) {
				switch currentPod.Status.Phase {
				case corev1.PodSucceeded, corev1.PodFailed:
					if deleteErr := k.coreClient.PodClient(request.Run.Namespace).Delete(ctx, currentPod.Name, metav1.DeleteOptions{}); deleteErr == nil {
						currentPod = nil
					}
				}
			}
		}
		if currentPod != nil {
			return waitForExecutorPod(ctx, k.coreClient, request.Run.Namespace, currentPod.Name)
		}
	}

	pod, err := buildCoordinatorExecutorPod(request)
	if err != nil {
		return nil, err
	}
	createdPod, err := k.coreClient.PodClient(request.Run.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	if request.HandleRecorder != nil {
		pods, podsErr := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_TaskPod{{
			Name: createdPod.Name,
			Uid:  string(createdPod.UID),
			Type: apiv2beta1.PipelineTaskDetail_EXECUTOR,
		}})
		if podsErr == nil {
			_ = request.HandleRecorder(pods, customStatusMetadata("", map[string]any{
				"executor": "kubernetes",
				"podName":  createdPod.Name,
				"podUID":   string(createdPod.UID),
			}))
		}
	}
	return waitForExecutorPod(ctx, k.coreClient, request.Run.Namespace, createdPod.Name)
}

type hostPathWorkspaceProvisioner struct{}

func (hostPathWorkspaceProvisioner) Type() string {
	return WorkspaceTypeHostPath
}

func (hostPathWorkspaceProvisioner) Provision(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob) (*WorkspaceHandle, error) {
	if run == nil {
		return nil, fmt.Errorf("run is required")
	}
	workspacePath := filepath.Join(WorkspaceRoot(), run.UUID, "workspace")
	if err := ensureContainerWritableDir(workspacePath); err != nil {
		return nil, err
	}
	return &WorkspaceHandle{
		Type: WorkspaceTypeHostPath,
		Metadata: map[string]string{
			"path": workspacePath,
		},
	}, nil
}

func (hostPathWorkspaceProvisioner) Cleanup(_ context.Context, _ *model.Run, workspace *WorkspaceHandle) error {
	if workspace == nil || workspace.Metadata == nil {
		return nil
	}
	if workspacePath := workspace.Metadata["path"]; workspacePath != "" {
		return os.RemoveAll(workspacePath)
	}
	return nil
}

type pvcWorkspaceProvisioner struct {
	coreClient           apiserverclient.KubernetesCoreInterface
	defaultWorkspaceSpec *corev1.PersistentVolumeClaimSpec
}

func (pvcWorkspaceProvisioner) Type() string {
	return WorkspaceTypePVC
}

func (p pvcWorkspaceProvisioner) Provision(ctx context.Context, run *model.Run, _ *pipelinespec.PipelineJob) (*WorkspaceHandle, error) {
	if run == nil || run.K8SName == "" {
		return nil, fmt.Errorf("run name is required for pvc workspace provisioning")
	}
	if p.coreClient != nil && p.coreClient.GetClientSet() != nil {
		pvcName := driver.GetWorkspacePVCName(run.K8SName)
		pvcClient := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(run.Namespace)
		if _, err := pvcClient.Get(ctx, pvcName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("failed to look up workspace pvc %q in namespace %q: %w", pvcName, run.Namespace, err)
			}
			spec := p.defaultWorkspaceSpec
			if spec == nil {
				spec = &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				}
			}
			if _, createErr := pvcClient.Create(ctx, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: run.Namespace,
				},
				Spec: *spec,
			}, metav1.CreateOptions{}); createErr != nil {
				return nil, fmt.Errorf("failed to create workspace pvc %q in namespace %q: %w", pvcName, run.Namespace, createErr)
			}
		}
	}
	return &WorkspaceHandle{
		Type: WorkspaceTypePVC,
		Metadata: map[string]string{
			"namespace": run.Namespace,
			"pvcName":   driver.GetWorkspacePVCName(run.K8SName),
		},
	}, nil
}

func (p pvcWorkspaceProvisioner) Cleanup(ctx context.Context, run *model.Run, workspace *WorkspaceHandle) error {
	if workspace == nil || workspace.Metadata == nil || p.coreClient == nil || p.coreClient.GetClientSet() == nil {
		return nil
	}
	namespace := workspace.Metadata["namespace"]
	if namespace == "" && run != nil {
		namespace = run.Namespace
	}
	pvcName := workspace.Metadata["pvcName"]
	if namespace == "" || pvcName == "" {
		return nil
	}
	err := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func containerCommandAndArgs(container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) (string, []string) {
	if container == nil {
		return "", nil
	}
	if len(container.Command) == 0 {
		return "", append([]string(nil), container.Args...)
	}
	command := container.Command[0]
	args := append([]string(nil), container.Command[1:]...)
	args = append(args, container.Args...)
	return command, args
}

func dockerMounts(request *TaskExecutionRequest) []string {
	if request == nil {
		return nil
	}
	mountSet := map[string]struct{}{}
	if tempMountSource := dockerTempMountSource(request); tempMountSource != "" {
		_ = ensureContainerWritableDir(tempMountSource)
		mountSet[bindMount(tempMountSource, os.TempDir())] = struct{}{}
	}
	if request.Workspace != nil && request.Workspace.Type == WorkspaceTypeHostPath {
		if workspacePath := request.Workspace.Metadata["path"]; workspacePath != "" {
			_ = ensureContainerWritableDir(workspacePath)
			mountSet[bindMount(workspacePath, component.WorkspaceMountPath)] = struct{}{}
		}
	}
	if localSDKRoot := os.Getenv(localSDKRootEnvVar); localSDKRoot != "" {
		mountSet[bindMount(localSDKRoot, localSDKMountPath)] = struct{}{}
	}
	if request.ArtifactLocalRoot != "" {
		_ = ensureContainerWritableDir(request.ArtifactLocalRoot)
		mountSet[bindMount(request.ArtifactLocalRoot, request.ArtifactLocalRoot)] = struct{}{}
		for _, schemeRoot := range artifactSchemeRoots {
			hostSchemeRoot := filepath.Join(request.ArtifactLocalRoot, schemeRoot)
			_ = ensureContainerWritableDir(hostSchemeRoot)
			mountSet[bindMount(hostSchemeRoot, "/"+schemeRoot)] = struct{}{}
		}
	}
	if request.ExecutorInput != nil && request.ExecutorInput.GetOutputs() != nil {
		for _, output := range request.ExecutorInput.GetOutputs().GetParameters() {
			dir := filepath.Dir(output.GetOutputFile())
			if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) {
				_ = ensureContainerWritableDir(dir)
				mountSet[bindMount(dir, dir)] = struct{}{}
			}
		}
		for _, artifactList := range request.ExecutorInput.GetOutputs().GetArtifacts() {
			for _, artifact := range artifactList.Artifacts {
				localPath, err := resolvedRuntimeArtifactLocalPath(
					request.ArtifactLocalRoot,
					dockerTempMountSource(request),
					artifact,
				)
				if err != nil {
					continue
				}
				dir := filepath.Dir(localPath)
				if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) {
					_ = ensureContainerWritableDir(dir)
					mountSet[bindMount(dir, dir)] = struct{}{}
				}
			}
		}
		if metadataPath := request.ExecutorInput.GetOutputs().GetOutputFile(); metadataPath != "" {
			dir := filepath.Dir(metadataPath)
			if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) {
				_ = ensureContainerWritableDir(dir)
				mountSet[bindMount(dir, dir)] = struct{}{}
			}
		}
	}
	mounts := make([]string, 0, len(mountSet))
	for mount := range mountSet {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)
	return mounts
}

func bindMount(source, target string) string {
	if runtime.GOOS == "linux" {
		return fmt.Sprintf("%s:%s:z", source, target)
	}
	return fmt.Sprintf("%s:%s", source, target)
}

func dockerProxyEnvArgs() []string {
	proxyEnvVars := []string{
		"http_proxy",
		"https_proxy",
		"no_proxy",
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
	}
	var args []string
	for _, envVar := range proxyEnvVars {
		if value, ok := os.LookupEnv(envVar); ok && value != "" {
			args = append(args, "-e", envVar+"="+value)
		}
	}
	return args
}

func dockerTempMountSource(request *TaskExecutionRequest) string {
	if request == nil {
		return ""
	}
	if request.ArtifactLocalRoot != "" {
		return dockerTempMountSourceForArtifactRoot(request.ArtifactLocalRoot)
	}
	if request.Run != nil && request.Run.UUID != "" {
		return dockerTempMountSourceForArtifactRoot(filepath.Join(WorkspaceRoot(), request.Run.UUID, "artifact-local"))
	}
	return ""
}

func dockerTempMountSourceForArtifactRoot(artifactLocalRoot string) string {
	if artifactLocalRoot == "" {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return ""
	}
	runID := filepath.Base(filepath.Dir(artifactLocalRoot))
	if runID == "." || runID == "/" || runID == "" {
		runID = "anonymous"
	}
	return filepath.Join(homeDir, ".cache", "kfp-v2-runtime", runID, "docker-tmp")
}

func isWithinMountedRoot(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func collectOutputParameters(
	executorInput *pipelinespec.ExecutorInput,
	outputParameterTypes map[string]pipelinespec.ParameterType_ParameterTypeEnum,
) (map[string]*structpb.Value, error) {
	if executorInput == nil || executorInput.GetOutputs() == nil {
		return nil, nil
	}
	values := map[string]*structpb.Value{}
	if outputFile := executorInput.GetOutputs().GetOutputFile(); outputFile != "" {
		if content, err := os.ReadFile(outputFile); err == nil && len(content) > 0 {
			executorOutput := &pipelinespec.ExecutorOutput{}
			if err := protojson.Unmarshal(content, executorOutput); err == nil {
				for name, value := range executorOutput.GetParameterValues() {
					values[name] = value
				}
			}
		}
	}
	for name, output := range executorInput.GetOutputs().GetParameters() {
		if _, ok := values[name]; ok {
			continue
		}
		content, err := os.ReadFile(output.GetOutputFile())
		if err != nil {
			continue
		}
		value, err := parseOutputValue(strings.TrimSpace(string(content)), outputParameterTypes[name])
		if err != nil {
			return nil, fmt.Errorf("failed to parse output parameter %q: %w", name, err)
		}
		values[name] = value
	}
	return values, nil
}

func parseOutputValue(raw string, parameterType pipelinespec.ParameterType_ParameterTypeEnum) (*structpb.Value, error) {
	switch parameterType {
	case pipelinespec.ParameterType_NUMBER_INTEGER, pipelinespec.ParameterType_NUMBER_DOUBLE:
		var number float64
		if _, err := fmt.Sscanf(raw, "%f", &number); err != nil {
			return nil, err
		}
		return structpb.NewNumberValue(number), nil
	case pipelinespec.ParameterType_BOOLEAN:
		booleanValue, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid boolean %q", raw)
		}
		return structpb.NewBoolValue(booleanValue), nil
	default:
		return structpb.NewStringValue(raw), nil
	}
}

func workspacePath(workspace *WorkspaceHandle) string {
	if workspace == nil || workspace.Metadata == nil {
		return ""
	}
	return workspace.Metadata["path"]
}

func customStatusMetadata(message string, props map[string]any) model.JSONData {
	status := model.JSONData{}
	if message != "" {
		status["message"] = message
	}
	if len(props) > 0 {
		status["customProperties"] = props
	}
	return status
}

func (d *runtimeObjectStoreDeps) GetOpenedBucketCache() map[string]*blob.Bucket {
	return d.openedBucketCache
}

func (d *runtimeObjectStoreDeps) SetOpenedBucket(key string, bucket *blob.Bucket) {
	d.openedBucketCache[key] = bucket
}

func (d *runtimeObjectStoreDeps) GetLauncherConfig() *config.Config {
	return d.launcherConfig
}

func (d *runtimeObjectStoreDeps) GetK8sClient() kubernetes.Interface {
	return d.k8sClient
}

func (d *runtimeObjectStoreDeps) GetNamespace() string {
	return d.namespace
}

func withArtifactLocalRootEnv(root string, fn func() error) error {
	artifactLocalPathEnvMu.Lock()
	defer artifactLocalPathEnvMu.Unlock()
	previous, hadPrevious := os.LookupEnv("ARTIFACT_LOCAL_PATH")
	if root != "" {
		if err := os.Setenv("ARTIFACT_LOCAL_PATH", root); err != nil {
			return err
		}
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("ARTIFACT_LOCAL_PATH", previous)
		} else {
			_ = os.Unsetenv("ARTIFACT_LOCAL_PATH")
		}
	}()
	return fn()
}

func downloadInputArtifacts(
	ctx context.Context,
	objectStore component.ObjectStoreClientInterface,
	executorInput *pipelinespec.ExecutorInput,
) error {
	if executorInput == nil || executorInput.GetInputs() == nil {
		return nil
	}
	for artifactKey, artifactList := range executorInput.GetInputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			if artifact.GetMetadata() != nil {
				if value, ok := artifact.GetMetadata().GetFields()["_kfp_workspace"]; ok && value.GetBoolValue() {
					continue
				}
			}
			localPath, err := localPathForRuntimeArtifact(artifact)
			if err != nil {
				continue
			}
			if _, statErr := os.Stat(localPath); statErr == nil {
				continue
			}
			if err := objectStore.DownloadArtifact(ctx, artifact.Uri, localPath, artifactKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeInputArtifactPathsForDocker(executorInput *pipelinespec.ExecutorInput) {
	if executorInput == nil || executorInput.GetInputs() == nil {
		return
	}
	for _, artifactList := range executorInput.GetInputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			if artifact == nil || artifact.GetCustomPath() == "" {
				continue
			}
			// Older pip-installed SDK runtimes derive input artifact paths from uri only.
			// Keep the custom path in both fields so local Docker execution is compatible.
			artifact.Uri = artifact.GetCustomPath()
		}
	}
}

func normalizeOutputArtifactPathsForDocker(
	executorInput *pipelinespec.ExecutorInput,
	artifactLocalRoot string,
	tempMountSource string,
) {
	if executorInput == nil || executorInput.GetOutputs() == nil {
		return
	}
	for _, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			if artifact == nil || artifact.GetCustomPath() != "" {
				continue
			}
			localPath, err := resolvedRuntimeArtifactLocalPath(
				artifactLocalRoot,
				tempMountSource,
				artifact,
			)
			if err != nil || localPath == "" {
				continue
			}
			artifact.CustomPath = &localPath
		}
	}
}

func compileDockerCommandAndArgs(
	executorInput *pipelinespec.ExecutorInput,
	command []string,
	args []string,
) ([]string, []string, error) {
	if executorInput == nil {
		return component.CompileCommandAndArgs(executorInput, command, args)
	}
	clonedExecutorInput, ok := proto.Clone(executorInput).(*pipelinespec.ExecutorInput)
	if !ok {
		return component.CompileCommandAndArgs(executorInput, command, args)
	}
	normalizeInputArtifactPathsForDocker(clonedExecutorInput)
	normalizeOutputArtifactPathsForDocker(
		clonedExecutorInput,
		artifactLocalRootFromOutputFile(clonedExecutorInput.GetOutputs().GetOutputFile()),
		dockerTempMountSourceForArtifactRoot(artifactLocalRootFromOutputFile(clonedExecutorInput.GetOutputs().GetOutputFile())),
	)
	for _, artifactList := range clonedExecutorInput.GetOutputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			if artifact == nil || artifact.GetCustomPath() == "" {
				continue
			}
			artifact.Uri = artifact.GetCustomPath()
		}
	}
	return component.CompileCommandAndArgs(clonedExecutorInput, command, args)
}

func prepareOutputArtifactFolders(executorInput *pipelinespec.ExecutorInput) error {
	if executorInput == nil || executorInput.GetOutputs() == nil {
		return nil
	}
	for _, parameter := range executorInput.GetOutputs().GetParameters() {
		if err := resetContainerWritableOutputPath(parameter.GetOutputFile()); err != nil {
			return err
		}
	}
	for _, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			artifactLocalRoot := artifactLocalRootFromOutputFile(executorInput.GetOutputs().GetOutputFile())
			localPath, err := resolvedRuntimeArtifactLocalPath(
				artifactLocalRoot,
				dockerTempMountSourceForArtifactRoot(artifactLocalRoot),
				artifact,
			)
			if err != nil {
				continue
			}
			if err := resetContainerWritableOutputPath(localPath); err != nil {
				return err
			}
		}
	}
	if outputFile := executorInput.GetOutputs().GetOutputFile(); outputFile != "" {
		if err := resetContainerWritableOutputPath(outputFile); err != nil {
			return err
		}
	}
	return nil
}

func ensureContainerWritableDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	if err := os.MkdirAll(path, 0o777); err != nil {
		return err
	}
	return os.Chmod(path, 0o777)
}

func resetContainerWritableOutputPath(path string) error {
	if path == "" {
		return nil
	}
	if err := ensureContainerWritableDir(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func localPathForRuntimeArtifact(artifact *pipelinespec.RuntimeArtifact) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("artifact is nil")
	}
	if artifact.CustomPath != nil && *artifact.CustomPath != "" {
		return *artifact.CustomPath, nil
	}
	return component.LocalPathForURI(artifact.Uri)
}

func resolvedRuntimeArtifactLocalPath(
	root string,
	tempMountSource string,
	artifact *pipelinespec.RuntimeArtifact,
) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("artifact is nil")
	}
	if artifact.CustomPath != nil && *artifact.CustomPath != "" {
		return hostPathForContainerPath(*artifact.CustomPath, root, tempMountSource), nil
	}
	artifactPathFn := func() (string, error) {
		return component.LocalPathForURI(artifact.Uri)
	}
	if root != "" {
		return withArtifactLocalPath(root, artifactPathFn)
	}
	return artifactPathFn()
}

func artifactLocalRootFromOutputFile(outputFile string) string {
	for _, schemeRoot := range artifactSchemeRoots {
		prefixedSchemeRoot := "/" + schemeRoot + "/"
		if index := strings.Index(outputFile, prefixedSchemeRoot); index >= 0 {
			return outputFile[:index]
		}
	}
	return ""
}

func hostPathForContainerPath(containerPath, artifactLocalRoot, tempMountSource string) string {
	if artifactLocalRoot != "" {
		cleanArtifactRoot := filepath.Clean(artifactLocalRoot)
		cleanPath := filepath.Clean(containerPath)
		if cleanPath == cleanArtifactRoot {
			return containerPath
		}
		artifactPrefix := cleanArtifactRoot + string(filepath.Separator)
		if strings.HasPrefix(cleanPath, artifactPrefix) {
			return containerPath
		}
	}
	if tempMountSource == "" {
		return containerPath
	}
	cleanContainerPath := filepath.Clean(containerPath)
	cleanTempDir := filepath.Clean(os.TempDir())
	if cleanContainerPath == cleanTempDir {
		return tempMountSource
	}
	prefix := cleanTempDir + string(filepath.Separator)
	if strings.HasPrefix(cleanContainerPath, prefix) {
		return filepath.Join(tempMountSource, strings.TrimPrefix(cleanContainerPath, prefix))
	}
	return containerPath
}

func buildCoordinatorExecutorPod(request *TaskExecutionRequest) (*corev1.Pod, error) {
	if request == nil || request.Container == nil || request.Run == nil || request.Task == nil || request.ComponentSpec == nil {
		return nil, fmt.Errorf("task execution request is incomplete")
	}
	serverConfig := coordinatorMLPipelineServerConfig()
	caCertPath := coordinatorLauncherCACertPath(request.MLPipelineTLSEnabled)
	podSpec, err := driver.BuildLauncherPodSpec(
		request.Container,
		request.ComponentSpec,
		request.ExecutorInput,
		request.Task.UUID,
		request.ParentTaskID,
		request.PipelineName,
		request.Run.UUID,
		request.Run.K8SName,
		"1",
		"true",
		"false",
		nil,
		request.Task.Fingerprint,
		nil,
		request.Task.Name,
		request.MLPipelineTLSEnabled,
		caCertPath,
		serverConfig.Address,
		serverConfig.Port,
	)
	if err != nil {
		return nil, err
	}
	augmentPodSpecForLauncher(podSpec)
	inputParameters, err := taskInputParameters(request.Task, request.ExecutorInput)
	if err != nil {
		return nil, err
	}
	driverOpts := drivercommon.Options{
		Run: &apiv2beta1.Run{
			RunId:       request.Run.UUID,
			DisplayName: request.Run.DisplayName,
			Tasks:       request.ResolverRunTasks,
		},
		Component:                request.ComponentSpec,
		Task:                     request.TaskSpec,
		ParentTask:               request.ResolverParentTask,
		KubernetesExecutorConfig: request.KubernetesExecutorConfig,
		IterationIndex:           -1,
		RunName:                  request.Run.K8SName,
		RunDisplayName:           request.Run.DisplayName,
		Namespace:                request.Run.Namespace,
		TaskName:                 request.Task.Name,
		MLPipelineServerAddress:  serverConfig.Address,
		MLPipelineServerPort:     serverConfig.Port,
	}
	if err := driver.ExtendPodSpecPatch(context.Background(), podSpec, driverOpts, inputParameters, nil); err != nil {
		return nil, err
	}
	if caCertPath != "" {
		attachCoordinatorCustomCABundle(podSpec)
	}
	if request.ArtifactLocalRoot != "" && len(podSpec.Containers) > 0 {
		podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, corev1.EnvVar{
			Name:  "ARTIFACT_LOCAL_PATH",
			Value: request.ArtifactLocalRoot,
		})
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeCoordinatorPodName(request.Task.Name, request.Task.UUID),
			Namespace: request.Run.Namespace,
			Labels: map[string]string{
				"pipelines.kubeflow.org/run_id":  request.Run.UUID,
				"pipelines.kubeflow.org/task_id": request.Task.UUID,
			},
		},
		Spec: func() corev1.PodSpec {
			podSpec.ServiceAccountName = request.Run.ServiceAccount
			return *podSpec
		}(),
	}
	applyPodMetadata(pod, request.KubernetesExecutorConfig)
	ensureCoordinatorPodAnnotations(pod)
	return pod, nil
}

func taskInputParameters(
	task *model.Task,
	executorInput *pipelinespec.ExecutorInput,
) ([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	parametersByKey := map[string]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	order := []string{}
	add := func(parameter *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) {
		if parameter == nil || parameter.GetParameterKey() == "" {
			return
		}
		if _, ok := parametersByKey[parameter.GetParameterKey()]; !ok {
			order = append(order, parameter.GetParameterKey())
		}
		parametersByKey[parameter.GetParameterKey()] = parameter
	}
	if task != nil && len(task.InputParameters) > 0 {
		typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
			return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
		}
		storedParameters, err := model.JSONSliceToProtoSlice(task.InputParameters, typeFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to decode task input parameters: %w", err)
		}
		for _, parameter := range storedParameters {
			add(parameter)
		}
	}
	if executorInput != nil && executorInput.GetInputs() != nil {
		for key, value := range executorInput.GetInputs().GetParameterValues() {
			if _, ok := parametersByKey[key]; ok {
				continue
			}
			add(&apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: key,
				Value:        value,
				Type:         apiv2beta1.IOType_COMPONENT_INPUT,
			})
		}
	}
	if len(order) == 0 {
		return nil, nil
	}
	parameters := make([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(order))
	for _, key := range order {
		parameters = append(parameters, parametersByKey[key])
	}
	return parameters, nil
}

func coordinatorMLPipelineServerConfig() *config.ServerConfig {
	serverConfig := config.GetMLPipelineServerConfig()
	if os.Getenv("LOCAL_API_SERVER") != "true" {
		return serverConfig
	}
	address := os.Getenv("KFP_V2_LOCAL_API_SERVER_ADDRESS")
	if address == "" {
		return serverConfig
	}
	port := os.Getenv("KFP_V2_LOCAL_API_SERVER_PORT")
	if port == "" {
		port = serverConfig.Port
	}
	return &config.ServerConfig{
		Address: address,
		Port:    port,
	}
}

func coordinatorLauncherCACertPath(mlPipelineTLSEnabled bool) string {
	if !mlPipelineTLSEnabled {
		return ""
	}
	if common.GetCaBundleSecretName() == "" && common.GetCaBundleConfigMapName() == "" {
		return ""
	}
	return common.CustomCaCertPath
}

func attachCoordinatorCustomCABundle(podSpec *corev1.PodSpec) {
	if podSpec == nil {
		return
	}
	caBundleKeyName := common.GetCABundleKey()
	if caBundleKeyName == "" {
		caBundleKeyName = "ca.crt"
	}
	volumeSource := corev1.VolumeSource{}
	if caBundleSecretName := common.GetCaBundleSecretName(); caBundleSecretName != "" {
		volumeSource.Secret = &corev1.SecretVolumeSource{
			SecretName: caBundleSecretName,
			Items: []corev1.KeyToPath{{
				Key:  caBundleKeyName,
				Path: "ca.crt",
			}},
		}
	} else if caBundleConfigMapName := common.GetCaBundleConfigMapName(); caBundleConfigMapName != "" {
		volumeSource.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: caBundleConfigMapName},
			Items: []corev1.KeyToPath{{
				Key:  caBundleKeyName,
				Path: "ca.crt",
			}},
		}
	} else {
		return
	}
	podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
		Name:         "custom-ca",
		VolumeSource: volumeSource,
	})
	if len(podSpec.Containers) > 0 {
		podSpec.Containers[0].VolumeMounts = append(
			podSpec.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: "custom-ca", MountPath: common.CABundleDir},
		)
	}
}

func applyPodMetadata(pod *corev1.Pod, cfg *kubernetesplatform.KubernetesExecutorConfig) {
	if pod == nil || cfg == nil || cfg.GetPodMetadata() == nil {
		return
	}
	if annotations := cfg.GetPodMetadata().GetAnnotations(); len(annotations) > 0 {
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		for key, value := range annotations {
			pod.Annotations[key] = value
		}
	}
	if labels := cfg.GetPodMetadata().GetLabels(); len(labels) > 0 {
		if pod.Labels == nil {
			pod.Labels = map[string]string{}
		}
		for key, value := range labels {
			pod.Labels[key] = value
		}
	}
}

func ensureCoordinatorPodAnnotations(pod *corev1.Pod) {
	if pod == nil {
		return
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	if _, ok := pod.Annotations[commonutil.AnnotationKeyIstioSidecarInject]; !ok {
		pod.Annotations[commonutil.AnnotationKeyIstioSidecarInject] = commonutil.AnnotationValueIstioSidecarInjectDisabled
	}
}

func augmentPodSpecForLauncher(podSpec *corev1.PodSpec) {
	if podSpec == nil {
		return
	}
	podSpec.RestartPolicy = corev1.RestartPolicyNever
	podSpec.Volumes = append(podSpec.Volumes,
		corev1.Volume{
			Name: "kfp-launcher",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: "kfp-launcher-token",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience:          "pipelines.kubeflow.org",
							ExpirationSeconds: int64Ptr(7200),
							Path:              "token",
						},
					}},
				},
			},
		},
		corev1.Volume{Name: "gcs-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "s3-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "minio-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "dot-local-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "dot-cache-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "dot-config-scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		corev1.Volume{Name: "artifact-local-root", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	)
	podSpec.InitContainers = append(podSpec.InitContainers, corev1.Container{
		Name:            "kfp-launcher",
		Image:           argocompiler.GetLauncherImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         argocompiler.GetLauncherCommand(),
		Args:            []string{"--copy", component.KFPLauncherPath},
		VolumeMounts:    []corev1.VolumeMount{{Name: "kfp-launcher", MountPath: component.VolumePathKFPLauncher}},
	})
	if len(podSpec.Containers) == 0 {
		return
	}
	podSpec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
	podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts,
		corev1.VolumeMount{Name: "kfp-launcher", MountPath: component.VolumePathKFPLauncher},
		corev1.VolumeMount{Name: "kfp-launcher-token", MountPath: "/var/run/secrets/kfp", ReadOnly: true},
		corev1.VolumeMount{Name: "gcs-scratch", MountPath: "/gcs"},
		corev1.VolumeMount{Name: "s3-scratch", MountPath: "/s3"},
		corev1.VolumeMount{Name: "minio-scratch", MountPath: "/minio"},
		corev1.VolumeMount{Name: "dot-local-scratch", MountPath: "/.local"},
		corev1.VolumeMount{Name: "dot-cache-scratch", MountPath: "/.cache"},
		corev1.VolumeMount{Name: "dot-config-scratch", MountPath: "/.config"},
		corev1.VolumeMount{Name: "artifact-local-root", MountPath: defaultArtifactLocalRoot},
	)
}

func makeCoordinatorPodName(taskName, taskID string) string {
	safeTask := strings.ToLower(taskName)
	safeTask = strings.ReplaceAll(safeTask, "_", "-")
	safeTask = strings.ReplaceAll(safeTask, ".", "-")
	var b strings.Builder
	for _, r := range safeTask {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "kfp-task"
	}
	shortID := taskID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("%s-%s", name, shortID)
}

func int64Ptr(value int64) *int64 {
	return &value
}

func insertLauncherFlag(command []string, flagName string, flagValue string) []string {
	if len(command) == 0 {
		return command
	}
	for index, value := range command {
		if value == "--" {
			updated := append([]string{}, command[:index]...)
			updated = append(updated, flagName, flagValue)
			updated = append(updated, command[index:]...)
			return updated
		}
	}
	return append(command, flagName, flagValue)
}

func applyPVCMounts(
	podSpec *corev1.PodSpec,
	cfg *kubernetesplatform.KubernetesExecutorConfig,
	executorInput *pipelinespec.ExecutorInput,
) {
	if podSpec == nil || cfg == nil || len(podSpec.Containers) == 0 || cfg.GetPvcMount() == nil {
		return
	}
	values := map[string]*structpb.Value{}
	if executorInput != nil && executorInput.GetInputs() != nil {
		values = executorInput.GetInputs().GetParameterValues()
	}
	for index, mount := range cfg.GetPvcMount() {
		if mount == nil {
			continue
		}
		pvcName := resolvePVCNameForMount(mount, values)
		mountPath := mount.GetMountPath()
		if pvcName == "" || mountPath == "" {
			continue
		}
		volumeName := fmt.Sprintf("kfp-pvc-%d", index)
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
			},
		})
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			SubPath:   mount.GetSubPath(),
		})
	}
}

func resolvePVCNameForMount(mount *kubernetesplatform.PvcMount, values map[string]*structpb.Value) string {
	if mount == nil {
		return ""
	}
	if mount.GetConstant() != "" {
		return mount.GetConstant()
	}
	if mount.GetComponentInputParameter() != "" {
		if value, ok := values[mount.GetComponentInputParameter()]; ok {
			return value.GetStringValue()
		}
	}
	if mount.GetPvcNameParameter() != nil && mount.GetPvcNameParameter().GetComponentInputParameter() != "" {
		if value, ok := values[mount.GetPvcNameParameter().GetComponentInputParameter()]; ok {
			return value.GetStringValue()
		}
	}
	return ""
}

func existingExecutorPodName(task *model.Task) string {
	if task == nil || task.Pods == nil {
		return ""
	}
	pods, err := model.JSONSliceToProtoSlice(task.Pods, func() *apiv2beta1.PipelineTaskDetail_TaskPod {
		return &apiv2beta1.PipelineTaskDetail_TaskPod{}
	})
	if err != nil {
		return ""
	}
	for _, pod := range pods {
		if pod.GetType() == apiv2beta1.PipelineTaskDetail_EXECUTOR && pod.GetName() != "" {
			return pod.GetName()
		}
	}
	return ""
}

func waitForExecutorPod(
	ctx context.Context,
	coreClient apiserverclient.KubernetesCoreInterface,
	namespace string,
	podName string,
) (*TaskExecutionResult, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		currentPod, err := coreClient.PodClient(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		switch currentPod.Status.Phase {
		case corev1.PodSucceeded:
			return &TaskExecutionResult{
				StatusMetadata: customStatusMetadata("", map[string]any{
					"executor": "kubernetes",
					"podName":  currentPod.Name,
					"phase":    string(currentPod.Status.Phase),
				}),
			}, nil
		case corev1.PodFailed:
			return nil, fmt.Errorf("kubernetes pod %s failed: %s", currentPod.Name, currentPod.Status.Message)
		}
	}
}
