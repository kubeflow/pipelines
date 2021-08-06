package component

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LauncherV2Options struct {
	Namespace,
	PodName,
	PodUID,
	PipelineRoot,
	MLMDServerAddress,
	MLMDServerPort string
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
}

func NewLauncherV2(executionID int64, executorInputJSON, componentSpecJSON string, cmdArgs []string, opts *LauncherV2Options) (l *LauncherV2, err error) {
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
	if len(opts.PipelineRoot) == 0 {
		config, err := getLauncherConfig(k8sClient, opts.Namespace)
		if err != nil {
			return nil, err
		}
		opts.PipelineRoot = getDefaultPipelineRoot(config)
		glog.Infof("PipelineRoot defaults to %q.", opts.PipelineRoot)
	}
	metadataClient, err := metadata.NewClient(opts.MLMDServerAddress, opts.MLMDServerPort)
	if err != nil {
		return nil, err
	}
	if err = addOutputs(executorInput, component.GetOutputDefinitions()); err != nil {
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
	}, nil
}

func (l *LauncherV2) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute component: %w", err)
		}
	}()
	execution, err := l.prePublish(ctx)
	if err != nil {
		return err
	}
	bucketConfig, err := objectstore.ParseBucketConfig(l.options.PipelineRoot)
	if err != nil {
		return err
	}
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, bucketConfig)
	if err != nil {
		return err
	}
	executorOutput, err := executeV2(ctx, l.executorInput, l.component, l.command, l.args, bucket, bucketConfig)
	if err != nil {
		return err
	}
	return l.publish(ctx, execution, executorOutput)
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

func (l *LauncherV2) publish(ctx context.Context, execution *metadata.Execution, executorOutput *pipelinespec.ExecutorOutput) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish results to ML Metadata: %w", err)
		}
	}()
	outputParameters, err := metadata.NewParameters(executorOutput.GetParameters())
	if err != nil {
		return err
	}
	// TODO(Bobgy): upload output artifacts.
	// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
	// to publish output artifacts to the context too.
	return l.metadataClient.PublishExecution(ctx, execution, outputParameters, nil, pb.Execution_COMPLETE)
}

// Add outputs info from component spec to executor input.
func addOutputs(executorInput *pipelinespec.ExecutorInput, outputs *pipelinespec.ComponentOutputsSpec) error {
	if executorInput == nil {
		return fmt.Errorf("cannot add outputs to nil executor input")
	}
	if executorInput.Outputs == nil {
		executorInput.Outputs = &pipelinespec.ExecutorInput_Outputs{}
	}
	if executorInput.Outputs.Parameters == nil {
		executorInput.Outputs.Parameters = make(map[string]*pipelinespec.ExecutorInput_OutputParameter)
	}
	// TODO(Bobgy): add output artifacts
	for name, _ := range outputs.GetParameters() {
		executorInput.Outputs.Parameters[name] = &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: fmt.Sprintf("/tmp/kfp/outputs/%s", name),
		}
	}
	return nil
}

func executeV2(ctx context.Context, executorInput *pipelinespec.ExecutorInput, component *pipelinespec.ComponentSpec, cmd string, args []string, bucket *blob.Bucket, bucketConfig *objectstore.Config) (*pipelinespec.ExecutorOutput, error) {
	executorOutput, err := execute(ctx, executorInput, cmd, args, bucket, bucketConfig)
	if err != nil {
		return nil, err
	}

	// Collect Output Parameters
	//
	// These are not added in execute(), because execute() is shared between v2 compatible and v2 engine launcher.
	// In v2 compatible mode, we get output parameter info from runtimeInfo. In v2 engine, we get it from component spec.
	// Because of the difference, we cannot put parameter collection logic in one method.
	if executorOutput.Parameters == nil {
		executorOutput.Parameters = make(map[string]*pipelinespec.Value)
	}
	outputParameters := executorOutput.GetParameters()
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
			return nil, fmt.Errorf("failed to find output parameter name=%q in component spec", name)
		}
		msg := func(err error) error {
			return fmt.Errorf("failed to read output parameter name=%q type=%q path=%q: %w", name, paramSpec.GetType(), param.GetOutputFile(), err)
		}
		b, err := ioutil.ReadFile(param.GetOutputFile())
		if err != nil {
			return nil, msg(err)
		}
		switch paramSpec.GetType() {
		case pipelinespec.PrimitiveType_STRING:
			outputParameters[name] = metadata.StringValue(string(b))
		case pipelinespec.PrimitiveType_INT:
			i, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 0)
			if err != nil {
				return nil, msg(err)
			}
			outputParameters[name] = metadata.IntValue(i)
		case pipelinespec.PrimitiveType_DOUBLE:
			f, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 0)
			if err != nil {
				return nil, msg(err)
			}
			outputParameters[name] = metadata.DoubleValue(f)
		default:
			return nil, msg(fmt.Errorf("unknown type. Expected STRING, INT or DOUBLE"))
		}
	}
	return executorOutput, nil
}
