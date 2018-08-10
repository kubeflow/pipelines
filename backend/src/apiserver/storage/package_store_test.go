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

const (
	fakeUUID      = "123e4567-e89b-12d3-a456-426655440000"
	fakeUUIDTwo   = "123e4567-e89b-12d3-a456-426655440001"
	fakeUUIDThree = "123e4567-e89b-12d3-a456-426655440002"
	fakeUUIDFour  = "123e4567-e89b-12d3-a456-426655440003"
)

func createPkg(name string) *model.Package {
	return &model.Package{Name: name, Parameters: `[{"Name": "param1"}]`, Status: model.PackageReady}
}

func TestListPackages_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	packageStore.CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})
	expectedPkg1 := model.Package{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg2 := model.Package{
		UUID:           fakeUUIDTwo,
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
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	packageStore.CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg4 := model.Package{
		UUID:           fakeUUIDFour,
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
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg3 := model.Package{
		UUID:           fakeUUIDThree,
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
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	packageStore.CreatePackage(createPkg("pkg2"))
	packageStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	packageStore.CreatePackage(createPkg("pkg1"))

	expectedPkg2 := model.Package{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg3 := model.Package{
		UUID:           fakeUUIDThree,
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
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady}
	expectedPkg4 := model.Package{
		UUID:           fakeUUIDFour,
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
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		UUID:           fakeUUID,
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
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	_, _, err := packageStore.ListPackages("" /*pageToken*/, 2 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/, false /*isDesc*/)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	pkgExpected := model.Package{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageReady,
	}

	pkg, err := packageStore.GetPackage(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestGetPackage_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})

	_, err := packageStore.GetPackage(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	_, err := packageStore.GetPackage(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	_, err := packageStore.GetPackage("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return internal error")
}

func TestCreatePackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pkgExpected := model.Package{
		UUID:           fakeUUID,
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
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()

	_, err := packageStore.CreatePackage(pkg)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create package to return error")
}

func TestDeletePackage(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	packageStore.CreatePackage(createPkg("pkg1"))
	err := packageStore.DeletePackage(fakeUUID)
	assert.Nil(t, err)
	_, err = packageStore.GetPackage(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePackageError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	err := packageStore.DeletePackage(fakeUUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePackageStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pkg, err := packageStore.CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)
	pkgExpected := model.Package{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PackageDeleting,
	}
	err = packageStore.UpdatePackageStatus(pkg.UUID, model.PackageDeleting)
	assert.Nil(t, err)
	pkg, err = packageStore.GetPackageWithStatus(fakeUUID, model.PackageDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg)
}

func TestUpdatePackageStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	packageStore := NewPackageStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	err := packageStore.UpdatePackageStatus(fakeUUID, model.PackageDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
