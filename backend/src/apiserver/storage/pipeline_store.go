// Copyright 2018 The Kubeflow Authors
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

var (
	// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
	// `pipelines` joined with `pipeline_versions`
	// This supports v1beta1 behavior.
	// The order of the selected columns must match the order used in scan rows.
	joinedColumns = []string{
		"pipelines.UUID",
		"pipelines.CreatedAtInSec",
		"pipelines.Name",
		"pipelines.Description",
		"pipelines.Status",
		"pipelines.Namespace",
		"pipeline_versions.UUID",
		"pipeline_versions.CreatedAtInSec",
		"pipeline_versions.Name",
		"pipeline_versions.Parameters",
		"pipeline_versions.PipelineId",
		"pipeline_versions.Status",
		"pipeline_versions.CodeSourceUrl",
		"pipeline_versions.Description",
		"pipeline_versions.PipelineSpec",
		"pipeline_versions.PipelineSpecURI",
	}

	// `pipelines`
	// The order of the selected columns must match the order used in scan rows.
	pipelineColumns = []string{
		"pipelines.UUID",
		"pipelines.CreatedAtInSec",
		"pipelines.Name",
		"pipelines.Description",
		"pipelines.Status",
		"pipelines.Namespace",
	}

	// `pipeline_versions`
	// The order of the selected columns must match the order used in scan rows.
	pipelineVersionColumns = []string{
		"pipeline_versions.UUID",
		"pipeline_versions.CreatedAtInSec",
		"pipeline_versions.Name",
		"pipeline_versions.Parameters",
		"pipeline_versions.PipelineId",
		"pipeline_versions.Status",
		"pipeline_versions.CodeSourceUrl",
		"pipeline_versions.Description",
		"pipeline_versions.PipelineSpec",
		"pipeline_versions.PipelineSpecURI",
	}
)

type PipelineStoreInterface interface {
	// TODO(gkcalat): As these calls use joins on two (potentially) large sets with one-many relationship,
	// let's keep them to avoid performance issues. consider removing after KFP v2 GA if users are not affected.
	//
	// `pipelines` left joined with `pipeline_versions`
	// This supports v1beta1 behavior.
	GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, *model.PipelineVersion, error)
	ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error)

	// `pipelines`
	CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error)
	GetPipelineWithStatus(pipelineId string, status model.PipelineStatus) (*model.Pipeline, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error)
	ListPipelines(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error)
	UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error
	DeletePipeline(pipelineId string) error
	UpdatePipelineDefaultVersion(pipelineId string, versionId string) error

	// `pipeline_versions`
	CreatePipelineVersion(pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error)
	GetPipelineVersionWithStatus(pipelineVersionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error)
	GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error)
	GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error)
	ListPipelineVersions(pipelineId string, opts *list.Options) ([]*model.PipelineVersion, int, string, error)
	UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error
	DeletePipelineVersion(pipelineVersionId string) error
	// DeletePipelineVersionAndUpdateDefaultV1(pipelineVersionId string) error // Used in v1beta1 only
}

type PipelineStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// Factory function for pipeline store.
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}

// SetUUIDGenerator is for unit tests in other packages who need to set uuid,
// since uuid is not exported.
func (s *PipelineStore) SetUUIDGenerator(newUUID util.UUIDGeneratorInterface) {
	s.uuid = newUUID
}

// Converts SQL response into []Pipeline (default version is set to nil).
func (s *PipelineStore) scanPipelinesRows(rows *sql.Rows) ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	for rows.Next() {
		var uuid, name, status, description, namespace sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&description,
			&status,
			&namespace,
		); err != nil {
			return nil, err
		}
		if uuid.Valid {
			pipelines = append(
				pipelines,
				&model.Pipeline{
					UUID:           uuid.String,
					CreatedAtInSec: createdAtInSec.Int64,
					Name:           name.String,
					Description:    description.String,
					Status:         model.PipelineStatus(status.String),
					Namespace:      namespace.String,
				},
			)
		}
	}
	return pipelines, nil
}

// Converts SQL response into []PipelineVersion.
func (s *PipelineStore) scanPipelineVersionsRows(rows *sql.Rows) ([]*model.PipelineVersion, error) {
	var pipelineVersions []*model.PipelineVersion
	for rows.Next() {
		var uuid, name, parameters, pipelineId, codeSourceUrl, status, description, pipelineSpec, pipelineSpecURI sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&parameters,
			&pipelineId,
			&status,
			&codeSourceUrl,
			&description,
			&pipelineSpec,
			&pipelineSpecURI,
		); err != nil {
			return nil, err
		}
		if uuid.Valid {
			pipelineVersions = append(
				pipelineVersions,
				&model.PipelineVersion{
					UUID:            uuid.String,
					CreatedAtInSec:  createdAtInSec.Int64,
					Name:            name.String,
					Parameters:      parameters.String,
					PipelineId:      pipelineId.String,
					CodeSourceUrl:   codeSourceUrl.String,
					Status:          model.PipelineVersionStatus(status.String),
					Description:     description.String,
					PipelineSpec:    pipelineSpec.String,
					PipelineSpecURI: pipelineSpecURI.String,
				},
			)
		}
	}
	return pipelineVersions, nil
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Parses SQL results of joining `pipelines` and `pipeline_versions` tables into []Pipelines.
// This supports v1beta1 behavior.
func (s *PipelineStore) scanJoinedRows(rows *sql.Rows) ([]*model.Pipeline, []*model.PipelineVersion, error) {
	var pipelines []*model.Pipeline
	var pipelineVersions []*model.PipelineVersion
	for rows.Next() {
		var uuid, name, description string
		var namespace sql.NullString
		var createdAtInSec int64
		var status model.PipelineStatus
		var versionUUID, versionName, versionParameters, versionPipelineId, versionCodeSourceUrl, versionStatus, versionDescription, pipelineSpec, pipelineSpecURI sql.NullString
		var versionCreatedAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&description,
			&status,
			&namespace,
			&versionUUID,
			&versionCreatedAtInSec,
			&versionName,
			&versionParameters,
			&versionPipelineId,
			&versionStatus,
			&versionCodeSourceUrl,
			&versionDescription,
			&pipelineSpec,
			&pipelineSpecURI,
		); err != nil {
			return nil, nil, err
		}
		pipelines = append(
			pipelines,
			&model.Pipeline{
				UUID:           uuid,
				CreatedAtInSec: createdAtInSec,
				Name:           name,
				Description:    description,
				Status:         status,
				Namespace:      namespace.String,
			},
		)
		pipelineVersions = append(
			pipelineVersions,
			&model.PipelineVersion{
				UUID:            versionUUID.String,
				CreatedAtInSec:  versionCreatedAtInSec.Int64,
				Name:            versionName.String,
				Parameters:      versionParameters.String,
				PipelineId:      versionPipelineId.String,
				Status:          model.PipelineVersionStatus(versionStatus.String),
				CodeSourceUrl:   versionCodeSourceUrl.String,
				Description:     versionDescription.String,
				PipelineSpec:    pipelineSpec.String,
				PipelineSpecURI: pipelineSpecURI.String,
			},
		)
		// }
	}
	return pipelines, pipelineVersions, nil
}

// Inserts a record into `pipelines` table.
// Call CreatePipelineVersion to create a record in`pipeline_versions`.
func (s *PipelineStore) CreatePipeline(p *model.Pipeline) (*model.Pipeline, error) {
	newPipeline := *p

	// Set pipeline creation time
	newPipeline.CreatedAtInSec = s.time.Now().Unix()

	// Set pipeline id
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline UUID")
	}
	newPipeline.UUID = id.String()

	// Create a query for the KFP DB
	sql, args, err := sq.
		Insert("pipelines").
		SetMap(
			sq.Eq{
				"UUID":           newPipeline.UUID,
				"CreatedAtInSec": newPipeline.CreatedAtInSec,
				"Name":           newPipeline.Name,
				"Description":    newPipeline.Description,
				"Status":         string(newPipeline.Status),
				"Namespace":      newPipeline.Namespace,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert pipeline to pipeline table: %v",
			err.Error())
	}

	// Insert into pipelines table
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err,
			`PipelineStore: Failed to start a transaction to create a new pipeline: %v`,
			err.Error())
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			tx.Rollback()
			return nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline. The name %v already exist. Please specify a new name", p.Name)
		}
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err,
			`PipelineStore: Failed to commit pipeline creation in a SQL
			transaction: %v`, err.Error())
	}
	return &newPipeline, nil
}

// Creates a PipelineVersion.
func (s *PipelineStore) CreatePipelineVersion(pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	newPipelineVersion := *pv

	// Set pipeline version id
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to generate a pipeline version UUID")
	}
	newPipelineVersion.UUID = id.String()

	// Set creation time
	newPipelineVersion.CreatedAtInSec = s.time.Now().Unix()

	// Prepare a query for inserting new version
	versionSql, versionArgs, versionErr := sq.
		Insert("pipeline_versions").
		SetMap(
			sq.Eq{
				"UUID":            newPipelineVersion.UUID,
				"CreatedAtInSec":  newPipelineVersion.CreatedAtInSec,
				"Name":            newPipelineVersion.Name,
				"Parameters":      newPipelineVersion.Parameters,
				"PipelineId":      newPipelineVersion.PipelineId,
				"Status":          string(newPipelineVersion.Status),
				"CodeSourceUrl":   newPipelineVersion.CodeSourceUrl,
				"Description":     newPipelineVersion.Description,
				"PipelineSpec":    newPipelineVersion.PipelineSpec,
				"PipelineSpecURI": newPipelineVersion.PipelineSpecURI,
			},
		).
		ToSql()
	if versionErr != nil {
		return nil, util.NewInternalServerError(
			versionErr,
			"Failed to create query to insert a pipeline version: %v",
			versionErr.Error())
	}

	// Insert a new pipeline version.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to insert a new pipeline version: %v",
			err.Error())
	}
	_, err = tx.Exec(versionSql, versionArgs...)
	if err != nil {
		tx.Rollback()
		if s.db.IsDuplicateError(err) {
			return nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline version. The name %v already exist. Please specify a new name: %v", pv.Name, err.Error())
		}
		return nil, util.NewInternalServerError(err, "Failed to add a pipeline version: %v",
			err.Error())
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to create a new pipeline version: %v",
			err.Error())
	}
	return &newPipelineVersion, nil
}

// Returns a pipeline wit status = PipelineReady.
func (s *PipelineStore) GetPipeline(id string) (*model.Pipeline, error) {
	return s.GetPipelineWithStatus(id, model.PipelineReady)
}

// Returns a pipeline with a specified status.
// Changes behavior compare to v1beta1: does not join with a default pipeline version.
func (s *PipelineStore) GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error) {
	// Prepare the query
	sql, args, err := sq.
		Select(pipelineColumns...).
		From("pipelines").
		Where(sq.And{sq.Eq{"pipelines.UUID": id}, sq.Eq{"pipelines.Status": status}}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get a pipeline: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get a pipeline: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse results of getting a pipeline: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	return pipelines[0], nil
}

// Returns a pipeline version with status PipelineVersionReady.
func (s *PipelineStore) GetPipelineVersion(versionId string) (*model.PipelineVersion, error) {
	return s.GetPipelineVersionWithStatus(versionId, model.PipelineVersionReady)
}

// Returns the latest pipeline version with status PipelineVersionReady for a given pipeline id.
func (s *PipelineStore) GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error) {
	// Prepare a SQL query
	sql, args, err := sq.
		Select(pipelineVersionColumns...).
		From("pipeline_versions").
		Where(sq.And{sq.Eq{"pipeline_versions.PipelineId": pipelineId}, sq.Eq{"pipeline_versions.Status": model.PipelineVersionReady}}).
		OrderBy("pipeline_versions.CreatedAtInSec DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to fetch the latest pipeline version: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed fetching the latest pipeline version: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse the latest pipeline version from SQL response: %v", err.Error())
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("PipelineVersion", fmt.Sprint(pipelineId))
	}
	return versions[0], nil
}

// Returns a pipeline version with specified status.
func (s *PipelineStore) GetPipelineVersionWithStatus(versionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error) {
	// Prepare a SQL query
	sql, args, err := sq.
		Select(pipelineVersionColumns...).
		From("pipeline_versions").
		Where(sq.And{sq.Eq{"pipeline_versions.UUID": versionId}, sq.Eq{"pipeline_versions.Status": status}}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to fetch a pipeline version: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed fetching pipeline version: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse a pipeline version from SQL response: %v", err.Error())
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("PipelineVersion", fmt.Sprint(versionId))
	}
	return versions[0], nil
}

// Returns the latest pipeline specified by name and namespace.
// Performance depends on the index (name, namespace) in `pipelines` table.
func (s *PipelineStore) GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error) {
	sqlTemp := sq.
		Select(pipelineColumns...).
		From("pipelines").
		Where(sq.And{
			sq.Eq{"pipelines.Name": name},

			sq.Eq{"pipelines.Status": model.PipelineReady},
		})
	if len(namespace) > 0 {
		sqlTemp = sqlTemp.
			Where(
				sq.Eq{"pipelines.Namespace": namespace},
			)
	}
	sql, args, err := sqlTemp.
		OrderBy("pipelines.CreatedAtInSec DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get a pipeline by name and namespace: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get a pipeline by name and namespace: %v", err.Error())
	}
	defer r.Close()
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse results of GetPipelineByNameAndNamespace: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(name))
	}
	return pipelines[0], nil
}

// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size.
// This will not join with `pipeline_versions` table, hence, total_size is the size of pipelines, not pipeline_versions.
func (s *PipelineStore) ListPipelines(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		query := opts.AddFilterToSelect(sqlBuilder).From("pipelines")
		if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			query = query.Where(
				sq.Eq{
					"pipelines.Status":    model.PipelineReady,
					"pipelines.Namespace": filterContext.ReferenceKey.ID,
				},
			)
		} else {
			query = query.Where(
				sq.Eq{"pipelines.Status": model.PipelineReady},
			)
		}
		return query
	}

	// SQL for row list
	sqlSelect := buildQuery(sq.Select(pipelineColumns...))
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlSelect).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to list pipelines: %v", err.Error())
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to count pipelines: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to start transaction to list pipelines: %v", err.Error())
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	pipelines, err := s.scanPipelinesRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of listing pipelines: %v", err.Error())
	}
	defer rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines: %v", err.Error())
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines: %v", err.Error())
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of counting pipelines: %v", err.Error())
	}
	defer sizeRow.Close()

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to commit listing pipelines: %v", err.Error())
	}

	// Split results on multiple pages if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, totalSize, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	return pipelines[:opts.PageSize], totalSize, npt, err
}

// Fetches pipeline versions for a specified pipeline id.
func (s *PipelineStore) ListPipelineVersions(pipelineId string, opts *list.Options) (versions []*model.PipelineVersion, totalSize int, nextPageToken string, err error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		return opts.AddFilterToSelect(sqlBuilder).
			From("pipeline_versions").
			Where(
				sq.And{
					sq.Eq{"pipeline_versions.PipelineId": pipelineId},
					sq.Eq{"pipeline_versions.Status": model.PipelineVersionReady},
				},
			)
	}

	// Prepare a SQL query
	sqlSelect := buildQuery(sq.Select(pipelineVersionColumns...))
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlSelect).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query for listing pipeline versions: %v", err.Error())
	}

	// Query for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to count pipeline versions: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to begin SQL query listing pipeline versions")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to begin SQL query listing pipeline versions: %v", err.Error())
	}

	// Fetch the rows
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipeline versions: %v", err.Error())
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipeline versions: %v", err.Error())
	}
	pipelineVersions, err := s.scanPipelineVersionsRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of listing pipeline versions: %v", err.Error())
	}
	defer rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipeline versions: %v", err.Error())
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipeline versions: %v", err.Error())
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of counting pipeline versions: %v", err.Error())
	}
	defer sizeRow.Close()

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipeline versions")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to commit transaction to list pipeline versions: %v", err.Error())
	}

	// Split results on multiple pages if needed
	if len(pipelineVersions) <= opts.PageSize {
		return pipelineVersions, total_size, "", nil
	}
	npt, err := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelineVersions[:opts.PageSize], total_size, npt, err
}

// Executes a SQL query with arguments and throws standardized error messages.
func (s *PipelineStore) ExecuteSQL(sql string, args []interface{}, op string, obj string) error {
	// Execute the query
	_, err := s.db.Exec(sql, args...)
	if err != nil {
		// tx.Rollback()
		return util.NewInternalServerError(
			err,
			"Failed to execute a query to %v a (an) %v: %v",
			op,
			obj,
			err.Error(),
		)
	}
	return nil
}

// Updates status of a pipeline.
func (s *PipelineStore) UpdatePipelineStatus(id string, status model.PipelineStatus) error {
	// Prepare the query
	sql, args, err := sq.
		Update("pipelines").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to update the pipeline status: %s", err.Error())
	}
	return s.ExecuteSQL(sql, args, "update", "pipeline status")
}

// Updates status of a pipeline version.
func (s *PipelineStore) UpdatePipelineVersionStatus(id string, status model.PipelineVersionStatus) error {
	sql, args, err := sq.
		Update("pipeline_versions").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to update the status of a pipeline version: %s", err.Error())
	}
	return s.ExecuteSQL(sql, args, "update", "status of a pipeline version")
}

// TODO(gkcalat): consider removing before v2beta1 GA as default version is deprecated. This requires changes to v1beta1 proto.
// Updates default pipeline version for a given pipeline.
// Supports v1beta1 behavior.
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

// Removes a pipeline from the DB.
// DB should take care of the corresponding records in pipeline_versions.
func (s *PipelineStore) DeletePipeline(id string) error {
	// Prepare the query
	sql, args, err := sq.Delete("pipelines").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete a pipeline: %v", err.Error())
	}
	return s.ExecuteSQL(sql, args, "delete", "pipeline")
}

// Deletes a pipeline version.
// This does not update the default version update.
func (s *PipelineStore) DeletePipelineVersion(versionId string) error {
	// Prepare the query
	sql, args, err := sq.
		Delete("pipeline_versions").
		Where(sq.Eq{"UUID": versionId}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete a pipeline version: %v", err.Error())
	}
	return s.ExecuteSQL(sql, args, "delete", "pipeline version")
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Returns the latest pipeline and the latest pipeline version specified by name and namespace.
// Performance depends on the index (name, namespace) in `pipelines` table.
// This supports v1beta1 behavior.
func (s *PipelineStore) GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, *model.PipelineVersion, error) {
	sqlTemp := sq.
		Select(joinedColumns...).
		From("pipelines").
		LeftJoin("pipeline_versions on pipelines.UUID = pipeline_versions.PipelineId").
		Where(sq.And{
			sq.Eq{"pipelines.Name": name},
			sq.Eq{"pipelines.Status": model.PipelineReady},
		})
	if len(namespace) > 0 {
		sqlTemp = sqlTemp.Where(sq.Eq{"pipelines.Namespace": namespace})
	}
	sql, args, err := sqlTemp.
		OrderBy("pipeline_versions.CreatedAtInSec DESC", "pipelines.CreatedAtInSec DESC"). // In case of duplicate (name, namespace combination), this will return the latest PipelineVersion
		Limit(1).
		ToSql()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create query to get pipeline and pipeline version by name and namespace: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to get pipeline and pipeline version by name and namespace: %v", err.Error())
	}
	defer r.Close()
	pipelines, pipelineVersions, err := s.scanJoinedRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of GetPipelineByNameAndNamespaceV1: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, nil, util.NewResourceNotFoundError("Pipeline and PipelineVersion", fmt.Sprint(name))
	}
	return pipelines[0], pipelineVersions[0], nil
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size. Total_size reflects the number of pipeline_versions (not pipelines).
// This supports v1beta1 behavior.
func (s *PipelineStore) ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		query := opts.AddFilterToSelect(sqlBuilder).From("pipelines").
			LeftJoin("pipeline_versions ON pipelines.UUID = pipeline_versions.PipelineId") // this results in total_size reflecting the number of pipeline_versions
		if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			query = query.Where(
				sq.Eq{
					"pipelines.Status":    model.PipelineReady,
					"pipelines.Namespace": filterContext.ReferenceKey.ID,
				},
			)
		} else {
			query = query.Where(
				sq.Eq{"pipelines.Status": model.PipelineReady},
			)
		}
		return query
	}
	sqlBuilder := buildQuery(sq.Select(joinedColumns...))

	// SQL for row list
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to prepare a query to list pipelines: %v", err.Error())
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to prepare a query to count pipelines: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("PipelineStore (v1beta1): Failed to start transaction to list pipelines")
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to start transaction to list pipelines: %v", err.Error())
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	pipelines, pipelineVersions, err := s.scanJoinedRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of listing pipelines: %v", err.Error())
	}
	defer rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to count pipelines: %v", err.Error())
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to count pipelines: %v", err.Error())
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of counting pipelines: %v", err.Error())
	}
	defer sizeRow.Close()

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("PipelineStore (v1beta1): Failed to commit transaction to list pipelines")
		return nil, nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to commit listing pipelines: %v", err.Error())
	}

	// Split results on multiple pages, if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, pipelineVersions, total_size, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	// npt2, err2 := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelines[:opts.PageSize], pipelineVersions[:opts.PageSize], total_size, npt, err
}
