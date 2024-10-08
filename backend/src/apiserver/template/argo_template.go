// Copyright 2018 The Kubeflow Authors
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
	"fmt"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (t *Argo) RunWorkflow(modelRun *model.Run, options RunWorkflowOptions) (util.ExecutionSpec, error) {
	workflow := util.NewWorkflow(t.wf.Workflow.DeepCopy())

	// Overwrite namespace from the run object
	if modelRun.Namespace != "" {
		workflow.SetExecutionNamespace(modelRun.Namespace)
	}
	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	// TODO: Add run_level flag with similar logic by reading flag value from create_run api.
	workflow.SetLabelsToAllTemplates(util.LabelKeyCacheEnabled, common.IsCacheEnabled())

	// Convert parameters
	parameters, err := modelToParametersMap(modelRun.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert parameters")
	}
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	// Replace macros
	formatter := util.NewRunParameterFormatter(options.RunId, options.RunAt)
	formattedParams := formatter.FormatWorkflowParameters(workflow.GetWorkflowParametersAsMap())
	workflow.OverrideParameters(formattedParams)

	setDefaultServiceAccount(workflow, modelRun.ServiceAccount)

	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)

	err = OverrideParameterWithSystemDefault(workflow)
	if err != nil {
		return nil, err
	}

	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(util.LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(util.AnnotationKeyRunName, modelRun.DisplayName)
	// Replace {{workflow.uid}} with runId
	err = workflow.ReplaceUID(options.RunId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to replace workflow ID")
	}
	workflow.SetPodMetadataLabels(util.LabelKeyWorkflowRunId, options.RunId)

	// Marking auto-added artifacts as optional. Otherwise most older workflows will start failing after upgrade to Argo 2.3.
	// TODO: Fix the components to explicitly declare the artifacts they really output.
	for templateIdx, template := range workflow.Workflow.Spec.Templates {
		for artIdx, artifact := range template.Outputs.Artifacts {
			if artifact.Name == "mlpipeline-ui-metadata" || artifact.Name == "mlpipeline-metrics" {
				workflow.Workflow.Spec.Templates[templateIdx].Outputs.Artifacts[artIdx].Optional = true
			}
		}
	}
	return workflow, nil
}

type Argo struct {
	wf *util.Workflow
}

func (t *Argo) ScheduledWorkflow(modelJob *model.Job, ownerReferences []metav1.OwnerReference) (*scheduledworkflow.ScheduledWorkflow, error) {
	workflow := util.NewWorkflow(t.wf.Workflow.DeepCopy())
	// Overwrite namespace from the job object
	if modelJob.Namespace != "" {
		workflow.SetExecutionNamespace(modelJob.Namespace)
	}
	parameters, err := modelToParametersMap(modelJob.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert parameters")
	}
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)
	setDefaultServiceAccount(workflow, modelJob.ServiceAccount)
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(modelJob.K8SName)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	// Marking auto-added artifacts as optional. Otherwise most older workflows will start failing after upgrade to Argo 2.3.
	// TODO: Fix the components to explicitly declare the artifacts they really output.
	workflow.PatchTemplateOutputArtifacts()

	// We assume that v1 Argo template use v1 parameters ignoring runtime config
	swfParameters, err := stringArrayToCRDParameters(modelJob.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert v1 parameters to CRD parameters")
	}
	crdTrigger, err := modelToCRDTrigger(modelJob.Trigger)
	if err != nil {
		return nil, err
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
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
				Parameters: swfParameters,
				Spec:       workflow.ToStringForSchedule(),
			},
			NoCatchup: util.BoolPointer(modelJob.NoCatchup),
		},
	}
	return scheduledWorkflow, nil
}

func (t *Argo) GetTemplateType() TemplateType {
	return V1
}

func NewArgoTemplate(bytes []byte) (*Argo, error) {
	wf, err := ValidateWorkflow(bytes)
	if err != nil {
		return nil, err
	}
	return &Argo{wf}, nil
}

func (t *Argo) Bytes() []byte {
	if t == nil {
		return nil
	}
	return []byte(t.wf.ToStringForStore())
}

func (t *Argo) IsV2() bool {
	if t == nil {
		return false
	}
	return t.wf.IsV2Compatible()
}

const (
	paramV2compatPipelineName = "pipeline-name"
)

func (t *Argo) V2PipelineName() string {
	if t == nil {
		return ""
	}
	return t.wf.GetWorkflowParametersAsMap()[paramV2compatPipelineName]
}

func (t *Argo) OverrideV2PipelineName(name, namespace string) {
	if t == nil || !t.wf.IsV2Compatible() {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	overrides := make(map[string]string)
	overrides[paramV2compatPipelineName] = pipelineRef
	t.wf.OverrideParameters(overrides)
}

func (t *Argo) ParametersJSON() (string, error) {
	if t == nil {
		return "", nil
	}
	return util.MarshalParameters(util.CurrentExecutionType(), t.wf.SpecParameters())
}

func NewArgoTemplateFromWorkflow(wf *workflowapi.Workflow) (*Argo, error) {
	return &Argo{wf: &util.Workflow{Workflow: wf}}, nil
}

func ValidateWorkflow(template []byte) (*util.Workflow, error) {
	var wf workflowapi.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Failed to parse the workflow template")
	}
	if wf.APIVersion != argoVersion {
		return nil, util.NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", argoVersion, wf.APIVersion)
	}
	if wf.Kind != argoK8sResource {
		return nil, util.NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", argoK8sResource, wf.Kind)
	}
	err = validate.ValidateWorkflow(nil, nil, &wf, validate.ValidateOpts{
		Lint:                       true,
		IgnoreEntrypoint:           true,
		WorkflowTemplateValidation: false, // not used by kubeflow
	})
	if err != nil {
		return nil, err
	}
	return util.NewWorkflow(&wf), nil
}

func AddRuntimeMetadata(wf *workflowapi.Workflow) {
	template := wf.Spec.Templates[0]
	template.Metadata.Annotations = map[string]string{"sidecar.istio.io/inject": "false"}
	template.Metadata.Labels = map[string]string{"pipelines.kubeflow.org/cache_enabled": "true"}
	wf.Spec.Templates[0] = template
}
