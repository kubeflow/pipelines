package template

import (
	"fmt"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
	"github.com/ghodss/yaml"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *Argo) RunWorkflow(apiRun *api.Run, options RunWorkflowOptions) (util.ExecutionSpec, error) {
	workflow := util.NewWorkflow(t.wf.Workflow.DeepCopy())

	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	// TODO: Add run_level flag with similar logic by reading flag value from create_run api.
	workflow.SetLabelsToAllTemplates(util.LabelKeyCacheEnabled, common.IsCacheEnabled())
	parameters := toParametersMap(apiRun.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	// Replace macros
	formatter := util.NewRunParameterFormatter(options.RunId, options.RunAt)
	formattedParams := formatter.FormatWorkflowParameters(workflow.GetWorkflowParametersAsMap())
	workflow.OverrideParameters(formattedParams)

	setDefaultServiceAccount(workflow, apiRun.GetServiceAccount())

	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)

	err := OverrideParameterWithSystemDefault(workflow)
	if err != nil {
		return nil, err
	}

	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(util.LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(util.AnnotationKeyRunName, apiRun.Name)
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

func (t *Argo) ScheduledWorkflow(apiJob *api.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	workflow := util.NewWorkflow(t.wf.Workflow.DeepCopy())

	parameters := toParametersMap(apiJob.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)
	setDefaultServiceAccount(workflow, apiJob.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	// Marking auto-added artifacts as optional. Otherwise most older workflows will start failing after upgrade to Argo 2.3.
	// TODO: Fix the components to explicitly declare the artifacts they really output.
	workflow.PatchTemplateOutputArtifacts()

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        apiJob.Enabled,
			MaxConcurrency: &apiJob.MaxConcurrency,
			Trigger:        *toCRDTrigger(apiJob.Trigger),
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCRDParameter(apiJob.GetPipelineSpec().GetParameters()),
				Spec:       workflow.ToStringForSchedule(),
			},
			NoCatchup: util.BoolPointer(apiJob.NoCatchup),
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
	return util.MarshalParameters(util.ArgoWorkflow, t.wf.SpecParameters())
}

func NewArgoTemplateFromWorkflow(wf *workflowapi.Workflow) (*Argo, error) {
	return &Argo{wf: &util.Workflow{Workflow: wf}}, nil
}

func ValidateWorkflow(template []byte) (*util.Workflow, error) {
	var wf workflowapi.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Failed to parse the workflow template.")
	}
	if wf.APIVersion != argoVersion {
		return nil, util.NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", argoVersion, wf.APIVersion)
	}
	if wf.Kind != argoK8sResource {
		return nil, util.NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", argoK8sResource, wf.Kind)
	}
	_, err = validate.ValidateWorkflow(nil, nil, &wf, validate.ValidateOpts{
		Lint:                       true,
		IgnoreEntrypoint:           true,
		WorkflowTemplateValidation: false, // not used by kubeflow
	})
	if err != nil {
		return nil, err
	}
	return util.NewWorkflow(&wf), nil
}
