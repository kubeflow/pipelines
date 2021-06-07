// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
