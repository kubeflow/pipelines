package storage

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"

	"github.com/jinzhu/gorm"
)

type PackageStoreInterface interface {
	ListPackages() ([]pipelinemanager.Package, error)
	GetPackage(packageId string) (pipelinemanager.Package, error)
	CreatePackage(pipelinemanager.Package) (pipelinemanager.Package, error)
}

type PackageStore struct {
	db *gorm.DB
}

func (s *PackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	var packages []pipelinemanager.Package
	// List all packages.
	if r := s.db.Find(&packages); r.Error != nil {
		return nil, util.NewInternalError("Failed to list packages", r.Error.Error())
	}
	return packages, nil
}

func (s *PackageStore) GetPackage(id string) (pipelinemanager.Package, error) {
	var pkg pipelinemanager.Package
	if r := s.db.First(&pkg, id); r.Error != nil {
		// Error returns when no package found.
		return pkg, util.NewResourceNotFoundError("Package", id)
	}
	return pkg, nil
}

func (s *PackageStore) CreatePackage(p pipelinemanager.Package) (pipelinemanager.Package, error) {
	if r := s.db.Create(&p); r.Error != nil {
		return p, util.NewInternalError("Failed to add package to package table", r.Error.Error())
	}
	return p, nil
}

// factory function for package store
func NewPackageStore(db *gorm.DB) *PackageStore {
	return &PackageStore{db: db}
}
