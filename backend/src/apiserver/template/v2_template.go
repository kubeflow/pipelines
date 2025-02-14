// Copyright 2021-2023 The Kubeflow Authors
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

package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/tektoncompiler"
	"google.golang.org/protobuf/encoding/protojson"
	goyaml "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type V2Spec struct {
	spec         *pipelinespec.PipelineSpec
	platformSpec *pipelinespec.PlatformSpec
}

var (
	Launcher = ""
)

// Converts modelJob to ScheduledWorkflow.
func (t *V2Spec) ScheduledWorkflow(modelJob *model.Job, ownerReferences []metav1.OwnerReference) (*scheduledworkflow.ScheduledWorkflow, error) {
	job := &pipelinespec.PipelineJob{}

	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to parse pipeline spec")
	}

	job.PipelineSpec = spec

	jobRuntimeConfig, err := modelToPipelineJobRuntimeConfig(&modelJob.RuntimeConfig)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert runtime config")
	}
	job.RuntimeConfig = jobRuntimeConfig
	if err = t.validatePipelineJobInputs(job); err != nil {
		return nil, util.Wrap(err, "invalid pipeline job inputs")
	}

	// Pick out Kubernetes platform configs
	var kubernetesSpec *pipelinespec.SinglePlatformSpec
	if t.platformSpec != nil {
		if _, ok := t.platformSpec.Platforms["kubernetes"]; ok {
			kubernetesSpec = t.platformSpec.Platforms["kubernetes"]
		}
	}

	var obj interface{}
	if util.CurrentExecutionType() == util.ArgoWorkflow {
		obj, err = argocompiler.Compile(job, kubernetesSpec, nil)
	} else if util.CurrentExecutionType() == util.TektonPipelineRun {
		obj, err = tektoncompiler.Compile(job, kubernetesSpec, &tektoncompiler.Options{LauncherImage: Launcher})
	}
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	// currently, there is only Argo implementation, so it's using `ArgoWorkflow` for now
	// later on, if a new runtime support will be added, we need a way to switch/specify
	// runtime. i.e using ENV var
	executionSpec, err := util.NewExecutionSpecFromInterface(util.CurrentExecutionType(), obj)
	if err != nil {
		return nil, util.NewInternalServerError(err, "error creating execution spec")
	}
	// Overwrite namespace from the job object
	if modelJob.Namespace != "" {
		executionSpec.SetExecutionNamespace(modelJob.Namespace)
	}
	if executionSpec.ServiceAccount() == "" {
		setDefaultServiceAccount(executionSpec, modelJob.ServiceAccount)
	}
	// Disable istio sidecar injection if not specified
	executionSpec.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(modelJob.K8SName)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	parameters, err := stringMapToCRDParameters(modelJob.RuntimeConfig.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Converting runtime config's parameters to CDR parameters failed")
	}
	crdTrigger, err := modelToCRDTrigger(modelJob.Trigger)
	if err != nil {
		return nil, util.Wrap(err, "converting model trigger to crd trigger failed")
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    swfGeneratedName,
			OwnerReferences: ownerReferences,
		},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        modelJob.Enabled,
			MaxConcurrency: &modelJob.MaxConcurrency,
			Trigger:        crdTrigger,
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: parameters,
				Spec:       executionSpec.ToStringForSchedule(),
			},
			NoCatchup:         util.BoolPointer(modelJob.NoCatchup),
			ExperimentId:      modelJob.ExperimentId,
			PipelineId:        modelJob.PipelineId,
			PipelineName:      modelJob.PipelineName,
			PipelineVersionId: modelJob.PipelineVersionId,
			ServiceAccount:    executionSpec.ServiceAccount(),
		},
	}
	return scheduledWorkflow, nil
}

func (t *V2Spec) GetTemplateType() TemplateType {
	return V2
}

func NewV2SpecTemplate(template []byte) (*V2Spec, error) {
	var v2Spec V2Spec
	decoder := goyaml.NewDecoder(bytes.NewReader(template))
	for {
		var value map[string]interface{}

		err := decoder.Decode(&value)
		// Break at end of file
		if errors.Is(err, io.EOF) {
			break
		}
		if value == nil {
			continue
		}
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("unable to decode yaml document: %s", err.Error()))
		}
		valueBytes, err := goyaml.Marshal(&value)
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("unable to marshal this yaml document: %s", err.Error()))
		}
		if isPipelineSpec(value) {
			// Pick out the yaml document with pipeline spec
			if v2Spec.spec != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "multiple pipeline specs provided")
			}
			jsonData, err := json.Marshal(value)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("cannot convert v2 pipeline spec to json format: %s", err.Error()))
			}
			var spec pipelinespec.PipelineSpec
			err = protojson.Unmarshal(jsonData, &spec)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("invalid v2 pipeline spec: %s", err.Error()))
			}
			if spec.GetSchemaVersion() != SCHEMA_VERSION_2_1_0 {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("KFP only supports schema version 2.1.0, but the pipeline spec has version %s", spec.GetSchemaVersion()))
			}
			if spec.GetPipelineInfo().GetName() == "" {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: name is empty")
			}
			match, _ := regexp.MatchString("[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*", spec.GetPipelineInfo().GetName())
			if !match {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: name should consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character")
			}
			if spec.GetRoot() == nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: root component is empty")
			}
			v2Spec.spec = &spec
		} else if IsPlatformSpecWithKubernetesConfig(valueBytes) {
			// Pick out the yaml document with platform spec
			if v2Spec.platformSpec != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPlatformSpec, "multiple platform specs provided")
			}
			var platformSpec pipelinespec.PlatformSpec
			platformSpecJson, err := yaml.YAMLToJSON(valueBytes)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPlatformSpec, fmt.Sprintf("cannot convert platform specific configs to json format: %s", err.Error()))
			}
			err = protojson.Unmarshal(platformSpecJson, &platformSpec)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPlatformSpec, fmt.Sprintf("cannot unmarshal platform specific configs: %s", err.Error()))
			}
			v2Spec.platformSpec = &platformSpec
		}
	}
	if v2Spec.spec == nil {
		return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "no pipeline spec is provided")
	}
	return &v2Spec, nil
}

func (t *V2Spec) Bytes() []byte {
	if t == nil {
		return nil
	}
	bytesSpec, err := protojson.Marshal(t.spec)
	if err != nil {
		// this is unexpected, cannot convert proto message to JSON
		return nil
	}
	bytes, err := yaml.JSONToYAML(bytesSpec)
	if err != nil {
		// this is unexpected, cannot convert JSON to YAML
		return nil
	}
	if t.platformSpec != nil {
		bytesExecutorConfig, err := protojson.Marshal(t.platformSpec)
		if err != nil {
			// this is unexpected, cannot convert proto message to JSON
			return nil
		}
		bytesExecutorConfigYaml, err := yaml.JSONToYAML(bytesExecutorConfig)
		if err != nil {
			// this is unexpected, cannot convert JSON to YAML
			return nil
		}
		bytes = append(bytes, []byte("\n---\n")...)
		bytes = append(bytes, bytesExecutorConfigYaml...)
	}
	return bytes
}

func (t *V2Spec) IsV2() bool {
	return true
}

func (t *V2Spec) V2PipelineName() string {
	if t == nil {
		return ""
	}
	return t.spec.GetPipelineInfo().GetName()
}

func (t *V2Spec) OverrideV2PipelineName(name, namespace string) {
	if t == nil {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	t.spec.PipelineInfo.Name = pipelineRef
}

func (t *V2Spec) ParametersJSON() (string, error) {
	// TODO(v2): implement this after pipeline spec can contain parameter defaults
	return "[]", nil
}

func (t *V2Spec) RunWorkflow(modelRun *model.Run, options RunWorkflowOptions) (util.ExecutionSpec, error) {
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := modelToPipelineJobRuntimeConfig(&modelRun.RuntimeConfig)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	if err = t.validatePipelineJobInputs(job); err != nil {
		return nil, util.Wrap(err, "invalid pipeline job inputs")
	}
	// Pick out Kubernetes platform configs
	var kubernetesSpec *pipelinespec.SinglePlatformSpec
	if t.platformSpec != nil {
		if _, ok := t.platformSpec.Platforms["kubernetes"]; ok {
			kubernetesSpec = t.platformSpec.Platforms["kubernetes"]
		}
	}

	var obj interface{}
	if util.CurrentExecutionType() == util.ArgoWorkflow {
		obj, err = argocompiler.Compile(job, kubernetesSpec, nil)
	} else if util.CurrentExecutionType() == util.TektonPipelineRun {
		obj, err = tektoncompiler.Compile(job, kubernetesSpec, nil)
	}
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	executionSpec, err := util.NewExecutionSpecFromInterface(util.CurrentExecutionType(), obj)
	if err != nil {
		return nil, util.Wrap(err, "Error creating execution spec")
	}
	// Overwrite namespace from the run object
	if modelRun.Namespace != "" {
		executionSpec.SetExecutionNamespace(modelRun.Namespace)
	}
	setDefaultServiceAccount(executionSpec, modelRun.ServiceAccount)
	// Disable istio sidecar injection if not specified
	executionSpec.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	// Add label to the workflow so it can be persisted by persistent agent later.
	executionSpec.SetLabels(util.LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	executionSpec.SetAnnotations(util.AnnotationKeyRunName, modelRun.DisplayName)
	// Replace {{workflow.uid}} with runId
	err = executionSpec.ReplaceUID(options.RunId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to replace workflow ID")
	}
	executionSpec.SetPodMetadataLabels(util.LabelKeyWorkflowRunId, options.RunId)
	return executionSpec, nil
}

func IsPlatformSpecWithKubernetesConfig(template []byte) bool {
	var platformSpec pipelinespec.PlatformSpec
	templateJson, err := yaml.YAMLToJSON(template)
	if err != nil {
		return false
	}
	err = protojson.Unmarshal(templateJson, &platformSpec)
	if err != nil {
		return false
	}
	_, ok := platformSpec.Platforms["kubernetes"]
	return ok
}

func (t *V2Spec) validatePipelineJobInputs(job *pipelinespec.PipelineJob) error {
	// If the pipeline requires no input, t.spec.GetRoot().GetInputDefinitions() will be nil,
	// but we still need to verify that no extra parameters are provided.
	requiredParams := make(map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec)
	if t.spec.GetRoot().GetInputDefinitions() != nil {
		requiredParams = t.spec.GetRoot().GetInputDefinitions().GetParameters()
	}

	runtimeConfig := job.GetRuntimeConfig()
	if runtimeConfig == nil {
		if len(requiredParams) != 0 {
			requiredParamNames := make([]string, 0)
			for name, _ := range requiredParams {
				requiredParamNames = append(requiredParamNames, name)
			}
			return util.NewInvalidInputError(
				"pipeline requiring input has no paramater(s) provided. Need parameter(s): %s",
				strings.Join(requiredParamNames, ", "),
			)
		} else {
			// both required parameters and inputs are empty
			return nil
		}
	}

	// Verify that required parameters are provided
	for name, param := range requiredParams {
		if input, ok := runtimeConfig.GetParameterValues()[name]; !ok {
			// If the parameter is optional, or there is a default value, it's ok to not have a user input
			if !param.GetIsOptional() && param.GetDefaultValue() == nil {
				return util.NewInvalidInputError("parameter %s is not optional, yet has neither default value "+
					"nor user provided value", name)
			}
		} else {
			// Verify the parameter type is correct
			switch param.GetParameterType() {
			case pipelinespec.ParameterType_PARAMETER_TYPE_ENUM_UNSPECIFIED:
				return util.NewInvalidInputError(fmt.Sprintf("input parameter %s has unspecified type", name))
			case pipelinespec.ParameterType_NUMBER_DOUBLE, pipelinespec.ParameterType_NUMBER_INTEGER:
				if _, ok := input.GetKind().(*structpb.Value_NumberValue); !ok {
					return util.NewInvalidInputError("input parameter %s requires type double or integer, "+
						"but the parameter value is not of number value type", name)
				}
			case pipelinespec.ParameterType_STRING:
				if _, ok := input.GetKind().(*structpb.Value_StringValue); !ok {
					return util.NewInvalidInputError("input parameter %s requires type string, but the "+
						"input parameter is not of string value type", name)
				}
			case pipelinespec.ParameterType_BOOLEAN:
				if _, ok := input.GetKind().(*structpb.Value_BoolValue); !ok {
					return util.NewInvalidInputError("input parameter %s requires type bool, but the input "+
						"parameter is not of bool value type", name)
				}
			case pipelinespec.ParameterType_LIST:
				if _, ok := input.GetKind().(*structpb.Value_ListValue); !ok {
					return util.NewInvalidInputError("input parameter %s requires type list, but the input "+
						"parameter is not of list value type", name)
				}
			case pipelinespec.ParameterType_STRUCT:
				if _, ok := input.GetKind().(*structpb.Value_StructValue); !ok {
					return util.NewInvalidInputError("input parameter %s requires type struct, but the input "+
						"parameter is not of struct value type", name)
				}
			case pipelinespec.ParameterType_TASK_FINAL_STATUS:
				return util.NewInvalidInputError("input parameter %s requires type TASK_FINAL_STATUS, which is "+
					"invalid for root component", name)
			default:
				return util.NewInvalidInputError("input parameter %s requires type unknown", name)
			}
		}
	}

	// Verify that only required parameters are provided
	extraParams := make([]string, 0)
	for name, _ := range runtimeConfig.GetParameterValues() {
		if _, ok := requiredParams[name]; !ok {
			extraParams = append(extraParams, name)
		}
	}
	if len(extraParams) > 0 {
		return util.NewInvalidInputError("parameter(s) provided are not required by pipeline: %s",
			strings.Join(extraParams, ", "))
	}

	return nil
}
