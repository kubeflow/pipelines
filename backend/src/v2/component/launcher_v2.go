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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LauncherV2Options struct {
	Namespace         string
	PodName           string
	PodUID            string
	PipelineName      string
	PublishLogs       string
	CachedFingerprint string
	CacheDisabled     bool
	IterationIndex    *int64
	ComponentSpec     *pipelinespec.ComponentSpec
	ImporterSpec      *pipelinespec.PipelineDeploymentConfig_ImporterSpec
	PipelineSpec      *structpb.Struct
	TaskSpec          *pipelinespec.PipelineTaskSpec
	ScopePath         util.ScopePath
	Run               *apiV2beta1.Run
	ParentTask        *apiV2beta1.PipelineTaskDetail
	Task              *apiV2beta1.PipelineTaskDetail
	// Set to true if apiserver is serving over TLS
	MLPipelineTLSEnabled bool
	MLPipelineServerAddress,
	MLPipelineServerPort,
	CaCertPath           string
}

type LauncherV2 struct {
	executorInput *pipelinespec.ExecutorInput
	command       string
	args          []string
	options       LauncherV2Options
	clientManager client_manager.ClientManagerInterface
	// Maintaining a cache of opened buckets will minimize
	// the number of calls to the object store, and api server
	openedBucketCache map[string]*blob.Bucket
	launcherConfig    *config.Config
	pipelineSpec      *structpb.Struct

	// BatchUpdater collects API updates and flushes them in batches
	// to reduce database round-trips
	batchUpdater *BatchUpdater

	// Dependency interfaces for testing
	fileSystem  FileSystem
	cmdExecutor CommandExecutor
	objectStore ObjectStoreClientInterface
}

// NewLauncherV2 is a factory function that returns an instance of LauncherV2.
func NewLauncherV2(
	executorInputJSON string,
	cmdArgs []string,
	opts *LauncherV2Options,
	clientManager client_manager.ClientManagerInterface,
) (l *LauncherV2, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create component launcher v2: %w", err)
		}
	}()

	executorInput := &pipelinespec.ExecutorInput{}
	err = protojson.Unmarshal([]byte(executorInputJSON), executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal executor input: %w", err)
	}
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("command and arguments are empty")
	}
	err = opts.validate()
	if err != nil {
		return nil, err
	}

	launcher := &LauncherV2{
		executorInput: executorInput,
		command:       cmdArgs[0],
		args:          cmdArgs[1:],
		options:       *opts,
		clientManager: clientManager,
		batchUpdater:  NewBatchUpdater(),
		// Initialize with production implementations
		fileSystem:        &OSFileSystem{},
		cmdExecutor:       &RealCommandExecutor{},
		openedBucketCache: make(map[string]*blob.Bucket),
		pipelineSpec:      opts.PipelineSpec,
	}

	// Object store is initialized after launcher creation
	launcher.objectStore = NewObjectStoreClient(launcher)
	return launcher, nil
}

// WithFileSystem allows overriding the file system (for testing)
func (l *LauncherV2) WithFileSystem(fs FileSystem) *LauncherV2 {
	l.fileSystem = fs
	return l
}

// WithCommandExecutor allows overriding the command executor (for testing)
func (l *LauncherV2) WithCommandExecutor(executor CommandExecutor) *LauncherV2 {
	l.cmdExecutor = executor
	return l
}

// WithObjectStore allows overriding the object store client (for testing)
func (l *LauncherV2) WithObjectStore(store ObjectStoreClientInterface) *LauncherV2 {
	l.objectStore = store
	return l
}

// stopWaitingArtifacts will create empty files to tell Modelcar sidecar containers to stop. Any errors encountered are
// logged since this is meant as a deferred function at the end of the launcher's execution.
func stopWaitingArtifacts(artifacts map[string]*pipelinespec.ArtifactList) {
	for _, artifactList := range artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}

		// Following the convention of downloadArtifacts in the launcher to only look at the first in the list.
		for _, artifact := range artifactList.Artifacts {
			inputArtifact := artifact

			// This should ideally verify that this is also a model input artifact, but this metadata doesn't seem to
			// be set on inputArtifact.
			if !strings.HasPrefix(inputArtifact.Uri, "oci://") {
				continue
			}

			localPath, err := LocalPathForURI(inputArtifact.Uri)
			if err != nil {
				continue
			}

			glog.Infof("Stopping Modelcar container for artifact %s", inputArtifact.Uri)

			launcherCompleteFile := strings.TrimSuffix(localPath, "/models") + "/launcher-complete"
			_, err = os.Create(launcherCompleteFile)
			if err != nil {
				glog.Errorf(
					"Failed to stop the artifact %s by creating %s: %v", inputArtifact.Uri, launcherCompleteFile, err,
				)

				continue
			}
		}
	}
}

// Execute calls executeV2, updates the cache, and creates artifacts for outputs.
func (l *LauncherV2) Execute(ctx context.Context) (executionErr error) {
	defer func() {
		if executionErr != nil {
			executionErr = fmt.Errorf("failed to execute component: %w", executionErr)
		}
	}()

	l.options.Task.Pods = append(l.options.Task.Pods, &apiV2beta1.PipelineTaskDetail_TaskPod{
		Name: l.options.PodName,
		Uid:  l.options.PodUID,
		Type: apiV2beta1.PipelineTaskDetail_EXECUTOR,
	})

	// Defer the final task status update to ensure we handle and propagate errors.
	defer func() {
		if executionErr != nil {
			l.options.Task.State = apiV2beta1.PipelineTaskDetail_FAILED
			l.options.Task.StatusMetadata = &apiV2beta1.PipelineTaskDetail_StatusMetadata{
				Message: executionErr.Error(),
			}
		}
		l.options.Task.EndTime = timestamppb.New(time.Now())
		// Queue the final task status update
		l.batchUpdater.QueueTaskUpdate(l.options.Task)

		// Flush all batched updates (artifacts, artifact-tasks, task updates)
		// This executes all queued operations that were accumulated during:
		// - uploadOutputArtifacts (artifact creation)
		// - executeV2 (task output parameter update)
		// - propagateOutputsUpDAG (artifact-task creation, parent task parameter updates)
		// - this final SUCCEEDED status update
		if flushErr := l.batchUpdater.Flush(ctx, l.clientManager.KFPAPIClient()); flushErr != nil {
			l.options.Task.State = apiV2beta1.PipelineTaskDetail_FAILED
			glog.Errorf("failed to flush batch updates: %v", flushErr)
			_, updateTaskErr := l.clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{TaskId: l.options.Task.GetTaskId(), Task: l.options.Task})
			if updateTaskErr != nil {
				glog.Errorf("failed to update task status: %v", updateTaskErr)
				// Return here, if we can't update this Task's status, then there's no point in proceeding.
				// This should never happen.
				return
			}
			// Do not return on flush error, we want to propagate the error to the upstream tasks.
		}
		// Refresh run before updating statuses
		fullView := apiV2beta1.GetRunRequest_FULL
		refreshedRun, getRunErr := l.clientManager.KFPAPIClient().GetRun(ctx, &apiV2beta1.GetRunRequest{RunId: l.options.Run.GetRunId(), View: &fullView})
		if getRunErr != nil {
			glog.Errorf("failed to refresh run: %w", getRunErr)
			return
		}
		l.options.Run = refreshedRun
		updateStatusErr := l.clientManager.KFPAPIClient().UpdateStatuses(ctx, l.options.Run, l.pipelineSpec, l.options.Task)
		if updateStatusErr != nil {
			glog.Errorf("failed to update statuses: %w", updateStatusErr)
			return
		}
	}()

	defer stopWaitingArtifacts(l.executorInput.GetInputs().GetArtifacts())

	// Close any open buckets in the cache
	defer func() {
		for _, bucket := range l.openedBucketCache {
			_ = bucket.Close()
		}
	}()

	// Fetch Launcher config and initialize KFP API client if not already set (testing mode)
	// Production path: fetch real config and create real client
	launcherConfig, executionErr := config.FetchLauncherConfigMap(ctx, l.clientManager.K8sClient(), l.options.Namespace)
	if executionErr != nil {
		return fmt.Errorf("failed to get launcher configmap: %w", executionErr)
	}
	l.launcherConfig = launcherConfig

	if executionErr = l.prepareOutputFolders(l.executorInput); executionErr != nil {
		return fmt.Errorf("failed to prepare output folders: %w", executionErr)
	}
	_, executionErr = l.executeV2(ctx)
	if executionErr != nil {
		return fmt.Errorf("failed to execute component: %w", executionErr)
	}
	l.options.Task.State = apiV2beta1.PipelineTaskDetail_SUCCEEDED
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
	if o.PipelineName == "" {
		return err("PipelineName")
	}
	if o.PipelineSpec == nil {
		return err("PipelineSpec")
	}
	return nil
}

// executeV2 handles placeholder substitution for inputs, calls execute to
// execute end user logic, and uploads the resulting output Artifacts.
func (l *LauncherV2) executeV2(ctx context.Context) (*pipelinespec.ExecutorOutput, error) {
	// Fill in placeholders with runtime values.
	compiledCmd, compiledArgs, err := compileCmdAndArgs(l.executorInput, l.command, l.args)
	if err != nil {
		return nil, err
	}

	executorOutput, err := l.execute(ctx, compiledCmd, compiledArgs)
	if err != nil {
		return nil, err
	}

	// These are not added in execute(), because execute() is shared between v2 compatible and v2 engine launcher.
	// In v2 compatible mode, we get output parameter info from runtimeInfo. In v2 engine, we get it from component spec.
	// Because of the difference, we cannot put parameter collection logic in one method.
	err = l.collectOutputParameters(executorOutput)
	if err != nil {
		return nil, err
	}

	// Upload artifacts from local disk to remote store.
	err = l.uploadOutputArtifacts(ctx, executorOutput)
	if err != nil {
		return nil, err
	}

	// Update task outputs for parameters before propagation
	if executorOutput != nil && len(executorOutput.GetParameterValues()) > 0 {
		params := make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(executorOutput.GetParameterValues()))
		for key, val := range executorOutput.GetParameterValues() {
			param := &apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: key,
				Type:         apiV2beta1.IOType_OUTPUT,
				Value:        val,
				Producer: &apiV2beta1.IOProducer{
					TaskName: l.options.TaskSpec.GetTaskInfo().GetName(),
				}}
			if l.options.IterationIndex != nil {
				param.Producer.Iteration = l.options.IterationIndex
				param.Type = apiV2beta1.IOType_ITERATOR_OUTPUT
			}
			params = append(params, param)
		}

		l.options.Task.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{Parameters: params}
		// Queue task update instead of executing immediately
		l.batchUpdater.QueueTaskUpdate(l.options.Task)
	}

	// Flush artifacts and task parameter updates BEFORE propagation
	// This ensures that when propagateOutputsUpDAG refreshes the task,
	// the artifacts will exist in the API and can be propagated up the DAG
	if err = l.batchUpdater.Flush(ctx, l.clientManager.KFPAPIClient()); err != nil {
		return nil, fmt.Errorf("failed to flush artifacts before propagation: %w", err)
	}

	// Propagate outputs up the DAG hierarchy for parents that declare these outputs
	err = l.propagateOutputsUpDAG(ctx)
	if err != nil {
		return nil, err
	}

	// Flush propagation updates (artifact-tasks and parent task parameter updates)
	// so that propagated outputs are visible to subsequent driver calls
	if err = l.batchUpdater.Flush(ctx, l.clientManager.KFPAPIClient()); err != nil {
		return nil, fmt.Errorf("failed to flush propagation updates: %w", err)
	}

	return executorOutput, nil
}

// collectOutputParameters collect output parameters from local disk and add them
// to executor output.
func (l *LauncherV2) collectOutputParameters(executorOutput *pipelinespec.ExecutorOutput) error {
	if executorOutput.ParameterValues == nil {
		executorOutput.ParameterValues = make(map[string]*structpb.Value)
	}
	outputParameters := executorOutput.GetParameterValues()
	for name, param := range l.executorInput.GetOutputs().GetParameters() {
		_, ok := outputParameters[name]
		if ok {
			// If the output parameter was already specified in output metadata file,
			// we don't need to collect it from file, because output metadata file has
			// the highest priority.
			continue
		}
		paramSpec, ok := l.options.ComponentSpec.GetOutputDefinitions().GetParameters()[name]
		if !ok {
			return fmt.Errorf("failed to find output parameter name=%q in component spec", name)
		}
		msg := func(err error) error {
			return fmt.Errorf("failed to read output parameter name=%q type=%q path=%q: %w", name, paramSpec.GetParameterType(), param.GetOutputFile(), err)
		}
		b, err := l.fileSystem.ReadFile(param.GetOutputFile())
		if err != nil {
			return msg(err)
		}
		value, err := textToPbValue(string(b), paramSpec.GetParameterType())
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
	return prettyJSON.String()
}

const OutputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

// We overwrite this as a DI mechanism for testing getLogWriter.
var osCreateFunc = os.Create

// getLogWriter returns an io.Writer that can either be single-channel to stdout
// or dual-channel to stdout AND a log file based on the URI of a log artifact
// in the supplied ArtifactList. Downstream, the resulting log file gets
// uploaded to the object store.
func getLogWriter(artifacts map[string]*pipelinespec.ArtifactList) (writer io.Writer) {
	logsArtifactList, ok := artifacts["executor-logs"]

	if !ok || len(logsArtifactList.Artifacts) != 1 {
		return os.Stdout
	}

	logURI := logsArtifactList.Artifacts[0].Uri
	logFilePath, err := LocalPathForURI(logURI)
	if err != nil {
		glog.Errorf("Error converting log artifact URI, %s, to file path.", logURI)
		return os.Stdout
	}

	logFile, err := osCreateFunc(logFilePath)
	if err != nil {
		glog.Errorf("Error creating logFilePath, %s.", logFilePath)
		return os.Stdout
	}

	return io.MultiWriter(os.Stdout, logFile)
}

// ExecuteForTesting is a test-only method that executes the launcher with mocked dependencies.
// It runs the full execution flow including artifact uploads but uses the provided mock dependencies.
// This method should only be used in tests.
func (l *LauncherV2) ExecuteForTesting(ctx context.Context) (*pipelinespec.ExecutorOutput, error) {
	return l.executeV2(ctx)
}

// execute downloads input artifacts, prepares the execution environment,
// executes the end user code, and returns the outputs.
func (l *LauncherV2) execute(
	ctx context.Context,
	cmd string,
	args []string,
) (*pipelinespec.ExecutorOutput, error) {

	// Used for local debugging.
	customOutputFile := os.Getenv("KFP_OUTPUT_FILE")
	if customOutputFile != "" {
		return l.getExecutorOutputFile(customOutputFile)
	}

	if err := l.downloadArtifacts(ctx); err != nil {
		return nil, err
	}

	if err := l.prepareOutputFolders(l.executorInput); err != nil {
		return nil, err
	}

	var writer io.Writer
	if l.options.PublishLogs == "true" {
		writer = getLogWriter(l.executorInput.Outputs.GetArtifacts())
	} else {
		writer = os.Stdout
	}

	defer glog.Flush()

	// If a custom CA path is input, append to system CA and save to a temp file for executor access.
	if l.options.CaCertPath != "" {
		var caBundleTmpPath string
		var err error
		if caBundleTmpPath, err = compileTempCABundleWithCustomCA(l.options.CaCertPath); err != nil {
			return nil, err
		}

		err = os.Setenv("REQUESTS_CA_BUNDLE", caBundleTmpPath)
		if err != nil {
			glog.Errorf("Error setting REQUESTS_CA_BUNDLE environment variable, %s", err.Error())
		}
		err = os.Setenv("AWS_CA_BUNDLE", caBundleTmpPath)
		if err != nil {
			glog.Errorf("Error setting AWS_CA_BUNDLE environment variable, %s", err.Error())
		}
		err = os.Setenv("SSL_CERT_FILE", caBundleTmpPath)
		if err != nil {
			glog.Errorf("Error setting SSL_CERT_FILE environment variable, %s", err.Error())
		}
	}

	// Execute end user code using the command executor interface.
	if err := l.cmdExecutor.Run(ctx, cmd, args, os.Stdin, writer, writer); err != nil {
		return nil, err
	}

	return l.getExecutorOutputFile(l.executorInput.GetOutputs().GetOutputFile())
}

// Create a temp file that contains the system CA bundle (and custom CA if it has been mounted).
func compileTempCABundleWithCustomCA(customCAPath string) (string, error) {
	// Possible certificate files; stop after finding one. List from https://go.dev/src/crypto/x509/root_linux.go
	var systemCAs = []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/pki/tls/cacert.pem",
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
		"/etc/ssl/cert.pem",
	}
	sslCertFilePath := os.Getenv("SSL_CERT_FILE")
	// If SSL_CERT_FILE is set to a different value than the custom CA path, append it to the beginning of the systemCAs slice.
	if sslCertFilePath != "" && sslCertFilePath != customCAPath {
		systemCAs = append([]string{sslCertFilePath}, systemCAs...)
	}
	tmpCaBundle, err := os.CreateTemp("", "ca-bundle-*.crt")
	if err != nil {
		return "", err
	}
	var systemCaBundle []byte
	for _, file := range systemCAs {
		systemCaBundle, err = os.ReadFile(file)
		if err != nil {
			glog.Errorf("Error reading CA bundle file, %s.", err)
		}
		// Once a non-nil system CA file is found, break.
		if systemCaBundle != nil {
			break
		}
	}
	// Append mounted custom CA cert to System CA bundle.
	customCa, err := os.ReadFile(customCAPath)
	if err != nil {
		return "", err
	}
	systemCaBundle = append(systemCaBundle, customCa...)
	_, err = tmpCaBundle.Write(systemCaBundle)
	if err != nil {
		return "", err
	}
	// Update temp file permissions to 444 (read-only).
	err = tmpCaBundle.Chmod(0444)
	if err != nil {
		return "", err
	}
	// Return filepath for tmpCaBundle
	return tmpCaBundle.Name(), nil
}

// uploadOutputArtifacts iterates over all the Artifacts retrieved from the
// executor output and uploads them to the object store and registers them
// with the KFP API.
func (l *LauncherV2) uploadOutputArtifacts(
	ctx context.Context,
	executorOutput *pipelinespec.ExecutorOutput,
) error {
	// Manage an opened bucket cache to minimize pool
	var openedBucketCache = map[string]*blob.Bucket{}
	defer func() {
		for _, bucket := range openedBucketCache {
			_ = bucket.Close()
		}
	}()

	// After successful execution and uploads, record outputs in KFP API
	// Create artifactsMap for each output port
	artifactsMap := map[string][]*apiV2beta1.Artifact{}
	for artifactKey, artifactList := range l.executorInput.GetOutputs().GetArtifacts() {
		artifactsMap[artifactKey] = []*apiV2beta1.Artifact{}
		for _, outputArtifact := range artifactList.Artifacts {
			glog.Infof("outputArtifact in uploadOutputArtifacts call: %s", outputArtifact.Name)
			// Merge executor output artifact info with executor input
			if list, ok := executorOutput.Artifacts[artifactKey]; ok && len(list.Artifacts) > 0 {
				mergeRuntimeArtifacts(list.Artifacts[0], outputArtifact)
			}
			// OCI artifactsMap are accessed via shared storage of a Modelcar
			if strings.HasPrefix(outputArtifact.Uri, "oci://") {
				continue
			}

			artifactType, err := inferArtifactType(outputArtifact.GetType())
			if err != nil {
				return fmt.Errorf("failed to infer artifact type for port %s: %w", artifactKey, err)
			}

			// Metric artifacts don't have a URI, only a numberValue
			if artifactType == apiV2beta1.Artifact_Metric {
				// Each key/value pair in `metadata` equates to a new Artifact
				for key, value := range outputArtifact.GetMetadata().GetFields() {
					numVal, ok := value.Kind.(*structpb.Value_NumberValue)
					if !ok {
						return fmt.Errorf("metric value %q must be a number, got %T", key, value.Kind)
					}
					artifact := &apiV2beta1.Artifact{
						Name:        key,
						Description: "",
						Type:        artifactType,
						NumberValue: &numVal.NumberValue,
						CreatedAt:   timestamppb.Now(),
						// Continue to retain the artifact in metadata for backwards compatibility.
						Metadata: map[string]*structpb.Value{
							key: value,
						},
						Namespace: l.options.Namespace,
					}
					artifactsMap[artifactKey] = append(artifactsMap[artifactKey], artifact)
				}
			} else {
				// In this case we can still encounter metrics of type ClassificationMetric or SlicedClassificationMetric
				// which do not have a numberValue, but nor do they have a URI, their values are stored only in metadata.
				artifact := &apiV2beta1.Artifact{
					Name:        outputArtifact.GetName(),
					Description: "",
					Type:        artifactType,
					Metadata:    outputArtifact.GetMetadata().GetFields(),
					CreatedAt:   timestamppb.Now(),
					Namespace:   l.options.Namespace,
				}

				// In the Classification metric case, the metric data is stored in metadata and
				// not object store
				isNotAMetric := apiV2beta1.Artifact_ClassificationMetric != artifactType &&
					apiV2beta1.Artifact_SlicedClassificationMetric != artifactType

				// If the artifact is not a metric, upload it to the object store and store the URI in the artifact
				if isNotAMetric {
					localPath, err := retrieveArtifactPath(outputArtifact)
					if err != nil {
						glog.Warningf("Output Artifact %q does not have a recognized storage URI %q. Skipping uploading to remote storage.",
							artifactKey, outputArtifact.Uri)
					}
					err = l.objectStore.UploadArtifact(ctx, localPath, outputArtifact.Uri, artifactKey)
					if err != nil {
						return fmt.Errorf("failed to upload output artifact %q to remote storage URI %q: %w", artifactKey, outputArtifact.Uri, err)
					}
					artifact.Uri = util.StringPointer(outputArtifact.Uri)
				}

				artifactsMap[artifactKey] = []*apiV2beta1.Artifact{artifact}
			}
		}
	}

	// Queue artifact creation requests (will be flushed in batch)
	for artifactKey, artifacts := range artifactsMap {
		for _, artifact := range artifacts {
			request := &apiV2beta1.CreateArtifactRequest{
				RunId:       l.options.Run.GetRunId(),
				TaskId:      l.options.Task.GetTaskId(),
				ProducerKey: artifactKey,
				Artifact:    artifact,
				Type:        apiV2beta1.IOType_OUTPUT,
			}
			if l.options.IterationIndex != nil {
				request.IterationIndex = l.options.IterationIndex
				request.Type = apiV2beta1.IOType_ITERATOR_OUTPUT
			}
			l.batchUpdater.QueueArtifact(request)
		}
	}
	return nil
}

// determineIOType determines the appropriate IOType for a propagated output based on the parent task type
// and output definition.
func determineIOType(
	isFirstLevel bool,
	currentIOType apiV2beta1.IOType,
	parentTask *apiV2beta1.PipelineTaskDetail,
	parentOutputKey string,
	parentOutputDefs *pipelinespec.ComponentOutputsSpec,
	isParameter bool,
) apiV2beta1.IOType {
	// For multi-level propagation, inherit the type from the previous level
	if !isFirstLevel {
		return currentIOType
	}

	// First level: determine type based on parent context
	if parentTask.GetType() == apiV2beta1.PipelineTaskDetail_LOOP {
		// For loop iterations, use ITERATOR_OUTPUT
		return apiV2beta1.IOType_ITERATOR_OUTPUT
	}

	// Check if this is a ONE_OF output for condition branches
	if parentTask.GetType() == apiV2beta1.PipelineTaskDetail_CONDITION_BRANCH {
		if isParameter {
			// For parameters, check if it's in output definitions and not a list
			if paramDef, exists := parentOutputDefs.GetParameters()[parentOutputKey]; exists {
				// If it's not marked as a list type, it's a ONE_OF output
				if paramDef.GetParameterType() != pipelinespec.ParameterType_LIST {
					return apiV2beta1.IOType_ONE_OF_OUTPUT
				}
			}
		} else {
			// For artifacts, check if it's in output definitions and not a list
			if artifactDef, exists := parentOutputDefs.GetArtifacts()[parentOutputKey]; exists {
				if !artifactDef.GetIsArtifactList() {
					return apiV2beta1.IOType_ONE_OF_OUTPUT
				}
			}
		}
	}

	// Default to OUTPUT for regular DAG outputs
	return apiV2beta1.IOType_OUTPUT
}

// propagateOutputsUpDAG traverses up the DAG hierarchy and creates artifact-task entries and parameter outputs
// for parent DAGs that declare the current task's outputs in their outputDefinitions.
// This enables output collection from child tasks (e.g., loop iterations) to parent DAGs.
func (l *LauncherV2) propagateOutputsUpDAG(ctx context.Context) error {
	// If this task has no parent, nothing to propagate
	if l.options.ParentTask == nil {
		return nil
	}

	// Refresh the Run once to get all tasks with their latest state
	// This eliminates the need for individual GetTask calls
	fullView := apiV2beta1.GetRunRequest_FULL
	refreshedRun, err := l.clientManager.KFPAPIClient().GetRun(ctx, &apiV2beta1.GetRunRequest{RunId: l.options.Run.GetRunId(), View: &fullView})
	if err != nil {
		return fmt.Errorf("failed to refresh run before propagation: %w", err)
	}

	// Build a map of TaskID -> TaskDetail for fast lookups
	taskMap := make(map[string]*apiV2beta1.PipelineTaskDetail)
	for _, task := range refreshedRun.GetTasks() {
		taskMap[task.GetTaskId()] = task
	}

	// Get the refreshed current task from the map
	currentTask, exists := taskMap[l.options.Task.GetTaskId()]
	if !exists {
		return fmt.Errorf("current task %s not found in refreshed run", l.options.Task.GetTaskId())
	}

	currentTaskOutputs := currentTask.GetOutputs()
	if currentTaskOutputs == nil {
		// No outputs to propagate
		return nil
	}

	hasArtifacts := len(currentTaskOutputs.GetArtifacts()) > 0
	hasParameters := len(currentTaskOutputs.GetParameters()) > 0
	if !hasArtifacts && !hasParameters {
		// No outputs to propagate
		return nil
	}

	// Start traversing up from the immediate parent
	parentTask := l.options.ParentTask
	currentScopePath := l.options.ScopePath
	isFirstLevel := true // Track if this is first-level propagation (from producing task to immediate parent)

	// Track propagated outputs (artifacts and parameters) for next level
	type propagatedInfo struct {
		key      string
		ioType   apiV2beta1.IOType
		producer *apiV2beta1.IOProducer
	}

	for parentTask != nil {
		// Get the parent's component spec to check outputDefinitions
		parentScopePath, err := util.ScopePathFromDotNotation(l.pipelineSpec, parentTask.GetScopePath())
		if err != nil {
			return fmt.Errorf("failed to get scope path for parent task %s: %w", parentTask.GetTaskId(), err)
		}

		parentComponentSpec := parentScopePath.GetLast().GetComponentSpec()
		if parentComponentSpec == nil {
			return fmt.Errorf("parent task %s has no component spec", parentTask.GetTaskId())
		}

		parentOutputDefs := parentComponentSpec.GetOutputDefinitions()
		if parentOutputDefs == nil {
			// Parent has no output definitions, stop propagating
			break
		}

		hasParentArtifactOutputs := len(parentOutputDefs.GetArtifacts()) > 0
		hasParentParameterOutputs := len(parentOutputDefs.GetParameters()) > 0

		if !hasParentArtifactOutputs && !hasParentParameterOutputs {
			// Parent has no output definitions, stop propagating
			break
		}

		// Get child task name for matching outputs
		childTaskName := currentScopePath.GetLast().GetTaskSpec().GetTaskInfo().GetName()

		newPropagatedArtifacts := make(map[string]propagatedInfo)
		newPropagatedParameters := make(map[string]propagatedInfo)

		// Propagate artifacts
		for _, artifactIO := range currentTaskOutputs.GetArtifacts() {
			for _, artifact := range artifactIO.GetArtifacts() {
				// Find the matching output key in parent's output definitions
				matchingParentKey := findMatchingParentOutputKeyForChild(
					childTaskName,
					parentComponentSpec,
					artifactIO.GetArtifactKey(),
					parentOutputDefs,
				)

				if matchingParentKey == "" {
					// This output is not declared in parent's outputDefinitions
					continue
				}

				// Determine the correct IOType
				ioType := determineIOType(
					isFirstLevel,
					artifactIO.GetType(),
					parentTask,
					matchingParentKey,
					parentOutputDefs,
					false, // isParameter = false
				)

				// Create artifact-task entry for the parent
				// Producer is the child task from parent's perspective, not the original producing task
				producer := &apiV2beta1.IOProducer{
					TaskName: childTaskName,
				}

				// Only a Runtime Task in an iteration can have an Output and an Iteration Index
				// for its output.
				if ioType == apiV2beta1.IOType_ITERATOR_OUTPUT && currentTask.TypeAttributes != nil && currentTask.TypeAttributes.IterationIndex != nil {
					producer.Iteration = artifactIO.Producer.Iteration
				}

				artifactTask := &apiV2beta1.ArtifactTask{
					ArtifactId: artifact.GetArtifactId(),
					TaskId:     parentTask.GetTaskId(),
					RunId:      l.options.Run.GetRunId(),
					Key:        matchingParentKey,
					Type:       ioType,
					Producer:   producer,
				}

				// Queue artifact-task creation instead of creating immediately
				l.batchUpdater.QueueArtifactTask(artifactTask)

				// Track this artifact for next level propagation with its IOType
				newPropagatedArtifacts[artifact.GetArtifactId()] = propagatedInfo{
					key:      matchingParentKey,
					ioType:   ioType,
					producer: producer,
				}
			}
		}

		// Propagate parameters
		// Use the parent task from the task map if we have parameters to propagate
		currentParentTask, exists := taskMap[parentTask.GetTaskId()]
		if !exists {
			return fmt.Errorf("parent task %s not found in task map", parentTask.GetTaskId())
		}

		// Initialize outputs if needed
		if currentParentTask.Outputs == nil {
			currentParentTask.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{}
		}

		for _, paramIO := range currentTaskOutputs.GetParameters() {
			// Find the matching output key in parent's output definitions
			matchingParentKey := findMatchingParentOutputKeyForChildParameter(
				childTaskName,
				parentComponentSpec,
				paramIO.GetParameterKey(),
				parentOutputDefs,
			)

			if matchingParentKey == "" {
				// This output is not declared in parent's outputDefinitions
				continue
			}

			// Determine the correct IOType
			ioType := determineIOType(
				isFirstLevel,
				paramIO.GetType(),
				parentTask,
				matchingParentKey,
				parentOutputDefs,
				true, // isParameter = true
			)

			// Create parameter entry for the parent
			// Producer is the child task from parent's perspective, not the original producing task
			paramProducer := &apiV2beta1.IOProducer{
				TaskName: childTaskName,
			}

			// Include iteration index for IOType_OUTPUT type
			if ioType == apiV2beta1.IOType_ITERATOR_OUTPUT && currentTask.TypeAttributes != nil && currentTask.TypeAttributes.IterationIndex != nil {
				paramProducer.Iteration = paramIO.Producer.Iteration
			}

			newParam := &apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: matchingParentKey,
				Value:        paramIO.GetValue(),
				Type:         ioType,
				Producer:     paramProducer,
			}

			// Accumulate parameter to parent task (will queue update later)
			currentParentTask.Outputs.Parameters = append(currentParentTask.Outputs.Parameters, newParam)

			// Track this parameter for next level propagation with its IOType
			// Use parameter key as the identifier since parameters don't have IDs like artifacts
			paramIdentifier := fmt.Sprintf("%s:%s", paramIO.GetParameterKey(), paramIO.GetValue().String())
			newPropagatedParameters[paramIdentifier] = propagatedInfo{
				key:      matchingParentKey,
				ioType:   ioType,
				producer: paramProducer,
			}
		}

		// Queue parent task update if we modified it with parameters
		if len(newPropagatedParameters) > 0 {
			l.batchUpdater.QueueTaskUpdate(currentParentTask)
		}

		// Move up to the next parent
		if parentTask.ParentTaskId == nil || *parentTask.ParentTaskId == "" {
			break
		}

		// Get the next parent task from the task map
		nextParent, exists := taskMap[*parentTask.ParentTaskId]
		if !exists {
			return fmt.Errorf("next parent task %s not found in task map", *parentTask.ParentTaskId)
		}

		// For the next level, we only want to propagate the outputs we just added to this parent
		// Build a new currentTaskOutputs with only the newly propagated outputs
		newTaskOutputs := &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Artifacts:  []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{},
			Parameters: []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{},
		}

		// Build artifact outputs for next level
		for artifactID, info := range newPropagatedArtifacts {
			// Find the artifact object
			var foundArtifact *apiV2beta1.Artifact
			for _, artifactIO := range currentTaskOutputs.GetArtifacts() {
				for _, artifact := range artifactIO.GetArtifacts() {
					if artifact.GetArtifactId() == artifactID {
						foundArtifact = artifact
						break
					}
				}
				if foundArtifact != nil {
					break
				}
			}

			if foundArtifact != nil {
				IOArtifact := &apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
					ArtifactKey: info.key,
					Artifacts:   []*apiV2beta1.Artifact{foundArtifact},
					Type:        info.ioType,
					Producer:    info.producer,
				}
				newTaskOutputs.Artifacts = append(newTaskOutputs.Artifacts, IOArtifact)
			}
		}

		// Build parameter outputs for next level
		for paramIdentifier, info := range newPropagatedParameters {
			// Find the parameter object by matching key and value
			var foundParam *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
			for _, paramIO := range currentTaskOutputs.GetParameters() {
				identifier := fmt.Sprintf("%s:%s", paramIO.GetParameterKey(), paramIO.GetValue().String())
				if identifier == paramIdentifier {
					foundParam = paramIO
					break
				}
			}

			if foundParam != nil {
				newTaskOutputs.Parameters = append(newTaskOutputs.Parameters, &apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					ParameterKey: info.key,
					Value:        foundParam.GetValue(),
					Type:         info.ioType,
					Producer:     foundParam.GetProducer(),
				})
			}
		}

		if len(newTaskOutputs.GetArtifacts()) == 0 && len(newTaskOutputs.GetParameters()) == 0 {
			// No more outputs to propagate
			break
		}

		// Move to the next level
		currentTaskOutputs = newTaskOutputs
		currentTask = parentTask
		parentTask = nextParent
		currentScopePath = parentScopePath
		isFirstLevel = false // After the first iteration, we're doing multi-level propagation
	}

	return nil
}

// findMatchingParentOutputKeyForChild finds the parent output key that corresponds to the child's output.
// This is a simplified version that takes the child task name directly as a parameter.
func findMatchingParentOutputKeyForChild(
	childTaskName string,
	parentComponentSpec *pipelinespec.ComponentSpec,
	childOutputKey string,
	parentOutputDefs *pipelinespec.ComponentOutputsSpec,
) string {
	// Get the task spec from the parent's perspective
	if parentComponentSpec == nil || parentComponentSpec.GetDag() == nil {
		return ""
	}

	// Look through parent's DAG tasks to find the child task
	for _, dagTask := range parentComponentSpec.GetDag().GetTasks() {
		if dagTask.GetTaskInfo().GetName() != childTaskName {
			continue
		}
		// Found the child task in parent's DAG
		// Check the task's output selectors
		if dagTask.GetComponentRef() != nil {
			// Look at the parent's output definitions to find which one uses this task's output
			for parentOutputKey := range parentOutputDefs.GetArtifacts() {
				// Check if this parent output is sourced from the child task
				// The parent output may be directly from task output or from an artifact selector
				if artifactSelectorMatches(parentComponentSpec, parentOutputKey, childTaskName, childOutputKey) {
					return parentOutputKey
				}
			}
		}
	}

	return ""
}

// findMatchingParentOutputKeyForChildParameter finds the parent output key that corresponds to the child's parameter output.
func findMatchingParentOutputKeyForChildParameter(
	childTaskName string,
	parentComponentSpec *pipelinespec.ComponentSpec,
	childOutputKey string,
	parentOutputDefs *pipelinespec.ComponentOutputsSpec,
) string {
	// Get the task spec from the parent's perspective
	if parentComponentSpec == nil || parentComponentSpec.GetDag() == nil {
		return ""
	}

	// Look through parent's DAG tasks to find the child task
	for _, dagTask := range parentComponentSpec.GetDag().GetTasks() {
		if dagTask.GetTaskInfo().GetName() != childTaskName {
			continue
		}

		// Found the child task in parent's DAG
		// Check the task's output selectors
		if dagTask.GetComponentRef() != nil {
			// Look at the parent's output definitions to find which one uses this task's parameter output
			for parentOutputKey := range parentOutputDefs.GetParameters() {
				// Check if this parent output is sourced from the child task
				if parameterSelectorMatches(parentComponentSpec, parentOutputKey, childTaskName, childOutputKey) {
					return parentOutputKey
				}
			}
		}
	}

	return ""
}

// parameterSelectorMatches checks if a parent output parameter selector matches the child task output
func parameterSelectorMatches(
	parentComponentSpec *pipelinespec.ComponentSpec,
	parentOutputKey string,
	childTaskName string,
	childOutputKey string,
) bool {
	// Check parameter selectors
	dag := parentComponentSpec.GetDag()
	if dag == nil || dag.GetOutputs() == nil {
		return false
	}

	parameterSelectors := dag.GetOutputs().GetParameters()
	if parameterSelectors == nil {
		return false
	}

	selector, exists := parameterSelectors[parentOutputKey]
	if !exists {
		return false
	}

	// Check if the selector references the child task
	// Check value_from_parameter (single parameter selector)
	if paramSelector := selector.GetValueFromParameter(); paramSelector != nil {
		if paramSelector.GetProducerSubtask() == childTaskName &&
			paramSelector.GetOutputParameterKey() == childOutputKey {
			return true
		}
	}

	// Check value_from_oneof (list of parameter selectors for condition branches)
	if oneofSelector := selector.GetValueFromOneof(); oneofSelector != nil {
		for _, paramSelector := range oneofSelector.GetParameterSelectors() {
			if paramSelector.GetProducerSubtask() == childTaskName &&
				paramSelector.GetOutputParameterKey() == childOutputKey {
				return true
			}
		}
	}

	return false
}

// artifactSelectorMatches checks if a parent output artifact selector matches the child task output
func artifactSelectorMatches(
	parentComponentSpec *pipelinespec.ComponentSpec,
	parentOutputKey string,
	childTaskName string,
	childOutputKey string,
) bool {
	// Check artifact selectors
	dag := parentComponentSpec.GetDag()
	if dag == nil || dag.GetOutputs() == nil {
		return false
	}

	artifactSelectors := dag.GetOutputs().GetArtifacts()
	if artifactSelectors == nil {
		return false
	}

	selector, exists := artifactSelectors[parentOutputKey]
	if !exists {
		return false
	}

	// Check if the selector references the child task
	for _, artifactSelector := range selector.GetArtifactSelectors() {
		if artifactSelector.GetProducerSubtask() == childTaskName &&
			artifactSelector.GetOutputArtifactKey() == childOutputKey {
			return true
		}
	}

	return false
}

// waitForModelcar assumes the Modelcar has already been validated by the init container on the launcher
// pod. This waits for the Modelcar as a sidecar container to be ready.
func waitForModelcar(artifactURI string, localPath string) error {
	glog.Infof("Waiting for the Modelcar %s to be available", artifactURI)

	for {
		_, err := os.Stat(localPath)
		if err == nil {
			glog.Infof("The Modelcar is now available at %s", localPath)

			return nil
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf(
				"failed to see if the artifact %s was ready at %s; ensure the main container and Modelcar "+
					"container have the same UID (can be set with the PIPELINE_RUN_AS_USER environment variable on "+
					"the API server): %v",
				artifactURI, localPath, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (l *LauncherV2) downloadArtifacts(ctx context.Context) error {
	for artifactKey, artifactList := range l.executorInput.GetInputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			// Skip downloading if the artifact is flagged as already present in the workspace
			if artifact.GetMetadata() != nil {
				if v, ok := artifact.GetMetadata().GetFields()["_kfp_workspace"]; ok && v.GetBoolValue() {
					continue
				}
			}
			// Iterating through the artifact list allows for collected artifacts to be properly consumed.
			localPath, err := LocalPathForURI(artifact.Uri)
			if err != nil {
				glog.Warningf("Input Artifact %q does not have a recognized storage URI %q. Skipping downloading to local path.", artifactKey, artifact.Uri)
				continue
			}
			// OCI artifacts are accessed via shared storage of a Modelcar
			if strings.HasPrefix(artifact.Uri, "oci://") {
				err := waitForModelcar(artifact.Uri, localPath)
				if err != nil {
					return err
				}
				continue
			}

			err = l.objectStore.DownloadArtifact(ctx, artifact.Uri, localPath, artifactKey)
			if err != nil {
				return fmt.Errorf("failed to download input artifact %q from remote storage URI %q: %w", artifactKey, artifact.Uri, err)
			}
		}
	}
	return nil
}

func compileCmdAndArgs(executorInput *pipelinespec.ExecutorInput, cmd string, args []string) (string, []string, error) {
	placeholders, err := getPlaceholders(executorInput)
	if err != nil {
		return "", nil, err
	}
	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}
	executorInputJSONKey := "{{$}}"
	executorInputJSONString := string(executorInputJSON)

	compiledCmd := strings.ReplaceAll(cmd, executorInputJSONKey, executorInputJSONString)
	compiledArgs := make([]string, 0, len(args))
	for placeholder, replacement := range placeholders {
		cmd = strings.ReplaceAll(cmd, placeholder, replacement)
	}
	for _, arg := range args {
		compiledArgTemplate := strings.ReplaceAll(arg, executorInputJSONKey, executorInputJSONString)
		for placeholder, replacement := range placeholders {
			compiledArgTemplate = strings.ReplaceAll(compiledArgTemplate, placeholder, replacement)
		}
		compiledArgs = append(compiledArgs, compiledArgTemplate)
	}
	return compiledCmd, compiledArgs, nil
}

// Add executor input placeholders to provided map.
func getPlaceholders(executorInput *pipelinespec.ExecutorInput) (placeholders map[string]string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get placeholders: %w", err)
		}
	}()
	placeholders = make(map[string]string)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}

	// Read input artifact metadata.
	for name, artifactList := range executorInput.GetInputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		inputArtifact := artifactList.Artifacts[0]

		// Prepare input uri placeholder.
		key := fmt.Sprintf(`{{$.inputs.artifacts['%s'].uri}}`, name)
		placeholders[key] = inputArtifact.Uri

		// If the artifact is marked as already in the workspace, map to the workspace path
		// with the same shape as LocalPathForURI, but rebased under the workspace mount.
		if inputArtifact.GetMetadata() != nil {
			if v, ok := inputArtifact.GetMetadata().GetFields()["_kfp_workspace"]; ok && v.GetBoolValue() {
				localPath, lerr := LocalWorkspacePathForURI(inputArtifact.Uri)
				if lerr != nil {
					return nil, fmt.Errorf("failed to get local workspace path for input artifact %q: %w", name, lerr)
				}
				key = fmt.Sprintf(`{{$.inputs.artifacts['%s'].path}}`, name)
				placeholders[key] = localPath
				continue
			}
		}

		localPath, err := LocalPathForURI(inputArtifact.Uri)
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

		localPath, err := LocalPathForURI(outputArtifact.Uri)
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

func getArtifactSchemaType(schema *pipelinespec.ArtifactTypeSchema) (string, error) {
	switch t := schema.Kind.(type) {
	case *pipelinespec.ArtifactTypeSchema_InstanceSchema:
		return t.InstanceSchema, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaTitle:
		return t.SchemaTitle, nil
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

	if src.CustomPath != nil && *src.CustomPath != "" {
		dst.CustomPath = src.CustomPath
	}
}

func (l *LauncherV2) getExecutorOutputFile(path string) (*pipelinespec.ExecutorOutput, error) {
	// collect user executor output file
	executorOutput := &pipelinespec.ExecutorOutput{
		ParameterValues: map[string]*structpb.Value{},
		Artifacts:       map[string]*pipelinespec.ArtifactList{},
	}

	_, err := l.fileSystem.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			glog.Infof("output metadata file does not exist in %s", path)
			// If file doesn't exist, return an empty ExecutorOutput.
			return executorOutput, nil
		} else {
			return nil, fmt.Errorf("failed to stat output metadata file %q: %w", path, err)
		}
	}

	b, err := l.fileSystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read output metadata file %q: %w", path, err)
	}
	glog.Infof("ExecutorOutput: %s", prettyPrint(string(b)))

	if err := protojson.Unmarshal(b, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshall ExecutorOutput in file %q: %w", path, err)
	}

	return executorOutput, nil
}

func LocalPathForURI(uri string) (string, error) {
	// Used for local debugging
	rootPath := os.Getenv("ARTIFACT_LOCAL_PATH")

	if strings.HasPrefix(uri, "gs://") {
		return fmt.Sprintf("%s/gcs/", rootPath) + strings.TrimPrefix(uri, "gs://"), nil
	}
	if strings.HasPrefix(uri, "minio://") {
		return fmt.Sprintf("%s/minio/", rootPath) + strings.TrimPrefix(uri, "minio://"), nil
	}
	if strings.HasPrefix(uri, "s3://") {
		return fmt.Sprintf("%s/s3/", rootPath) + strings.TrimPrefix(uri, "s3://"), nil
	}
	if strings.HasPrefix(uri, "oci://") {
		return fmt.Sprintf("%s/oci/", rootPath) + strings.ReplaceAll(strings.TrimPrefix(uri, "oci://"), "/", "_") + "/models", nil
	}
	return "", fmt.Errorf("failed to generate local path for URI %s: unsupported storage scheme", uri)
}

func retrieveArtifactPath(artifact *pipelinespec.RuntimeArtifact) (string, error) {
	// If artifact custom path is set, use custom path. Otherwise, use URI.
	customPath := artifact.CustomPath
	if customPath != nil {
		return *customPath, nil
	} else {
		return LocalPathForURI(artifact.Uri)
	}
}


// LocalWorkspacePathForURI returns the local workspace path for a given artifact URI.
// It preserves the same path shape as LocalPathForURI, but rebases it under the
// workspace artifacts directory: /kfp-workspace/.artifacts/...
func LocalWorkspacePathForURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "oci://") {
		return "", fmt.Errorf("failed to generate workspace path for URI %s: OCI not supported for workspace artifacts", uri)
	}
	localPath, err := LocalPathForURI(uri)
	if err != nil {
		return "", err
	}
	// Rebase under the workspace mount, stripping the leading '/'
	return filepath.Join(WorkspaceMountPath, ".artifacts", strings.TrimPrefix(localPath, "/")), nil
}

func (l *LauncherV2) prepareOutputFolders(executorInput *pipelinespec.ExecutorInput) error {
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		dir := filepath.Dir(parameter.OutputFile)
		if err := l.fileSystem.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}

		for _, outputArtifact := range artifactList.Artifacts {

			localPath, err := LocalPathForURI(outputArtifact.Uri)
			if err != nil {
				return fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
			}

			if err := l.fileSystem.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
			}
		}
	}

	return nil
}
