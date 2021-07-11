package driver

import (
	"context"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
)

type Execution struct {
	ID            int64
	Context       int64 // only specified when this is a DAG execution
	ExecutorInput pipelinespec.ExecutorInput
}

func RootDAG(pipelineName string, runID string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, mlmd *metadata.Client) (*Execution, error) {
	ctx := context.Background()
	pipeline, err := mlmd.GetPipeline(ctx, pipelineName, runID)
	if err != nil {
		return nil, err
	}
	// TODO(Bobgy): resolve pipeline level inputs as ExecutorInput
	// ecfg := metadata.GenerateExecutionConfig(executorInput)
	ecfg := &metadata.ExecutionConfig{}
	execution, err := mlmd.CreateExecution(ctx, pipeline, "root", "", "", ecfg)
	if err != nil {
		return nil, err
	}
	return &Execution{ID: execution.GetID(), Context: pipeline.GetRunCtxID()}, nil
}
