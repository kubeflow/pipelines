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
	PackageStore() storage.PackageStoreInterface
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	ObjectStore() storage.ObjectStoreInterface
	ScheduledWorkflow() scheduledworkflowclient.ScheduledWorkflowInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
}

type ResourceManager struct {
	packageStore      storage.PackageStoreInterface
	pipelineStore     storage.PipelineStoreInterface
	jobStore          storage.JobStoreInterface
	objectStore       storage.ObjectStoreInterface
	scheduledWorkflow scheduledworkflowclient.ScheduledWorkflowInterface
	time              util.TimeInterface
	uuid              util.UUIDGeneratorInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		packageStore:      clientManager.PackageStore(),
		pipelineStore:     clientManager.PipelineStore(),
		jobStore:          clientManager.JobStore(),
		objectStore:       clientManager.ObjectStore(),
		scheduledWorkflow: clientManager.ScheduledWorkflow(),
		time:              clientManager.Time(),
		uuid:              clientManager.UUID(),
	}
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) ListPackages(pageToken string, pageSize int, sortByFieldName string, isDesc bool) (pkgs []model.Package, nextPageToken string, err error) {
	return r.packageStore.ListPackages(pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) GetPackage(packageId uint32) (*model.Package, error) {
	return r.packageStore.GetPackage(packageId)
}

func (r *ResourceManager) DeletePackage(packageId uint32) error {
	_, err := r.packageStore.GetPackage(packageId)
	if err != nil {
		return util.Wrap(err, "Delete package failed")
	}

	// Mark package as deleting so it's not visible to user.
	err = r.packageStore.UpdatePackageStatus(packageId, model.PackageDeleting)
	if err != nil {
		return util.Wrap(err, "Delete package failed")
	}

	// Delete package file and DB entry.
	// Not fail the request if this step failed. A background job will do the cleanup.
	// https://github.com/googleprivate/ml/issues/388
	err = r.objectStore.DeleteFile(storage.PackageFolder, fmt.Sprint(packageId))
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete package file for package %v", packageId))
		return nil
	}
	err = r.packageStore.DeletePackage(packageId)
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete package DB entry for package %v", packageId))
	}
	return nil
}

func (r *ResourceManager) CreatePackage(name string, pkgFile []byte) (*model.Package, error) {
	// Extract the parameter from the package
	params, err := util.GetParameters(pkgFile)
	if err != nil {
		return nil, util.Wrap(err, "Create package failed")
	}

	// Create an entry with status of creating the package
	pkg := &model.Package{Name: name, Parameters: params, Status: model.PackageCreating}
	newPkg, err := r.packageStore.CreatePackage(pkg)
	if err != nil {
		return nil, util.Wrap(err, "Create package failed")
	}

	// Store the package file
	err = r.objectStore.AddFile(pkgFile, storage.PackageFolder, fmt.Sprint(newPkg.ID))
	if err != nil {
		return nil, util.Wrap(err, "Create package failed")
	}

	newPkg.Status = model.PackageReady
	err = r.packageStore.UpdatePackageStatus(newPkg.ID, newPkg.Status)
	if err != nil {
		return nil, util.Wrap(err, "Create package failed")
	}
	return newPkg, nil
}

func (r *ResourceManager) UpdatePackageStatus(packageId uint32, status model.PackageStatus) error {
	return r.packageStore.UpdatePackageStatus(packageId, status)
}

func (r *ResourceManager) GetPackageTemplate(packageId uint32) ([]byte, error) {
	// Verify package exist
	_, err := r.packageStore.GetPackage(packageId)
	if err != nil {
		return nil, util.Wrap(err, "Get package template failed")
	}

	template, err := r.objectStore.GetFile(storage.PackageFolder, fmt.Sprint(packageId))
	if err != nil {
		return nil, util.Wrap(err, "Get package template failed")
	}

	return template, nil
}

func (r *ResourceManager) GetJob(pipelineId string, jobId string) (*model.JobDetail, error) {
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Get job failed")
	}
	return r.jobStore.GetJob(pipelineId, jobId)
}

func (r *ResourceManager) ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string, isDesc bool) (jobs []model.Job, nextPageToken string, err error) {
	_, err = r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed")
	}
	return r.jobStore.ListJobs(pipelineId, pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) (pipelines []model.Pipeline, nextPageToken string, err error) {
	return r.pipelineStore.ListPipelines(pageToken, pageSize, sortByFieldName, isDesc)
}

func (r *ResourceManager) GetPipeline(id string) (*model.Pipeline, error) {
	return r.pipelineStore.GetPipeline(id)
}

func (r *ResourceManager) CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error) {
	var workflow workflow.Workflow
	err := r.objectStore.GetFromYamlFile(&workflow, storage.PackageFolder, fmt.Sprint(pipeline.PackageId))
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: toScheduledWorkflowName(pipeline.DisplayName)},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        pipeline.Enabled,
			MaxConcurrency: &pipeline.MaxConcurrency,
			Trigger: scheduledworkflow.Trigger{
				CronSchedule:     toCrdCronSchedule(pipeline.CronSchedule),
				PeriodicSchedule: toCrdPeriodicSchedule(pipeline.PeriodicSchedule),
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCrdParameter(pipeline.Parameters),
				Spec:       workflow.Spec,
			},
		},
	}

	newScheduledWorkflow, err := r.scheduledWorkflow.Create(scheduledWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a scheduled workflow for (%s)", scheduledWorkflow.Name)
	}
	pipeline.UUID = string(newScheduledWorkflow.UID)
	pipeline.Name = newScheduledWorkflow.Name
	pipeline.Namespace = newScheduledWorkflow.Namespace
	pipeline.Conditions = util.NewScheduledWorkflow(newScheduledWorkflow).ConditionSummary()
	return r.pipelineStore.CreatePipeline(pipeline)
}

func (r *ResourceManager) EnablePipeline(pipelineID string, enabled bool) error {
	pipeline, err := r.checkPipelineExist(pipelineID)
	if err != nil {
		return util.Wrap(err, "Enable/Disable pipeline failed")
	}
	_, err = r.scheduledWorkflow.Patch(
		pipeline.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"spec":{"enabled":%s}}`, strconv.FormatBool(enabled))))
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to enable/disable pipeline CRD. Enabled: %v, pipelineID: %v",
			enabled, pipelineID)
	}

	err = r.pipelineStore.EnablePipeline(pipelineID, enabled)
	if err != nil {
		return util.Wrapf(err, "Failed to enable/disable pipeline. Enabled: %v, pipelineID: %v",
			enabled, pipelineID)
	}

	return nil
}

func (r *ResourceManager) DeletePipeline(pipelineID string) error {
	pipeline, err := r.checkPipelineExist(pipelineID)
	if err != nil {
		return util.Wrap(err, "Delete pipeline failed")
	}
	err = r.scheduledWorkflow.Delete(pipeline.Name, &v1.DeleteOptions{})
	if err != nil {
		return util.NewInternalServerError(err, "Delete pipeline CRD failed.")
	}
	err = r.pipelineStore.DeletePipeline(pipelineID)
	if err != nil {
		return util.Wrap(err, "Delete pipeline failed")
	}
	return nil
}

// checkPipelineExist The Kubernetes API doesn't support CRUD by UID. This method
// retrieve the pipeline metadata from the database, then retrieve the CRD
// using the pipeline name, and compare the given pipeline id is same as the CRD.
func (r *ResourceManager) checkPipelineExist(pipelineID string) (*model.Pipeline, error) {
	pipeline, err := r.pipelineStore.GetPipeline(pipelineID)
	if err != nil {
		return nil, util.Wrap(err, "Check pipeline exist failed")
	}
	scheduledWorkflow, err := r.scheduledWorkflow.Get(pipeline.Name, v1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Check pipeline exist failed")
	}
	if scheduledWorkflow == nil || string(scheduledWorkflow.UID) != pipelineID {
		return nil, util.NewResourceNotFoundError("pipeline", pipeline.Name)
	}
	return pipeline, nil
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

func toCrdParameter(paramsString string) []scheduledworkflow.Parameter {
	swParams := make([]scheduledworkflow.Parameter, 0)
	var params []workflow.Parameter
	json.Unmarshal([]byte(paramsString), &params)
	for _, param := range params {
		swParam := scheduledworkflow.Parameter{
			Name:  param.Name,
			Value: *param.Value,
		}
		swParams = append(swParams, swParam)
	}
	return swParams
}

// Process the pipeline name to remove special char, prepend with "pipeline-" prefix, and
// truncate size to <=25
func toScheduledWorkflowName(displayName string) string {
	const (
		// K8s resource name only allow lower case alphabetic char, number and -
		swfCompatibleNameRegx = "[^a-z0-9-]+"
	)
	reg := regexp.MustCompile(swfCompatibleNameRegx)
	processedName := "pipeline-" + reg.ReplaceAllString(strings.ToLower(displayName), "")
	return processedName[:int(math.Min(float64(len(processedName)), 25))]
}

func (r *ResourceManager) ReportWorkflowResource(resource string) error {
	var workflow workflow.Workflow
	err := json.Unmarshal([]byte(resource), &workflow)
	if err != nil {
		return util.NewInvalidInputError("Could not unmarshal workflow: %v: %v", err, resource)
	}
	err = r.jobStore.UpdateJob(util.NewWorkflow(&workflow))
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
	err = r.pipelineStore.UpdatePipeline(util.NewScheduledWorkflow(&scheduledWorkflow))
	if err != nil {
		return err
	}
	return nil
}
