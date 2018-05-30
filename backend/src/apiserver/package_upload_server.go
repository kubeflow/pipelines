// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/resource"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/pkg/errors"
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
	apiPkg, err := ToApiPackage(newPkg)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, errors.Wrap(err, "Error creating package"))
		return
	}
	pkgJson, err := json.Marshal(apiPkg)
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
