package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"

	"ml/apiserver/src/dao"
	"ml/apiserver/src/util"

	"github.com/golang/glog"
	_ "github.com/mattn/go-sqlite3"
)

const (
	k8sServiceHost = "KUBERNETES_SERVICE_HOST"
	k8sTCPPort     = "KUBERNETES_PORT_443_TCP_PORT"
	k8sTokenFile   = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

type DBConfig struct {
	DriverName     string
	DataSourceName string
}

type Config struct {
	DBConfig DBConfig
}

// Container for all service clients
type ClientManager struct {
	db          *sql.DB
	packageDao dao.PackageDaoInterface
	jobDao      dao.JobDaoInterface
}

func (clientManager *ClientManager) Init(config Config) {
	glog.Infof("initializing client manager")

	dbConfig := config.DBConfig
	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := sql.Open(dbConfig.DriverName, dbConfig.DataSourceName)
	util.CheckErr(err)
	clientManager.db = db
	clientManager.packageDao = dao.NewPackageDao(db)

	argoClient := getArgoClient()
	clientManager.jobDao = dao.NewJobDao(argoClient)

	glog.Infof("initialized client manager successfully")
}

func (clientManager *ClientManager) End() {
	clientManager.db.Close()
}

// Get configurations from the user defined config file.
func getConfig(configPath string) Config {
	var config Config
	if configPath != "" {
		b, err := ioutil.ReadFile(configPath)
		if err != nil {
			glog.Fatalf("Failed to read config at %s: %v", configPath, err)
		}
		if err := json.Unmarshal(b, &config); err != nil {
			glog.Fatalf("Failed to parse config file at %s: %v", configPath, err)
		}
	}
	return config
}

// Get Argo's K8s CRD API client.
func getArgoClient() dao.ArgoClientInterface {
	k8ServiceHost := os.Getenv(k8sServiceHost)
	if k8ServiceHost == "" {
		glog.Fatalf("Kubernetes Service Host is not found.")
	}

	k8TCPPort := os.Getenv(k8sTCPPort)
	if k8TCPPort == "" {
		glog.Fatalf("Kubernetes TCP Port is not found.")
	}

	k8TokenByte, err := ioutil.ReadFile(k8sTokenFile)
	if err != nil {
		glog.Fatalf("Reading Kubernetes Token file failed. Error: %v", err)
	}

	if len(k8TokenByte) == 0 {
		glog.Fatalf("Reading Kubernetes Token file failed. No token found.")
	}
	return &dao.ArgoClient{K8ServiceHost: k8ServiceHost, K8TCPPort: k8TCPPort, K8Token: string(k8TokenByte)}
}

// NewClientManager creates and Init a new instance of ClientManager
func NewClientManager(configPath string) ClientManager {
	clientManager := ClientManager{}
	config := getConfig(configPath)
	clientManager.Init(config)

	return clientManager
}
