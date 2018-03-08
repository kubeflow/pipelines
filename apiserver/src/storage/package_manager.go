package storage

import (
	"bytes"
	"ml/apiserver/src/common"
	"ml/apiserver/src/util"

	"github.com/minio/minio-go"
)

// Manager managing acutal package file.
type PackageManagerInterface interface {
	// Create the package file
	CreatePackageFile(template []byte, fileName string) error

	// Get the package file
	GetPackageFile(fileName string) ([]byte, error)
}

// Managing package using Minio
type MinioPackageManager struct {
	minioClient common.MinioClientInterface
	bucketName  string
}

func (m *MinioPackageManager) CreatePackageFile(template []byte, fileName string) error {
	_, err := m.minioClient.PutObject(m.bucketName, fileName, bytes.NewReader(template), -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalError("Failed to store a new package.", err.Error())
	}
	return nil
}

func (m *MinioPackageManager) GetPackageFile(fileName string) ([]byte, error) {
	reader, err := m.minioClient.GetObject(m.bucketName, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalError("Failed to store a new package.", err.Error())
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.Bytes(), nil
}

func NewMinioPackageManager(minioClient common.MinioClientInterface, bucketName string) *MinioPackageManager {
	return &MinioPackageManager{minioClient: minioClient, bucketName: bucketName}
}
