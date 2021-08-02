package driver

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
)

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
	// TODO(Bobgy): fill in namespace and run resource.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "namespace", "run-resource")
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
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.DAGExecutionID != 0 {
		return fmt.Errorf("DAG execution ID is unnecessary")
	}
	if opts.DAGContextID != 0 {
		return fmt.Errorf("DAG context ID is unncessary")
	}
	return nil
}

// TODO(Bobgy): 7-17, continue to build CLI args for container driver
func Container(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	err = validateContainer(opts)
	if err != nil {
		return nil, err
	}
	// TODO(Bobgy): fill in namespace and run resource
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "namespace", "runresource")
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
		Inputs: inputs,
	}
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	return &Execution{
		ID:            createdExecution.GetID(),
		ExecutorInput: executorInput,
	}, nil
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
	}
	if task.GetInputs() != nil {
		for name, paramSpec := range task.GetInputs().Parameters {
			paramError := func(err error) error {
				return fmt.Errorf("failed to resolve input parameter %s with spec %s: %w", name, paramSpec, err)
			}
			if paramSpec.GetParameterExpressionSelector() != "" {
				return nil, paramError(fmt.Errorf("parameter expression selector not implemented yet"))
			}
			componentInput := paramSpec.GetComponentInputParameter()
			if componentInput != "" {
				v, ok := inputParams[componentInput]
				if !ok {
					return nil, paramError(fmt.Errorf("parent DAG does not have input parameter %s", componentInput))
				}
				inputs.Parameters[name] = v
			} else {
				return nil, paramError(fmt.Errorf("parameter spec not implemented yet"))
			}
		}
		if len(task.GetInputs().GetArtifacts()) > 0 {
			return nil, fmt.Errorf("failed to resolve inputs: artifact inputs not implemented yet")
		}
	}
	return inputs, nil
}
