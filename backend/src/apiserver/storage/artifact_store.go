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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
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

func computeArtifactWriteIdentity(
	namespace, runUUID, taskID, operationID string,
) (string, error) {
	var identity bytes.Buffer

	for _, value := range []string{
		namespace,
		runUUID,
		taskID,
		operationID,
	} {
		if err := writeLengthPrefixedString(&identity, value); err != nil {
			return "", err
		}
	}

	digest := sha256.Sum256(identity.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

func computeArtifactWritePayloadHash(artifact *model.Artifact) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("artifact is nil")
	}

	var payload bytes.Buffer

	if err := binary.Write(&payload, binary.BigEndian, int32(artifact.Type)); err != nil {
		return "", err
	}

	if artifact.URI == nil {
		if err := binary.Write(&payload, binary.BigEndian, uint8(0)); err != nil {
			return "", err
		}
	} else {
		if err := binary.Write(&payload, binary.BigEndian, uint8(1)); err != nil {
			return "", err
		}
		if err := writeLengthPrefixedString(&payload, *artifact.URI); err != nil {
			return "", err
		}
	}

	if err := writeLengthPrefixedString(&payload, artifact.Name); err != nil {
		return "", err
	}

	if err := writeLengthPrefixedString(&payload, artifact.Description); err != nil {
		return "", err
	}

	metadataBytes, err := canonicalArtifactMetadataBytes(artifact.Metadata)
	if err != nil {
		return "", err
	}
	if err := writeLengthPrefixedString(&payload, string(metadataBytes)); err != nil {
		return "", err
	}

	if artifact.NumberValue == nil {
		if err := binary.Write(&payload, binary.BigEndian, uint8(0)); err != nil {
			return "", err
		}
	} else {
		if err := binary.Write(&payload, binary.BigEndian, uint8(1)); err != nil {
			return "", err
		}
		if err := binary.Write(&payload, binary.BigEndian, *artifact.NumberValue); err != nil {
			return "", err
		}
	}

	digest := sha256.Sum256(payload.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

// Ensure that ClientManager implements the resource.ClientManagerInterface interface.
var _ ArtifactStoreInterface = &ArtifactStore{}

type ArtifactStoreInterface interface {
	// CreateArtifact creates an artifact entry in the database.
	CreateArtifact(*model.Artifact) (*model.Artifact, error)

	// CreateArtifactWithTask atomically creates an artifact row and its output link.
	CreateArtifactWithTask(
		artifact *model.Artifact,
		artifactTask *model.ArtifactTask,
		operationID string,
	) (*model.Artifact, *model.ArtifactTask, error)

	// CreateArtifactsWithTasks atomically creates a batch of artifacts and output links.
	CreateArtifactsWithTasks(
		artifacts []*model.Artifact,
		artifactTasks []*model.ArtifactTask,
		operationIDs []string,
	) ([]*model.Artifact, []*model.ArtifactTask, error)

	// FindOrCreateArtifactWithTask atomically reuses an existing artifact that matches the
	// stable reuse identity, or creates one when no match exists. Concurrent callers that
	// race on the same identity share one artifact row and each still get their own link.
	FindOrCreateArtifactWithTask(*model.Artifact, *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error)

	// GetArtifact fetches an artifact with a given id.
	GetArtifact(string) (*model.Artifact, error)

	// GetArtifactsByURI fetches artifacts with exact Namespace + URI equality.
	GetArtifactsByURI(string, string) ([]*model.Artifact, error)

	// ListArtifacts fetches artifacts for given filtering and listing options.
	// It returns the current page of artifacts, the total count across all pages,
	// the next page token, and an error.
	ListArtifacts(*model.FilterContext, *list.Options) ([]*model.Artifact, int, string, error)
}

type ArtifactStore struct {
	dbDialect                       dialect.DBDialect
	db                              *sql.DB
	time                            util.TimeInterface
	uuid                            util.UUIDGeneratorInterface
	createArtifactTaskInTransaction func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error)
	createArtifactWriteIdentity     func(tx *sql.Tx, identity *model.ArtifactWriteIdentity) error
}

// NewArtifactStore creates a new ArtifactStore.
func NewArtifactStore(db *sql.DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface, d dialect.DBDialect) *ArtifactStore {
	store := &ArtifactStore{
		dbDialect: d,
		db:        db,
		time:      time,
		uuid:      uuid,
	}
	store.createArtifactTaskInTransaction = func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
		return createArtifactTaskWithExecutor(tx.Exec, store.uuid, artifactTask, store.dbDialect)
	}
	store.createArtifactWriteIdentity = func(tx *sql.Tx, identity *model.ArtifactWriteIdentity) error {
		return store.createArtifactWriteIdentityInTransaction(tx, identity)
	}
	return store
}

func (s *ArtifactStore) CreateArtifact(artifact *model.Artifact) (*model.Artifact, error) {
	return s.createArtifactWithExecutor(s.db.Exec, artifact)
}

func (s *ArtifactStore) createArtifactWithExecutor(exec func(string, ...any) (sql.Result, error), artifact *model.Artifact) (*model.Artifact, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
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

	sql, args, err := qb.
		Insert(q(artifactTableName)).
		SetMap(
			sq.Eq{
				q("UUID"):            newArtifact.UUID,
				q("Namespace"):       newArtifact.Namespace,
				q("Type"):            newArtifact.Type,
				q("URI"):             newArtifact.URI,
				q("URIHash"):         newArtifact.URIHash,
				q("Name"):            newArtifact.Name,
				q("Description"):     newArtifact.Description,
				q("CreatedAtInSec"):  newArtifact.CreatedAtInSec,
				q("LastUpdateInSec"): newArtifact.LastUpdateInSec,
				q("Metadata"):        metadataJSON,
				q("NumberValue"):     newArtifact.NumberValue,
				q("IdentityKey"):     newArtifact.IdentityKey,
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

func (s *ArtifactStore) createArtifactWithTaskInTransaction(
	tx *sql.Tx,
	artifact *model.Artifact,
	artifactTask *model.ArtifactTask,
) (*model.Artifact, *model.ArtifactTask, error) {
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

	return newArtifact, newArtifactTask, nil
}

// CreateArtifactWithTask atomically creates an artifact row and its output link.
// Keeping this transaction inside storage preserves the store-first boundary and
// prevents callers from reaching into the raw DB just to keep `artifacts` and
// `artifact_tasks` in sync for one logical API operation.
func (s *ArtifactStore) CreateArtifactWithTask(
	artifact *model.Artifact,
	artifactTask *model.ArtifactTask,
	operationID string,
) (*model.Artifact, *model.ArtifactTask, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to start transaction for creating artifact and artifact-task",
		)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact create transaction: %v", rbErr)
		}
	}()

	artifactTaskCopy := *artifactTask
	if err := artifactTaskCopy.SyncIterationFromProducer(); err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to derive artifact-task iteration: %v",
			err.Error(),
		)
	}
	iterationIndex := artifactTaskCopy.Iteration

	if operationID == "" {
		newArtifact, newArtifactTask, err :=
			s.createArtifactWithTaskInTransaction(
				tx,
				artifact,
				&artifactTaskCopy,
			)
		if err != nil {
			return nil, nil, util.Wrap(err, "Failed to create artifact and artifact-task relationship")
		}
		if err := tx.Commit(); err != nil {
			return nil, nil, util.NewInternalServerError(
				err,
				"Failed to commit transaction for creating artifact and artifact-task",
			)
		}
		return newArtifact, newArtifactTask, nil
	}

	identity, err := computeArtifactWriteIdentity(
		artifact.Namespace,
		artifactTaskCopy.RunUUID,
		artifactTaskCopy.TaskID,
		operationID,
	)
	if err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to compute artifact write identity",
		)
	}

	payloadHash, err := computeArtifactWritePayloadHash(artifact)
	if err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to compute artifact write payload hash",
		)
	}

	existingIdentity, err := s.getArtifactWriteIdentity(tx, identity)
	if err != nil {
		return nil, nil, err
	}

	if existingIdentity != nil {
		if existingIdentity.PayloadHash != payloadHash {
			return nil, nil, util.NewInvalidInputError(
				"Artifact write identity %q already exists with a different payload",
				identity,
			)
		}

		existingArtifact, err := s.getArtifactWithExecutor(
			tx,
			existingIdentity.ArtifactID,
		)
		if err != nil {
			return nil, nil, err
		}
		if existingArtifact == nil {
			return nil, nil, util.NewInternalServerError(
				fmt.Errorf("artifact %q not found", existingIdentity.ArtifactID),
				"Artifact write identity points to missing artifact",
			)
		}

		artifactTaskCopy := *artifactTask
		artifactTaskCopy.ArtifactID = existingArtifact.UUID
		artifactTaskCopy.Iteration = iterationIndex

		existingArtifactTask, err :=
			s.getArtifactTaskByUniqueLinkWithExecutor(tx, &artifactTaskCopy)
		if err != nil {
			return nil, nil, err
		}
		if existingArtifactTask == nil {
			return nil, nil, util.NewInternalServerError(
				fmt.Errorf(
					"artifact-task link for artifact %q not found",
					existingArtifact.UUID,
				),
				"Artifact write identity points to missing artifact-task link",
			)
		}

		return existingArtifact, existingArtifactTask, nil
	}

	newArtifact, newArtifactTask, err :=
		s.createArtifactWithTaskInTransaction(tx, artifact, &artifactTaskCopy)
	if err != nil {
		return nil, nil, err
	}

	writeIdentity := &model.ArtifactWriteIdentity{
		Identity:    identity,
		Namespace:   artifact.Namespace,
		RunUUID:     artifactTaskCopy.RunUUID,
		TaskID:      artifactTaskCopy.TaskID,
		OperationID: operationID,
		PayloadHash: payloadHash,
		ArtifactID:  newArtifact.UUID,
	}

	if err := s.createArtifactWriteIdentity(tx, writeIdentity); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to commit transaction for creating artifact and artifact-task",
		)
	}

	return newArtifact, newArtifactTask, nil
}

// CreateArtifactsWithTasks atomically creates a batch of artifacts and output links.
// This method is intentionally all-or-nothing so a later artifact-task failure
// cannot leave earlier artifacts committed without their matching link rows.
func (s *ArtifactStore) CreateArtifactsWithTasks(
	artifacts []*model.Artifact,
	artifactTasks []*model.ArtifactTask,
	operationIDs []string,
) ([]*model.Artifact, []*model.ArtifactTask, error) {
	if len(artifacts) != len(artifactTasks) ||
		len(artifacts) != len(operationIDs) {
		return nil, nil, util.NewInvalidInputError(
			"artifacts, artifact tasks, and operation IDs must have the same length",
		)
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
		artifactTaskCopy := *artifactTasks[index]

		if err := artifactTaskCopy.SyncIterationFromProducer(); err != nil {
			return nil, nil, util.Wrap(err, fmt.Sprintf("Failed to sync artifact-task iteration %d", index))
		}

		iterationIndex := artifactTaskCopy.Iteration
		operationID := operationIDs[index]

		if operationID == "" {
			newArtifact, newArtifactTask, err :=
				s.createArtifactWithTaskInTransaction(
					tx,
					artifact,
					&artifactTaskCopy,
				)
			if err != nil {
				return nil, nil, util.Wrap(
					err,
					fmt.Sprintf(
						"Failed to create artifact and artifact-task relationship %d",
						index,
					),
				)
			}

			createdArtifacts = append(createdArtifacts, newArtifact)
			createdArtifactTasks = append(createdArtifactTasks, newArtifactTask)
			continue
		}

		identity, err := computeArtifactWriteIdentity(
			artifact.Namespace,
			artifactTaskCopy.RunUUID,
			artifactTaskCopy.TaskID,
			operationID,
		)
		if err != nil {
			return nil, nil, util.Wrap(
				err,
				fmt.Sprintf("Failed to compute artifact write identity %d", index),
			)
		}

		payloadHash, err := computeArtifactWritePayloadHash(artifact)
		if err != nil {
			return nil, nil, util.Wrap(
				err,
				fmt.Sprintf("Failed to compute artifact payload hash %d", index),
			)
		}

		existingIdentity, err := s.getArtifactWriteIdentity(tx, identity)
		if err != nil {
			return nil, nil, util.Wrap(
				err,
				fmt.Sprintf("Failed to look up artifact write identity %d", index),
			)
		}

		if existingIdentity != nil {
			if existingIdentity.PayloadHash != payloadHash {
				return nil, nil, util.NewInvalidInputError(
					"Artifact write identity %q already exists with a different payload",
					identity,
				)
			}

			existingArtifact, err := s.getArtifactWithExecutor(tx, existingIdentity.ArtifactID)
			if err != nil {
				return nil, nil, util.Wrap(
					err,
					fmt.Sprintf("Failed to retrieve replayed artifact %d", index),
				)
			}

			if existingArtifact == nil {
				return nil, nil, util.NewInternalServerError(
					fmt.Errorf("Artifact write identity %q references a missing artifact", identity),
					"Artifact write identity references a missing artifact",
				)
			}

			artifactTaskCopy.ArtifactID = existingArtifact.UUID
			artifactTaskCopy.Iteration = iterationIndex
			existingArtifactTask, err :=
				s.getArtifactTaskByUniqueLinkWithExecutor(tx, &artifactTaskCopy)
			if err != nil {
				return nil, nil, util.Wrap(
					err,
					fmt.Sprintf("Failed to retrieve replayed artifact-task relationship %d", index),
				)
			}

			if existingArtifactTask == nil {
				return nil, nil, util.NewInternalServerError(
					fmt.Errorf(
						"Artifact write identity %q references a missing artifact-task relationship",
						identity,
					),
					"Artifact write identity references a missing artifact-task relationship",
				)
			}

			createdArtifacts = append(createdArtifacts, existingArtifact)
			createdArtifactTasks = append(createdArtifactTasks, existingArtifactTask)
			continue
		}

		newArtifact, newArtifactTask, err :=
			s.createArtifactWithTaskInTransaction(tx, artifact, &artifactTaskCopy)
		if err != nil {
			return nil, nil, util.Wrap(
				err,
				fmt.Sprintf("Failed to create artifact and artifact-task relationship %d", index),
			)
		}

		writeIdentity := &model.ArtifactWriteIdentity{
			Identity:    identity,
			Namespace:   artifact.Namespace,
			RunUUID:     artifactTaskCopy.RunUUID,
			TaskID:      artifactTaskCopy.TaskID,
			OperationID: operationID,
			PayloadHash: payloadHash,
			ArtifactID:  newArtifact.UUID,
		}

		if err := s.createArtifactWriteIdentityInTransaction(tx, writeIdentity); err != nil {
			return nil, nil, util.Wrap(
				err,
				fmt.Sprintf("Failed to persist artifact write identity %d", index),
			)
		}

		createdArtifacts = append(createdArtifacts, newArtifact)
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

	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, util.NewInternalServerError(
			err,
			"Failed to start transaction for finding or creating artifact",
		)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact find-or-create transaction: %v", rbErr)
		}
	}()

	createdArtifact, createdArtifactTask, err :=
		s.createArtifactWithTaskInTransaction(tx, &artifactToCreate, artifactTask)

	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, nil, util.NewInternalServerError(
				err,
				"Failed to commit transaction for finding or creating artifact",
			)
		}
		return createdArtifact, createdArtifactTask, nil
	}

	// Another concurrent writer may have inserted the same identity key first.
	_ = tx.Rollback()

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

	createdLink, err := createArtifactTaskWithExecutor(s.db.Exec, s.uuid, &artifactTaskCopy, s.dbDialect)
	if err == nil {
		return existingArtifact, createdLink, nil
	}

	existingLink, findErr := s.getArtifactTaskByUniqueLink(&artifactTaskCopy)
	if findErr == nil && existingLink != nil {
		return existingArtifact, existingLink, nil
	}
	return nil, nil, err
}

func (s *ArtifactStore) getArtifactWriteIdentity(
	tx *sql.Tx,
	identity string,
) (*model.ArtifactWriteIdentity, error) {
	q := s.dbDialect.QuoteIdentifier

	query := fmt.Sprintf(
		"SELECT %s, %s, %s, %s, %s, %s, %s FROM %s WHERE %s = ?",
		q("Identity"),
		q("Namespace"),
		q("RunUUID"),
		q("TaskID"),
		q("OperationID"),
		q("PayloadHash"),
		q("ArtifactID"),
		q("artifact_write_identities"),
		q("Identity"),
	)

	var result model.ArtifactWriteIdentity
	err := tx.QueryRow(query, identity).Scan(
		&result.Identity,
		&result.Namespace,
		&result.RunUUID,
		&result.TaskID,
		&result.OperationID,
		&result.PayloadHash,
		&result.ArtifactID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to get artifact write identity",
		)
	}

	return &result, nil
}

func (s *ArtifactStore) createArtifactWriteIdentityInTransaction(
	tx *sql.Tx,
	identity *model.ArtifactWriteIdentity,
) error {
	q := s.dbDialect.QuoteIdentifier

	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s, %s, %s, %s) VALUES (?, ?, ?, ?, ?, ?, ?)",
		q("artifact_write_identities"),
		q("Identity"),
		q("Namespace"),
		q("RunUUID"),
		q("TaskID"),
		q("OperationID"),
		q("PayloadHash"),
		q("ArtifactID"),
	)

	_, err := tx.Exec(
		query,
		identity.Identity,
		identity.Namespace,
		identity.RunUUID,
		identity.TaskID,
		identity.OperationID,
		identity.PayloadHash,
		identity.ArtifactID,
	)
	if err != nil {
		return util.NewInternalServerError(
			err,
			"Failed to create artifact write identity",
		)
	}

	return nil
}

func (s *ArtifactStore) getArtifactTaskByUniqueLinkWithExecutor(
	exec interface {
		Query(string, ...any) (*sql.Rows, error)
	},
	artifactTask *model.ArtifactTask,
) (*model.ArtifactTask, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()

	if artifactTask == nil {
		return nil, nil
	}

	sqlQuery, args, err := qb.
		Select(
			q("UUID"),
			q("ArtifactID"),
			q("TaskID"),
			q("Type"),
			q("Iteration"),
			q("RunUUID"),
			q("Producer"),
			q("ArtifactKey"),
		).
		From(q(artifactTaskTableName)).
		Where(sq.Eq{
			q("ArtifactID"):  artifactTask.ArtifactID,
			q("TaskID"):      artifactTask.TaskID,
			q("Type"):        artifactTask.Type,
			q("Iteration"):   artifactTask.Iteration,
			q("ArtifactKey"): artifactTask.ArtifactKey,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to create query to get artifact-task by unique link: %v",
			err.Error(),
		)
	}

	rows, err := exec.Query(sqlQuery, args...)
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to get artifact-task by unique link: %v",
			err.Error(),
		)
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

	if err := rows.Scan(
		&uuid,
		&artifactID,
		&taskID,
		&ioType,
		&iteration,
		&runUUID,
		&producerBytes,
		&artifactKey,
	); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to scan artifact-task by unique link: %v",
			err.Error(),
		)
	}

	var producer model.JSONData
	if producerBytes != nil {
		if err := producer.Scan(producerBytes); err != nil {
			return nil, util.NewInternalServerError(
				err,
				"Failed to parse artifact-task producer: %v",
				err.Error(),
			)
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

func (s *ArtifactStore) getArtifactTaskByUniqueLink(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	if artifactTask == nil {
		return nil, nil
	}
	sql, args, err := qb.
		Select(
			q("UUID"),
			q("ArtifactID"),
			q("TaskID"),
			q("Type"),
			q("Iteration"),
			q("RunUUID"),
			q("Producer"),
			q("ArtifactKey"),
		).
		From(q(artifactTaskTableName)).
		Where(sq.Eq{
			q("ArtifactID"):  artifactTask.ArtifactID,
			q("TaskID"):      artifactTask.TaskID,
			q("Type"):        artifactTask.Type,
			q("Iteration"):   artifactTask.Iteration,
			q("ArtifactKey"): artifactTask.ArtifactKey,
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

func (s *ArtifactStore) getArtifactWithExecutor(
	exec interface {
		Query(string, ...any) (*sql.Rows, error)
	},
	artifactID string,
) (*model.Artifact, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()

	query, args, err := qb.
		Select(dialect.QuoteAll(q, artifactColumns)...).
		From(q(artifactTableName)).
		Where(sq.Eq{
			q("UUID"): artifactID,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to create query to get artifact: %v",
			err.Error(),
		)
	}

	rows, err := exec.Query(query, args...)
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to get artifact: %v",
			err.Error(),
		)
	}
	defer rows.Close()

	artifacts, err := s.scanRows(rows)
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to scan artifact: %v",
			err.Error(),
		)
	}
	if len(artifacts) == 0 {
		return nil, nil
	}

	return artifacts[0], nil
}

func (s *ArtifactStore) getArtifactByIdentityKey(namespace, identityKey string) (*model.Artifact, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	if identityKey == "" {
		return nil, nil
	}
	sql, args, err := qb.
		Select(dialect.QuoteAll(q, artifactColumns)...).
		From(q(artifactTableName)).
		Where(sq.Eq{
			q("Namespace"):   namespace,
			q("IdentityKey"): identityKey,
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
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	errorF := func(err error) ([]*model.Artifact, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list artifacts: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := qb.Select(dialect.QuoteAll(q, artifactColumns)...).From(q(artifactTableName))

	// Apply namespace filtering if provided
	if filterContext != nil && filterContext.ReferenceKey != nil {
		if filterContext.Type == model.NamespaceResourceType {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("Namespace"): filterContext.ID})
		} else {
			return nil, 0, "", util.NewInvalidInputError("Unsupported artifact filter type %q", filterContext.Type)
		}
	}

	sqlBuilder = opts.AddFilterToSelect(sqlBuilder, q)

	rowsSQL, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder, q, s.dbDialect.StringCollation()).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := qb.Select("count(*)").From(q(artifactTableName))
	if filterContext != nil && filterContext.ReferenceKey != nil {
		if filterContext.Type == model.NamespaceResourceType {
			countBuilder = countBuilder.Where(sq.Eq{q("Namespace"): filterContext.ID})
		} else {
			return nil, 0, "", util.NewInvalidInputError("Unsupported artifact filter type %q", filterContext.Type)
		}
	}
	sizeSQL, sizeArgs, err := opts.AddFilterToSelect(countBuilder, q).ToSql()
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
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	sql, args, err := qb.
		Select(dialect.QuoteAll(q, artifactColumns)...).
		From(q(artifactTableName)).
		Where(sq.Eq{q("UUID"): id}).
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
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	if uri == "" {
		return nil, util.NewInvalidInputError("uri is required for GetArtifactsByURI")
	}

	uriHash := artifactURIHash(uri)
	sql, args, err := qb.
		Select(dialect.QuoteAll(q, artifactColumns)...).
		From(q(artifactTableName)).
		Where(sq.Eq{
			q("Namespace"): namespace,
			q("URIHash"):   uriHash,
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
