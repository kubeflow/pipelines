// Copyright 2018-2022 The Kubeflow Authors
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

package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"
)

func getOwningExperimentUUID(references []*apiv1beta1.ResourceReference) (string, error) {
	var experimentUUID string
	for _, ref := range references {
		if ref.Key.Type == apiv1beta1.ResourceType_EXPERIMENT && ref.Relationship == apiv1beta1.Relationship_OWNER {
			experimentUUID = ref.Key.Id
			break
		}
	}

	if experimentUUID == "" {
		return "", util.NewInternalServerError(nil, "Missing owning experiment UUID")
	}
	return experimentUUID, nil
}

// Fetches ResourceReferences from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetResourceReferenceFromRunInterface(r interface{}) ([]*apiv1beta1.ResourceReference, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return r.(*apiv1beta1.Run).GetResourceReferences(), nil
	case *apiv2beta1.Run:
		return nil, nil
	default:
		return nil, util.NewUnknownApiVersionError("GetResourceReferenceFromRunInterface()", fmt.Sprintf("ResourceReference from %T", reflect.TypeOf(r)))
	}
}

// Fetches a PipelineId from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetPipelineIdFromRunInterface(r interface{}) (string, error) {
	switch runType := r.(type) {
	case *apiv1beta1.Run:
		return r.(*apiv1beta1.Run).GetPipelineSpec().GetPipelineId(), nil
	case *apiv2beta1.Run:
		pipelineId := r.(*apiv2beta1.Run).GetPipelineId()
		if pipelineId == "" {
			if pipelineIdValue, ok := r.(*apiv2beta1.Run).GetPipelineSpec().GetFields()["PipelineId"]; ok {
				pipelineId = pipelineIdValue.GetStringValue()
			} else {
				return "", util.NewResourceNotFoundError(fmt.Sprintf("PipelineId not found in %T. This could be because PipelineId is set to an empty string or invalid PipelineId field in PipelineSpec", runType), "")
			}
		}
		return pipelineId, nil
	default:
		return "", util.NewUnknownApiVersionError("GetPipelineIdFromRunInterface()", fmt.Sprintf("PipelineId from %T", reflect.TypeOf(r)))
	}
}

// Fetches a ExperimentId from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetExperimentIdFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return "", nil
	case *apiv2beta1.Run:
		return r.(*apiv2beta1.Run).GetExperimentId(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetExperimentIdFromRunInterface()", fmt.Sprintf("ExperimentId from %T", reflect.TypeOf(r)))
	}
}

// Fetches a RuntimeState from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetStateFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return "", nil
	case *apiv2beta1.Run:
		return r.(*apiv2beta1.Run).GetState().String(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetStateFromRunInterface()", fmt.Sprintf("State from %T", reflect.TypeOf(r)))
	}
}

// Fetches a RuntimeStateHistory from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetStateHistoryFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return "", nil
	case *apiv2beta1.Run:
		if serializedState, err := json.Marshal(r.(*apiv2beta1.Run).GetStateHistory()); err != nil {
			return "", err
		} else {
			return string(serializedState), err
		}
	default:
		return "", util.NewUnknownApiVersionError("GetStateHistoryFromRunInterface()", fmt.Sprintf("StateHistory from %T", reflect.TypeOf(r)))
	}
}

// Fetches a DisplayName from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetDisplayNameFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return r.(*apiv1beta1.Run).GetName(), nil
	case *apiv2beta1.Run:
		return r.(*apiv2beta1.Run).GetDisplayName(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetDisplayNameFromRunInterface()", fmt.Sprintf("DisplayName from %T", reflect.TypeOf(r)))
	}
}

// Converts structpb.Struct into a yaml string
func ProtobufStructToYamlString(s *structpb.Struct) (string, error) {
	bytes, err := yaml.Marshal(s.AsMap())
	// bytes, err := json.Marshal(s.AsMap())
	if err != nil {
		return "", util.Wrap(err, "Failed to convert a protobuf struct into a yaml string.")
	}
	return string(bytes), nil
}

// Converts a yaml string into structpb.Struct (v2)
func YamlStringToProtobufStruct(s string) (*structpb.Struct, error) {
	var m *map[string]interface{}
	// var m *structpb.Struct
	err := yaml.Unmarshal([]byte(s), m)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert a yaml string into a map[string]interface{}.")
	}
	st, err := structpb.NewStruct(*m)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert a yaml string into a protobuf struct.")
	}
	return st, nil
}

// Fetches a PipelineRoot from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetPipelineRootFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return r.(*apiv1beta1.Run).GetPipelineSpec().GetRuntimeConfig().GetPipelineRoot(), nil
	case *apiv2beta1.Run:
		return r.(*apiv2beta1.Run).GetRuntimeConfig().GetPipelineRoot(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetPipelineRootFromRunInterface()", fmt.Sprintf("PipelineRoot from %T", reflect.TypeOf(r)))
	}
}

// Fetches a RuntimeConfig from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetRuntimeConfigFromRunInterface(r interface{}) (map[string]interface{}, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		// Fetch from parameters in V1 template
		var newParameters []map[string]*structpb.Value
		oldParameters := r.(*apiv1beta1.Run).GetPipelineSpec().GetParameters()
		if oldParameters == nil || len(oldParameters) == 0 {
			oldParams := r.(*apiv1beta1.Run).GetPipelineSpec().GetRuntimeConfig().GetParameters()
			for n, v := range oldParams {
				newParameters = append(
					newParameters,
					map[string]*structpb.Value{
						"Name":  structpb.NewStringValue(n),
						"Value": v,
					},
				)
			}
		} else {
			for i := range oldParameters {
				newParameters[i] = map[string]*structpb.Value{
					"Name":  structpb.NewStringValue(oldParameters[i].GetName()),
					"Value": structpb.NewStringValue(oldParameters[i].GetValue()),
				}
			}
		}
		// Convert RuntimeConfig
		newRuntimeConfig := map[string]interface{}{
			"Parameters":   newParameters,
			"PipelineRoot": r.(*apiv1beta1.Run).GetPipelineSpec().GetRuntimeConfig().GetPipelineRoot(),
		}
		return newRuntimeConfig, nil
	case *apiv2beta1.Run:
		newRuntimeConfig := map[string]interface{}{
			"Parameters":   r.(*apiv2beta1.Run).GetRuntimeConfig().GetParameters(),
			"PipelineRoot": r.(*apiv2beta1.Run).GetRuntimeConfig().GetPipelineRoot(),
		}
		return newRuntimeConfig, nil
	default:
		return nil, util.NewUnknownApiVersionError("GetRuntimeConfigFromRunInterface()", fmt.Sprintf("RuntimeConfig from %T", reflect.TypeOf(r)))
	}
}

// Fetches a RunDetails from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetRunDetailsFromRunInterface(r interface{}) (string, error) {
	switch r.(type) {
	case *apiv1beta1.Run:
		return "", nil
	case *apiv2beta1.Run:
		return r.(*apiv2beta1.Run).GetRunDetails().String(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetRunDetailsFromRunInterface()", fmt.Sprintf("RunDetails from %T", reflect.TypeOf(r)))
	}
}

func getPipelineVersionIdFromResourceReferences(resourceManager *resource.ResourceManager, resourceReferences []*apiv1beta1.ResourceReference) string {
	var pipelineVersionId = ""
	for _, resourceReference := range resourceReferences {
		if resourceReference.Key.Type == apiv1beta1.ResourceType_PIPELINE_VERSION && resourceReference.Relationship == apiv1beta1.Relationship_CREATOR {
			pipelineVersionId = resourceReference.Key.Id
		}
	}
	return pipelineVersionId
}

// Verify the input resource references has one and only reference which is owner experiment.
func ValidateExperimentResourceReference(resourceManager *resource.ResourceManager, references []*apiv1beta1.ResourceReference) error {
	if references == nil || len(references) == 0 || references[0] == nil {
		return util.NewInvalidInputError("The resource reference is empty. Please specify which experiment owns this resource.")
	}
	if len(references) > 1 {
		return util.NewInvalidInputError("Got more resource references than expected. Please only specify which experiment owns this resource.")
	}
	if references[0].Key.Type != apiv1beta1.ResourceType_EXPERIMENT {
		return util.NewInvalidInputError("Unexpected resource type. Expected:%v. Got: %v",
			apiv1beta1.ResourceType_EXPERIMENT, references[0].Key.Type)
	}
	if references[0].Key.Id == "" {
		return util.NewInvalidInputError("Resource ID is empty. Please specify a valid ID")
	}
	if references[0].Relationship != apiv1beta1.Relationship_OWNER {
		return util.NewInvalidInputError("Unexpected relationship for the experiment. Expected: %v. Got: %v",
			apiv1beta1.Relationship_OWNER, references[0].Relationship)
	}
	if _, err := resourceManager.GetExperiment(references[0].Key.Id); err != nil {
		return util.Wrap(err, "Failed to get experiment.")
	}
	return nil
}

func ValidatePipelineSpecAndResourceReferences(resourceManager *resource.ResourceManager, spec *apiv1beta1.PipelineSpec, resourceReferences []*apiv1beta1.ResourceReference) error {
	pipelineId := spec.GetPipelineId()
	workflowManifest := spec.GetWorkflowManifest()
	pipelineManifest := spec.GetPipelineManifest()
	pipelineVersionId := getPipelineVersionIdFromResourceReferences(resourceManager, resourceReferences)

	if workflowManifest != "" || pipelineManifest != "" {
		if workflowManifest != "" && pipelineManifest != "" {
			return util.NewInvalidInputError("Please don't specify both workflow manifest and pipeline manifest.")
		}
		if pipelineId != "" || pipelineVersionId != "" {
			return util.NewInvalidInputError("Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
		}
		if err := validateWorkflowManifest(workflowManifest); err != nil {
			return err
		}
		if err := validatePipelineManifest(pipelineManifest); err != nil {
			return err
		}
	} else {
		if pipelineId == "" && pipelineVersionId == "" {
			return util.NewInvalidInputError("Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
		}
		if err := validatePipelineId(resourceManager, pipelineId); err != nil {
			return err
		}
		if pipelineVersionId != "" {
			// verify pipelineVersionId exists
			pipelineVersion, err := resourceManager.GetPipelineVersion(pipelineVersionId)
			if err != nil {
				return util.Wrap(err, "Get pipelineVersionId failed.")
			}
			// verify pipelineId should be parent of pipelineVersionId
			if pipelineId != "" && pipelineVersion.PipelineId != pipelineId {
				return util.NewInvalidInputError("pipeline ID should be parent of pipeline version.")
			}
		}
	}
	if spec.GetParameters() != nil && spec.GetRuntimeConfig() != nil {
		return util.NewInvalidInputError("Please don't specify both parameters and runtime config.")
	}
	if err := validateParameters(spec.GetParameters()); err != nil {
		return err
	}
	if err := validateRuntimeConfig(spec.GetRuntimeConfig()); err != nil {
		return err
	}
	return nil
}

func validateParameters(parameters []*apiv1beta1.Parameter) error {
	if parameters != nil {
		paramsBytes, err := json.Marshal(parameters)
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to Marshall the pipeline parameters into bytes. Parameters: %s",
				printParameters(parameters))
		}
		if len(paramsBytes) > util.MaxParameterBytes {
			return util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
		}
	}
	return nil
}

func validateRuntimeConfig(runtimeConfig *apiv1beta1.PipelineSpec_RuntimeConfig) error {
	if runtimeConfig.GetParameters() != nil {
		paramsBytes, err := json.Marshal(runtimeConfig.GetParameters())
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to Marshall the runtime config parameters into bytes.")
		}
		if len(paramsBytes) > util.MaxParameterBytes {
			return util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
		}
	}
	return nil
}

func validatePipelineId(resourceManager *resource.ResourceManager, pipelineId string) error {
	if pipelineId != "" {
		// Verify pipeline exist
		if _, err := resourceManager.GetPipeline(pipelineId); err != nil {
			return util.Wrap(err, "Get pipelineId failed.")
		}
	}
	return nil
}

func validateWorkflowManifest(workflowManifest string) error {
	if workflowManifest != "" {
		// Verify valid workflow template
		var workflow util.Workflow
		if err := json.Unmarshal([]byte(workflowManifest), &workflow); err != nil {
			return util.NewInvalidInputErrorWithDetails(err,
				"Invalid argo workflow format. Workflow: "+workflowManifest)
		}
	}
	return nil
}

func validatePipelineManifest(pipelineManifest string) error {
	if pipelineManifest != "" {
		// Verify valid IR spec
		spec := &pipelinespec.PipelineSpec{}
		if err := yaml.Unmarshal([]byte(pipelineManifest), spec); err != nil {
			return util.NewInvalidInputErrorWithDetails(err,
				"Invalid IR spec format.")
		}
	}
	return nil
}

func printParameters(params []*apiv1beta1.Parameter) string {
	var s strings.Builder
	for _, p := range params {
		s.WriteString(p.String())
	}
	return s.String()
}

// Returns workflow template []byte array from PipelineSpec
func getWorkflowSpecBytesFromPipelineSpec(spec *apiv1beta1.PipelineSpec) ([]byte, error) {
	if spec.GetWorkflowManifest() != "" {
		return []byte(spec.GetWorkflowManifest()), nil
	}
	return nil, util.NewInvalidInputError("Please provide a valid pipeline spec")
}
