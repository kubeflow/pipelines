package common

import (
	"io"

	"github.com/minio/minio-go"
)

// Create interface for minio client struct, making it more unit testable.
type MinioClientInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error)
	GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (io.Reader, error)
}

type MinioClient struct {
	Client *minio.Client
}

func (c *MinioClient) PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return c.Client.PutObject(bucketName, objectName, reader, objectSize, opts)
}

func (c *MinioClient) GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (io.Reader, error) {
	return c.Client.GetObject(bucketName, objectName, opts)
}
