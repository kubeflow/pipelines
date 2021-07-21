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
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/cacheutils"
	api "github.com/kubeflow/pipelines/v2/kfp-api"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Launcher is used to launch KFP components. It handles the recording of the
// appropriate metadata for lineage.
type Launcher struct {
	options                 *LauncherOptions
	runtimeInfo             *runtimeInfo
	placeholderReplacements map[string]string
	metadataClient          *metadata.Client
	cacheClient             *cacheutils.Client
	bucketConfig            *bucketConfig
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

type bucketConfig struct {
	scheme      string
	bucketName  string
	prefix      string
	queryString string
}

func (b *bucketConfig) bucketURL() string {
	u := b.scheme + b.bucketName

	// append prefix=b.prefix to existing queryString
	q := b.queryString
	if len(b.prefix) > 0 {
		if len(q) > 0 {
			q = q + "&prefix=" + b.prefix
		} else {
			q = "?prefix=" + b.prefix
		}
	}

	u = u + q
	return u
}

func (b *bucketConfig) keyFromURI(uri string) (string, error) {
	prefixedBucket := b.scheme + path.Join(b.bucketName, b.prefix)
	if !strings.HasPrefix(uri, prefixedBucket) {
		return "", fmt.Errorf("URI %q does not have expected bucket prefix %q", uri, prefixedBucket)
	}

	key := strings.TrimLeft(strings.TrimPrefix(uri, prefixedBucket), "/")
	if len(key) == 0 {
		return "", fmt.Errorf("URI %q has empty key given prefixed bucket %q", uri, prefixedBucket)
	}
	return key, nil
}

func (b *bucketConfig) uriFromKey(blobKey string) string {
	return b.scheme + path.Join(b.bucketName, b.prefix, blobKey)
}

var bucketPattern = regexp.MustCompile(`(^[a-z][a-z0-9]+:///?)([^/?]+)(/[^?]*)?(\?.+)?$`)

func parseBucketConfig(path string) (*bucketConfig, error) {
	ms := bucketPattern.FindStringSubmatch(path)
	if ms == nil || len(ms) != 5 {
		return nil, fmt.Errorf("Unrecognized pipeline root format: %q", path)
	}

	// TODO: Verify/add support for file:///.
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" {
		return nil, fmt.Errorf("Unsupported Cloud bucket: %q", path)
	}

	prefix := strings.TrimPrefix(ms[3], "/")
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return &bucketConfig{
		scheme:      ms[1],
		bucketName:  ms[2],
		prefix:      prefix,
		queryString: ms[4],
	}, nil
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
	return nil
}

const outputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"
const defaultPipelineRoot = "minio://mlpipeline/v2/artifacts"
const launcherConfigName = "kfp-launcher"
const configKeyDefaultPipelineRoot = "defaultPipelineRoot"

// NewLauncher creates a new launcher object using the JSON-encoded runtimeInfo
// and specified options.
func NewLauncher(runtimeInfo string, options *LauncherOptions) (*Launcher, error) {
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
		options.PipelineRoot = defaultPipelineRoot
		config, err := getLauncherConfig(k8sClient, options.Namespace)
		if err != nil {
			return nil, err
		}
		// Launcher config is optional, so it can be nil when err == nil.
		if config != nil {
			// The key defaultPipelineRoot is also optional in launcher config.
			defaultRootFromConfig := config.Data[configKeyDefaultPipelineRoot]
			if defaultRootFromConfig != "" {
				options.PipelineRoot = defaultRootFromConfig
			}
		}
		glog.Infof("PipelineRoot defaults to %q.", options.PipelineRoot)
	}

	bc, err := parseBucketConfig(options.PipelineRoot)
	if err != nil {
		return nil, err
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
		bucketConfig:   bc,
		k8sClient:      k8sClient,
		cmdArgs:        cmdArgs,
	}, nil
}

// RunComponent runs the current KFP component using the specified command and
// arguments.
func (l *Launcher) RunComponent(ctx context.Context) error {
	executorInput, err := l.runtimeInfo.generateExecutorInput(l.generateOutputURI, outputMetadataFilepath)
	if err != nil {
		return fmt.Errorf("failure while generating ExecutorInput: %w", err)
	}
	if l.options.EnableCaching {
		glog.Infof("enable caching")
		return l.executeWithCacheEnabled(ctx, executorInput)
	} else {
		return l.executeWithoutCacheEnabled(ctx, executorInput)
	}
}

func (l *Launcher) executeWithoutCacheEnabled(ctx context.Context, executorInput *pipelinespec.ExecutorInput) error {
	cmd := l.cmdArgs[0]
	args := make([]string, len(l.cmdArgs)-1)
	_ = copy(args, l.cmdArgs[1:])
	pipeline, err := l.metadataClient.GetPipeline(ctx, l.options.PipelineName, l.options.RunID, l.options.Namespace, l.options.RunResource)
	if err != nil {
		return fmt.Errorf("unable to get pipeline with PipelineName %q PipelineRunID %q: %w", l.options.PipelineName, l.options.RunID, err)
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
	execution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return fmt.Errorf("unable to create execution: %w", err)
	}
	executorOutput, err := l.execute(ctx, executorInput, cmd, args)
	if err != nil {
		return err
	}
	return l.publish(ctx, executorInput, executorOutput, execution)
}

func (l *Launcher) executeWithCacheEnabled(ctx context.Context, executorInput *pipelinespec.ExecutorInput) error {
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
	cachedMLMDExecutionID, err := l.cacheClient.GetExecutionCache(fingerPrint, l.options.PipelineName)
	if err != nil {
		return fmt.Errorf("failure while getting executionCache: %w", err)
	}

	pipeline, err := l.metadataClient.GetPipeline(ctx, l.options.PipelineName, l.options.RunID, l.options.Namespace, l.options.RunResource)
	if err != nil {
		return fmt.Errorf("unable to get pipeline with PipelineName %q PipelineRunID %q: %w", l.options.PipelineName, l.options.RunID, err)
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
	executions, err := l.metadataClient.GetExecutions(ctx, []int64{cachedMLMDExecutionIDInt64})
	if err != nil {
		return fmt.Errorf("failure while getting execution of cachedMLMDExecutionID %v: %w", cachedMLMDExecutionIDInt64, err)
	}
	if len(executions) == 0 {
		return fmt.Errorf("the execution with id %s does not exist in MLMD", cachedMLMDExecutionID)
	}
	if len(executions) > 1 {
		return fmt.Errorf("got multiple executions with id %s in MLMD", cachedMLMDExecutionID)
	}
	cachedExecution := executions[0]

	outputParameters, err := l.storeOutputParameterValueFromCache(cachedExecution)
	if err != nil {
		return fmt.Errorf("failed to store output parameter value from cache: %w", err)
	}
	outputArtifacts, err := l.storeOutputArtifactMetadataFromCache(ctx, executorInput.Outputs, cachedMLMDExecutionIDInt64)
	if err != nil {
		return fmt.Errorf("failed to store output artifact metadata from cache: %w", err)
	}

	if err := l.metadataClient.PublishExecution(ctx, createdExecution, outputParameters, outputArtifacts, pb.Execution_CACHED); err != nil {
		return fmt.Errorf("unable to publish execution: %w", err)
	}
	glog.Infof("Cached")
	return nil
}

func (l *Launcher) storeOutputParameterValueFromCache(cachedExecution *pb.Execution) (*metadata.Parameters, error) {
	mlmdOutputParameters, err := cacheutils.GetMLMDOutputParams(cachedExecution)
	if err != nil {
		return nil, err
	}
	// Read output parameters.
	outputParameters := &metadata.Parameters{
		IntParameters:    make(map[string]int64),
		StringParameters: make(map[string]string),
		DoubleParameters: make(map[string]float64),
	}

	for name, param := range l.runtimeInfo.OutputParameters {
		filename := param.Path
		outputParamValue, ok := mlmdOutputParameters[name]
		if !ok {
			return nil, fmt.Errorf("can't find parameter %v in mlmdOutputParameters", name)
		}
		if err := ioutil.WriteFile(filename, []byte(outputParamValue), 0644); err != nil {
			return nil, fmt.Errorf("failed to write output parameter %q to file %q: %w", name, filename, err)
		}
		switch param.Type {
		case "STRING":
			outputParameters.StringParameters[name] = outputParamValue
		case "INT":
			i, err := strconv.ParseInt(strings.TrimSpace(outputParamValue), 10, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to parse parameter name=%q value =%v to int: %w", name, outputParamValue, err)
			}
			outputParameters.IntParameters[name] = i
		case "DOUBLE":
			f, err := strconv.ParseFloat(strings.TrimSpace(outputParamValue), 0)
			return nil, fmt.Errorf("failed to parse parameter name=%q value =%v to double: %w", name, outputParamValue, err)
			outputParameters.DoubleParameters[name] = f
		default:
			return nil, fmt.Errorf("unknown type. Expected STRING, INT or DOUBLE")
		}
	}
	return outputParameters, nil
}

func (l *Launcher) storeOutputArtifactMetadataFromCache(ctx context.Context, executorInputOutputs *pipelinespec.ExecutorInput_Outputs, cachedMLMDExecutionID int64) ([]*metadata.OutputArtifact, error) {
	mlmdOutputArtifactsByName, err := l.metadataClient.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
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
		mlmdArtifacts, ok := mlmdOutputArtifactsByName[runtimeArtifact.GetName()]
		if !ok || len(mlmdArtifacts) == 0 {
			return nil, fmt.Errorf("unable to find artifact with name %v in mlmd output artifacts", runtimeArtifact.GetName())
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		mlmdArtifact := mlmdArtifacts[0]
		if err := os.MkdirAll(path.Dir(artifact.MetadataPath), 0644); err != nil {
			return nil, fmt.Errorf("unable to make local directory %v for outputArtifact %v: %w", artifact.MetadataPath, name, err)
		}

		b, err := protojson.Marshal(mlmdArtifact)
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(artifact.MetadataPath, b, 0644); err != nil {
			return nil, err
		}
		registeredMLMDArtifacts = append(registeredMLMDArtifacts, &metadata.OutputArtifact{
			Name:     name,
			Artifact: mlmdArtifact,
			Schema:   runtimeArtifact.Type.GetInstanceSchema(),
		})
	}
	return registeredMLMDArtifacts, nil
}

func (l *Launcher) executeWithoutCacheHit(ctx context.Context, executorInput *pipelinespec.ExecutorInput, createdExecution *metadata.Execution, cmd, fingerPrint string, args []string) error {
	executedStartedTime := time.Now().Unix()
	executorOutput, err := l.execute(ctx, executorInput, cmd, args)
	if err != nil {
		return err
	}
	err = l.publish(ctx, executorInput, executorOutput, createdExecution)
	if err != nil {
		return err
	}
	id := createdExecution.GetID()
	if id == nil {
		return fmt.Errorf("failed to get id from createdExecution")
	}
	task := &api.Task{
		PipelineName:    l.options.PipelineName,
		RunId:           l.options.RunID,
		MlmdExecutionID: strconv.FormatInt(*id, 10),
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

func (l *Launcher) execute(ctx context.Context, executorInput *pipelinespec.ExecutorInput, cmd string, args []string) (*pipelinespec.ExecutorOutput, error) {
	if err := l.downloadArtifacts(ctx, executorInput); err != nil {
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

	// Collect outputs
	return getExecutorOutput()
}

func (l *Launcher) publish(ctx context.Context, executorInput *pipelinespec.ExecutorInput, executorOutput *pipelinespec.ExecutorOutput, execution *metadata.Execution) error {
	// Dump output parameters in user emitted ExecutorOutput file to local path,
	// so that they can be collected as argo output parameters.
	if err := l.dumpOutputParameters(executorOutput); err != nil {
		return err
	}
	outputArtifacts, err := l.uploadOutputArtifacts(ctx, executorInput, executorOutput)
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
	for name, parameter := range executorOutput.Parameters {
		wrap := func(err error) error {
			return fmt.Errorf("failed to dump output parameter %q in executor output to disk: %w", name, err)
		}
		var value string
		switch t := parameter.Value.(type) {
		case *pipelinespec.Value_StringValue:
			value = parameter.GetStringValue()
		case *pipelinespec.Value_DoubleValue:
			value = strconv.FormatFloat(parameter.GetDoubleValue(), 'f', -1, 64)
		case *pipelinespec.Value_IntValue:
			value = strconv.FormatInt(parameter.GetIntValue(), 10)
		default:
			return wrap(fmt.Errorf("unknown PipelineSpec Value type %T", t))
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

func (l *Launcher) uploadOutputArtifacts(ctx context.Context, executorInput *pipelinespec.ExecutorInput, executorOutput *pipelinespec.ExecutorOutput) ([]*metadata.OutputArtifact, error) {
	bucket, err := l.openBucket()
	if err != nil {
		return nil, fmt.Errorf("Failed to open bucket %q: %v", l.bucketConfig.bucketName, err)
	}
	defer bucket.Close()

	// Register artifacts with MLMD.
	outputArtifacts := make([]*metadata.OutputArtifact, 0, len(l.runtimeInfo.OutputArtifacts))
	for name, artifactList := range executorInput.Outputs.Artifacts {
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
			blobKey, err := l.bucketConfig.keyFromURI(outputArtifact.Uri)
			if err != nil {
				return nil, fmt.Errorf("failed to upload output artifact %q: %w", name, err)
			}
			if err := uploadBlob(ctx, bucket, localDir, blobKey); err != nil {
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
		schema, err := getRuntimeArtifactSchema(outputArtifact)
		if err != nil {
			return nil, fmt.Errorf("failed to determine schema for output %q: %w", name, err)
		}
		mlmdArtifact, err := l.metadataClient.RecordArtifact(ctx, name, schema, outputArtifact, pb.Artifact_LIVE)
		if err != nil {
			return nil, metadataErr(err)
		}
		outputArtifacts = append(outputArtifacts, mlmdArtifact)

		rtoa, ok := l.runtimeInfo.OutputArtifacts[name]
		if !filepath.IsAbs(rtoa.MetadataPath) {
			return nil, metadataErr(fmt.Errorf("unexpected output artifact metadata file %q: must be absolute local path", rtoa.MetadataPath))
		}
		if !ok {
			return nil, metadataErr(errors.New("unable to find output artifact in RuntimeInfo"))
		}
		if err := os.MkdirAll(path.Dir(rtoa.MetadataPath), 0644); err != nil {
			return nil, metadataErr(err)
		}

		b, err := mlmdArtifact.Marshal()
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(rtoa.MetadataPath, b, 0644); err != nil {
			return nil, err
		}
	}
	return outputArtifacts, nil
}

func (l *Launcher) readOutputParameters() (*metadata.Parameters, error) {
	outputParameters := &metadata.Parameters{
		IntParameters:    make(map[string]int64),
		StringParameters: make(map[string]string),
		DoubleParameters: make(map[string]float64),
	}
	for n, op := range l.runtimeInfo.OutputParameters {
		msg := func(err error) error {
			return fmt.Errorf("Failed to read output parameter name=%q type=%q path=%q: %w", n, op.Type, op.Path, err)
		}
		b, err := ioutil.ReadFile(op.Path)
		if err != nil {
			return nil, msg(err)
		}
		switch op.Type {
		case "STRING":
			outputParameters.StringParameters[n] = string(b)
		case "INT":
			i, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 0)
			if err != nil {
				return nil, msg(err)
			}
			outputParameters.IntParameters[n] = i
		case "DOUBLE":
			f, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 0)
			if err != nil {
				return nil, msg(err)
			}
			outputParameters.DoubleParameters[n] = f
		default:
			return nil, msg(fmt.Errorf("unknown type. Expected STRING, INT or DOUBLE"))
		}
	}
	return outputParameters, nil
}

func (l *Launcher) generateOutputURI(name string) string {
	blobKey := path.Join(l.options.PipelineName, l.options.RunID, l.options.TaskName, name)
	return l.bucketConfig.uriFromKey(blobKey)
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
	return "", fmt.Errorf("found URI with unsupported storage scheme: %s", uri)
}

func (l *Launcher) downloadArtifacts(ctx context.Context, executorInput *pipelinespec.ExecutorInput) error {
	bucket, err := l.openBucket()
	if err != nil {
		return fmt.Errorf("Failed to open bucket %q: %v", l.bucketConfig.bucketName, err)
	}
	defer bucket.Close()

	// Read input artifact metadata.
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
		blobKey, err := l.bucketConfig.keyFromURI(inputArtifact.Uri)
		if err != nil {
			return copyErr(err)
		}
		if err := downloadBlob(ctx, bucket, localPath, blobKey); err != nil {
			return copyErr(err)
		}
	}
	return nil
}

// Add executor input placeholders to provided map.
func getPlaceholders(executorInput *pipelinespec.ExecutorInput) (map[string]string, error) {
	placeholders := make(map[string]string)
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
	for name, artifactList := range executorInput.Outputs.Artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].uri}}`, name)] = outputArtifact.Uri

		localPath, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			return nil, fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
		}
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].path}}`, name)] = localPath
	}

	// Prepare input parameter placeholders.
	for name, parameter := range executorInput.Inputs.Parameters {
		key := fmt.Sprintf(`{{$.inputs.parameters['%s']}}`, name)
		switch t := parameter.Value.(type) {
		case *pipelinespec.Value_StringValue:
			placeholders[key] = parameter.GetStringValue()
		case *pipelinespec.Value_DoubleValue:
			placeholders[key] = strconv.FormatFloat(parameter.GetDoubleValue(), 'f', -1, 64)
		case *pipelinespec.Value_IntValue:
			placeholders[key] = strconv.FormatInt(parameter.GetIntValue(), 10)
		default:
			return nil, fmt.Errorf("unknown PipelineSpec Value type %T", t)
		}
	}

	// Prepare output parameter placeholders.
	for name, parameter := range executorInput.Outputs.Parameters {
		key := fmt.Sprintf(`{{$.outputs.parameters['%s'].output_file}}`, name)
		placeholders[key] = parameter.OutputFile
	}

	return placeholders, nil
}

func prepareOutputFolders(executorInput *pipelinespec.ExecutorInput) error {
	for name, parameter := range executorInput.Outputs.Parameters {
		dir := filepath.Dir(parameter.OutputFile)
		if err := os.MkdirAll(dir, 0644); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.Outputs.Artifacts {
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

func getRuntimeArtifactSchema(rta *pipelinespec.RuntimeArtifact) (string, error) {
	switch t := rta.Type.Kind.(type) {
	case *pipelinespec.ArtifactTypeSchema_InstanceSchema:
		return t.InstanceSchema, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaTitle:
		return "title: " + t.SchemaTitle, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaUri:
		return "", fmt.Errorf("SchemaUri is unsupported, found in RuntimeArtifact %+v", rta)
	default:
		return "", fmt.Errorf("unknown type %T in RuntimeArtifact %+v", t, rta)
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

// TODO(neuromage): Move these helper functions to a storage package and add tests.
func uploadFile(ctx context.Context, bucket *blob.Bucket, localFilePath, blobFilePath string) error {
	errorF := func(err error) error {
		return fmt.Errorf("uploadFile(): unable to complete copying %q to remote storage %q: %w", localFilePath, blobFilePath, err)
	}

	w, err := bucket.NewWriter(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open writer for bucket: %w", err))
	}

	r, err := os.Open(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for reading: %w", localFilePath, err))
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	if err = w.Close(); err != nil {
		return errorF(fmt.Errorf("failed to close Writer for bucket: %w", err))
	}

	glog.Infof("uploadFile(localFilePath=%q, blobFilePath=%q)", localFilePath, blobFilePath)
	return nil
}

func downloadFile(ctx context.Context, bucket *blob.Bucket, blobFilePath, localFilePath string) (err error) {
	errorF := func(err error) error {
		return fmt.Errorf("downloadFile(): unable to complete copying %q to local storage %q: %w", blobFilePath, localFilePath, err)
	}

	r, err := bucket.NewReader(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open reader for bucket: %w", err))
	}
	defer r.Close()

	localDir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(localDir, 0644); err != nil {
		return errorF(fmt.Errorf("failed to create local directory %q: %w", localDir, err))
	}

	w, err := os.Create(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for writing: %w", localFilePath, err))
	}
	defer func() {
		errClose := w.Close()
		if err == nil && errClose != nil {
			// override named return value "err" when there's a close error
			err = errorF(errClose)
		}
	}()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	return nil
}

func uploadBlob(ctx context.Context, bucket *blob.Bucket, localPath, blobPath string) error {

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("unable to stat local filepath %q: %w", localPath, err)
	}

	if !fileInfo.IsDir() {
		return uploadFile(ctx, bucket, localPath, blobPath)
	}

	// localPath is a directory.
	files, err := ioutil.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("unable to list local directory %q: %w", localPath, err)
	}

	for _, f := range files {
		if f.IsDir() {
			err = uploadBlob(ctx, bucket, filepath.Join(localPath, f.Name()), blobPath+"/"+f.Name())
			if err != nil {
				return err
			}
		} else {
			blobFilePath := filepath.Join(blobPath, filepath.Base(f.Name()))
			localFilePath := filepath.Join(localPath, f.Name())
			if err := uploadFile(ctx, bucket, localFilePath, blobFilePath); err != nil {
				return err
			}
		}

	}

	return nil
}

func downloadBlob(ctx context.Context, bucket *blob.Bucket, localDir, blobDir string) error {
	iter := bucket.List(&blob.ListOptions{Prefix: blobDir})
	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to list objects in remote storage %q: %w", blobDir, err)
		}
		if obj.IsDir {
			// TODO: is this branch possible?

			// Object stores list all files with the same prefix,
			// there is no need to recursively list each folder.
			continue
		} else {
			relativePath, err := filepath.Rel(blobDir, obj.Key)
			if err != nil {
				return fmt.Errorf("unexpected object key %q when listing %q: %w", obj.Key, blobDir, err)
			}
			if err := downloadFile(ctx, bucket, obj.Key, filepath.Join(localDir, relativePath)); err != nil {
				return err
			}
		}
	}
	return nil
}

func getExecutorOutput() (*pipelinespec.ExecutorOutput, error) {
	executorOutput := &pipelinespec.ExecutorOutput{
		Parameters: map[string]*pipelinespec.Value{},
		Artifacts:  map[string]*pipelinespec.ArtifactList{},
	}

	_, err := os.Stat(outputMetadataFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return an empty ExecutorOutput.
			return executorOutput, nil
		} else {
			return nil, fmt.Errorf("failed to stat output metadata file %q: %w", outputMetadataFilepath, err)
		}
	}

	b, err := ioutil.ReadFile(outputMetadataFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output metadata file %q: %w", outputMetadataFilepath, err)
	}

	if err := protojson.Unmarshal(b, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshall ExecutorOutput in file %q: %w", outputMetadataFilepath, err)
	}

	return executorOutput, nil
}

func (l *Launcher) openBucket() (*blob.Bucket, error) {
	if l.bucketConfig.scheme == "minio://" {
		cred, err := objectstore.GetMinioCredential(l.k8sClient, l.options.Namespace)
		if err != nil {
			return nil, fmt.Errorf("Failed to get minio credential: %w", err)
		}
		sess, err := session.NewSession(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(cred.AccessKey, cred.SecretKey, ""),
			Region:           aws.String("minio"),
			Endpoint:         aws.String(objectstore.MinioDefaultEndpoint()),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		})

		if err != nil {
			return nil, fmt.Errorf("Failed to create session to access minio: %v", err)
		}
		minioBucket, err := s3blob.OpenBucket(context.Background(), sess, l.bucketConfig.bucketName, nil)
		if err != nil {
			return nil, err
		}
		// Directly calling s3blob.OpenBucket does not allow overriding prefix via l.bucketConfig.bucketURL().
		// Therefore, we need to explicitly configure the prefixed bucket.
		return blob.PrefixedBucket(minioBucket, l.bucketConfig.prefix), nil

	}
	return blob.OpenBucket(context.Background(), l.bucketConfig.bucketURL())
}

func getLauncherConfig(clientSet *kubernetes.Clientset, namespace string) (*v1.ConfigMap, error) {
	config, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.Background(), launcherConfigName, metav1.GetOptions{})
	if err != nil {
		if k8errors.IsNotFound(err) {
			glog.Infof("cannot find launcher configmap: name=%q namespace=%q", launcherConfigName, namespace)
			// LauncherConfig is optional, so ignore not found error.
			return nil, nil
		}
		return nil, err
	}
	return config, nil
}
