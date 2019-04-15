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

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	scheduledworkflowclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ClientManagerInterface interface {
	ExperimentStore() storage.ExperimentStoreInterface
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	RunStore() storage.RunStoreInterface
	ResourceReferenceStore() storage.ResourceReferenceStoreInterface
	DBStatusStore() storage.DBStatusStoreInterface
	DefaultExperimentStore() storage.DefaultExperimentStoreInterface
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
	resourceReferenceStore  storage.ResourceReferenceStoreInterface
	dBStatusStore           storage.DBStatusStoreInterface
	defaultExperimentStore  storage.DefaultExperimentStoreInterface
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
		resourceReferenceStore:  clientManager.ResourceReferenceStore(),
		dBStatusStore:           clientManager.DBStatusStore(),
		defaultExperimentStore:  clientManager.DefaultExperimentStore(),
		objectStore:             clientManager.ObjectStore(),
		workflowClient:          clientManager.Workflow(),
		scheduledWorkflowClient: clientManager.ScheduledWorkflow(),
		time: clientManager.Time(),
		uuid: clientManager.UUID(),
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

func (r *ResourceManager) ListExperiments(opts *list.Options) (
	experiments []*model.Experiment, total_size int, nextPageToken string, err error) {
	return r.experimentStore.ListExperiments(opts)
}

func (r *ResourceManager) DeleteExperiment(experimentID string) error {
	_, err := r.experimentStore.GetExperiment(experimentID)
	if err != nil {
		return util.Wrap(err, "Delete experiment failed")
	}
	return r.experimentStore.DeleteExperiment(experimentID)
}

func (r *ResourceManager) ListPipelines(opts *list.Options) (
	pipelines []*model.Pipeline, total_size int, nextPageToken string, err error) {
	return r.pipelineStore.ListPipelines(opts)
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
	// https://github.com/kubeflow/pipelines/issues/388
	err = r.objectStore.DeleteFile(storage.CreatePipelinePath(fmt.Sprint(pipelineId)))
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

func (r *ResourceManager) CreatePipeline(name string, description string, pipelineFile []byte) (*model.Pipeline, error) {
	// Extract the parameter from the pipeline
	params, err := util.GetParameters(pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	// Create an entry with status of creating the pipeline
	pipeline := &model.Pipeline{Name: name, Description: description, Parameters: params, Status: model.PipelineCreating}
	newPipeline, err := r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	// Store the pipeline file
	err = r.objectStore.AddFile(pipelineFile, storage.CreatePipelinePath(fmt.Sprint(newPipeline.UUID)))
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

	template, err := r.objectStore.GetFile(storage.CreatePipelinePath(fmt.Sprint(pipelineId)))
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
	if err = json.Unmarshal(workflowSpecManifestBytes, &workflow); err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow bytes: %s", string(workflowSpecManifestBytes))
	}

	parameters := toParametersMap(apiRun.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err = workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	// Create argo workflow CRD resource
	newWorkflow, err := r.workflowClient.Create(workflow.Get())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a workflow for (%s)", workflow.Name)
	}

	// Add a reference to the default experiment if run does not already have a containing experiment
	err = r.setExperimentIfNotPresent(apiRun)
	if err != nil {
		return nil, err
	}

	// Store run metadata into database
	runDetail, err := ToModelRunDetail(apiRun, util.NewWorkflow(newWorkflow), string(workflowSpecManifestBytes))
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert run model")
	}

	// Assign the create at time.
	runDetail.CreatedAtInSec = r.time.Now().Unix()
	return r.runStore.CreateRun(runDetail)
}

func (r *ResourceManager) GetRun(runId string) (*model.RunDetail, error) {
	return r.runStore.GetRun(runId)
}

func (r *ResourceManager) ListRuns(filterContext *common.FilterContext,
	opts *list.Options) (runs []*model.Run, total_size int, nextPageToken string, err error) {
	return r.runStore.ListRuns(filterContext, opts)
}

func (r *ResourceManager) ArchiveRun(runId string) error {
	return r.runStore.ArchiveRun(runId)
}

func (r *ResourceManager) UnarchiveRun(runId string) error {
	return r.runStore.UnarchiveRun(runId)
}

func (r *ResourceManager) DeleteRun(runID string) error {
	runDetail, err := r.checkRunExist(runID)
	if err != nil {
		return util.Wrap(err, "Delete run failed")
	}
	err = r.workflowClient.Delete(runDetail.Name, &v1.DeleteOptions{})
	if err != nil {
		// API won't need to delete the workflow CRD
		// once persistent agent sync the state to DB and set TTL for it.
		glog.Warningf("Failed to delete run %v. Error: %v", runDetail.Name, err.Error())
	}
	err = r.runStore.DeleteRun(runID)
	if err != nil {
		return util.Wrap(err, "Delete run failed")
	}
	return nil
}

func (r *ResourceManager) ListJobs(filterContext *common.FilterContext,
	opts *list.Options) (jobs []*model.Job, total_size int, nextPageToken string, err error) {
	return r.jobStore.ListJobs(filterContext, opts)
}

// TerminateWorkflow terminates a workflow by setting its activeDeadlineSeconds to 0
func TerminateWorkflow(wfClient workflowclient.WorkflowInterface, name string) error {
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"activeDeadlineSeconds": 0,
		},
	}

	patch, err := json.Marshal(patchObj)
	if err != nil {
		return util.NewInternalServerError(err, "Unexpected error while marshalling a patch object.")
	}

	var operation = func() error {
		_, err = wfClient.Patch(name, types.MergePatchType, patch)
		return err
	}
	var backoffPolicy = backoff.WithMaxRetries(backoff.NewConstantBackOff(100), 10)
	err = backoff.Retry(operation, backoffPolicy)
	return err
}

func (r *ResourceManager) TerminateRun(runId string) error {
	runDetail, err := r.checkRunExist(runId)
	if err != nil {
		return util.Wrap(err, "Delete run failed")
	}

	err = r.runStore.TerminateRun(runId)
	if err != nil {
		return util.Wrap(err, "Terminate run failed")
	}

	err = TerminateWorkflow(r.workflowClient, runDetail.Run.Name)
	return util.Wrap(err, "Terminate run failed")
}

func (r *ResourceManager) GetJob(id string) (*model.Job, error) {
	return r.jobStore.GetJob(id)
}

func (r *ResourceManager) CreateJob(apiJob *api.Job) (*model.Job, error) {
	// Get workflow from pipeline spec, which might be pipeline ID or an argo workflow
	workflowSpecManifestBytes, err := r.getWorkflowSpecBytes(apiJob.GetPipelineSpec())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch workflow spec.")
	}
	var workflow util.Workflow
	err = json.Unmarshal(workflowSpecManifestBytes, &workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow bytes: %s", string(workflowSpecManifestBytes))
	}

	// Verify no additional parameter provided
	err = workflow.VerifyParameters(toParametersMap(apiJob.PipelineSpec.Parameters))
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        apiJob.Enabled,
			MaxConcurrency: &apiJob.MaxConcurrency,
			Trigger:        *toCRDTrigger(apiJob.Trigger),
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCRDParameter(apiJob.PipelineSpec.Parameters),
				Spec:       workflow.Spec,
			},
		},
	}
	newScheduledWorkflow, err := r.scheduledWorkflowClient.Create(scheduledWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a scheduled workflow for (%s)", scheduledWorkflow.Name)
	}
	job, err := ToModelJob(apiJob, util.NewScheduledWorkflow(newScheduledWorkflow), string(workflowSpecManifestBytes))
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

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
	runId := string(workflow.UID)
	jobId := workflow.ScheduledWorkflowUUIDAsStringOrEmpty()

	if jobId == "" {
		// If a run doesn't have owner UID, it's a one-time run created by Pipeline API server.
		// In this case the DB entry should already been created when argo workflow CRD is created.
		return r.runStore.UpdateRun(runId, workflow.Condition(), workflow.FinishedAt(), workflow.ToStringForStore())
	}

	// Get the experiment resource reference for job.
	experimentRef, err := r.resourceReferenceStore.GetResourceReference(jobId, common.Job, common.Experiment)
	if err != nil {
		return util.Wrap(err, "Failed to retrieve the experiment ID for the job that created the run.")
	}
	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             runId,
			DisplayName:      workflow.Name,
			Name:             workflow.Name,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Namespace:        workflow.Namespace,
			CreatedAtInSec:   workflow.CreationTimestamp.Unix(),
			ScheduledAtInSec: workflow.ScheduledAtInSecOr0(),
			FinishedAtInSec:  workflow.FinishedAt(),
			Conditions:       workflow.Condition(),
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: workflow.GetSpec().ToStringForStore(),
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  string(workflow.UID),
					ResourceType:  common.Run,
					ReferenceUUID: jobId,
					ReferenceType: common.Job,
					Relationship:  common.Creator,
				},
				{
					ResourceUUID:  string(workflow.UID),
					ResourceType:  common.Run,
					ReferenceUUID: experimentRef.ReferenceUUID,
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
		},
	}
	return r.runStore.CreateOrUpdateRun(runDetail)
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

// checkRunExist The Kubernetes API doesn't support CRUD by UID. This method
// retrieve the run metadata from the database, then retrieve the CRD
// using the run name, and compare the given run id is same as the CRD.
func (r *ResourceManager) checkRunExist(runID string) (*model.RunDetail, error) {
	runDetail, err := r.runStore.GetRun(runID)
	if err != nil {
		return nil, util.Wrap(err, "Check run exist failed")
	}
	return runDetail, nil
}

func (r *ResourceManager) getWorkflowSpecBytes(spec *api.PipelineSpec) ([]byte, error) {
	if spec.GetPipelineId() != "" {
		var workflow util.Workflow
		err := r.objectStore.GetFromYamlFile(&workflow, storage.CreatePipelinePath(spec.GetPipelineId()))
		if err != nil {
			return nil, util.Wrap(err, "Get pipeline YAML failed.")
		}

		return []byte(workflow.ToStringForStore()), nil
	} else if spec.GetWorkflowManifest() != "" {
		return []byte(spec.GetWorkflowManifest()), nil
	}
	return nil, util.NewInvalidInputError("Please provide a valid pipeline spec")
}

// setDefaultExperimentIfNotPresent If the provided run does not include a reference to a containing
// experiment, then we fetch the default experiment's ID and create a reference to that.
func (r *ResourceManager) setExperimentIfNotPresent(apiRun *api.Run) error {
	// First check if there is already a referenced experiment
	for _, ref := range apiRun.ResourceReferences {
		if ref.Key.Type == api.ResourceType_EXPERIMENT && ref.Relationship == api.Relationship_OWNER {
			return nil
		}
	}

	// Create reference to the default experiment
	defaultExperimentId, err := r.GetDefaultExperimentId()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to retrieve default experiment")
	}
	if defaultExperimentId == "" {
		return util.NewInternalServerError(errors.New("Default experiment ID was empty"), "Default experiment was not set")
	}
	defaultExperimentRef := &api.ResourceReference{
		Key: &api.ResourceKey{
			Id:   defaultExperimentId,
			Type: api.ResourceType_EXPERIMENT,
		},
		Relationship: api.Relationship_OWNER,
	}
	apiRun.ResourceReferences = append(apiRun.ResourceReferences, defaultExperimentRef)

	return nil
}

func (r *ResourceManager) ReportMetric(metric *api.RunMetric, runUUID string) error {
	return r.runStore.ReportMetric(ToModelRunMetric(metric, runUUID))
}

// ReadArtifact parses run's workflow to find artifact file path and reads the content of the file
// from object store.
func (r *ResourceManager) ReadArtifact(runID string, nodeID string, artifactName string) ([]byte, error) {
	run, err := r.runStore.GetRun(runID)
	if err != nil {
		return nil, err
	}
	var storageWorkflow workflowapi.Workflow
	err = json.Unmarshal([]byte(run.WorkflowRuntimeManifest), &storageWorkflow)
	if err != nil {
		// This should never happen.
		return nil, util.NewInternalServerError(
			err, "failed to unmarshal workflow '%s'", run.WorkflowRuntimeManifest)
	}
	workflow := util.NewWorkflow(&storageWorkflow)
	artifactPath := workflow.FindObjectStoreArtifactKeyOrEmpty(nodeID, artifactName)
	if artifactPath == "" {
		return nil, util.NewResourceNotFoundError(
			"arifact", common.CreateArtifactPath(runID, nodeID, artifactName))
	}
	return r.objectStore.GetFile(artifactPath)
}

func (r *ResourceManager) GetDefaultExperimentId() (string, error) {
	return r.defaultExperimentStore.GetDefaultExperimentId()
}

func (r *ResourceManager) SetDefaultExperimentId(id string) error {
	return r.defaultExperimentStore.SetDefaultExperimentId(id)
}

func (r *ResourceManager) HaveSamplesLoaded() (bool, error) {
	return r.dBStatusStore.HaveSamplesLoaded()
}

func (r *ResourceManager) MarkSampleLoaded() error {
	return r.dBStatusStore.MarkSampleLoaded()
}
