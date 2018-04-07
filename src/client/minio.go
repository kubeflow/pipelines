package client

import (
	"fmt"

	"github.com/golang/glog"
	minio "github.com/minio/minio-go"
)

func CreateMinioClientOrFatal(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string) *minio.Client {
	minioClient, err := minio.New(fmt.Sprintf("%s:%s", minioServiceHost, minioServicePort),
		accessKey, secretKey, false /* Secure connection */)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}
	return minioClient
}
