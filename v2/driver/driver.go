package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/kubeflow/pipelines/v2/cacheutils"
	"github.com/kubeflow/pipelines/v2/component"
	"github.com/kubeflow/pipelines/v2/config"
	"github.com/kubeflow/pipelines/v2/expression"
	"github.com/kubeflow/pipelines/v2/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TODO(capri-xiyue): Move driver to component package
// Driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// optional, iteration index. -1 means not an iteration.
	IterationIndex int

	// optional, required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	Namespace     string

	// optional, required by non-root drivers
	Task           *pipelinespec.PipelineTaskSpec
	DAGExecutionID int64

	// optional, required only by container driver
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
}

// Identifying information used for error messages
func (o Options) info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.RunID)
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", task=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.Task.GetComponentRef().GetName() != "" {
		msg = msg + fmt.Sprintf(", component=%q", o.Task.GetComponentRef().GetName())
	}
	if o.DAGExecutionID != 0 {
		msg = msg + fmt.Sprintf(", dagExecutionID=%v", o.DAGExecutionID)
	}
	if o.IterationIndex >= 0 {
		msg = msg + fmt.Sprintf(", iterationIndex=%v", o.IterationIndex)
	}
	if o.RuntimeConfig != nil {
		msg = msg + ", runtimeConfig" // this only means runtimeConfig is not empty
	}
	if o.Component.GetImplementation() != nil {
		msg = msg + ", componentSpec" // this only means componentSpec is not empty
	}
	return msg
}

type Execution struct {
	ID             int64
	ExecutorInput  *pipelinespec.ExecutorInput
	IterationCount *int  // number of iterations, -1 means not an iterator
	Condition      *bool // true -> trigger the task, false -> not trigger the task, nil -> the task is unconditional

	// only specified when this is a Container execution
	Cached       *bool
	PodSpecPatch string
}

func (e *Execution) WillTrigger() bool {
	if e == nil || e.Condition == nil {
		return true
	}
	return *e.Condition
}

func RootDAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.info(), err)
		}
	}()
	err = validateRootDAG(opts)
	if err != nil {
		return nil, err
	}
	// TODO(v2): in pipeline spec, rename GCS output directory to pipeline root.
	pipelineRoot := opts.RuntimeConfig.GetGcsOutputDirectory()
	if pipelineRoot != "" {
		glog.Infof("PipelineRoot=%q", pipelineRoot)
	} else {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
		}
		k8sClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
		}
		cfg, err := config.FromConfigMap(ctx, k8sClient, opts.Namespace)
		if err != nil {
			return nil, err
		}
		pipelineRoot = cfg.DefaultPipelineRoot()
		glog.Infof("PipelineRoot=%q from default config", pipelineRoot)
	}
	// TODO(Bobgy): fill in run resource.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, opts.Namespace, "run-resource", pipelineRoot)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: opts.RuntimeConfig.GetParameterValues(),
		},
	}
	// TODO(Bobgy): validate executorInput matches component spec types
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.Name = fmt.Sprintf("run/%s", opts.RunID)
	exec, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", exec)
	// No need to return ExecutorInput, because tasks in the DAG will resolve
	// needed info from MLMD.
	return &Execution{ID: exec.GetID()}, nil
}

func validateRootDAG(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.RunID == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.DAGExecutionID != 0 {
		return fmt.Errorf("DAG execution ID is unnecessary")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

func Container(ctx context.Context, opts Options, mlmd *metadata.Client, cacheClient *cacheutils.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	err = validateContainer(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts.Task, opts.Component.GetInputDefinitions(), mlmd, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}
	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(pipeline.GetPipelineRoot(), opts.Task.GetTaskInfo().GetName(), opts.Component.GetOutputDefinitions())
	}

	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	if execution.WillTrigger() && opts.Task.GetCachingOptions().GetEnableCache() {
		glog.Infof("Task {%s} enables cache", opts.Task.GetTaskInfo().GetName())
		fingerPrint, err := getFingerPrint(opts, executorInput)
		if err != nil {
			return execution, fmt.Errorf("failure while getting fingerPrint: %w", err)
		}
		cachedMLMDExecutionID, err := cacheClient.GetExecutionCache(fingerPrint, "pipeline/"+opts.PipelineName, opts.Namespace)
		if err != nil {
			return execution, fmt.Errorf("failure while getting executionCache: %w", err)
		}
		ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
		ecfg.FingerPrint = fingerPrint
	}
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	if !execution.WillTrigger() {
		return execution, nil
	}

	cached := false
	execution.Cached = &cached
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, executorInput, opts.Component.GetOutputDefinitions(), mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return execution, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
			return execution, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		glog.Infof("Cached")
		*execution.Cached = true
		return execution, nil
	}

	execution.PodSpecPatch, err = makePodSpecPatch(opts.Container, opts.Component, executorInput, execution.ID, opts.PipelineName, opts.RunID)
	if err != nil {
		return execution, err
	}
	return execution, nil
}

// makePodSpecPatch generates a strategic merge patch for pod spec, it is merged
// to container base template generated in compiler/container.go. Therefore, only
// dynamic values are patched here. The volume mounts / configmap mounts are
// defined in compiler, because they are static.
func makePodSpecPatch(
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	componentSpec *pipelinespec.ComponentSpec,
	executorInput *pipelinespec.ExecutorInput,
	executionID int64,
	pipelineName string,
	runID string,
) (string, error) {
	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return "", fmt.Errorf("failed to make podSpecPatch: %w", err)
	}
	componentJSON, err := protojson.Marshal(componentSpec)
	if err != nil {
		return "", fmt.Errorf("failed to make podSpecPatch: %w", err)
	}

	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	launcherCmd := []string{
		// TODO(Bobgy): workaround argo emissary executor bug, after we upgrade to an argo version with the bug fix, we can remove the following line.
		// Reference: https://github.com/argoproj/argo-workflows/issues/7406
		"/var/run/argo/argoexec", "emissary", "--",
		component.KFPLauncherPath,
		// TODO(Bobgy): no need to pass pipeline_name and run_id, these info can be fetched via pipeline context and pipeline run context which have been created by root DAG driver.
		"--pipeline_name", pipelineName,
		"--run_id", runID,
		"--execution_id", fmt.Sprintf("%v", executionID),
		"--executor_input", string(executorInputJSON),
		"--component_spec", string(componentJSON),
		"--pod_name",
		fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid",
		fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--mlmd_server_address",
		fmt.Sprintf("$(%s)", component.EnvMetadataHost),
		"--mlmd_server_port",
		fmt.Sprintf("$(%s)", component.EnvMetadataPort),
		"--", // separater before user command and args
	}
	res := k8score.ResourceRequirements{
		Limits: map[k8score.ResourceName]k8sres.Quantity{},
	}
	memoryLimit := container.GetResources().GetMemoryLimit()
	if memoryLimit != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%vG", memoryLimit))
		if err != nil {
			return "", err
		}
		res.Limits[k8score.ResourceMemory] = q
	}
	cpuLimit := container.GetResources().GetCpuLimit()
	if cpuLimit != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%v", cpuLimit))
		if err != nil {
			return "", err
		}
		res.Limits[k8score.ResourceCPU] = q
	}
	accelerator := container.GetResources().GetAccelerator()
	if accelerator != nil {
		return "", fmt.Errorf("accelerator resources are not supported yet: https://github.com/kubeflow/pipelines/issues/7043")
	}
	podSpec := &k8score.PodSpec{
		Containers: []k8score.Container{{
			Name:      "main", // argo task user container is always called "main"
			Command:   launcherCmd,
			Args:      userCmdArgs,
			Image:     container.Image,
			Resources: res,
		}},
	}
	podSpecPatchBytes, err := json.Marshal(podSpec)
	if err != nil {
		return "", fmt.Errorf("JSON marshaling pod spec patch: %w", err)
	}
	return string(podSpecPatchBytes), nil
}

// TODO(Bobgy): merge DAG driver and container driver, because they are very similar.
func DAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.info(), err)
		}
	}()
	err = validateDAG(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts.Task, opts.Component.GetInputDefinitions(), mlmd, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()
	if opts.Task.GetArtifactIterator() != nil {
		return execution, fmt.Errorf("ArtifactIterator is not implemented")
	}
	isIterator := opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0
	if execution.WillTrigger() && isIterator {
		iterator := opts.Task.GetParameterIterator()
		value, ok := executorInput.GetInputs().GetParameterValues()[iterator.GetItems().GetInputParameter()]
		report := func(err error) error {
			return fmt.Errorf("iterating on item input %q failed: %w", iterator.GetItemInput(), err)
		}
		if !ok {
			return execution, report(fmt.Errorf("cannot find input parameter"))
		}
		items, err := getItems(value)
		if err != nil {
			return execution, report(err)
		}
		count := len(items)
		ecfg.IterationCount = &count
		execution.IterationCount = &count
	}
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	return execution, nil
}

// Get iteration items from a structpb.Value.
// Return value may be
// * a list of JSON serializable structs
// * a list of structpb.Value
func getItems(value *structpb.Value) (items []*structpb.Value, err error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_ListValue:
		return v.ListValue.GetValues(), nil
	case *structpb.Value_StringValue:
		listValue := structpb.Value{}
		if err = listValue.UnmarshalJSON([]byte(v.StringValue)); err != nil {
			return nil, err
		}
		return listValue.GetListValue().GetValues(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be iterated", v)
	}
}

func reuseCachedOutputs(ctx context.Context, executorInput *pipelinespec.ExecutorInput, outputDefinitions *pipelinespec.ComponentOutputsSpec, mlmd *metadata.Client, cachedMLMDExecutionID string) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {
	cachedMLMDExecutionIDInt64, err := strconv.ParseInt(cachedMLMDExecutionID, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while transfering cachedMLMDExecutionID %s from string to int64: %w", cachedMLMDExecutionID, err)
	}
	execution, err := mlmd.GetExecution(ctx, cachedMLMDExecutionIDInt64)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while getting execution of cachedMLMDExecutionID %v: %w", cachedMLMDExecutionIDInt64, err)
	}
	executorOutput := &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	}
	_, outputs, err := execution.GetParameters()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect output parameters from cache: %w", err)
	}
	executorOutput.ParameterValues = outputs
	outputArtifacts, err := collectOutputArtifactMetadataFromCache(ctx, executorInput, cachedMLMDExecutionIDInt64, mlmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed collect output artifact metadata from cache: %w", err)
	}
	return executorOutput, outputArtifacts, nil
}

func collectOutputArtifactMetadataFromCache(ctx context.Context, executorInput *pipelinespec.ExecutorInput, cachedMLMDExecutionID int64, mlmd *metadata.Client) ([]*metadata.OutputArtifact, error) {
	outputArtifacts, err := mlmd.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MLMDOutputArtifactsByName by executionId %v: %w", cachedMLMDExecutionID, err)
	}

	// Register artifacts with MLMD.
	registeredMLMDArtifacts := make([]*metadata.OutputArtifact, 0, len(executorInput.GetOutputs().GetArtifacts()))
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		artifact := artifactList.Artifacts[0]
		outputArtifact, ok := outputArtifacts[name]
		if !ok {
			return nil, fmt.Errorf("unable to find artifact with name %v in mlmd output artifacts", name)
		}
		outputArtifact.Schema = artifact.GetType().GetInstanceSchema()
		registeredMLMDArtifacts = append(registeredMLMDArtifacts, outputArtifact)
	}
	return registeredMLMDArtifacts, nil

}

func getFingerPrint(opts Options, executorInput *pipelinespec.ExecutorInput) (string, error) {
	outputParametersTypeMap := make(map[string]string)
	for outputParamName, outputParamSpec := range opts.Component.GetOutputDefinitions().GetParameters() {
		outputParametersTypeMap[outputParamName] = outputParamSpec.GetParameterType().String()
	}
	userCmdArgs := make([]string, 0, len(opts.Container.Command)+len(opts.Container.Args))
	userCmdArgs = append(userCmdArgs, opts.Container.Command...)
	userCmdArgs = append(userCmdArgs, opts.Container.Args...)

	cacheKey, err := cacheutils.GenerateCacheKey(executorInput.GetInputs(), executorInput.GetOutputs(), outputParametersTypeMap, userCmdArgs, opts.Container.Image)
	if err != nil {
		return "", fmt.Errorf("failure while generating CacheKey: %w", err)
	}
	fingerPrint, err := cacheutils.GenerateFingerPrint(cacheKey)
	return fingerPrint, err
}

func validateContainer(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid container driver args: %w", err)
		}
	}()
	if opts.Container == nil {
		return fmt.Errorf("container spec is required")
	}
	return validateNonRoot(opts)
}

func validateDAG(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid DAG driver args: %w", err)
		}
	}()
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	return validateNonRoot(opts)
}

func validateNonRoot(opts Options) error {
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.RunID == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.Task.GetTaskInfo().GetName() == "" {
		return fmt.Errorf("task spec is required")
	}
	if opts.RuntimeConfig != nil {
		return fmt.Errorf("runtime config is unnecessary")
	}
	if opts.DAGExecutionID == 0 {
		return fmt.Errorf("DAG execution ID is required")
	}
	return nil
}

func resolveInputs(ctx context.Context, dag *metadata.DAG, iterationIndex *int, pipeline *metadata.Pipeline, task *pipelinespec.PipelineTaskSpec, inputsSpec *pipelinespec.ComponentInputsSpec, mlmd *metadata.Client, expr *expression.Expr) (inputs *pipelinespec.ExecutorInput_Inputs, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to resolve inputs: %w", err)
		}
	}()
	inputParams, _, err := dag.Execution.GetParameters()
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG input parameters %+v", inputParams)
	inputs = &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}
	isIterationDriver := iterationIndex != nil

	handleParameterExpressionSelector := func() error {
		for name, paramSpec := range task.GetInputs().GetParameters() {
			var selector string
			if selector = paramSpec.GetParameterExpressionSelector(); selector == "" {
				continue
			}
			wrap := func(e error) error {
				return fmt.Errorf("resolving parameter %q: evaluation of parameter expression selector %q failed: %w", name, selector, e)
			}
			value, ok := inputs.ParameterValues[name]
			if !ok {
				return wrap(fmt.Errorf("value not found in inputs"))
			}
			selected, err := expr.Select(value, selector)
			if err != nil {
				return wrap(err)
			}
			inputs.ParameterValues[name] = selected
		}
		return nil
	}
	handleParamTypeValidationAndConversion := func() error {
		// TODO(Bobgy): verify whether there are inputs not in the inputs spec.
		for name, spec := range inputsSpec.GetParameters() {
			if task.GetParameterIterator() != nil {
				if !isIterationDriver && task.GetParameterIterator().GetItemInput() == name {
					// It's expected that an iterator does not have iteration item input parameter,
					// because only iterations get the item input parameter.
					continue
				}
				if isIterationDriver && task.GetParameterIterator().GetItems().GetInputParameter() == name {
					// It's expected that an iteration does not have iteration items input parameter,
					// because only the iterator has it.
					continue
				}
			}
			value, hasValue := inputs.GetParameterValues()[name]
			if !hasValue {
				if spec.GetDefaultValue() == nil {
					return fmt.Errorf("input parameter %q is required", name)
				}
				inputs.GetParameterValues()[name] = spec.GetDefaultValue()
				value = spec.GetDefaultValue()
			}
			switch spec.GetParameterType() {
			case pipelinespec.ParameterType_STRING:
				_, isValueString := value.GetKind().(*structpb.Value_StringValue)
				if !isValueString {
					// TODO(Bobgy): discuss whether we want to allow auto type conversion
					// all parameter types can be consumed as JSON string
					text, err := metadata.PbValueToText(value)
					if err != nil {
						return fmt.Errorf("converting input parameter %q to string: %w", name, err)
					}
					inputs.GetParameterValues()[name] = structpb.NewStringValue(text)
				}
			default:
				typeMismatch := func(actual string) error {
					return fmt.Errorf("input parameter %q type mismatch: expect %s, got %s", name, spec.GetParameterType(), actual)
				}
				switch v := value.GetKind().(type) {
				case *structpb.Value_NullValue:
					return fmt.Errorf("got null for input parameter %q", name)
				case *structpb.Value_StringValue:
					// TODO(Bobgy): consider whether we support parsing string as JSON for any other types.
					if spec.GetParameterType() != pipelinespec.ParameterType_STRING {
						return typeMismatch("string")
					}
				case *structpb.Value_NumberValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_DOUBLE && spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_INTEGER {
						return typeMismatch("number")
					}
				case *structpb.Value_BoolValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_BOOLEAN {
						return typeMismatch("bool")
					}
				case *structpb.Value_ListValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_LIST {
						return typeMismatch("list")
					}
				case *structpb.Value_StructValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_STRUCT {
						return typeMismatch("struct")
					}
				default:
					return fmt.Errorf("unknown protobuf.Value type: %T", v)
				}
			}
		}
		return nil
	}
	// this function has many branches, so it's hard to add more postprocess steps
	// TODO(Bobgy): consider splitting this function into several sub functions
	defer func() {
		if err == nil {
			err = handleParameterExpressionSelector()
		}
		if err == nil {
			err = handleParamTypeValidationAndConversion()
		}
	}()
	// resolve input parameters
	if isIterationDriver {
		// resolve inputs for iteration driver is very different
		artifacts, err := mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID())
		if err != nil {
			return nil, err
		}
		inputs.ParameterValues = inputParams
		inputs.Artifacts = artifacts
		switch {
		case task.GetArtifactIterator() != nil:
			return nil, fmt.Errorf("artifact iterator not implemented yet")
		case task.GetParameterIterator() != nil:
			itemsInput := task.GetParameterIterator().GetItems().GetInputParameter()
			items, err := getItems(inputs.ParameterValues[itemsInput])
			if err != nil {
				return nil, err
			}
			if *iterationIndex >= len(items) {
				return nil, fmt.Errorf("bug: %v items found, but getting index %v", len(items), *iterationIndex)
			}
			delete(inputs.ParameterValues, itemsInput)
			inputs.ParameterValues[task.GetParameterIterator().GetItemInput()] = items[*iterationIndex]
		default:
			return nil, fmt.Errorf("bug: iteration_index>=0, but task iterator is empty")
		}
		return inputs, nil
	}
	// get executions in context on demand
	var tasksCache map[string]*metadata.Execution
	getDAGTasks := func() (map[string]*metadata.Execution, error) {
		if tasksCache != nil {
			return tasksCache, nil
		}
		tasks, err := mlmd.GetExecutionsInDAG(ctx, dag, pipeline)
		if err != nil {
			return nil, err
		}
		tasksCache = tasks
		return tasks, nil
	}
	for name, paramSpec := range task.GetInputs().GetParameters() {
		paramError := func(err error) error {
			return fmt.Errorf("resolving input parameter %s with spec %s: %w", name, paramSpec, err)
		}
		switch t := paramSpec.Kind.(type) {
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
			componentInput := paramSpec.GetComponentInputParameter()
			if componentInput == "" {
				return nil, paramError(fmt.Errorf("empty component input"))
			}
			v, ok := inputParams[componentInput]
			if !ok {
				return nil, paramError(fmt.Errorf("parent DAG does not have input parameter %s", componentInput))
			}
			inputs.ParameterValues[name] = v

		case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
			taskOutput := paramSpec.GetTaskOutputParameter()
			if taskOutput.GetProducerTask() == "" {
				return nil, paramError(fmt.Errorf("producer task is empty"))
			}
			if taskOutput.GetOutputParameterKey() == "" {
				return nil, paramError(fmt.Errorf("output parameter key is empty"))
			}
			tasks, err := getDAGTasks()
			if err != nil {
				return nil, paramError(err)
			}
			producer, ok := tasks[taskOutput.GetProducerTask()]
			if !ok {
				return nil, paramError(fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()))
			}
			_, outputs, err := producer.GetParameters()
			if err != nil {
				return nil, paramError(fmt.Errorf("get producer output parameters: %w", err))
			}
			param, ok := outputs[taskOutput.GetOutputParameterKey()]
			if !ok {
				return nil, paramError(fmt.Errorf("cannot find output parameter key %q in producer task %q", taskOutput.GetOutputParameterKey(), taskOutput.GetProducerTask()))
			}
			inputs.ParameterValues[name] = param
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
			runtimeValue := paramSpec.GetRuntimeValue()
			switch t := runtimeValue.Value.(type) {
			case *pipelinespec.ValueOrRuntimeParameter_Constant:
				inputs.ParameterValues[name] = runtimeValue.GetConstant()
			// TODO(v2): clean up pipelinespec.Value usages
			case *pipelinespec.ValueOrRuntimeParameter_ConstantValue:
				inputs.Parameters[name] = runtimeValue.GetConstantValue()
			default:
				return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
			}

		// TODO(Bobgy): implement the following cases
		// case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
		default:
			return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
		}
	}
	for name, artifactSpec := range task.GetInputs().GetArtifacts() {
		artifactError := func(err error) error {
			return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
		}
		switch t := artifactSpec.Kind.(type) {
		case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
			return nil, artifactError(fmt.Errorf("component input artifact not implemented yet"))

		case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
			taskOutput := artifactSpec.GetTaskOutputArtifact()
			if taskOutput.GetProducerTask() == "" {
				return nil, artifactError(fmt.Errorf("producer task is empty"))
			}
			if taskOutput.GetOutputArtifactKey() == "" {
				return nil, artifactError(fmt.Errorf("output artifact key is empty"))
			}
			tasks, err := getDAGTasks()
			if err != nil {
				return nil, artifactError(err)
			}
			producer, ok := tasks[taskOutput.GetProducerTask()]
			if !ok {
				return nil, artifactError(fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()))
			}
			// TODO(Bobgy): cache results
			outputs, err := mlmd.GetOutputArtifactsByExecutionId(ctx, producer.GetID())
			if err != nil {
				return nil, artifactError(err)
			}
			artifact, ok := outputs[taskOutput.GetOutputArtifactKey()]
			if !ok {
				return nil, artifactError(fmt.Errorf("cannot find output artifact key %q in producer task %q", taskOutput.GetOutputArtifactKey(), taskOutput.GetProducerTask()))
			}
			runtimeArtifact, err := artifact.ToRuntimeArtifact()
			if err != nil {
				return nil, artifactError(err)
			}
			inputs.Artifacts[name] = &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
			}
		default:
			return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
		}
	}
	// TODO(Bobgy): validate executor inputs match component inputs definition
	return inputs, nil
}

func provisionOutputs(pipelineRoot, taskName string, outputsSpec *pipelinespec.ComponentOutputsSpec) *pipelinespec.ExecutorInput_Outputs {
	outputs := &pipelinespec.ExecutorInput_Outputs{
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
		Parameters: make(map[string]*pipelinespec.ExecutorInput_OutputParameter),
		OutputFile: component.OutputMetadataFilepath,
	}
	for name, artifact := range outputsSpec.GetArtifacts() {
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					Uri:      generateOutputURI(pipelineRoot, name, taskName),
					Type:     artifact.GetArtifactType(),
					Metadata: artifact.GetMetadata(),
				},
			},
		}
	}

	for name := range outputsSpec.GetParameters() {
		outputs.Parameters[name] = &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: fmt.Sprintf("/tmp/kfp/outputs/%s", name),
		}
	}
	return outputs
}

func generateOutputURI(root, artifactName string, taskName string) string {
	// we cannot path.Join(root, taskName, artifactName), because root
	// contains scheme like gs:// and path.Join cleans up scheme to gs:/
	return fmt.Sprintf("%s/%s", strings.TrimRight(root, "/"), path.Join(taskName, artifactName))
}
