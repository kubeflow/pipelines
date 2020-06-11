package model

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/stretchr/testify/assert"
)

// Test model name usage in sorting clause
func TestAddStatusFilterToSelect(t *testing.T) {
	listable := &Run{
		UUID:           "run_id_1",
		CreatedAtInSec: 1,
		Name:           "run_name_1",
		Conditions:     "Succeeded",
	}
	protoFilter := &api.Filter{}
	protoFilter.Predicates = []*api.Predicate{
		{
			Key:   "status",
			Op:    api.Predicate_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "Succeeded"},
		},
	}
	listableOptions, err := list.NewOptions(listable, 10, "name", protoFilter)
	assert.Nil(t, err)
	sqlBuilder := sq.Select("*").From("run_details")
	sql, args, err := listableOptions.AddFilterToSelect(sqlBuilder).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE Conditions = ?") // filtering on status, aka Conditions in db
	assert.Contains(t, args, "Succeeded")

	notEqualProtoFilter := &api.Filter{}
	notEqualProtoFilter.Predicates = []*api.Predicate{
		{
			Key:   "status",
			Op:    api.Predicate_NOT_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "somevalue"},
		},
	}
	listableOptions, err = list.NewOptions(listable, 10, "name", notEqualProtoFilter)
	assert.Nil(t, err)
	sqlBuilder = sq.Select("*").From("run_details")
	sql, args, err = listableOptions.AddFilterToSelect(sqlBuilder).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE Conditions <> ?") // filtering on status, aka Conditions in db
	assert.Contains(t, args, "somevalue")
}
