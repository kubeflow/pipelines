// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"ml/src/resource"
	"ml/src/storage"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

const (
	sleepDurationBetweenRunsFlagName = "sleep_duration_between_runs"
	dbDriverNameFlagName             = "db_driver_name"
	sqliteDatasourceNameFlagName     = "sqlite_datasource_name"
	userFlagName                     = "user"
	mysqlServiceHostFlagName         = "mysql_service_host"
	mysqlServicePortFlagName         = "mysql_service_port"
	mysqlDBNameFlagName              = "mysql_db_name"
	minioServiceHostFlagName         = "minio_service_host"
	minioServicePortFlagName         = "minio_service_port"
	minioAccessKeyFlagName           = "minio_access_key"
	minioSecretKeyFlagName           = "minio_secret_key"
	minioBucketNameFlagName          = "minio_bucket_name"
	namespaceFlagName                = "namespace"
)

type Controller struct {
	resourceManager *resource.ResourceManager
}

func NewController(resourceManager *resource.ResourceManager) *Controller {
	return &Controller{
		resourceManager: resourceManager,
	}
}

func getNextStartTimeAfter(schedule string, referenceTime time.Time) (time.Time, error) {
	sched, err := cron.Parse(schedule)
	if err != nil {
		return time.Unix(0, 0).UTC(), errors.Wrapf(err, "Unparseable schedule: %s", schedule)
	}
	return sched.Next(referenceTime), nil
}

func mustRun(schedule string, lastJobRunAt time.Time, now time.Time, pipelineEnabledAt time.Time) (
	bool, time.Time, error) {

	// Determine the time after which the schedule takes effect.
	var referenceTime time.Time
	if lastJobRunAt.Before(pipelineEnabledAt) {
		referenceTime = pipelineEnabledAt
	} else {
		referenceTime = lastJobRunAt
	}

	// Determinte the next time a job should start.
	nextStartTime, err := getNextStartTimeAfter(schedule, referenceTime)
	if err != nil {
		return false, time.Unix(0, 0).UTC(), errors.Wrapf(err,
			"Could not determine the next time to run a job. Schedule: %v, LastJobRunAt: %v, now: %v, pipelineEnabledAt: %v.",
			schedule, lastJobRunAt, now, pipelineEnabledAt)
	}

	// Return whether a new job must be run, as well as the time at which the job was supposed
	// to be scheduled.
	return nextStartTime.Before(now), nextStartTime, nil
}

func (c Controller) runForSingleRow(pipeline *storage.PipelineAndLatestJob) (
	bool, time.Time, error) {

	// Validating pipeline.PipelineSchedule
	if pipeline.PipelineSchedule == "" {
		return false, time.Unix(0, 0).UTC(),
			fmt.Errorf("The schedule should not be empty: %v", pipeline)
	}

	// Validating pipeline.PipelineEnabledAtInSec
	if pipeline.PipelineEnabledAtInSec == 0 {
		return false, time.Unix(0, 0).UTC(),
			fmt.Errorf("PipelineEnabledAtInSec should not be 0: %v", pipeline)
	}

	pipelineEnabledAt := time.Unix(pipeline.PipelineEnabledAtInSec, 0).UTC()

	// Converting pipeline.JobScheduledAtInSec to a Time.
	// Note that pipeline.JobScheduledAtInSec can be null if there is no job for this pipeline.
	// In this case, we set the time of the last scheduled job to the earliest possible time,
	// which causes it to be ignored in subsequent code.
	var lastJobScheduledAt time.Time
	if pipeline.JobScheduledAtInSec != nil && *pipeline.JobScheduledAtInSec != 0 {
		lastJobScheduledAt = time.Unix(*pipeline.JobScheduledAtInSec, 0).UTC()
	} else {
		lastJobScheduledAt = time.Unix(0, 0).UTC()
	}

	now := c.resourceManager.GetTime().Now().UTC()

	mustRun, scheduledTime, err := mustRun(
		pipeline.PipelineSchedule,
		lastJobScheduledAt,
		now,
		pipelineEnabledAt)

	if err != nil {
		return false, time.Unix(0, 0).UTC(), errors.Wrapf(err,
			"Could not figure out whether a job should be created at time '%v' for pipeline: %+v",
			now, pipeline)
	}

	glog.Infof(
		"Should a pipeline run for pipeline %v? %v. Details: DB row: %+v, schedule: %v, lastJobScheduledAt: %v, now: %v, pipelineEnabledAt: %v, scheduledTime: %v",
		pipeline.PipelineID, mustRun, pipeline, pipeline.PipelineSchedule, lastJobScheduledAt, now,
		pipelineEnabledAt, scheduledTime)

	if mustRun {

		jobDetail, err := c.resourceManager.CreateJobFromPipelineID(pipeline.PipelineID,
			scheduledTime.Unix())
		if err != nil {
			return false, time.Unix(0, 0), errors.Wrapf(err, "Failed to create a job for pipeline: %+v",
				pipeline)
		}
		glog.Infof("Successfully created job '%v' for scheduled time '%v' for pipeline: %+v",
			jobDetail.Workflow.Name, jobDetail.Job.ScheduledAtInSec, pipeline)
		return true, scheduledTime, nil
	}

	return false, scheduledTime, nil
}

func (c Controller) runForQuery() error {

	iterator, err := c.resourceManager.GetPipelineAndLatestJobIterator()

	if err != nil {
		return err
	}

	defer iterator.Close()

	for iterator.Next() {

		pipelineAndLatestJob, err := iterator.Get()
		if err != nil {
			glog.Errorf(
				"Error while iterating over pipelines to find jobs to schedule (moving on to the next pipeline): %+v",
				err)
			continue
		}

		ran, scheduledTime, err := c.runForSingleRow(pipelineAndLatestJob)
		if err != nil {
			glog.Errorf(
				"Error while scheduling a job for a pipeline (moving on to the next pipeline). PipelineAndLatestJob: %v. Error: %+v",
				pipelineAndLatestJob, err)
			continue
		} else if ran {
			glog.Infof("Scheduled a job for time '%v' for pipeline: %v", scheduledTime,
				pipelineAndLatestJob)
		} else {
			glog.Infof("Not yet time to schedule a job for time '%v' for pipeline: %v", scheduledTime,
				pipelineAndLatestJob)
		}
	}

	return nil
}

func (c Controller) run(sleepDurationBetweenRuns time.Duration) error {

	for {
		glog.Info("Querying for jobs to schedule...")
		err := c.runForQuery()
		if err != nil {
			glog.Errorf("Error while scheduling jobs (will try again): %+v", err)
		} else {
			glog.Info("Successfully scheduled any job that needed to be scheduled.")
		}
		time.Sleep(sleepDurationBetweenRuns)
	}
}

func checkFlagNotEmptyOrFatal(flag *string, flagName string) {
	if *flag == "" {
		glog.Fatalf("The flag '%v' must be specified.", flagName)
	}
}

func main() {
	sleepDurationBetweenRuns := flag.Duration(sleepDurationBetweenRunsFlagName, 10*time.Second,
		"Duration between subsequent checks of whether new jobs are scheduled to run.")
	dbDriverName := flag.String(dbDriverNameFlagName, "",
		"The name of the database driver.")
	sqliteDatasourceName := flag.String(sqliteDatasourceNameFlagName, "",
		"The name of the SQLite datasource.")
	user := flag.String(userFlagName, "root", "The user name to connect to MySQL.")
	mysqlServiceHost := flag.String(mysqlServiceHostFlagName, "",
		"The MySQL service host.")
	mysqlServicePort := flag.String(mysqlServicePortFlagName, "",
		"The MySQL service port.")
	mysqlDBName := flag.String(mysqlDBNameFlagName, "", "The MySQL database name.")
	minioServiceHost := flag.String(minioServiceHostFlagName, "", "The Minio service host.")
	minioServicePort := flag.String(minioServicePortFlagName, "", "The Minio service port.")
	minioAccessKey := flag.String(minioAccessKeyFlagName, "", "The Minio access key.")
	minioSecretKey := flag.String(minioSecretKeyFlagName, "", "The Minio secret key.")
	minioBucketName := flag.String(minioBucketNameFlagName, "", "The Minio bucket name.")
	namespace := flag.String(namespaceFlagName, "", "The pod namespace")

	flag.Parse()

	glog.Infof("Starting the controller...")

	checkFlagNotEmptyOrFatal(dbDriverName, dbDriverNameFlagName)

	if *sqliteDatasourceName == "" {
		checkFlagNotEmptyOrFatal(mysqlServiceHost, mysqlServiceHostFlagName)
		checkFlagNotEmptyOrFatal(mysqlServicePort, mysqlServicePortFlagName)
		checkFlagNotEmptyOrFatal(mysqlDBName, mysqlDBNameFlagName)
	}

	checkFlagNotEmptyOrFatal(minioServiceHost, minioServiceHostFlagName)
	checkFlagNotEmptyOrFatal(minioServicePort, minioServicePortFlagName)
	checkFlagNotEmptyOrFatal(minioAccessKey, minioAccessKeyFlagName)
	checkFlagNotEmptyOrFatal(minioSecretKey, minioSecretKeyFlagName)
	checkFlagNotEmptyOrFatal(minioBucketName, minioBucketNameFlagName)
	checkFlagNotEmptyOrFatal(namespace, namespaceFlagName)

	clientManager, err := NewClientManager(&ClientManagerParams{
		DBDriverName:         *dbDriverName,
		SqliteDatasourceName: *sqliteDatasourceName,
		User:                 *user,
		MysqlServiceHost:     *mysqlServiceHost,
		MysqlServicePort:     *mysqlServicePort,
		MysqlDBName:          *mysqlDBName,
		MinioServiceHost:     *minioServiceHost,
		MinioServicePort:     *minioServicePort,
		MinioAccessKey:       *minioAccessKey,
		MinioSecretKey:       *minioSecretKey,
		MinioBucketName:      *minioBucketName,
		Namespace:            *namespace})
	if err != nil {
		glog.Fatalf("Could not instantiate the ClientManager: %+v", err)
	}
	defer clientManager.Close()

	manager := resource.NewResourceManager(clientManager)
	controller := NewController(manager)
	controller.run(*sleepDurationBetweenRuns)
}
