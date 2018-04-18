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
	"ml/src/model"
	"ml/src/util"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createPkg(name string) *model.Package {
	return &model.Package{Name: name, Parameters: []model.Parameter{{Name: "param1"}}}
}

func TestListPackages(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageStore().CreatePackage(createPkg("pkg2"))
	expectedPkg1 := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}}}
	expectedPkg2 := model.Package{
		ID:             2,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 2, OwnerType: "packages"}}}
	pkgsExpected := []model.Package{expectedPkg1, expectedPkg2}

	pkgs, err := store.PackageStore().ListPackages()
	assert.Nil(t, err)
	assert.Equal(t, pkgsExpected, pkgs, "Got unexpected packages.")
}

func TestListPackagesError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, err := store.PackageStore().ListPackages()
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected to list packages to return error")
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
	}

	pkg, err := store.PackageStore().GetPackage(1)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected package.")
}

func TestGetPackage_NotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	_, err := store.PackageStore().GetPackage(1)
	assert.Equal(t, http.StatusNotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return not found")
}

func TestGetPackage_InternalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, err := store.PackageStore().GetPackage(123)
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected get package to return internal error")
}

func TestCreatePackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pkgExpected := model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     []model.Parameter{{Name: "param1", OwnerID: 1, OwnerType: "packages"}}}

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
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected create package to return error")
}
