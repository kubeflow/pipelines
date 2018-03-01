package storage

import (
	"ml/apiserver/src/message/pipelinemanager"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func initializePackageDB() (PackageStoreInterface, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("mysql", db)
	return &PackageStore{db: gormDB}, mock
}

func TestListPackages(t *testing.T) {
	expectedPackages := []pipelinemanager.Package{
		{Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Package123", Parameters: []pipelinemanager.Parameter{}},
		{Metadata: &pipelinemanager.Metadata{ID: 2}, Name: "Package456", Parameters: []pipelinemanager.Parameter{}}}
	ps, mock := initializePackageDB()
	packagesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description"}).
		AddRow(1, nil, nil, nil, "Package123", "").
		AddRow(2, nil, nil, nil, "Package456", "")
	mock.ExpectQuery("SELECT (.*) FROM `packages`").WillReturnRows(packagesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)
	packages, _ := ps.ListPackages()

	assert.Equal(t, expectedPackages, packages, "Got unexpected packages")
}

func TestGetPackage(t *testing.T) {
	expectedPackage := pipelinemanager.Package{
		Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Package123", Parameters: []pipelinemanager.Parameter{}}
	ps, mock := initializePackageDB()
	packagesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description"}).
		AddRow(1, nil, nil, nil, "Package123", "")
	mock.ExpectQuery("SELECT (.*) FROM `packages`").WillReturnRows(packagesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)

	pkg, _ := ps.GetPackage(123)

	assert.Equal(t, expectedPackage, pkg, "Got unexpected package")
}

func TestCreatePackage(t *testing.T) {
	pkg := pipelinemanager.Package{Name: "Package123"}
	ps, mock := initializePackageDB()
	mock.ExpectExec("INSERT INTO `packages`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), pkg.Name, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pkg, err := ps.CreatePackage(pkg)
	assert.Nil(t, err, "Unexpected error creating package")
	assert.Equal(t, uint(1), pkg.ID, "ID should be assigned")
}
