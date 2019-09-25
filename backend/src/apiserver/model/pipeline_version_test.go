package model

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/stretchr/testify/assert"
)

// Test model name usage in sorting clause
func TestAddSortingToSelect(t *testing.T) {
	listable := &PipelineVersion{
		UUID:           "version_id_1",
		CreatedAtInSec: 1,
		Name:           "version_name_1",
		Parameters:     "",
		PipelineId:     "pipeline_id_1",
		Status:         PipelineVersionReady,
		CodeSourceUrl:  "",
	}
	protoFilter := &api.Filter{}
	listableOptions, err := list.NewOptions(listable, 10, "name", protoFilter)
	assert.Nil(t, err)
	sqlBuilder := sq.Select("*").From("pipeline_versions")
	sql, _, err := listableOptions.AddSortingToSelect(sqlBuilder).ToSql()
	assert.Nil(t, err)

	assert.Contains(t, sql, "pipeline_versions.Name") // sorting field
	assert.Contains(t, sql, "pipeline_versions.UUID") // primary key field
}
