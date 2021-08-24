package component

import (
	"context"
	"fmt"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ImporterLauncherOptions struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
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
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	metadataClient, err := metadata.NewClient(launcherV2Opts.MLMDServerAddress, launcherV2Opts.MLMDServerPort)
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
	// there's no need to pass any parameters, because pipeline and pipeline run context have been created by root DAG driver.
	pipeline, err := l.metadataClient.GetPipeline(ctx, l.importerLauncherOptions.PipelineName, l.importerLauncherOptions.RunID, "", "", "")
	if err != nil {
		return err
	}
	ecfg := &metadata.ExecutionConfig{
		TaskName:  l.task.GetTaskInfo().GetName(),
		PodName:   l.launcherV2Options.PodName,
		PodUID:    l.launcherV2Options.PodUID,
		Namespace: l.launcherV2Options.Namespace,
	}
	createdExecution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	return nil
}

func (l *ImportLauncher) FindOrNewArtifactToImport(ctx context.Context) (artifact *pb.Artifact, err error) {

}

func (l *ImportLauncher) ImportSpecToMLMDArtifact(ctx context.Context, importerSpec *pipelinespec.PipelineDeploymentConfig_ImporterSpec) (artifact *pb.Artifact, err error) {

	schema, err := getArtifactSchema(importerSpec.TypeSchema)
	if err != nil {
		return nil, fmt.Errorf("Failed to get schema from importer spec: %w", err)
	}
	artifactTypeId, err := l.metadataClient.GetOrInsertArtifactType(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to get or insert artifact type with schema %s: %w", schema, err)
	}

	if importerSpec.GetArtifactUri().GetConstantValue() == nil || importerSpec.GetArtifactUri().GetConstantValue().GetStringValue() == "" {
		return nil, fmt.Errorf("failed to get artifactUri from ImporterSpec")
	}
	artifactUri := importerSpec.GetArtifactUri().GetConstantValue().GetStringValue()
	state := pb.Artifact_LIVE

	return &pb.Artifact{
		TypeId: &artifactTypeId,
		State:  &state,
		Uri:    &artifactUri,
	}, nil

}
