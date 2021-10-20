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

package util

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
	"github.com/ghodss/yaml"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/kubeflow/pipelines/v2/compiler"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TemplateType string

const (
	V1      TemplateType = "v1Argo"
	V2      TemplateType = "v2"
	Unknown TemplateType = "Unknown"

	argoGroup       = "argoproj.io/"
	argoVersion     = "argoproj.io/v1alpha1"
	argoK8sResource = "Workflow"

	DefaultPipelineRunnerServiceAccount = "pipeline-runner"
	HasDefaultBucketEnvVar              = "HAS_DEFAULT_BUCKET"
	ProjectIDEnvVar                     = "PROJECT_ID"
	DefaultBucketNameEnvVar             = "BUCKET_NAME"
)

// Unmarshal parameters from JSON encoded string.
func UnmarshalParameters(paramsString string) ([]v1alpha1.Parameter, error) {
	if paramsString == "" {
		return nil, nil
	}
	var params []v1alpha1.Parameter
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, NewInternalServerError(err, "Parameters have wrong format")
	}
	return params, nil
}

// Marshal parameters to JSON encoded string.
// This also checks result is not longer than a limit.
func MarshalParameters(params []v1alpha1.Parameter) (string, error) {
	if params == nil {
		return "[]", nil
	}
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

func ValidateWorkflow(template []byte) (*Workflow, error) {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to parse the workflow template.")
	}
	if wf.APIVersion != argoVersion {
		return nil, NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", argoVersion, wf.APIVersion)
	}
	if wf.Kind != argoK8sResource {
		return nil, NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", argoK8sResource, wf.Kind)
	}
	_, err = validate.ValidateWorkflow(nil, nil, &wf, validate.ValidateOpts{
		Lint:                       true,
		IgnoreEntrypoint:           true,
		WorkflowTemplateValidation: false, // not used by kubeflow
	})
	if err != nil {
		return nil, err
	}
	return NewWorkflow(&wf), nil
}

var ErrorInvalidPipelineSpec = fmt.Errorf("pipeline spec is invalid")

// InferTemplateFormat infers format from pipeline template.
// There is no guarantee that the template is valid in inferred format, so validation
// is still needed.
func InferTemplateFormat(template []byte) TemplateType {
	switch {
	case len(template) == 0:
		return Unknown
	case isArgoWorkflow(template):
		return V1
	case isPipelineSpec(template):
		return V2
	default:
		return Unknown
	}
}

// isArgoWorkflow returns whether template is in argo workflow spec format.
func isArgoWorkflow(template []byte) bool {
	var meta metav1.TypeMeta
	err := yaml.Unmarshal(template, &meta)
	if err != nil {
		return false
	}
	return strings.HasPrefix(meta.APIVersion, argoGroup) && meta.Kind == argoK8sResource
}

// isPipelineSpec returns whether template is in KFP api/v2alpha1/PipelineSpec format.
func isPipelineSpec(template []byte) bool {
	var spec pipelinespec.PipelineSpec
	err := protojson.Unmarshal(template, &spec)
	return err == nil && spec.GetPipelineInfo().GetName() != "" && spec.GetRoot() != nil
}

// Pipeline template
type Template interface {
	IsV2() bool
	// Overrides v2 pipeline name to distinguish shared/namespaced pipelines.
	// The name is used as ML Metadata pipeline context name.
	OverrideV2PipelineName(name, namespace string)
	// Gets parameters in JSON format.
	ParametersJSON() (string, error)
	// Get bytes content.
	Bytes() []byte
	GetTemplateType() TemplateType

	//Get workflow
	RunWorkflow(apiRun *api.Run, options RunWorkflowOptions) (*Workflow, error)

	ScheduledWorkflow(apiJob *api.Job) (*scheduledworkflow.ScheduledWorkflow, error)
}

type RunWorkflowOptions struct {
	RunId string
	RunAt int64
}

func NewTemplate(bytes []byte) (Template, error) {
	format := InferTemplateFormat(bytes)
	switch format {
	case V1:
		return NewArgoTemplate(bytes)
	case V2:
		return NewV2SpecTemplate(bytes)
	default:
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "unknown template format")
	}
}

type ArgoTemplate struct {
	wf *Workflow
}

func (t *ArgoTemplate) ScheduledWorkflow(apiJob *api.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	workflow := NewWorkflow(t.wf.Workflow.DeepCopy())

	parameters := toParametersMap(apiJob.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)
	setDefaultServiceAccount(workflow, apiJob.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(AnnotationKeyIstioSidecarInject, AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, Wrap(err, "Create job failed")
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
			NoCatchup: BoolPointer(apiJob.NoCatchup),
		},
	}

	// Marking auto-added artifacts as optional. Otherwise most older workflows will start failing after upgrade to Argo 2.3.
	// TODO: Fix the components to explicitly declare the artifacts they really output.
	for templateIdx, template := range scheduledWorkflow.Spec.Workflow.Spec.Templates {
		for artIdx, artifact := range template.Outputs.Artifacts {
			if artifact.Name == "mlpipeline-ui-metadata" || artifact.Name == "mlpipeline-metrics" {
				scheduledWorkflow.Spec.Workflow.Spec.Templates[templateIdx].Outputs.Artifacts[artIdx].Optional = true
			}
		}
	}
	return scheduledWorkflow, nil
}

func (t *ArgoTemplate) GetTemplateType() TemplateType {
	return V1
}

func NewArgoTemplate(bytes []byte) (*ArgoTemplate, error) {
	wf, err := ValidateWorkflow(bytes)
	if err != nil {
		return nil, err
	}
	return &ArgoTemplate{wf}, nil
}

func (t *ArgoTemplate) Bytes() []byte {
	if t == nil {
		return nil
	}
	return []byte(t.wf.ToStringForStore())
}

func (t *ArgoTemplate) IsV2() bool {
	if t == nil {
		return false
	}
	return t.wf.IsV2Compatible()
}

func (t *ArgoTemplate) OverrideV2PipelineName(name, namespace string) {
	if !t.wf.IsV2Compatible() {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	overrides := make(map[string]string)
	overrides["pipeline-name"] = pipelineRef
	t.wf.OverrideParameters(overrides)
}

func (t *ArgoTemplate) ParametersJSON() (string, error) {
	if t == nil {
		return "", nil
	}
	return MarshalParameters(t.wf.Spec.Arguments.Parameters)
}

func (t *ArgoTemplate) RunWorkflow(apiRun *api.Run, options RunWorkflowOptions) (*Workflow, error) {
	workflow := NewWorkflow(t.wf.Workflow.DeepCopy())

	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	// TODO: Add run_level flag with similar logic by reading flag value from create_run api.
	workflow.SetLabelsToAllTemplates(LabelKeyCacheEnabled, common.IsCacheEnabled())
	parameters := toParametersMap(apiRun.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	// Replace macros
	formatter := NewRunParameterFormatter(options.RunId, options.RunAt)
	formattedParams := formatter.FormatWorkflowParameters(workflow.GetWorkflowParametersAsMap())
	workflow.OverrideParameters(formattedParams)

	setDefaultServiceAccount(workflow, apiRun.GetServiceAccount())

	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(AnnotationKeyIstioSidecarInject, AnnotationValueIstioSidecarInjectDisabled)

	err := OverrideParameterWithSystemDefault(workflow)
	if err != nil {
		return nil, err
	}

	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(AnnotationKeyRunName, apiRun.Name)
	// Replace {{workflow.uid}} with runId
	err = workflow.ReplaceUID(options.RunId)
	if err != nil {
		return nil, NewInternalServerError(err, "Failed to replace workflow ID")
	}
	workflow.SetPodMetadataLabels(LabelKeyWorkflowRunId, options.RunId)

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

type V2SpecTemplate struct {
	spec *pipelinespec.PipelineSpec
}

func (t *V2SpecTemplate) ScheduledWorkflow(apiJob *api.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, Wrap(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := toPipelineJobRuntimeConfig(apiJob.GetPipelineSpec().GetRuntimeConfig())
	if err != nil {
		return nil, Wrap(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	wf, err := compiler.Compile(job, nil)
	if err != nil {
		return nil, Wrap(err, "Failed to compile job")
	}
	workflow := NewWorkflow(wf)
	setDefaultServiceAccount(workflow, apiJob.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(AnnotationKeyIstioSidecarInject, AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, Wrap(err, "Create job failed")
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
			NoCatchup: BoolPointer(apiJob.NoCatchup),
		},
	}
	return scheduledWorkflow, nil
}

func (t *V2SpecTemplate) GetTemplateType() TemplateType {
	return V2
}

func NewV2SpecTemplate(template []byte) (*V2SpecTemplate, error) {
	var spec pipelinespec.PipelineSpec
	err := protojson.Unmarshal(template, &spec)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("invalid v2 pipeline spec: %s", err.Error()))
	}
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: name is empty")
	}
	if spec.GetRoot() == nil {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: root component is empty")
	}
	return &V2SpecTemplate{spec: &spec}, nil
}

func (t *V2SpecTemplate) Bytes() []byte {
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

func (t *V2SpecTemplate) IsV2() bool {
	return true
}

func (t *V2SpecTemplate) OverrideV2PipelineName(name, namespace string) {
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	t.spec.PipelineInfo.Name = pipelineRef
}

func (t *V2SpecTemplate) ParametersJSON() (string, error) {
	// TODO(v2): implement this after pipeline spec can contain parameter defaults
	return "[]", nil
}

func (t *V2SpecTemplate) RunWorkflow(apiRun *api.Run, options RunWorkflowOptions) (*Workflow, error) {
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		return nil, Wrap(err, "Failed marshal pipeline spec to json")
	}
	spec := &structpb.Struct{}
	if err := protojson.Unmarshal(bytes, spec); err != nil {
		return nil, Wrap(err, "Failed to parse pipeline spec")
	}
	job := &pipelinespec.PipelineJob{PipelineSpec: spec}
	jobRuntimeConfig, err := toPipelineJobRuntimeConfig(apiRun.GetPipelineSpec().GetRuntimeConfig())
	if err != nil {
		return nil, Wrap(err, "Failed to convert to PipelineJob RuntimeConfig")
	}
	job.RuntimeConfig = jobRuntimeConfig
	wf, err := compiler.Compile(job, nil)
	if err != nil {
		return nil, Wrap(err, "Failed to compile job")
	}
	workflow := NewWorkflow(wf)
	setDefaultServiceAccount(workflow, apiRun.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(AnnotationKeyIstioSidecarInject, AnnotationValueIstioSidecarInjectDisabled)
	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(AnnotationKeyRunName, apiRun.Name)
	// Replace {{workflow.uid}} with runId
	err = workflow.ReplaceUID(options.RunId)
	if err != nil {
		return nil, NewInternalServerError(err, "Failed to replace workflow ID")
	}
	workflow.SetPodMetadataLabels(LabelKeyWorkflowRunId, options.RunId)
	return workflow, nil

}

func toParametersMap(apiParams []*api.Parameter) map[string]string {
	// Preprocess workflow by appending parameter and add pipeline specific labels
	desiredParamsMap := make(map[string]string)
	for _, param := range apiParams {
		desiredParamsMap[param.Name] = param.Value
	}
	return desiredParamsMap
}

// Patch the system-specified default parameters if available.
func OverrideParameterWithSystemDefault(workflow *Workflow) error {
	// Patch the default value to workflow spec.
	if common.GetBoolConfigWithDefault(HasDefaultBucketEnvVar, false) {
		patchedSlice := make([]wfv1.Parameter, 0)
		for _, currentParam := range workflow.Spec.Arguments.Parameters {
			if currentParam.Value != nil {
				desiredValue, err := PatchPipelineDefaultParameter(currentParam.Value.String())
				if err != nil {
					return fmt.Errorf("failed to patch default value to pipeline. Error: %v", err)
				}
				patchedSlice = append(patchedSlice, wfv1.Parameter{
					Name:  currentParam.Name,
					Value: wfv1.AnyStringPtr(desiredValue),
				})
			} else if currentParam.Default != nil {
				desiredValue, err := PatchPipelineDefaultParameter(currentParam.Default.String())
				if err != nil {
					return fmt.Errorf("failed to patch default value to pipeline. Error: %v", err)
				}
				patchedSlice = append(patchedSlice, wfv1.Parameter{
					Name:  currentParam.Name,
					Value: wfv1.AnyStringPtr(desiredValue),
				})
			}
		}
		workflow.Spec.Arguments.Parameters = patchedSlice
	}
	return nil
}

// Mutate default values of specified pipeline spec.
// Args:
//  text: (part of) pipeline file in string.
func PatchPipelineDefaultParameter(text string) (string, error) {
	defaultBucket := common.GetStringConfig(DefaultBucketNameEnvVar)
	projectId := common.GetStringConfig(ProjectIDEnvVar)
	toPatch := map[string]string{
		"{{kfp-default-bucket}}": defaultBucket,
		"{{kfp-project-id}}":     projectId,
	}
	for key, value := range toPatch {
		text = strings.Replace(text, key, value, -1)
	}
	return text, nil
}

func setDefaultServiceAccount(workflow *Workflow, serviceAccount string) {
	if len(serviceAccount) > 0 {
		workflow.SetServiceAccount(serviceAccount)
		return
	}
	workflowServiceAccount := workflow.Spec.ServiceAccountName
	if len(workflowServiceAccount) == 0 || workflowServiceAccount == DefaultPipelineRunnerServiceAccount {
		// To reserve SDK backward compatibility, the backend only replaces
		// serviceaccount when it is empty or equal to default value set by SDK.
		workflow.SetServiceAccount(common.GetStringConfigWithDefault(common.DefaultPipelineRunnerServiceAccount, DefaultPipelineRunnerServiceAccount))
	}
}

// Process the job name to remove special char, prepend with "job-" prefix if empty, and
// truncate size to <=25
func toSWFCRDResourceGeneratedName(displayName string) (string, error) {
	const (
		// K8s resource name only allow lower case alphabetic char, number and -
		swfCompatibleNameRegx = "[^a-z0-9-]+"
	)
	reg, err := regexp.Compile(swfCompatibleNameRegx)
	if err != nil {
		return "", NewInternalServerError(err, "Failed to compile ScheduledWorkflow name replacer Regex.")
	}
	processedName := reg.ReplaceAllString(strings.ToLower(displayName), "")
	if processedName == "" {
		processedName = "job-"
	}
	return Truncate(processedName, 25), nil
}

func toCRDTrigger(apiTrigger *api.Trigger) *scheduledworkflow.Trigger {
	var crdTrigger scheduledworkflow.Trigger
	if apiTrigger.GetCronSchedule() != nil {
		crdTrigger.CronSchedule = toCRDCronSchedule(apiTrigger.GetCronSchedule())
	}
	if apiTrigger.GetPeriodicSchedule() != nil {
		crdTrigger.PeriodicSchedule = toCRDPeriodicSchedule(apiTrigger.GetPeriodicSchedule())
	}
	return &crdTrigger
}

func toCRDCronSchedule(cronSchedule *api.CronSchedule) *scheduledworkflow.CronSchedule {
	if cronSchedule == nil || cronSchedule.Cron == "" {
		return nil
	}
	crdCronSchedule := scheduledworkflow.CronSchedule{}
	crdCronSchedule.Cron = cronSchedule.Cron

	if cronSchedule.StartTime != nil {
		startTime := metav1.NewTime(time.Unix(cronSchedule.StartTime.Seconds, 0))
		crdCronSchedule.StartTime = &startTime
	}
	if cronSchedule.EndTime != nil {
		endTime := metav1.NewTime(time.Unix(cronSchedule.EndTime.Seconds, 0))
		crdCronSchedule.EndTime = &endTime
	}
	return &crdCronSchedule
}

func toCRDPeriodicSchedule(periodicSchedule *api.PeriodicSchedule) *scheduledworkflow.PeriodicSchedule {
	if periodicSchedule == nil || periodicSchedule.IntervalSecond == 0 {
		return nil
	}
	crdPeriodicSchedule := scheduledworkflow.PeriodicSchedule{}
	crdPeriodicSchedule.IntervalSecond = periodicSchedule.IntervalSecond
	if periodicSchedule.StartTime != nil {
		startTime := metav1.NewTime(time.Unix(periodicSchedule.StartTime.Seconds, 0))
		crdPeriodicSchedule.StartTime = &startTime
	}
	if periodicSchedule.EndTime != nil {
		endTime := metav1.NewTime(time.Unix(periodicSchedule.EndTime.Seconds, 0))
		crdPeriodicSchedule.EndTime = &endTime
	}
	return &crdPeriodicSchedule
}

func toCRDParameter(apiParams []*api.Parameter) []scheduledworkflow.Parameter {
	var swParams []scheduledworkflow.Parameter
	for _, apiParam := range apiParams {
		swParam := scheduledworkflow.Parameter{
			Name:  apiParam.Name,
			Value: apiParam.Value,
		}
		swParams = append(swParams, swParam)
	}
	return swParams
}

func toPipelineJobRuntimeConfig(apiRuntimeConfig *api.PipelineSpec_RuntimeConfig) (*pipelinespec.PipelineJob_RuntimeConfig, error) {
	if apiRuntimeConfig == nil {
		return nil, nil
	}
	runTimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
	runTimeConfig.Parameters = make(map[string]*pipelinespec.Value)
	for k, v := range apiRuntimeConfig.GetParameters() {
		value := &pipelinespec.Value{}
		switch t := v.Value.(type) {
		case *api.Value_StringValue:
			value.Value = &pipelinespec.Value_StringValue{StringValue: v.GetStringValue()}
		case *api.Value_DoubleValue:
			value.Value = &pipelinespec.Value_DoubleValue{DoubleValue: v.GetDoubleValue()}
		case *api.Value_IntValue:
			value.Value = &pipelinespec.Value_IntValue{IntValue: v.GetIntValue()}
		default:
			return nil, fmt.Errorf("unknown property type in pipelineSpec runtimeConfig Parameters: %T", t)
		}
		runTimeConfig.Parameters[k] = value
	}
	if apiRuntimeConfig.GetPipelineRoot() != "" {
		runTimeConfig.GcsOutputDirectory = apiRuntimeConfig.GetPipelineRoot()
	}
	return runTimeConfig, nil
}
