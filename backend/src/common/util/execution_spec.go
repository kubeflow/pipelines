// Copyright 2022 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"errors"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExecutionType string

const (
	ArgoWorkflow      ExecutionType = "Workflow"
	TektonPipelineRun ExecutionType = "PipelineRun"
	Unknown           ExecutionType = "Unknown"
)

// Abastract type for the resource needed by the underlying execution runtime
// i.e Workflow is for Argo, PipelineRun is for Tekton and etc.
// TODO: add more functions to make ExecutionSpec full represent Workflow
type ExecutionSpec interface {
	// ExecutionType
	ExecutionType() ExecutionType

	// SetServiceAccount Set the service account to run the ExecutionSpec.
	SetServiceAccount(serviceAccount string)

	// OverrideParameters overrides some of the parameters.
	OverrideParameters(desiredParams map[string]string)

	// SetAnnotationsToAllTemplatesIfKeyNotExist sets annotations on all templates in a Workflow
	// if the annotation key does not exist
	SetAnnotationsToAllTemplatesIfKeyNotExist(key string, value string)

	SetLabels(key string, value string)
	SetAnnotations(key string, value string)

	ReplaceUID(id string) error
	SetPodMetadataLabels(key string, value string)

	// Get ServiceAccountName
	ServiceAccount() string

	// FindObjectStoreArtifactKeyOrEmpty loops through all node running statuses and look up the first
	// S3 artifact with the specified nodeID and artifactName. Returns empty if nothing is found.
	// TODO: move to ExecutionStatus
	FindObjectStoreArtifactKeyOrEmpty(nodeID string, artifactName string) string

	// TODO: move to ExecutionStatus
	Condition() string
}

// Convert YAML in bytes into ExecutionSpec instance
func NewExecutionSpec(bytes []byte) (ExecutionSpec, error) {
	if len(bytes) == 0 {
		return nil, NewInvalidInputError("empty input")
	}
	var meta metav1.TypeMeta
	err := yaml.Unmarshal(bytes, &meta)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}

	switch meta.Kind {
	case string(ArgoWorkflow):
		return NewWorkflowFromBytes(bytes)
	case string(TektonPipelineRun):
		return nil, NewInvalidInputError("Not implemented yet")
	default:
		return nil, NewInvalidInputError("Unknown execution spec")
	}
}

// Construct a ExecutionSpec based on the data struct. Use this to
// leverage the existing Workflow creation for Argo. Need to support
// other runtime when implementation is added.
func NewExecutionSpecFromInterface(execType ExecutionType, obj interface{}) (ExecutionSpec, error) {
	// try and error, try Workflow first, add other type later
	switch execType {
	case ArgoWorkflow:
		return NewWorkflowFromInterface(obj)
	default:
		return nil, NewInternalServerError(
			errors.New("ExecutionType is not supported"), "type:%s", execType)
	}
}
