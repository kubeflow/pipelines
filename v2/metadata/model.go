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
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
)

const (
	prefixInput  = "input:"
	prefixOutput = "output:"
)

func (e *Execution) GetParameters() (inputs, outputs map[string]*pipelinespec.Value, err error) {
	inputs = make(map[string]*pipelinespec.Value)
	outputs = make(map[string]*pipelinespec.Value)
	defer func() {
		if err != nil {
			err = fmt.Errorf("execution(ID=%v).GetParameters failed: %w", e.GetID(), err)
		}
	}()
	if e == nil || e.execution == nil {
		return nil, nil, nil
	}
	for key, value := range e.execution.CustomProperties {
		if strings.HasPrefix(key, prefixInput) {
			name := strings.TrimPrefix(key, prefixInput)
			kfpValue, err := mlmdValueToPipelineSpecValue(value)
			if err != nil {
				return nil, nil, err
			}
			inputs[name] = kfpValue
		} else if strings.HasPrefix(key, prefixOutput) {
			name := strings.TrimPrefix(key, prefixOutput)
			kfpValue, err := mlmdValueToPipelineSpecValue(value)
			if err != nil {
				return nil, nil, err
			}
			outputs[name] = kfpValue
		}
	}
	return inputs, outputs, nil
}
