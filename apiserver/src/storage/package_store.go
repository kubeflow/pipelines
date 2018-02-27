package storage

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"

	"github.com/jmoiron/sqlx"
)

type PackageStoreInterface interface {
	ListPackages() ([]pipelinemanager.Package, error)
	GetPackage(packageId string) (pipelinemanager.Package, error)
	CreatePackage(pipelinemanager.Package) error
}

type PackageStore struct {
	db *sqlx.DB
}

func (s *PackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	var packages []pipelinemanager.Package
	err := s.db.Select(&packages, "SELECT * FROM package ORDER BY package.name")

	if err != nil {
		return nil, util.NewInternalError("Failed to list packages", err.Error())
	}
	return packages, nil
}

func (s *PackageStore) GetPackage(packageId string) (pipelinemanager.Package, error) {
	var pkg pipelinemanager.Package
	err := s.db.Get(&pkg, "SELECT * FROM package WHERE id = $1", packageId)
	if err != nil {
		// Error returns when no package found.
		return pkg, util.NewResourceNotFoundError("Package", packageId)
	}
	return pkg, nil
}

func (s *PackageStore) CreatePackage(p pipelinemanager.Package) error {
	_, err := s.db.NamedExec("INSERT INTO package (id, name, description) VALUES(:id, :name, :description)", p)
	if err != nil {
		return util.NewInternalError("Failed to add package to package table", err.Error())
	}
	return nil
}

// factory function for package store
func NewPackageStore(db *sqlx.DB) *PackageStore {
	return &PackageStore{db: db}
}
