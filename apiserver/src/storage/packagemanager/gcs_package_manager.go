package packagemanager

import (
	"mime/multipart"
)

type GCSPackageManager struct {
}

func (m *GCSPackageManager) StorePackage(file multipart.File, fileHeader *multipart.FileHeader) error {
	// TODO
	return nil
}

func (m *GCSPackageManager) GetPackage(fileName string) (multipart.File, error) {
	// TODO
	return nil, nil
}
