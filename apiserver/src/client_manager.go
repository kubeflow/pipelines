package main

import (
	"encoding/json"
	"io/ioutil"
	"ml/apiserver/src/util"
	"os"

	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"

	"fmt"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/minio/minio-go"
)

const (
	k8sServiceHost   = "KUBERNETES_SERVICE_HOST"
	k8sTCPPort       = "KUBERNETES_PORT_443_TCP_PORT"
	k8sTokenFile     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	minioServiceHost = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort = "MINIO_SERVICE_SERVICE_PORT"
)

type DBConfig struct {
	DriverName     string
	DataSourceName string
}

type PackageManagerConfig struct {
	AccessKey       string
	SecretAccessKey string
	BucketName      string
}

type Config struct {
	DBConfig             DBConfig
	PackageManagerConfig PackageManagerConfig
}

// Container for all service clients
type ClientManager struct {
	db             *gorm.DB
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
}

func (clientManager *ClientManager) Init(config Config) {
	glog.Infof("initializing client manager")

	dbConfig := config.DBConfig

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(dbConfig.DriverName, dbConfig.DataSourceName)
	util.TerminateIfError(err)
	// Create table
	db.AutoMigrate(&pipelinemanager.Package{}, &pipelinemanager.Pipeline{}, &pipelinemanager.Parameter{})

	// Initiate package store
	clientManager.db = db
	clientManager.packageStore = storage.NewPackageStore(db)

	// Initiate pipeline store
	clientManager.db = db
	clientManager.pipelineStore = storage.NewPipelineStore(db)

	// Initiate job store
	argoClient := getArgoClient()
	clientManager.jobStore = storage.NewJobStore(argoClient)

	// Initiate package manager.
	clientManager.packageManager = getMinioClient(config.PackageManagerConfig)

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
// TODO(yangpa): Use Viper to get env variable as configuration. https://github.com/spf13/viper
func getArgoClient() storage.ArgoClientInterface {
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
	return &storage.ArgoClient{K8ServiceHost: k8ServiceHost, K8TCPPort: k8TCPPort, K8Token: string(k8TokenByte)}
}

// TODO(yangpa): Use Viper to get env variable as configuration. https://github.com/spf13/viper
func getMinioClient(config PackageManagerConfig) storage.PackageManagerInterface {
	minioServiceHost := os.Getenv(minioServiceHost)
	if minioServiceHost == "" {
		glog.Fatalf("Minio Service Host is not found.")
	}

	minioServicePort := os.Getenv(minioServicePort)
	if minioServicePort == "" {
		glog.Fatalf("Minio Service Port is not found.")
	}

	minioClient, err := minio.New(
		fmt.Sprintf("%s:%s", minioServiceHost, minioServicePort),
		config.AccessKey,
		config.SecretAccessKey,
		false)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}

	err = minioClient.MakeBucket(config.BucketName, "")
	if err != nil {
		// Check to see if we already own this bucket.
		exists, err := minioClient.BucketExists(config.BucketName)
		if err == nil && exists {
			glog.Infof("We already own %s\n", config.BucketName)
		} else {
			glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
		}
	}
	glog.Infof("Successfully created %s\n", config.BucketName)
	return &storage.MinioPackageManager{MinioClient: minioClient, BucketName: config.BucketName}
}

// NewClientManager creates and Init a new instance of ClientManager
func NewClientManager(configPath string) ClientManager {
	clientManager := ClientManager{}
	config := getConfig(configPath)
	clientManager.Init(config)

	return clientManager
}
