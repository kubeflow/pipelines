package main

import (
	"database/sql"
	"github.com/googleprivate/ml/webserver/src/util"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

// Container for all service clients
type ClientManager struct {
	db *sql.DB
}

func (c *ClientManager) Init(config Config) {
	log.Printf("initializing database connection")

	dbConfig := config.DBConfig
	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := sql.Open(dbConfig.DriverName, dbConfig.DataSourceName)
	util.CheckErr(err)
	c.db = db

	log.Printf("initializing database successfully")
}

func (c *ClientManager) End() {
	c.db.Close()
}

// NewClientManager creates and Init a new instance of ClientManager
func NewClientManager(config Config) ClientManager {
	cm := ClientManager{}
	cm.Init(config)

	return cm
}
