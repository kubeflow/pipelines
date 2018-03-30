package client

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func CreateMySQLConfig(user string, mysqlServiceHost string,
	mysqlServicePort string, dbName string) *mysql.Config {
	return &mysql.Config{
		User:   user,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%s", mysqlServiceHost, mysqlServicePort),
		Params: map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local"},
		DBName: dbName,
	}
}

func CreateGormClient(driverName string, sqliteDatasourceName string, user string,
	mysqlServiceHost string, mysqlServicePort string, dbName string) (*gorm.DB, error) {

	var dataSourceName string

	switch driverName {
	case "mysql":
		dataSourceName = CreateMySQLConfig(user, mysqlServiceHost, mysqlServicePort, dbName).FormatDSN()
	case "sqlite3":
		dataSourceName = sqliteDatasourceName
	default:
		return nil, errors.Errorf("Driver %v is not supported", driverName)
	}

	return gorm.Open(driverName, dataSourceName)
}
