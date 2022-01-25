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

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package metadata

import (
	"fmt"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

// A hacky way to get Execution from pb.Execution, usually you should get
// an Execution from this metadata package directly without using ml_metadata.Execution
func NewExecution(e *pb.Execution) *Execution {
	return &Execution{execution: e}
}

func (e *Execution) GetParameters() (inputs, outputs map[string]*structpb.Value, err error) {
	inputs = make(map[string]*structpb.Value)
	outputs = make(map[string]*structpb.Value)
	defer func() {
		if err != nil {
			err = fmt.Errorf("execution(ID=%v).GetParameters failed: %w", e.GetID(), err)
		}
	}()
	if e == nil || e.execution == nil {
		return nil, nil, nil
	}
	if stored_inputs, ok := e.execution.CustomProperties[keyInputs]; ok {
		for name, value := range stored_inputs.GetStructValue().GetFields() {
			inputs[name] = value
		}
	}
	if stored_outputs, ok := e.execution.CustomProperties[keyOutputs]; ok {
		for name, value := range stored_outputs.GetStructValue().GetFields() {
			outputs[name] = value
		}
	}
	return inputs, outputs, nil
}
