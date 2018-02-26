package packagemanager

import (
	"io"
	"mime/multipart"
	"ml/apiserver/src/util"
	"os"
)

// Managing package using K8s PersistentVolume
type PersistentVolumePackageManager struct {
	VolumeLocation string
}

func (m *PersistentVolumePackageManager) StorePackage(file multipart.File, fileHeader *multipart.FileHeader) error {
	fileName := fileHeader.Filename

	// Create a file with the same name as user provided
	out, err := os.OpenFile(m.VolumeLocation+fileName,
		os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		return util.NewInternalError("Failed to create a new package. Error:<%s>", err.Error())
	}
	defer out.Close()

	io.Copy(out, file)
	return nil
}

func (m *PersistentVolumePackageManager) GetPackage(fileName string) (multipart.File, error) {
	// TODO
	return nil, nil
}
