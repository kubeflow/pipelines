package component

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

type ImportLauncher struct {
	opts              LauncherV2Options
	clientManager     client_manager.ClientManagerInterface
	objectStore       ObjectStoreClientInterface
	openedBucketCache map[string]*blob.Bucket
	launcherConfig    *config.Config
}

func NewImporterLauncher(
	launcherV2Opts *LauncherV2Options,
	clientManager client_manager.ClientManagerInterface,
) (l *ImportLauncher, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create importer launcher: %w", err)
		}
	}()
	err = launcherV2Opts.validate()
	if err != nil {
		return nil, err
	}
	launcher := &ImportLauncher{
		opts:              *launcherV2Opts,
		clientManager:     clientManager,
		openedBucketCache: make(map[string]*blob.Bucket),
	}
	launcher.objectStore = NewObjectStoreClient(launcher)
	return launcher, nil
}

func (l *ImportLauncher) Execute(ctx context.Context) (executionErr error) {
	defer func() {
		if executionErr != nil {
			executionErr = fmt.Errorf("failed to execute importer component: %w", executionErr)
		}
	}()

	// Close any open buckets in the cache
	defer func() {
		for _, bucket := range l.openedBucketCache {
			_ = bucket.Close()
		}
	}()

	// Fetch Launcher config
	launcherConfig, executionErr := config.FetchLauncherConfigMap(ctx, l.clientManager.K8sClient(), l.opts.Namespace)
	if executionErr != nil {
		return fmt.Errorf("failed to get launcher configmap: %w", executionErr)
	}
	l.launcherConfig = launcherConfig

	kfpAPI := l.clientManager.KFPAPIClient()

	downloadToWorkspace := false
	if l.opts.ImporterSpec.GetDownloadToWorkspace() {
		downloadToWorkspace = true
	}

	// Create the task, we will continue to update this as needed.
	parentTaskID := l.opts.ParentTask.GetTaskId()
	createdTask, executionErr := kfpAPI.CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
		Task: &apiV2beta1.PipelineTaskDetail{
			Name:         l.opts.TaskSpec.GetTaskInfo().GetName(),
			DisplayName:  l.opts.TaskSpec.GetTaskInfo().GetName(),
			RunId:        l.opts.Run.RunId,
			ParentTaskId: &parentTaskID,
			Type:         apiV2beta1.PipelineTaskDetail_IMPORTER,
			State:        apiV2beta1.PipelineTaskDetail_RUNNING,
			ScopePath:    l.opts.ScopePath.DotNotation(),
			CreateTime:   timestamppb.Now(),
			TypeAttributes: &apiV2beta1.PipelineTaskDetail_TypeAttributes{
				DownloadToWorkspace: util.BoolPointer(downloadToWorkspace),
			},
			Pods: []*apiV2beta1.PipelineTaskDetail_TaskPod{
				{
					Name: l.opts.PodName,
					Uid:  l.opts.PodUID,
					Type: apiV2beta1.PipelineTaskDetail_EXECUTOR,
				},
			},
		},
	})
	if executionErr != nil {
		return executionErr
	}

	// The defer statement is used to ensure we propagate any errors
	// encountered in this task execution.
	defer func() {
		if executionErr != nil {
			createdTask.State = apiV2beta1.PipelineTaskDetail_FAILED
		} else {
			createdTask.State = apiV2beta1.PipelineTaskDetail_SUCCEEDED
		}
		createdTask.EndTime = timestamppb.Now()
		_, updateErr := kfpAPI.UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
			TaskId: createdTask.TaskId,
			Task:   createdTask,
		})
		if updateErr != nil {
			glog.Errorf("failed to update task: %v", updateErr)
			return
		}
		// Propagate any statuses up the DAG.
		updateStatusErr := l.clientManager.KFPAPIClient().UpdateStatuses(ctx, l.opts.Run, l.opts.PipelineSpec, l.opts.Task)
		if updateStatusErr != nil {
			glog.Errorf("failed to update statuses: %v", updateStatusErr)
			return
		}
	}()

	if createdTask == nil {
		return fmt.Errorf("failed to create task for importer execution")
	}
	l.opts.Task = createdTask

	if createdTask.Outputs == nil {
		createdTask.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Artifacts: make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, 0),
		}
	} else if createdTask.Outputs.Artifacts == nil {
		createdTask.Outputs.Artifacts = make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, 0)
	}

	// Handle artifact creation and links to Importer Task
	artifactToImport, executionErr := l.ImportSpecToArtifact()
	if executionErr != nil {
		return executionErr
	}

	// Determine if the Artifact already exists.
	preExistingArtifact, executionErr := l.findMatchedArtifact(ctx, artifactToImport)
	if executionErr != nil {
		return executionErr
	}

	// Get the output artifact name from the component spec.
	artifactOutputKey, executionErr := l.getArtifactOutputKey()
	if executionErr != nil {
		return executionErr
	}

	// If reimport is true or the artifact does not already exist we create a new artifact
	if l.opts.ImporterSpec.Reimport || preExistingArtifact == nil {
		glog.Infof("Creating new artifact for importer task %s", l.opts.TaskSpec.GetTaskInfo().GetName())
		_, executionErr = kfpAPI.CreateArtifact(ctx, &apiV2beta1.CreateArtifactRequest{
			Artifact:    artifactToImport,
			RunId:       l.opts.Run.RunId,
			TaskId:      createdTask.TaskId,
			ProducerKey: artifactOutputKey,
			Type:        apiV2beta1.IOType_OUTPUT,
		})
		if executionErr != nil {
			return executionErr
		}
	} else {
		glog.Infof("Reusing existing artifact %s for importer task %s", preExistingArtifact.GetArtifactId(), l.opts.TaskSpec.GetTaskInfo().GetName())
		// If reimporting then we just need to create a new link to this Importer task via
		// and ArtifactTask entry.
		_, executionErr = kfpAPI.CreateArtifactTask(ctx, &apiV2beta1.CreateArtifactTaskRequest{
			ArtifactTask: &apiV2beta1.ArtifactTask{
				ArtifactId: preExistingArtifact.GetArtifactId(),
				TaskId:     createdTask.TaskId,
				RunId:      l.opts.Run.RunId,
				Key:        artifactOutputKey,
				Type:       apiV2beta1.IOType_OUTPUT,
				Producer: &apiV2beta1.IOProducer{
					TaskName: l.opts.TaskSpec.GetTaskInfo().GetName(),
				},
			},
		})
		if executionErr != nil {
			return executionErr
		}
	}

	return nil
}

func (l *ImportLauncher) findMatchedArtifact(ctx context.Context, artifactToMatch *apiV2beta1.Artifact) (matchedArtifact *apiV2beta1.Artifact, err error) {
	artifacts, err := l.clientManager.KFPAPIClient().ListArtifactsByURI(ctx, artifactToMatch.GetUri(), l.opts.Namespace)
	if err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		if artifact.GetUri() == artifactToMatch.GetUri() {
			return artifact, nil
		}
	}
	for _, candidateArtifact := range artifacts {
		if artifactsAreEqual(artifactToMatch, candidateArtifact) {
			return candidateArtifact, nil
		}
	}
	// No match found
	return nil, nil
}

func artifactsAreEqual(artifact1, artifact2 *apiV2beta1.Artifact) bool {
	if artifact1.GetType() != artifact2.GetType() {
		return false
	}
	if artifact1.GetUri() != artifact2.GetUri() {
		return false
	}
	if artifact1.GetName() != artifact2.GetName() {
		return false
	}
	if artifact1.GetDescription() != artifact2.GetDescription() {
		return false
	}
	// Compare metadata fields
	metadata1 := artifact1.GetMetadata()
	metadata2 := artifact2.GetMetadata()
	if len(metadata1) != len(metadata2) {
		return false
	}
	for k, v1 := range metadata1 {
		if v2, exists := metadata2[k]; !exists || v1 != v2 {
			return false
		}
	}
	return true
}

func (l *ImportLauncher) ImportSpecToArtifact() (artifact *apiV2beta1.Artifact, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create Artifact from ImporterSpec: %w", err)
		}
	}()

	importerSpec := l.opts.ImporterSpec
	artifactType, err := inferArtifactType(importerSpec.GetTypeSchema())
	if err != nil {
		return nil, fmt.Errorf("failed to get schemaType from importer spec: %w", err)
	}
	// Resolve artifact URI. Can be one of two sources:
	// 1) Constant
	// 2) Runtime Parameter
	// TODO(Humair): The logic here is very similar to how InputParameters are resolved in the driver's resolver package.
	// We should consolidate this logic.
	var artifactUri string
	switch {
	case importerSpec.GetArtifactUri().GetConstant() != nil:
		glog.Infof("Artifact URI as constant: %+v", importerSpec.GetArtifactUri().GetConstant())
		artifactUri = importerSpec.GetArtifactUri().GetConstant().GetStringValue()
		if artifactUri == "" {
			return nil, fmt.Errorf("empty Artifact URI constant value")
		}
	case importerSpec.GetArtifactUri().GetRuntimeParameter() != "":
		paramName := importerSpec.GetArtifactUri().GetRuntimeParameter()
		taskInput, ok := l.opts.TaskSpec.GetInputs().GetParameters()[paramName]
		if !ok {
			return nil, fmt.Errorf("cannot find parameter %s in task input to fetch artifact uri", paramName)
		}
		componentInput := taskInput.GetComponentInputParameter()
		var ioParam *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
		for _, inputParam := range l.opts.ParentTask.GetInputs().GetParameters() {
			if inputParam.ParameterKey == componentInput {
				ioParam = inputParam
				break
			}
		}
		if ioParam == nil {
			return nil, fmt.Errorf("cannot find parameter %s in parent task input to fetch artifact uri", componentInput)
		}
		artifactUri = ioParam.GetValue().GetStringValue()
		if artifactUri == "" {
			return nil, fmt.Errorf("empty artifact URI runtime value for parameter %s", paramName)
		}
	default:
		return nil, fmt.Errorf("artifact uri not provided")
	}

	// TODO(HumairAK): Allow user to specify a canonical artifact Name & Description when importing
	// For now we infer the name from the URI object name.
	artifactName, err := inferArtifactName(artifactUri)
	if err != nil {
		return nil, fmt.Errorf("failed to extract filename from artifact uri: %w", err)
	}
	artifact = &apiV2beta1.Artifact{
		Name:        artifactName,
		Description: "",
		Type:        artifactType,
		Uri:         &artifactUri,
		CreatedAt:   timestamppb.Now(),
		Namespace:   l.opts.Namespace,
	}
	if importerSpec.Metadata != nil {
		artifact.Metadata = importerSpec.Metadata.GetFields()
	}
	if strings.HasPrefix(artifactUri, "oci://") {
		// OCI artifacts are not supported when workspace is used
		if l.opts.ImporterSpec.GetDownloadToWorkspace() {
			return nil, fmt.Errorf("importer workspace download does not support OCI registries")
		}

		if artifactType != apiV2beta1.Artifact_Model {
			return nil, fmt.Errorf("the %s artifact type does not support OCI registries", apiV2beta1.Artifact_Model)
		}
		return artifact, nil
	}

	// Download the artifact into the workspace
	if l.opts.ImporterSpec.GetDownloadToWorkspace() {
		localPath, err := LocalWorkspacePathForURI(artifactUri)
		if err != nil {
			return nil, fmt.Errorf("failed to get local path for uri %q: %w", artifactUri, err)
		}
		// Get the artifact output key from the component spec
		artifactKey, err := l.getArtifactOutputKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get artifact output key: %w", err)
		}
		ctx := context.Background()
		glog.Infof("Downloading artifact %q (artifact key %q) to workspace path %q", artifactUri, artifactKey, localPath)
		err = l.objectStore.DownloadArtifact(ctx, artifactUri, localPath, artifactKey)
		if err != nil {
			return nil, fmt.Errorf("failed to download artifact to workspace: %w", err)
		}
	}
	return artifact, nil
}

func (l *ImportLauncher) getArtifactOutputKey() (string, error) {
	outputNames := make([]string, 0, len(l.opts.ComponentSpec.GetOutputDefinitions().GetArtifacts()))
	for name := range l.opts.ComponentSpec.GetOutputDefinitions().GetArtifacts() {
		outputNames = append(outputNames, name)
	}
	if len(outputNames) != 1 {
		return "", fmt.Errorf("failed to extract output artifact name from componentOutputSpec")
	}
	return outputNames[0], nil
}

func inferArtifactType(typeSchema *pipelinespec.ArtifactTypeSchema) (apiV2beta1.Artifact_ArtifactType, error) {
	schemaType, err := getArtifactSchemaType(typeSchema)
	if err != nil {
		return apiV2beta1.Artifact_TYPE_UNSPECIFIED, fmt.Errorf("failed to get schemaType from importer spec: %w", err)
	}
	return artifactTypeSchemaToArtifactType(schemaType)
}

func inferArtifactName(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI: %w", err)
	}
	// For cases like "s3://bucket/path/to/file.txt"
	if parsed.Scheme != "" && parsed.Host != "" {
		return path.Base(parsed.Path), nil
	}
	// For "https://minio.local/bucket/path/to/file.txt"
	if parsed.Scheme != "" && parsed.Host == "" {
		return path.Base(parsed.Path), nil
	}
	// For URLs without a scheme, e.g. "bucket/path/to/file.txt"
	cleaned := strings.TrimSuffix(uri, "/")
	return path.Base(cleaned), nil
}

// ObjectStoreDependencies interface implementation for ImportLauncher

func (l *ImportLauncher) GetOpenedBucketCache() map[string]*blob.Bucket {
	return l.openedBucketCache
}

func (l *ImportLauncher) SetOpenedBucket(key string, bucket *blob.Bucket) {
	l.openedBucketCache[key] = bucket
}

func (l *ImportLauncher) GetLauncherConfig() *config.Config {
	return l.launcherConfig
}

func (l *ImportLauncher) GetK8sClient() kubernetes.Interface {
	return l.clientManager.K8sClient()
}

func (l *ImportLauncher) GetNamespace() string {
	return l.opts.Namespace
}
