package poc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
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

// WorkspaceHandle is a versioned, durable reference to a run workspace.
type WorkspaceHandle struct {
	Version  int               `json:"version"`
	Type     string            `json:"type,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WorkloadExecutorPlugin owns backend workloads, not pipeline orchestration.
type WorkloadExecutorPlugin interface {
	Type() string
	Capabilities() ExecutorCapabilities
	Capacity() ExecutorCapacity
	CheckRequirements(context.Context) error
	Start(context.Context) error
	Shutdown(context.Context) error
	Submit(context.Context, TaskExecutionRequest) (ExecutionReference, error)
	Wait(context.Context, ExecutionReference) (TaskExecutionResult, error)
	Cancel(context.Context, ExecutionReference) (CancellationResult, error)
	Logs(context.Context, ExecutionReference) (ExecutionLogSource, error)
	Cleanup(context.Context, ExecutionReference) error
}

// LostTaskRecoverer reconciles an attempt without creating a second workload.
type LostTaskRecoverer interface {
	Recover(context.Context, TaskAttemptIdentity, *ExecutionReference) (RecoveryResult, error)
}

// ExecutionLogReader streams retained attempt logs without publishing artifacts.
type ExecutionLogReader interface {
	ReadLogs(context.Context, ExecutionReference, bool, io.Writer) error
}

// CancellationState describes what the backend established about termination.
type CancellationState string

const (
	CancellationConfirmed CancellationState = "confirmed"
	CancellationPending   CancellationState = "pending"
	CancellationUnknown   CancellationState = "unknown"
)

// CancellationResult releases capacity only when termination is confirmed.
type CancellationResult struct {
	State        CancellationState
	NeverStarted bool
	// Reference may contain termination proof that must be retained even with an observation error.
	Reference *ExecutionReference
}

// ExecutionLogSource keeps live and retained output bound to one attempt.
type ExecutionLogSource struct {
	Identity TaskAttemptIdentity
	stream   func(context.Context, bool, io.Writer) error
}

// Stream copies attempt output without interpreting backend log locations.
func (s ExecutionLogSource) Stream(ctx context.Context, follow bool, destination io.Writer) error {
	if s.stream == nil {
		return fmt.Errorf("execution log source is unavailable for attempt %s", s.Identity.AttemptID)
	}
	return s.stream(ctx, follow, destination)
}

// ExecutorCapabilities advertises supported workloads and recovery behavior.
type ExecutorCapabilities struct {
	RecoverLostTasks               bool
	SupportsMultiUser              bool
	SupportsContainers             bool
	SupportsKubernetes             bool
	SupportedWorkspaceProvisioners []string
}

// ExecutorCapacity optionally limits concurrently active attempts.
type ExecutorCapacity struct {
	MaxActiveTasks *int
}

// TaskAttemptIdentity remains stable across observer and publication retries.
type TaskAttemptIdentity struct {
	RunID     string `json:"runId"`
	TaskID    string `json:"taskId"`
	AttemptID string `json:"attemptId"`
	Namespace string `json:"namespace,omitempty"`
}

// TaskExecutionRequest contains only the resolved, durable inputs for one attempt.
type TaskExecutionRequest struct {
	Identity             TaskAttemptIdentity                                          `json:"identity"`
	Namespace            string                                                       `json:"namespace,omitempty"`
	TaskName             string                                                       `json:"taskName,omitempty"`
	Container            *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec `json:"container,omitempty"`
	ExecutorInput        *pipelinespec.ExecutorInput                                  `json:"executorInput,omitempty"`
	OutputParameterTypes map[string]pipelinespec.ParameterType_ParameterTypeEnum      `json:"outputParameterTypes,omitempty"`
	Workspace            *WorkspaceHandle                                             `json:"workspace,omitempty"`
	ArtifactLocalRoot    string                                                       `json:"artifactLocalRoot,omitempty"`
	Pod                  *corev1.Pod                                                  `json:"pod,omitempty"`
}

// ExecutionReference persists backend identity separately from framework state.
type ExecutionReference struct {
	Executor string              `json:"executor"`
	Version  int                 `json:"version"`
	Identity TaskAttemptIdentity `json:"identity"`
	Data     model.JSONData      `json:"data"`
}

// TaskOutcome classifies terminal execution, excluding observer errors.
type TaskOutcome string

const (
	TaskOutcomeSuccess TaskOutcome = "success"
	TaskOutcomeFailure TaskOutcome = "failure"
	TaskOutcomeError   TaskOutcome = "error"
)

// TaskFailure distinguishes workload failures from infrastructure errors.
type TaskFailure struct {
	Class     string `json:"class"`
	Message   string `json:"message"`
	Transient bool   `json:"transient,omitempty"`
	Fatal     bool   `json:"fatal,omitempty"`
}

// ExecutionRejectedError establishes that submission did not start a workload.
type ExecutionRejectedError struct {
	Failure TaskFailure
}

func (e *ExecutionRejectedError) Error() string { return e.Failure.Message }

// RecoveryState records what a backend can establish about a lost attempt.
type RecoveryState string

const (
	RecoveryExisting  RecoveryState = "existing"
	RecoveryCompleted RecoveryState = "completed"
	RecoveryStopped   RecoveryState = "stopped"
	RecoveryUnknown   RecoveryState = "unknown"
)

// RecoveryResult never authorizes a duplicate submission for an unknown attempt.
type RecoveryResult struct {
	State     RecoveryState        `json:"state"`
	Reference *ExecutionReference  `json:"reference,omitempty"`
	Result    *TaskExecutionResult `json:"result,omitempty"`
}

// WorkspaceRequest contains only provisioning inputs for a requested workspace.
type WorkspaceRequest struct {
	RunID        string
	RunName      string
	Namespace    string
	Requirements *corev1.PersistentVolumeClaimSpec
}

// ExecutionTarget selects a backend attachment without exposing a run or job.
type ExecutionTarget struct {
	Executor  string
	Namespace string
}

// WorkspaceProvisionerPlugin owns run workspace allocation and attachments.
type WorkspaceProvisionerPlugin interface {
	Type() string
	Provision(context.Context, WorkspaceRequest) (*WorkspaceHandle, error)
	ResolveAttachment(context.Context, *WorkspaceHandle, ExecutionTarget) (*WorkspaceHandle, error)
	Cleanup(context.Context, *WorkspaceHandle) error
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
	Kind            string            `json:"kind,omitempty"`
	APIVersion      string            `json:"apiVersion,omitempty"`
	Mode            string            `json:"mode,omitempty"`
	Executor        string            `json:"executor,omitempty"`
	ExecutorInfo    map[string]string `json:"executorInfo,omitempty"`
	Workspace       *WorkspaceHandle  `json:"workspace,omitempty"`
	Tasks           []RuntimeTask     `json:"tasks,omitempty"`
	RetryGeneration int64             `json:"retryGeneration,omitempty"`
	WorkspaceRetry  bool              `json:"workspaceRetry,omitempty"`
}

// TaskPreparationRequest belongs to the framework, never to an executor plugin.
type TaskPreparationRequest struct {
	Run                      *model.Run
	Task                     *model.Task
	Workspace                *WorkspaceHandle
	Container                *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	ExecutorInput            *pipelinespec.ExecutorInput
	OutputParameterTypes     map[string]pipelinespec.ParameterType_ParameterTypeEnum
	ComponentSpec            *pipelinespec.ComponentSpec
	TaskSpec                 *pipelinespec.PipelineTaskSpec
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig
	ResolverRunTasks         []*apiv2.PipelineTask
	ResolverParentTask       *apiv2.PipelineTask
	ParentTaskID             string
	PipelineName             string
	TaskDisplayName          string
	ArtifactLocalRoot        string
	MLPipelineTLSEnabled     bool
	ObjectStore              component.ObjectStoreClientInterface
}

// TaskExecutionResult is terminal; success includes durable output publication.
type TaskExecutionResult struct {
	Outcome          TaskOutcome                `json:"outcome"`
	Failure          *TaskFailure               `json:"failure,omitempty"`
	OutputParameters map[string]*structpb.Value `json:"outputParameters,omitempty"`
	StatusMetadata   model.JSONData             `json:"statusMetadata,omitempty"`
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

var artifactSchemeRoots = []string{"minio", "gcs", "s3", "oci", "file"}

// PrepareDockerTask resolves immutable paths; transfers belong to executor submission.
func PrepareDockerTask(ctx context.Context, request *TaskPreparationRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if request == nil || request.Container == nil || request.Run == nil {
		return fmt.Errorf("container task preparation request is incomplete")
	}
	if outputs := request.ExecutorInput.GetOutputs(); request.ArtifactLocalRoot == "" &&
		(len(outputs.GetParameters()) != 0 || len(outputs.GetArtifacts()) != 0 || outputs.GetOutputFile() != "") {
		return fmt.Errorf("docker output staging requires an attempt-local artifact root")
	}
	if request.KubernetesExecutorConfig != nil {
		requirements := proto.Clone(request.KubernetesExecutorConfig).(*kubernetesplatform.KubernetesExecutorConfig)
		// Docker create already pulls only when the image is absent.
		if requirements.ImagePullPolicy == string(corev1.PullIfNotPresent) {
			requirements.ImagePullPolicy = ""
		}
		if hasExecutionRequirement(requirements.ProtoReflect()) {
			return fmt.Errorf("docker does not support Kubernetes execution requirements; use the Kubernetes executor")
		}
	}
	if _, err := dockerResourceArgs(request.Container, request.ExecutorInput); err != nil {
		return err
	}
	if _, _, err := dockerEnvironmentArgs(TaskExecutionRequest{
		Container: request.Container, ExecutorInput: request.ExecutorInput, ArtifactLocalRoot: request.ArtifactLocalRoot,
	}); err != nil {
		return err
	}
	return withArtifactLocalRootEnv(request.ArtifactLocalRoot, func() error {
		if err := rebaseDockerInputArtifacts(request.ExecutorInput, request.ArtifactLocalRoot); err != nil {
			return err
		}
		tempRoot := dockerTempMountSource(&TaskExecutionRequest{
			Identity: TaskAttemptIdentity{RunID: request.Run.UUID}, ArtifactLocalRoot: request.ArtifactLocalRoot,
		})
		rebaseDockerOutputFiles(request.ExecutorInput, request.ArtifactLocalRoot, tempRoot)
		normalizeOutputArtifactPathsForDocker(
			request.ExecutorInput,
			request.ArtifactLocalRoot,
			tempRoot,
		)
		return nil
	})
}

func hasExecutionRequirement(message protoreflect.Message) bool {
	if len(message.GetUnknown()) != 0 {
		return true
	}
	present := false
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !field.IsList() && !field.IsMap() && field.Kind() == protoreflect.MessageKind {
			present = hasExecutionRequirement(value.Message())
		} else {
			present = true
		}
		return !present
	})
	return present
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
		mountSet[bindMount(tempMountSource, dockerContainerTempRoot)] = struct{}{}
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
			dir := filepath.Dir(hostPathForContainerPath(output.GetOutputFile(), request.ArtifactLocalRoot, dockerTempMountSource(request)))
			if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) && !isWithinMountedRoot(dir, dockerTempMountSource(request)) {
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
				if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) && !isWithinMountedRoot(dir, dockerTempMountSource(request)) {
					_ = ensureContainerWritableDir(dir)
					mountSet[bindMount(dir, dir)] = struct{}{}
				}
			}
		}
		if metadataPath := request.ExecutorInput.GetOutputs().GetOutputFile(); metadataPath != "" {
			dir := filepath.Dir(hostPathForContainerPath(metadataPath, request.ArtifactLocalRoot, dockerTempMountSource(request)))
			if dir != "" && dir != "." && !isWithinMountedRoot(dir, request.ArtifactLocalRoot) && !isWithinMountedRoot(dir, dockerTempMountSource(request)) {
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
	if request.Identity.RunID != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, ".cache", "kfp-v2-runtime", attemptWorkloadName(request.Identity), "docker-tmp")
	}
	return ""
}

func dockerTempMountSourceForArtifactRoot(artifactLocalRoot string) string {
	if artifactLocalRoot == "" {
		return ""
	}
	return filepath.Join(artifactLocalRoot, "docker-tmp")
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
	root string,
) error {
	if executorInput == nil || executorInput.GetInputs() == nil {
		return nil
	}
	downloaded := map[string]bool{}
	for artifactKey, artifactList := range executorInput.GetInputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			if isWorkspaceRuntimeArtifact(artifact) {
				continue
			}
			localPath, err := localPathForRuntimeArtifact(artifact)
			if err != nil {
				return fmt.Errorf("resolve input artifact %q: %w", artifactKey, err)
			}
			if downloaded[localPath] {
				continue
			}
			if root == "" || localPath == root || !isWithinMountedRoot(localPath, root) {
				return fmt.Errorf("input artifact %q is outside the attempt staging root", artifactKey)
			}
			if objectStore == nil {
				return fmt.Errorf("input artifact %q requires configured object storage", artifactKey)
			}
			if err := ensureContainerWritableDir(filepath.Dir(localPath)); err != nil {
				return err
			}
			temporary, err := os.MkdirTemp(filepath.Dir(localPath), ".input-download-")
			if err != nil {
				return err
			}
			temporaryPath := filepath.Join(temporary, "data")
			if err := objectStore.DownloadArtifact(ctx, artifact.Uri, temporaryPath, artifactKey); err != nil {
				_ = os.RemoveAll(temporary)
				return err
			}
			if err := os.RemoveAll(localPath); err != nil {
				_ = os.RemoveAll(temporary)
				return err
			}
			if err := os.Rename(temporaryPath, localPath); err != nil {
				_ = os.RemoveAll(temporary)
				return err
			}
			if err := os.Remove(temporary); err != nil {
				return err
			}
			downloaded[localPath] = true
		}
	}
	return nil
}

// Apply legacy URI localization only to compilation clones, never persisted inputs.
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
			if artifact == nil || isWorkspaceRuntimeArtifact(artifact) {
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
				return fmt.Errorf("resolve output artifact staging path: %w", err)
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
	if root == "" {
		return component.LocalPathForURI(artifact.Uri)
	}
	// Explicit roots avoid process-global environment changes during concurrent submissions.
	if strings.HasPrefix(artifact.Uri, "oci://") {
		return filepath.Join(root, "oci", strings.ReplaceAll(strings.TrimPrefix(artifact.Uri, "oci://"), "/", "_"), "models"), nil
	}
	for _, scheme := range []struct{ prefix, directory string }{
		{"gs://", "gcs"}, {"minio://", "minio"}, {"s3://", "s3"}, {"file://", "file"},
	} {
		if !strings.HasPrefix(artifact.Uri, scheme.prefix) {
			continue
		}
		schemeRoot := filepath.Join(root, scheme.directory)
		path := filepath.Join(schemeRoot, strings.TrimPrefix(strings.TrimPrefix(artifact.Uri, scheme.prefix), "/"))
		relative, err := filepath.Rel(schemeRoot, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("artifact URI escapes the configured root: %s", artifact.Uri)
		}
		return path, nil
	}
	return "", fmt.Errorf("unsupported artifact URI scheme: %s", artifact.Uri)
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
	for _, tempRoot := range []string{dockerContainerTempRoot, os.TempDir()} {
		cleanTempDir := filepath.Clean(tempRoot)
		if cleanContainerPath == cleanTempDir {
			return tempMountSource
		}
		prefix := cleanTempDir + string(filepath.Separator)
		if strings.HasPrefix(cleanContainerPath, prefix) {
			return filepath.Join(tempMountSource, strings.TrimPrefix(cleanContainerPath, prefix))
		}
	}
	return containerPath
}

func buildCoordinatorExecutorPod(request *TaskPreparationRequest) (*corev1.Pod, error) {
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
		proxy.GetConfig(),
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
		Run: &apiv2.Run{
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
	pod.Labels[commonutil.LabelKeyWorkflowRunId] = request.Run.UUID
	ensureCoordinatorPodAnnotations(pod)
	return pod, nil
}

func taskInputParameters(
	task *model.Task,
	executorInput *pipelinespec.ExecutorInput,
) ([]*apiv2.PipelineTask_InputOutputs_IOParameter, error) {
	parametersByKey := map[string]*apiv2.PipelineTask_InputOutputs_IOParameter{}
	order := []string{}
	add := func(parameter *apiv2.PipelineTask_InputOutputs_IOParameter) {
		if parameter == nil || parameter.GetParameterKey() == "" {
			return
		}
		if _, ok := parametersByKey[parameter.GetParameterKey()]; !ok {
			order = append(order, parameter.GetParameterKey())
		}
		parametersByKey[parameter.GetParameterKey()] = parameter
	}
	if task != nil && len(task.InputParameters) > 0 {
		typeFunc := func() *apiv2.PipelineTask_InputOutputs_IOParameter {
			return &apiv2.PipelineTask_InputOutputs_IOParameter{}
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
			add(&apiv2.PipelineTask_InputOutputs_IOParameter{
				ParameterKey: key,
				Value:        value,
				Type:         apiv2.IOType_COMPONENT_INPUT,
			})
		}
	}
	if len(order) == 0 {
		return nil, nil
	}
	parameters := make([]*apiv2.PipelineTask_InputOutputs_IOParameter, 0, len(order))
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
		Name:                     "kfp-launcher",
		Image:                    argocompiler.GetLauncherImage(),
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		Command:                  argocompiler.GetLauncherCommand(),
		Args:                     []string{"--copy", component.KFPLauncherPath},
		VolumeMounts:             []corev1.VolumeMount{{Name: "kfp-launcher", MountPath: component.VolumePathKFPLauncher}},
	})
	if len(podSpec.Containers) == 0 {
		return
	}
	podSpec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
	podSpec.Containers[0].TerminationMessagePolicy = corev1.TerminationMessageFallbackToLogsOnError
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

func resolvePVCNameForMount(mount *kubernetesplatform.PvcMount, values map[string]*structpb.Value) string {
	if mount == nil {
		return ""
	}
	if mount.GetConstant() != "" { //nolint:staticcheck // SA1019: support deprecated PVC name fields
		return mount.GetConstant() //nolint:staticcheck // SA1019
	}
	if mount.GetComponentInputParameter() != "" { //nolint:staticcheck // SA1019: support deprecated PVC name fields
		if value, ok := values[mount.GetComponentInputParameter()]; ok { //nolint:staticcheck // SA1019
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
