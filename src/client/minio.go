package client

import (
	"fmt"

	"github.com/golang/glog"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

func CreateMinioClient(minioServiceHost string, minioServicePort string,
		accessKey string, secretKey string) (*minio.Client, error) {
	minioClient, err := minio.New(fmt.Sprintf("%s:%s", minioServiceHost, minioServicePort),
		accessKey, secretKey, false /* Secure connection */)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating minio client: %+v", err)
	}
	return minioClient, nil
}

func CreateMinioClientOrFatal(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string) *minio.Client {
	minioClient, err := CreateMinioClient(minioServiceHost, minioServicePort,
		accessKey, secretKey)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}
	return minioClient
}
