package dao

import (
	"database/sql"
	"errors"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
)

type PackageDaoInterface interface {
	ListPackages() ([]pipelinemanager.Package, error)
	GetPackage(packageId string) (pipelinemanager.Package, error)
}

type PackageDao struct {
	db *sql.DB
}

func (dao *PackageDao) ListPackages() ([]pipelinemanager.Package, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM package")

	if err != nil {
		return nil, util.NewInternalError("Failed to query the package table. Error:<%s>", err.Error())
	}

	var packages []pipelinemanager.Package

	for rows.Next() {
		var t pipelinemanager.Package
		err = rows.Scan(&t.Id, &t.Description)
		if err != nil {
			return nil, util.NewInternalError("Failed to parse the package row. Error:<%s>", err.Error())
		}
		packages = append(packages, t)
	}
	return packages, nil
}

func (dao *PackageDao) GetPackage(packageId string) (pipelinemanager.Package, error) {
	// TODO(yangpa): Ignore the implementation. Use ORM to fetch data later
	rows, err := dao.db.Query("SELECT * FROM package WHERE id=" + packageId)
	var t pipelinemanager.Package
	if err != nil {
		return t, err
	}

	if !rows.Next() {
		return t, errors.New("package not found")
	}

	err = rows.Scan(&t.Id, &t.Description)
	return t, err
}

// factory function for package DAO
func NewPackageDao(db *sql.DB) *PackageDao {
	return &PackageDao{db: db}
}
