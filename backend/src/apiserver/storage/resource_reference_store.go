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

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/json"
)

var resourceReferenceColumns = []string{
	"ResourceUUID", "ResourceType", "ReferenceUUID",
	"ReferenceName", "ReferenceType", "Relationship", "Payload",
}

type ResourceReferenceStoreInterface interface {
	// Retrieve the resource reference for a given resource id, type and a reference type.
	GetResourceReference(resourceId string, resourceType model.ResourceType,
		referenceType model.ResourceType) (*model.ResourceReference, error)
}

type ResourceReferenceStore struct {
	db            *DB
	pipelineStore PipelineStoreInterface
	dialect       dialect.DBDialect
}

// Create a resource reference.
// This is always in company with creating a parent resource so a transaction is needed as input.
func (s *ResourceReferenceStore) CreateResourceReferences(tx *sql.Tx, refs []*model.ResourceReference) error {
	if len(refs) > 0 {
		q := s.dialect.QuoteIdentifier
		qb := s.dialect.QueryBuilder()
		resourceRefSQLBuilder := qb.
			Insert(q("resource_references")).
			Columns(
				q("ResourceUUID"), q("ResourceType"), q("ReferenceUUID"),
				q("ReferenceName"), q("ReferenceType"), q("Relationship"),
				q("Payload"),
			)
		for _, ref := range refs {
			if !s.checkReferenceExist(tx, ref.ReferenceUUID, ref.ReferenceType) {
				return util.NewResourceNotFoundError(string(ref.ReferenceType), ref.ReferenceUUID)
			}
			payload, err := json.Marshal(ref)
			if err != nil {
				return util.NewInternalServerError(err, "Failed to stream resource reference model to a json payload")
			}
			resourceRefSQLBuilder = resourceRefSQLBuilder.Values(
				ref.ResourceUUID, ref.ResourceType, ref.ReferenceUUID, ref.ReferenceName, ref.ReferenceType, ref.Relationship, string(payload))
		}
		refSQL, refArgs, err := resourceRefSQLBuilder.ToSql()
		if err != nil {
			return util.NewInternalServerError(err, "Failed to create query to store resource references")
		}
		_, err = tx.Exec(refSQL, refArgs...)
		if err != nil {
			return util.NewInternalServerError(err, "Failed to store resource references")
		}
	}
	return nil
}

func (s *ResourceReferenceStore) checkReferenceExist(tx *sql.Tx, referenceId string, referenceType model.ResourceType) bool {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	var selectBuilder sq.SelectBuilder
	switch referenceType {
	case model.JobResourceType:
		selectBuilder = qb.Select("1").From(q("jobs")).Where(sq.Eq{q("UUID"): referenceId})
	case model.ExperimentResourceType:
		selectBuilder = qb.Select("1").From(q("experiments")).Where(sq.Eq{q("UUID"): referenceId})
	case model.PipelineVersionResourceType:
		if s.pipelineStore != nil {
			pv, _ := s.pipelineStore.GetPipelineVersion(referenceId)

			return pv != nil
		}
		selectBuilder = qb.Select("1").From(q("pipeline_versions")).Where(sq.Eq{q("UUID"): referenceId})
	case model.PipelineResourceType:
		if s.pipelineStore != nil {
			p, _ := s.pipelineStore.GetPipeline(referenceId)

			return p != nil
		}
		selectBuilder = qb.Select("1").From(q("pipelines")).Where(sq.Eq{q("UUID"): referenceId})
	case model.NamespaceResourceType:
		// This function is called to check the data validity when the data are transformed according to the DB schema.
		// Since there is not a separate table to store the namespace data, thus always returning true.
		return true
	default:
		return false
	}
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return false
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return false
	}
	defer rows.Close()
	return rows.Next()
}

// Delete all resource references for a specific resource.
// This is always in company with creating a parent resource so a transaction is needed as input.
func (s *ResourceReferenceStore) DeleteResourceReferences(tx *sql.Tx, id string, resourceType model.ResourceType) error {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	refSQL, refArgs, err := qb.
		Delete(q("resource_references")).
		Where(sq.Or{
			sq.Eq{q("ResourceUUID"): id, q("ResourceType"): resourceType},
			sq.Eq{q("ReferenceUUID"): id, q("ReferenceType"): resourceType},
		}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete resource references for %s %s due to SQL syntax error", resourceType, id)
	}
	_, err = tx.Exec(refSQL, refArgs...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete resource references for %s %s due to SQL execution error", resourceType, id)
	}
	return nil
}

func (s *ResourceReferenceStore) GetResourceReference(resourceId string, resourceType model.ResourceType,
	referenceType model.ResourceType,
) (*model.ResourceReference, error) {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	quotedCols := make([]string, len(resourceReferenceColumns))
	for i, c := range resourceReferenceColumns {
		quotedCols[i] = q(c)
	}
	sql, args, err := qb.Select(quotedCols...).
		From(q("resource_references")).
		Where(sq.Eq{
			q("ResourceUUID"):  resourceId,
			q("ResourceType"):  resourceType,
			q("ReferenceType"): referenceType,
		}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to create query to get resource reference. "+
				"Resource ID: %s. Resource Type: %s. Reference Type: %s", resourceId, resourceType, referenceType)
	}
	row, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to get resource reference. "+
				"Resource ID: %s. Resource Type: %s. Reference Type: %s", resourceId, resourceType, referenceType)
	}
	defer row.Close()
	reference, err := s.scanRows(row)
	if err != nil || len(reference) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get resource reference: %v", err.Error())
	}
	if len(reference) == 0 {
		return nil, util.NewResourcesNotFoundError(
			"Resource ID: %s. Resource Type: %s. Reference Type: %s", resourceId, resourceType, referenceType)
	}
	return &reference[0], nil
}

func (s *ResourceReferenceStore) scanRows(r *sql.Rows) ([]model.ResourceReference, error) {
	var references []model.ResourceReference
	for r.Next() {
		var resourceUUID, resourceType, referenceUUID, referenceName, referenceType, relationship, payload string
		err := r.Scan(
			&resourceUUID, &resourceType, &referenceUUID, &referenceName, &referenceType, &relationship, &payload)
		if err != nil {
			return nil, err
		}
		references = append(references, model.ResourceReference{
			ResourceUUID:  resourceUUID,
			ResourceType:  model.ResourceType(resourceType),
			ReferenceUUID: referenceUUID,
			ReferenceName: referenceName,
			ReferenceType: model.ResourceType(referenceType),
			Relationship:  model.Relationship(relationship),
			Payload:       model.LargeText(payload),
		})
	}
	return references, nil
}

func NewResourceReferenceStore(db *DB, pipelineStore PipelineStoreInterface, d dialect.DBDialect) *ResourceReferenceStore {
	// If pipelineStore is specified and it is not nil, it will be used instead of the DB.
	// This will make pipelines and pipeline versions to get stored in K8s instead of Database.
	return &ResourceReferenceStore{db: db, pipelineStore: pipelineStore, dialect: d}
}
