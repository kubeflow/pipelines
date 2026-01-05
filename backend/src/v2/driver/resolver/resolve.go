// Copyright 2025 The Kubeflow Authors
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

package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

var paramError = func(paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec, err error) error {
	return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
}

var ErrResolvedParameterNull = errors.New("the resolved input is null")

type ParameterMetadata struct {
	// This is the key of the parameter in this task's inputs.
	Key                string
	ParameterIO        *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	InputParameterSpec *pipelinespec.TaskInputsSpec_InputParameterSpec
	ParameterIterator  *pipelinespec.ParameterIteratorSpec
}

type ArtifactMetadata struct {
	Key                   string
	DownloadedToWorkSpace bool
	// InputArtifactSpec is mutually exclusive with ArtifactIterator
	InputArtifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
	ArtifactIterator  *pipelinespec.ArtifactIteratorSpec
	ArtifactIO        *apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
}

// InputMetadata collects artifacts and parameters as arrays because
// the "key" is not unique in the case of Iterator parameters.
type InputMetadata struct {
	Parameters []ParameterMetadata
	Artifacts  []ArtifactMetadata
}

func ResolveInputs(ctx context.Context, opts common.Options) (*InputMetadata, *int, error) {
	inputMetadata := &InputMetadata{
		Parameters: []ParameterMetadata{},
		Artifacts:  []ArtifactMetadata{},
	}

	// Handle parameters
	resolvedParameters, err := resolveParameters(opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Parameters = resolvedParameters

	// Handle Artifacts
	resolvedArtifacts, err := resolveArtifacts(opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Artifacts = resolvedArtifacts

	// Note that we can only have one of the two.
	var iterationCount *int
	artifactIterator := opts.Task.GetArtifactIterator()
	parameterIterator := opts.Task.GetParameterIterator()
	switch {
	case parameterIterator != nil && artifactIterator != nil:
		return nil, nil, errors.New("cannot have both parameter and artifact iterators")
	case parameterIterator != nil:
		pm, count, err := resolveParameterIterator(opts, inputMetadata.Parameters)
		if err != nil {
			return nil, nil, err
		}
		if len(pm) == 0 {
			return nil, nil, fmt.Errorf("parameter iterator is empty")
		}
		iterationCount = count
		inputMetadata.Parameters = append(inputMetadata.Parameters, pm...)
	case artifactIterator != nil:
		am, count, err := resolveArtifactIterator(opts, inputMetadata.Artifacts)
		if err != nil {
			return nil, nil, err
		}
		if len(am) == 0 {
			return nil, nil, fmt.Errorf("artifact iterator is empty")
		}
		iterationCount = count
		inputMetadata.Artifacts = append(inputMetadata.Artifacts, am...)
	}
	return inputMetadata, iterationCount, nil
}
