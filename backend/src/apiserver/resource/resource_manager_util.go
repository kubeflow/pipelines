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

package resource

import (
	"regexp"
	"strings"
	"time"

	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/common/util"
	scheduledworkflow "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
		startTime := v1.NewTime(time.Unix(cronSchedule.StartTime.Seconds, 0))
		crdCronSchedule.StartTime = &startTime
	}
	if cronSchedule.EndTime != nil {
		endTime := v1.NewTime(time.Unix(cronSchedule.EndTime.Seconds, 0))
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
		startTime := v1.NewTime(time.Unix(periodicSchedule.StartTime.Seconds, 0))
		crdPeriodicSchedule.StartTime = &startTime
	}
	if periodicSchedule.EndTime != nil {
		endTime := v1.NewTime(time.Unix(periodicSchedule.EndTime.Seconds, 0))
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

// Process the job name to remove special char, prepend with "job-" prefix if empty, and
// truncate size to <=25
func toSWFCRDResourceGeneratedName(displayName string) (string, error) {
	const (
		// K8s resource name only allow lower case alphabetic char, number and -
		swfCompatibleNameRegx = "[^a-z0-9-]+"
	)
	reg, err := regexp.Compile(swfCompatibleNameRegx)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to compile ScheduledWorkflow name replacer Regex.")
	}
	processedName := reg.ReplaceAllString(strings.ToLower(displayName), "")
	if processedName == "" {
		processedName = "job-"
	}
	return util.Truncate(processedName, 25), nil
}

func toParametersMap(apiParams []*api.Parameter) map[string]string {
	// Preprocess workflow by appending parameter and add pipeline specific labels
	desiredParamsMap := make(map[string]string)
	for _, param := range apiParams {
		desiredParamsMap[param.Name] = param.Value
	}
	return desiredParamsMap
}
