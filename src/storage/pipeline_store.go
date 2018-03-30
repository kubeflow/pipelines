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

package storage

import (
	"database/sql"
	"fmt"
	"ml/src/message"
	"ml/src/util"

	"github.com/jinzhu/gorm"
)

type PipelineStoreInterface interface {
	ListPipelines() ([]message.Pipeline, error)
	GetPipeline(id uint) (message.Pipeline, error)
	CreatePipeline(message.Pipeline) (message.Pipeline, error)
	GetPipelineAndLatestJobIterator() (*PipelineAndLatestJobIterator, error)
}

type PipelineStore struct {
	db   *gorm.DB
	time util.TimeInterface
}

func (s *PipelineStore) ListPipelines() ([]message.Pipeline, error) {
	var pipelines []message.Pipeline
	// List the pipelines as well as their parameters.
	// Preload parameter table first to optimize DB transaction.
	if r := s.db.Preload("Parameters").Find(&pipelines); r.Error != nil {
		return nil, util.NewInternalError("Failed to list pipelines", r.Error.Error())
	}
	return pipelines, nil
}

func (s *PipelineStore) GetPipeline(id uint) (message.Pipeline, error) {
	var pipeline message.Pipeline
	// Get the pipeline as well as its parameter.
	if r := s.db.Preload("Parameters").First(&pipeline, id); r.Error != nil {
		// Error returns when no pipeline found.
		return pipeline, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	return pipeline, nil
}

func (s *PipelineStore) CreatePipeline(p message.Pipeline) (message.Pipeline, error) {
	p.Enabled = true
	p.EnabledAtInSec = s.time.Now().Unix()

	if r := s.db.Create(&p); r.Error != nil {
		return p, util.NewInternalError("Failed to add pipeline to pipeline table", r.Error.Error())
	}
	return p, nil
}

// factory function for pipeline store
func NewPipelineStore(db *gorm.DB, time util.TimeInterface) *PipelineStore {
	return &PipelineStore{
		db:   db,
		time: time,
	}
}

type PipelineAndLatestJob struct {
	PipelineID             *string
	PipelineName           *string
	PipelineSchedule       *string
	JobName                *string
	JobScheduledAtInSec    *int64
	PipelineEnabled        *bool
	PipelineEnabledAtInSec *int64
}

func (p *PipelineAndLatestJob) String() string {
	return fmt.Sprintf(
		"PipelineAndLatestJob{PipelineID: %v, PipelineName: %v, PipelineSchedule: %v, JobName: %v, JobScheduledAtInSec: %v, PipelineEnabled: %v, PipelineEnabledAtInSec: %v}",
		util.StringNilOrValue(p.PipelineID),
		util.StringNilOrValue(p.PipelineName),
		util.StringNilOrValue(p.PipelineSchedule),
		util.StringNilOrValue(p.JobName),
		util.Int64NilOrValue(p.JobScheduledAtInSec),
		util.BoolNilOrValue(p.PipelineEnabled),
		util.Int64NilOrValue(p.PipelineEnabledAtInSec))
}

type PipelineAndLatestJobIterator struct {
	rows *sql.Rows
}

func (s *PipelineStore) GetPipelineAndLatestJobIterator() (
	*PipelineAndLatestJobIterator, error) {

	rows, err := s.db.Raw(`SELECT 
		pipelines.ID AS pipeline_id, 
		pipelines.name AS pipeline_name, 
		pipelines.schedule AS pipeline_schedule, 
		jobs.name AS job_name, 
		MAX(jobs.scheduled_at_in_sec) AS job_scheduled_at_in_sec,
		pipelines.enabled AS pipeline_enabled,
		pipelines.enabled_at_in_sec AS pipeline_enabled_at_in_sec
		FROM pipelines
		LEFT JOIN jobs
		ON (pipelines.ID=jobs.pipeline_id)
		WHERE
			pipelines.schedule != "" AND
			pipelines.enabled = 1
		GROUP BY
			pipelines.ID,
			pipelines.name,
			pipelines.schedule,
			pipelines.enabled,
			pipelines.enabled_at_in_sec
		ORDER BY jobs.scheduled_at_in_sec ASC`).Rows()

	if err != nil {
		return nil, err
	}

	return newPipelineAndLatestJobIterator(rows), nil
}

func newPipelineAndLatestJobIterator(rows *sql.Rows) *PipelineAndLatestJobIterator {
	return &PipelineAndLatestJobIterator{
		rows: rows,
	}
}

func (p *PipelineAndLatestJobIterator) Next() bool {
	return p.rows.Next()
}

func (p *PipelineAndLatestJobIterator) Get() (*PipelineAndLatestJob, error) {
	var result PipelineAndLatestJob
	err := p.rows.Scan(
		&result.PipelineID,
		&result.PipelineName,
		&result.PipelineSchedule,
		&result.JobName,
		&result.JobScheduledAtInSec,
		&result.PipelineEnabled,
		&result.PipelineEnabledAtInSec)
	return &result, err
}

func (p *PipelineAndLatestJobIterator) Close() error {
	return p.rows.Close()
}
