// Copyright 2018 Google LLC
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
	"errors"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
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

func TestListPackagesError(t *testing.T) {
	ps, mock := initializePackageDB()
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnError(errors.New("something"))
	_, err := ps.ListPackages()

	assert.IsType(t, new(util.InternalError), err, "Expect to list packages to return error")
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

func TestGetPackageNotFoundError(t *testing.T) {
	ps, mock := initializePackageDB()
	mock.ExpectQuery("SELECT (.*) FROM `packages`").WillReturnError(errors.New("something"))
	_, err := ps.GetPackage(123)
	assert.IsType(t, new(util.ResourceNotFoundError), err, "Expect get package to return error")
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

func TestCreatePackageError(t *testing.T) {
	pkg := pipelinemanager.Package{Name: "Package123"}
	ps, mock := initializePackageDB()
	mock.ExpectExec("INSERT INTO `packages`").WillReturnError(errors.New("something"))

	_, err := ps.CreatePackage(pkg)
	assert.IsType(t, new(util.InternalError), err, "Expect create package to return error")
}
