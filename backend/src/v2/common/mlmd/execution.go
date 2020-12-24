package mlmd

import (
	"github.com/kubeflow/pipelines/backend/src/v2/common"
	mlmdPb "github.com/kubeflow/pipelines/third_party/ml-metadata/go_client/ml_metadata/proto"
)

// TODO(Bobgy): refactor driver and publisher to all use this helper
type KfpExecution struct {
	execution *mlmdPb.Execution
}

func NewKfpExecution(execution *mlmdPb.Execution) *KfpExecution {
	return &KfpExecution{execution: execution}
}

func (e *KfpExecution) String() string {
	return e.execution.String()
}

func (e *KfpExecution) GetOutputParameter(parameterName string) *mlmdPb.Value {
	return e.execution.GetCustomProperties()[common.ExecutionPropertyPrefixOutputParam+parameterName]
}
