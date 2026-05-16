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
		"pipelines.DisplayName",
		"pipelines.Description",
		"pipelines.Status",
		"pipelines.Namespace",
		"pipeline_versions.UUID",
		"pipeline_versions.CreatedAtInSec",
		"pipeline_versions.Name",
		"pipeline_versions.DisplayName",
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
		"pipelines.DisplayName",
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
		"pipeline_versions.DisplayName",
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
	CreatePipelineAndPipelineVersion(pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) (*model.Pipeline, *model.PipelineVersion, error)

	// `pipelines`
	CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error)
	GetPipelineWithStatus(pipelineId string, status model.PipelineStatus) (*model.Pipeline, error)
	GetPipeline(pipelineId string) (*model.Pipeline, error)
	GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error)
	ListPipelines(filterContext *model.FilterContext, opts *list.Options, tagFilters map[string]string) ([]*model.Pipeline, int, string, error)
	UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error
	UpdatePipelineFields(pipelineID string, displayName string, tags map[string]string) error
	DeletePipeline(pipelineId string) error
	UpdatePipelineDefaultVersion(pipelineId string, versionId string) error

	// `pipeline_versions`
	CreatePipelineVersion(pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error)
	GetPipelineVersionWithStatus(pipelineVersionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error)
	GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error)
	GetPipelineVersionByName(name string) (*model.PipelineVersion, error)
	GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error)
	ListPipelineVersions(pipelineID string, opts *list.Options, tagFilters map[string]string) ([]*model.PipelineVersion, int, string, error)
	UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error
	UpdatePipelineVersionFields(pipelineVersionID string, displayName string, tags map[string]string) error
	DeletePipelineVersion(pipelineVersionId string) error

	// `pipeline_tags`
	CreateOrUpdatePipelineTags(pipelineID string, tags map[string]string) error
	GetPipelineTags(pipelineID string) (map[string]string, error)
	GetPipelineTagsForPipelines(pipelineIds []string) (map[string]map[string]string, error)
	DeletePipelineTags(pipelineID string) error

	// `pipeline_version_tags`
	CreateOrUpdatePipelineVersionTags(pipelineVersionID string, tags map[string]string) error
	GetPipelineVersionTags(pipelineVersionID string) (map[string]string, error)
	GetPipelineVersionTagsForVersions(pipelineVersionIds []string) (map[string]map[string]string, error)
	DeletePipelineVersionTags(pipelineVersionID string) error
}

type PipelineStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
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
		return nil, nil, util.NewInternalServerError(err, "Failed to create a query to get pipeline and pipeline version with name %v and namespace %v", name, namespace)
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to get pipeline and pipeline version with name %v and namespace %v", name, namespace)
	}
	defer r.Close()
	pipelines, pipelineVersions, err := s.scanJoinedRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, nil, util.NewInternalServerError(err, "Failed to parse results of fetching pipeline with name %v and namespace %v", name, namespace)
	}
	if len(pipelines) == 0 {
		return nil, nil, util.NewResourceNotFoundError("Namespace/Pipeline and PipelineVersion", fmt.Sprintf("%v/%v", namespace, name))
	}
	return pipelines[0], pipelineVersions[0], nil
}

// GetPipelineByNameAndNamespace returns the latest pipeline specified by name and namespace, including its tags.
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
		return nil, util.NewInternalServerError(err, "Failed to create query to get a pipeline with name %v and namespace %v", name, namespace)
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get a pipeline with name %v and namespace %v", name, namespace)
	}
	defer r.Close()
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse results of fetching a pipeline with name %v and namespace %v", name, namespace)
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Namespace/Pipeline", fmt.Sprintf("%v/%v", namespace, name))
	}
	pipeline := pipelines[0]
	tags, err := s.GetPipelineTags(pipeline.UUID)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to load tags for pipeline %v", pipeline.UUID)
	}
	pipeline.Tags = tags
	return pipeline, nil
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size. Total_size reflects the number of pipeline_versions (not pipelines).
// This supports v1beta1 behavior.
func (s *PipelineStore) ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	subQuery := sq.Select("t1.pvid, t1.pid").FromSelect(
		sq.Select("UUID AS pvid, PipelineId AS pid, ROW_NUMBER () OVER (PARTITION BY PipelineId ORDER BY CreatedAtInSec DESC) rn").
			From("pipeline_versions"), "t1").
		Where(sq.Or{sq.Eq{"rn": 1}, sq.Eq{"rn": nil}})

	buildQuery := func(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
		query := opts.AddFilterToSelect(sqlBuilder).From("pipelines").
			JoinClause(subQuery.Prefix("LEFT JOIN (").Suffix(") t2 ON pipelines.UUID = t2.pid")).
			LeftJoin("pipeline_versions ON t2.pvid = pipeline_versions.UUID")
		if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			query = query.Where(
				sq.Eq{
					"pipelines.Namespace": filterContext.ReferenceKey.ID,
				},
			)
		}
		query = query.Where(
			sq.Eq{"pipelines.Status": model.PipelineReady},
		)
		return query
	}
	sqlBuilder := buildQuery(sq.Select(joinedColumns...))

	// SQL for row list
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to list pipelines")
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSql, sizeArgs, err := buildQuery(sq.Select("count(*)")).ToSql()
	if err != nil {
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to count pipelines")
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list pipelines")
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to start transaction to list pipelines")
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines")
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines")
	}
	pipelines, pipelineVersions, err := s.scanJoinedRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of listing pipelines")
	}
	defer rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines")
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines")
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of counting pipelines")
	}
	defer sizeRow.Close()

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipelines")
		return nil, nil, 0, "", util.NewInternalServerError(err, "Failed to commit listing pipelines")
	}

	// Split results on multiple pages, if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, pipelineVersions, totalSize, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	// npt2, err2 := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelines[:opts.PageSize], pipelineVersions[:opts.PageSize], totalSize, npt, err
}

// Runs two SQL queries in a transaction to return a list of matching pipelines, as well as their
// total_size. The total_size does not reflect the page size.
// This will not join with `pipeline_versions` table, hence, total_size is the size of pipelines, not pipeline_versions.
// tagFilters is an optional map of tag key->value pairs to filter pipelines by. If nil or empty, no tag filtering is applied.
func (s *PipelineStore) ListPipelines(filterContext *model.FilterContext, opts *list.Options, tagFilters map[string]string) ([]*model.Pipeline, int, string, error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) (sq.SelectBuilder, error) {
		query := opts.AddFilterToSelect(sqlBuilder).From("pipelines")
		if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			query = query.Where(
				sq.Eq{
					"pipelines.Namespace": filterContext.ReferenceKey.ID,
				},
			)
		}
		query = query.Where(
			sq.Eq{"pipelines.Status": model.PipelineReady},
		)
		// Apply tag filters: for each tag, add a subquery ensuring the pipeline has that tag.
		for key, value := range tagFilters {
			subQuery := sq.Select("PipelineId").From("pipeline_tags").Where(sq.And{
				sq.Eq{"TagKey": key},
				sq.Eq{"TagValue": value},
			})
			subSQL, subArgs, subErr := subQuery.ToSql()
			if subErr != nil {
				return query, util.NewInternalServerError(subErr, "Failed to build tag filter subquery for tag key %q", key)
			}
			query = query.Where("pipelines.UUID IN ("+subSQL+")", subArgs...)
		}
		return query, nil
	}

	// SQL for row list
	sqlSelect, err := buildQuery(sq.Select(pipelineColumns...))
	if err != nil {
		return nil, 0, "", err
	}
	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlSelect).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to list pipelines")
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSelect, err := buildQuery(sq.Select("count(*)"))
	if err != nil {
		return nil, 0, "", err
	}
	sizeSQL, sizeArgs, err := sizeSelect.ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to count pipelines")
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to start transaction to list pipelines")
	}

	// Get pipelines
	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines")
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to execute SQL for listing pipelines")
	}
	pipelines, err := s.scanPipelinesRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of listing pipelines")
	}
	rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSQL, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines")
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipelines")
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of counting pipelines")
	}
	sizeRow.Close()

	// Load tags for the returned pipelines (inside the transaction for consistency)
	if len(pipelines) > 0 {
		pipelineIds := make([]string, len(pipelines))
		for i, p := range pipelines {
			pipelineIds[i] = p.UUID
		}
		tagsMap, err := queryTagsForEntities(tx, "pipeline_tags", "PipelineId", pipelineIds)
		if err != nil {
			tx.Rollback()
			return nil, 0, "", util.NewInternalServerError(err, "Failed to load tags for listed pipelines")
		}
		for _, p := range pipelines {
			if tags, ok := tagsMap[p.UUID]; ok {
				p.Tags = tags
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipelines")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to commit listing pipelines")
	}

	// Split results on multiple pages if needed
	if len(pipelines) <= opts.PageSize {
		return pipelines, totalSize, "", nil
	}
	npt, err := opts.NextPageToken(pipelines[opts.PageSize])
	return pipelines[:opts.PageSize], totalSize, npt, err
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Parses SQL results of joining `pipelines` and `pipeline_versions` tables into []Pipelines.
// This supports v1beta1 behavior.
func (s *PipelineStore) scanJoinedRows(rows *sql.Rows) ([]*model.Pipeline, []*model.PipelineVersion, error) {
	var pipelines []*model.Pipeline
	var pipelineVersions []*model.PipelineVersion
	for rows.Next() {
		var uuid, name, displayName, description string
		var namespace sql.NullString
		var status model.PipelineStatus
		var versionUUID, versionName, versionDisplayName, versionParameters, versionPipelineId, versionCodeSourceUrl, versionStatus, versionDescription, pipelineSpec, pipelineSpecURI sql.NullString
		var createdAtInSec, versionCreatedAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&displayName,
			&description,
			&status,
			&namespace,
			&versionUUID,
			&versionCreatedAtInSec,
			&versionName,
			&versionDisplayName,
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
				CreatedAtInSec: createdAtInSec.Int64,
				Name:           name,
				DisplayName:    displayName,
				Description:    model.LargeText(description),
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
				DisplayName:     versionDisplayName.String,
				Parameters:      model.LargeText(versionParameters.String),
				PipelineId:      versionPipelineId.String,
				Status:          model.PipelineVersionStatus(versionStatus.String),
				CodeSourceUrl:   versionCodeSourceUrl.String,
				Description:     model.LargeText(versionDescription.String),
				PipelineSpec:    model.LargeText(pipelineSpec.String),
				PipelineSpecURI: model.LargeText(pipelineSpecURI.String),
			},
		)
	}
	return pipelines, pipelineVersions, nil
}

// Converts SQL response into []Pipeline (default version is set to nil).
func (s *PipelineStore) scanPipelinesRows(rows *sql.Rows) ([]*model.Pipeline, error) {
	var pipelines []*model.Pipeline
	for rows.Next() {
		var uuid, name, displayName, status, description, namespace sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&displayName,
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
					DisplayName:    displayName.String,
					Description:    model.LargeText(description.String),
					Status:         model.PipelineStatus(status.String),
					Namespace:      namespace.String,
				},
			)
		}
	}
	return pipelines, nil
}

// GetPipeline returns a pipeline with status = PipelineReady, including its tags.
func (s *PipelineStore) GetPipeline(id string) (*model.Pipeline, error) {
	pipeline, err := s.GetPipelineWithStatus(id, model.PipelineReady)
	if err != nil {
		return nil, err
	}
	tags, err := s.GetPipelineTags(id)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to load tags for pipeline %v", id)
	}
	pipeline.Tags = tags
	return pipeline, nil
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
		return nil, util.NewInternalServerError(err, "Failed to create query to get a pipeline with id %v and status %v", id, string(status))
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get a pipeline with id %v and status %v", id, string(status))
	}
	defer r.Close()

	// Parse results
	pipelines, err := s.scanPipelinesRows(r)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse results of getting a pipeline with id %v and status %v", id, string(status))
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline", id)
	}
	return pipelines[0], nil
}

// Removes a pipeline from the DB.
// DB should take care of the corresponding records in pipeline_versions.
func (s *PipelineStore) DeletePipeline(id string) error {
	// Prepare the query
	sql, args, err := sq.Delete("pipelines").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete a pipeline with id %v", id)
	}
	return s.ExecuteSQL(sql, args, "delete", "pipeline")
}

// Creates a pipeline and a pipeline version in a single transaction.
func (s *PipelineStore) CreatePipelineAndPipelineVersion(p *model.Pipeline, pv *model.PipelineVersion) (*model.Pipeline, *model.PipelineVersion, error) {
	newPipeline := *p
	newPipelineVersion := *pv

	// Set creation time
	newPipeline.CreatedAtInSec = s.time.Now().Unix()
	newPipelineVersion.CreatedAtInSec = s.time.Now().Unix()

	// Set ids
	pID, err := s.uuid.NewRandom()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to create a pipeline UUID")
	}
	pvID, err := s.uuid.NewRandom()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to create a pipeline version UUID")
	}
	newPipeline.UUID = pID.String()
	newPipelineVersion.UUID = pvID.String()
	newPipelineVersion.PipelineId = newPipeline.UUID

	// Set temporal status. This needs to be updated in a follow-up call.
	newPipeline.Status = model.PipelineCreating
	newPipelineVersion.Status = model.PipelineVersionCreating

	// Create queries for the KFP DB
	pipelineSql, pipelineArgs, err := sq.
		Insert("pipelines").
		SetMap(
			sq.Eq{
				"UUID":           newPipeline.UUID,
				"CreatedAtInSec": newPipeline.CreatedAtInSec,
				"Name":           newPipeline.Name,
				"DisplayName":    newPipeline.DisplayName,
				"Description":    newPipeline.Description,
				"Status":         string(newPipeline.Status),
				"Namespace":      newPipeline.Namespace,
				// Parameters and DefaultVersionId are deprecated and set to empty string
				"DefaultVersionId": "",
				"Parameters":       "",
			},
		).
		ToSql()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to create query to insert a pipeline")
	}
	versionSql, versionArgs, err := sq.
		Insert("pipeline_versions").
		SetMap(
			sq.Eq{
				"UUID":            newPipelineVersion.UUID,
				"CreatedAtInSec":  newPipelineVersion.CreatedAtInSec,
				"Name":            newPipelineVersion.Name,
				"DisplayName":     newPipelineVersion.DisplayName,
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
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to create query to insert a pipeline version")
	}

	// Insert into pipelines table
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to start a transaction to create a new pipeline and a new pipeline version")
	}

	_, err = tx.Exec(pipelineSql, pipelineArgs...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			tx.Rollback()
			return nil, nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline. The name %v already exists. Please specify a new name", p.Name)
		}
		tx.Rollback()
		return nil, nil, util.NewInternalServerError(err, "Failed to insert a new pipeline")
	}

	_, err = tx.Exec(versionSql, versionArgs...)
	if err != nil {
		tx.Rollback()
		return nil, nil, util.NewInternalServerError(err, "Failed to insert a new pipeline version")
	}

	// Insert pipeline tags if provided
	if err := insertTagsInTx(tx, "pipeline_tags", "PipelineId", newPipeline.UUID, newPipeline.Tags); err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	// Insert pipeline version tags if provided
	if err := insertTagsInTx(tx, "pipeline_version_tags", "PipelineVersionId", newPipelineVersion.UUID, newPipelineVersion.Tags); err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to update pipelines and pipeline_versions in a transaction")
	}
	newPipelineVersion.Tags = pv.Tags
	return &newPipeline, &newPipelineVersion, nil
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
				"DisplayName":    newPipeline.DisplayName,
				"Description":    newPipeline.Description,
				"Status":         string(newPipeline.Status),
				"Namespace":      newPipeline.Namespace,
				// Parameters and DefaultVersionId are deprecated and set to empty string
				"DefaultVersionId": "",
				"Parameters":       "",
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert pipeline to pipeline table")
	}

	// Insert into pipelines table
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start a transaction to create a new pipeline")
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			tx.Rollback()
			return nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline. The name %v already exist. Please specify a new name", p.Name)
		}
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table")
	}

	// Insert tags if provided
	if err := insertTagsInTx(tx, "pipeline_tags", "PipelineId", newPipeline.UUID, newPipeline.Tags); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to commit pipeline creation in a SQL transaction")
	}
	return &newPipeline, nil
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
		return util.NewInternalServerError(err, "Failed to create query to update status to %v of pipeline %v", string(status), id)
	}
	return s.ExecuteSQL(sql, args, "update", "pipeline status")
}

// insertTagsInTx inserts tags into the specified tag table within the given transaction
// using a single batch INSERT statement. Returns an error if the insert fails; the
// caller is responsible for rolling back.
func insertTagsInTx(tx *sql.Tx, tableName, idColumn, entityID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	builder := sq.Insert(tableName).Columns(idColumn, "TagKey", "TagValue")
	for key, value := range tags {
		builder = builder.Values(entityID, key, value)
	}
	tagSQL, tagArgs, err := builder.ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to insert tags for %v %v", tableName, entityID)
	}
	if _, err := tx.Exec(tagSQL, tagArgs...); err != nil {
		return util.NewInternalServerError(err, "Failed to insert tags for %v %v", tableName, entityID)
	}
	return nil
}

// upsertTagsInTx upserts tags using INSERT ... ON DUPLICATE KEY UPDATE (MySQL)
// or INSERT ... ON CONFLICT DO UPDATE SET (SQLite) via the db.Upsert() abstraction.
// This updates TagValue in-place for existing keys without deleting and re-inserting
// the row, avoiding unnecessary row churn in replication binlogs and trigger side effects.
// The tag tables have a composite primary key on (idColumn, TagKey).
func (s *PipelineStore) upsertTagsInTx(tx *sql.Tx, tableName, idColumn, entityID string, tags map[string]string) error {
	for key, value := range tags {
		insertBuilder := sq.Insert(tableName).
			Columns(idColumn, "TagKey", "TagValue").
			Values(entityID, key, value)
		insertSQL, args, err := insertBuilder.ToSql()
		if err != nil {
			return util.NewInternalServerError(err, "Failed to build upsert query for tag %q in %v %v", key, tableName, entityID)
		}
		upsertSQL := s.db.Upsert(insertSQL, fmt.Sprintf("%s, TagKey", idColumn), true, "TagValue")
		if _, err := tx.Exec(upsertSQL, args...); err != nil {
			return util.NewInternalServerError(err, "Failed to upsert tag %q for %v %v", key, tableName, entityID)
		}
	}
	return nil
}

// updateEntityFields updates mutable fields (DisplayName) and tags for a pipeline
// or pipeline version in a single transaction.
// Tags are upserted using INSERT ... ON DUPLICATE KEY UPDATE (MySQL) or
// INSERT ... ON CONFLICT DO UPDATE SET (SQLite) for new/changed tags, followed
// by a DELETE of any stale keys whose keys are no longer in the provided set.
func (s *PipelineStore) updateEntityFields(entityTable, tagTable, idColumn, id, displayName string, tags map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to start transaction to update %v %v", entityTable, id)
	}
	if displayName != "" {
		sqlStr, args, err := sq.Update(entityTable).SetMap(sq.Eq{"DisplayName": displayName}).Where(sq.Eq{"UUID": id}).ToSql()
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Failed to create query to update %v %v", entityTable, id)
		}
		if _, err := tx.Exec(sqlStr, args...); err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Failed to update fields for %v %v", entityTable, id)
		}
	}
	// Only modify tags when explicitly provided (non-nil).
	// A nil tags value means "no change"; an empty map means "clear all tags".
	if tags != nil {
		// Upsert provided tags (insert new keys, update existing values).
		if err := s.upsertTagsInTx(tx, tagTable, idColumn, id, tags); err != nil {
			tx.Rollback()
			return err
		}
		// Delete stale tags whose keys are no longer in the provided set.
		deleteStaleQuery := sq.Delete(tagTable).Where(sq.Eq{idColumn: id})
		if len(tags) > 0 {
			tagKeys := make([]string, 0, len(tags))
			for key := range tags {
				tagKeys = append(tagKeys, key)
			}
			deleteStaleQuery = deleteStaleQuery.Where(sq.NotEq{"TagKey": tagKeys})
		}
		delSQL, delArgs, err := deleteStaleQuery.ToSql()
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Failed to create delete-stale query for %v tags %v", entityTable, id)
		}
		if _, err := tx.Exec(delSQL, delArgs...); err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Failed to delete stale tags for %v %v", entityTable, id)
		}
	}
	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(err, "Failed to commit update for %v %v", entityTable, id)
	}
	return nil
}

// UpdatePipelineFields updates mutable fields (DisplayName) and tags for a pipeline.
func (s *PipelineStore) UpdatePipelineFields(id string, displayName string, tags map[string]string) error {
	return s.updateEntityFields("pipelines", "pipeline_tags", "PipelineId", id, displayName, tags)
}

// UpdatePipelineVersionFields updates mutable fields (DisplayName) and tags for a pipeline version.
func (s *PipelineStore) UpdatePipelineVersionFields(id string, displayName string, tags map[string]string) error {
	return s.updateEntityFields("pipeline_versions", "pipeline_version_tags", "PipelineVersionId", id, displayName, tags)
}

// Updates status of a pipeline version.
func (s *PipelineStore) UpdatePipelineVersionStatus(id string, status model.PipelineVersionStatus) error {
	sql, args, err := sq.
		Update("pipeline_versions").
		SetMap(sq.Eq{"Status": status}).
		Where(sq.Eq{"UUID": id}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to update status to %v of a pipeline version %v", string(status), id)
	}
	return s.ExecuteSQL(sql, args, "update", "status of a pipeline version")
}

// Factory function for pipeline store.
func NewPipelineStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PipelineStore {
	return &PipelineStore{db: db, time: time, uuid: uuid}
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
				"DisplayName":     newPipelineVersion.DisplayName,
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
			"Failed to create query to insert a pipeline version")
	}

	// Insert a new pipeline version.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to insert a new pipeline version")
	}
	_, err = tx.Exec(versionSql, versionArgs...)
	if err != nil {
		tx.Rollback()
		if s.db.IsDuplicateError(err) {
			return nil, util.NewAlreadyExistError(
				"Failed to create a new pipeline version. The name %v already exist. Specify a new name", pv.Name)
		}
		return nil, util.NewInternalServerError(err, "Failed to add a pipeline version")
	}

	// Insert tags within the same transaction to prevent deadlocks caused by
	// opening a second transaction on foreign-key-related tables.
	if err := insertTagsInTx(tx, "pipeline_version_tags", "PipelineVersionId", newPipelineVersion.UUID, pv.Tags); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to create a new pipeline version")
	}
	newPipelineVersion.Tags = pv.Tags
	return &newPipelineVersion, nil
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
		return util.NewInternalServerError(err, "Failed to create query to update the default version to %v for pipeline %v", versionId, pipelineId)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the default version to %v for pipeline %v", versionId, pipelineId)
	}

	return nil
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
		return nil, util.NewInternalServerError(err, "Failed to create query to fetch the latest pipeline version for pipeline %v", pipelineId)
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed fetching the latest pipeline version for pipeline %v", pipelineId)
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse the latest pipeline version from SQL response for pipeline %v", pipelineId)
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("PipelineVersion", pipelineId)
	}
	version := versions[0]
	tags, err := s.GetPipelineVersionTags(version.UUID)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to load tags for latest pipeline version %v", version.UUID)
	}
	version.Tags = tags
	return version, nil
}

// GetPipelineVersion returns a pipeline version with status PipelineVersionReady, including its tags.
func (s *PipelineStore) GetPipelineVersion(versionId string) (*model.PipelineVersion, error) {
	version, err := s.GetPipelineVersionWithStatus(versionId, model.PipelineVersionReady)
	if err != nil {
		return nil, err
	}
	tags, err := s.GetPipelineVersionTags(versionId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to load tags for pipeline version %v", versionId)
	}
	version.Tags = tags
	return version, nil
}

func (s *PipelineStore) GetPipelineVersionByName(name string) (*model.PipelineVersion, error) {
	version, err := s.getPipelineVersionByCol("Name", name, model.PipelineVersionReady)
	if err != nil {
		return nil, err
	}
	tags, err := s.GetPipelineVersionTags(version.UUID)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to load tags for pipeline version %v", version.UUID)
	}
	version.Tags = tags
	return version, nil
}

// Returns a pipeline version with specified status.
func (s *PipelineStore) GetPipelineVersionWithStatus(versionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error) {
	return s.getPipelineVersionByCol("UUID", versionId, status)
}

// getPipelineVersionByCol retrieves a PipelineVersion filtered on
// colName with colVal and the given status. This is particularly
// useful for fetching pipeline Version by either UUID or Name columns.
func (s *PipelineStore) getPipelineVersionByCol(colName, colVal string, status model.PipelineVersionStatus) (*model.PipelineVersion, error) {
	// Prepare a SQL query
	sql, args, err := sq.
		Select(pipelineVersionColumns...).
		From("pipeline_versions").
		Where(sq.And{
			sq.Eq{fmt.Sprintf("pipeline_versions.%s", colName): colVal}, sq.Eq{"pipeline_versions.Status": status}}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to fetch a pipeline version with %v=%v and status=%v", colName, colVal, string(status))
	}

	// Execute the query
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed fetching pipeline version with %v=%v and status=%v", colName, colVal, string(status))
	}
	defer r.Close()

	// Parse results
	versions, err := s.scanPipelineVersionsRows(r)
	if err != nil || len(versions) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to parse a pipeline version from SQL response with %v=%v and status=%v", colName, colVal, string(status))
	}
	if len(versions) == 0 {
		return nil, util.NewResourceNotFoundError("PipelineVersion", colVal)
	}
	return versions[0], nil
}

// Converts SQL response into []PipelineVersion.
func (s *PipelineStore) scanPipelineVersionsRows(rows *sql.Rows) ([]*model.PipelineVersion, error) {
	var pipelineVersions []*model.PipelineVersion
	for rows.Next() {
		var uuid, name, displayName, parameters, pipelineId, codeSourceUrl, status, description, pipelineSpec, pipelineSpecURI sql.NullString
		var createdAtInSec sql.NullInt64
		if err := rows.Scan(
			&uuid,
			&createdAtInSec,
			&name,
			&displayName,
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
					DisplayName:     displayName.String,
					Parameters:      model.LargeText(parameters.String),
					PipelineId:      pipelineId.String,
					CodeSourceUrl:   codeSourceUrl.String,
					Status:          model.PipelineVersionStatus(status.String),
					Description:     model.LargeText(description.String),
					PipelineSpec:    model.LargeText(pipelineSpec.String),
					PipelineSpecURI: model.LargeText(pipelineSpecURI.String),
				},
			)
		}
	}
	return pipelineVersions, nil
}

// Fetches pipeline versions for a specified pipeline id.
// tagFilters is an optional map of tag key->value pairs to filter pipeline versions by. If nil or empty, no tag filtering is applied.
func (s *PipelineStore) ListPipelineVersions(pipelineID string, opts *list.Options, tagFilters map[string]string) (versions []*model.PipelineVersion, totalSize int, nextPageToken string, err error) {
	buildQuery := func(sqlBuilder sq.SelectBuilder) (sq.SelectBuilder, error) {
		query := opts.AddFilterToSelect(sqlBuilder).
			From("pipeline_versions").
			Where(
				sq.And{
					sq.Eq{"pipeline_versions.PipelineId": pipelineID},
					sq.Eq{"pipeline_versions.Status": model.PipelineVersionReady},
				},
			)
		// Apply tag filters: for each tag, add a subquery ensuring the pipeline version has that tag.
		for key, value := range tagFilters {
			subQuery := sq.Select("PipelineVersionId").From("pipeline_version_tags").Where(sq.And{
				sq.Eq{"TagKey": key},
				sq.Eq{"TagValue": value},
			})
			subSQL, subArgs, subErr := subQuery.ToSql()
			if subErr != nil {
				return query, util.NewInternalServerError(subErr, "Failed to build tag filter subquery for tag key %q", key)
			}
			query = query.Where("pipeline_versions.UUID IN ("+subSQL+")", subArgs...)
		}
		return query, nil
	}

	// Prepare a SQL query
	sqlSelect, err := buildQuery(sq.Select(pipelineVersionColumns...))
	if err != nil {
		return nil, 0, "", err
	}
	rowsSQL, rowsArgs, err := opts.AddPaginationToSelect(sqlSelect).ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query for listing pipeline versions for pipeline %v", pipelineID)
	}

	// Query for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sizeSelect, err := buildQuery(sq.Select("count(*)"))
	if err != nil {
		return nil, 0, "", err
	}
	sizeSQL, sizeArgs, err := sizeSelect.ToSql()
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to prepare a query to count pipeline versions for pipeline %v", pipelineID)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to begin SQL query listing pipeline versions")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to begin SQL query listing pipeline versions for pipeline %v", pipelineID)
	}

	// Fetch the rows
	rows, err := tx.Query(rowsSQL, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipeline versions for pipeline %v", pipelineID)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list pipeline versions for pipeline %v", pipelineID)
	}
	pipelineVersions, err := s.scanPipelineVersionsRows(rows)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of listing pipeline versions for pipeline %v", pipelineID)
	}
	rows.Close()

	// Count pipelines
	sizeRow, err := tx.Query(sizeSQL, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipeline versions for pipeline %v", pipelineID)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to count pipeline versions for pipeline %v", pipelineID)
	}
	totalSizeCount, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return nil, 0, "", util.NewInternalServerError(err, "Failed to parse results of counting pipeline versions for pipeline %v", pipelineID)
	}
	sizeRow.Close()

	// Load tags for the returned pipeline versions (inside the transaction for consistency)
	if len(pipelineVersions) > 0 {
		versionIDs := make([]string, len(pipelineVersions))
		for i, pv := range pipelineVersions {
			versionIDs[i] = pv.UUID
		}
		allTags, err := queryTagsForEntities(tx, "pipeline_version_tags", "PipelineVersionId", versionIDs)
		if err != nil {
			tx.Rollback()
			return nil, 0, "", util.NewInternalServerError(err, "Failed to load tags for listed pipeline versions")
		}
		for _, pv := range pipelineVersions {
			if tags, ok := allTags[pv.UUID]; ok {
				pv.Tags = tags
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list pipeline versions")
		return nil, 0, "", util.NewInternalServerError(err, "Failed to commit transaction to list pipeline versions for pipeline %v", pipelineID)
	}

	// Split results on multiple pages if needed
	if len(pipelineVersions) <= opts.PageSize {
		return pipelineVersions, totalSizeCount, "", nil
	}
	npt, err := opts.NextPageToken(pipelineVersions[opts.PageSize])
	return pipelineVersions[:opts.PageSize], totalSizeCount, npt, err
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
		return util.NewInternalServerError(err, "Failed to create query to delete a pipeline version %v", versionId)
	}
	return s.ExecuteSQL(sql, args, "delete", "pipeline version")
}

// SetUUIDGenerator is for unit tests in other packages who need to set uuid,
// since uuid is not exported.
func (s *PipelineStore) SetUUIDGenerator(newUUID util.UUIDGeneratorInterface) {
	s.uuid = newUUID
}

// Executes a SQL query with arguments and throws standardized error messages.
func (s *PipelineStore) ExecuteSQL(sql string, args []interface{}, op string, obj string) error {
	// Execute the query
	_, err := s.db.Exec(sql, args...)
	if err != nil {
		// tx.Rollback()
		return util.NewInternalServerError(
			err,
			"Failed to execute a query to %v a (an) %v",
			op,
			obj,
		)
	}
	return nil
}

// getTags returns all tags for a given entity from the specified table.
// Returns nil if no tags are found.
func (s *PipelineStore) getTags(tableName, idColumn, entityID string) (map[string]string, error) {
	sqlStr, args, err := sq.Select("TagKey", "TagValue").
		From(tableName).
		Where(sq.Eq{idColumn: entityID}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get tags for %v %v", tableName, entityID)
	}
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get tags for %v %v", tableName, entityID)
	}
	defer rows.Close()

	var tags map[string]string
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to scan tag row for %v %v", tableName, entityID)
		}
		if tags == nil {
			tags = make(map[string]string)
		}
		tags[key] = value
	}
	return tags, nil
}

// sqlQuerier is satisfied by both *sql.DB and *sql.Tx, allowing tag queries
// to run either standalone or inside an existing transaction.
type sqlQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// getTagsForEntities returns tags for multiple entities as a nested map: entityId -> {key: value}.
func (s *PipelineStore) getTagsForEntities(tableName, idColumn string, entityIds []string) (map[string]map[string]string, error) {
	return queryTagsForEntities(s.db, tableName, idColumn, entityIds)
}

// queryTagsForEntities loads tags using the provided sqlQuerier (either *sql.DB or *sql.Tx).
func queryTagsForEntities(q sqlQuerier, tableName, idColumn string, entityIds []string) (map[string]map[string]string, error) {
	if len(entityIds) == 0 {
		return make(map[string]map[string]string), nil
	}
	sqlStr, args, err := sq.Select(idColumn, "TagKey", "TagValue").
		From(tableName).
		Where(sq.Eq{idColumn: entityIds}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get tags for %v", tableName)
	}
	rows, err := q.Query(sqlStr, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get tags for %v", tableName)
	}
	defer rows.Close()

	result := make(map[string]map[string]string)
	for rows.Next() {
		var entityID, key, value string
		if err := rows.Scan(&entityID, &key, &value); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to scan tag row for %v", tableName)
		}
		if _, ok := result[entityID]; !ok {
			result[entityID] = make(map[string]string)
		}
		result[entityID][key] = value
	}
	return result, nil
}

// deleteTags removes all tags for a given entity from the specified table.
func (s *PipelineStore) deleteTags(tableName, idColumn, entityID string) error {
	sqlStr, args, err := sq.Delete(tableName).Where(sq.Eq{idColumn: entityID}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete tags for %v %v", tableName, entityID)
	}
	return s.ExecuteSQL(sqlStr, args, "delete", tableName)
}

// Pipeline tag methods delegate to the generic helpers.

func (s *PipelineStore) CreateOrUpdatePipelineTags(pipelineID string, tags map[string]string) error {
	return s.updateEntityFields("pipelines", "pipeline_tags", "PipelineId", pipelineID, "", tags)
}

func (s *PipelineStore) GetPipelineTags(pipelineID string) (map[string]string, error) {
	return s.getTags("pipeline_tags", "PipelineId", pipelineID)
}

func (s *PipelineStore) GetPipelineTagsForPipelines(pipelineIds []string) (map[string]map[string]string, error) {
	return s.getTagsForEntities("pipeline_tags", "PipelineId", pipelineIds)
}

func (s *PipelineStore) DeletePipelineTags(pipelineID string) error {
	return s.deleteTags("pipeline_tags", "PipelineId", pipelineID)
}

// Pipeline version tag methods delegate to the generic helpers.

func (s *PipelineStore) CreateOrUpdatePipelineVersionTags(pipelineVersionID string, tags map[string]string) error {
	return s.updateEntityFields("pipeline_versions", "pipeline_version_tags", "PipelineVersionId", pipelineVersionID, "", tags)
}

func (s *PipelineStore) GetPipelineVersionTags(pipelineVersionID string) (map[string]string, error) {
	return s.getTags("pipeline_version_tags", "PipelineVersionId", pipelineVersionID)
}

func (s *PipelineStore) GetPipelineVersionTagsForVersions(pipelineVersionIds []string) (map[string]map[string]string, error) {
	return s.getTagsForEntities("pipeline_version_tags", "PipelineVersionId", pipelineVersionIds)
}

func (s *PipelineStore) DeletePipelineVersionTags(pipelineVersionID string) error {
	return s.deleteTags("pipeline_version_tags", "PipelineVersionId", pipelineVersionID)
}
