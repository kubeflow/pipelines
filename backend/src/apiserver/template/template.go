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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
)

var ErrorInvalidPipelineSpec = fmt.Errorf("pipeline spec is invalid")

// inferTemplateFormat infers format from pipeline template.
// There is no guarantee that the template is valid in inferred format, so validation
// is still needed.
func inferTemplateFormat(template []byte) TemplateType {
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
	templateJson, err := yaml.YAMLToJSON(template)
	if err != nil {
		return false
	}
	err = protojson.Unmarshal(templateJson, &spec)
	return err == nil && spec.GetPipelineInfo().GetName() != "" && spec.GetRoot() != nil
}

// Pipeline template.
type Template interface {
	IsV2() bool
	// Gets v2 pipeline name.
	V2PipelineName() string
	// Overrides v2 pipeline name to distinguish shared/namespaced pipelines.
	// The name is used as ML Metadata pipeline context name.
	OverrideV2PipelineName(name, namespace string)
	// Gets parameters in JSON format.
	ParametersJSON() (string, error)
	// Get bytes content.
	Bytes() []byte
	GetTemplateType() TemplateType

	// Get workflow
	RunWorkflow(modelRun *model.Run, options RunWorkflowOptions) (util.ExecutionSpec, error)

	ScheduledWorkflow(modelJob *model.Job) (*scheduledworkflow.ScheduledWorkflow, error)
}

type RunWorkflowOptions struct {
	RunId string
	RunAt int64
}

func NewFromString(s string) (Template, error) {
	bytes := []byte(s)
	return New(bytes)
}

func New(bytes []byte) (Template, error) {
	format := inferTemplateFormat(bytes)
	switch format {
	case V1:
		return NewArgoTemplate(bytes)
	case V2:
		return NewV2SpecTemplate(bytes)
	default:
		return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "unknown template format")
	}
}

func modelToPipelineJobRuntimeConfig(modelRuntimeConfig *model.RuntimeConfig) (*pipelinespec.PipelineJob_RuntimeConfig, error) {
	if modelRuntimeConfig == nil {
		return nil, nil
	}
	parameters := new(map[string]*structpb.Value)
	if modelRuntimeConfig.Parameters != "" {
		err := json.Unmarshal([]byte(modelRuntimeConfig.Parameters), parameters)
		if err != nil {
			return nil, err
		}
	}
	runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
	runtimeConfig.ParameterValues = *parameters
	runtimeConfig.GcsOutputDirectory = modelRuntimeConfig.PipelineRoot
	return runtimeConfig, nil
}

func modelToCRDParameters(modelParams string) ([]scheduledworkflow.Parameter, error) {
	var swParams []scheduledworkflow.Parameter
	var parameters map[string]*structpb.Value
	if modelParams == "" {
		return swParams, nil
	}
	err := json.Unmarshal([]byte(modelParams), &parameters)
	if err != nil {
		return nil, err
	}
	for name, value := range parameters {
		valueBytes, err := value.MarshalJSON()
		if err != nil {
			return nil, err
		}
		swParam := scheduledworkflow.Parameter{
			Name:  name,
			Value: string(valueBytes),
		}
		swParams = append(swParams, swParam)
	}
	return swParams, nil
}

func modelToParametersMap(modelParameters string) (map[string]string, error) {
	var paramsMapList []*map[string]string
	desiredParamsMap := make(map[string]string)
	if modelParameters == "" {
		return desiredParamsMap, nil
	}
	err := json.Unmarshal([]byte(modelParameters), &paramsMapList)
	if err != nil {
		return nil, err
	}
	for _, param := range paramsMapList {
		desiredParamsMap[(*param)["name"]] = (*param)["value"]
	}
	return desiredParamsMap, nil
}

func modelToCRDTrigger(modelTrigger model.Trigger) (scheduledworkflow.Trigger, error) {
	crdTrigger := scheduledworkflow.Trigger{}
	// CronSchedule and PeriodicSchedule can have at most one being non-empty
	if modelTrigger.CronSchedule != (model.CronSchedule{}) {
		// Check if CronSchedule is non-empty
		crdCronSchedule := scheduledworkflow.CronSchedule{}
		if modelTrigger.Cron != nil {
			crdCronSchedule.Cron = *modelTrigger.Cron
		}
		if modelTrigger.CronScheduleStartTimeInSec != nil {
			startTime := metav1.NewTime(time.Unix(*modelTrigger.CronScheduleStartTimeInSec, 0))
			crdCronSchedule.StartTime = &startTime
		}
		if modelTrigger.CronScheduleEndTimeInSec != nil {
			endTime := metav1.NewTime(time.Unix(*modelTrigger.CronScheduleEndTimeInSec, 0))
			crdCronSchedule.EndTime = &endTime
		}
		crdTrigger.CronSchedule = &crdCronSchedule
	} else if modelTrigger.PeriodicSchedule != (model.PeriodicSchedule{}) {
		// Check if PeriodicSchedule is non-empty
		crdPeriodicSchedule := scheduledworkflow.PeriodicSchedule{}
		if modelTrigger.IntervalSecond != nil {
			crdPeriodicSchedule.IntervalSecond = *modelTrigger.IntervalSecond
		}
		if modelTrigger.PeriodicScheduleStartTimeInSec != nil {
			startTime := metav1.NewTime(time.Unix(*modelTrigger.PeriodicScheduleStartTimeInSec, 0))
			crdPeriodicSchedule.StartTime = &startTime
		}
		if modelTrigger.PeriodicScheduleEndTimeInSec != nil {
			endTime := metav1.NewTime(time.Unix(*modelTrigger.PeriodicScheduleEndTimeInSec, 0))
			crdPeriodicSchedule.EndTime = &endTime
		}
		crdTrigger.PeriodicSchedule = &crdPeriodicSchedule
	}
	return crdTrigger, nil
}

// Patch the system-specified default parameters if available.
func OverrideParameterWithSystemDefault(execSpec util.ExecutionSpec) error {
	// Patch the default value to workflow spec.
	if common.GetBoolConfigWithDefault(common.HasDefaultBucketEnvVar, false) {
		params := execSpec.SpecParameters()
		patched := make(util.SpecParameters, 0, len(params))
		for _, currentParam := range params {
			if currentParam.Value != nil {
				desiredValue, err := common.PatchPipelineDefaultParameter(*currentParam.Value)
				if err != nil {
					return fmt.Errorf("failed to patch default value to pipeline. Error: %v", err)
				}
				patched = append(patched, util.SpecParameter{Name: currentParam.Name, Value: &desiredValue})
			} else if currentParam.Default != nil {
				desiredValue, err := common.PatchPipelineDefaultParameter(*currentParam.Default)
				if err != nil {
					return fmt.Errorf("failed to patch default value to pipeline. Error: %v", err)
				}
				patched = append(patched, util.SpecParameter{Name: currentParam.Name, Default: &desiredValue})
			}
		}
		execSpec.SetSpecParameters(patched)
	}
	return nil
}

func setDefaultServiceAccount(workflow util.ExecutionSpec, serviceAccount string) {
	if len(serviceAccount) > 0 {
		workflow.SetServiceAccount(serviceAccount)
		return
	}
	workflowServiceAccount := workflow.ServiceAccount()
	if len(workflowServiceAccount) == 0 || workflowServiceAccount == common.DefaultPipelineRunnerServiceAccount {
		// To reserve SDK backward compatibility, the backend only replaces
		// serviceaccount when it is empty or equal to default value set by SDK.
		workflow.SetServiceAccount(common.GetStringConfigWithDefault(common.DefaultPipelineRunnerServiceAccountFlag, common.DefaultPipelineRunnerServiceAccount))
	}
}

// Process the job name to remove special char, prepend with "job-" prefix if empty, and
// truncate size to <=25.
func toSWFCRDResourceGeneratedName(displayName string) (string, error) {
	const (
		// K8s resource name only allow lower case alphabetic char, number and -
		swfCompatibleNameRegx = "[^a-z0-9-]+"
	)
	reg, err := regexp.Compile(swfCompatibleNameRegx)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to compile ScheduledWorkflow name replacer Regex")
	}
	processedName := reg.ReplaceAllString(strings.ToLower(displayName), "")
	if processedName == "" {
		processedName = "job-"
	}
	return util.Truncate(processedName, 25), nil
}
