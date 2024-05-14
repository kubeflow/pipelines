// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package component

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LauncherV2Options struct {
	Namespace,
	PodName,
	PodUID,
	MLMDServerAddress,
	MLMDServerPort,
	PipelineName,
	RunID string
	// set to true if ml pipeline server is serving over tls
	MLPipelineTLSEnabled bool
}

type LauncherV2 struct {
	executionID   int64
	executorInput *pipelinespec.ExecutorInput
	component     *pipelinespec.ComponentSpec
	command       string
	args          []string
	options       LauncherV2Options

	// clients
	metadataClient metadata.ClientInterface
	k8sClient      kubernetes.Interface
	cacheClient    *cacheutils.Client
}

// Client is the struct to hold the Kubernetes Clientset
type kubernetesClient struct {
	Clientset kubernetes.Interface
}

func NewLauncherV2(ctx context.Context, executionID int64, executorInputJSON, componentSpecJSON string, cmdArgs []string, opts *LauncherV2Options) (l *LauncherV2, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create component launcher v2: %w", err)
		}
	}()
	if executionID == 0 {
		return nil, fmt.Errorf("must specify execution ID")
	}
	executorInput := &pipelinespec.ExecutorInput{}
	err = protojson.Unmarshal([]byte(executorInputJSON), executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal executor input: %w", err)
	}
	glog.Infof("input ComponentSpec:%s\n", prettyPrint(componentSpecJSON))
	component := &pipelinespec.ComponentSpec{}
	err = protojson.Unmarshal([]byte(componentSpecJSON), component)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec: %w\ncomponentSpec: %v", err, prettyPrint(componentSpecJSON))
	}
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("command and arguments are empty")
	}
	err = opts.validate()
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
	metadataClient, err := metadata.NewClient(opts.MLMDServerAddress, opts.MLMDServerPort)
	if err != nil {
		return nil, err
	}
	cacheClient, err := cacheutils.NewClient(opts.MLPipelineTLSEnabled)
	if err != nil {
		return nil, err
	}
	return &LauncherV2{
		executionID:    executionID,
		executorInput:  executorInput,
		component:      component,
		command:        cmdArgs[0],
		args:           cmdArgs[1:],
		options:        *opts,
		metadataClient: metadataClient,
		k8sClient:      k8sClient,
		cacheClient:    cacheClient,
	}, nil
}

func (l *LauncherV2) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute component: %w", err)
		}
	}()
	// publish execution regardless the task succeeds or not
	var execution *metadata.Execution
	var executorOutput *pipelinespec.ExecutorOutput
	var outputArtifacts []*metadata.OutputArtifact
	status := pb.Execution_FAILED
	defer func() {
		if perr := l.publish(ctx, execution, executorOutput, outputArtifacts, status); perr != nil {
			if err != nil {
				err = fmt.Errorf("failed to publish execution with error %w after execution failed: %w", perr, err)
			} else {
				err = perr
			}
		}
		glog.Infof("publish success.")
	}()
	executedStartedTime := time.Now().Unix()
	execution, err = l.prePublish(ctx)
	if err != nil {
		return err
	}
	fingerPrint := execution.FingerPrint()
	bucketSessionInfo, err := objectstore.GetSessionInfoFromString(execution.GetPipeline().GetPipelineBucketSession())
	if err != nil {
		return err
	}
	pipelineRoot := execution.GetPipeline().GetPipelineRoot()
	bucketConfig, err := objectstore.ParseBucketConfig(pipelineRoot, bucketSessionInfo)
	if err != nil {
		return err
	}
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, bucketConfig)
	if err != nil {
		return err
	}
	if err = prepareOutputFolders(l.executorInput); err != nil {
		return err
	}
	executorOutput, outputArtifacts, err = executeV2(ctx, l.executorInput, l.component, l.command, l.args, bucket, bucketConfig, l.metadataClient, l.options.Namespace, l.k8sClient)
	if err != nil {
		return err
	}
	status = pb.Execution_COMPLETE
	// if fingerPrint is not empty, it means this task enables cache but it does not hit cache, we need to create cache entry for this task
	if fingerPrint != "" {
		id := execution.GetID()
		if id == 0 {
			return fmt.Errorf("failed to get id from createdExecution")
		}
		task := &api.Task{
			//TODO how to differentiate between shared pipeline and namespaced pipeline
			PipelineName:    "pipeline/" + l.options.PipelineName,
			Namespace:       l.options.Namespace,
			RunId:           l.options.RunID,
			MlmdExecutionID: strconv.FormatInt(id, 10),
			CreatedAt:       &timestamp.Timestamp{Seconds: executedStartedTime},
			FinishedAt:      &timestamp.Timestamp{Seconds: time.Now().Unix()},
			Fingerprint:     fingerPrint,
		}
		return l.cacheClient.CreateExecutionCache(ctx, task)
	}
	return nil
}

func (l *LauncherV2) Info() string {
	content, err := protojson.Marshal(l.executorInput)
	if err != nil {
		content = []byte("{}")
	}
	return strings.Join([]string{
		"launcher info:",
		fmt.Sprintf("executorInput=%s\n", prettyPrint(string(content))),
	}, "\n")
}

func (o *LauncherV2Options) validate() error {
	empty := func(s string) bool { return len(s) == 0 }
	err := func(s string) error { return fmt.Errorf("invalid launcher options: must specify %s", s) }

	if empty(o.Namespace) {
		return err("Namespace")
	}
	if empty(o.PodName) {
		return err("PodName")
	}
	if empty(o.PodUID) {
		return err("PodUID")
	}
	if empty(o.MLMDServerAddress) {
		return err("MLMDServerAddress")
	}
	if empty(o.MLMDServerPort) {
		return err("MLMDServerPort")
	}
	return nil
}

// publish pod info to MLMD, before running user command
func (l *LauncherV2) prePublish(ctx context.Context) (execution *metadata.Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to pre-publish Pod info to ML Metadata: %w", err)
		}
	}()
	execution, err = l.metadataClient.GetExecution(ctx, l.executionID)
	if err != nil {
		return nil, err
	}
	ecfg := &metadata.ExecutionConfig{
		PodName:   l.options.PodName,
		PodUID:    l.options.PodUID,
		Namespace: l.options.Namespace,
	}
	return l.metadataClient.PrePublishExecution(ctx, execution, ecfg)
}

// TODO(Bobgy): consider passing output artifacts info from executor output.
func (l *LauncherV2) publish(
	ctx context.Context,
	execution *metadata.Execution,
	executorOutput *pipelinespec.ExecutorOutput,
	outputArtifacts []*metadata.OutputArtifact,
	status pb.Execution_State,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish results to ML Metadata: %w", err)
		}
	}()
	outputParameters := executorOutput.GetParameterValues()
	// TODO(Bobgy): upload output artifacts.
	// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
	// to publish output artifacts to the context too.
	// return l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts, pb.Execution_COMPLETE)
	return l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts, status)
}

func executeV2(
	ctx context.Context,
	executorInput *pipelinespec.ExecutorInput,
	component *pipelinespec.ComponentSpec,
	cmd string,
	args []string,
	bucket *blob.Bucket,
	bucketConfig *objectstore.Config,
	metadataClient metadata.ClientInterface,
	namespace string,
	k8sClient kubernetes.Interface,
) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {

	// Add parameter default values to executorInput, if there is not already a user input.
	// This process is done in the launcher because we let the component resolve default values internally.
	// Variable executorInputWithDefault is a copy so we don't alter the original data.
	executorInputWithDefault, err := addDefaultParams(executorInput, component)
	if err != nil {
		return nil, nil, err
	}

	// Fill in placeholders with runtime values.
	placeholders, err := getPlaceholders(executorInputWithDefault)
	if err != nil {
		return nil, nil, err
	}
	for placeholder, replacement := range placeholders {
		cmd = strings.ReplaceAll(cmd, placeholder, replacement)
	}
	for i := range args {
		arg := args[i]
		for placeholder, replacement := range placeholders {
			arg = strings.ReplaceAll(arg, placeholder, replacement)
		}
		args[i] = arg
	}

	executorOutput, err := execute(ctx, executorInput, cmd, args, bucket, bucketConfig, namespace, k8sClient)
	if err != nil {
		return nil, nil, err
	}
	// These are not added in execute(), because execute() is shared between v2 compatible and v2 engine launcher.
	// In v2 compatible mode, we get output parameter info from runtimeInfo. In v2 engine, we get it from component spec.
	// Because of the difference, we cannot put parameter collection logic in one method.
	err = collectOutputParameters(executorInput, executorOutput, component)
	if err != nil {
		return nil, nil, err
	}
	// TODO(Bobgy): should we log metadata per each artifact, or batched after uploading all artifacts.
	outputArtifacts, err := uploadOutputArtifacts(ctx, executorInput, executorOutput, uploadOutputArtifactsOptions{
		bucketConfig:   bucketConfig,
		bucket:         bucket,
		metadataClient: metadataClient,
	})
	if err != nil {
		return nil, nil, err
	}
	// TODO(Bobgy): only return executor output. Merge info in output artifacts
	// to executor output.
	return executorOutput, outputArtifacts, nil
}

// collectOutputParameters collect output parameters from local disk and add them
// to executor output.
func collectOutputParameters(executorInput *pipelinespec.ExecutorInput, executorOutput *pipelinespec.ExecutorOutput, component *pipelinespec.ComponentSpec) error {
	if executorOutput.ParameterValues == nil {
		executorOutput.ParameterValues = make(map[string]*structpb.Value)
	}
	outputParameters := executorOutput.GetParameterValues()
	for name, param := range executorInput.GetOutputs().GetParameters() {
		_, ok := outputParameters[name]
		if ok {
			// If the output parameter was already specified in output metadata file,
			// we don't need to collect it from file, because output metadata file has
			// the highest priority.
			continue
		}
		paramSpec, ok := component.GetOutputDefinitions().GetParameters()[name]
		if !ok {
			return fmt.Errorf("failed to find output parameter name=%q in component spec", name)
		}
		msg := func(err error) error {
			return fmt.Errorf("failed to read output parameter name=%q type=%q path=%q: %w", name, paramSpec.GetParameterType(), param.GetOutputFile(), err)
		}
		b, err := ioutil.ReadFile(param.GetOutputFile())
		if err != nil {
			return msg(err)
		}
		value, err := metadata.TextToPbValue(string(b), paramSpec.GetParameterType())
		if err != nil {
			return msg(err)
		}
		outputParameters[name] = value
	}
	return nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(prettyJSON.Bytes())
}

const OutputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

func execute(
	ctx context.Context,
	executorInput *pipelinespec.ExecutorInput,
	cmd string,
	args []string,
	bucket *blob.Bucket,
	bucketConfig *objectstore.Config,
	namespace string,
	k8sClient kubernetes.Interface,
) (*pipelinespec.ExecutorOutput, error) {
	if err := downloadArtifacts(ctx, executorInput, bucket, bucketConfig, namespace, k8sClient); err != nil {
		return nil, err
	}
	if err := prepareOutputFolders(executorInput); err != nil {
		return nil, err
	}

	// Run user program.
	executor := exec.Command(cmd, args...)
	executor.Stdin = os.Stdin
	executor.Stdout = os.Stdout
	executor.Stderr = os.Stderr
	defer glog.Flush()
	if err := executor.Run(); err != nil {
		return nil, err
	}

	// Collect outputs from output metadata file.
	return getExecutorOutputFile(executorInput.GetOutputs().GetOutputFile())
}

type uploadOutputArtifactsOptions struct {
	bucketConfig   *objectstore.Config
	bucket         *blob.Bucket
	metadataClient metadata.ClientInterface
}

func uploadOutputArtifacts(ctx context.Context, executorInput *pipelinespec.ExecutorInput, executorOutput *pipelinespec.ExecutorOutput, opts uploadOutputArtifactsOptions) ([]*metadata.OutputArtifact, error) {
	// Register artifacts with MLMD.
	outputArtifacts := make([]*metadata.OutputArtifact, 0, len(executorInput.GetOutputs().GetArtifacts()))
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := artifactList.Artifacts[0]

		// Merge executor output artifact info with executor input
		if list, ok := executorOutput.Artifacts[name]; ok && len(list.Artifacts) > 0 {
			mergeRuntimeArtifacts(list.Artifacts[0], outputArtifact)
		}

		// Upload artifacts from local path to remote storages.
		localDir, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			glog.Warningf("Output Artifact %q does not have a recognized storage URI %q. Skipping uploading to remote storage.", name, outputArtifact.Uri)
		} else {
			blobKey, err := opts.bucketConfig.KeyFromURI(outputArtifact.Uri)
			if err != nil {
				return nil, fmt.Errorf("failed to upload output artifact %q: %w", name, err)
			}
			if err := objectstore.UploadBlob(ctx, opts.bucket, localDir, blobKey); err != nil {
				//  We allow components to not produce output files
				if errors.Is(err, os.ErrNotExist) {
					glog.Warningf("Local filepath %q does not exist", localDir)
				} else {
					return nil, fmt.Errorf("failed to upload output artifact %q to remote storage URI %q: %w", name, outputArtifact.Uri, err)
				}
			}
		}

		// Write out the metadata.
		metadataErr := func(err error) error {
			return fmt.Errorf("unable to produce MLMD artifact for output %q: %w", name, err)
		}
		// TODO(neuromage): Consider batching these instead of recording one by one.
		schema, err := getArtifactSchema(outputArtifact.GetType())
		if err != nil {
			return nil, fmt.Errorf("failed to determine schema for output %q: %w", name, err)
		}
		mlmdArtifact, err := opts.metadataClient.RecordArtifact(ctx, name, schema, outputArtifact, pb.Artifact_LIVE)
		if err != nil {
			return nil, metadataErr(err)
		}
		outputArtifacts = append(outputArtifacts, mlmdArtifact)
	}
	return outputArtifacts, nil
}

func downloadArtifacts(ctx context.Context, executorInput *pipelinespec.ExecutorInput, defaultBucket *blob.Bucket, defaultBucketConfig *objectstore.Config, namespace string, k8sClient kubernetes.Interface) error {
	// Read input artifact metadata.
	nonDefaultBuckets, err := fetchNonDefaultBuckets(ctx, executorInput.GetInputs().GetArtifacts(), defaultBucketConfig, namespace, k8sClient)
	closeNonDefaultBuckets := func(buckets map[string]*blob.Bucket) {
		for name, bucket := range nonDefaultBuckets {
			if closeBucketErr := bucket.Close(); closeBucketErr != nil {
				glog.Warningf("failed to close bucket %q: %q", name, err.Error())
			}
		}
	}
	defer closeNonDefaultBuckets(nonDefaultBuckets)
	if err != nil {
		return fmt.Errorf("failed to fetch non default buckets: %w", err)
	}
	for name, artifactList := range executorInput.GetInputs().GetArtifacts() {
		// TODO(neuromage): Support concat-based placholders for arguments.
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		inputArtifact := artifactList.Artifacts[0]
		localPath, err := localPathForURI(inputArtifact.Uri)
		if err != nil {
			glog.Warningf("Input Artifact %q does not have a recognized storage URI %q. Skipping downloading to local path.", name, inputArtifact.Uri)
			continue
		}
		// Copy artifact to local storage.
		copyErr := func(err error) error {
			return fmt.Errorf("failed to download input artifact %q from remote storage URI %q: %w", name, inputArtifact.Uri, err)
		}
		// TODO: Selectively copy artifacts for which .path was actually specified
		// on the command line.
		bucket := defaultBucket
		bucketConfig := defaultBucketConfig
		if !strings.HasPrefix(inputArtifact.Uri, defaultBucketConfig.PrefixedBucket()) {
			nonDefaultBucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(inputArtifact.Uri)
			if err != nil {
				return fmt.Errorf("failed to parse bucketConfig for output artifact %q with uri %q: %w", name, inputArtifact.GetUri(), err)
			}
			nonDefaultBucket, ok := nonDefaultBuckets[nonDefaultBucketConfig.PrefixedBucket()]
			if !ok {
				return fmt.Errorf("failed to get bucket when downloading input artifact %s with bucket key %s: %w", name, nonDefaultBucketConfig.PrefixedBucket(), err)
			}
			bucket = nonDefaultBucket
			bucketConfig = nonDefaultBucketConfig
		}
		blobKey, err := bucketConfig.KeyFromURI(inputArtifact.Uri)
		if err != nil {
			return copyErr(err)
		}
		if err := objectstore.DownloadBlob(ctx, bucket, localPath, blobKey); err != nil {
			return copyErr(err)
		}

	}
	return nil
}

func fetchNonDefaultBuckets(
	ctx context.Context,
	artifacts map[string]*pipelinespec.ArtifactList,
	defaultBucketConfig *objectstore.Config,
	namespace string,
	k8sClient kubernetes.Interface,
) (buckets map[string]*blob.Bucket, err error) {
	nonDefaultBuckets := make(map[string]*blob.Bucket)
	for name, artifactList := range artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		artifact := artifactList.Artifacts[0]
		// The artifact does not belong under the s3 path for this run
		// Reasons:
		// 1. Artifact is cached from a different run, so it may still be in the default bucket, but under a different run id subpath
		// 2. Artifact is imported from a different bucket, or obj store
		// a. If imported, artifact bucket can still be specified in kfp-launcher config (not implemented)
		// b. If imported, artifact bucket can not be in kfp-launcher config, in this case, return no session and rely on env for aws config
		if !strings.HasPrefix(artifact.Uri, defaultBucketConfig.PrefixedBucket()) {
			nonDefaultBucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(artifact.Uri)
			if err != nil {
				return nonDefaultBuckets, fmt.Errorf("failed to parse bucketConfig for output artifact %q with uri %q: %w", name, artifact.GetUri(), err)
			}
			// If the run is cached, it will be in the same bucket but under a different path, re-use the default bucket
			// session in this case.
			if (nonDefaultBucketConfig.Scheme == defaultBucketConfig.Scheme) && (nonDefaultBucketConfig.BucketName == defaultBucketConfig.BucketName) {
				nonDefaultBucketConfig.Session = defaultBucketConfig.Session
			}
			nonDefaultBucket, err := objectstore.OpenBucket(ctx, k8sClient, namespace, nonDefaultBucketConfig)
			if err != nil {
				return nonDefaultBuckets, fmt.Errorf("failed to open bucket for output artifact %q with uri %q: %w", name, artifact.GetUri(), err)
			}
			nonDefaultBuckets[nonDefaultBucketConfig.PrefixedBucket()] = nonDefaultBucket
		}

	}
	return nonDefaultBuckets, nil

}

// Add executor input placeholders to provided map.
func getPlaceholders(executorInput *pipelinespec.ExecutorInput) (placeholders map[string]string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get placeholders: %w", err)
		}
	}()
	placeholders = make(map[string]string)
	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}
	placeholders["{{$}}"] = string(executorInputJSON)

	// Read input artifact metadata.
	for name, artifactList := range executorInput.GetInputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		inputArtifact := artifactList.Artifacts[0]

		// Prepare input uri placeholder.
		key := fmt.Sprintf(`{{$.inputs.artifacts['%s'].uri}}`, name)
		placeholders[key] = inputArtifact.Uri

		localPath, err := localPathForURI(inputArtifact.Uri)
		if err != nil {
			// Input Artifact does not have a recognized storage URI
			continue
		}

		// Prepare input path placeholder.
		key = fmt.Sprintf(`{{$.inputs.artifacts['%s'].path}}`, name)
		placeholders[key] = localPath
	}

	// Prepare output artifact placeholders.
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].uri}}`, name)] = outputArtifact.Uri

		localPath, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			return nil, fmt.Errorf("resolve output artifact %q's local path: %w", name, err)
		}
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].path}}`, name)] = localPath
	}

	// Prepare input parameter placeholders.
	for name, parameter := range executorInput.GetInputs().GetParameterValues() {
		key := fmt.Sprintf(`{{$.inputs.parameters['%s']}}`, name)
		switch t := parameter.Kind.(type) {
		case *structpb.Value_StringValue:
			placeholders[key] = parameter.GetStringValue()
		case *structpb.Value_NumberValue:
			placeholders[key] = strconv.FormatFloat(parameter.GetNumberValue(), 'f', -1, 64)
		case *structpb.Value_BoolValue:
			placeholders[key] = strconv.FormatBool(parameter.GetBoolValue())
		case *structpb.Value_ListValue:
			b, err := json.Marshal(parameter.GetListValue())
			if err != nil {
				return nil, fmt.Errorf("failed to JSON-marshal list input parameter %q: %w", name, err)
			}
			placeholders[key] = string(b)
		case *structpb.Value_StructValue:
			b, err := json.Marshal(parameter.GetStructValue())
			if err != nil {
				return nil, fmt.Errorf("failed to JSON-marshal dict input parameter %q: %w", name, err)
			}
			placeholders[key] = string(b)
		default:
			return nil, fmt.Errorf("unknown PipelineSpec Value type %T", t)
		}
	}

	// Prepare output parameter placeholders.
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		key := fmt.Sprintf(`{{$.outputs.parameters['%s'].output_file}}`, name)
		placeholders[key] = parameter.OutputFile
	}

	return placeholders, nil
}

func getArtifactSchema(schema *pipelinespec.ArtifactTypeSchema) (string, error) {
	switch t := schema.Kind.(type) {
	case *pipelinespec.ArtifactTypeSchema_InstanceSchema:
		return t.InstanceSchema, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaTitle:
		return "title: " + t.SchemaTitle, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaUri:
		return "", fmt.Errorf("SchemaUri is unsupported")
	default:
		return "", fmt.Errorf("unknown type %T in ArtifactTypeSchema %+v", t, schema)
	}
}

func mergeRuntimeArtifacts(src, dst *pipelinespec.RuntimeArtifact) {
	if len(src.Uri) > 0 {
		dst.Uri = src.Uri
	}

	if src.Metadata != nil {
		if dst.Metadata == nil {
			dst.Metadata = src.Metadata
		} else {
			for k, v := range src.Metadata.Fields {
				dst.Metadata.Fields[k] = v
			}
		}
	}
}

func getExecutorOutputFile(path string) (*pipelinespec.ExecutorOutput, error) {
	// collect user executor output file
	executorOutput := &pipelinespec.ExecutorOutput{
		ParameterValues: map[string]*structpb.Value{},
		Artifacts:       map[string]*pipelinespec.ArtifactList{},
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			glog.Infof("output metadata file does not exist in %s", path)
			// If file doesn't exist, return an empty ExecutorOutput.
			return executorOutput, nil
		} else {
			return nil, fmt.Errorf("failed to stat output metadata file %q: %w", path, err)
		}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read output metadata file %q: %w", path, err)
	}
	glog.Infof("ExecutorOutput: %s", prettyPrint(string(b)))

	if err := protojson.Unmarshal(b, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshall ExecutorOutput in file %q: %w", path, err)
	}

	return executorOutput, nil
}

func localPathForURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "gs://") {
		return "/gcs/" + strings.TrimPrefix(uri, "gs://"), nil
	}
	if strings.HasPrefix(uri, "minio://") {
		return "/minio/" + strings.TrimPrefix(uri, "minio://"), nil
	}
	if strings.HasPrefix(uri, "s3://") {
		return "/s3/" + strings.TrimPrefix(uri, "s3://"), nil
	}
	return "", fmt.Errorf("failed to generate local path for URI %s: unsupported storage scheme", uri)
}

func prepareOutputFolders(executorInput *pipelinespec.ExecutorInput) error {
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		dir := filepath.Dir(parameter.OutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]

		localPath, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			return fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
		}

		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
		}
	}

	return nil
}

// Adds default parameter values if there is no user provided value
func addDefaultParams(
	executorInput *pipelinespec.ExecutorInput,
	component *pipelinespec.ComponentSpec,
) (*pipelinespec.ExecutorInput, error) {
	// Make a deep copy so we don't alter the original data
	executorInputWithDefaultMsg := proto.Clone(executorInput)
	executorInputWithDefault, ok := executorInputWithDefaultMsg.(*pipelinespec.ExecutorInput)
	if !ok {
		return nil, fmt.Errorf("bug: cloned executor input message does not have expected type")
	}

	if executorInputWithDefault.GetInputs().GetParameterValues() == nil {
		executorInputWithDefault.Inputs.ParameterValues = make(map[string]*structpb.Value)
	}
	for name, value := range component.GetInputDefinitions().GetParameters() {
		_, hasInput := executorInputWithDefault.GetInputs().GetParameterValues()[name]
		if value.GetDefaultValue() != nil && !hasInput {
			executorInputWithDefault.GetInputs().GetParameterValues()[name] = value.GetDefaultValue()
		}
	}
	return executorInputWithDefault, nil
}
