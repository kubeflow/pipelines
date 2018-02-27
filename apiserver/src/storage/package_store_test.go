package storage

import (
	"testing"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"github.com/jmoiron/sqlx"
	"ml/apiserver/src/message/pipelinemanager"
	"reflect"
)

func initialize() (PackageStoreInterface, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return &PackageStore{db: sqlxDB}, mock
}

func TestListPackages(t *testing.T) {
	expectedPackages := []pipelinemanager.Package{
		{Id: "123", Name: "Package123"},
		{Id: "456", Name: "Package456"}}
	ps, mock := initialize()
	rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow("123", "Package123", "").
			AddRow("456", "Package456", "")
	mock.ExpectQuery("SELECT (.*) FROM package ORDER BY package.name").WillReturnRows(rows)

	packages, _ := ps.ListPackages()
	if !reflect.DeepEqual(packages, expectedPackages) {
		t.Errorf("Unexpecte package returned. Expect %v. Got %v", expectedPackages, packages)
	}
}

func TestGetPackage(t *testing.T) {
	expectedPackage := pipelinemanager.Package{Id: "123", Name: "Package123"}
	ps, mock := initialize()
	rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow("123", "Package123", "")
	mock.ExpectQuery("SELECT (.*) FROM package").WillReturnRows(rows)

	packages, _ := ps.GetPackage("123")
	if !reflect.DeepEqual(packages, expectedPackage) {
		t.Errorf("Unexpecte package returned. Expect %v. Got %v", expectedPackage, packages)
	}
}

func TestCreatePackage(t *testing.T) {
	pkg := pipelinemanager.Package{Id: "123", Name: "Package123"}
	ps, mock := initialize()
	mock.ExpectExec("INSERT INTO package (.*)").
			WithArgs(pkg.Id, pkg.Name, pkg.Description).
			WillReturnResult(sqlmock.NewResult(0, 1))

	err := ps.CreatePackage(pkg)
	if err != nil {
		t.Errorf("Unexpected error creating package. Error: %s", err.Error())
	}
}
