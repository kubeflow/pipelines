package driver

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/cacheutils"
	"github.com/kubeflow/pipelines/v2/component"
	"github.com/kubeflow/pipelines/v2/config"
	"github.com/kubeflow/pipelines/v2/metadata"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"path"
	"strconv"
	"strings"
)

// TODO Move driver to component package
// Driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	// required by non-root drivers
	Task *pipelinespec.PipelineTaskSpec
	// required only by container driver
	DAGExecutionID int64
	DAGContextID   int64
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	// required only by root DAG driver
	Namespace string
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
	if o.DAGContextID != 0 {
		msg = msg + fmt.Sprintf(", dagContextID=%v", o.DAGContextID)
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
	ID            int64
	Context       int64 // only specified when this is a DAG execution
	ExecutorInput *pipelinespec.ExecutorInput
	Cached        bool // only specified when this is a Container execution
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
			Parameters: opts.RuntimeConfig.Parameters,
		},
	}
	// TODO(Bobgy): validate executorInput matches component spec types
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.IsRootDAG = true
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	exec, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", exec)
	// No need to return ExecutorInput, because tasks in the DAG will resolve
	// needed info from MLMD.
	return &Execution{ID: exec.GetID(), Context: pipeline.GetRunCtxID()}, nil
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
	if opts.DAGContextID != 0 {
		return fmt.Errorf("DAG context ID is unncessary")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unncessary")
	}
	return nil
}

// TODO(Bobgy): 7-17, continue to build CLI args for container driver
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
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID, opts.DAGContextID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	inputs, err := resolveInputs(ctx, dag, opts.Task, mlmd)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs:  inputs,
		Outputs: provisionOutputs(dag.GetPipelineRoot(), opts.Task.GetTaskInfo().GetName(), opts.Component.GetOutputDefinitions()),
	}
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName

	if opts.Task.GetCachingOptions() != nil && opts.Task.GetCachingOptions().GetEnableCache() {
		glog.Infof("Task {%s} enables cache", opts.Task.GetTaskInfo().GetName())
		fingerPrint, err := getFingerPrint(opts, executorInput)
		if err != nil {
			return nil, fmt.Errorf("failure while getting fingerPrint: %w", err)
		}
		cachedMLMDExecutionID, err := cacheClient.GetExecutionCache(fingerPrint, "pipeline/" + opts.PipelineName, opts.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failure while getting executionCache: %w", err)
		}
		ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
		ecfg.FingerPrint = fingerPrint
	}
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", createdExecution)

	if opts.Task.GetCachingOptions() != nil && opts.Task.GetCachingOptions().EnableCache && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, executorInput, opts.Component.GetOutputDefinitions(), mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return nil, err
		}
		outputParameters, err := metadata.NewParameters(executorOutput.GetParameters())
		if err != nil {
			return nil, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, outputParameters, outputArtifacts, pb.Execution_CACHED); err != nil {
			return nil, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		glog.Infof("Cached")
		return  &Execution{
			ID:            createdExecution.GetID(),
			ExecutorInput: executorInput,
			Cached: true,
		}, nil

	}
	return &Execution{
		ID:            createdExecution.GetID(),
		ExecutorInput: executorInput,
	}, nil
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
	cachedExecution := execution.GetExecution()
	executorOutput := &pipelinespec.ExecutorOutput{
		Parameters: map[string]*pipelinespec.Value{},
		Artifacts:  map[string]*pipelinespec.ArtifactList{},
	}
	if err := collectOutPutParametersFromCache(executorOutput, outputDefinitions, executorInput, cachedExecution); err != nil {
		return nil, nil, fmt.Errorf("failed to collect output parameters from cache: %w", err)
	}
	outputArtifacts, err := collectOutputArtifactMetadataFromCache(ctx, executorInput, cachedMLMDExecutionIDInt64, mlmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed collect output artifact metadata from cache: %w", err)
	}
	return executorOutput, outputArtifacts, nil

}

func collectOutPutParametersFromCache(executorOutput *pipelinespec.ExecutorOutput, outputDefinitions *pipelinespec.ComponentOutputsSpec, executorInput *pipelinespec.ExecutorInput, cachedExecution *pb.Execution, ) error {
	mlmdOutputParameters, err := cacheutils.GetMLMDOutputParams(cachedExecution)
	if err != nil {
		return err
	}
	outputParameters := executorOutput.GetParameters()
	for name, _ := range executorInput.GetOutputs().GetParameters() {
		paramSpec, ok := outputDefinitions.GetParameters()[name]
		if !ok {
			return fmt.Errorf("can't find parameter %v in outputDefinitions", name)
		}
		outputParamValue, ok := mlmdOutputParameters[name]
		if !ok {
			return fmt.Errorf("can't find parameter %v in mlmdOutputParameters", name)
		}
		switch paramSpec.GetType() {
		case pipelinespec.PrimitiveType_STRING:
			outputParameters[name] = metadata.StringValue(outputParamValue)
		case pipelinespec.PrimitiveType_INT:
			i, err := strconv.ParseInt(strings.TrimSpace(outputParamValue), 10, 0)
			if err != nil {
				return fmt.Errorf("failed to parse parameter name=%q value =%v to int: %w", name, outputParamValue, err)
			}
			outputParameters[name] = metadata.IntValue(i)
		case pipelinespec.PrimitiveType_DOUBLE:
			f, err := strconv.ParseFloat(strings.TrimSpace(outputParamValue), 0)
			if err != nil {
				return fmt.Errorf("failed to parse parameter name=%q value =%v to double: %w", name, outputParamValue, err)
			}
			outputParameters[name] = metadata.DoubleValue(f)
		default:
			return fmt.Errorf("unknown type. Expected STRING, INT or DOUBLE")
		}
	}
	return nil
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
	if opts.DAGContextID == 0 {
		return fmt.Errorf("DAG context ID is required")
	}
	if opts.Container == nil {
		return fmt.Errorf("container spec is required")
	}
	return nil
}

func resolveInputs(ctx context.Context, dag *metadata.DAG, task *pipelinespec.PipelineTaskSpec, mlmd *metadata.Client) (*pipelinespec.ExecutorInput_Inputs, error) {
	inputParams, _, err := dag.Execution.GetParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve inputs: %w", err)
	}
	glog.Infof("parent DAG input parameters %+v", inputParams)
	inputs := &pipelinespec.ExecutorInput_Inputs{
		Parameters: make(map[string]*pipelinespec.Value),
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
	}
	// get executions in context on demand
	var tasksCache map[string]*metadata.Execution
	getDAGTasks := func() (map[string]*metadata.Execution, error) {
		if tasksCache != nil {
			return tasksCache, nil
		}
		tasks, err := mlmd.GetExecutionsInDAG(ctx, dag)
		if err != nil {
			return nil, err
		}
		tasksCache = tasks
		return tasks, nil
	}
	for name, paramSpec := range task.GetInputs().GetParameters() {
		paramError := func(err error) error {
			return fmt.Errorf("failed to resolve input parameter %s with spec %s: %w", name, paramSpec, err)
		}
		if paramSpec.GetParameterExpressionSelector() != "" {
			return nil, paramError(fmt.Errorf("parameter expression selector not implemented yet"))
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
			inputs.Parameters[name] = v

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
			inputs.Parameters[name] = param
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
			runtimeValue := paramSpec.GetRuntimeValue()
			switch t := runtimeValue.Value.(type) {
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
		Artifacts: make(map[string]*pipelinespec.ArtifactList),
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
