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

// The order of the selected columns must match the order used in scan rows.
var pipelineColumns = []string{
	"pipelines.UUID",
	"pipelines.CreatedAtInSec",
	"pipelines.Name",
	"pipelines.Description",
	"pipelines.Parameters",
	"pipelines.Status",
	"pipelines.DefaultVersionId",
	"pipeline_versions.UUID",
	"pipeline_versions.CreatedAtInSec",
	"pipeline_versions.Name",
	"pipeline_versions.Parameters",
	"pipeline_versions.PipelineId",
	"pipeline_versions.Status",
	"pipeline_versions.CodeSourceUrl",
}

var pipelineVersionColumns = []string{
	"pipeline_versions.UUID",
	"pipeline_versions.CreatedAtInSec",
	"pipeline_versions.Name",
	"pipeline_versions.Parameters",
	"pipeline_versions.PipelineId",
	"pipeline_versions.Status",
	"pipeline_versions.CodeSourceUrl",
}

type PipelineStoreInterface interface {
	ListPipelines(opts *list.Options) ([]*model.Pipeline, int, string, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error)
	DeletePipeline(pipelineId string) error
	CreatePipeline(*model.Pipeline) (*model.Pipeline, error)
	UpdatePipelineStatus(string, model.PipelineStatus) error
	UpdatePipelineDefaultVersion(string, string) error

	CreatePipelineVersion(*model.PipelineVersion, bool) (*model.PipelineVersion, error)
	GetPipelineVersion(versionId string) (*model.PipelineVersion, error)
	GetPipelineVersionWithStatus(versionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error)
	ListPipelineVersions(pipelineId string, opts *list.Options) ([]*model.PipelineVersion, int, string, error)
	DeletePipelineVersion(pipelineVersionId string) error
	// Change status of a particular version.
	UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error
	// TODO(jingzhang36): remove this temporary method after resource manager's
	// CreatePipeline stops using it.
	UpdatePipelineAndVersionsStatus(id string, status model.PipelineStatus, pipelineVersionId string, pipelineVersionStatus model.PipelineVersionStatus) error
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
		return opts.AddFilterToSelect(sqlBuilder).
			From("pipelines").
			LeftJoin("pipeline_versions ON pipelines.DefaultVersionId = pipeline_versions.UUID").
			Where(sq.Eq{"pipelines.Status": model.PipelineReady})
	}

	sqlBuilder := buildQuery(sq.Select(pipelineColumns...))

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
		var versionUUID, versionName, versionParameters, versionPipelineId, versionCodeSourceUrl, versionStatus sql.NullString
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
			&versionCodeSourceUrl); err != nil {
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
					CodeSourceUrl:  versionCodeSourceUrl.String,
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
		Select(pipelineColumns...).
		From("pipelines").
		LeftJoin("pipeline_versions on pipelines.DefaultVersionId = pipeline_versions.UUID").
		Where(sq.And{sq.Eq{"pipelines.uuid": id}, sq.Eq{"pipelines.Status": status}}).
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
	// Set up creation time, UUID and sql query for pipeline.
	newPipeline := *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline id.")
	}
	newPipeline.UUID = id.String()
	// TODO(jingzhang36): remove default version id assignment after version API
	// is ready.
	newPipeline.DefaultVersionId = id.String()
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

	// Set up creation time, UUID and sql query for pipeline.
	// TODO(jingzhang36): remove version related operations from CreatePipeline
	// when version API is ready. Before that we create an implicit version
	// inside CreatePipeline method. And this implicit version has the same UUID
	// as pipeline; and thus FE can use either pipeline UUID or version UUID to
	// retrieve pipeline package.
	if newPipeline.DefaultVersion == nil {
		newPipeline.DefaultVersion = &model.PipelineVersion{
			Name:          newPipeline.Name,
			Parameters:    newPipeline.Parameters,
			Status:        model.PipelineVersionCreating,
			CodeSourceUrl: ""}
	}
	newPipeline.DefaultVersion.CreatedAtInSec = now
	newPipeline.DefaultVersion.PipelineId = id.String()
	newPipeline.DefaultVersion.UUID = id.String()
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
				"CodeSourceUrl":  newPipeline.DefaultVersion.CodeSourceUrl}).
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
			return nil, util.NewAlreadyExistError(
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
			return nil, util.NewAlreadyExistError(
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

func (s *PipelineStore) UpdatePipelineAndVersionsStatus(id string, status model.PipelineStatus, pipelineVersionId string, pipelineVersionStatus model.PipelineVersionStatus) error {
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(
			err,
			"Failed to start a transaction: %s",
			err.Error())
	}

	sql, args, err := sq.
		Update("pipelines").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to create query to update the pipeline status: %s", err.Error())
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to update the pipeline status: %s", err.Error())
	}
	sql, args, err = sq.
		Update("pipeline_versions").
		SetMap(sq.Eq{"Status": pipelineVersionStatus}).
		Where(sq.Eq{"UUID": pipelineVersionId}).
		ToSql()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			`Failed to create query to update the pipeline version
			status: %s`, err.Error())
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to update the pipeline version status: %s", err.Error())
	}

	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(err,
			"Failed to update pipeline status and its version status: %v", err)
	}
	return nil
}

// factory function for pipeline store
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}

func (s *PipelineStore) CreatePipelineVersion(v *model.PipelineVersion, updatePipelineDefaultVersion bool) (*model.PipelineVersion, error) {
	newPipelineVersion := *v
	newPipelineVersion.CreatedAtInSec = s.time.Now().Unix()
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline version id.")
	}
	newPipelineVersion.UUID = id.String()

	// Prepare queries of inserting new version and updating default version.
	versionSql, versionArgs, versionErr := sq.
		Insert("pipeline_versions").
		SetMap(
			sq.Eq{
				"UUID":           newPipelineVersion.UUID,
				"CreatedAtInSec": newPipelineVersion.CreatedAtInSec,
				"Name":           newPipelineVersion.Name,
				"Parameters":     newPipelineVersion.Parameters,
				"PipelineId":     newPipelineVersion.PipelineId,
				"Status":         string(newPipelineVersion.Status),
				"CodeSourceUrl":  newPipelineVersion.CodeSourceUrl}).
		ToSql()
	if versionErr != nil {
		return nil, util.NewInternalServerError(
			versionErr,
			"Failed to create query to insert version to pipeline version table: %v",
			versionErr.Error())
	}
	pipelineSql, pipelineArgs, pipelineErr := sq.
		Update("pipelines").
		SetMap(sq.Eq{"DefaultVersionId": newPipelineVersion.UUID}).
		Where(sq.Eq{"UUID": newPipelineVersion.PipelineId}).
		ToSql()
	if pipelineErr != nil {
		return nil, util.NewInternalServerError(
			pipelineErr,
			"Failed to create query to update pipeline default version id: %v",
			pipelineErr.Error())
	}

	// In a single transaction, insert new version and update default version.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to start a transaction: %v",
			err.Error())
	}

	_, err = tx.Exec(versionSql, versionArgs...)
	if err != nil {
		tx.Rollback()
		if s.db.IsDuplicateError(err) {
			return nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline version. The name %v already exist. Please specify a new name.", v.Name)
		}
		return nil, util.NewInternalServerError(err, "Failed to add version to pipeline version table: %v",
			err.Error())
	}

	if updatePipelineDefaultVersion {
		_, err = tx.Exec(pipelineSql, pipelineArgs...)
		if err != nil {
			tx.Rollback()
			return nil, util.NewInternalServerError(err, "Failed to update pipeline default version id: %v",
				err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create new pipeline version: %v",
			err.Error())
	}

	return &newPipelineVersion, nil
}

func (s *PipelineStore) UpdatePipelineDefaultVersion(pipelineId string, versionId string) error {
	sql, args, err := sq.
		Update("pipelines").
		SetMap(sq.Eq{"DefaultVersionId": versionId}).
		Where(sq.Eq{"UUID": pipelineId}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to update the pipeline default version: %s", err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the pipeline default version: %s", err.Error())
	}

	return nil
}

func (s *PipelineStore) GetPipelineVersion(versionId string) (*model.PipelineVersion, error) {
	return s.GetPipelineVersionWithStatus(versionId, model.PipelineVersionReady)
}

func (s *PipelineStore) GetPipelineVersionWithStatus(versionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error) {
	sql, args, err := sq.
		Select(pipelineVersionColumns...).
		From("pipeline_versions").
		Where(sq.And{sq.Eq{"UUID": versionId}, sq.Eq{"Status": status}}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get pipeline version: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline version: %v", err.Error())
	}
	defer r.Close()
	versions, err := s.scanPipelineVersionRows(r)

	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline version: %v", err.Error())
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("Version", fmt.Sprint(versionId))
	}
	return versions[0], nil
}

func (s *PipelineStore) scanPipelineVersionRows(rows *sql.Rows) ([]*model.PipelineVersion, error) {
	var pipelineVersions []*model.PipelineVersion
	for rows.Next() {
		var uuid, name, parameters, pipelineId, codeSourceUrl, status sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&parameters,
			&pipelineId,
			&status,
			&codeSourceUrl,
		); err != nil {
			return nil, err
		}
		if uuid.Valid {
			pipelineVersions = append(pipelineVersions, &model.PipelineVersion{
				UUID:           uuid.String,
				CreatedAtInSec: createdAtInSec.Int64,
				Name:           name.String,
				Parameters:     parameters.String,
				PipelineId:     pipelineId.String,
				CodeSourceUrl:  codeSourceUrl.String,
				Status:         model.PipelineVersionStatus(status.String)})
		}
	}
	return pipelineVersions, nil
}

func (s *PipelineStore) ListPipelineVersions(pipelineId string, opts *list.Options) ([]*model.PipelineVersion, int, string, error) {
	errorF := func(err error) ([]*model.PipelineVersion, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipeline versions: %v", err)
	}

	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		return opts.AddFilterToSelect(sqlBuilder).
			From("pipeline_versions").
			Where(sq.And{sq.Eq{"PipelineId": pipelineId}, sq.Eq{"status": model.PipelineVersionReady}})
	}

	// SQL for pipeline version list
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(
		buildQuery(sq.Select(pipelineVersionColumns...))).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size of pipeline versions.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same
	// rows queried.
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
	pipelineVersions, err := s.scanPipelineVersionRows(rows)
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

	if len(pipelineVersions) <= opts.PageSize {
		return pipelineVersions, total_size, "", nil
	}

	npt, err := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelineVersions[:opts.PageSize], total_size, npt, err
}

func (s *PipelineStore) DeletePipelineVersion(versionId string) error {
	// If this version is used as default version for a pipeline, we have to
	// find a new default version for that pipeline, which is usually the latest
	// version of that pipeline. Then we'll have 3 operations in a single
	// transactions: (1) delete version (2) get new default version id (3) use
	// new default version id to update pipeline.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(
			err,
			"Failed to start an transaction while trying to delete pipeline version: %v",
			err.Error())
	}

	// (1) delete version.
	_, err = tx.Exec(
		"delete from pipeline_versions where UUID = ?",
		versionId)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(
			err,
			"Failed to delete pipeline version: %v",
			err.Error())
	}

	// (2) check whether this version is used as default version.
	r, err := tx.Query(
		"select UUID from pipelines where DefaultVersionId = ?",
		versionId)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(
			err,
			`Failed to query pipelines table while deleting pipeline version:
			%v`,
			err.Error())
	}
	var pipelineId = ""
	if r.Next() {
		if err := r.Scan(&pipelineId); err != nil {
			tx.Rollback()
			return util.NewInternalServerError(
				err,
				"Failed to get pipeline id for version id: %v",
				err.Error())
		}
	}
	r.Close()
	if len(pipelineId) == 0 {
		// The deleted version is not used as a default version. So no extra
		// work is needed. We commit the deletion now.
		if err := tx.Commit(); err != nil {
			return util.NewInternalServerError(
				err,
				"Failed to delete pipeline version: %v",
				err.Error())
		}
		return nil
	}

	// (3) find a new default version.
	r, err = tx.Query(
		`select UUID from pipeline_versions
		where PipelineId = ? and Status = ?
		order by CreatedAtInSec DESC
		limit 1`,
		pipelineId,
		model.PipelineVersionReady)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(
			err,
			"Failed to get a new default version id: %v",
			err.Error())
	}
	var newDefaultVersionId = ""
	if r.Next() {
		if err := r.Scan(&newDefaultVersionId); err != nil {
			tx.Rollback()
			return util.NewInternalServerError(
				err,
				"Failed to get a new default version id: %v",
				err.Error())
		}
	}
	r.Close()
	if len(newDefaultVersionId) == 0 {
		// No new default version. The pipeline's default version id will be
		// null.
		_, err = tx.Exec(
			"update pipelines set DefaultVersionId = null where UUID = ?",
			pipelineId)
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(
				err,
				"Failed to update pipeline's default version id: %v",
				err.Error())
		}
	} else {
		_, err = tx.Exec(
			"update pipelines set DefaultVersionId = ? where UUID = ?",
			newDefaultVersionId, pipelineId)
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(
				err,
				"Failed to update pipeline's default version id: %v",
				err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(
			err,
			"Failed to delete pipeline version: %v",
			err.Error())
	}
	return nil
}

// SetUUIDGenerator is for unit tests in other packages who need to set uuid,
// since uuid is not exported.
func (s *PipelineStore) SetUUIDGenerator(new_uuid util.UUIDGeneratorInterface) {
	s.uuid = new_uuid
}
