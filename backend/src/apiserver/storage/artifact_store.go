// Copyright 2025 The Kubeflow Authors
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

// Package storage provides the storage layer for the API server.
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

const artifactTableName = "artifacts"

var artifactColumns = []string{
	"UUID",
	"Namespace",
	"Type",
	"URI",
	"Name",
	"Description",
	"CreatedAtInSec",
	"LastUpdateInSec",
	"Metadata",
	"NumberValue",
}

// Ensure that ClientManager implements the resource.ClientManagerInterface interface.
var _ ArtifactStoreInterface = &ArtifactStore{}

type ArtifactStoreInterface interface {
	// Create an artifact entry in the database.
	CreateArtifact(artifact *model.Artifact) (*model.Artifact, error)

	// Fetches an artifact with a given id.
	GetArtifact(id string) (*model.Artifact, error)

	// Fetches artifacts for given filtering and listing options.
	ListArtifacts(filterContext *model.FilterContext, opts *list.Options) ([]*model.Artifact, int, string, error)
}

type ArtifactStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// NewArtifactStore creates a new ArtifactStore.
func NewArtifactStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *ArtifactStore {
	return &ArtifactStore{
		db:   db,
		time: time,
		uuid: uuid,
	}
}

func (s *ArtifactStore) CreateArtifact(artifact *model.Artifact) (*model.Artifact, error) {
	// Set up UUID for artifact.
	newArtifact := *artifact
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an artifact id")
	}
	newArtifact.UUID = id.String()

	// Set creation timestamps
	now := s.time.Now().Unix()
	newArtifact.CreatedAtInSec = now
	newArtifact.LastUpdateInSec = now

	// Convert metadata to JSON string for storage
	metadataJSON, err := newArtifact.Metadata.Value()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal artifact metadata")
	}

	sql, args, err := sq.
		Insert(artifactTableName).
		SetMap(
			sq.Eq{
				"UUID":            newArtifact.UUID,
				"Namespace":       newArtifact.Namespace,
				"Type":            newArtifact.Type,
				"URI":             newArtifact.URI,
				"Name":            newArtifact.Name,
				"Description":     newArtifact.Description,
				"CreatedAtInSec":  newArtifact.CreatedAtInSec,
				"LastUpdateInSec": newArtifact.LastUpdateInSec,
				"Metadata":        metadataJSON,
				"NumberValue":     newArtifact.NumberValue,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact to artifact table: %v",
			err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add artifact to artifact table: %v",
			err.Error())
	}

	return &newArtifact, nil
}

func (s *ArtifactStore) scanRows(rows *sql.Rows) ([]*model.Artifact, error) {
	var artifacts []*model.Artifact
	for rows.Next() {
		var uuid, namespace string
		var name, uri, description sql.NullString
		var artifactType int32
		var createdAtInSec, lastUpdateInSec int64
		var metadataBytes []byte
		var numberValue sql.NullFloat64

		err := rows.Scan(
			&uuid,
			&namespace,
			&artifactType,
			&uri,
			&name,
			&description,
			&createdAtInSec,
			&lastUpdateInSec,
			&metadataBytes,
			&numberValue,
		)
		if err != nil {
			return artifacts, err
		}

		// Parse metadata JSON
		var metadata model.JSONData
		if metadataBytes != nil {
			err = metadata.Scan(metadataBytes)
			if err != nil {
				return artifacts, util.NewInternalServerError(err, "Failed to parse artifact metadata")
			}
		}

		artifact := &model.Artifact{
			UUID:            uuid,
			Namespace:       namespace,
			Type:            model.ArtifactType(artifactType),
			Name:            name.String,
			Description:     description.String,
			CreatedAtInSec:  createdAtInSec,
			LastUpdateInSec: lastUpdateInSec,
			Metadata:        metadata,
		}
		if numberValue.Valid {
			artifact.NumberValue = &numberValue.Float64
		}

		if uri.Valid {
			artifact.URI = &uri.String
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func (s *ArtifactStore) ListArtifacts(filterContext *model.FilterContext, opts *list.Options) ([]*model.Artifact, int, string, error) {
	errorF := func(err error) ([]*model.Artifact, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list artifacts: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(artifactColumns...).From(artifactTableName)

	// Apply namespace filtering if provided
	if filterContext != nil && filterContext.ReferenceKey != nil {
		if filterContext.Type == model.NamespaceResourceType {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ID})
		}
	}

	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSQL, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := sq.Select("count(*)").From(artifactTableName)
	if filterContext != nil && filterContext.ReferenceKey != nil {
		if filterContext.Type == model.NamespaceResourceType {
			countBuilder = countBuilder.Where(sq.Eq{"Namespace": filterContext.ID})
		}
	}
	sizeSQL, sizeArgs, err := opts.AddFilterToSelect(countBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the totalSize of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list artifacts")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSQL, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	artifacts, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSQL, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list artifacts")
		return errorF(err)
	}

	if len(artifacts) <= opts.PageSize {
		return artifacts, totalSize, "", nil
	}

	npt, err := opts.NextPageToken(artifacts[opts.PageSize])
	return artifacts[:opts.PageSize], totalSize, npt, err
}

func (s *ArtifactStore) GetArtifact(id string) (*model.Artifact, error) {
	sql, args, err := sq.
		Select(artifactColumns...).
		From(artifactTableName).
		Where(sq.Eq{"UUID": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact: %v", err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact: %v", err.Error())
	}
	defer r.Close()

	artifacts, err := s.scanRows(r)
	if err != nil || len(artifacts) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get artifact: %v", err.Error())
	}
	if len(artifacts) == 0 {
		return nil, util.NewResourceNotFoundError("artifact", fmt.Sprint(id))
	}

	return artifacts[0], nil
}
