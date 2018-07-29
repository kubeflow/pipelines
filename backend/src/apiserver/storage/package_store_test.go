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
	"testing"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func createPkg(name string) *model.Package {
	return &model.Package{Name: name, Parameters: `[{"Name": "param1"}]`, Status: model.PackageReady}
}

func TestListPackages_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())

	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1, expectedPkg2}

	pkgs, nextPageToken, err := packageStore.ListPackages("" /*pageToken*/, 10 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPackages_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg4 := model.Package{
		ID:             4,
		CreatedAtInSec: 4,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1, expectedPkg4}
	pkgs, nextPageToken, err := packageStore.ListPackages("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)

	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg3 := model.Package{
		ID:             3,
		CreatedAtInSec: 3,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected2 := []model.Package{expectedPkg2, expectedPkg3}

	pkgs, nextPageToken, err = packageStore.ListPackages(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pkgsExpected2, pkgs)
}

func TestListPackages_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.CreatePackage(createPkg("pkg1"))

	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg3 := model.Package{
		ID:             3,
		CreatedAtInSec: 3,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg3, expectedPkg2}
	pkgs, nextPageToken, err := packageStore.ListPackages("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/, true /*isDesc*/)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)

	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg4 := model.Package{
		ID:             4,
		CreatedAtInSec: 4,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected2 := []model.Package{expectedPkg4, expectedPkg1}
	pkgs, nextPageToken, err = packageStore.ListPackages(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/, true /*isDesc*/)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pkgsExpected2, pkgs)
}

func TestListPackages_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1}

	pkgs, nextPageToken, err := packageStore.ListPackages("" /*pageToken*/, 2 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/, false /*isDesc*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPackagesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, _, err := packageStore.ListPackages("" /*pageToken*/, 2 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/, false /*isDesc*/)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(createPkg("pkg1"))
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady,
	}

	pkg, err := packageStore.GetPackage(1)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestGetPackage_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})

	_, err := packageStore.GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())

	_, err := packageStore.GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, err := packageStore.GetPackage(123)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return internal error")
}

func TestCreatePackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}

	pkg := createPkg("pkg1")
	pkg, err := packageStore.CreatePackage(pkg)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestCreatePackageError(t *testing.T) {
	pkg := &model.Package{Name: "Package123"}
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	db.Close()

	_, err := packageStore.CreatePackage(pkg)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create package to return error")
}

func TestDeletePackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	packageStore.CreatePackage(createPkg("pkg1"))
	err := packageStore.DeletePackage(1)
	assert.Nil(t, err)
	_, err = packageStore.GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePackageError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	err := packageStore.DeletePackage(1)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePackageStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	pkg, err := packageStore.CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageDeleting,
	}
	err = packageStore.UpdatePackageStatus(pkg.ID, model.PackageDeleting)
	assert.Nil(t, err)
	r := db.First(&pkg, pkg.ID)
	assert.Nil(t, r.Error)
	assert.Equal(t, pkgExpected, *pkg)
}

func TestUpdatePackageStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	err := packageStore.UpdatePackageStatus(1, model.PackageDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
