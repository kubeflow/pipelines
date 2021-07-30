package component

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
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
	command       string
	args          []string
	options       LauncherV2Options

	// clients
	metadataClient *metadata.Client
	k8sClient      *kubernetes.Clientset
}

func NewLauncherV2(executionID int64, executorInputJSON string, cmdArgs []string, opts *LauncherV2Options) (l *LauncherV2, err error) {
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
	return &LauncherV2{
		executionID:    executionID,
		executorInput:  executorInput,
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
	_, err = execute(ctx, l.executorInput, l.command, l.args, bucket, bucketConfig)
	if err != nil {
		return err
	}
	return l.publish(ctx, execution)
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

func (l *LauncherV2) publish(ctx context.Context, execution *metadata.Execution) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish results to ML Metadata: %w", err)
		}
	}()
	// TODO(Bobgy): read output parameters from local path, and add them to executorOutput.
	// TODO(Bobgy): upload output artifacts.
	// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
	// to publish output artifacts to the context too.
	if err := l.metadataClient.PublishExecution(ctx, execution, nil, nil, pb.Execution_COMPLETE); err != nil {
		return fmt.Errorf("unable to publish execution: %w", err)
	}
	return nil
}
