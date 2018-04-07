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

type ResourceManager struct {
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
	time           util.TimeInterface
}

func NewResourceManager(
	packageStore storage.PackageStoreInterface,
	pipelineStore storage.PipelineStoreInterface,
	jobStore storage.JobStoreInterface,
	packageManager storage.PackageManagerInterface,
	time util.TimeInterface) *ResourceManager {
	return &ResourceManager{
		packageStore:   packageStore,
		pipelineStore:  pipelineStore,
		jobStore:       jobStore,
		packageManager: packageManager,
		time:           time,
	}
}

func NewResourceManagerTestOnly(store *storage.FakeStore) *ResourceManager {
	return NewResourceManager(store.PackageStore, store.PipelineStore, store.JobStore,
		store.PackageManager, store.Time)
}

func (r ResourceManager) GetPackageStore() storage.PackageStoreInterface {
	return r.packageStore
}

func (r ResourceManager) GetPipelineStore() storage.PipelineStoreInterface {
	return r.pipelineStore
}

func (r ResourceManager) GetJobStore() storage.JobStoreInterface {
	return r.jobStore
}

func (r ResourceManager) GetPackageManager() storage.PackageManagerInterface {
	return r.packageManager
}

func (r ResourceManager) GetTime() util.TimeInterface {
	return r.time
}

func (r ResourceManager) CreatePipeline(pipeline *message.Pipeline) (error, string) {
	// Verify the package exists
	pkg, err := r.packageStore.GetPackage(pipeline.PackageId)
	if err != nil {
		return err, "CreatePipeline_ValidPackageExist"
	}

	// If the pipeline runs on a schedule
	if pipeline.Schedule != "" {
		// Validate the pipeline schedule.
		_, err := cron.Parse(pipeline.Schedule)
		if err != nil {
			error := util.NewInvalidInputError(
				fmt.Sprintf("The pipeline schedule cannot be parsed: %s: %s", pipeline.Schedule, err),
				err.Error())
			return error, "CreatePipeline_ValidSchedule"
		}
	}

	// Create pipeline metadata
	err = r.pipelineStore.CreatePipeline(pipeline)
	if err != nil {
		return err, "CreatePipeline"
	}

	// If there is no pipeline schedule, the job is created immediately.
	if pipeline.Schedule == "" {

		_, err := r.createJobFromPipeline(pipeline, pkg.Name, time.Now().Unix())
		if err != nil {
			return err, "CreatePipeline_CreateJob"
		}
	}

	return nil, ""
}

func (r ResourceManager) CreateJobFromPipelineID(pipelineID uint, scheduledAtInSec int64) (
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

func (r ResourceManager) createJobFromPipeline(pipeline *message.Pipeline, pkgName string,
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

func (r ResourceManager) EnablePipeline(pipelineID uint, enabled bool) error {
	// Note: no validation needed.
	err := r.pipelineStore.EnablePipeline(pipelineID, enabled)
	if err != nil {
		return util.Wrapf(err, "Failed to enable/disable pipeline. Enabled: %v, pipelineID: %v",
			enabled, pipelineID)
	}

	return nil
}

func (r ResourceManager) GetPipelineAndLatestJobIterator() (*storage.PipelineAndLatestJobIterator,
	error) {
	return r.pipelineStore.GetPipelineAndLatestJobIterator()
}
