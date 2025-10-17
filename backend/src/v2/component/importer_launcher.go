package component

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ImporterLauncherOptions struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
	// required, parent DAG execution ID
	ParentDagID int64
}

func (o *ImporterLauncherOptions) validate() error {
	if o == nil {
		return fmt.Errorf("empty importer launcher options")
	}
	if o.PipelineName == "" {
		return fmt.Errorf("importer launcher options: pipeline name is empty")
	}
	if o.RunID == "" {
		return fmt.Errorf("importer launcher options: Run ID is empty")
	}
	if o.ParentDagID == 0 {
		return fmt.Errorf("importer launcher options: Parent DAG ID is not provided")
	}
	return nil
}

type ImportLauncher struct {
	component               *pipelinespec.ComponentSpec
	importer                *pipelinespec.PipelineDeploymentConfig_ImporterSpec
	task                    *pipelinespec.PipelineTaskSpec
	launcherV2Options       LauncherV2Options
	importerLauncherOptions ImporterLauncherOptions

	// clients
	metadataClient *metadata.Client
	k8sClient      *kubernetes.Clientset
}

func NewImporterLauncher(ctx context.Context, componentSpecJSON, importerSpecJSON, taskSpecJSON string, launcherV2Opts *LauncherV2Options, importerLauncherOpts *ImporterLauncherOptions) (l *ImportLauncher, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create importer launcher: %w", err)
		}
	}()
	component := &pipelinespec.ComponentSpec{}
	err = protojson.Unmarshal([]byte(componentSpecJSON), component)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec: %w", err)
	}
	importer := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}
	err = protojson.Unmarshal([]byte(importerSpecJSON), importer)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal importer spec: %w", err)
	}
	task := &pipelinespec.PipelineTaskSpec{}
	err = protojson.Unmarshal([]byte(taskSpecJSON), task)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task spec: %w", err)
	}
	err = launcherV2Opts.validate()
	if err != nil {
		return nil, err
	}
	err = importerLauncherOpts.validate()
	if err != nil {
		return nil, err
	}
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	tlsCfg, err := util.GetTLSConfig(launcherV2Opts.CaCertPath)
	if err != nil {
		return nil, err
	}
	metadataClient, err := metadata.NewClient(launcherV2Opts.MLMDServerAddress, launcherV2Opts.MLMDServerPort, tlsCfg)
	if err != nil {
		return nil, err
	}
	return &ImportLauncher{
		component:               component,
		importer:                importer,
		task:                    task,
		launcherV2Options:       *launcherV2Opts,
		importerLauncherOptions: *importerLauncherOpts,
		metadataClient:          metadataClient,
		k8sClient:               k8sClient,
	}, nil
}

func (l *ImportLauncher) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute importer component: %w", err)
		}
	}()
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := l.metadataClient.GetPipeline(ctx, l.importerLauncherOptions.PipelineName, l.importerLauncherOptions.RunID, "", "", "", "")
	if err != nil {
		return err
	}
	ecfg := &metadata.ExecutionConfig{
		TaskName:  l.task.GetTaskInfo().GetName(),
		PodName:   l.launcherV2Options.PodName,
		PodUID:    l.launcherV2Options.PodUID,
		Namespace: l.launcherV2Options.Namespace,
		ExecutionType: func() metadata.ExecutionType {
			if l.importer.GetDownloadToWorkspace() {
				return metadata.ImporterWorkspaceExecutionTypeName
			}
			return metadata.ImporterExecutionTypeName
		}(),
		ParentDagID: l.importerLauncherOptions.ParentDagID,
	}
	createdExecution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return err
	}
	artifact, err := l.findOrNewArtifactToImport(ctx, createdExecution)
	if err != nil {
		return err
	}
	outputArtifactName, err := l.getOutPutArtifactName()
	if err != nil {
		return err
	}
	outputArtifact := &metadata.OutputArtifact{
		Name:     outputArtifactName,
		Artifact: artifact,
		Schema:   l.component.OutputDefinitions.Artifacts[outputArtifactName].GetArtifactType().GetInstanceSchema(),
	}
	outputArtifacts := []*metadata.OutputArtifact{outputArtifact}
	if err := l.metadataClient.PublishExecution(ctx, createdExecution, nil, outputArtifacts, pb.Execution_COMPLETE); err != nil {
		return fmt.Errorf("failed to publish results of importer execution to ML Metadata: %w", err)
	}

	return nil
}

func (l *ImportLauncher) findOrNewArtifactToImport(ctx context.Context, execution *metadata.Execution) (artifact *pb.Artifact, err error) {
	// TODO consider moving logic to package metadata so that *pb.Artifact won't get exposed outside of package metadata
	artifactToImport, err := l.ImportSpecToMLMDArtifact(ctx)
	if err != nil {
		return nil, err
	}
	if l.importer.Reimport {
		return artifactToImport, nil
	}
	matchedArtifact, err := l.metadataClient.FindMatchedArtifact(ctx, artifactToImport, execution.GetPipeline().GetCtxID())
	if err != nil {
		return nil, err
	}
	if matchedArtifact != nil {
		return matchedArtifact, nil
	}
	return artifactToImport, nil
}

func (l *ImportLauncher) ImportSpecToMLMDArtifact(ctx context.Context) (artifact *pb.Artifact, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create MLMD artifact from ImporterSpec: %w", err)
		}
	}()

	schema, err := getArtifactSchema(l.importer.TypeSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from importer spec: %w", err)
	}
	artifactTypeId, err := l.metadataClient.GetOrInsertArtifactType(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get or insert artifact type with schema %s: %w", schema, err)
	}

	// Resolve artifact URI. Can be one of two sources:
	// 1) Constant
	// 2) Runtime Parameter
	var artifactUri string
	if l.importer.GetArtifactUri().GetConstant() != nil {
		glog.Infof("Artifact URI as constant: %+v", l.importer.GetArtifactUri().GetConstant())
		artifactUri = l.importer.GetArtifactUri().GetConstant().GetStringValue()
		if artifactUri == "" {
			return nil, fmt.Errorf("empty Artifact URI constant value")
		}
	} else if l.importer.GetArtifactUri().GetRuntimeParameter() != "" {
		// When URI is provided using Runtime Parameter, need to retrieve it from dag execution in MLMD
		paramName := l.importer.GetArtifactUri().GetRuntimeParameter()
		taskInput, ok := l.task.GetInputs().GetParameters()[paramName]
		if !ok {
			return nil, fmt.Errorf("cannot find parameter %s in task input to fetch artifact uri", paramName)
		}
		componentInput := taskInput.GetComponentInputParameter()
		dag, err := l.metadataClient.GetDAG(ctx, l.importerLauncherOptions.ParentDagID)
		if err != nil {
			return nil, fmt.Errorf("error retrieving dag execution for parameter %s: %w", paramName, err)
		}
		glog.Infof("parent DAG: %+v", dag.Execution)
		inputParams, _, err := dag.Execution.GetParameters()
		if err != nil {
			return nil, fmt.Errorf("error retrieving input parameters from dag execution for parameter %s: %w", paramName, err)
		}
		v, ok := inputParams[componentInput]
		if !ok {
			return nil, fmt.Errorf("error resolving artifact URI: parent DAG does not have input parameter %s", componentInput)
		}
		artifactUri = v.GetStringValue()
		glog.Infof("Artifact URI from runtime parameter: %s", artifactUri)
		if artifactUri == "" {
			return nil, fmt.Errorf("empty artifact URI runtime value for parameter %s", paramName)
		}
	} else {
		return nil, fmt.Errorf("artifact uri not provided")
	}

	state := pb.Artifact_LIVE

	artifact = &pb.Artifact{
		TypeId:           &artifactTypeId,
		State:            &state,
		Uri:              &artifactUri,
		Properties:       make(map[string]*pb.Value),
		CustomProperties: make(map[string]*pb.Value),
	}
	if l.importer.Metadata != nil {
		for k, v := range l.importer.Metadata.Fields {
			value, err := metadata.StructValueToMLMDValue(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert structValue : %w", err)
			}
			artifact.CustomProperties[k] = value
		}
	}

	if strings.HasPrefix(artifactUri, "oci://") {
		// OCI artifacts are not supported when workspace is used
		if l.importer.GetDownloadToWorkspace() {
			return nil, fmt.Errorf("importer workspace download does not support OCI registries")
		}
		artifactType, err := metadata.SchemaToArtifactType(schema)
		if err != nil {
			return nil, fmt.Errorf("converting schema to artifact type failed: %w", err)
		}
		if *artifactType.Name != "system.Model" {
			return nil, fmt.Errorf("the %s artifact type does not support OCI registries", *artifactType.Name)
		}
		return artifact, nil
	}

	provider, err := objectstore.ParseProviderFromPath(artifactUri)
	if err != nil {
		return nil, fmt.Errorf("no provider scheme found in artifact URI: %s", artifactUri)
	}

	// Assume all imported artifacts will rely on execution environment for store provider session info
	storeSessionInfo := objectstore.SessionInfo{
		Provider: provider,
		Params: map[string]string{
			"fromEnv": "true",
		},
	}
	storeSessionInfoJSON, err := json.Marshal(storeSessionInfo)
	if err != nil {
		return nil, err
	}
	storeSessionInfoStr := string(storeSessionInfoJSON)
	artifact.CustomProperties["store_session_info"] = metadata.StringValue(storeSessionInfoStr)

	// Download the artifact into the workspace
	if l.importer.GetDownloadToWorkspace() {
		bucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(artifactUri)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bucket config while downloading artifact into workspace with uri %q: %w", artifactUri, err)
		}
		// Resolve and attach session info from kfp-launcher config for the artifact provider
		if cfg, cfgErr := config.FromConfigMap(ctx, l.k8sClient, l.launcherV2Options.Namespace); cfgErr != nil {
			glog.Warningf("failed to load launcher config for workspace download: %v", cfgErr)
		} else if cfg != nil {
			if sess, sessErr := cfg.GetStoreSessionInfo(artifactUri); sessErr != nil {
				glog.Warningf("failed to resolve store session info for %q: %v", artifactUri, sessErr)
			} else {
				bucketConfig.SessionInfo = &sess
			}
		}
		blobKey, err := bucketConfig.KeyFromURI(artifactUri)
		if err != nil {
			return nil, fmt.Errorf("failed to derive blob key from uri %q while downloading artifact into workspace: %w", artifactUri, err)
		}
		workspaceRoot := filepath.Join(WorkspaceMountPath, ".artifacts")
		if err := os.MkdirAll(workspaceRoot, 0755); err != nil {
			return nil, fmt.Errorf("failed to create workspace directory %q: %w", workspaceRoot, err)
		}
		bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.launcherV2Options.Namespace, bucketConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to open bucket for uri %q: %w", artifactUri, err)
		}
		defer bucket.Close()
		if err := objectstore.DownloadBlob(ctx, bucket, workspaceRoot, blobKey); err != nil {
			return nil, fmt.Errorf("failed to download artifact to workspace: %w", err)
		}
	}
	return artifact, nil
}

func (l *ImportLauncher) getOutPutArtifactName() (string, error) {
	outPutNames := make([]string, 0, len(l.component.GetOutputDefinitions().GetArtifacts()))
	for name := range l.component.GetOutputDefinitions().GetArtifacts() {
		outPutNames = append(outPutNames, name)
	}
	if len(outPutNames) != 1 {
		return "", fmt.Errorf("failed to extract output artifact name from componentOutputSpec")
	}
	return outPutNames[0], nil

}
