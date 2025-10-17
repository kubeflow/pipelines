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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"google.golang.org/protobuf/proto"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes"
)

type LauncherV2Options struct {
	Namespace,
	PodName,
	PodUID,
	MLMDServerAddress,
	MLMDServerPort,
	PipelineName,
	RunID string
	PublishLogs   string
	CacheDisabled bool
	// Set to true if apiserver is serving over TLS
	MLPipelineTLSEnabled bool
	// Set to true if metadata server is serving over TLS
	MLMDTLSEnabled bool
	CaCertPath     string
}

type LauncherV2 struct {
	executionID   int64
	executorInput *pipelinespec.ExecutorInput
	component     *pipelinespec.ComponentSpec
	command       string
	args          []string
	options       LauncherV2Options
	clientManager client_manager.ClientManagerInterface
}

// Client is the struct to hold the Kubernetes Clientset
type kubernetesClient struct {
	Clientset kubernetes.Interface
}

// NewLauncherV2 is a factory function that returns an instance of LauncherV2.
func NewLauncherV2(
	ctx context.Context,
	executionID int64,
	executorInputJSON,
	componentSpecJSON string,
	cmdArgs []string,
	opts *LauncherV2Options,
	clientManager client_manager.ClientManagerInterface,
) (l *LauncherV2, err error) {
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
	return &LauncherV2{
		executionID:   executionID,
		executorInput: executorInput,
		component:     component,
		command:       cmdArgs[0],
		args:          cmdArgs[1:],
		options:       *opts,
		clientManager: clientManager,
	}, nil
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

// Execute calls executeV2, updates the cache, and publishes the results to MLMD.
func (l *LauncherV2) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute component: %w", err)
		}
	}()

	defer stopWaitingArtifacts(l.executorInput.GetInputs().GetArtifacts())

	// publish execution regardless the task succeeds or not
	var execution *metadata.Execution
	var executorOutput *pipelinespec.ExecutorOutput
	var outputArtifacts []*metadata.OutputArtifact
	status := pb.Execution_FAILED
	defer func() {
		if execution == nil {
			glog.Errorf("Skipping publish since execution is nil. Original err is: %v", err)
			return
		}

		if perr := l.publish(ctx, execution, executorOutput, outputArtifacts, status); perr != nil {
			if err != nil {
				err = fmt.Errorf("failed to publish execution with error %s after execution failed: %s", perr.Error(), err.Error())
			} else {
				err = perr
			}
		}
		glog.Infof("publish success.")
		// At the end of the current task, we check the statuses of all tasks in
		// the current DAG and update the DAG's status accordingly.
		dag, err := l.clientManager.MetadataClient().GetDAG(ctx, execution.GetExecution().CustomProperties["parent_dag_id"].GetIntValue())
		if err != nil {
			glog.Errorf("DAG Status Update: failed to get DAG: %s", err.Error())
		}
		pipeline, _ := l.clientManager.MetadataClient().GetPipelineFromExecution(ctx, execution.GetID())
		err = l.clientManager.MetadataClient().UpdateDAGExecutionsState(ctx, dag, pipeline)
		if err != nil {
			glog.Errorf("failed to update DAG state: %s", err.Error())
		}
	}()
	executedStartedTime := time.Now().Unix()
	execution, err = l.prePublish(ctx)
	if err != nil {
		return err
	}
	fingerPrint := execution.FingerPrint()
	storeSessionInfo, err := objectstore.GetSessionInfoFromString(execution.GetPipeline().GetStoreSessionInfo())
	if err != nil {
		return err
	}
	pipelineRoot := execution.GetPipeline().GetPipelineRoot()
	bucketConfig, err := objectstore.ParseBucketConfig(pipelineRoot, storeSessionInfo)
	if err != nil {
		return err
	}
	bucket, err := objectstore.OpenBucket(ctx, l.clientManager.K8sClient(), l.options.Namespace, bucketConfig)
	if err != nil {
		return err
	}
	if err = prepareOutputFolders(l.executorInput); err != nil {
		return err
	}
	executorOutput, outputArtifacts, err = executeV2(
		ctx,
		l.executorInput,
		l.component,
		l.command,
		l.args,
		bucket,
		bucketConfig,
		l.clientManager.MetadataClient(),
		l.options.Namespace,
		l.clientManager.K8sClient(),
		l.options.PublishLogs,
		l.options.CaCertPath,
	)
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
			// TODO how to differentiate between shared pipeline and namespaced pipeline
			PipelineName:    "pipeline/" + l.options.PipelineName,
			Namespace:       l.options.Namespace,
			RunId:           l.options.RunID,
			MlmdExecutionID: strconv.FormatInt(id, 10),
			CreatedAt:       timestamppb.New(time.Unix(executedStartedTime, 0)),
			FinishedAt:      timestamppb.New(time.Unix(time.Now().Unix(), 0)),
			Fingerprint:     fingerPrint,
		}
		return l.clientManager.CacheClient().CreateExecutionCache(ctx, task)
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
	execution, err = l.clientManager.MetadataClient().GetExecution(ctx, l.executionID)
	if err != nil {
		return nil, err
	}
	ecfg := &metadata.ExecutionConfig{
		PodName:   l.options.PodName,
		PodUID:    l.options.PodUID,
		Namespace: l.options.Namespace,
	}
	return l.clientManager.MetadataClient().PrePublishExecution(ctx, execution, ecfg)
}

// TODO(Bobgy): consider passing output artifacts info from executor output.
func (l *LauncherV2) publish(
	ctx context.Context,
	execution *metadata.Execution,
	executorOutput *pipelinespec.ExecutorOutput,
	outputArtifacts []*metadata.OutputArtifact,
	status pb.Execution_State,
) (err error) {
	if execution == nil {
		return fmt.Errorf("failed to publish results to ML Metadata: execution is nil")
	}

	var outputParameters map[string]*structpb.Value
	if executorOutput != nil {
		outputParameters = executorOutput.GetParameterValues()
	}

	// TODO(Bobgy): upload output artifacts.
	// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
	// to publish output artifacts to the context too.
	// return l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts, pb.Execution_COMPLETE)
	err = l.clientManager.MetadataClient().PublishExecution(ctx, execution, outputParameters, outputArtifacts, status)
	if err != nil {
		return fmt.Errorf("failed to publish results to ML Metadata: %w", err)
	}

	return nil
}

// executeV2 handles placeholder substitution for inputs, calls execute to
// execute end user logic, and uploads the resulting output Artifacts.
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
	publishLogs string,
	customCAPath string,
) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {

	// Add parameter default values to executorInput, if there is not already a user input.
	// This process is done in the launcher because we let the component resolve default values internally.
	// Variable executorInputWithDefault is a copy so we don't alter the original data.
	executorInputWithDefault, err := addDefaultParams(executorInput, component)
	if err != nil {
		return nil, nil, err
	}

	// Fill in placeholders with runtime values.
	compiledCmd, compiledArgs, err := compileCmdAndArgs(executorInputWithDefault, cmd, args)
	if err != nil {
		return nil, nil, err
	}

	executorOutput, err := execute(
		ctx,
		executorInput,
		compiledCmd,
		compiledArgs,
		bucket,
		bucketConfig,
		namespace,
		k8sClient,
		publishLogs,
		customCAPath,
	)
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
		b, err := os.ReadFile(param.GetOutputFile())
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

// execute downloads input artifacts, prepares the execution environment,
// executes the end user code, and returns the outputs.
func execute(
	ctx context.Context,
	executorInput *pipelinespec.ExecutorInput,
	cmd string,
	args []string,
	bucket *blob.Bucket,
	bucketConfig *objectstore.Config,
	namespace string,
	k8sClient kubernetes.Interface,
	publishLogs string,
	customCAPath string,
) (*pipelinespec.ExecutorOutput, error) {
	if err := downloadArtifacts(ctx, executorInput, bucket, bucketConfig, namespace, k8sClient); err != nil {
		return nil, err
	}

	if err := prepareOutputFolders(executorInput); err != nil {
		return nil, err
	}

	var writer io.Writer
	if publishLogs == "true" {
		writer = getLogWriter(executorInput.Outputs.GetArtifacts())
	} else {
		writer = os.Stdout
	}

	// If a custom CA path is input, append to system CA and save to a temp file for executor access.
	if customCAPath != "" {
		var caBundleTmpPath string
		var err error
		if caBundleTmpPath, err = compileTempCABundleWithCustomCA(customCAPath); err != nil {
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

	// Prepare command that will execute end user code.
	command := exec.Command(cmd, args...)
	command.Stdin = os.Stdin
	// Pipe stdout/stderr to the aforementioned multiWriter.
	command.Stdout = writer
	command.Stderr = writer
	defer glog.Flush()

	// Execute end user code.
	if err := command.Run(); err != nil {
		return nil, err
	}

	return getExecutorOutputFile(executorInput.GetOutputs().GetOutputFile())
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

		for _, outputArtifact := range artifactList.Artifacts {
			glog.Infof("outputArtifact in uploadOutputArtifacts call: ", outputArtifact.Name)

			// Merge executor output artifact info with executor input
			if list, ok := executorOutput.Artifacts[name]; ok && len(list.Artifacts) > 0 {
				mergeRuntimeArtifacts(list.Artifacts[0], outputArtifact)
			}

			// Upload artifacts from local path to remote storages.
			localDir, err := LocalPathForURI(outputArtifact.Uri)
			if err != nil {
				glog.Warningf("Output Artifact %q does not have a recognized storage URI %q. Skipping uploading to remote storage.", name, outputArtifact.Uri)
			} else if !strings.HasPrefix(outputArtifact.Uri, "oci://") {
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
			mlmdArtifact, err := opts.metadataClient.RecordArtifact(ctx, name, schema, outputArtifact, pb.Artifact_LIVE, opts.bucketConfig)
			if err != nil {
				return nil, metadataErr(err)
			}
			outputArtifacts = append(outputArtifacts, mlmdArtifact)
		}
	}
	return outputArtifacts, nil
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
		for _, artifact := range artifactList.Artifacts {
			// Iterating through the artifact list allows for collected artifacts to be properly consumed.
			inputArtifact := artifact
			// Skip downloading if the artifact is flagged as already present in the workspace
			if inputArtifact.GetMetadata() != nil {
				if v, ok := inputArtifact.GetMetadata().GetFields()["_kfp_workspace"]; ok && v.GetBoolValue() {
					continue
				}
			}
			localPath, err := LocalPathForURI(inputArtifact.Uri)
			if err != nil {
				glog.Warningf("Input Artifact %q does not have a recognized storage URI %q. Skipping downloading to local path.", name, inputArtifact.Uri)

				continue
			}

			// OCI artifacts are accessed via shared storage of a Modelcar
			if strings.HasPrefix(inputArtifact.Uri, "oci://") {
				err := waitForModelcar(inputArtifact.Uri, localPath)
				if err != nil {
					return err
				}

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

		// OCI artifacts are accessed via shared storage of a Modelcar
		if strings.HasPrefix(artifact.Uri, "oci://") {
			continue
		}

		// The artifact does not belong under the object store path for this run. Cases:
		// 1. Artifact is cached from a different run, so it may still be in the default bucket, but under a different run id subpath
		// 2. Artifact is imported from the same bucket, but from a different path (re-use the same session)
		// 3. Artifact is imported from a different bucket, or obj store (default to using user env in this case)
		if !strings.HasPrefix(artifact.Uri, defaultBucketConfig.PrefixedBucket()) {
			nonDefaultBucketConfig, parseErr := objectstore.ParseBucketConfigForArtifactURI(artifact.Uri)
			if parseErr != nil {
				return nonDefaultBuckets, fmt.Errorf("failed to parse bucketConfig for output artifact %q with uri %q: %w", name, artifact.GetUri(), parseErr)
			}
			// check if it's same bucket but under a different path, re-use the default bucket session in this case.
			if (nonDefaultBucketConfig.Scheme == defaultBucketConfig.Scheme) && (nonDefaultBucketConfig.BucketName == defaultBucketConfig.BucketName) {
				nonDefaultBucketConfig.SessionInfo = defaultBucketConfig.SessionInfo
			}
			nonDefaultBucket, bucketErr := objectstore.OpenBucket(ctx, k8sClient, namespace, nonDefaultBucketConfig)
			if bucketErr != nil {
				return nonDefaultBuckets, fmt.Errorf("failed to open bucket for output artifact %q with uri %q: %w", name, artifact.GetUri(), bucketErr)
			}
			nonDefaultBuckets[nonDefaultBucketConfig.PrefixedBucket()] = nonDefaultBucket
		}

	}
	return nonDefaultBuckets, nil

}

func compileCmdAndArgs(executorInput *pipelinespec.ExecutorInput, cmd string, args []string) (string, []string, error) {
	placeholders, err := getPlaceholders(executorInput)

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

		// If the artifact is marked as already in the workspace, map the workspace path.
		if inputArtifact.GetMetadata() != nil {
			if v, ok := inputArtifact.GetMetadata().GetFields()["_kfp_workspace"]; ok && v.GetBoolValue() {
				bucketConfig, err := objectstore.ParseBucketConfigForArtifactURI(inputArtifact.Uri)
				if err == nil {
					blobKey, err := bucketConfig.KeyFromURI(inputArtifact.Uri)
					if err == nil {
						localPath := filepath.Join(WorkspaceMountPath, ".artifacts", blobKey)
						key = fmt.Sprintf(`{{$.inputs.artifacts['%s'].path}}`, name)
						placeholders[key] = localPath
						continue
					}
				}
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

	b, err := os.ReadFile(path)
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
	if strings.HasPrefix(uri, "gs://") {
		return "/gcs/" + strings.TrimPrefix(uri, "gs://"), nil
	}
	if strings.HasPrefix(uri, "minio://") {
		return "/minio/" + strings.TrimPrefix(uri, "minio://"), nil
	}
	if strings.HasPrefix(uri, "s3://") {
		return "/s3/" + strings.TrimPrefix(uri, "s3://"), nil
	}
	if strings.HasPrefix(uri, "oci://") {
		return "/oci/" + strings.ReplaceAll(strings.TrimPrefix(uri, "oci://"), "/", "_") + "/models", nil
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

		for _, outputArtifact := range artifactList.Artifacts {

			localPath, err := LocalPathForURI(outputArtifact.Uri)
			if err != nil {
				return fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
			}

			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
			}
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
