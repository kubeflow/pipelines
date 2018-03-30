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

package schedule

import (
	"flag"
	"fmt"
	"ml/src/client"
	"ml/src/storage"
	"ml/src/util"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

const(
	sleepDurationBetweenRunsFlagName = "sleep_duration_between_runs"
	dbDriverNameFlagName = "db_driver_name"
	sqliteDatasourceNameFlagName = "sqlite_datasource_name"
	userFlagName = "user"
	mysqlServiceHostFlagName = "mysql_service_host"
	mysqlServicePortFlagName = "mysql_service_port"
	mysqlDBNameFlagName = "mysql_db_name"
)

type Controller struct {
	pipelineStore storage.PipelineStoreInterface
	time          util.TimeInterface
}

func NewController(pipelineStore storage.PipelineStoreInterface,
	time util.TimeInterface) *Controller {
	return &Controller{
		pipelineStore: pipelineStore,
		time:          time,
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
	return now.Before(nextStartTime), nextStartTime, nil
}

func (c Controller) runForSingleRow(pipeline *storage.PipelineAndLatestJob) (
	bool, time.Time, error) {

	// Validating pipeline.PipelineSchedule
	if pipeline.PipelineSchedule == nil || *pipeline.PipelineSchedule == "" {
		return false, time.Unix(0, 0).UTC(),
			fmt.Errorf("The schedule should not be nil nor empty: %v", pipeline)
	}

	// Validating pipeline.PipelineEnabledAtInSec
	if pipeline.PipelineEnabledAtInSec == nil || *pipeline.PipelineEnabledAtInSec == 0 {
		return false, time.Unix(0, 0).UTC(),
			fmt.Errorf("PipelineEnabledAtInSec should not be nil nor 0: %v", pipeline)
	}

	pipelineEnabledAt := time.Unix(*pipeline.PipelineEnabledAtInSec, 0).UTC()

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

	now := c.time.Now().UTC()

	mustRun, scheduledTime, err := mustRun(
		*pipeline.PipelineSchedule,
		lastJobScheduledAt,
		now,
		pipelineEnabledAt)

	if err != nil {
		return false, time.Unix(0, 0).UTC(), errors.Wrapf(err,
			"Could not figure out whether a job should be created at time '%v' for pipeline: %v",
			now, pipeline)
	}

	if mustRun {

		// TODO: Schedule a job
		return true, scheduledTime, nil
	}

	return false, scheduledTime, nil
}

func (c Controller) runForQuery() error {

	iterator, err := c.pipelineStore.GetPipelineAndLatestJobIterator()

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
			glog.Infof("No yet time to schedule a job for time '%v' for pipeline: %v", scheduledTime,
				pipelineAndLatestJob)
		}
	}

	return nil
}

func (c Controller) run(sleepDurationBetweenRuns time.Duration) error {

	for {
		err := c.runForQuery()
		if err != nil {
			glog.Errorf("Error while scheduling jobs (will try again): %+v", err)
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
	flag.Parse()

	checkFlagNotEmptyOrFatal(dbDriverName, dbDriverNameFlagName)

	if *sqliteDatasourceName == "" {
		checkFlagNotEmptyOrFatal(mysqlServiceHost, mysqlServiceHostFlagName)
		checkFlagNotEmptyOrFatal(mysqlServicePort, mysqlServicePortFlagName)
		checkFlagNotEmptyOrFatal(mysqlDBName, mysqlDBNameFlagName)
	}

	time := util.NewRealTime()
	gormClient, err := client.CreateGormClient(
		*dbDriverName,
		*sqliteDatasourceName,
		*user,
		*mysqlServiceHost,
		*mysqlServicePort,
		*mysqlDBName)

	if err != nil {
		glog.Fatalf("The GORM client could not be created: %+v", err)
	}

	defer gormClient.Close()

	pipelineStore := storage.NewPipelineStore(gormClient, time)

	controller := NewController(pipelineStore, time)

	controller.run(*sleepDurationBetweenRuns)
}
