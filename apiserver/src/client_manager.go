package main

import (
	"database/sql"

	"github.com/golang/glog"
	"github.com/googleprivate/ml/apiserver/src/dao"
	"github.com/googleprivate/ml/apiserver/src/util"
	_ "github.com/mattn/go-sqlite3"
)

// Container for all service clients
type ClientManager struct {
	db          *sql.DB
	templateDao dao.TemplateDaoInterface
}

func (clientManager *ClientManager) Init(config Config) {
	glog.Infof("initializing database connection")

	dbConfig := config.DBConfig
	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := sql.Open(dbConfig.DriverName, dbConfig.DataSourceName)
	util.CheckErr(err)
	clientManager.db = db
	clientManager.templateDao = dao.NewTemplateDao(db)

	glog.Infof("initializing database successfully")
}

func (clientManager *ClientManager) End() {
	clientManager.db.Close()
}

// NewClientManager creates and Init a new instance of ClientManager
func NewClientManager(config Config) ClientManager {
	clientManager := ClientManager{}
	clientManager.Init(config)

	return clientManager
}
