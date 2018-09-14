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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

// These are valid conditions of a ScheduledWorkflow.
const (
	FormFileKey        = "uploadfile"
	NameQueryStringKey = "name"
	MaxFileNameLength  = 100
)

type PipelineUploadServer struct {
	resourceManager *resource.ResourceManager
}

// HTTP multipart endpoint for uploading pipeline file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) UploadPipeline(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Upload pipeline called")
	file, header, err := r.FormFile(FormFileKey)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read pipeline form file"))
		return
	}
	defer file.Close()

	// Read file to byte array
	pipelineFile, err := ioutil.ReadAll(file)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Error read pipeline bytes"))
		return
	}

	encodedFileName := r.URL.Query().Get(NameQueryStringKey)
	fileNameBytes, err := base64.StdEncoding.DecodeString(encodedFileName)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Invalid file name."))
		return
	}
	fileName := string(fileNameBytes)
	if fileName == "" {
		fileName = header.Filename
	}
	if len(fileName) > MaxFileNameLength {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.NewInvalidInputError("File name too long. Support maximum length of 100"))
		return
	}
	newPipeline, err := s.resourceManager.CreatePipeline(fileName, pipelineFile)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline"))
		return
	}
	apiPipeline := ToApiPipeline(newPipeline)
	pipelineJson, err := json.Marshal(apiPipeline)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline"))
		return
	}
	w.Write(pipelineJson)
}

func (s *PipelineUploadServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to upload pipelines. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error uploading pipeline"))
	}
	w.Write(errBytes)
}
