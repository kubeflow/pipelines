package storage

import (
	"database/sql"

	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/json"
)

var resourceReferenceColumns = []string{"ResourceUUID", "ResourceType", "ReferenceUUID",
	"ReferenceName", "ReferenceType", "Relationship", "Payload"}

type ResourceReferenceStoreInterface interface {
	// Retrieve the resource reference for a given resource id, type and a reference type.
	GetResourceReference(resourceId string, resourceType common.ResourceType,
		referenceType common.ResourceType) (*model.ResourceReference, error)
}

type ResourceReferenceStore struct {
	db *DB
}

// Create a resource reference.
// This is always in company with creating a parent resource so a transaction is needed as input.
func (s *ResourceReferenceStore) CreateResourceReferences(tx *sql.Tx, refs []*model.ResourceReference) error {
	if len(refs) > 0 {
		resourceRefSqlBuilder := sq.
			Insert("resource_references").
			Columns("ResourceUUID", "ResourceType", "ReferenceUUID", "ReferenceName", "ReferenceType", "Relationship", "Payload")
		for _, ref := range refs {
			if !s.checkReferenceExist(tx, ref.ReferenceUUID, ref.ReferenceType) {
				return util.NewResourceNotFoundError(string(ref.ReferenceType), ref.ReferenceUUID)
			}
			payload, err := json.Marshal(ref)
			if err != nil {
				return util.NewInternalServerError(err, "Failed to stream resource reference model to a json payload")
			}
			resourceRefSqlBuilder = resourceRefSqlBuilder.Values(
				ref.ResourceUUID, ref.ResourceType, ref.ReferenceUUID, ref.ReferenceName, ref.ReferenceType, ref.Relationship, string(payload))
		}
		refSql, refArgs, err := resourceRefSqlBuilder.ToSql()
		if err != nil {
			return util.NewInternalServerError(err, "Failed to create query to store resource references.")
		}
		_, err = tx.Exec(refSql, refArgs...)
		if err != nil {
			return util.NewInternalServerError(err, "Failed to store resource references.")
		}
	}
	return nil
}

func (s *ResourceReferenceStore) checkReferenceExist(tx *sql.Tx, referenceId string, referenceType common.ResourceType) bool {
	var selectBuilder sq.SelectBuilder
	switch referenceType {
	case common.Job:
		selectBuilder = sq.Select("1").From("jobs").Where(sq.Eq{"uuid": referenceId})
	case common.Experiment:
		selectBuilder = sq.Select("1").From("experiments").Where(sq.Eq{"uuid": referenceId})
	case common.PipelineVersion:
		selectBuilder = sq.Select("1").From("pipeline_versions").Where(sq.Eq{"uuid": referenceId})
	case common.Namespace:
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
	var exists bool
	err = tx.QueryRow(fmt.Sprintf("SELECT exists (%s)", query), args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false
	}
	return exists
}

// Delete all resource references for a specific resource.
// This is always in company with creating a parent resource so a transaction is needed as input.
func (s *ResourceReferenceStore) DeleteResourceReferences(tx *sql.Tx, id string, resourceType common.ResourceType) error {
	refSql, refArgs, err := sq.
		Delete("resource_references").
		Where(sq.Or{
			sq.Eq{"ResourceUUID": id, "ResourceType": resourceType},
			sq.Eq{"ReferenceUUID": id, "ReferenceType": resourceType}}).
		ToSql()
	_, err = tx.Exec(refSql, refArgs...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete resource references for %s %s", resourceType, id)
	}
	return nil
}

func (s *ResourceReferenceStore) GetResourceReference(resourceId string, resourceType common.ResourceType,
	referenceType common.ResourceType) (*model.ResourceReference, error) {
	sql, args, err := sq.Select(resourceReferenceColumns...).
		From("resource_references").
		Where(sq.Eq{
			"ResourceUUID":  resourceId,
			"ResourceType":  resourceType,
			"ReferenceType": referenceType}).
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
			ResourceType:  common.ResourceType(resourceType),
			ReferenceUUID: referenceUUID,
			ReferenceName: referenceName,
			ReferenceType: common.ResourceType(referenceType),
			Relationship:  common.Relationship(relationship),
			Payload:       payload,
		})
	}
	return references, nil
}

func NewResourceReferenceStore(db *DB) *ResourceReferenceStore {
	return &ResourceReferenceStore{db: db}
}
