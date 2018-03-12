package main

import (
	"fmt"
	"io/ioutil"
	"ml/apiserver/src/common"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/util"

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

// Container for all service clients
type ClientManager struct {
	db             *gorm.DB
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
}

func (clientManager *ClientManager) Init() {
	glog.Infof("Initializing client manager")

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(getConfig("DBConfig.DriverName"), getConfig("DBConfig.DataSourceName"))
	util.TerminateIfError(err)

	// Create table
	db.AutoMigrate(&pipelinemanager.Package{}, &pipelinemanager.Pipeline{}, &pipelinemanager.Parameter{})

	// Initialize package store
	clientManager.db = db
	clientManager.packageStore = storage.NewPackageStore(db)

	// Initialize pipeline store
	clientManager.db = db
	clientManager.pipelineStore = storage.NewPipelineStore(db)

	// Initialize job store
	argoClient := getArgoClient()
	clientManager.jobStore = storage.NewJobStore(argoClient)

	// Initialize package manager.
	clientManager.packageManager = getMinioClient()

	glog.Infof("Client manager initialized successfully")
}

func (clientManager *ClientManager) End() {
	clientManager.db.Close()
}

// Get Argo's K8s CRD API client.
func getArgoClient() storage.ArgoClientInterface {
	k8ServiceHost := getConfig(k8sServiceHost)

	k8TCPPort := getConfig(k8sTCPPort)

	k8TokenByte, err := ioutil.ReadFile(k8sTokenFile)
	if err != nil {
		glog.Fatalf("Reading Kubernetes Token file failed. Error: %v", err)
	}

	if len(k8TokenByte) == 0 {
		glog.Fatalf("Reading Kubernetes Token file failed. No token found.")
	}
	return &storage.ArgoClient{K8ServiceHost: k8ServiceHost, K8TCPPort: k8TCPPort, K8Token: string(k8TokenByte)}
}

func getMinioClient() storage.PackageManagerInterface {
	minioServiceHost := getConfig(minioServiceHost)
	minioServicePort := getConfig(minioServicePort)
	minioClient, err := minio.New(
		fmt.Sprintf("%s:%s", minioServiceHost, minioServicePort),
		getConfig("PackageManagerConfig.AccessKey"),
		getConfig("PackageManagerConfig.SecretAccessKey"),
		false /* Secure connection */)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}

	bucketName := getConfig("PackageManagerConfig.BucketName")
	err = minioClient.MakeBucket(bucketName, "")
	if err != nil {
		// Check to see if we already own this bucket.
		exists, err := minioClient.BucketExists(bucketName)
		if err == nil && exists {
			glog.Infof("We already own %s\n", bucketName)
		} else {
			glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
		}
	}
	glog.Infof("Successfully created %s\n", bucketName)
	return storage.NewMinioPackageManager(&common.MinioClient{Client: minioClient}, bucketName)
}

// newClientManager creates and Init a new instance of ClientManager
func newClientManager() ClientManager {
	clientManager := ClientManager{}
	clientManager.Init()

	return clientManager
}
