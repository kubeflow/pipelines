package storage

import (
	"database/sql"
	"fmt"

	"github.com/golang/glog"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

func NewFakeDb() (*sql.DB, error) {
	// Initialize GORM
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}
	// Create tables
	db.AutoMigrate(&model.Pipeline{}, &model.RunDetail{}, &model.JobDetail{})
	return db.DB(), nil
}

func NewFakeDbOrFatal() *sql.DB {
	db, err := NewFakeDb()
	if err != nil {
		println(err.Error())
		glog.Fatalf("The fake DB doesn't create successfully. Fail fast.")
	}
	return db
}
