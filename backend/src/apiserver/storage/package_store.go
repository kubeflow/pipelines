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
	"bytes"
	"database/sql"
	"fmt"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type PackageStoreInterface interface {
	ListPackages(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Package, string, error)
	GetPackage(packageId string) (*model.Package, error)
	GetPackageWithStatus(id string, status model.PackageStatus) (*model.Package, error)
	DeletePackage(packageId string) error
	CreatePackage(*model.Package) (*model.Package, error)
	UpdatePackageStatus(string, model.PackageStatus) error
}

type PackageStore struct {
	db   *sql.DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

func (s *PackageStore) ListPackages(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Package, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetPackageTablePrimaryKeyColumn(), isDesc)
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
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM packages WHERE Status = '%v'", model.PackageReady))
	toPaginationQuery("AND", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))
	r, err := s.db.Query(query.String())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list packages: %v", err.Error())
	}
	defer r.Close()
	packages, err := s.scanRows(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list packages: %v", err.Error())
	}
	return s.toListablePackages(packages), nil
}

func (s *PackageStore) scanRows(rows *sql.Rows) ([]model.Package, error) {
	var packages []model.Package
	for rows.Next() {
		var uuid, name, parameters, description string
		var createdAtInSec int64
		var status model.PackageStatus
		if err := rows.Scan(&uuid, &createdAtInSec, &name, &description, &parameters, &status); err != nil {
			return packages, err
		}
		packages = append(packages, model.Package{
			UUID:           uuid,
			CreatedAtInSec: createdAtInSec,
			Name:           name,
			Description:    description,
			Parameters:     parameters,
			Status:         status})
	}
	return packages, nil
}

func (s *PackageStore) GetPackage(id string) (*model.Package, error) {
	return s.GetPackageWithStatus(id, model.PackageReady)
}

func (s *PackageStore) GetPackageWithStatus(id string, status model.PackageStatus) (*model.Package, error) {
	r, err := s.db.Query("SELECT * FROM packages WHERE uuid=? AND status=? LIMIT 1", id, status)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get package: %v", err.Error())
	}
	defer r.Close()
	packages, err := s.scanRows(r)

	if err != nil || len(packages) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get package: %v", err.Error())
	}
	if len(packages) == 0 {
		return nil, util.NewResourceNotFoundError("Package", fmt.Sprint(id))
	}
	return &packages[0], nil
}

func (s *PackageStore) DeletePackage(id string) error {
	_, err := s.db.Exec(`DELETE FROM packages WHERE UUID=?`, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete package: %v", err.Error())
	}
	return nil
}

func (s *PackageStore) CreatePackage(p *model.Package) (*model.Package, error) {
	newPackage := *p
	now := s.time.Now().Unix()
	newPackage.CreatedAtInSec = now
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a package id.")
	}
	newPackage.UUID = id.String()
	stmt, err := s.db.Prepare(
		`INSERT INTO packages (UUID, CreatedAtInSec,Name,Description,Parameters,Status)
						VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add package to package table: %v",
			err.Error())
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		newPackage.UUID,
		newPackage.CreatedAtInSec,
		newPackage.Name,
		newPackage.Description,
		newPackage.Parameters,
		string(newPackage.Status))
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add package to package table: %v",
			err.Error())
	}
	return &newPackage, nil
}

func (s *PackageStore) UpdatePackageStatus(id string, status model.PackageStatus) error {
	_, err := s.db.Exec(`UPDATE packages SET Status=? WHERE UUID=?`, status, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to update the package metadata: %s", err.Error())
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
func NewPackageStore(db *sql.DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *PackageStore {
	return &PackageStore{db: db, time: time, uuid: uuid}
}
