// Copyright 2018 Google LLC
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
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"k8s.io/apimachinery/pkg/util/json"
)

// ScheduledWorkflow is a type to help manipulate ScheduledWorkflow objects.
type ScheduledWorkflow struct {
	*swfapi.ScheduledWorkflow
}

// NewScheduledWorkflow creates an instance of ScheduledWorkflow.
func NewScheduledWorkflow(swf *swfapi.ScheduledWorkflow) *ScheduledWorkflow {
	return &ScheduledWorkflow{
		swf,
	}
}

// Get converts this object to a swfapi.ScheduledWorkflow.
func (s *ScheduledWorkflow) Get() *swfapi.ScheduledWorkflow {
	return s.ScheduledWorkflow
}

func (s *ScheduledWorkflow) CronScheduleStartTimeInSecOrNull() *int64 {
	if s.Spec.CronSchedule != nil && s.Spec.CronSchedule.StartTime != nil {
		return Int64Pointer(s.Spec.CronSchedule.StartTime.Unix())
	}
	return nil
}

func (s *ScheduledWorkflow) CronScheduleEndTimeInSecOrNull() *int64 {
	if s.Spec.CronSchedule != nil && s.Spec.CronSchedule.EndTime != nil {
		return Int64Pointer(s.Spec.CronSchedule.EndTime.Unix())
	}
	return nil
}

func (s *ScheduledWorkflow) CronOrEmpty() string {
	if s.Spec.CronSchedule != nil {
		return s.Spec.CronSchedule.Cron
	}
	return ""
}

func (s *ScheduledWorkflow) PeriodicScheduleStartTimeInSecOrNull() *int64 {
	if s.Spec.PeriodicSchedule != nil && s.Spec.PeriodicSchedule.StartTime != nil {
		return Int64Pointer(s.Spec.PeriodicSchedule.StartTime.Unix())
	}
	return nil
}

func (s *ScheduledWorkflow) PeriodicScheduleEndTimeInSecOrNull() *int64 {
	if s.Spec.PeriodicSchedule != nil && s.Spec.PeriodicSchedule.EndTime != nil {
		return Int64Pointer(s.Spec.PeriodicSchedule.EndTime.Unix())
	}
	return nil
}

func (s *ScheduledWorkflow) MaxConcurrencyOr0() int64 {
	if s.Spec.MaxConcurrency != nil {
		return *s.Spec.MaxConcurrency
	}
	return 0
}

func (s *ScheduledWorkflow) NoCatchupOrFalse() bool {
	if s.Spec.NoCatchup != nil {
		return *s.Spec.NoCatchup
	}
	return false
}

func (s *ScheduledWorkflow) IntervalSecondOr0() int64 {
	if s.Spec.PeriodicSchedule != nil {
		return s.Spec.PeriodicSchedule.IntervalSecond
	}
	return 0
}

func (s *ScheduledWorkflow) ConditionSummary() string {
	if s.Status.Conditions == nil || len(s.Status.Conditions) == 0 {
		return "NO_STATUS"
	}
	// Only return the latest status
	return string(s.Status.Conditions[len(s.Status.Conditions)-1].Type)
}

func (s *ScheduledWorkflow) ParametersAsString() (string, error) {
	workflowParams := make([]workflowapi.Parameter, 0)

	var params []swfapi.Parameter
	if s.ScheduledWorkflow.Spec.Workflow == nil {
		params = make([]swfapi.Parameter, 0)
	} else {
		params = s.ScheduledWorkflow.Spec.Workflow.Parameters
	}

	for _, param := range params {
		workflowParam := workflowapi.Parameter{
			Name:  param.Name,
			Value: &param.Value,
		}
		workflowParams = append(workflowParams, workflowParam)
	}
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return "", NewInvalidInputError(
			"Failed to marshall parameters as string in scheduled workflow (%v/%v/%v): %v: %+v",
			s.UID, s.Namespace, s.Name, err, params)
	}
	return string(paramsBytes), nil
}

func (s *ScheduledWorkflow) ToStringForStore() string {
	swf, err := json.Marshal(s.ScheduledWorkflow)
	if err != nil {
		glog.Errorf("Could not marshal the scheduled workflow: %v", s.ScheduledWorkflow)
		return ""
	}
	return string(swf)
}
