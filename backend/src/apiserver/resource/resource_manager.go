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

	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
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
	ExperimentStore() storage.ExperimentStoreInterface
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	RunStore() storage.RunStoreInterface
	ObjectStore() storage.ObjectStoreInterface
	Workflow() workflowclient.WorkflowInterface
	ScheduledWorkflow() scheduledworkflowclient.ScheduledWorkflowInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
}

type ResourceManager struct {
	experimentStore         storage.ExperimentStoreInterface
	pipelineStore           storage.PipelineStoreInterface
	jobStore                storage.JobStoreInterface
	runStore                storage.RunStoreInterface
	objectStore             storage.ObjectStoreInterface
	workflowClient          workflowclient.WorkflowInterface
	scheduledWorkflowClient scheduledworkflowclient.ScheduledWorkflowInterface
	time                    util.TimeInterface
	uuid                    util.UUIDGeneratorInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		experimentStore:         clientManager.ExperimentStore(),
		pipelineStore:           clientManager.PipelineStore(),
		jobStore:                clientManager.JobStore(),
		runStore:                clientManager.RunStore(),
		objectStore:             clientManager.ObjectStore(),
		workflowClient:          clientManager.Workflow(),
		scheduledWorkflowClient: clientManager.ScheduledWorkflow(),
		time:                    clientManager.Time(),
		uuid:                    clientManager.UUID(),
	}
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) CreateExperiment(experiment *model.Experiment) (*model.Experiment, error) {
	return r.experimentStore.CreateExperiment(experiment)
}

func (r *ResourceManager) GetExperiment(experimentId string) (*model.Experiment, error) {
	return r.experimentStore.GetExperiment(experimentId)
}

func (r *ResourceManager) ListExperiments(context *common.PaginationContext) (
	experiments []model.Experiment, nextPageToken string, err error) {
	return r.experimentStore.ListExperiments(context)
}

func (r *ResourceManager) ListPipelines(context *common.PaginationContext) (
	pipelines []model.Pipeline, nextPageToken string, err error) {
	return r.pipelineStore.ListPipelines(context)
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
	err = r.objectStore.DeleteFile(storage.PipelineFolder, fmt.Sprint(pipelineId))
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
	err = r.objectStore.AddFile(pipelineFile, storage.PipelineFolder, fmt.Sprint(newPipeline.UUID))
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

	template, err := r.objectStore.GetFile(storage.PipelineFolder, fmt.Sprint(pipelineId))
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed")
	}

	return template, nil
}

func (r *ResourceManager) CreateRun(apiRun *api.Run) (*model.RunDetail, error) {
	// Get workflow from pipeline spec, which might be pipeline ID or an argo workflow
	workflowSpecManifestBytes, err := r.getWorkflowSpecBytes(apiRun.GetPipelineSpec())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch workflow spec.")
	}
	var workflow util.Workflow
	err = json.Unmarshal(workflowSpecManifestBytes, &workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow bytes: %s", string(workflowSpecManifestBytes))
	}

	parameters := toParametersMap(apiRun.GetPipelineSpec().GetParameters())
	// Append provided parameter
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	workflow.OverrideParameters(parameters)

	// Create argo workflow CRD resource
	newWorkflow, err := r.workflowClient.Create(workflow.Get())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a workflow for (%s)", workflow.Name)
	}

	// Store run metadata into database
	run, err := ToModelRun(apiRun, string(workflowSpecManifestBytes))
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert run model")
	}

	run.UUID = string(newWorkflow.UID)
	run.Name = newWorkflow.Name
	run.Namespace = newWorkflow.Namespace
	run.Conditions = util.NewWorkflow(newWorkflow).Condition()
	run.CreatedAtInSec = r.time.Now().Unix()

	runDetail := &model.RunDetail{Run: *run}
	workflowRuntimeManifest, err := json.Marshal(newWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Unable to marshal a workflow: %+v", newWorkflow)
	}
	runDetail.WorkflowRuntimeManifest = string(workflowRuntimeManifest)
	// TODO: store the resource reference
	return r.runStore.CreateRun(runDetail)
}

func (r *ResourceManager) GetRun(runId string) (*model.RunDetail, error) {
	return r.runStore.GetRun(runId)
}

func (r *ResourceManager) ListRunsV2(context *common.PaginationContext) (runs []model.Run, nextPageToken string, err error) {
	return r.runStore.ListRuns(nil, context)
}

func (r *ResourceManager) ListRuns(jobId string, context *common.PaginationContext) (runs []model.Run, nextPageToken string, err error) {
	_, err = r.jobStore.GetJob(jobId)
	if err != nil {
		return nil, "", util.Wrap(err, "List runs failed")
	}
	return r.runStore.ListRuns(util.StringPointer(jobId), context)
}

func (r *ResourceManager) ListJobs(context *common.PaginationContext) (jobs []model.Job, nextPageToken string, err error) {
	return r.jobStore.ListJobs(context)
}

func (r *ResourceManager) GetJob(id string) (*model.Job, error) {
	return r.jobStore.GetJob(id)
}

// TODO create resource reference for experiment-> job.
func (r *ResourceManager) CreateJob(apiJob *api.Job) (*model.Job, error) {
	job, err := ToModelJob(apiJob)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	// Verify pipeline exist
	_, err = r.pipelineStore.GetPipeline(job.PipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline failed")
	}

	var workflow util.Workflow
	err = r.objectStore.GetFromYamlFile(&workflow, storage.PipelineFolder, job.PipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	// Verify no additional parameter provided
	err = workflow.VerifyParameters(toParametersMap(apiJob.GetParameters()))
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(job.DisplayName)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled: job.Enabled,
			Trigger: scheduledworkflow.Trigger{
				CronSchedule:     toCrdCronSchedule(job.CronSchedule),
				PeriodicSchedule: toCRDPeriodicSchedule(job.PeriodicSchedule),
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCRDParameter(apiJob.GetParameters()),
				Spec:       workflow.Spec,
			},
		},
	}
	if job.MaxConcurrency != 0 {
		scheduledWorkflow.Spec.MaxConcurrency = &job.MaxConcurrency
	}

	newScheduledWorkflow, err := r.scheduledWorkflowClient.Create(scheduledWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a scheduled workflow for (%s)", scheduledWorkflow.Name)
	}
	job.UUID = string(newScheduledWorkflow.UID)
	job.Name = newScheduledWorkflow.Name
	job.Namespace = newScheduledWorkflow.Namespace
	job.Conditions = util.NewScheduledWorkflow(newScheduledWorkflow).ConditionSummary()
	now := r.time.Now().Unix()
	job.CreatedAtInSec = now
	job.UpdatedAtInSec = now
	return r.jobStore.CreateJob(job)
}

func (r *ResourceManager) EnableJob(jobID string, enabled bool) error {
	job, err := r.checkJobExist(jobID)
	if err != nil {
		return util.Wrap(err, "Enable/Disable job failed")
	}
	_, err = r.scheduledWorkflowClient.Patch(
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
	err = r.scheduledWorkflowClient.Delete(job.Name, &v1.DeleteOptions{})
	if err != nil {
		return util.NewInternalServerError(err, "Delete job CRD failed.")
	}
	err = r.jobStore.DeleteJob(jobID)
	if err != nil {
		return util.Wrap(err, "Delete job failed")
	}
	return nil
}

func (r *ResourceManager) ReportWorkflowResource(workflow *util.Workflow) error {
	return r.runStore.ReportRun(workflow)
}

func (r *ResourceManager) ReportScheduledWorkflowResource(swf *util.ScheduledWorkflow) error {
	return r.jobStore.UpdateJob(swf)
}

// checkJobExist The Kubernetes API doesn't support CRUD by UID. This method
// retrieve the job metadata from the database, then retrieve the CRD
// using the job name, and compare the given job id is same as the CRD.
func (r *ResourceManager) checkJobExist(jobID string) (*model.Job, error) {
	job, err := r.jobStore.GetJob(jobID)
	if err != nil {
		return nil, util.Wrap(err, "Check job exist failed")
	}
	scheduledWorkflow, err := r.scheduledWorkflowClient.Get(job.Name, v1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Check job exist failed")
	}
	if scheduledWorkflow == nil || string(scheduledWorkflow.UID) != jobID {
		return nil, util.NewResourceNotFoundError("job", job.Name)
	}
	return job, nil
}

func (r *ResourceManager) getWorkflowSpecBytes(spec *api.PipelineSpec) ([]byte, error) {
	if spec.GetPipelineId() != "" {
		// Verify pipeline exist
		_, err := r.pipelineStore.GetPipeline(spec.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Get pipeline failed")
		}
		workflowYAMLBytes, err := r.objectStore.GetFile(storage.PipelineFolder, spec.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Get pipeline YAML failed.")
		}
		workflowJsonBytes, err := yaml.YAMLToJSON(workflowYAMLBytes)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to convert workflow to JSON. Workflow: %s", string(workflowYAMLBytes))
		}
		return workflowJsonBytes, nil
	} else if spec.GetWorkflowManifest() != "" {
		return []byte(spec.GetWorkflowManifest()), nil
	}
	return nil, util.NewInvalidInputError("Please provide a valid pipeline spec")
}

func (r *ResourceManager) ReportMetric(metric *api.RunMetric, runUUID string) error {
	return r.runStore.ReportMetric(ToModelRunMetric(metric, runUUID))
}
