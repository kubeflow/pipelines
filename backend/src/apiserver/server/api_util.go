// Copyright 2018-2023 The Kubeflow Authors
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
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/ghodss/yaml"
)

const (
	// This regex expresses the following constraints:
	// * Allows lowercase/uppercase letters
	// * Allows "_", "-" and numbers in the middle
	// * Additionally, numbers are also allowed at the end
	// * At most 64 characters
	metricNamePattern = "^[a-zA-Z]([-_a-zA-Z0-9]{0,62}[a-zA-Z0-9])?$"
)

// Returns namespace inferred from v1beta1 API resource references.
// Supports v1beta1 API.
func getNamespaceFromResourceReferenceV1(resourceRefs []*apiv1beta1.ResourceReference) string {
	namespace := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.GetKey() != nil {
			if resourceRef.GetKey().GetType() == apiv1beta1.ResourceType_NAMESPACE {
				namespace = resourceRef.GetKey().GetId()
				break
			}
		}
	}
	return namespace
}

// Returns experiment id inferred from v1beta1 API resource references.
// Supports v1beta1 API.
func getExperimentIdFromResourceReferencesV1(resourceRefs []*apiv1beta1.ResourceReference) string {
	experimentId := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.GetKey() != nil {
			if resourceRef.GetKey().GetType() == apiv1beta1.ResourceType_EXPERIMENT {
				experimentId = resourceRef.GetKey().GetId()
				break
			}
		}
	}
	return experimentId
}

// Returns pipeline id inferred from v1beta1 API resource references.
// Supports v1beta1 API.
func getPipelineIdFromResourceReferencesV1(resourceRefs []*apiv1beta1.ResourceReference) string {
	pipelineId := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.GetKey() != nil {
			if resourceRef.GetKey().GetType() == apiv1beta1.ResourceType_PIPELINE {
				pipelineId = resourceRef.GetKey().GetId()
				break
			}
		}
	}
	return pipelineId
}

// Returns pipeline version id inferred from v1beta1 API resource references.
// Support v1beta1 API.
func getPipelineVersionFromResourceReferencesV1(resourceReferences []*apiv1beta1.ResourceReference) string {
	var pipelineVersionId = ""
	for _, resourceReference := range resourceReferences {
		if resourceReference.GetKey() != nil {
			if resourceReference.GetKey().GetType() == apiv1beta1.ResourceType_PIPELINE_VERSION && resourceReference.GetRelationship() == apiv1beta1.Relationship_CREATOR {
				pipelineVersionId = resourceReference.GetKey().GetId()
			}
		}
	}
	return pipelineVersionId
}

// Returns id of the recurring run inferred from v1beta1 API resource references.
// Supports v1beta1 API.
func getJobIdFromResourceReferencesV1(resourceRefs []*apiv1beta1.ResourceReference) string {
	jobId := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.GetKey() != nil {
			if resourceRef.GetKey().GetType() == apiv1beta1.ResourceType_JOB {
				jobId = resourceRef.GetKey().GetId()
				break
			}
		}
	}
	return jobId
}

// Converts structpb.Struct to a yaml string
func protobufStructToYamlString(s *structpb.Struct) (string, error) {
	bytes, err := yaml.Marshal(s)
	// bytes, err := json.Marshal(s.AsMap())
	if err != nil {
		return "", util.Wrap(err, "Failed to convert a protobuf struct to a yaml string")
	}
	return string(bytes), nil
}

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
		return nil, util.NewUnknownApiVersionError("GetResourceReferenceFromRunInterface()", r)
	}
}

// Converts a yaml string into structpb.Struct (v2)
func yamlStringToProtobufStruct(s string) (*structpb.Struct, error) {
	st := &structpb.Struct{}
	err := yaml.Unmarshal([]byte(s), st)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert a yaml string into a protobuf struct")
	}
	return st, nil
	// j, err := yaml.YAMLToJSON([]byte(s))
	// if err != nil {
	// 	return nil, util.Wrap(err, "Failed to convert a yaml string to json string")
	// }
	// pipelineSpec := &structpb.Struct{}
	// err = pipelineSpec.UnmarshalJSON(j)
	// if err != nil {
	// 	return nil, util.Wrap(err, "Failed to convert a json string into a protobuf struct")
	// }
	// return pipelineSpec, nil
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
		return "", util.NewUnknownApiVersionError("GetPipelineRootFromRunInterface()", r)
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
		return nil, util.NewUnknownApiVersionError("GetRuntimeConfigFromRunInterface()", r)
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
		return "", util.NewUnknownApiVersionError("GetRunDetailsFromRunInterface()", r)
	}
}

// Verify the input resource references has one and only reference which is owner experiment.
func ValidateExperimentResourceReference(resourceManager *resource.ResourceManager, references []*apiv1beta1.ResourceReference) error {
	if references == nil || len(references) == 0 || references[0] == nil {
		return util.NewInvalidInputError("The resource reference is empty. Please specify which experiment owns this resource")
	}
	if len(references) > 1 {
		return util.NewInvalidInputError("Got more resource references than expected. Please only specify which experiment owns this resource")
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
		return util.Wrap(err, "Failed to get experiment")
	}
	return nil
}

func ValidatePipelineSpecAndResourceReferences(resourceManager *resource.ResourceManager, spec *apiv1beta1.PipelineSpec, resourceReferences []*apiv1beta1.ResourceReference) error {
	pipelineId := spec.GetPipelineId()
	workflowManifest := spec.GetWorkflowManifest()
	pipelineManifest := spec.GetPipelineManifest()
	pipelineVersionId := getPipelineVersionFromResourceReferencesV1(resourceReferences)

	if workflowManifest != "" || pipelineManifest != "" {
		if workflowManifest != "" && pipelineManifest != "" {
			return util.NewInvalidInputError("Please don't specify both workflow manifest and pipeline manifest")
		}
		if pipelineId != "" || pipelineVersionId != "" {
			return util.NewInvalidInputError("Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest")
		}
		if err := validateWorkflowManifest(workflowManifest); err != nil {
			return err
		}
		if err := validatePipelineManifest(pipelineManifest); err != nil {
			return err
		}
	} else {
		if pipelineId == "" && pipelineVersionId == "" {
			return util.NewInvalidInputError("Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version)")
		}
		if err := validatePipelineId(resourceManager, pipelineId); err != nil {
			return err
		}
		if pipelineVersionId != "" {
			// verify pipelineVersionId exists
			pipelineVersion, err := resourceManager.GetPipelineVersion(pipelineVersionId)
			if err != nil {
				return util.Wrap(err, "Get pipelineVersionId failed")
			}
			// verify pipelineId should be parent of pipelineVersionId
			if pipelineId != "" && pipelineVersion.PipelineId != pipelineId {
				return util.NewInvalidInputError("pipeline ID should be parent of pipeline version")
			}
		}
	}
	if spec.GetParameters() != nil && spec.GetRuntimeConfig() != nil {
		return util.NewInvalidInputError("Please don't specify both parameters and runtime config")
	}
	if err := validateParameters(spec.GetParameters()); err != nil {
		return err
	}
	if err := validateRuntimeConfigV1(spec.GetRuntimeConfig()); err != nil {
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
				apiParametersToStringV1(parameters))
		}
		if len(paramsBytes) > util.MaxParameterBytes {
			return util.NewInvalidInputError("The input parameter length exceed maximum size of %v", util.MaxParameterBytes)
		}
	}
	return nil
}

func validateRuntimeConfigV1(runtimeConfig *apiv1beta1.PipelineSpec_RuntimeConfig) error {
	if runtimeConfig.GetParameters() != nil {
		paramsBytes, err := json.Marshal(runtimeConfig.GetParameters())
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to Marshall the runtime config parameters into bytes")
		}
		if len(paramsBytes) > util.MaxParameterBytes {
			return util.NewInvalidInputError("The input parameter length exceed maximum size of %v", util.MaxParameterBytes)
		}
	}
	return nil
}

func validatePipelineId(resourceManager *resource.ResourceManager, pipelineId string) error {
	if pipelineId != "" {
		// Verify pipeline exist
		if _, err := resourceManager.GetPipeline(pipelineId); err != nil {
			return util.Wrap(err, "Get pipelineId failed")
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
				"Invalid IR spec format")
		}
	}
	return nil
}

func apiParametersToStringV1(params []*apiv1beta1.Parameter) string {
	var s strings.Builder
	for _, p := range params {
		s.WriteString(p.String())
	}
	return s.String()
}

func getWorkflowSpecBytesFromPipelineSpecV1(spec *apiv1beta1.PipelineSpec) ([]byte, error) {
	if spec.GetWorkflowManifest() != "" {
		return []byte(spec.GetWorkflowManifest()), nil
	}
	return nil, util.NewInvalidInputError("Please provide a valid pipeline spec")
}

// Validates a run metric fields from request.
func validateRunMetric(metric *model.RunMetric) error {
	matched, err := regexp.MatchString(metricNamePattern, metric.Name)
	if err != nil {
		// This should never happen.
		return util.NewInternalServerError(
			err, "failed to compile pattern '%s'", metricNamePattern)
	}
	if !matched {
		return util.NewInvalidInputError(
			"metric.name '%s' doesn't match with the pattern '%s'", metric.Name, metricNamePattern)
	}
	if metric.NodeID == "" {
		return util.NewInvalidInputError("metric.node_id must not be empty")
	}
	if len(metric.NodeID) > 128 {
		return util.NewInvalidInputError(
			"metric.node_id '%s' cannot be longer than 128 characters", metric.NodeID)
	}
	return nil
}
