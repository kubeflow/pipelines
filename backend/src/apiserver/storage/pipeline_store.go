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

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type PipelineStoreInterface interface {
	ListPipelines(opts *list.Options) ([]*model.Pipeline, int, string, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error)
	DeletePipeline(pipelineId string) error
	CreatePipeline(*model.Pipeline) (*model.Pipeline, error)
	UpdatePipelineStatus(string, model.PipelineStatus) error
}

type PipelineStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size.
func (s *PipelineStore) ListPipelines(opts *list.Options) ([]*model.Pipeline, int, string, error) {
	errorF := func(err error) ([]*model.Pipeline, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipelines: %v", err)
	}

	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		return sqlBuilder.From("pipelines").Where(sq.Eq{"Status": model.PipelineReady})
	}

	sqlBuilder := buildQuery(sq.Select("*"))

	// SQL for row list
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list pipelines")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	pipelines, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipelines")
		return errorF(err)
	}

	if len(pipelines) <= opts.PageSize {
		return pipelines, total_size, "", nil
	}

	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	return pipelines[:opts.PageSize], total_size, npt, err
}

func (s *PipelineStore) scanRows(rows *sql.Rows) ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	for rows.Next() {
		var uuid, name, parameters, description string
		var createdAtInSec int64
		var status model.PipelineStatus
		if err := rows.Scan(&uuid, &createdAtInSec, &name, &description, &parameters, &status); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, &model.Pipeline{
			UUID:           uuid,
			CreatedAtInSec: createdAtInSec,
			Name:           name,
			Description:    description,
			Parameters:     parameters,
			Status:         status})
	}
	return pipelines, nil
}

func (s *PipelineStore) GetPipeline(id string) (*model.Pipeline, error) {
	return s.GetPipelineWithStatus(id, model.PipelineReady)
}

func (s *PipelineStore) GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error) {
	sql, args, err := sq.
		Select("*").
		From("pipelines").
		Where(sq.Eq{"uuid": id}).
		Where(sq.Eq{"status": status}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get pipeline: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err.Error())
	}
	defer r.Close()
	pipelines, err := s.scanRows(r)

	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	return pipelines[0], nil
}

func (s *PipelineStore) DeletePipeline(id string) error {
	sql, args, err := sq.Delete("pipelines").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete pipeline: %v", err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete pipeline: %v", err.Error())
	}
	return nil
}

func (s *PipelineStore) CreatePipeline(p *model.Pipeline) (*model.Pipeline, error) {
	newPipeline := *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline id.")
	}
	newPipeline.UUID = id.String()
	sql, args, err := sq.
		Insert("pipelines").
		SetMap(
			sq.Eq{
				"UUID":           newPipeline.UUID,
				"CreatedAtInSec": newPipeline.CreatedAtInSec,
				"Name":           newPipeline.Name,
				"Description":    newPipeline.Description,
				"Parameters":     newPipeline.Parameters,
				"Status":         string(newPipeline.Status)}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert pipeline to pipeline table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			return nil, util.NewInvalidInputError(
				"Failed to create a new pipeline. The name %v already exist. Please specify a new name.", p.Name)
		}
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	return &newPipeline, nil
}

func (s *PipelineStore) UpdatePipelineStatus(id string, status model.PipelineStatus) error {
	sql, args, err := sq.
		Update("pipelines").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to update the pipeline metadata: %s", err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the pipeline metadata: %s", err.Error())
	}
	return nil
}

// factory function for pipeline store
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}
