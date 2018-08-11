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
	"bytes"
	"database/sql"
	"fmt"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type PipelineStoreInterface interface {
	ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Pipeline, string, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error)
	DeletePipeline(pipelineId string) error
	CreatePipeline(*model.Pipeline) (*model.Pipeline, error)
	UpdatePipelineStatus(string, model.PipelineStatus) error
}

type PipelineStore struct {
	db   *sql.DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

func (s *PipelineStore) ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Pipeline, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetPipelineTablePrimaryKeyColumn(), isDesc)
	if err != nil {
		return nil, "", err
	}
	models, pageToken, err := listModel(paginationContext, s.queryPipelineTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List pipeline failed.")
	}
	return s.toPipelines(models), pageToken, err
}

func (s *PipelineStore) queryPipelineTable(context *PaginationContext) ([]model.ListableDataModel, error) {
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM pipelines WHERE Status = '%v'", model.PipelineReady))
	toPaginationQuery("AND", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))
	r, err := s.db.Query(query.String())
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
	r, err := s.db.Query("SELECT * FROM pipelines WHERE uuid=? AND status=? LIMIT 1", id, status)
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
	_, err := s.db.Exec(`DELETE FROM pipelines WHERE UUID=?`, id)
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
	stmt, err := s.db.Prepare(
		`INSERT INTO pipelines (UUID, CreatedAtInSec,Name,Description,Parameters,Status)
						VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		newPipeline.UUID,
		newPipeline.CreatedAtInSec,
		newPipeline.Name,
		newPipeline.Description,
		newPipeline.Parameters,
		string(newPipeline.Status))
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	return &newPipeline, nil
}

func (s *PipelineStore) UpdatePipelineStatus(id string, status model.PipelineStatus) error {
	_, err := s.db.Exec(`UPDATE pipelines SET Status=? WHERE UUID=?`, status, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the pipeline metadata: %s", err.Error())
	}
	return nil
}

func (s *PipelineStore) toListablePipelines(pkgs []model.Pipeline) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(pkgs))
	for i := range models {
		models[i] = pkgs[i]
	}
	return models
}

func (s *PipelineStore) toPipelines(models []model.ListableDataModel) []model.Pipeline {
	pkgs := make([]model.Pipeline, len(models))
	for i := range models {
		pkgs[i] = models[i].(model.Pipeline)
	}
	return pkgs
}

// factory function for pipeline store
func NewPipelineStore(db *sql.DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}
