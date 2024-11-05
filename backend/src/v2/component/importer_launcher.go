package component

import (
	"context"
	"fmt"

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
	metadataClient, err := metadata.NewClient(launcherV2Opts.MLMDServerAddress, launcherV2Opts.MLMDServerPort, launcherV2Opts.MetadataTLSEnabled, launcherV2Opts.CaCertPath)
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
		TaskName:      l.task.GetTaskInfo().GetName(),
		PodName:       l.launcherV2Options.PodName,
		PodUID:        l.launcherV2Options.PodUID,
		Namespace:     l.launcherV2Options.Namespace,
		ExecutionType: metadata.ImporterExecutionTypeName,
		ParentDagID:   l.importerLauncherOptions.ParentDagID,
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
