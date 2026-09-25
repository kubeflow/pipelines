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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
	structpb "google.golang.org/protobuf/types/known/structpb"
	goyaml "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TemplateType string

const (
	V2      TemplateType = "v2"
	Unknown TemplateType = "Unknown"

	SCHEMA_VERSION_2_1_0 = "2.1.0"
)

var (
	ErrorInvalidPipelineSpec   = fmt.Errorf("pipeline spec is invalid")
	ErrorInvalidPlatformSpec   = fmt.Errorf("platform spec is invalid")
	errUnsupportedArgoWorkflow = errors.New("legacy Argo Workflow pipelines are no longer supported; rewrite the pipeline with the KFP v2 SDK and upload compiled PipelineSpec IR YAML")
)

// inferTemplateFormat infers format from pipeline template.
// There is no guarantee that the template is valid in inferred format, so validation
// is still needed.
func inferTemplateFormat(template []byte) (TemplateType, error) {
	decoder := goyaml.NewDecoder(bytes.NewReader(template))
	for {
		var value map[string]interface{}

		err := decoder.Decode(&value)
		if err != nil {
			var typeErr *goyaml.TypeError
			if errors.As(err, &typeErr) {
				continue
			}
			if errors.Is(err, io.EOF) {
				return Unknown, nil
			}
			return Unknown, err
		}
		if value == nil {
			continue
		}
		apiVersion, _ := value["apiVersion"].(string)
		if value["kind"] == "Workflow" && strings.HasPrefix(apiVersion, "argoproj.io/") {
			return Unknown, errUnsupportedArgoWorkflow
		}
		if isPipelineSpec(value) {
			return V2, nil
		}
	}
}

// isPipelineSpec returns whether template is in KFP api/v2alpha1/PipelineSpec format.
func isPipelineSpec(value map[string]interface{}) bool {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var spec pipelinespec.PipelineSpec
	err = protojson.Unmarshal(jsonData, &spec)
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

	IsCacheDisabled() bool
}

type RunWorkflowOptions struct {
	RunID string
	RunAt int64
}

type TemplateOptions struct {
	CacheDisabled        bool
	DefaultWorkspace     *corev1.PersistentVolumeClaimSpec
	MLPipelineTLSEnabled bool
	DefaultRunAsUser     *int64
	DefaultRunAsGroup    *int64
	DefaultRunAsNonRoot  *bool
	DefaultHostUsers     *bool
}

func New(bytes []byte, opts TemplateOptions) (Template, error) {
	format, parseErr := inferTemplateFormat(bytes)
	switch format {
	case V2:
		return NewV2SpecTemplate(bytes, opts)
	default:
		if errors.Is(parseErr, errUnsupportedArgoWorkflow) {
			return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, parseErr.Error())
		}
		if parseErr != nil {
			return nil, util.NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("failed to parse pipeline spec YAML: %v", parseErr))
		}
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
			return nil, util.NewInternalServerError(err, "error unmarshalling model runtime config parameters")
		}
	}
	runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
	runtimeConfig.ParameterValues = *parameters
	runtimeConfig.GcsOutputDirectory = string(modelRuntimeConfig.PipelineRoot)
	return runtimeConfig, nil
}

// Converts serialized runtime config's parameters to []scheduledworkflow.Parameter.
// Assumes that the serialized parameters will take a form of
// map[string]*structpb.Value, which works for runtimeConfig.Parameters  such as
// {"param1":"value1","param2":"value2"}.
func StringMapToCRDParameters(modelParams string) ([]scheduledworkflow.Parameter, error) {
	var swParams []scheduledworkflow.Parameter
	var parameters map[string]*structpb.Value
	if modelParams == "" {
		return swParams, nil
	}
	err := json.Unmarshal([]byte(modelParams), &parameters)
	if err != nil {
		return nil, util.NewInternalServerError(err, "error unmarshalling model parameters")
	}
	for name, value := range parameters {
		valueBytes, err := value.MarshalJSON()
		if err != nil {
			return nil, util.NewInternalServerError(err, "error marshaling model parameters")
		}
		swParam := scheduledworkflow.Parameter{
			Name:  name,
			Value: string(valueBytes),
		}
		swParams = append(swParams, swParam)
	}
	return swParams, nil
}

func modelToCRDTrigger(modelTrigger model.Trigger) (scheduledworkflow.Trigger, error) {
	crdTrigger := scheduledworkflow.Trigger{}
	// CronSchedule and PeriodicSchedule can have at most one being non-empty
	if !modelTrigger.CronSchedule.IsEmpty() {
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
	} else if !modelTrigger.PeriodicSchedule.IsEmpty() {
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
