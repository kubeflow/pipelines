// Copyright 2018-2022 The Kubeflow Authors
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
	// TODO (gkcalat): consider removing after KFP v2 GA if users are not affected.
	// `pipelines` joined with `pipeline_versions`
	// This supports v1beta1 behavior.
	// The order of the selected columns must match the order used in scan rows.
	joinedColumns = []string{
		"pipelines.UUID",
		"pipelines.CreatedAtInSec",
		"pipelines.Name",
		"pipelines.Description",
		// "pipelines.Parameters",
		"pipelines.Status",
		"pipelines.Namespace",
		// "pipelines.DefaultVersionId",
		// "pipelines.Url",
		"pipeline_versions.UUID",
		"pipeline_versions.CreatedAtInSec",
		"pipeline_versions.Name",
		"pipeline_versions.Parameters",
		"pipeline_versions.PipelineId",
		"pipeline_versions.Status",
		"pipeline_versions.CodeSourceUrl",
		"pipeline_versions.Description",
		// "pipeline_versions.PipelineSpec",
		// "pipeline_versions.PipelineSpecURI",
	}

	// `pipelines`
	// The order of the selected columns must match the order used in scan rows.
	pipelineColumns = []string{
		"pipelines.UUID",
		"pipelines.CreatedAtInSec",
		"pipelines.Name",
		"pipelines.Description",
		// "pipelines.Parameters",
		"pipelines.Status",
		"pipelines.Namespace",
		// "pipelines.DefaultVersionId",
		// "pipelines.Url",
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
	// TODO (gkcalat): As these calls use joins on two (potentially) large sets with one-many relationship,
	// let's keep them to avoid performance issues. consider removing after KFP v2 GA if users are not affected.
	//
	// `pipelines` left joined with `pipeline_versions`
	// This supports v1beta1 behavior.
	GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, error)
	ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error)

	// `pipelines`
	CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error)
	GetPipelineWithStatus(pipelineId string, status model.PipelineStatus) (*model.Pipeline, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error)
	ListPipelines(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error)
	UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error
	DeletePipeline(pipelineId string) error
	//UpdatePipelineDefaultVersionV1(pipelineId string, defaultPipelineVersionId string) error // Used in v1beta1 only

	// `pipeline_versions`
	CreatePipelineVersion(pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error)
	GetPipelineVersionWithStatus(pipelineVersionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error)
	GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error)
	GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error)
	ListPipelineVersions(pipelineId string, opts *list.Options) ([]*model.PipelineVersion, int, string, error)
	UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error
	DeletePipelineVersion(pipelineVersionId string) error
	//DeletePipelineVersionAndUpdateDefaultV1(pipelineVersionId string) error // Used in v1beta1 only
}

type PipelineStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// Factory function for pipeline store
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
}

// SetUUIDGenerator is for unit tests in other packages who need to set uuid,
// since uuid is not exported.
func (s *PipelineStore) SetUUIDGenerator(new_uuid util.UUIDGeneratorInterface) {
	s.uuid = new_uuid
}

// Converts SQL response into []Pipeline (default version is set to nil).
func (s *PipelineStore) scanPipelinesRows(rows *sql.Rows) ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	for rows.Next() {
		var uuid, name, status, description, namespace sql.NullString
		// var parameters, defaultVersionId, url sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&description,
			// &parameters,
			&status,
			&namespace,
			// &defaultVersionId,
			// &url,
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

// TODO (gkcalat): consider removing after KFP v2 GA if users are not affected.
// Parses SQL results of joining `pipelines` and `pipeline_versions` tables into []Pipelines.
// This supports v1beta1 behavior.
func (s *PipelineStore) scanJoinedRows(rows *sql.Rows) ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	for rows.Next() {
		var uuid, name, description string
		var namespace sql.NullString
		var createdAtInSec int64
		var status model.PipelineStatus
		var versionUUID, versionName, versionParameters, versionPipelineId, versionCodeSourceUrl, versionStatus, versionDescription sql.NullString
		var versionCreatedAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&description,
			// &parameters,
			&status,
			&namespace,
			// &defaultVersionId,
			&versionUUID,
			&versionCreatedAtInSec,
			&versionName,
			&versionParameters,
			&versionPipelineId,
			&versionStatus,
			&versionCodeSourceUrl,
			&versionDescription); err != nil {
			return nil, err
		}
		// if defaultVersionId.Valid {
		// 	pipelines = append(
		// 		pipelines,
		// 		&model.Pipeline{
		// 			UUID:           uuid,
		// 			CreatedAtInSec: createdAtInSec,
		// 			Name:           name,
		// 			Description:    description,
		// 			Status:         status,
		// 			Namespace:      namespace.String,
		// 		},
		// 	)
		// } else {
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
		// }
	}
	return pipelines, nil
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
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create a pipeline UUID.")
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
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create query to insert pipeline to pipeline table: %v",
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
				"PipelineStore: Failed to create a new pipeline. The name %v already exist. Please specify a new name.", p.Name)
		}
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to add pipeline to pipeline table: %v",
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

// Creates a PipelineVersion
func (s *PipelineStore) CreatePipelineVersion(pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	newPipelineVersion := *pv

	// Set pipeline version id
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to generate a pipeline version UUID.")
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
			"PipelineStore: Failed to create query to insert a pipeline version: %v",
			versionErr.Error())
	}

	// Insert a new pipeline version.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"PipelineStore: Failed to insert a new pipeline version: %v",
			err.Error())
	}
	_, err = tx.Exec(versionSql, versionArgs...)
	if err != nil {
		tx.Rollback()
		if s.db.IsDuplicateError(err) {
			return nil, util.NewAlreadyExistError(
				"PipelineStore: Failed to create a new pipeline version. The name %v already exist. Please specify a new name: %v", pv.Name, err.Error())
		}
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to add a pipeline version: %v",
			err.Error())
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create a new pipeline version: %v",
			err.Error())
	}
	return &newPipelineVersion, nil
}

// Returns a pipeline wit status = PipelineReady
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
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create query to get a pipeline: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to get a pipeline: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to parse results of getting a pipeline: %v", err.Error())
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
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create query to fetch the latest pipeline version: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed fetching the latest pipeline version: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to parse the latest pipeline version from SQL response: %v", err.Error())
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
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create query to fetch a pipeline version: %v", err.Error())
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed fetching pipeline version: %v", err.Error())
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to parse a pipeline version from SQL response: %v", err.Error())
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("PipelineVersion", fmt.Sprint(versionId))
	}
	return versions[0], nil
}

// Returns the latest pipeline specified by name and namespace.
// Performance depends on the index (name, namespace) in `pipelines` table.
func (s *PipelineStore) GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error) {
	sql, args, err := sq.
		Select(pipelineColumns...).
		From("pipelines").
		Where(sq.And{
			sq.Eq{"pipelines.Name": name},
			sq.Eq{"pipelines.Namespace": namespace},
			sq.Eq{"pipelines.Status": model.PipelineReady},
		}).
		OrderBy("pipelines.CreatedAtInSec DESC").
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to create query to get a pipeline by name and namespace: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to get a pipeline by name and namespace: %v", err.Error())
	}
	defer r.Close()
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "PipelineStore: Failed to parse results of GetPipelineByNameAndNamespace: %v", err.Error())
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
				sq.Eq{"pipelines.Status": model.PipelineReady,
					"pipelines.Namespace": filterContext.ReferenceKey.ID},
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
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to prepare a query to list pipelines: %v", err.Error())
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to prepare a query to count pipelines: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("PipelineStore: Failed to start transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to start transaction to list pipelines: %v", err.Error())
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	pipelines, err := s.scanPipelinesRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to parse results of listing pipelines: %v", err.Error())
	}
	rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to count pipelines: %v", err.Error())
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to parse results of counting pipelines: %v", err.Error())
	}
	sizeRow.Close()

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("PipelineStore: Failed to commit transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to commit listing pipelines: %v", err.Error())
	}

	// Split results on multiple pages if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, total_size, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	return pipelines[:opts.PageSize], total_size, npt, err
}

// Fetches pipeline versions for a specified pipeline id
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
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to prepare a query for listing pipeline versions: %v.", err.Error())
	}

	// Query for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to prepare a query to count pipeline versions: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("PipelineStore: Failed to begin SQL query listing pipeline versions.")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to begin SQL query listing pipeline versions: %v.", err.Error())
	}

	// Fetch the rows
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to list pipeline versions: %v.", err.Error())
	}
	pipelineVersions, err := s.scanPipelineVersionsRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to parse results of listing pipeline versions: %v.", err.Error())
	}
	rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to count pipeline versions: %v", err.Error())
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to parse results of counting pipeline versions: %v", err.Error())
	}
	sizeRow.Close()

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("PipelineStore: Failed to commit transaction to list pipeline versions")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore: Failed to commit transaction to list pipeline versions: %v.", err.Error())
	}

	// Split results on multiple pages if needed
	if len(pipelineVersions) <= opts.PageSize {
		return pipelineVersions, total_size, "", nil
	}
	npt, err := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelineVersions[:opts.PageSize], total_size, npt, err
}

// Executes a SQL query with arguments and throws standardized error messages
func (s *PipelineStore) ExecuteSQL(sql string, args []interface{}, op string, obj string) error {
	// Execute the query
	_, err := s.db.Exec(sql, args...)
	if err != nil {
		// tx.Rollback()
		return util.NewInternalServerError(
			err,
			"PipelineStore: Failed to execute a query to %v a (an) %v: %v",
			op,
			obj,
			err.Error(),
		)
	}
	return nil
}

// Updates status of a pipeline
func (s *PipelineStore) UpdatePipelineStatus(id string, status model.PipelineStatus) error {
	// Prepare the query
	sql, args, err := sq.
		Update("pipelines").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "PipelineStore: Failed to create query to update the pipeline status: %s", err.Error())
	}
	return s.ExecuteSQL(sql, args, "update", "pipeline status")
}

// Updates status of a pipeline version
func (s *PipelineStore) UpdatePipelineVersionStatus(id string, status model.PipelineVersionStatus) error {
	sql, args, err := sq.
		Update("pipeline_versions").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"PipelineStore: Failed to create query to update the status of a pipeline version: %s", err.Error())
	}
	return s.ExecuteSQL(sql, args, "update", "status of a pipeline version")
}

// Removes a pipeline from the DB.
// DB should take care of the corresponding records in pipeline_versions.
func (s *PipelineStore) DeletePipeline(id string) error {
	// Prepare the query
	sql, args, err := sq.Delete("pipelines").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "PipelineStore: Failed to create query to delete a pipeline: %v", err.Error())
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
		return util.NewInternalServerError(err, "PipelineStore: Failed to create query to delete a pipeline version: %v", err.Error())
	}
	return s.ExecuteSQL(sql, args, "delete", "pipeline version")
}

// TODO (gkcalat): consider removing after KFP v2 GA if users are not affected.
// Returns the latest pipeline and the latest pipeline version specified by name and namespace.
// Performance depends on the index (name, namespace) in `pipelines` table.
// This supports v1beta1 behavior.
func (s *PipelineStore) GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, error) {
	sql, args, err := sq.
		Select(joinedColumns...).
		From("pipelines").
		LeftJoin("pipeline_versions on pipelines.UUID = pipeline_versions.PipelineId").
		Where(sq.And{
			sq.Eq{"pipelines.Name": name},
			sq.Eq{"pipelines.Namespace": namespace},
			sq.Eq{"pipelines.Status": model.PipelineReady},
		}).
		OrderBy("pipeline_versions.CreatedAtInSec DESC", "pipelines.CreatedAtInSec DESC"). // In case of duplicate (name, namespace combination), this will return the latest PipelineVersion
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create query to get pipeline and pipeline version by name and namespace: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to get pipeline and pipeline version by name and namespace: %v", err.Error())
	}
	defer r.Close()
	pipelines, err := s.scanJoinedRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of GetPipelineByNameAndNamespaceV1: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline and PipelineVersion", fmt.Sprint(name))
	}
	return pipelines[0], nil
}

// TODO (gkcalat): consider removing after KFP v2 GA if users are not affected.
// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size. Total_size reflects the number of pipeline_versions (not pipelines).
// This supports v1beta1 behavior.
func (s *PipelineStore) ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		query := opts.AddFilterToSelect(sqlBuilder).From("pipelines").
			LeftJoin("pipeline_versions ON pipelines.UUID = pipeline_versions.PipelineId") // this results in total_size reflecting the number of pipeline_versions
		if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			query = query.Where(
				sq.Eq{"pipelines.Status": model.PipelineReady,
					"pipelines.Namespace": filterContext.ReferenceKey.ID},
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
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to prepare a query to list pipelines: %v", err.Error())
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to prepare a query to count pipelines: %v", err.Error())
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("PipelineStore (v1beta1): Failed to start transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to start transaction to list pipelines: %v", err.Error())
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to execute SQL for listing pipelines: %v", err.Error())
	}
	pipelines, err := s.scanJoinedRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of listing pipelines: %v", err.Error())
	}
	rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to count pipelines: %v", err.Error())
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to parse results of counting pipelines: %v", err.Error())
	}
	sizeRow.Close()

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("PipelineStore (v1beta1): Failed to commit transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to commit listing pipelines: %v", err.Error())
	}

	// Split results on multiple pages, if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, total_size, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	return pipelines[:opts.PageSize], total_size, npt, err
}

// // Deprecated in v2beta1
// // TODO (gkcalat): consider removing after KFP v2 GA if users are not affected
// // Updates the default pipeline version.
// // This is used in the v1beta1 flow of deleting a pipeline version and updating the default version to the latest one.
// func (s *PipelineStore) UpdatePipelineDefaultVersionV1(pipelineId string, versionId string) error {
// 	sql, args, err := sq.
// 		Update("pipelines").
// 		SetMap(sq.Eq{"DefaultVersionId": versionId}).
// 		Where(sq.Eq{"UUID": pipelineId}).
// 		ToSql()
// 	if err != nil {
// 		return util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create query to update the pipeline default version: %s", err.Error())
// 	}
// 	return s.ExecuteSQL(sql, args, "update", "default pipeline version (v1beta1)")
// }

// // Deprecated in v2beta1
// // TODO (gkcalat): consider removing after KFP v2 GA if users are not affected.
// // Deletes a pipeline version and updates the corresponding default pipeline version.
// func (s *PipelineStore) DeletePipelineVersionAndUpdateDefaultV1(versionId string) error {
// 	// If this version is used as default version for a pipeline, we have to
// 	// find a new default version for that pipeline, which is usually the latest
// 	// version of that pipeline. Then we'll have 3 operations in a single
// 	// transactions: (1) delete version (2) get new default version id (3) use
// 	// new default version id to update pipeline.
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return util.NewInternalServerError(
// 			err,
// 			"Failed to start an transaction while trying to delete pipeline version: %v",
// 			err.Error())
// 	}
// 	// (1) delete version.
// 	_, err = tx.Exec(
// 		"delete from pipeline_versions where UUID = ?",
// 		versionId)
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(
// 			err,
// 			"Failed to delete pipeline version: %v",
// 			err.Error())
// 	}
// 	// (2) check whether this version is used as default version.
// 	r, err := tx.Query(
// 		"select UUID from pipelines where DefaultVersionId = ?",
// 		versionId)
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(
// 			err,
// 			`Failed to query pipelines table while deleting pipeline version:
// 			%v`,
// 			err.Error())
// 	}
// 	var pipelineId = ""
// 	if r.Next() {
// 		if err := r.Scan(&pipelineId); err != nil {
// 			tx.Rollback()
// 			return util.NewInternalServerError(
// 				err,
// 				"Failed to get pipeline id for version id: %v",
// 				err.Error())
// 		}
// 	}
// 	r.Close()
// 	if len(pipelineId) == 0 {
// 		// The deleted version is not used as a default version. So no extra
// 		// work is needed. We commit the deletion now.
// 		if err := tx.Commit(); err != nil {
// 			return util.NewInternalServerError(
// 				err,
// 				"Failed to delete pipeline version: %v",
// 				err.Error())
// 		}
// 		return nil
// 	}
// 	// (3) find a new default version.
// 	r, err = tx.Query(
// 		`select UUID from pipeline_versions
// 		where PipelineId = ? and Status = ?
// 		order by CreatedAtInSec DESC
// 		limit 1`,
// 		pipelineId,
// 		model.PipelineVersionReady)
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(
// 			err,
// 			"Failed to get a new default version id: %v",
// 			err.Error())
// 	}
// 	var newDefaultVersionId = ""
// 	if r.Next() {
// 		if err := r.Scan(&newDefaultVersionId); err != nil {
// 			tx.Rollback()
// 			return util.NewInternalServerError(
// 				err,
// 				"Failed to get a new default version id: %v",
// 				err.Error())
// 		}
// 	}
// 	r.Close()
// 	if len(newDefaultVersionId) == 0 {
// 		// No new default version. The pipeline's default version id will be
// 		// null.
// 		_, err = tx.Exec(
// 			"update pipelines set DefaultVersionId = null where UUID = ?",
// 			pipelineId)
// 		if err != nil {
// 			tx.Rollback()
// 			return util.NewInternalServerError(
// 				err,
// 				"Failed to update pipeline's default version id: %v",
// 				err.Error())
// 		}
// 	} else {
// 		_, err = tx.Exec(
// 			"update pipelines set DefaultVersionId = ? where UUID = ?",
// 			newDefaultVersionId, pipelineId)
// 		if err != nil {
// 			tx.Rollback()
// 			return util.NewInternalServerError(
// 				err,
// 				"Failed to update pipeline's default version id: %v",
// 				err.Error())
// 		}
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return util.NewInternalServerError(
// 			err,
// 			"Failed to delete pipeline version: %v",
// 			err.Error())
// 	}
// 	return nil
// }

// // Deprecated in v2beta1
// // Updates status for a Pipeline and a PipelineVersion in as single transaction.
// // This supports v1beta1 behavior.
// func (s *PipelineStore) UpdatePipelineAndVersionsStatus(id string, status model.PipelineStatus, pipelineVersionId string, pipelineVersionStatus model.PipelineVersionStatus) error {
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return util.NewInternalServerError(
// 			err,
// 			"PipelineStore (v1beta1): Failed to Pipeline and PipelineVersion status: %s",
// 			err.Error())
// 	}
// 	sql, args, err := sq.
// 		Update("pipelines").
// 		SetMap(sq.Eq{"Status": status}).
// 		Where(sq.Eq{"UUID": id}).
// 		ToSql()
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create query to update the pipeline status: %s", err.Error())
// 	}
// 	_, err = tx.Exec(sql, args...)
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to update the pipeline status: %s", err.Error())
// 	}
// 	sql, args, err = sq.
// 		Update("pipeline_versions").
// 		SetMap(sq.Eq{"Status": pipelineVersionStatus}).
// 		Where(sq.Eq{"UUID": pipelineVersionId}).
// 		ToSql()
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(err,
// 			`PipelineStore (v1beta1): Failed to create query to update the pipeline version
// 			status: %s`, err.Error())
// 	}
// 	_, err = tx.Exec(sql, args...)
// 	if err != nil {
// 		tx.Rollback()
// 		return util.NewInternalServerError(err,
// 			"PipelineStore (v1beta1): Failed to update the pipeline version status: %s", err.Error())
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return util.NewInternalServerError(err,
// 			"PipelineStore (v1beta1): Failed to update pipeline status and its version status: %v", err)
// 	}
// 	return nil
// }

// // Deprecated in v2beta1
// // Inserts records into both `pipelines` and `pipeline_versions` tables.
// // This supports v1beta1 behavior where pipelines and pipeline_versions tables get updated in a single transaction.
// func (s *PipelineStore) CreatePipelineV1(p *model.Pipeline) (*model.Pipeline, error) {
// 	// Set up creation time, UUID and sql query for pipeline.
// 	newPipeline := *p
// 	now := s.time.Now().Unix()
// 	newPipeline.CreatedAtInSec = now
// 	id, err := s.uuid.NewRandom()
// 	if err != nil {
// 		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create a pipeline id.")
// 	}
// 	newPipeline.UUID = id.String()
// 	// TODO(jingzhang36): remove default version id assignment after version API
// 	// is ready.
// 	newPipeline.DefaultVersionId = id.String()
// 	sql, args, err := sq.
// 		Insert("pipelines").
// 		SetMap(
// 			sq.Eq{
// 				"UUID":             newPipeline.UUID,
// 				"CreatedAtInSec":   newPipeline.CreatedAtInSec,
// 				"Name":             newPipeline.Name,
// 				"Description":      newPipeline.Description,
// 				"Parameters":       newPipeline.Parameters,
// 				"Status":           string(newPipeline.Status),
// 				"Namespace":        newPipeline.Namespace,
// 				"DefaultVersionId": newPipeline.DefaultVersionId}).
// 		ToSql()
// 	if err != nil {
// 		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to create query to insert pipeline to pipeline table: %v",
// 			err.Error())
// 	}
// 	// Set up creation time, UUID and sql query for pipeline.
// 	// TODO(jingzhang36): remove version related operations from CreatePipeline
// 	// when version API is ready. Before that we create an implicit version
// 	// inside CreatePipeline method. And this implicit version has the same UUID
// 	// as pipeline; and thus FE can use either pipeline UUID or version UUID to
// 	// retrieve pipeline package.
// 	if newPipeline.DefaultVersion == nil {
// 		newPipeline.DefaultVersion = &model.PipelineVersion{
// 			Name:          newPipeline.Name,
// 			Parameters:    newPipeline.Parameters,
// 			Status:        model.PipelineVersionCreating,
// 			CodeSourceUrl: ""}
// 	}
// 	newPipeline.DefaultVersion.CreatedAtInSec = now
// 	newPipeline.DefaultVersion.PipelineId = id.String()
// 	newPipeline.DefaultVersion.UUID = id.String()
// 	sqlPipelineVersions, argsPipelineVersions, err := sq.
// 		Insert("pipeline_versions").
// 		SetMap(
// 			sq.Eq{
// 				"UUID":           newPipeline.DefaultVersion.UUID,
// 				"CreatedAtInSec": newPipeline.DefaultVersion.CreatedAtInSec,
// 				"Name":           newPipeline.DefaultVersion.Name,
// 				"Parameters":     newPipeline.DefaultVersion.Parameters,
// 				"Status":         string(newPipeline.DefaultVersion.Status),
// 				"PipelineId":     newPipeline.UUID,
// 				"Description":    newPipeline.DefaultVersion.Description,
// 				"CodeSourceUrl":  newPipeline.DefaultVersion.CodeSourceUrl}).
// 		ToSql()
// 	if err != nil {
// 		return nil, util.NewInternalServerError(err,
// 			"PipelineStore (v1beta1): Failed to create query to insert pipeline version to pipeline_versions table: %v", err.Error())
// 	}
// 	// In a transaction, we insert into both pipelines and pipeline_versions.
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return nil, util.NewInternalServerError(err,
// 			"PipelineStore (v1beta1): Failed to start a transaction to create a new pipeline: %v",
// 			err.Error())
// 	}
// 	_, err = tx.Exec(sql, args...)
// 	if err != nil {
// 		if s.db.IsDuplicateError(err) {
// 			tx.Rollback()
// 			return nil, util.NewAlreadyExistError(
// 				"PipelineStore (v1beta1): Failed to create a new pipeline. The name %v already exist. Please specify a new name.", p.Name)
// 		}
// 		tx.Rollback()
// 		return nil, util.NewInternalServerError(err, "PipelineStore (v1beta1): Failed to add pipeline to pipeline table: %v",
// 			err.Error())
// 	}
// 	_, err = tx.Exec(sqlPipelineVersions, argsPipelineVersions...)
// 	if err != nil {
// 		if s.db.IsDuplicateError(err) {
// 			tx.Rollback()
// 			return nil, util.NewAlreadyExistError(
// 				`PipelineStore (v1beta1): Failed to create a new pipeline version. The name %v already
// 				exist. Please specify a new name.`, p.DefaultVersion.Name)
// 		}
// 		tx.Rollback()
// 		return nil, util.NewInternalServerError(err,
// 			"PipelineStore (v1beta1): Failed to add pipeline version to pipeline_versions table: %v",
// 			err.Error())
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return nil, util.NewInternalServerError(err,
// 			`PipelineStore (v1beta1): Failed to update pipelines and pipeline_versions in a
// 			transaction: %v`, err.Error())
// 	}
// 	return &newPipeline, nil
// }
