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
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

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
	"URIHash",
	"Name",
	"Description",
	"CreatedAtInSec",
	"LastUpdateInSec",
	"Metadata",
	"NumberValue",
	"IdentityKey",
}

// artifactURIHash returns the SHA-256 hex digest of uri, or "" when uri is empty.
func artifactURIHash(uri string) string {
	if uri == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(uri))
	return hex.EncodeToString(sum[:])
}

// Ensure that ClientManager implements the resource.ClientManagerInterface interface.
var _ ArtifactStoreInterface = &ArtifactStore{}

type ArtifactStoreInterface interface {
	// CreateArtifact creates an artifact entry in the database.
	CreateArtifact(artifact *model.Artifact) (*model.Artifact, error)

	// CreateArtifactWithTask atomically creates an artifact row and its output link.
	CreateArtifactWithTask(artifact *model.Artifact, artifactTask *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error)

	// CreateArtifactsWithTasks atomically creates a batch of artifacts and output links.
	CreateArtifactsWithTasks(artifacts []*model.Artifact, artifactTasks []*model.ArtifactTask) ([]*model.Artifact, []*model.ArtifactTask, error)

	// FindOrCreateArtifactWithTask atomically reuses an existing artifact that matches the
	// stable reuse identity, or creates one when no match exists. Concurrent callers that
	// race on the same identity share one artifact row and each still get their own link.
	FindOrCreateArtifactWithTask(artifact *model.Artifact, artifactTask *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error)

	// GetArtifact fetches an artifact with a given id.
	GetArtifact(id string) (*model.Artifact, error)

	// GetArtifactsByURI fetches artifacts with exact Namespace + URI equality.
	// This is a dedicated lookup path that avoids paginated ListArtifacts + COUNT.
	GetArtifactsByURI(namespace, uri string) ([]*model.Artifact, error)

	// ListArtifacts fetches artifacts for given filtering and listing options.
	// It returns the current page of artifacts, the total count across all pages,
	// the next page token, and an error.
	ListArtifacts(filterContext *model.FilterContext, opts *list.Options) ([]*model.Artifact, int, string, error)
}

type ArtifactStore struct {
	db                              *DB
	time                            util.TimeInterface
	uuid                            util.UUIDGeneratorInterface
	createArtifactTaskInTransaction func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error)
}

// NewArtifactStore creates a new ArtifactStore.
func NewArtifactStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *ArtifactStore {
	store := &ArtifactStore{
		db:   db,
		time: time,
		uuid: uuid,
	}
	store.createArtifactTaskInTransaction = func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
		return createArtifactTaskWithExecutor(tx.Exec, store.uuid, artifactTask)
	}
	return store
}

func (s *ArtifactStore) CreateArtifact(artifact *model.Artifact) (*model.Artifact, error) {
	return s.createArtifactWithExecutor(s.db.Exec, artifact)
}

func (s *ArtifactStore) createArtifactWithExecutor(exec func(string, ...any) (sql.Result, error), artifact *model.Artifact) (*model.Artifact, error) {
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

	uri := ""
	if newArtifact.URI != nil {
		uri = *newArtifact.URI
	}
	newArtifact.URIHash = artifactURIHash(uri)

	sql, args, err := sq.
		Insert(artifactTableName).
		SetMap(
			sq.Eq{
				"UUID":            newArtifact.UUID,
				"Namespace":       newArtifact.Namespace,
				"Type":            newArtifact.Type,
				"URI":             newArtifact.URI,
				"URIHash":         newArtifact.URIHash,
				"Name":            newArtifact.Name,
				"Description":     newArtifact.Description,
				"CreatedAtInSec":  newArtifact.CreatedAtInSec,
				"LastUpdateInSec": newArtifact.LastUpdateInSec,
				"Metadata":        metadataJSON,
				"NumberValue":     newArtifact.NumberValue,
				"IdentityKey":     newArtifact.IdentityKey,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact to artifact table: %v",
			err.Error())
	}

	_, err = exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add artifact to artifact table: %v",
			err.Error())
	}

	return &newArtifact, nil
}

// CreateArtifactWithTask atomically creates an artifact row and its output link.
// Keeping this transaction inside storage preserves the store-first boundary and
// prevents callers from reaching into the raw DB just to keep `artifacts` and
// `artifact_tasks` in sync for one logical API operation.
func (s *ArtifactStore) CreateArtifactWithTask(artifact *model.Artifact, artifactTask *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to start transaction for creating artifact and artifact-task")
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact create transaction: %v", rbErr)
		}
	}()

	newArtifact, err := s.createArtifactWithExecutor(tx.Exec, artifact)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create artifact")
	}

	artifactTaskCopy := *artifactTask
	artifactTaskCopy.ArtifactID = newArtifact.UUID
	newArtifactTask, err := s.createArtifactTaskInTransaction(tx, &artifactTaskCopy)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create artifact-task relationship")
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to commit transaction for creating artifact and artifact-task")
	}
	return newArtifact, newArtifactTask, nil
}

// CreateArtifactsWithTasks atomically creates a batch of artifacts and output links.
// This method is intentionally all-or-nothing so a later artifact-task failure
// cannot leave earlier artifacts committed without their matching link rows.
func (s *ArtifactStore) CreateArtifactsWithTasks(artifacts []*model.Artifact, artifactTasks []*model.ArtifactTask) ([]*model.Artifact, []*model.ArtifactTask, error) {
	if len(artifacts) != len(artifactTasks) {
		return nil, nil, util.NewInvalidInputError("artifacts and artifactTasks must have the same length")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to start transaction for bulk artifact creation")
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback bulk artifact create transaction: %v", rbErr)
		}
	}()

	createdArtifacts := make([]*model.Artifact, 0, len(artifacts))
	createdArtifactTasks := make([]*model.ArtifactTask, 0, len(artifactTasks))
	for index, artifact := range artifacts {
		newArtifact, err := s.createArtifactWithExecutor(tx.Exec, artifact)
		if err != nil {
			return nil, nil, util.Wrapf(err, "Failed to create artifact %d", index)
		}
		createdArtifacts = append(createdArtifacts, newArtifact)

		artifactTaskCopy := *artifactTasks[index]
		artifactTaskCopy.ArtifactID = newArtifact.UUID
		newArtifactTask, err := s.createArtifactTaskInTransaction(tx, &artifactTaskCopy)
		if err != nil {
			return nil, nil, util.Wrapf(err, "Failed to create artifact-task relationship %d", index)
		}
		createdArtifactTasks = append(createdArtifactTasks, newArtifactTask)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to commit transaction for bulk artifact creation")
	}
	return createdArtifacts, createdArtifactTasks, nil
}

// FindOrCreateArtifactWithTask reuses an artifact that matches the stable reuse identity
// or creates one. IdentityKey uniqueness makes concurrent reimport=false creates share one
// row while unconditional creates leave IdentityKey NULL and may intentionally duplicate.
func (s *ArtifactStore) FindOrCreateArtifactWithTask(artifact *model.Artifact, artifactTask *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error) {
	if artifact == nil {
		return nil, nil, util.NewInvalidInputError("artifact is required")
	}
	if artifactTask == nil {
		return nil, nil, util.NewInvalidInputError("artifactTask is required")
	}

	identityKey, err := computeArtifactIdentityKey(artifact)
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to compute artifact identity key")
	}

	existingArtifact, err := s.getArtifactByIdentityKey(artifact.Namespace, identityKey)
	if err != nil {
		return nil, nil, err
	}
	if existingArtifact != nil {
		return s.linkExistingArtifact(existingArtifact, artifactTask)
	}

	uri := ""
	if artifact.URI != nil {
		uri = *artifact.URI
	}
	if uri != "" {
		candidates, err := s.GetArtifactsByURI(artifact.Namespace, uri)
		if err != nil {
			return nil, nil, err
		}
		for _, candidate := range candidates {
			if modelArtifactsEqualForReuse(artifact, candidate) {
				return s.linkExistingArtifact(candidate, artifactTask)
			}
		}
	}

	artifactToCreate := *artifact
	artifactToCreate.IdentityKey = &identityKey
	createdArtifact, createdArtifactTask, err := s.CreateArtifactWithTask(&artifactToCreate, artifactTask)
	if err == nil {
		return createdArtifact, createdArtifactTask, nil
	}

	// Another concurrent writer may have inserted the same identity key first.
	existingArtifact, findErr := s.getArtifactByIdentityKey(artifact.Namespace, identityKey)
	if findErr == nil && existingArtifact != nil {
		return s.linkExistingArtifact(existingArtifact, artifactTask)
	}
	return nil, nil, err
}

// linkExistingArtifact attaches an artifact-task link to an already-persisted artifact.
// Duplicate deliveries for the same logical UniqueLink are treated as success so importer
// retries remain idempotent after TaskStore collapses duplicate task creates.
func (s *ArtifactStore) linkExistingArtifact(existingArtifact *model.Artifact, artifactTask *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error) {
	artifactTaskCopy := *artifactTask
	artifactTaskCopy.ArtifactID = existingArtifact.UUID
	if err := artifactTaskCopy.SyncIterationFromProducer(); err != nil {
		return nil, nil, util.NewInternalServerError(err, "Failed to derive artifact-task iteration: %v", err.Error())
	}

	existingLink, err := s.getArtifactTaskByUniqueLink(&artifactTaskCopy)
	if err != nil {
		return nil, nil, err
	}
	if existingLink != nil {
		return existingArtifact, existingLink, nil
	}

	createdLink, err := createArtifactTaskWithExecutor(s.db.Exec, s.uuid, &artifactTaskCopy)
	if err == nil {
		return existingArtifact, createdLink, nil
	}

	existingLink, findErr := s.getArtifactTaskByUniqueLink(&artifactTaskCopy)
	if findErr == nil && existingLink != nil {
		return existingArtifact, existingLink, nil
	}
	return nil, nil, err
}

func (s *ArtifactStore) getArtifactTaskByUniqueLink(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
	if artifactTask == nil {
		return nil, nil
	}
	sql, args, err := sq.
		Select(
			"UUID",
			"ArtifactID",
			"TaskID",
			"Type",
			"Iteration",
			"RunUUID",
			"Producer",
			"ArtifactKey",
		).
		From(artifactTaskTableName).
		Where(sq.Eq{
			"ArtifactID":  artifactTask.ArtifactID,
			"TaskID":      artifactTask.TaskID,
			"Type":        artifactTask.Type,
			"Iteration":   artifactTask.Iteration,
			"ArtifactKey": artifactTask.ArtifactKey,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact-task by unique link: %v", err.Error())
	}
	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact-task by unique link: %v", err.Error())
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	var (
		uuid, artifactID, taskID, runUUID, artifactKey string
		ioType                                         int32
		iteration                                      int64
		producerBytes                                  []byte
	)
	if err := rows.Scan(&uuid, &artifactID, &taskID, &ioType, &iteration, &runUUID, &producerBytes, &artifactKey); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to scan artifact-task by unique link: %v", err.Error())
	}
	var producer model.JSONData
	if producerBytes != nil {
		if err := producer.Scan(producerBytes); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to parse artifact-task producer: %v", err.Error())
		}
	}
	return &model.ArtifactTask{
		UUID:        uuid,
		ArtifactID:  artifactID,
		TaskID:      taskID,
		Type:        model.IOType(ioType),
		Iteration:   iteration,
		RunUUID:     runUUID,
		Producer:    producer,
		ArtifactKey: artifactKey,
	}, nil
}

func computeArtifactIdentityKey(artifact *model.Artifact) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("artifact is nil")
	}
	uri := ""
	if artifact.URI != nil {
		uri = *artifact.URI
	}

	var identity bytes.Buffer
	if err := binary.Write(&identity, binary.BigEndian, int32(artifact.Type)); err != nil {
		return "", err
	}
	for _, value := range []string{uri, artifact.Name, artifact.Description} {
		if err := writeLengthPrefixedString(&identity, value); err != nil {
			return "", err
		}
	}
	metadataBytes, err := canonicalArtifactMetadataBytes(artifact.Metadata)
	if err != nil {
		return "", err
	}
	if err := writeLengthPrefixedString(&identity, string(metadataBytes)); err != nil {
		return "", err
	}
	digest := sha256.Sum256(identity.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

func canonicalArtifactMetadataBytes(metadata model.JSONData) ([]byte, error) {
	if metadata == nil {
		return []byte("{}"), nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]interface{}, len(metadata))
	for _, key := range keys {
		ordered[key] = metadata[key]
	}
	return json.Marshal(ordered)
}

func modelArtifactsEqualForReuse(left, right *model.Artifact) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Type != right.Type {
		return false
	}
	leftURI := ""
	if left.URI != nil {
		leftURI = *left.URI
	}
	rightURI := ""
	if right.URI != nil {
		rightURI = *right.URI
	}
	if leftURI != rightURI {
		return false
	}
	if left.Name != right.Name || left.Description != right.Description {
		return false
	}
	leftMetadata, err := canonicalArtifactMetadataBytes(left.Metadata)
	if err != nil {
		return false
	}
	rightMetadata, err := canonicalArtifactMetadataBytes(right.Metadata)
	if err != nil {
		return false
	}
	return bytes.Equal(leftMetadata, rightMetadata)
}

func (s *ArtifactStore) getArtifactByIdentityKey(namespace, identityKey string) (*model.Artifact, error) {
	if identityKey == "" {
		return nil, nil
	}
	sql, args, err := sq.
		Select(artifactColumns...).
		From(artifactTableName).
		Where(sq.Eq{
			"Namespace":   namespace,
			"IdentityKey": identityKey,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact by identity key: %v", err.Error())
	}
	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact by identity key: %v", err.Error())
	}
	defer rows.Close()
	artifacts, err := s.scanRows(rows)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to scan artifact by identity key: %v", err.Error())
	}
	if len(artifacts) == 0 {
		return nil, nil
	}
	return artifacts[0], nil
}

func (s *ArtifactStore) scanRows(rows *sql.Rows) ([]*model.Artifact, error) {
	var artifacts []*model.Artifact
	for rows.Next() {
		var uuid, namespace string
		var name, uri, uriHash, description, identityKey sql.NullString
		var artifactType int32
		var createdAtInSec, lastUpdateInSec int64
		var metadataBytes []byte
		var numberValue sql.NullFloat64

		err := rows.Scan(
			&uuid,
			&namespace,
			&artifactType,
			&uri,
			&uriHash,
			&name,
			&description,
			&createdAtInSec,
			&lastUpdateInSec,
			&metadataBytes,
			&numberValue,
			&identityKey,
		)
		if err != nil {
			return nil, err
		}

		// Parse metadata JSON
		var metadata model.JSONData
		if metadataBytes != nil {
			err = metadata.Scan(metadataBytes)
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to parse artifact metadata")
			}
		}

		artifact := &model.Artifact{
			UUID:            uuid,
			Namespace:       namespace,
			Type:            model.ArtifactType(artifactType),
			URIHash:         uriHash.String,
			Name:            name.String,
			Description:     description.String,
			CreatedAtInSec:  createdAtInSec,
			LastUpdateInSec: lastUpdateInSec,
			Metadata:        metadata,
		}
		if numberValue.Valid {
			artifact.NumberValue = &numberValue.Float64
		}
		if identityKey.Valid {
			artifact.IdentityKey = &identityKey.String
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
		} else {
			return nil, 0, "", util.NewInvalidInputError("Unsupported artifact filter type %q", filterContext.Type)
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
		} else {
			return nil, 0, "", util.NewInvalidInputError("Unsupported artifact filter type %q", filterContext.Type)
		}
	}
	sizeSQL, sizeArgs, err := opts.AddFilterToSelect(countBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the totalSize of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		return errorF(err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact list transaction: %v", rbErr)
		}
	}

	rows, err := tx.Query(rowsSQL, rowsArgs...)
	if err != nil {
		rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		rollback()
		return errorF(err)
	}
	artifacts, err := s.scanRows(rows)
	if err != nil {
		rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSQL, sizeArgs...)
	if err != nil {
		rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		rollback()
		return errorF(err)
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		return errorF(err)
	}

	if len(artifacts) <= opts.PageSize {
		return artifacts, totalSize, "", nil
	}

	npt, err := opts.NextPageToken(artifacts[opts.PageSize])
	if err != nil {
		return errorF(err)
	}
	return artifacts[:opts.PageSize], totalSize, npt, nil
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
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact: %v", err.Error())
	}
	if len(artifacts) > 1 {
		return nil, util.NewInternalServerError(errors.New("multiple artifacts found"), "Failed to get artifact %s: multiple rows returned", id)
	}
	if len(artifacts) == 0 {
		return nil, util.NewResourceNotFoundError("artifact", fmt.Sprint(id))
	}

	return artifacts[0], nil
}

// GetArtifactsByURI returns artifacts matching Namespace and URI exactly.
// Unlike ListArtifacts, this path issues a single equality query and does not
// run a COUNT(*) or pagination loop. Lookups use the indexed (Namespace,
// URIHash) columns. Exact URI equality is re-checked in Go to protect against
// hash collisions. URIHash is populated on every write; migration backfill
// covers any pre-existing rows, so empty-hash fallbacks are intentionally
// omitted.
//
// An empty namespace is valid and required in single-user mode, where
// ReplaceNamespace clears namespaces before persistence. Multi-user callers
// must still supply a non-empty namespace at the API authorization layer.
func (s *ArtifactStore) GetArtifactsByURI(namespace, uri string) ([]*model.Artifact, error) {
	if uri == "" {
		return nil, util.NewInvalidInputError("uri is required for GetArtifactsByURI")
	}

	uriHash := artifactURIHash(uri)
	sql, args, err := sq.
		Select(artifactColumns...).
		From(artifactTableName).
		Where(sq.Eq{
			"Namespace": namespace,
			"URIHash":   uriHash,
		}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifacts by URI: %v", err.Error())
	}

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifacts by URI: %v", err.Error())
	}
	defer rows.Close()

	artifacts, err := s.scanRows(rows)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to scan artifacts by URI: %v", err.Error())
	}

	// Protect against rare SHA-256 collisions by requiring exact URI equality.
	matched := make([]*model.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.URI != nil && *artifact.URI == uri {
			matched = append(matched, artifact)
		}
	}
	return matched, nil
}
