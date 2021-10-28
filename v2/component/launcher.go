// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package component contains types to run an ML component from the launcher
// binary.
package component

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/cacheutils"
	"github.com/kubeflow/pipelines/v2/config"
	api "github.com/kubeflow/pipelines/v2/kfp-api"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const OutputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

// Launcher is used to launch KFP components. It handles the recording of the
// appropriate metadata for lineage.
type Launcher struct {
	options                 *LauncherOptions
	runtimeInfo             *runtimeInfo
	placeholderReplacements map[string]string
	metadataClient          *metadata.Client
	cacheClient             *cacheutils.Client
	bucketConfig            *objectstore.Config
	k8sClient               *kubernetes.Clientset
	cmdArgs                 []string
}

// LauncherOptions are options used when creating Launcher.
type LauncherOptions struct {
	PipelineName      string
	PipelineRoot      string
	RunID             string
	RunResource       string
	Namespace         string
	PodName           string
	PodUID            string
	TaskName          string
	Image             string
	MLMDServerAddress string
	MLMDServerPort    string
	EnableCaching     bool
}

func (o *LauncherOptions) validate() error {
	empty := func(s string) bool { return len(s) == 0 }
	err := func(s string) error { return fmt.Errorf("Invalid launcher options: must specify %s", s) }

	if empty(o.PipelineName) {
		return err("PipelineName")
	}
	if empty(o.RunID) {
		return err("RunID")
	}
	if empty(o.RunResource) {
		return err("RunResource")
	}
	if empty(o.Namespace) {
		return err("Namespace")
	}
	if empty(o.PodName) {
		return err("PodName")
	}
	if empty(o.PodUID) {
		return err("PodUID")
	}
	if empty(o.TaskName) {
		return err("TaskName")
	}
	if empty(o.MLMDServerAddress) {
		return err("MLMDServerAddress")
	}
	if empty(o.MLMDServerPort) {
		return err("MLMDServerPort")
	}
	if strings.HasPrefix(o.PipelineName, "namespace/") {
		s := strings.SplitN(o.PipelineName, "/", 4)
		if len(s) != 4 {
			return fmt.Errorf("invalid PipelineName options for namespaced pipelines, need to follow 'namespace/${namespace}/pipeline/${pipelineName}': %s", o.PipelineName)
		}
		namespace := s[1]
		if namespace != o.Namespace {
			return fmt.Errorf("the namespace %s extracted from pipelineName is not equal to the namespace %s in launcher options", namespace, o.Namespace)
		}
	}
	return nil
}

// NewLauncher creates a new launcher object using the JSON-encoded runtimeInfo
// and specified options.
func NewLauncher(ctx context.Context, runtimeInfo string, options *LauncherOptions) (*Launcher, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize kubernetes client set: %w", err)
	}
	if err := options.validate(); err != nil {
		return nil, err
	}

	if len(options.PipelineRoot) == 0 {
		cfg, err := config.FromConfigMap(ctx, k8sClient, options.Namespace)
		if err != nil {
			return nil, err
		}
		options.PipelineRoot = cfg.DefaultPipelineRoot()
		glog.Infof("PipelineRoot defaults to %q.", options.PipelineRoot)
	}

	rt, err := parseRuntimeInfo(runtimeInfo)
	if err != nil {
		return nil, err
	}
	cmdArgs, err := parseArgs(flag.Args(), rt)
	if err != nil {
		return nil, err
	}

	metadataClient, err := metadata.NewClient(options.MLMDServerAddress, options.MLMDServerPort)
	if err != nil {
		return nil, err
	}

	cacheClient, err := cacheutils.NewClient()
	if err != nil {
		return nil, err
	}

	return &Launcher{
		options:        options,
		runtimeInfo:    rt,
		metadataClient: metadataClient,
		cacheClient:    cacheClient,
		k8sClient:      k8sClient,
		cmdArgs:        cmdArgs,
	}, nil
}

// RunComponent runs the current KFP component using the specified command and
// arguments.
func (l *Launcher) RunComponent(ctx context.Context) error {
	pipeline, err := l.metadataClient.GetPipeline(ctx, l.options.PipelineName, l.options.RunID, l.options.Namespace, l.options.RunResource, l.options.PipelineRoot)
	if err != nil {
		return fmt.Errorf("unable to get pipeline with PipelineName %q PipelineRunID %q: %w", l.options.PipelineName, l.options.RunID, err)
	}
	// l.generateOutputUri needs l.bucketConfig, so l.bucketConfig must be inited first
	l.bucketConfig, err = objectstore.ParseBucketConfig(pipeline.GetPipelineRoot())
	if err != nil {
		return err
	}
	executorInput, err := l.runtimeInfo.generateExecutorInput(l.generateOutputURI, OutputMetadataFilepath)
	if err != nil {
		return fmt.Errorf("failure while generating ExecutorInput: %w", err)
	}
	if l.options.EnableCaching {
		glog.Infof("enable caching")
		return l.executeWithCacheEnabled(ctx, executorInput, pipeline)
	} else {
		return l.executeWithoutCacheEnabled(ctx, executorInput, pipeline)
	}
}

func (l *Launcher) executeWithoutCacheEnabled(ctx context.Context, executorInput *pipelinespec.ExecutorInput, pipeline *metadata.Pipeline) error {
	cmd := l.cmdArgs[0]
	args := make([]string, len(l.cmdArgs)-1)
	_ = copy(args, l.cmdArgs[1:])
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return fmt.Errorf("failed to generate execution config: %w", err)
	}
	ecfg.Image = l.options.Image
	ecfg.Namespace = l.options.Namespace
	ecfg.PodName = l.options.PodName
	ecfg.PodUID = l.options.PodUID
	ecfg.TaskName = l.options.TaskName
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	execution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return fmt.Errorf("unable to create execution: %w", err)
	}
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, l.bucketConfig)
	if err != nil {
		return err
	}
	defer bucket.Close()
	executorOutput, err := execute(ctx, executorInput, cmd, args, bucket, l.bucketConfig, l.options.Namespace, l.k8sClient)
	if err != nil {
		return err
	}
	return l.publish(ctx, executorInput, executorOutput, execution)
}

func (l *Launcher) executeWithCacheEnabled(ctx context.Context, executorInput *pipelinespec.ExecutorInput, pipeline *metadata.Pipeline) error {
	cmd := l.cmdArgs[0]
	args := make([]string, len(l.cmdArgs)-1)
	_ = copy(args, l.cmdArgs[1:])
	outputParametersTypeMap := make(map[string]string)
	for outputParamName, outputParam := range l.runtimeInfo.OutputParameters {
		outputParametersTypeMap[outputParamName] = outputParam.Type
	}
	cacheKey, err := cacheutils.GenerateCacheKey(executorInput.GetInputs(), executorInput.GetOutputs(), outputParametersTypeMap, l.cmdArgs, l.options.Image)
	if err != nil {
		return fmt.Errorf("failure while generating CacheKey: %w", err)
	}
	fingerPrint, err := cacheutils.GenerateFingerPrint(cacheKey)
	cachedMLMDExecutionID, err := l.cacheClient.GetExecutionCache(fingerPrint, l.options.PipelineName, l.options.Namespace)
	if err != nil {
		return fmt.Errorf("failure while getting executionCache: %w", err)
	}

	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return fmt.Errorf("failed to generate execution config: %w", err)
	}
	ecfg.Image = l.options.Image
	ecfg.Namespace = l.options.Namespace
	ecfg.PodName = l.options.PodName
	ecfg.PodUID = l.options.PodUID
	ecfg.TaskName = l.options.TaskName
	ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	// TODO(capri-xiyue): what should cached execution's metadata look like?
	execution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return fmt.Errorf("unable to create execution: %w", err)
	}
	if cachedMLMDExecutionID == "" {
		return l.executeWithoutCacheHit(ctx, executorInput, execution, cmd, fingerPrint, args)
	} else {
		return l.executeWithCacheHit(ctx, executorInput, execution, cachedMLMDExecutionID)
	}
}

func (l *Launcher) executeWithCacheHit(ctx context.Context, executorInput *pipelinespec.ExecutorInput, createdExecution *metadata.Execution, cachedMLMDExecutionID string) error {
	if err := prepareOutputFolders(executorInput); err != nil {
		return err
	}
	cachedMLMDExecutionIDInt64, err := strconv.ParseInt(cachedMLMDExecutionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failure while transfering cachedMLMDExecutionID %s from string to int64: %w", cachedMLMDExecutionID, err)
	}
	execution, err := l.metadataClient.GetExecution(ctx, cachedMLMDExecutionIDInt64)
	if err != nil {
		return fmt.Errorf("failure while getting execution of cachedMLMDExecutionID %v: %w", cachedMLMDExecutionIDInt64, err)
	}

	outputParameters, err := l.storeOutputParameterValueFromCache(execution)
	if err != nil {
		return fmt.Errorf("failed to store output parameter value from cache: %w", err)
	}
	outputArtifacts, err := l.storeOutputArtifactMetadataFromCache(ctx, executorInput.GetOutputs(), cachedMLMDExecutionIDInt64)
	if err != nil {
		return fmt.Errorf("failed to store output artifact metadata from cache: %w", err)
	}

	if err := l.metadataClient.PublishExecution(ctx, createdExecution, outputParameters, outputArtifacts, pb.Execution_CACHED); err != nil {
		return fmt.Errorf("unable to publish execution: %w", err)
	}
	glog.Infof("Cached")
	return nil
}

func (l *Launcher) storeOutputParameterValueFromCache(execution *metadata.Execution) (map[string]*structpb.Value, error) {
	_, outputs, err := execution.GetParameters()
	if err != nil {
		return nil, err
	}
	for name, param := range l.runtimeInfo.OutputParameters {
		filename := param.Path
		value, ok := outputs[name]
		if !ok {
			return nil, fmt.Errorf("can't find parameter %v in mlmdOutputParameters", name)
		}
		text, err := metadata.PbValueToText(value)
		if err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(filename, []byte(text), 0644); err != nil {
			return nil, fmt.Errorf("failed to write output parameter %q to file %q: %w", name, filename, err)
		}
	}
	return outputs, nil
}

func (l *Launcher) storeOutputArtifactMetadataFromCache(ctx context.Context, executorInputOutputs *pipelinespec.ExecutorInput_Outputs, cachedMLMDExecutionID int64) ([]*metadata.OutputArtifact, error) {
	outputArtifacts, err := l.metadataClient.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MLMDOutputArtifactsByName by executionId %v: %w", cachedMLMDExecutionID, err)
	}

	// Register artifacts with MLMD.
	registeredMLMDArtifacts := make([]*metadata.OutputArtifact, 0, len(l.runtimeInfo.OutputArtifacts))
	for name, artifact := range l.runtimeInfo.OutputArtifacts {
		if !filepath.IsAbs(artifact.MetadataPath) {
			return nil, fmt.Errorf("unexpected output artifact metadata file %q: must be absolute local path", artifact.MetadataPath)
		}
		runTimeArtifactList, ok := executorInputOutputs.Artifacts[name]
		if !ok {
			return nil, fmt.Errorf("unable to find output artifact  %v in ExecutorInput.Outputs", name)
		}
		if len(runTimeArtifactList.Artifacts) == 0 {
			continue
		}
		runtimeArtifact := runTimeArtifactList.Artifacts[0]
		outputArtifact, ok := outputArtifacts[runtimeArtifact.GetName()]
		if !ok {
			return nil, fmt.Errorf("unable to find artifact with name %v in mlmd output artifacts", runtimeArtifact.GetName())
		}
		outputArtifact.Schema = runtimeArtifact.GetType().GetInstanceSchema()
		if err := os.MkdirAll(path.Dir(artifact.MetadataPath), 0644); err != nil {
			return nil, fmt.Errorf("unable to make local directory %v for outputArtifact %v: %w", artifact.MetadataPath, name, err)
		}

		b, err := outputArtifact.Marshal()
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(artifact.MetadataPath, b, 0644); err != nil {
			return nil, err
		}
		registeredMLMDArtifacts = append(registeredMLMDArtifacts, outputArtifact)
	}
	return registeredMLMDArtifacts, nil
}

func (l *Launcher) executeWithoutCacheHit(ctx context.Context, executorInput *pipelinespec.ExecutorInput, createdExecution *metadata.Execution, cmd, fingerPrint string, args []string) error {
	executedStartedTime := time.Now().Unix()
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, l.bucketConfig)
	if err != nil {
		return err
	}
	defer bucket.Close()
	executorOutput, err := execute(ctx, executorInput, cmd, args, bucket, l.bucketConfig, l.options.Namespace, l.k8sClient)
	if err != nil {
		return err
	}
	err = l.publish(ctx, executorInput, executorOutput, createdExecution)
	if err != nil {
		return err
	}
	id := createdExecution.GetID()
	if id == 0 {
		return fmt.Errorf("failed to get id from createdExecution")
	}
	task := &api.Task{
		PipelineName:    l.options.PipelineName,
		Namespace:       l.options.Namespace,
		RunId:           l.options.RunID,
		MlmdExecutionID: strconv.FormatInt(id, 10),
		CreatedAt:       &timestamp.Timestamp{Seconds: executedStartedTime},
		FinishedAt:      &timestamp.Timestamp{Seconds: time.Now().Unix()},
		Fingerprint:     fingerPrint,
	}
	err = l.cacheClient.CreateExecutionCache(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}
	return nil
}

func execute(ctx context.Context, executorInput *pipelinespec.ExecutorInput, cmd string, args []string, bucket *blob.Bucket, bucketConfig *objectstore.Config, namespace string, k8sClient *kubernetes.Clientset) (*pipelinespec.ExecutorOutput, error) {
	if err := downloadArtifacts(ctx, executorInput, bucket, bucketConfig, namespace, k8sClient); err != nil {
		return nil, err
	}
	if err := prepareOutputFolders(executorInput); err != nil {
		return nil, err
	}

	// Fill in placeholders with runtime values.
	placeholders, err := getPlaceholders(executorInput)
	if err != nil {
		return nil, err
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

func (l *Launcher) publish(ctx context.Context, executorInput *pipelinespec.ExecutorInput, executorOutput *pipelinespec.ExecutorOutput, execution *metadata.Execution) error {
	// Dump output parameters in user emitted ExecutorOutput file to local path,
	// so that they can be collected as argo output parameters.
	if err := l.dumpOutputParameters(executorOutput); err != nil {
		return err
	}
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, l.bucketConfig)
	if err != nil {
		return err
	}
	defer bucket.Close()
	outputArtifacts, err := uploadOutputArtifacts(ctx, executorInput, executorOutput, uploadOutputArtifactsOptions{
		bucketConfig:   l.bucketConfig,
		bucket:         bucket,
		metadataClient: l.metadataClient,
	})
	if err != nil {
		return err
	}
	err = l.dumpOutputArtifactsMetadata(outputArtifacts)
	if err != nil {
		return err
	}
	outputParameters, err := l.readOutputParameters()
	if err != nil {
		return err
	}
	if err := l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts, pb.Execution_COMPLETE); err != nil {
		return fmt.Errorf("unable to publish createdExecution: %w", err)
	}
	return nil
}

func (l *Launcher) dumpOutputParameters(executorOutput *pipelinespec.ExecutorOutput) error {
	for name, parameter := range executorOutput.ParameterValues {
		wrap := func(err error) error {
			return fmt.Errorf("failed to dump output parameter %q in executor output to disk: %w", name, err)
		}
		value, err := metadata.PbValueToText(parameter)
		if err != nil {
			return wrap(err)
		}
		outputParam, ok := l.runtimeInfo.OutputParameters[name]
		if !ok {
			return wrap(fmt.Errorf("parameter is not defined in component"))
		}
		filename := outputParam.Path
		if err := ioutil.WriteFile(filename, []byte(value), 0644); err != nil {
			return wrap(fmt.Errorf("failed writing to file %q: %w", filename, err))
		}
	}
	return nil
}

type uploadOutputArtifactsOptions struct {
	bucketConfig   *objectstore.Config
	bucket         *blob.Bucket
	metadataClient *metadata.Client
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

func (l *Launcher) dumpOutputArtifactsMetadata(outputArtifacts []*metadata.OutputArtifact) error {
	for _, oa := range outputArtifacts {
		dumpError := func(err error) error {
			return fmt.Errorf("failed to dump output artifact metadata for name=%q: %w", oa.Name, err)
		}
		rtoa, ok := l.runtimeInfo.OutputArtifacts[oa.Name]
		if !filepath.IsAbs(rtoa.MetadataPath) {
			return dumpError(fmt.Errorf("unexpected output artifact metadata file %q: must be absolute local path", rtoa.MetadataPath))
		}
		if !ok {
			return dumpError(errors.New("unable to find output artifact in RuntimeInfo"))
		}
		if err := os.MkdirAll(path.Dir(rtoa.MetadataPath), 0644); err != nil {
			return dumpError(err)
		}
		b, err := oa.Marshal()
		if err != nil {
			return dumpError(err)
		}
		if err := ioutil.WriteFile(rtoa.MetadataPath, b, 0644); err != nil {
			return dumpError(err)
		}
	}
	return nil
}

func (l *Launcher) readOutputParameters() (map[string]*structpb.Value, error) {
	outputParameters := make(map[string]*structpb.Value)

	for n, op := range l.runtimeInfo.OutputParameters {
		wrap := func(err error) error {
			return fmt.Errorf("Failed to read output parameter name=%q type=%q path=%q: %w", n, op.Type, op.Path, err)
		}

		b, err := ioutil.ReadFile(op.Path)
		if err != nil {
			return nil, wrap(err)
		}
		switch op.Type {
		case "STRING":
			outputParameters[n] = structpb.NewStringValue(string(b))
		case "NUMBER_INTEGER", "NUMBER_DOUBLE":
			f, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 0)
			if err != nil {
				return nil, wrap(fmt.Errorf("failed to parse number parameter: %w", err))
			}
			outputParameters[n] = structpb.NewNumberValue(f)
		case "BOOLEAN":
			b, err := strconv.ParseBool(strings.TrimSpace(string(b)))
			if err != nil {
				return nil, wrap(fmt.Errorf("failed to parse boolean parameter: %w", err))
			}
			outputParameters[n] = structpb.NewBoolValue(b)
		case "LIST":
			value := &structpb.Value{}
			if err := value.UnmarshalJSON(b); err != nil {
				return nil, wrap(fmt.Errorf("failed to parse list parameter: %w", err))

			}
			outputParameters[n] = value
		case "STRUCT":
			value := &structpb.Value{}
			if err := value.UnmarshalJSON(b); err != nil {
				return nil, wrap(fmt.Errorf("failed to parse dict parameter: %w", err))

			}
			outputParameters[n] = value
		default:
			return nil, wrap(fmt.Errorf("unknown ParameterType %q", op.Type))
		}
	}
	return outputParameters, nil
}

func (l *Launcher) generateOutputURI(name string) string {
	blobKey := path.Join(l.options.TaskName, name)
	return l.bucketConfig.UriFromKey(blobKey)
}

func downloadArtifacts(ctx context.Context, executorInput *pipelinespec.ExecutorInput, defaultBucket *blob.Bucket, defaultBucketConfig *objectstore.Config, namespace string, k8sClient *kubernetes.Clientset) error {
	// Read input artifact metadata.
	nonDefaultBuckets, err := fetchNonDefaultBuckets(ctx, executorInput.Inputs.Artifacts, defaultBucketConfig, namespace, k8sClient)
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
	for name, artifactList := range executorInput.Inputs.Artifacts {
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

func fetchNonDefaultBuckets(ctx context.Context, artifacts map[string]*pipelinespec.ArtifactList, defaultBucketConfig *objectstore.Config, namespace string, k8sClient *kubernetes.Clientset) (buckets map[string]*blob.Bucket, err error) {
	nonDefaultBuckets := make(map[string]*blob.Bucket)
	for name, artifactList := range artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		artifact := artifactList.Artifacts[0]
		if !strings.HasPrefix(artifact.Uri, defaultBucketConfig.PrefixedBucket()) {
			nonDefaultBucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(artifact.Uri)
			if err != nil {
				return nonDefaultBuckets, fmt.Errorf("failed to parse bucketConfig for output artifact %q with uri %q: %w", name, artifact.GetUri(), err)
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
	for name, artifactList := range executorInput.Inputs.Artifacts {
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
	for name, parameter := range executorInput.Inputs.ParameterValues {
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
		if err := os.MkdirAll(dir, 0644); err != nil {
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

		if err := os.MkdirAll(filepath.Dir(localPath), 0644); err != nil {
			return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
		}
	}

	return nil
}
