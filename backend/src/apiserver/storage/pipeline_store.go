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
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type PipelineStoreInterface interface {
	ListPipelines(context *common.PaginationContext) ([]model.Pipeline, string, error)
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

func (s *PipelineStore) ListPipelines(context *common.PaginationContext) ([]model.Pipeline, string, error) {
	models, pageToken, err := listModel(context, s.queryPipelineTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List pipeline failed.")
	}
	return s.toPipelines(models), pageToken, err
}

func (s *PipelineStore) queryPipelineTable(context *common.PaginationContext) ([]model.ListableDataModel, error) {
	sqlBuilder := sq.Select("*").From("pipelines").Where(sq.Eq{"Status": model.PipelineReady})
	sql, args, err := toPaginationQuery(sqlBuilder, context).Limit(uint64(context.PageSize)).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to list pipelines: %v",
			err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list pipelines: %v", err.Error())
	}
	defer r.Close()
	pipelines, err := s.scanRows(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list pipelines: %v", err.Error())
	}
	return s.toListablePipelines(pipelines), nil
}

func (s *PipelineStore) scanRows(rows *sql.Rows) ([]model.Pipeline, error) {
	var pipelines []model.Pipeline
	for rows.Next() {
		var uuid, name, parameters, description string
		var createdAtInSec int64
		var status model.PipelineStatus
		if err := rows.Scan(&uuid, &createdAtInSec, &name, &description, &parameters, &status); err != nil {
			return pipelines, err
		}
		pipelines = append(pipelines, model.Pipeline{
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
	return &pipelines[0], nil
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

func (s *PipelineStore) toListablePipelines(pipelines []model.Pipeline) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(pipelines))
	for i := range models {
		models[i] = pipelines[i]
	}
	return models
}

func (s *PipelineStore) toPipelines(models []model.ListableDataModel) []model.Pipeline {
	pipelines := make([]model.Pipeline, len(models))
	for i := range models {
		pipelines[i] = models[i].(model.Pipeline)
	}
	return pipelines
}

// factory function for pipeline store
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}
