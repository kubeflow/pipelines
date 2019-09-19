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

	// Change status of a particular version.
	UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error
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
		return sqlBuilder.
			From("pipelines").
			LeftJoin("pipeline_versions ON pipelines.DefaultVersionId = pipeline_versions.UUID").
			Where(sq.Or{
				sq.And{
					sq.Eq{"pipelines.Status": model.PipelineReady},
					sq.Eq{"pipelines.DefaultVersionId": nil}},
				sq.And{
					sq.Eq{"pipelines.Status": model.PipelineReady},
					sq.Eq{"pipeline_versions.Status": model.PipelineVersionReady}}})
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
		var defaultVersionId sql.NullString
		var createdAtInSec int64
		var status model.PipelineStatus
		// var pipelineVersion model.PipelineVersion
		var versionUUID, versionName, versionParameters, versionPipelineId, versionCodeSourceUrls, versionStatus sql.NullString
		var versionCreatedAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&description,
			&parameters,
			&status,
			&defaultVersionId,
			&versionUUID,
			&versionCreatedAtInSec,
			&versionName,
			&versionParameters,
			&versionPipelineId,
			&versionStatus,
			&versionCodeSourceUrls); err != nil {
			return nil, err
		}
		if defaultVersionId.Valid {
			pipelines = append(pipelines, &model.Pipeline{
				UUID:             uuid,
				CreatedAtInSec:   createdAtInSec,
				Name:             name,
				Description:      description,
				Parameters:       parameters,
				Status:           status,
				DefaultVersionId: defaultVersionId.String,
				DefaultVersion: &model.PipelineVersion{
					UUID:           versionUUID.String,
					CreatedAtInSec: versionCreatedAtInSec.Int64,
					Name:           versionName.String,
					Parameters:     versionParameters.String,
					PipelineId:     versionPipelineId.String,
					Status:         model.PipelineVersionStatus(versionStatus.String),
					CodeSourceUrls: versionCodeSourceUrls.String,
				}})
		} else {
			pipelines = append(pipelines, &model.Pipeline{
				UUID:             uuid,
				CreatedAtInSec:   createdAtInSec,
				Name:             name,
				Description:      description,
				Parameters:       parameters,
				Status:           status,
				DefaultVersionId: "",
				DefaultVersion:   nil})
		}
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
		LeftJoin("pipeline_versions on pipelines.DefaultVersionId = pipeline_versions.UUID").
		Where(sq.Eq{"pipelines.Status": status}).
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
	// Set up creation time for both pipeline and pipeline version.
	newPipeline := *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	newPipeline.DefaultVersion.CreatedAtInSec = now

	// Set up UUID for pipeline and pipeline version.
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline id.")
	}
	newPipeline.UUID = id.String()
	newPipeline.DefaultVersion.PipelineId = id.String()

	// TODO(jingzhang36): before we expose versions to FE, we have to use same
	// UUID for pipeline and its (only) version and thus FE use pipeline UUID
	// instead of version UUID to to properly retrieve pipeline package.
	// id, err = s.uuid.NewRandom()
	// if err != nil {
	// 	return nil, util.NewInternalServerError(
	// 		err,
	// 		"Failed to create a pipeline version id.")
	// }
	newPipeline.DefaultVersionId = id.String()
	newPipeline.DefaultVersion.UUID = id.String()

	// Prepare sql queries for inserting into pipelines and pipeline_versions.
	sql, args, err := sq.
		Insert("pipelines").
		SetMap(
			sq.Eq{
				"UUID":             newPipeline.UUID,
				"CreatedAtInSec":   newPipeline.CreatedAtInSec,
				"Name":             newPipeline.Name,
				"Description":      newPipeline.Description,
				"Parameters":       newPipeline.Parameters,
				"Status":           string(newPipeline.Status),
				"DefaultVersionId": newPipeline.DefaultVersionId}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert pipeline to pipeline table: %v",
			err.Error())
	}
	sqlPipelineVersions, argsPipelineVersions, err := sq.
		Insert("pipeline_versions").
		SetMap(
			sq.Eq{
				"UUID":           newPipeline.DefaultVersion.UUID,
				"CreatedAtInSec": newPipeline.DefaultVersion.CreatedAtInSec,
				"Name":           newPipeline.DefaultVersion.Name,
				"Parameters":     newPipeline.DefaultVersion.Parameters,
				"Status":         string(newPipeline.DefaultVersion.Status),
				"PipelineId":     newPipeline.UUID,
				"CodeSourceUrls": newPipeline.DefaultVersion.CodeSourceUrls}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err,
			`Failed to create query to insert pipeline version to
			pipeline_versions table: %v`, err.Error())
	}

	// In a transaction, we insert into both pipelines and pipeline_versions.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err,
			`Failed to start a transaction to create a new pipeline: %v`,
			err.Error())
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			tx.Rollback()
			return nil, util.NewInvalidInputError(
				"Failed to create a new pipeline. The name %v already exist. Please specify a new name.", p.Name)
		}
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	_, err = tx.Exec(sqlPipelineVersions, argsPipelineVersions...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			tx.Rollback()
			return nil, util.NewInvalidInputError(
				`Failed to create a new pipeline version. The name %v already
				exist. Please specify a new name.`, p.DefaultVersion.Name)
		}
		tx.Rollback()
		return nil, util.NewInternalServerError(err,
			"Failed to add pipeline version to pipeline_versions table: %v",
			err.Error())
	}
	if err := tx.Commit(); err != nil {
		return nil, util.NewInternalServerError(err,
			`Failed to update pipelines and pipeline_versions in a
			transaction: %v`, err.Error())
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

func (s *PipelineStore) UpdatePipelineVersionStatus(id string, status model.PipelineVersionStatus) error {
	sql, args, err := sq.
		Update("pipeline_versions").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			`Failed to create query to update the pipeline version
			metadata: %s`, err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to update the pipeline version metadata: %s", err.Error())
	}
	return nil
}

// factory function for pipeline store
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}
