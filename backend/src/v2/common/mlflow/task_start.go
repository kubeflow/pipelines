package mlflow

import (
	"context"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type RequestContext = commonmlflow.RequestContext

func CreateNestedRun(ctx context.Context, requestCtx *RequestContext, taskName string, experimentID string, parentRunID string) (string, error) {
	if experimentID == "" {
		return "", util.NewInvalidInputError("MLflow experiment_id is required to create run")
	}
	if requestCtx == nil || requestCtx.Client == nil {
		return "", util.NewInvalidInputError("MLflow request context is required")
	}
	if parentRunID == "" {
		return "", util.NewInvalidInputError("MLflow parent_run_id is required to create run")
	}
	parentRunTag := commonmlflow.Tag{Key: commonmlflow.ParentRunTagKey, Value: parentRunID}
	return requestCtx.Client.CreateRun(ctx, experimentID, taskName, []commonmlflow.Tag{parentRunTag})
}

func UpdateRun(ctx context.Context, requestCtx *RequestContext, runID string, status string, endTime int64) error {
	if runID == "" {
		return util.NewInvalidInputError("MLflow run_id is required to update run")
	}
	if requestCtx == nil || requestCtx.Client == nil {
		return util.NewInvalidInputError("MLflow request context is required")
	}
	return requestCtx.Client.UpdateRun(ctx, runID, ExecutionStateToMLflowTerminalStatus(status), &endTime)
}

func LogBatch(ctx context.Context, requestCtx *RequestContext, runID string, metrics []commonmlflow.Metric, params []commonmlflow.Param, tags []commonmlflow.Tag) error {
	if runID == "" {
		return util.NewInvalidInputError("MLflow run_id is required to log batch")
	}
	if metrics == nil && params == nil && tags == nil {
		return util.NewInvalidInputError("at least one of metrics, params, or tags must be provided")
	}
	req := commonmlflow.LogBatchRequest{RunID: runID, Metrics: metrics, Params: params, Tags: tags}
	return requestCtx.Client.LogBatch(ctx, req)
}
