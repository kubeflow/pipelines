package component

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
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
	// Determine execution type
	executionType := metadata.ExecutionType(metadata.ImporterExecutionTypeName)
	if l.importer.GetDownloadToWorkspace() {
		executionType = metadata.ExecutionType(metadata.ImporterWorkspaceExecutionTypeName)
	}

	ecfg := &metadata.ExecutionConfig{
		TaskName:      l.task.GetTaskInfo().GetName(),
		PodName:       l.launcherV2Options.PodName,
		PodUID:        l.launcherV2Options.PodUID,
		Namespace:     l.launcherV2Options.Namespace,
		ExecutionType: executionType,
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

	if strings.HasPrefix(artifactUri, "huggingface://") {
		// HuggingFace artifacts
		if err := l.handleHuggingFaceImport(ctx, artifactUri, artifact); err != nil {
			return nil, err
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
		bucketConfig, err := l.resolveBucketConfigForURI(ctx, artifactUri)
		if err != nil {
			return nil, err
		}
		localPath, err := LocalWorkspacePathForURI(artifactUri)
		if err != nil {
			return nil, fmt.Errorf("failed to get local path for uri %q: %w", artifactUri, err)
		}
		blobKey, err := bucketConfig.KeyFromURI(artifactUri)
		if err != nil {
			return nil, fmt.Errorf("failed to derive blob key from uri %q while downloading artifact into workspace: %w", artifactUri, err)
		}
		bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.launcherV2Options.Namespace, bucketConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to open bucket for uri %q: %w", artifactUri, err)
		}
		defer bucket.Close()
		glog.Infof("Downloading artifact %q (blob key %q) to workspace path %q", artifactUri, blobKey, localPath)
		if err := objectstore.DownloadBlob(ctx, bucket, localPath, blobKey); err != nil {
			return nil, fmt.Errorf("failed to download artifact to workspace: %w", err)
		}
	}
	return artifact, nil
}

func (l *ImportLauncher) handleHuggingFaceImport(ctx context.Context, artifactURI string, artifact *pb.Artifact) error {
	parts := strings.TrimPrefix(artifactURI, "huggingface://")

	if parts == artifactURI {
		return fmt.Errorf("invalid artifact URI: %q\n"+
			"For HuggingFace Hub models and datasets, use the 'huggingface://' URI scheme.\n"+
			"Examples:\n"+
			"  huggingface://gpt2\n"+
			"  huggingface://meta-llama/Llama-2-7b\n"+
			"  huggingface://wikitext?repo_type=dataset",
			artifactURI)
	}

	var queryStr string
	if idx := strings.Index(parts, "?"); idx != -1 {
		queryStr = parts[idx+1:]
		parts = parts[:idx]
	}

	pathParts := strings.Split(parts, "/")
	// An empty path ("huggingface://") yields [""] after Split; treat this as invalid.
	if len(pathParts) < 1 || (len(pathParts) == 1 && pathParts[0] == "") {
		return fmt.Errorf("invalid HuggingFace URI format: %q, expected huggingface://repo_id[/revision]", artifactURI)
	}

	repoID := strings.Join(pathParts, "/")
	revision := "main"

	// Distinguish revisions from file paths:
	// - Files have extensions (.bin, .safetensors, .json, .txt, etc.)
	// - Revisions are branch/tag names (main, master, v1.0, develop, etc.)
	// - Repo names can have slashes (org/repo) and no dots (e.g., meta-llama/Llama-2-7b)
	//
	// Heuristic for 2-part paths (repo/X): If X looks like a common revision name, treat it as revision.
	// Heuristic for 3+ part paths (org/repo/X): If X has no dot, treat it as revision.
	if len(pathParts) >= 2 && pathParts[len(pathParts)-1] != "" {
		lastPart := pathParts[len(pathParts)-1]
		// Files have extensions
		switch {
		case strings.Contains(lastPart, "."):
			// Keep as part of repoID (it's a file path)
		case len(pathParts) == 2:
			// For 2-part paths, only treat as revision if it matches common branch/tag patterns
			commonRevisions := map[string]bool{
				"main": true, "master": true, "develop": true, "dev": true,
				"stable": true, "latest": true,
			}
			// Also match version patterns like v1.0, v2.1.3, release-1.0
			lowerLast := strings.ToLower(lastPart)
			isCommonRevision := commonRevisions[lowerLast]
			isVersionPattern := strings.HasPrefix(lowerLast, "v") || strings.HasPrefix(lowerLast, "release")

			if isCommonRevision || isVersionPattern {
				revision = lastPart
				repoID = pathParts[0]
			}
			// Otherwise, treat 2-part as org/repo (no revision extraction)
		default:
			// For 3+ parts, assume last part without dot is a revision
			revision = lastPart
			repoID = strings.Join(pathParts[:len(pathParts)-1], "/")
		}
	}

	repoType := "model"
	allowPatterns := ""
	ignorePatterns := ""
	supportedParams := map[string]bool{
		"repo_type":       true,
		"allow_patterns":  true,
		"ignore_patterns": true,
	}
	if queryStr != "" {
		vals, err := url.ParseQuery(queryStr)
		if err != nil {
			return fmt.Errorf("failed to parse query parameters for %q: %w", artifactURI, err)
		}
		// Check for unsupported parameters and warn
		for param := range vals {
			if !supportedParams[param] {
				glog.Warningf("Parameter %q is not supported by the KFP HuggingFace importer and will be ignored. "+
					"Supported parameters: repo_type, allow_patterns, ignore_patterns", param)
			}
		}
		if v := vals.Get("repo_type"); v != "" {
			repoType = v
		}
		if v := vals.Get("allow_patterns"); v != "" {
			allowPatterns = v
		}
		if v := vals.Get("ignore_patterns"); v != "" {
			ignorePatterns = v
		}
	}

	glog.Infof("Downloading HuggingFace repo_id=%q revision=%q repo_type=%q allow_patterns=%q ignore_patterns=%q", repoID, revision, repoType, allowPatterns, ignorePatterns)

	storeSessionInfo := objectstore.SessionInfo{
		Provider: "huggingface",
		Params: map[string]string{
			"fromEnv": "true",
		},
	}
	storeSessionInfoJSON, err := json.Marshal(storeSessionInfo)
	if err != nil {
		return err
	}
	storeSessionInfoStr := string(storeSessionInfoJSON)
	artifact.CustomProperties["store_session_info"] = metadata.StringValue(storeSessionInfoStr)

	artifact.CustomProperties["hf_repo_id"] = metadata.StringValue(repoID)
	artifact.CustomProperties["hf_revision"] = metadata.StringValue(revision)
	artifact.CustomProperties["hf_repo_type"] = metadata.StringValue(repoType)
	if allowPatterns != "" {
		artifact.CustomProperties["hf_allow_patterns"] = metadata.StringValue(allowPatterns)
	}
	if ignorePatterns != "" {
		artifact.CustomProperties["hf_ignore_patterns"] = metadata.StringValue(ignorePatterns)
	}

	if l.importer.GetDownloadToWorkspace() {
		if err := l.downloadHuggingFaceModel(ctx, repoID, revision, repoType, allowPatterns, ignorePatterns); err != nil {
			return fmt.Errorf("failed to download HuggingFace model: %w", err)
		}
	}

	return nil
}

func (l *ImportLauncher) downloadHuggingFaceModel(ctx context.Context, repoID, revision, repoType, allowPatterns, ignorePatterns string) error {
	artifactURI := fmt.Sprintf("huggingface://%s", repoID)
	if revision != "main" {
		artifactURI = fmt.Sprintf("huggingface://%s/%s", repoID, revision)
	}

	bucketConfig, err := l.resolveBucketConfigForURI(ctx, artifactURI)
	if err != nil {
		return err
	}

	localPath, err := LocalWorkspacePathForURI(artifactURI)
	if err != nil {
		return fmt.Errorf("failed to get local path for uri %q: %w", artifactURI, err)
	}

	hfToken := ""
	if bucketConfig.SessionInfo != nil {
		token, err := objectstore.GetHuggingFaceTokenFromK8sSecret(ctx, l.k8sClient, l.launcherV2Options.Namespace, bucketConfig.SessionInfo)
		if err != nil {
			glog.Warningf("Failed to retrieve HuggingFace token from session info: %v. Proceeding with unauthenticated download (public models only)", err)
		} else {
			hfToken = token
		}
	}

	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", localPath, err)
	}

	// Detect if repoID points to a specific file: if the last path component contains a dot,
	// it likely has a file extension (.bin, .safetensors, .json, etc.)
	isSpecificFile := false
	if strings.Contains(repoID, "/") {
		lastSlashIdx := strings.LastIndex(repoID, "/")
		lastComponent := repoID[lastSlashIdx+1:]
		isSpecificFile = strings.Contains(lastComponent, ".")
	}

	if isSpecificFile {
		lastSlash := strings.LastIndex(repoID, "/")
		filename := repoID[lastSlash+1:]
		actualRepoID := repoID[:lastSlash]

		glog.Infof("Downloading specific file from HuggingFace repo_id=%q revision=%q filename=%q repo_type=%q to %q", actualRepoID, revision, filename, repoType, localPath)

		pythonScript := fmt.Sprintf(`
from huggingface_hub import hf_hub_download
import inspect
import os
repo_id = %q
revision = %q
filename = %q
repo_type = %q
local_dir = %q
token = os.environ.get('HF_TOKEN', None)
sig = inspect.signature(hf_hub_download)
kwargs = {}
for param in ['repo_id', 'filename', 'revision', 'repo_type', 'local_dir', 'token']:
    if param in sig.parameters:
        kwargs[param] = locals()[param]
hf_hub_download(**kwargs)
`, actualRepoID, revision, filename, repoType, localPath)

		cmd := exec.CommandContext(ctx, "python3", "-c", pythonScript)
		if hfToken != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("HF_TOKEN=%s", hfToken))
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to download HuggingFace file: %w, output: %s", err, string(output))
		}
		glog.Infof("Successfully downloaded HuggingFace file to %q", localPath)
	} else {
		glog.Infof("Downloading HuggingFace repo_id=%q revision=%q repo_type=%q to %q", repoID, revision, repoType, localPath)

		pythonScript := fmt.Sprintf(`
from huggingface_hub import snapshot_download
import inspect
import os
repo_id = %q
revision = %q
repo_type = %q
local_dir = %q
allow_patterns = %q
ignore_patterns = %q
token = os.environ.get('HF_TOKEN', None)
# Build kwargs dynamically based on what parameters the function accepts
# This ensures compatibility with current and future versions of huggingface-hub
sig = inspect.signature(snapshot_download)
kwargs = {}
base_params = ['repo_id', 'revision', 'repo_type', 'local_dir', 'token']
optional_params = {'allow_patterns': allow_patterns, 'ignore_patterns': ignore_patterns}
for param in base_params:
    if param in sig.parameters:
        kwargs[param] = locals()[param]
for param, value in optional_params.items():
    if value:
        if param in sig.parameters:
            kwargs[param] = value
        else:
            import warnings
            warnings.warn(
                f"Optional parameter '{param}' was specified but is not supported by the installed version of huggingface-hub; it will be ignored."
            )
snapshot_download(**kwargs)
`, repoID, revision, repoType, localPath, allowPatterns, ignorePatterns)

		cmd := exec.CommandContext(ctx, "python3", "-c", pythonScript)
		if hfToken != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("HF_TOKEN=%s", hfToken))
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to download HuggingFace model: %w, output: %s", err, string(output))
		}
		glog.Infof("Successfully downloaded HuggingFace repo to %q", localPath)
	}
	return nil
}

// resolveBucketConfigForURI parses bucket configuration for a given artifact URI and
// attaches session information from the kfp-launcher ConfigMap when available.
func (l *ImportLauncher) resolveBucketConfigForURI(ctx context.Context, uri string) (*objectstore.Config, error) {
	bucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bucket config while resolving uri %q: %w", uri, err)
	}
	// Resolve and attach session info from kfp-launcher config for the artifact provider
	if cfg, err := config.FromConfigMap(ctx, l.k8sClient, l.launcherV2Options.Namespace); err != nil {
		glog.Warningf("failed to load launcher config while resolving bucket config: %v", err)
	} else if cfg != nil {
		if sess, err := cfg.GetStoreSessionInfo(uri); err != nil {
			glog.Warningf("failed to resolve store session info for %q: %v", uri, err)
		} else {
			bucketConfig.SessionInfo = &sess
		}
	}
	return bucketConfig, nil
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
