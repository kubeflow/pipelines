package template

import (
	"fmt"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"regexp"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/kubeflow/pipelines/v2/compiler"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type V2Spec struct {
	spec *pipelinespec.PipelineSpec
}

func (t *V2Spec) ScheduledWorkflow(apiJob *api.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, util.Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.Wrap(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := toPipelineJobRuntimeConfig(apiJob.GetPipelineSpec().GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	wf, err := compiler.Compile(job, nil)
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	workflow := util.NewWorkflow(wf)
	setDefaultServiceAccount(workflow, apiJob.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        apiJob.Enabled,
			MaxConcurrency: &apiJob.MaxConcurrency,
			Trigger:        *toCRDTrigger(apiJob.Trigger),
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCRDParameter(apiJob.GetPipelineSpec().GetParameters()),
				Spec:       workflow.Spec,
			},
			NoCatchup: util.BoolPointer(apiJob.NoCatchup),
		},
	}
	return scheduledWorkflow, nil
}

func (t *V2Spec) GetTemplateType() TemplateType {
	return V2
}

func NewV2SpecTemplate(template []byte) (*V2Spec, error) {
	var spec pipelinespec.PipelineSpec
	err := protojson.Unmarshal(template, &spec)
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
	return &V2Spec{spec: &spec}, nil
}

func (t *V2Spec) Bytes() []byte {
	if t == nil {
		return nil
	}
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		// this is unexpected
		return nil
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

func (t *V2Spec) RunWorkflow(apiRun *api.Run, options RunWorkflowOptions) (*util.Workflow, error) {
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, util.Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, util.Wrap(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := toPipelineJobRuntimeConfig(apiRun.GetPipelineSpec().GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	wf, err := compiler.Compile(job, nil)
	if err != nil {
		return nil, util.Wrap(err, "Failed to compile job")
	}
	workflow := util.NewWorkflow(wf)
	setDefaultServiceAccount(workflow, apiRun.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
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
	return workflow, nil

}
