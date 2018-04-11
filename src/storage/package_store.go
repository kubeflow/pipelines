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
	"ml/src/message"
	"ml/src/util"

	"github.com/jinzhu/gorm"
)

type PackageStoreInterface interface {
	ListPackages() ([]message.Package, error)
	GetPackage(packageId uint) (*message.Package, error)
	CreatePackage(*message.Package) error
}

type PackageStore struct {
	db *gorm.DB
}

func (s *PackageStore) ListPackages() ([]message.Package, error) {
	var packages []message.Package
	// List all packages.
	if r := s.db.Preload("Parameters").Find(&packages); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list packages: %v", r.Error.Error())
	}
	return packages, nil
}

func (s *PackageStore) GetPackage(id uint) (*message.Package, error) {
	var pkg message.Package
	r := s.db.Preload("Parameters").First(&pkg, id)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Package", fmt.Sprint(id))
	}
	if r.Error != nil {
		// TODO query can return multiple errors. log all of the errors when error handling v2 in place.
		return nil, util.NewInternalServerError(r.Error, "Failed to get package: %v", r.Error.Error())
	}
	return &pkg, nil
}

func (s *PackageStore) CreatePackage(p *message.Package) error {
	if r := s.db.Create(&p); r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to add package to package table: %v",
			r.Error.Error())
	}
	return nil
}

// factory function for package store
func NewPackageStore(db *gorm.DB) *PackageStore {
	return &PackageStore{db: db}
}
