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
	"ml/backend/src/model"
	"ml/backend/src/util"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func createPkg(name string) *model.Package {
	return &model.Package{Name: name, Parameters: []model.Parameter{{Name: "param1"}}, Status: model.PackageReady}
}

func TestListPackages_FilterOutNotReady(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageStore().CreatePackage(createPkg("pkg2"))
	store.PackageStore().CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageReady}
	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 2, OwnerType: "packages"}},
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1, expectedPkg2}

	pkgs, nextPageToken, err := store.PackageStore().ListPackages("" /*pageToken*/, 10 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPackages_Pagination(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageStore().CreatePackage(createPkg("pkg2"))
	store.PackageStore().CreatePackage(createPkg("pkg2"))
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageReady}
	expectedPkg4 := model.Package{
		ID:             4,
		CreatedAtInSec: 4,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 4, OwnerType: "packages"}},
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1, expectedPkg4}
	pkgs, nextPageToken, err := store.PackageStore().ListPackages("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)

	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 2, OwnerType: "packages"}},
		Status:         model.PackageReady}
	expectedPkg3 := model.Package{
		ID:             3,
		CreatedAtInSec: 3,
		Name:           "pkg2",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 3, OwnerType: "packages"}},
		Status:         model.PackageReady}
	pkgsExpected2 := []model.Package{expectedPkg2, expectedPkg3}

	pkgs, nextPageToken, err = store.PackageStore().ListPackages(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pkgsExpected2, pkgs)
}

func TestListPackages_Pagination_LessThanPageSize(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageReady}
	pkgsExpected := []model.Package{expectedPkg1}

	pkgs, nextPageToken, err := store.PackageStore().ListPackages("" /*pageToken*/, 2 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPackagesError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, _, err := store.PackageStore().ListPackages("" /*pageToken*/, 2 /*pageSize*/, model.GetPackageTablePrimaryKeyColumn() /*sortByFieldName*/)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageReady,
	}

	pkg, err := store.PackageStore().GetPackage(1)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestGetPackage_NotFound_Creating(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(&model.Package{Name: "pkg3", Status: model.PackageCreating})

	_, err := store.PackageStore().GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_NotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	_, err := store.PackageStore().GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_InternalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, err := store.PackageStore().GetPackage(123)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return internal error")
}

func TestCreatePackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageReady}

	pkg := createPkg("pkg1")
	pkg, err := store.PackageStore().CreatePackage(pkg)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestCreatePackageError(t *testing.T) {
	pkg := &model.Package{Name: "Package123"}
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()

	_, err := store.PackageStore().CreatePackage(pkg)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create package to return error")
}

func TestDeletePackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	err := store.PackageStore().DeletePackage(1)
	assert.Nil(t, err)
	_, err = store.PackageStore().GetPackage(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePackageError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	err := store.PackageStore().DeletePackage(1)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePackageStatus(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pkg, err := store.PackageStore().CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}},
		Status:         model.PackageDeleting,
	}
	err = store.PackageStore().UpdatePackageStatus(pkg.ID, model.PackageDeleting)
	assert.Nil(t, err)
	r := store.DB().First(&pkg, pkg.ID)
	assert.Nil(t, r.Error)
	assert.Equal(t, pkgExpected, *pkg)
}

func TestUpdatePackageStatusError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	err := store.PackageStore().UpdatePackageStatus(1, model.PackageDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
