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
	"ml/backend/src/model"
	"ml/backend/src/util"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const (
	pipelineNotFoundString = "Pipeline"
)

type PipelineStoreInterface interface {
	ListPipelines() ([]model.Pipeline, error)
	GetPipeline(id uint) (*model.Pipeline, error)
	CreatePipeline(*model.Pipeline) (*model.Pipeline, error)
	DeletePipeline(id uint) error
	GetPipelineAndLatestJobIterator() (*PipelineAndLatestJobIterator, error)
	EnablePipeline(id uint, enabled bool) error
	UpdatePipelineStatus(id uint, status model.PipelineStatus) error
}

type PipelineStore struct {
	db   *gorm.DB
	time util.TimeInterface
}

func (s *PipelineStore) ListPipelines() ([]model.Pipeline, error) {
	var pipelines []model.Pipeline
	// List the pipelines as well as their parameters.
	// Preload parameter table first to optimize DB transaction.
	if r := s.db.Preload("Parameters").Where("status = ?", model.PipelineReady).Find(&pipelines); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list pipelines: %v",
			r.Error.Error())
	}
	return pipelines, nil
}

func (s *PipelineStore) GetPipeline(id uint) (*model.Pipeline, error) {
	var pipeline model.Pipeline
	// Get the pipeline as well as its parameter.
	r := s.db.Preload("Parameters").Where("status = ?", model.PipelineReady).First(&pipeline, id)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	if r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to get pipeline: %v", r.Error.Error())
	}
	return &pipeline, nil
}

func (s *PipelineStore) DeletePipeline(id uint) error {
	r := s.db.Exec(`DELETE FROM pipelines WHERE id=?`, id)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to delete pipeline: %v", r.Error.Error())
	}
	return nil
}

func (s *PipelineStore) CreatePipeline(p *model.Pipeline) (*model.Pipeline, error) {
	newPipeline := *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	newPipeline.UpdatedAtInSec = now
	newPipeline.EnabledAtInSec = now
	newPipeline.Enabled = true

	if r := s.db.Create(&newPipeline); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to add pipeline to pipeline table: %v",
			r.Error.Error())
	}
	return &newPipeline, nil
}

func (s *PipelineStore) EnablePipeline(id uint, enabled bool) error {

	// Note: We need to query the DB before performing the update so that we don't modify the
	// time at which the pipeline was enabled if it is already enabled.

	// TODO: add retries / timeouts for the whole transaction.
	// https://github.com/googleprivate/ml/issues/245

	errorMessage := fmt.Sprintf("Error when enabling pipeline %v to %v", id, enabled)

	// Begin transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, errorMessage)
	}

	// Get pipeline
	pipeline := model.Pipeline{ID: id}
	tx = tx.Find(&pipeline)
	if tx.RecordNotFound() {
		tx.Rollback()
		return util.NewResourceNotFoundError(pipelineNotFoundString, fmt.Sprint(id))
	} else if tx.Error != nil {
		tx.Rollback()
		return errors.Wrap(tx.Error, errorMessage)
	}

	// If the pipeline is already in the desired state, there is nothing to do.
	if pipeline.Enabled == enabled {
		tx.Rollback()
		return nil
	}

	now := s.time.Now().Unix()

	tx = tx.Exec(`UPDATE pipelines SET 
			enabled = ?, 
			enabled_at_in_sec = ?, 
			updated_at_in_sec = ?  
		WHERE 
			id = ?`, enabled, now, now, id)

	if tx.Error != nil {
		tx.Rollback()
		return errors.Wrap(tx.Error, errorMessage)
	}

	return errors.Wrap(tx.Commit().Error, errorMessage)
}

func (s *PipelineStore) UpdatePipelineStatus(id uint, status model.PipelineStatus) error {
	r := s.db.Exec(`UPDATE pipelines SET status=? WHERE id=?`, status, id)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to update the pipeline metadata: %s", r.Error.Error())
	}
	return nil
}

// factory function for pipeline store
func NewPipelineStore(db *gorm.DB, time util.TimeInterface) *PipelineStore {
	return &PipelineStore{
		db:   db,
		time: time,
	}
}

type PipelineAndLatestJob struct {
	PipelineID             uint
	PipelineName           string
	PipelineSchedule       string
	JobName                *string
	JobScheduledAtInSec    *int64
	PipelineEnabled        bool
	PipelineEnabledAtInSec int64
}

func (p *PipelineAndLatestJob) String() string {
	return fmt.Sprintf(
		"PipelineAndLatestJob{PipelineID: %v, PipelineName: %v, PipelineSchedule: %v, JobName: %v, JobScheduledAtInSec: %v, PipelineEnabled: %v, PipelineEnabledAtInSec: %v}",
		p.PipelineID,
		p.PipelineName,
		p.PipelineSchedule,
		util.StringNilOrValue(p.JobName),
		util.Int64NilOrValue(p.JobScheduledAtInSec),
		p.PipelineEnabled,
		p.PipelineEnabledAtInSec)
}

type PipelineAndLatestJobIterator struct {
	db   *gorm.DB
	rows *sql.Rows
}

func (s *PipelineStore) GetPipelineAndLatestJobIterator() (*PipelineAndLatestJobIterator, error) {
	return newPipelineAndLatestJobIterator(s.db)
}

func newPipelineAndLatestJobIterator(db *gorm.DB) (*PipelineAndLatestJobIterator, error) {
	rows, err := db.Raw(`SELECT 
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
			pipelines.enabled = 1 AND
			pipelines.status = ?
		GROUP BY
			pipelines.ID,
			pipelines.name,
			pipelines.schedule,
			pipelines.enabled,
			pipelines.enabled_at_in_sec
		ORDER BY jobs.scheduled_at_in_sec ASC`, model.PipelineReady).Rows()

	if err != nil {
		return nil, err
	}

	return &PipelineAndLatestJobIterator{
		db:   db,
		rows: rows,
	}, nil
}

func (p *PipelineAndLatestJobIterator) Next() bool {
	return p.rows.Next()
}

func (p *PipelineAndLatestJobIterator) Get() (*PipelineAndLatestJob, error) {
	var result PipelineAndLatestJob
	err := p.db.ScanRows(p.rows, &result)
	return &result, err
}

func (p *PipelineAndLatestJobIterator) Close() error {
	return p.rows.Close()
}
