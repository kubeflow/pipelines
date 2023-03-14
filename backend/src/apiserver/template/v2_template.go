// Copyright 2021 The Kubeflow Authors
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
	"fmt"
	"regexp"

	"github.com/ghodss/yaml"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"google.golang.org/protobuf/encoding/protojson"
	goyaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type V2Spec struct {
	spec         *pipelinespec.PipelineSpec
	platformSpec *pipelinespec.PlatformSpec
}

// Converts modelJob to ScheduledWorkflow.
func (t *V2Spec) ScheduledWorkflow(modelJob *model.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	job := &pipelinespec.PipelineJob{}

	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, util.Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.Wrap(err, "Failed to parse pipeline spec")
	}

	job.PipelineSpec = spec

	jobRuntimeConfig, err := modelToPipelineJobRuntimeConfig(&modelJob.RuntimeConfig)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert runtime config")
	}
	job.RuntimeConfig = jobRuntimeConfig

	obj, err := argocompiler.Compile(job, nil)
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	// currently, there is only Argo implementation, so it's using `ArgoWorkflow` for now
	// later on, if a new runtime support will be added, we need a way to switch/specify
	// runtime. i.e using ENV var
	executionSpec, err := util.NewExecutionSpecFromInterface(util.ArgoWorkflow, obj)
	if err != nil {
		return nil, util.NewInternalServerError(err, "not Workflow struct")
	}
	// Overwrite namespace from the job object
	if modelJob.Namespace != "" {
		executionSpec.SetExecutionNamespace(modelJob.Namespace)
	}
	setDefaultServiceAccount(executionSpec, modelJob.ServiceAccount)
	// Disable istio sidecar injection if not specified
	executionSpec.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(modelJob.K8SName)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	parameters, err := modelToCRDParameters(modelJob.RuntimeConfig.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Converting model.Job parameters to CDR parameters failed")
	}
	crdTrigger, err := modelToCRDTrigger(modelJob.Trigger)
	if err != nil {
		return nil, err
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        modelJob.Enabled,
			MaxConcurrency: &modelJob.MaxConcurrency,
			Trigger:        crdTrigger,
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: parameters,
				Spec:       executionSpec.ToStringForSchedule(),
			},
			NoCatchup: util.BoolPointer(modelJob.NoCatchup),
		},
	}
	return scheduledWorkflow, nil
}

func (t *V2Spec) GetTemplateType() TemplateType {
	return V2
}

func NewV2SpecTemplate(template []byte) (*V2Spec, error) {
	var spec pipelinespec.PipelineSpec
	var v2Spec V2Spec
	decoder := goyaml.NewDecoder(bytes.NewReader(template))
	for {
		var value map[string]interface{}
		// Break at end of file
		if decoder.Decode(&value) != nil {
			break
		}
		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("unable to marshal this yaml document: %s", err.Error()))
		}
		if isPipelineSpec(valueBytes) {
			// Pick out the yaml document with pipeline spec
			pipelineSpecJson, err := yaml.YAMLToJSON(valueBytes)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("cannot convert v2 pipeline spec to json format: %s", err.Error()))
			}
			err = protojson.Unmarshal(pipelineSpecJson, &spec)
			if err != nil {
				return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("invalid v2 pipeline spec: %s", err.Error()))
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
		} else if isKubernetesExecutorConfig(valueBytes) {
			// Pick out the yaml document with platform spec
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

	bytesExecutorConfig, err := protojson.Marshal(t.platformSpec)
	if err != nil {
		// this is unexpected, cannot convert proto message to JSON
		return nil
	}
	bytes := append(bytesSpec, bytesExecutorConfig...)
	bytesYAML, err := yaml.JSONToYAML(bytes)
	if err != nil {
		// this is unexpected, cannot convert JSON to YAML
		return nil
	}
	return bytesYAML
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
		return nil, util.Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.Wrap(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := modelToPipelineJobRuntimeConfig(&modelRun.RuntimeConfig)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	obj, err := argocompiler.Compile(job, nil)
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	executionSpec, err := util.NewExecutionSpecFromInterface(util.ArgoWorkflow, obj)
	if err != nil {
		return nil, util.NewInternalServerError(err, "not Workflow struct")
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

func isKubernetesExecutorConfig(template []byte) bool {
	var platformSpec pipelinespec.PlatformSpec
	templateJson, err := yaml.YAMLToJSON(template)
	if err != nil {
		return false
	}
	err = protojson.Unmarshal(templateJson, &platformSpec)
	if err != nil {
		return false
	}
	return true
}
