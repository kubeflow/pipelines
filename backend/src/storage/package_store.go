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
	"fmt"
	"ml/backend/src/model"
	"ml/backend/src/util"

	"github.com/jinzhu/gorm"
)

type PackageStoreInterface interface {
	ListPackages(pageToken string, pageSize int, sortByFieldName string) ([]model.Package, string, error)
	GetPackage(packageId uint32) (*model.Package, error)
	DeletePackage(packageId uint32) error
	CreatePackage(*model.Package) (*model.Package, error)
	UpdatePackageStatus(uint32, model.PackageStatus) error
}

type PackageStore struct {
	db   *gorm.DB
	time util.TimeInterface
}

func (s *PackageStore) ListPackages(pageToken string, pageSize int, sortByFieldName string) ([]model.Package, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetPackageTablePrimaryKeyColumn())
	if err != nil {
		return nil, "", err
	}
	models, pageToken, err := listModel(paginationContext, s.queryPackageTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List package failed.")
	}
	return s.toPackages(models), pageToken, err
}

func (s *PackageStore) queryPackageTable(context *PaginationContext) ([]model.ListableDataModel, error) {
	var packages []model.Package
	query := s.db.Preload("Parameters").Where("Status = ? ", model.PackageReady)
	paginationQuery, err := toPaginationQuery(query, context)
	if err != nil {
		return nil, util.Wrap(err, "Error creating pagination query when listing packages.")
	}
	if r := paginationQuery.Limit(context.pageSize).Find(&packages); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list packages: %v", r.Error.Error())
	}
	return s.toListablePackages(packages), nil
}

func (s *PackageStore) GetPackage(id uint32) (*model.Package, error) {
	var pkg model.Package
	r := s.db.Preload("Parameters").Where("Status = ?", model.PackageReady).First(&pkg, id)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Package", fmt.Sprint(id))
	}
	if r.Error != nil {
		// TODO query can return multiple errors. log all of the errors when error handling v2 in place.
		return nil, util.NewInternalServerError(r.Error, "Failed to get package: %v", r.Error.Error())
	}
	return &pkg, nil
}

func (s *PackageStore) DeletePackage(id uint32) error {
	r := s.db.Exec(`DELETE FROM packages WHERE ID=?`, id)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to delete package: %v", r.Error.Error())
	}
	return nil
}

func (s *PackageStore) CreatePackage(p *model.Package) (*model.Package, error) {
	newPackage := *p
	now := s.time.Now().Unix()
	newPackage.CreatedAtInSec = now
	if r := s.db.Create(&newPackage); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to add package to package table: %v",
			r.Error.Error())
	}
	return &newPackage, nil
}

func (s *PackageStore) UpdatePackageStatus(id uint32, status model.PackageStatus) error {
	r := s.db.Exec(`UPDATE packages SET Status=? WHERE ID=?`, status, id)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to update the package metadata: %s", r.Error.Error())
	}
	return nil
}

func (s *PackageStore) toListablePackages(pkgs []model.Package) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(pkgs))
	for i := range models {
		models[i] = pkgs[i]
	}
	return models
}

func (s *PackageStore) toPackages(models []model.ListableDataModel) []model.Package {
	pkgs := make([]model.Package, len(models))
	for i := range models {
		pkgs[i] = models[i].(model.Package)
	}
	return pkgs
}

// factory function for package store
func NewPackageStore(db *gorm.DB, time util.TimeInterface) *PackageStore {
	return &PackageStore{db: db, time: time}
}
