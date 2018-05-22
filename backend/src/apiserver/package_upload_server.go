package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ml/backend/api"
	"ml/backend/src/resource"
	"ml/backend/src/util"
	"net/http"

	"github.com/golang/glog"
)

type PackageUploadServer struct {
	resourceManager *resource.ResourceManager
}

func (s *PackageUploadServer) UploadPackage(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Upload package called")
	file, header, err := r.FormFile("uploadfile")
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read package form file"))
		return
	}
	defer file.Close()

	// Read file to byte array
	pkgFile, err := ioutil.ReadAll(file)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Error read package bytes"))
		return
	}

	newPkg, err := s.resourceManager.CreatePackage(header.Filename, pkgFile)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating package"))
		return
	}

	pkgJson, err := json.Marshal(ToApiPackage(newPkg))
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating package"))
		return
	}
	w.Write(pkgJson)
}

func (s *PackageUploadServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to upload packages. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error uploading package"))
	}
	w.Write(errBytes)
}
