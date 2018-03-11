package storage

import (
	"errors"
	"io"
	"ml/apiserver/src/util"
	"strings"
	"testing"

	"github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

type FakeMinioClient struct {
}

func (c *FakeMinioClient) PutObject(bucketName, objectName string, reader io.Reader,
	objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return 1, nil
}
func (c *FakeMinioClient) GetObject(bucketName, objectName string,
	opts minio.GetObjectOptions) (io.Reader, error) {
	return strings.NewReader("I'm a file"), nil
}

type FakeBadMinioClient struct {
}

func (c *FakeBadMinioClient) PutObject(bucketName, objectName string, reader io.Reader,
	objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return 0, errors.New("some error")
}
func (c *FakeBadMinioClient) GetObject(bucketName, objectName string,
	opts minio.GetObjectOptions) (io.Reader, error) {
	return nil, errors.New("some error")
}

func TestCreatePackageFile(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeMinioClient{}}
	error := manager.CreatePackageFile([]byte{}, "file  name")
	assert.Nil(t, error, "Expect create package successfully.")
}

func TestCreatePackageFileError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	error := manager.CreatePackageFile([]byte{}, "field name")
	assert.IsType(t, new(util.InternalError), error, "Expect new internal error.")
}

func TestGetPackageFile(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeMinioClient{}}
	file, error := manager.GetPackageFile("file name")
	assert.Nil(t, error, "Expect get package successfully.")
	assert.Equal(t, file, []byte("I'm a file"))
}

func TestGetPackageFileError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	_, error := manager.GetPackageFile("file name")
	assert.IsType(t, new(util.InternalError), error, "Expect new internal error.")
}
