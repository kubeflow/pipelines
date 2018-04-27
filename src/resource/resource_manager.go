package resource

import (
	"fmt"
	"ml/src/model"
	"ml/src/storage"
	"ml/src/util"

	"github.com/golang/glog"
	"github.com/robfig/cron"
)

type ClientManagerInterface interface {
	PackageStore() storage.PackageStoreInterface
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	PackageManager() storage.PackageManagerInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
}

type ResourceManager struct {
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
	time           util.TimeInterface
	uuid           util.UUIDGeneratorInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		packageStore:   clientManager.PackageStore(),
		pipelineStore:  clientManager.PipelineStore(),
		jobStore:       clientManager.JobStore(),
		packageManager: clientManager.PackageManager(),
		time:           clientManager.Time(),
		uuid:           clientManager.UUID(),
	}
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) ListPackages() ([]model.Package, error) {
	return r.packageStore.ListPackages()
}

func (r *ResourceManager) GetPackage(packageId uint) (*model.Package, error) {
	return r.packageStore.GetPackage(packageId)
}

func (r *ResourceManager) CreatePackage(p *model.Package) (*model.Package, error) {
	return r.packageStore.CreatePackage(p)
}

func (r *ResourceManager) CreatePackageFile(template []byte, fileName string) error {
	return r.packageManager.CreatePackageFile(template, fileName)
}

func (r *ResourceManager) GetTemplate(pkgName string) ([]byte, error) {
	return r.packageManager.GetTemplate(pkgName)
}

func (r *ResourceManager) ListPipelines() ([]model.Pipeline, error) {
	return r.pipelineStore.ListPipelines()
}

func (r *ResourceManager) GetPipeline(id uint) (*model.Pipeline, error) {
	return r.pipelineStore.GetPipeline(id)
}

func (r *ResourceManager) DeletePipeline(id uint) error {
	return r.pipelineStore.DeletePipeline(id)
}

func (r *ResourceManager) CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error) {
	// Verify the package exists
	pkg, err := r.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		return nil, err
	}

	// If the pipeline runs on a schedule
	if pipeline.Schedule != "" {
		// Validate the pipeline schedule.
		_, err := cron.Parse(pipeline.Schedule)
		if err != nil {
			error := util.NewInvalidInputError(
				err,
				fmt.Sprintf("The pipeline schedule cannot be parsed: %s: %s", pipeline.Schedule, err),
				err.Error())
			return nil, error
		}
	}

	// Create pipeline metadata
	pipeline, err = r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	// If there is no pipeline schedule, the job is created immediately.
	if pipeline.Schedule == "" {
		_, err := r.createJobFromPipeline(pipeline, pkg.Name, r.time.Now().Unix())
		if err != nil {
			return nil, err
		}
	}

	return pipeline, nil
}

func (r *ResourceManager) CreateJobFromPipelineID(pipelineID uint, scheduledAtInSec int64) (
	*model.JobDetail, error) {
	// Get the pipeline.
	pipeline, err := r.pipelineStore.GetPipeline(pipelineID)
	if err != nil {
		return nil, util.Wrapf(err, "Could not get pipeline from pipeline ID: %v", pipelineID)
	}

	// Get the package.
	// TODO: should the package be indexed by the package primary key in the object store?
	// https://github.com/googleprivate/ml/issues/274
	pkg, err := r.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		return nil, util.Wrapf(err, "Could not get the package from the package ID: %v",
			pipeline.PackageId)
	}

	// Create the job.
	jobDetail, err := r.createJobFromPipeline(pipeline, pkg.Name, scheduledAtInSec)
	if err != nil {
		return nil, util.Wrapf(err, "Could not create a job for pipeline %v and package %v.",
			pipeline.ID, pkg.Name)
	}

	return jobDetail, nil
}

func (r *ResourceManager) createJobFromPipeline(pipeline *model.Pipeline, pkgName string,
	scheduledAtInSec int64) (*model.JobDetail, error) {
	template, err := r.packageManager.GetTemplate(pkgName)
	if err != nil {
		return nil, err
	}

	// Inject parameters user provided to the pipeline template.
	workflow, err := util.InjectParameters(template, pipeline.Parameters)
	if err != nil {
		return nil, err
	}

	// Define the time at which the workflow is created in the DB. The same time should be stored
	// in the DB and substituted in the workflow parameters (i.e. time.Now() should not be
	// called multiple times.
	createdAtInSec := r.time.Now().Unix()

	// Format the parameters of the workflow and the workflow name
	formatter := util.NewWorkflowFormatter(r.uuid, scheduledAtInSec, createdAtInSec)
	formatter.Format(workflow)

	// Create job.
	jobDetail, err := r.jobStore.CreateJob(pipeline.ID, workflow, scheduledAtInSec, createdAtInSec)
	if err != nil {
		return nil, err
	}

	glog.Infof("Successfully created job %v for pipeline %v and package %v.", jobDetail.Workflow.Name,
		pipeline.ID, pkgName)

	return jobDetail, nil
}

func (r *ResourceManager) EnablePipeline(pipelineID uint, enabled bool) error {
	// Note: no validation needed.
	err := r.pipelineStore.EnablePipeline(pipelineID, enabled)
	if err != nil {
		return util.Wrapf(err, "Failed to enable/disable pipeline. Enabled: %v, pipelineID: %v",
			enabled, pipelineID)
	}

	return nil
}

func (r *ResourceManager) GetJob(pipelineId uint, jobName string) (*model.JobDetail, error) {
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Get job failed")
	}
	return r.jobStore.GetJob(pipelineId, jobName)
}

func (r *ResourceManager) ListJobs(pipelineId uint) ([]model.Job, error) {
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "List jobs failed")
	}
	return r.jobStore.ListJobs(pipelineId)
}

func (r *ResourceManager) GetPipelineAndLatestJobIterator() (*storage.PipelineAndLatestJobIterator,
	error) {
	return r.pipelineStore.GetPipelineAndLatestJobIterator()
}
