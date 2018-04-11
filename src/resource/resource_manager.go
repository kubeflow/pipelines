package resource

import (
	"fmt"
	"ml/src/message"
	"ml/src/storage"
	"ml/src/util"
	"time"

	"github.com/golang/glog"
	"github.com/robfig/cron"
)

type ClientManagerInterface interface {
	PackageStore()   storage.PackageStoreInterface
	PipelineStore()  storage.PipelineStoreInterface
	JobStore()       storage.JobStoreInterface
	PackageManager() storage.PackageManagerInterface
	Time()           util.TimeInterface
}

type ResourceManager struct {
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
	time           util.TimeInterface
}

func NewResourceManager(clientManager ClientManagerInterface) *ResourceManager {
	return &ResourceManager{
		packageStore: clientManager.PackageStore(),
		pipelineStore: clientManager.PipelineStore(),
		jobStore: clientManager.JobStore(),
		packageManager: clientManager.PackageManager(),
		time: clientManager.Time(),
	}
}

func (r *ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r *ResourceManager) ListPackages() ([]message.Package, error) {
	return r.packageStore.ListPackages()
}

func (r *ResourceManager) GetPackage(packageId uint) (*message.Package, error) {
	return r.packageStore.GetPackage(packageId)
}

func (r *ResourceManager) CreatePackage(p *message.Package) error {
	return r.packageStore.CreatePackage(p)
}

func (r *ResourceManager) CreatePackageFile(template []byte, fileName string) error {
	return r.packageManager.CreatePackageFile(template, fileName)
}

func (r *ResourceManager) GetTemplate(pkgName string) ([]byte, error) {
	return r.packageManager.GetTemplate(pkgName)
}

func (r *ResourceManager) ListPipelines() ([]message.Pipeline, error) {
	return r.pipelineStore.ListPipelines()
}

func (r *ResourceManager) GetPipeline(id uint) (*message.Pipeline, error) {
	return r.pipelineStore.GetPipeline(id)
}

func (r *ResourceManager) CreatePipeline(pipeline *message.Pipeline) error {
	// Verify the package exists
	pkg, err := r.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		return err
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
			return error
		}
	}

	// Create pipeline metadata
	err = r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return err
	}

	// If there is no pipeline schedule, the job is created immediately.
	if pipeline.Schedule == "" {

		_, err := r.createJobFromPipeline(pipeline, pkg.Name, time.Now().Unix())
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceManager) CreateJobFromPipelineID(pipelineID uint, scheduledAtInSec int64) (
	*message.JobDetail, error) {
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

func (r *ResourceManager) createJobFromPipeline(pipeline *message.Pipeline, pkgName string,
	scheduledAtInSec int64) (*message.JobDetail, error) {
	template, err := r.packageManager.GetTemplate(pkgName)
	if err != nil {
		return nil, err
	}

	// Inject parameters user provided to the pipeline template.
	workflow, err := util.InjectParameters(template, pipeline.Parameters)
	if err != nil {
		return nil, err
	}

	jobDetail, err := r.jobStore.CreateJob(pipeline.ID, workflow, scheduledAtInSec)
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

func (r *ResourceManager) GetJob(pipelineId uint, jobName string) (*message.JobDetail, error) {
	return r.jobStore.GetJob(pipelineId, jobName)
}
func (r *ResourceManager) ListJobs(pipelineId uint) ([]message.Job, error) {
	return r.jobStore.ListJobs(pipelineId)
}

func (r *ResourceManager) GetPipelineAndLatestJobIterator() (*storage.PipelineAndLatestJobIterator,
	error) {
	return r.pipelineStore.GetPipelineAndLatestJobIterator()
}
