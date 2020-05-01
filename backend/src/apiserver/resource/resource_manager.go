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
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
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

const (
	defaultPipelineRunnerServiceAccount = "pipeline-runner"
	HasDefaultBucketEnvVar              = "HAS_DEFAULT_BUCKET"
	ProjectIDEnvVar                     = "PROJECT_ID"
	DefaultBucketNameEnvVar             = "BUCKET_NAME"
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
	ArgoClient() client.ArgoClientInterface
	SwfClient() client.SwfClientInterface
	KubernetesCoreClient() client.KubernetesCoreInterface
	KFAMClient() client.KFAMClientInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
}

type ResourceManager struct {
	experimentStore        storage.ExperimentStoreInterface
	pipelineStore          storage.PipelineStoreInterface
	jobStore               storage.JobStoreInterface
	runStore               storage.RunStoreInterface
	resourceReferenceStore storage.ResourceReferenceStoreInterface
	dBStatusStore          storage.DBStatusStoreInterface
	defaultExperimentStore storage.DefaultExperimentStoreInterface
	objectStore            storage.ObjectStoreInterface
	argoClient             client.ArgoClientInterface
	swfClient              client.SwfClientInterface
	k8sCoreClient          client.KubernetesCoreInterface
	kfamClient             client.KFAMClientInterface
	time                   util.TimeInterface
	uuid                   util.UUIDGeneratorInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		experimentStore:        clientManager.ExperimentStore(),
		pipelineStore:          clientManager.PipelineStore(),
		jobStore:               clientManager.JobStore(),
		runStore:               clientManager.RunStore(),
		resourceReferenceStore: clientManager.ResourceReferenceStore(),
		dBStatusStore:          clientManager.DBStatusStore(),
		defaultExperimentStore: clientManager.DefaultExperimentStore(),
		objectStore:            clientManager.ObjectStore(),
		argoClient:             clientManager.ArgoClient(),
		swfClient:              clientManager.SwfClient(),
		k8sCoreClient:          clientManager.KubernetesCoreClient(),
		kfamClient:             clientManager.KFAMClient(),
		time:                   clientManager.Time(),
		uuid:                   clientManager.UUID(),
	}
}

func (r *ResourceManager) getWorkflowClient(namespace string) workflowclient.WorkflowInterface {
	return r.argoClient.Workflow(namespace)
}

func (r *ResourceManager) getScheduledWorkflowClient(namespace string) scheduledworkflowclient.ScheduledWorkflowInterface {
	return r.swfClient.ScheduledWorkflow(namespace)
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) CreateExperiment(apiExperiment *api.Experiment) (*model.Experiment, error) {
	experiment, err := r.ToModelExperiment(apiExperiment)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert experiment model")
	}
	return r.experimentStore.CreateExperiment(experiment)
}

func (r *ResourceManager) GetExperiment(experimentId string) (*model.Experiment, error) {
	return r.experimentStore.GetExperiment(experimentId)
}

func (r *ResourceManager) ListExperiments(filterContext *common.FilterContext, opts *list.Options) (
	experiments []*model.Experiment, total_size int, nextPageToken string, err error) {
	return r.experimentStore.ListExperiments(filterContext, opts)
}

func (r *ResourceManager) DeleteExperiment(experimentID string) error {
	_, err := r.experimentStore.GetExperiment(experimentID)
	if err != nil {
		return util.Wrap(err, "Delete experiment failed")
	}
	return r.experimentStore.DeleteExperiment(experimentID)
}

func (r *ResourceManager) ArchiveExperiment(experimentId string) error {
	// To archive an experiment
	// (1) update our persistent agent to disable CRDs of jobs in experiment
	// (2) update database to
	// (2.1) archive experiemnts
	// (2.2) archive runs
	// (2.3) disable jobs
	opts, err := list.NewOptions(&model.Job{}, 50, "name", nil)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create list jobs options when archiving experiment. ")
	}
	for {
		jobs, _, newToken, err := r.jobStore.ListJobs(&common.FilterContext{
			ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: experimentId}}, opts)
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to list jobs of to-be-archived experiment. expID: %v", experimentId)
		}
		for _, job := range jobs {
			_, err = r.getScheduledWorkflowClient(job.Namespace).Patch(
				job.Name,
				types.MergePatchType,
				[]byte(fmt.Sprintf(`{"spec":{"enabled":%s}}`, strconv.FormatBool(false))))
			if err != nil {
				return util.NewInternalServerError(err,
					"Failed to disable job CRD. jobID: %v", job.UUID)
			}
		}
		if newToken == "" {
			break
		} else {
			opts, err = list.NewOptionsFromToken(newToken, 50)
			if err != nil {
				return util.NewInternalServerError(err,
					"Failed to create list jobs options from page token when archiving experiment. ")
			}
		}
	}
	return r.experimentStore.ArchiveExperiment(experimentId)
}

func (r *ResourceManager) UnarchiveExperiment(experimentId string) error {
	return r.experimentStore.UnarchiveExperiment(experimentId)
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
	// TODO(jingzhang36): For now (before exposing version API), we have only 1
	// file with both pipeline and version pointing to it;  so it is ok to do
	// the deletion as follows. After exposing version API, we can have multiple
	// versions and hence multiple files, and we shall improve performance by
	// either using async deletion in order for this method to be non-blocking
	// or or exploring other performance optimization tools provided by gcs.
	err = r.objectStore.DeleteFile(r.objectStore.GetPipelineKey(fmt.Sprint(pipelineId)))
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
	pipeline := &model.Pipeline{
		Name:        name,
		Description: description,
		Parameters:  params,
		Status:      model.PipelineCreating,
		DefaultVersion: &model.PipelineVersion{
			Name:       name,
			Parameters: params,
			Status:     model.PipelineVersionCreating}}
	newPipeline, err := r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	// Store the pipeline file to a path dependent on pipeline version
	err = r.objectStore.AddFile(pipelineFile,
		r.objectStore.GetPipelineKey(fmt.Sprint(newPipeline.DefaultVersion.UUID)))
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}

	newPipeline.Status = model.PipelineReady
	newPipeline.DefaultVersion.Status = model.PipelineVersionReady
	err = r.pipelineStore.UpdatePipelineAndVersionsStatus(
		newPipeline.UUID,
		newPipeline.Status,
		newPipeline.DefaultVersionId,
		newPipeline.DefaultVersion.Status)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed")
	}
	return newPipeline, nil
}

func (r *ResourceManager) UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error {
	return r.pipelineStore.UpdatePipelineStatus(pipelineId, status)
}

func (r *ResourceManager) UpdatePipelineVersionStatus(pipelineId string, status model.PipelineVersionStatus) error {
	return r.pipelineStore.UpdatePipelineVersionStatus(pipelineId, status)
}

func (r *ResourceManager) GetPipelineTemplate(pipelineId string) ([]byte, error) {
	// Verify pipeline exist
	pipeline, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed")
	}

	if pipeline.DefaultVersion == nil {
		return nil, util.Wrap(err,
			"Get pipeline template failed since no default version is defined")
	}
	template, err := r.objectStore.GetFile(r.objectStore.GetPipelineKey(fmt.Sprint(pipeline.DefaultVersion.UUID)))
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed")
	}

	return template, nil
}

func (r *ResourceManager) CreateRun(apiRun *api.Run) (*model.RunDetail, error) {
	// Get workflow from either of the two places:
	// (1) raw pipeline manifest in pipeline_spec
	// (2) pipeline version in resource_references
	// And the latter takes priority over the former
	var workflowSpecManifestBytes []byte
	err := ConvertPipelineIdToDefaultPipelineVersion(apiRun.PipelineSpec, &apiRun.ResourceReferences, r)
	if err != nil {
		return nil, util.Wrap(err, "Failed to find default version to create run with pipeline id.")
	}
	workflowSpecManifestBytes, err = r.getWorkflowSpecBytesFromPipelineVersion(apiRun.GetResourceReferences())
	if err != nil {
		workflowSpecManifestBytes, err = r.getWorkflowSpecBytesFromPipelineSpec(apiRun.GetPipelineSpec())
		if err != nil {
			return nil, util.Wrap(err, "Failed to fetch workflow spec.")
		}
	}
	uuid, err := r.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to generate run ID.")
	}
	runId := uuid.String()

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

	r.setDefaultServiceAccount(&workflow, apiRun.GetServiceAccount())

	// Disable istio sidecar injection
	workflow.SetAnnotationsToAllTemplates(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	// TODO: Add run_level flag with similar logic by reading flag value from create_run api.
	workflow.SetLabelsToAllTemplates(util.LabelKeyCacheEnabled, common.IsCacheEnabled())
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	err = OverrideParameterWithSystemDefault(workflow, apiRun)
	if err != nil {
		return nil, err
	}

	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(util.LabelKeyWorkflowRunId, runId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(util.AnnotationKeyRunName, apiRun.Name)
	// Replace {{workflow.uid}} with runId
	err = workflow.ReplaceUID(runId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to replace workflow ID")
	}

	// Marking auto-added artifacts as optional. Otherwise most older workflows will start failing after upgrade to Argo 2.3.
	// TODO: Fix the components to explicitly declare the artifacts they really output.
	for templateIdx, template := range workflow.Workflow.Spec.Templates {
		for artIdx, artifact := range template.Outputs.Artifacts {
			if artifact.Name == "mlpipeline-ui-metadata" || artifact.Name == "mlpipeline-metrics" {
				workflow.Workflow.Spec.Templates[templateIdx].Outputs.Artifacts[artIdx].Optional = true
			}
		}
	}

	// Add a reference to the default experiment if run does not already have a containing experiment
	ref, err := r.getDefaultExperimentIfNoExperiment(apiRun.GetResourceReferences())
	if err != nil {
		return nil, err
	}
	if ref != nil {
		apiRun.ResourceReferences = append(apiRun.GetResourceReferences(), ref)
	}

	namespace, err := r.getNamespaceFromExperiment(apiRun.GetResourceReferences())
	if err != nil {
		return nil, err
	}

	// Create argo workflow CRD resource
	newWorkflow, err := r.getWorkflowClient(namespace).Create(workflow.Get())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a workflow for (%s)", workflow.Name)
	}

	// Store run metadata into database
	runDetail, err := r.ToModelRunDetail(apiRun, runId, util.NewWorkflow(newWorkflow), string(workflowSpecManifestBytes))
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
	namespace, err := r.GetNamespaceFromRunID(runID)
	if err != nil {
		return util.Wrap(err, "Delete run failed")
	}
	err = r.getWorkflowClient(namespace).Delete(runDetail.Name, &v1.DeleteOptions{})
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
	opts *list.Options) (jobs []*model.Job, _ int, nextPageToken string, err error) {
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
		return util.Wrap(err, "Terminate run failed")
	}

	namespace, err := r.GetNamespaceFromRunID(runId)
	if err != nil {
		return util.Wrap(err, "Terminate run failed")
	}

	err = r.runStore.TerminateRun(runId)
	if err != nil {
		return util.Wrap(err, "Terminate run failed")
	}

	err = TerminateWorkflow(r.getWorkflowClient(namespace), runDetail.Run.Name)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to terminate the run")
	}
	return nil
}

func (r *ResourceManager) RetryRun(runId string) error {
	runDetail, err := r.checkRunExist(runId)
	if err != nil {
		return util.Wrap(err, "Retry run failed")
	}
	namespace, err := r.GetNamespaceFromRunID(runId)
	if err != nil {
		return util.Wrap(err, "Retry run failed")
	}

	if runDetail.WorkflowRuntimeManifest == "" {
		return util.NewBadRequestError(errors.New("workflow cannot be retried"), "Workflow must be Failed/Error to retry")
	}
	var workflow util.Workflow
	if err := json.Unmarshal([]byte(runDetail.WorkflowRuntimeManifest), &workflow); err != nil {
		return util.NewInternalServerError(err, "Failed to retrieve the runtime pipeline spec from the run")
	}

	newWorkflow, podsToDelete, err := formulateRetryWorkflow(&workflow)
	if err != nil {
		return util.Wrap(err, "Retry run failed.")
	}

	if err = deletePods(r.k8sCoreClient, podsToDelete, namespace); err != nil {
		return util.NewInternalServerError(err, "Retry run failed. Failed to clean up the failed pods from previous run.")
	}

	// First try to update workflow
	updateError := r.updateWorkflow(newWorkflow, namespace)
	if updateError != nil {
		// Remove resource version
		newWorkflow.ResourceVersion = ""
		newCreatedWorkflow, createError := r.getWorkflowClient(namespace).Create(newWorkflow.Workflow)
		if createError != nil {
			return util.NewInternalServerError(createError,
				"Retry run failed. Failed to create or update the run. Update Error: %s, Create Error: %s",
				updateError.Error(), createError.Error())
		}
		newWorkflow = util.NewWorkflow(newCreatedWorkflow)
	}
	err = r.runStore.UpdateRun(runId, newWorkflow.Condition(), 0, newWorkflow.ToStringForStore())
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the database entry.")
	}
	return nil
}

func (r *ResourceManager) updateWorkflow(newWorkflow *util.Workflow, namespace string) error {
	// If fail to get the workflow, return error.
	latestWorkflow, err := r.getWorkflowClient(namespace).Get(newWorkflow.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	// Update the workflow's resource version to latest.
	newWorkflow.ResourceVersion = latestWorkflow.ResourceVersion
	_, err = r.getWorkflowClient(namespace).Update(newWorkflow.Workflow)
	return err
}

func (r *ResourceManager) GetJob(id string) (*model.Job, error) {
	return r.jobStore.GetJob(id)
}

func (r *ResourceManager) CreateJob(apiJob *api.Job) (*model.Job, error) {
	// Get workflow from either of the two places:
	// (1) raw pipeline manifest in pipeline_spec
	// (2) pipeline version in resource_references
	// And the latter takes priority over the former
	var workflowSpecManifestBytes []byte
	err := ConvertPipelineIdToDefaultPipelineVersion(apiJob.PipelineSpec, &apiJob.ResourceReferences, r)
	if err != nil {
		return nil, util.Wrap(err, "Failed to find default version to create job with pipeline id.")
	}
	workflowSpecManifestBytes, err = r.getWorkflowSpecBytesFromPipelineVersion(apiJob.GetResourceReferences())
	if err != nil {
		workflowSpecManifestBytes, err = r.getWorkflowSpecBytesFromPipelineSpec(apiJob.GetPipelineSpec())
		if err != nil {
			return nil, util.Wrap(err, "Failed to fetch workflow spec.")
		}
	}

	var workflow util.Workflow
	err = json.Unmarshal(workflowSpecManifestBytes, &workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow bytes: %s", string(workflowSpecManifestBytes))
	}

	// Verify no additional parameter provided
	err = workflow.VerifyParameters(toParametersMap(apiJob.GetPipelineSpec().GetParameters()))
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}

	r.setDefaultServiceAccount(&workflow, apiJob.GetServiceAccount())

	// Disable istio sidecar injection
	workflow.SetAnnotationsToAllTemplates(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)

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
				Parameters: toCRDParameter(apiJob.GetPipelineSpec().GetParameters()),
				Spec:       workflow.Spec,
			},
			NoCatchup: util.BoolPointer(apiJob.NoCatchup),
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

	// Add a reference to the default experiment if run does not already have a containing experiment
	ref, err := r.getDefaultExperimentIfNoExperiment(apiJob.GetResourceReferences())
	if err != nil {
		return nil, err
	}
	if ref != nil {
		apiJob.ResourceReferences = append(apiJob.GetResourceReferences(), ref)
	}

	namespace, err := r.getNamespaceFromExperiment(apiJob.GetResourceReferences())
	if err != nil {
		return nil, err
	}

	newScheduledWorkflow, err := r.getScheduledWorkflowClient(namespace).Create(scheduledWorkflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a scheduled workflow for (%s)", scheduledWorkflow.Name)
	}

	job, err := r.ToModelJob(apiJob, util.NewScheduledWorkflow(newScheduledWorkflow), string(workflowSpecManifestBytes))
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

	_, err = r.getScheduledWorkflowClient(job.Namespace).Patch(
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

	err = r.getScheduledWorkflowClient(job.Namespace).Delete(job.Name, &v1.DeleteOptions{})
	if err != nil {
		return util.NewInternalServerError(err, "Delete job CRD failed.")
	}
	err = r.jobStore.DeleteJob(jobID)
	if err != nil {
		return util.Wrap(err, "Delete job failed")
	}
	return nil
}

func (r *ResourceManager) UpdateJob(apiJob *api.Job) (err error) {
	defer func() {
		err = util.Wrapf(err, "update job %q failed", apiJob.Name)
	}()

	job, err := r.checkJobExist(apiJob.Id)
	if err != nil {
		return err
	}

	cli := r.getScheduledWorkflowClient(job.Namespace)
	swf, err := cli.Get(job.Name, v1.GetOptions{})
	if err != nil {
		return err
	}

	w := util.Workflow{Workflow: &workflowapi.Workflow{
		Spec: swf.Spec.Workflow.Spec,
	}}

	err = w.VerifyParameters(toParametersMap(apiJob.GetPipelineSpec().GetParameters()))
	if err != nil {
		return util.Wrap(err, "failed to verify new params")
	}

	swf.Spec.MaxConcurrency = &apiJob.MaxConcurrency
	swf.Spec.NoCatchup = &apiJob.NoCatchup
	swf.Spec.Trigger = *toCRDTrigger(apiJob.Trigger)
	swf.Spec.Workflow.Parameters = toCRDParameter(apiJob.GetPipelineSpec().GetParameters())

	if len(apiJob.GetServiceAccount()) > 0 {
		swf.Spec.Workflow.Spec.ServiceAccountName = apiJob.GetServiceAccount()
	}

	job.DisplayName = apiJob.Name
	job.Description = apiJob.Description

	err = job.UpdateWorkflow(&util.ScheduledWorkflow{ScheduledWorkflow: swf})
	if err != nil {
		return err
	}

	_, err = cli.Update(swf)
	if err != nil {
		return err
	}

	return r.jobStore.UpdateJob(job)
}

func (r *ResourceManager) ReportWorkflowResource(workflow *util.Workflow) error {
	if _, ok := workflow.ObjectMeta.Labels[util.LabelKeyWorkflowRunId]; !ok {
		// Skip reporting if the workflow doesn't have the run id label
		return util.NewInvalidInputError("Workflow missing the Run ID label")
	}
	runId := workflow.ObjectMeta.Labels[util.LabelKeyWorkflowRunId]
	jobId := workflow.ScheduledWorkflowUUIDAsStringOrEmpty()
	if len(workflow.Namespace) == 0 {
		return util.NewInvalidInputError("Workflow missing namespace")
	}

	if workflow.PersistedFinalState() {
		// If workflow's final state has being persisted, the workflow should be garbage collected.
		err := r.getWorkflowClient(workflow.Namespace).Delete(workflow.Name, &v1.DeleteOptions{})
		if err != nil {
			return util.NewInternalServerError(err, "Failed to delete the completed workflow for run %s", runId)
		}
	}

	if jobId == "" {
		// If a run doesn't have job ID, it's a one-time run created by Pipeline API server.
		// In this case the DB entry should already been created when argo workflow CRD is created.

		err := r.runStore.UpdateRun(runId, workflow.Condition(), workflow.FinishedAt(), workflow.ToStringForStore())
		if err != nil {
			return util.Wrap(err, "Failed to update the run.")
		}
	} else {
		// Get the experiment resource reference for job.
		experimentRef, err := r.resourceReferenceStore.GetResourceReference(jobId, common.Job, common.Experiment)
		if err != nil {
			return util.Wrap(err, "Failed to retrieve the experiment ID for the job that created the run.")
		}
		jobName, err := r.getResourceName(common.Job, jobId)
		if err != nil {
			return util.Wrap(err, "Failed to retrieve the job name for the job that created the run.")
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
					WorkflowSpecManifest: workflow.GetWorkflowSpec().ToStringForStore(),
				},
				ResourceReferences: []*model.ResourceReference{
					{
						ResourceUUID:  runId,
						ResourceType:  common.Run,
						ReferenceUUID: jobId,
						ReferenceName: jobName,
						ReferenceType: common.Job,
						Relationship:  common.Creator,
					},
					{
						ResourceUUID:  runId,
						ResourceType:  common.Run,
						ReferenceUUID: experimentRef.ReferenceUUID,
						ReferenceName: experimentRef.ReferenceName,
						ReferenceType: common.Experiment,
						Relationship:  common.Owner,
					},
				},
			},
			PipelineRuntime: model.PipelineRuntime{
				WorkflowRuntimeManifest: workflow.ToStringForStore(),
			},
		}
		err = r.runStore.CreateOrUpdateRun(runDetail)
		if err != nil {
			return util.Wrap(err, "Failed to create or update the run.")
		}
	}

	if workflow.IsInFinalState() {
		err := AddWorkflowLabel(r.getWorkflowClient(workflow.Namespace), workflow.Name, util.LabelKeyWorkflowPersistedFinalState, "true")
		if err != nil {
			return util.Wrap(err, "Failed to add PersistedFinalState label to workflow")
		}
	}

	return nil
}

// AddWorkflowLabel add label for a workflow
func AddWorkflowLabel(wfClient workflowclient.WorkflowInterface, name string, labelKey string, labelValue string) error {
	patchObj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				labelKey: labelValue,
			},
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

func (r *ResourceManager) ReportScheduledWorkflowResource(swf *util.ScheduledWorkflow) error {
	job, err := r.checkJobExist(string(swf.UID))
	if err != nil {
		return util.Wrapf(err, "failed to check job %q exists", swf.Name)
	}

	err = job.UpdateWorkflow(swf)
	if err != nil {
		return util.Wrapf(err, "failed to update job %q workflow", job.Name)
	}

	return r.jobStore.UpdateJob(job)
}

// checkJobExist The Kubernetes API doesn't support CRUD by UID. This method
// retrieve the job metadata from the database, then retrieve the CRD
// using the job name, and compare the given job id is same as the CRD.
func (r *ResourceManager) checkJobExist(jobID string) (*model.Job, error) {
	job, err := r.jobStore.GetJob(jobID)
	if err != nil {
		return nil, util.Wrap(err, "Check job exist failed")
	}

	scheduledWorkflow, err := r.getScheduledWorkflowClient(job.Namespace).Get(job.Name, v1.GetOptions{})
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

func (r *ResourceManager) getWorkflowSpecBytesFromPipelineSpec(spec *api.PipelineSpec) ([]byte, error) {
	if spec.GetWorkflowManifest() != "" {
		return []byte(spec.GetWorkflowManifest()), nil
	}
	return nil, util.NewInvalidInputError("Please provide a valid pipeline spec")
}

func (r *ResourceManager) getWorkflowSpecBytesFromPipelineVersion(references []*api.ResourceReference) ([]byte, error) {
	var pipelineVersionId = ""
	for _, reference := range references {
		if reference.Key.Type == api.ResourceType_PIPELINE_VERSION && reference.Relationship == api.Relationship_CREATOR {
			pipelineVersionId = reference.Key.Id
		}
	}
	if len(pipelineVersionId) == 0 {
		return nil, util.NewInvalidInputError("No pipeline version.")
	}
	var workflow util.Workflow
	err := r.objectStore.GetFromYamlFile(&workflow, r.objectStore.GetPipelineKey(pipelineVersionId))
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline YAML failed.")
	}

	return []byte(workflow.ToStringForStore()), nil
}

// Used to initialize the Experiment database with a default to be used for runs
func (r *ResourceManager) CreateDefaultExperiment() (string, error) {
	// First check that we don't already have a default experiment ID in the DB.
	defaultExperimentId, err := r.GetDefaultExperimentId()
	if err != nil {
		return "", fmt.Errorf("Failed to check if default experiment exists. Err: %v", err)
	}
	// If default experiment ID is already present, don't fail, simply return.
	if defaultExperimentId != "" {
		glog.Infof("Default experiment already exists! ID: %v", defaultExperimentId)
		return "", nil
	}

	// Create default experiment
	defaultExperiment := &api.Experiment{
		Name:        "Default",
		Description: "All runs created without specifying an experiment will be grouped here.",
	}
	experiment, err := r.CreateExperiment(defaultExperiment)
	if err != nil {
		return "", fmt.Errorf("Failed to create default experiment. Err: %v", err)
	}

	// Set default experiment ID in the DB
	err = r.SetDefaultExperimentId(experiment.UUID)
	if err != nil {
		return "", fmt.Errorf("Failed to set default experiment ID. Err: %v", err)
	}

	glog.Infof("Default experiment is set. ID is: %v", experiment.UUID)
	return experiment.UUID, nil
}

// getDefaultExperimentIfNoExperiment If the provided run does not include a reference to a containing
// experiment, then we fetch the default experiment's ID and create a reference to that.
func (r *ResourceManager) getDefaultExperimentIfNoExperiment(references []*api.ResourceReference) (*api.ResourceReference, error) {
	// First check if there is already a referenced experiment
	for _, ref := range references {
		if ref.Key.Type == api.ResourceType_EXPERIMENT && ref.Relationship == api.Relationship_OWNER {
			return nil, nil
		}
	}
	if common.IsMultiUserMode() {
		return nil, util.NewInvalidInputError("Experiment is required in resource references.")
	}
	return r.getDefaultExperimentResourceReference(references)
}

func (r *ResourceManager) getDefaultExperimentResourceReference(references []*api.ResourceReference) (*api.ResourceReference, error) {
	// Create reference to the default experiment
	defaultExperimentId, err := r.GetDefaultExperimentId()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to retrieve default experiment")
	}
	if defaultExperimentId == "" {
		glog.Info("No default experiment was found. Creating a new default experiment")
		defaultExperimentId, err = r.CreateDefaultExperiment()
		if defaultExperimentId == "" || err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create new default experiment")
		}
	}
	defaultExperimentRef := &api.ResourceReference{
		Key: &api.ResourceKey{
			Id:   defaultExperimentId,
			Type: api.ResourceType_EXPERIMENT,
		},
		Relationship: api.Relationship_OWNER,
	}

	return defaultExperimentRef, nil
}

func (r *ResourceManager) ReportMetric(metric *api.RunMetric, runUUID string) error {
	return r.runStore.ReportMetric(r.ToModelRunMetric(metric, runUUID))
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
			"artifact", common.CreateArtifactPath(runID, nodeID, artifactName))
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

func (r *ResourceManager) getDefaultSA() string {
	return common.GetStringConfigWithDefault(common.DefaultPipelineRunnerServiceAccount, defaultPipelineRunnerServiceAccount)
}

func (r *ResourceManager) CreatePipelineVersion(apiVersion *api.PipelineVersion, pipelineFile []byte) (*model.PipelineVersion, error) {
	// Extract the parameters from the pipeline
	params, err := util.GetParameters(pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline version failed")
	}

	// Extract pipeline id
	var pipelineId = ""
	for _, resourceReference := range apiVersion.ResourceReferences {
		if resourceReference.Key.Type == api.ResourceType_PIPELINE && resourceReference.Relationship == api.Relationship_OWNER {
			pipelineId = resourceReference.Key.Id
		}
	}
	if len(pipelineId) == 0 {
		return nil, util.Wrap(err, "Create pipeline version failed due to missing pipeline id")
	}

	// Construct model.PipelineVersion
	version := &model.PipelineVersion{
		Name:          apiVersion.Name,
		PipelineId:    pipelineId,
		Status:        model.PipelineVersionCreating,
		Parameters:    params,
		CodeSourceUrl: apiVersion.CodeSourceUrl,
	}
	version, err = r.pipelineStore.CreatePipelineVersion(version)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline version failed")
	}

	// Store the pipeline file
	err = r.objectStore.AddFile(pipelineFile, r.objectStore.GetPipelineKey(fmt.Sprint(version.UUID)))
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline version failed")
	}

	// After pipeline version being created in DB and pipeline file being
	// saved in minio server, set this pieline version to status ready.
	version.Status = model.PipelineVersionReady
	err = r.pipelineStore.UpdatePipelineVersionStatus(version.UUID, version.Status)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline version failed")
	}

	return version, nil
}

func (r *ResourceManager) GetPipelineVersion(versionId string) (*model.PipelineVersion, error) {
	return r.pipelineStore.GetPipelineVersion(versionId)
}

func (r *ResourceManager) ListPipelineVersions(pipelineId string, opts *list.Options) (pipelines []*model.PipelineVersion, total_size int, nextPageToken string, err error) {
	return r.pipelineStore.ListPipelineVersions(pipelineId, opts)
}

func (r *ResourceManager) DeletePipelineVersion(pipelineVersionId string) error {
	_, err := r.pipelineStore.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return util.Wrap(err, "Delete pipeline version failed")
	}

	// Mark pipeline as deleting so it's not visible to user.
	err = r.pipelineStore.UpdatePipelineVersionStatus(pipelineVersionId, model.PipelineVersionDeleting)
	if err != nil {
		return util.Wrap(err, "Delete pipeline version failed")
	}

	err = r.objectStore.DeleteFile(r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersionId)))
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete pipeline file for pipeline version %v", pipelineVersionId))
		return util.Wrap(err, "Delete pipeline version failed")
	}
	err = r.pipelineStore.DeletePipelineVersion(pipelineVersionId)
	if err != nil {
		glog.Errorf("%v", errors.Wrapf(err, "Failed to delete pipeline DB entry for pipeline %v", pipelineVersionId))
		return util.Wrap(err, "Delete pipeline version failed")
	}

	return nil
}

func (r *ResourceManager) GetPipelineVersionTemplate(versionId string) ([]byte, error) {
	// Verify pipeline version exist
	_, err := r.pipelineStore.GetPipelineVersion(versionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline version template failed")
	}

	template, err := r.objectStore.GetFile(r.objectStore.GetPipelineKey(fmt.Sprint(versionId)))
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline version template failed")
	}

	return template, nil
}

func (r *ResourceManager) IsRequestAuthorized(userIdentity string, namespace string) (bool, error) {
	return r.kfamClient.IsAuthorized(userIdentity, namespace)
}

func (r *ResourceManager) GetNamespaceFromExperimentID(experimentID string) (string, error) {
	experiment, err := r.GetExperiment(experimentID)
	if err != nil {
		return "", util.Wrap(err, "Failed to get namespace from experiment ID.")
	}
	return experiment.Namespace, nil
}

func (r *ResourceManager) GetNamespaceFromRunID(runId string) (string, error) {
	runDetail, err := r.GetRun(runId)
	if err != nil {
		return "", util.Wrap(err, "Failed to get namespace from run id.")
	}
	return runDetail.Namespace, nil
}

func (r *ResourceManager) GetNamespaceFromJobID(jobId string) (string, error) {
	job, err := r.GetJob(jobId)
	if err != nil {
		return "", util.Wrap(err, "Failed to get namespace from Job ID.")
	}
	return job.Namespace, nil
}

func (r *ResourceManager) setDefaultServiceAccount(workflow *util.Workflow, serviceAccount string) {
	if len(serviceAccount) > 0 {
		workflow.SetServiceAccount(serviceAccount)
		return
	}
	workflowServiceAccount := workflow.Spec.ServiceAccountName
	if len(workflowServiceAccount) == 0 || workflowServiceAccount == defaultPipelineRunnerServiceAccount {
		// To reserve SDK backward compatibility, the backend only replaces
		// serviceaccount when it is empty or equal to default value set by SDK.
		workflow.SetServiceAccount(r.getDefaultSA())
	}
}

func (r *ResourceManager) getNamespaceFromExperiment(references []*api.ResourceReference) (string, error) {
	experimentID := common.GetExperimentIDFromAPIResourceReferences(references)
	experiment, err := r.GetExperiment(experimentID)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to get experiment.")
	}

	namespace := experiment.Namespace
	if len(namespace) == 0 {
		if common.IsMultiUserMode() {
			return "", util.NewInternalServerError(errors.New("Missing namespace"), "Experiment %v doesn't have a namespace.", experiment.Name)
		} else {
			namespace = common.GetPodNamespace()
		}
	}
	return namespace, nil
}
