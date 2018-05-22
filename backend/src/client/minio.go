package client

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
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
	accessKey string, secretKey string, initConnectionTimeout time.Duration) *minio.Client {
	var minioClient *minio.Client
	var err error
	var operation = func() error {
		minioClient, err = CreateMinioClient(minioServiceHost, minioServicePort,
			accessKey, secretKey)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}
	return minioClient
}
