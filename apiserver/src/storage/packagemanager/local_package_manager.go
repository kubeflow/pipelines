package packagemanager

import (
	"io"
	"mime/multipart"
	"ml/apiserver/src/util"
	"os"
	"bytes"
	"io/ioutil"
)

// Managing package using K8s PersistentVolume
type PersistentVolumePackageManager struct {
	VolumeLocation string
}

func (m *PersistentVolumePackageManager) CreatePackageFile(file multipart.File, fileHeader *multipart.FileHeader) error {
	fileName := fileHeader.Filename

	// Create a file with the same name as user provided
	out, err := os.OpenFile(m.VolumeLocation+fileName,
		os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		return util.NewInternalError("Failed to store a new package.", err.Error())
	}
	defer out.Close()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		return util.NewInternalError("Failed to copy package.", err.Error())
	}

	err = checkValidPackage(buf.Bytes())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, buf)
	if err != nil {
		return util.NewInternalError("Failed to store the package", err.Error())
	}

	return nil
}

func (m *PersistentVolumePackageManager) GetPackageFile(fileName string) ([]byte, error) {
	b, err := ioutil.ReadFile(m.VolumeLocation + fileName)
	if err != nil {
		return nil, util.NewInternalError("Failed to retrieve the package", err.Error())
	}
	return b, nil
}
