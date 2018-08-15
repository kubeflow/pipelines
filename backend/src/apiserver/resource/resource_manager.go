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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"regexp"

	"math"

	workflow "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/apiserver/storage"
	"github.com/googleprivate/ml/backend/src/common/util"
	scheduledworkflow "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	scheduledworkflowclient "github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ClientManagerInterface interface {
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	RunStore() storage.RunStoreInterface
	ObjectStore() storage.ObjectStoreInterface
	ScheduledWorkflow() scheduledworkflowclient.ScheduledWorkflowInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
}

type ResourceManager struct {
	pipelineStore     storage.PipelineStoreInterface
	jobStore          storage.JobStoreInterface
	runStore          storage.RunStoreInterface
	objectStore       storage.ObjectStoreInterface
	scheduledWorkflow scheduledworkflowclient.ScheduledWorkflowInterface
	time              util.TimeInterface
	uuid              util.UUIDGeneratorInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		pipelineStore:     clientManager.PipelineStore(),
		jobStore:          clientManager.JobStore(),
		runStore:          clientManager.RunStore(),
		objectStore:       clientManager.ObjectStore(),
		scheduledWorkflow: clientManager.ScheduledWorkflow(),
		time:              clientManager.Time(),
		uuid:              clientManager.UUID(),
	}
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) (pipelines []model.Pipeline, nextPageToken string, err error) {
	return r.pipelineStore.ListPipelines(pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) GetPipeline(pipelineId string) (*model.Pipeline, error) {
	return r.pipelineStore.GetPipeline(pipelineId)
}

func (r *ResourceManager) DeletePipeline(pipelineId string) error {
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return util.Wrap(err, "Delete pipeline failed")
	}

	// Mark pipeline as deleting so it's not visible to user.
	err = r.pipelineStore.UpdatePipelineStatus(pipelineId, model.PipelineDeleting)
	if err != nil {
		return util.Wrap(err, "Delete pipeline failed")
	}

	// Delete pipeline file and DB entry.
	// Not fail the request if this step failed. A background run will do the cleanup.
	// https://github.com/googleprivate/ml/issues/388
	err = r.objectStore.DeleteFile(storage.JobFolder, fmt.Sprint(pipelineId))
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete pipeline file for pipeline %v", pipelineId))
		return nil
	}
	err = r.pipelineStore.DeletePipeline(pipelineId)
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete pipeline DB entry for pipeline %v", pipelineId))
	}
	return nil
}

func (r *ResourceManager) CreatePipeline(name string, pipelineFile []byte) (*model.Pipeline, error) {
	// Extract the parameter from the pipeline
	params, err := util.GetParameters(pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	// Create an entry with status of creating the pipeline
	pipeline := &model.Pipeline{Name: name, Parameters: params, Status: model.PipelineCreating}
	newPipeline, err := r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	// Store the pipeline file
	err = r.objectStore.AddFile(pipelineFile, storage.JobFolder, fmt.Sprint(newPipeline.UUID))
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	newPipeline.Status = model.PipelineReady
	err = r.pipelineStore.UpdatePipelineStatus(newPipeline.UUID, newPipeline.Status)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}
	return newPipeline, nil
}

func (r *ResourceManager) UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error {
	return r.pipelineStore.UpdatePipelineStatus(pipelineId, status)
}

func (r *ResourceManager) GetPipelineTemplate(pipelineId string) ([]byte, error) {
	// Verify pipeline exist
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed")
	}

	template, err := r.objectStore.GetFile(storage.JobFolder, fmt.Sprint(pipelineId))
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed")
	}

	return template, nil
}

func (r *ResourceManager) GetRun(runId string) (*model.RunDetail, error) {
	return r.runStore.GetRun(runId)
}

func (r *ResourceManager) ListRunsV2(pageToken string, pageSize int, sortByFieldName string, isDesc bool) (runs []model.Run, nextPageToken string, err error) {
	return r.runStore.ListRuns(nil, pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) ListRuns(jobId string, pageToken string, pageSize int, sortByFieldName string, isDesc bool) (runs []model.Run, nextPageToken string, err error) {
	_, err = r.jobStore.GetJob(jobId)
	if err != nil {
		return nil, "", util.Wrap(err, "List runs failed")
	}
	return r.runStore.ListRuns(util.StringPointer(jobId), pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) ListJobs(pageToken string, pageSize int, sortByFieldName string, isDesc bool) (jobs []model.Job, nextPageToken string, err error) {
	return r.jobStore.ListJobs(pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) GetJob(id string) (*model.Job, error) {
	return r.jobStore.GetJob(id)
}

func (r *ResourceManager) CreateJob(job *model.Job) (*model.Job, error) {
	var workflow workflow.Workflow
	err := r.objectStore.GetFromYamlFile(&workflow, storage.JobFolder, fmt.Sprint(job.PipelineId))
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	inputParams, err := toCrdParameter(job.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	// Verify no additional parameter provided
	if err = verifyParameters(inputParams, workflow.Spec.Arguments.Parameters); err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: toScheduledWorkflowName(job.DisplayName)},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        job.Enabled,
			MaxConcurrency: &job.MaxConcurrency,
			Trigger: scheduledworkflow.Trigger{
				CronSchedule:     toCrdCronSchedule(job.CronSchedule),
				PeriodicSchedule: toCrdPeriodicSchedule(job.PeriodicSchedule),
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: inputParams,
				Spec:       workflow.Spec,
			},
		},
	}

	newScheduledWorkflow, err := r.scheduledWorkflow.Create(scheduledWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a scheduled workflow for (%s)", scheduledWorkflow.Name)
	}
	job.UUID = string(newScheduledWorkflow.UID)
	job.Name = newScheduledWorkflow.Name
	job.Namespace = newScheduledWorkflow.Namespace
	job.Conditions = util.NewScheduledWorkflow(newScheduledWorkflow).ConditionSummary()
	return r.jobStore.CreateJob(job)
}

func verifyParameters(inputParams []scheduledworkflow.Parameter, templateParams []workflow.Parameter) error {
	templateParamsMap := make(map[string]*string)
	for _, param := range templateParams {
		templateParamsMap[param.Name] = param.Value
	}
	for _, params := range inputParams {
		_, ok := templateParamsMap[params.Name]
		if !ok {
			return util.NewInvalidInputError("Unrecognized input parameter: %v", params.Name)
		}
	}
	return nil
}

func (r *ResourceManager) EnableJob(jobID string, enabled bool) error {
	job, err := r.checkJobExist(jobID)
	if err != nil {
		return util.Wrap(err, "Enable/Disable job failed")
	}
	_, err = r.scheduledWorkflow.Patch(
		job.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"spec":{"enabled":%s}}`, strconv.FormatBool(enabled))))
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to enable/disable job CRD. Enabled: %v, jobID: %v",
			enabled, jobID)
	}

	err = r.jobStore.EnableJob(jobID, enabled)
	if err != nil {
		return util.Wrapf(err, "Failed to enable/disable job. Enabled: %v, jobID: %v",
			enabled, jobID)
	}

	return nil
}

func (r *ResourceManager) DeleteJob(jobID string) error {
	job, err := r.checkJobExist(jobID)
	if err != nil {
		return util.Wrap(err, "Delete job failed")
	}
	err = r.scheduledWorkflow.Delete(job.Name, &v1.DeleteOptions{})
	if err != nil {
		return util.NewInternalServerError(err, "Delete job CRD failed.")
	}
	err = r.jobStore.DeleteJob(jobID)
	if err != nil {
		return util.Wrap(err, "Delete job failed")
	}
	return nil
}

// checkJobExist The Kubernetes API doesn't support CRUD by UID. This method
// retrieve the job metadata from the database, then retrieve the CRD
// using the job name, and compare the given job id is same as the CRD.
func (r *ResourceManager) checkJobExist(jobID string) (*model.Job, error) {
	job, err := r.jobStore.GetJob(jobID)
	if err != nil {
		return nil, util.Wrap(err, "Check job exist failed")
	}
	scheduledWorkflow, err := r.scheduledWorkflow.Get(job.Name, v1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Check job exist failed")
	}
	if scheduledWorkflow == nil || string(scheduledWorkflow.UID) != jobID {
		return nil, util.NewResourceNotFoundError("job", job.Name)
	}
	return job, nil
}

func toCrdCronSchedule(cronSchedule model.CronSchedule) *scheduledworkflow.CronSchedule {
	if cronSchedule.Cron == nil {
		return nil
	}
	crdCronSchedule := scheduledworkflow.CronSchedule{}
	crdCronSchedule.Cron = *cronSchedule.Cron
	if cronSchedule.CronScheduleStartTimeInSec != nil {
		startTime := v1.NewTime(time.Unix(*cronSchedule.CronScheduleStartTimeInSec, 0))
		crdCronSchedule.StartTime = &startTime
	}
	if cronSchedule.CronScheduleEndTimeInSec != nil {
		endTime := v1.NewTime(time.Unix(*cronSchedule.CronScheduleEndTimeInSec, 0))
		crdCronSchedule.EndTime = &endTime
	}
	return &crdCronSchedule
}

func toCrdPeriodicSchedule(periodicSchedule model.PeriodicSchedule) *scheduledworkflow.PeriodicSchedule {
	if periodicSchedule.IntervalSecond == nil {
		return nil
	}
	crdPeriodicSchedule := scheduledworkflow.PeriodicSchedule{}
	crdPeriodicSchedule.IntervalSecond = *periodicSchedule.IntervalSecond
	if periodicSchedule.PeriodicScheduleStartTimeInSec != nil {
		startTime := v1.NewTime(time.Unix(*periodicSchedule.PeriodicScheduleStartTimeInSec, 0))
		crdPeriodicSchedule.StartTime = &startTime
	}
	if periodicSchedule.PeriodicScheduleEndTimeInSec != nil {
		endTime := v1.NewTime(time.Unix(*periodicSchedule.PeriodicScheduleEndTimeInSec, 0))
		crdPeriodicSchedule.EndTime = &endTime
	}
	return &crdPeriodicSchedule
}

func toCrdParameter(paramsString string) ([]scheduledworkflow.Parameter, error) {
	swParams := make([]scheduledworkflow.Parameter, 0)
	var params []workflow.Parameter
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		// The parameter string is from the marshaled job parameter field.
		// Unmarshalling it should never get error. Return internal exception if failed.
		return swParams, util.NewInternalServerError(err, "Failed to parse the parameter CRD")
	}
	for _, param := range params {
		swParam := scheduledworkflow.Parameter{
			Name:  param.Name,
			Value: *param.Value,
		}
		swParams = append(swParams, swParam)
	}
	return swParams, nil
}

// Process the job name to remove special char, prepend with "job-" prefix, and
// truncate size to <=25
func toScheduledWorkflowName(displayName string) string {
	const (
		// K8s resource name only allow lower case alphabetic char, number and -
		swfCompatibleNameRegx = "[^a-z0-9-]+"
	)
	reg := regexp.MustCompile(swfCompatibleNameRegx)
	processedName := "job-" + reg.ReplaceAllString(strings.ToLower(displayName), "")
	return processedName[:int(math.Min(float64(len(processedName)), 25))]
}

func (r *ResourceManager) ReportWorkflowResource(resource string) error {
	var workflow workflow.Workflow
	err := json.Unmarshal([]byte(resource), &workflow)
	if err != nil {
		return util.NewInvalidInputError("Could not unmarshal workflow: %v: %v", err, resource)
	}
	err = r.runStore.CreateOrUpdateRun(util.NewWorkflow(&workflow))
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourceManager) ReportScheduledWorkflowResource(resource string) error {
	var scheduledWorkflow scheduledworkflow.ScheduledWorkflow
	err := json.Unmarshal([]byte(resource), &scheduledWorkflow)
	if err != nil {
		return util.NewInvalidInputError("Could not unmarshal scheduled workflow: %v: %v",
			err, resource)
	}
	err = r.jobStore.UpdateJob(util.NewScheduledWorkflow(&scheduledWorkflow))
	if err != nil {
		return err
	}
	return nil
}
