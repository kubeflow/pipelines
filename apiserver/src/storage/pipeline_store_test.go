package storage

import (
	"ml/apiserver/src/message/pipelinemanager"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func initializePipelineDB() (PipelineStoreInterface, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("mysql", db)
	return &PipelineStore{db: gormDB}, mock
}

func TestListPipelines(t *testing.T) {
	expectedPipelines := []pipelinemanager.Pipeline{
		{Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []pipelinemanager.Parameter{}},
		{Metadata: &pipelinemanager.Metadata{ID: 2}, Name: "Pipeline456", PackageId: 2, Parameters: []pipelinemanager.Parameter{}}}
	ps, mock := initializePipelineDB()
	pipelinesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "package_id"}).
		AddRow(1, nil, nil, nil, "Pipeline123", "", 1).
		AddRow(2, nil, nil, nil, "Pipeline456", "", 2)
	mock.ExpectQuery("SELECT (.*) FROM `pipelines`").WillReturnRows(pipelinesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)
	pipelines, _ := ps.ListPipelines()

	assert.Equal(t, expectedPipelines, pipelines, "Got unexpected pipelines")
}

func TestGetPipeline(t *testing.T) {
	expectedPipeline := pipelinemanager.Pipeline{
		Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []pipelinemanager.Parameter{}}
	ps, mock := initializePipelineDB()
	pipelinesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "package_id"}).
		AddRow(1, nil, nil, nil, "Pipeline123", "", 1)
	mock.ExpectQuery("SELECT (.*) FROM `pipelines`").WillReturnRows(pipelinesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)

	pkg, _ := ps.GetPipeline(123)

	assert.Equal(t, expectedPipeline, pkg, "Got unexpected pipeline")
}

func TestCreatePipeline(t *testing.T) {
	pkg := pipelinemanager.Pipeline{Name: "Pipeline123"}
	ps, mock := initializePipelineDB()
	mock.ExpectExec("INSERT INTO `pipelines`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), pkg.Name, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pkg, err := ps.CreatePipeline(pkg)
	assert.Nil(t, err, "Unexpected error creating pipeline")
	assert.Equal(t, uint(1), pkg.ID, "ID should be assigned")
}
