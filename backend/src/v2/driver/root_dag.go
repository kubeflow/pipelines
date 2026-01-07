package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RootDAG handles initial root dag task creation
// and runtime parameter resolution.
func RootDAG(ctx context.Context, opts common.Options, clientManager client_manager.ClientManagerInterface) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.Info(), err)
		}
	}()

	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	glog.V(4).Info("RootDAG opts: ", string(b))
	if err = validateRootDAG(opts); err != nil {
		return nil, err
	}
	if clientManager == nil {
		return nil, fmt.Errorf("api client is nil")
	}

	// Build minimal PipelineTaskDetail for root DAG task under the run.
	// Inputs: pass runtime parameters into task inputs for record.
	var inputs *apiV2beta1.PipelineTaskDetail_InputOutputs
	if opts.RuntimeConfig != nil && opts.RuntimeConfig.GetParameterValues() != nil {
		params := make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(opts.RuntimeConfig.GetParameterValues()))
		for name, val := range opts.RuntimeConfig.GetParameterValues() {
			n := name
			params = append(params, &apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: n,
				Value:        val,
				Type:         apiV2beta1.IOType_RUNTIME_VALUE_INPUT,
				Producer: &apiV2beta1.IOProducer{
					TaskName: "ROOT",
				},
			})
		}
		inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{Parameters: params}
	}
	pd := &apiV2beta1.PipelineTaskDetail{
		Name:           "ROOT",
		DisplayName:    opts.RunDisplayName,
		RunId:          opts.Run.GetRunId(),
		Type:           apiV2beta1.PipelineTaskDetail_ROOT,
		Inputs:         inputs,
		TypeAttributes: &apiV2beta1.PipelineTaskDetail_TypeAttributes{},
		State:          apiV2beta1.PipelineTaskDetail_RUNNING,
		ScopePath:      opts.ScopePath.DotNotation(),
		CreateTime:     timestamppb.Now(),
		Pods: []*apiV2beta1.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: apiV2beta1.PipelineTaskDetail_DRIVER,
			},
		},
	}
	task, err := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{Task: pd})
	if err != nil {
		return nil, err
	}
	execution = &Execution{
		TaskID: task.TaskId,
	}
	return execution, nil
}
