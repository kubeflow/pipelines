package component

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/objectstore"
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
	options        LauncherV2Options
	executorInput  *pipelinespec.ExecutorInput
	command        string
	args           []string
	metadataClient *metadata.Client
	k8sClient      *kubernetes.Clientset
}

func NewLauncherV2(executorInputJSON string, cmdArgs []string, opts *LauncherV2Options) (l *LauncherV2, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create component launcher v2: %w", err)
		}
	}()
	err = opts.validate()
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{}
	err = protojson.Unmarshal([]byte(executorInputJSON), executorInput)
	if err != nil {
		return nil, err
	}
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("command and arguments are empty")
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
	return &LauncherV2{options: *opts, executorInput: executorInput, command: cmdArgs[0], args: cmdArgs[1:], metadataClient: metadataClient, k8sClient: k8sClient}, nil
}

func (l *LauncherV2) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute component: %w", err)
		}
	}()
	bucketConfig, err := objectstore.ParseBucketConfig(l.options.PipelineRoot)
	if err != nil {
		return err
	}
	bucket, err := objectstore.OpenBucket(ctx, l.k8sClient, l.options.Namespace, bucketConfig)
	if err != nil {
		return err
	}
	_, err = execute(ctx, l.executorInput, l.command, l.args, bucket, bucketConfig)
	return err
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
