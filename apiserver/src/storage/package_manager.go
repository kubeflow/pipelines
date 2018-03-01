package storage

import (
	"bytes"
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
	MinioClient *minio.Client
	BucketName  string
}

func (m *MinioPackageManager) CreatePackageFile(template []byte, fileName string) error {
	_, err := m.MinioClient.PutObject(m.BucketName, fileName, bytes.NewReader(template), -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalError("Failed to store a new package.", err.Error())
	}
	return nil
}

func (m *MinioPackageManager) GetPackageFile(fileName string) ([]byte, error) {
	object, err := m.MinioClient.GetObject(m.BucketName, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalError("Failed to store a new package.", err.Error())
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	return buf.Bytes(), nil
}
