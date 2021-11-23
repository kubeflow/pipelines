package component

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubeflow/pipelines/v2/cacheutils"
	api "github.com/kubeflow/pipelines/v2/kfp-api"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
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
}

type LauncherV2 struct {
	executionID   int64
	executorInput *pipelinespec.ExecutorInput
	component     *pipelinespec.ComponentSpec
	command       string
	args          []string
	options       LauncherV2Options

	// clients
	metadataClient *metadata.Client
	k8sClient      *kubernetes.Clientset
	cacheClient    *cacheutils.Client
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
	component := &pipelinespec.ComponentSpec{}
	err = protojson.Unmarshal([]byte(componentSpecJSON), component)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec: %w", err)
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
	cacheClient, err := cacheutils.NewClient()
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
	executedStartedTime := time.Now().Unix()
	execution, err := l.prePublish(ctx)
	if err != nil {
		return err
	}
	fingerPrint := execution.FingerPrint()
	bucketConfig, err := objectstore.ParseBucketConfig(execution.GetPipeline().GetPipelineRoot())
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
	executorOutput, outputArtifacts, err := executeV2(ctx, l.executorInput, l.component, l.command, l.args, bucket, bucketConfig, l.metadataClient, l.options.Namespace, l.k8sClient)
	if err != nil {
		return err
	}
	if err := l.publish(ctx, execution, executorOutput, outputArtifacts); err != nil {
		return err
	}
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
func (l *LauncherV2) publish(ctx context.Context, execution *metadata.Execution, executorOutput *pipelinespec.ExecutorOutput, outputArtifacts []*metadata.OutputArtifact) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish results to ML Metadata: %w", err)
		}
	}()
	outputParameters := executorOutput.GetParameterValues()
	// TODO(Bobgy): upload output artifacts.
	// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
	// to publish output artifacts to the context too.
	return l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts, pb.Execution_COMPLETE)
}

func executeV2(ctx context.Context, executorInput *pipelinespec.ExecutorInput, component *pipelinespec.ComponentSpec, cmd string, args []string, bucket *blob.Bucket, bucketConfig *objectstore.Config, metadataClient *metadata.Client, namespace string, k8sClient *kubernetes.Clientset) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {
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
