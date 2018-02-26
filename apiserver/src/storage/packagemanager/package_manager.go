package packagemanager

import (
	"mime/multipart"
)

type PackageManagerInterface interface {
	// Store the package file
	StorePackage(file multipart.File, fileHeader *multipart.FileHeader) error

	// Get the package file
	GetPackage(fileName string) (multipart.File, error)
}
