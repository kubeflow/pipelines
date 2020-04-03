// Copyright 2020 Google LLC
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

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
)

type OperationType string

const (
	OperationTypeAdd     OperationType = "add"
	OperationTypeReplace OperationType = "replace"
	OperationTypeRemove  OperationType = "remove"
)

// patchOperation is an operation of a JSON patch, see https://tools.ietf.org/html/rfc6902 .
type patchOperation struct {
	Op    OperationType `json:"op"`
	Path  string        `json:"path"`
	Value interface{}   `json:"value,omitempty"`
}

// admitFunc is a callback for admission controller logic. Given an AdmissionRequest, it returns the sequence of patch
// operations to be applied in case of success, or the error that will be shown when the operation is rejected.
type admitFunc func(_ *v1beta1.AdmissionRequest, clientMgr ClientManagerInterface) ([]patchOperation, error)

const (
	ContentType     string = "Content-Type"
	JsonContentType string = "application/json"
)

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

// isKubeNamespace checks if the given namespace is a Kubernetes-owned namespace.
func isKubeNamespace(ns string) bool {
	return ns == metav1.NamespacePublic || ns == metav1.NamespaceSystem
}

// doServeAdmitFunc parses the HTTP request for an admission controller webhook, and -- in case of a well-formed
// request -- delegates the admission control logic to the given admitFunc. The response body is then returned as raw
// bytes.
func doServeAdmitFunc(w http.ResponseWriter, r *http.Request, admit admitFunc, clientMgr ClientManagerInterface) ([]byte, error) {
	// Step 1: Request validation. Only handle POST requests with a body and json content type.

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("Invalid method %q, only POST requests are allowed", r.Method)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Could not read request body: %v", err)
	}

	if contentType := r.Header.Get(ContentType); contentType != JsonContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Unsupported content type %q, only %q is supported", contentType, JsonContentType)
	}

	// Step 2: Parse the AdmissionReview request.

	var admissionReviewReq v1beta1.AdmissionReview

	_, _, err = universalDeserializer.Decode(body, nil, &admissionReviewReq)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Could not deserialize request: %v", err)
	}
	if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("Malformed admission review request: request body is nil")
	}

	// Step 3: Construct the AdmissionReview response.

	// Apply the admit() function only for non-Kubernetes namespaces. For objects in Kubernetes namespaces, return
	// an empty set of patch operations.
	if isKubeNamespace(admissionReviewReq.Request.Namespace) {
		return allowedResponse(admissionReviewReq.Request.UID, nil), nil
	}

	var patchOps []patchOperation

	patchOps, err = admit(admissionReviewReq.Request, clientMgr)
	if err != nil {
		return errorResponse(admissionReviewReq.Request.UID, err), nil
	}

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("Could not marshal JSON patch: %v", err)
	}

	return allowedResponse(admissionReviewReq.Request.UID, patchBytes), nil
}

// serveAdmitFunc is a wrapper around doServeAdmitFunc that adds error handling and logging.
func serveAdmitFunc(w http.ResponseWriter, r *http.Request, admit admitFunc, clientMgr ClientManagerInterface) {
	log.Print("Handling webhook request ...")

	var writeErr error
	if bytes, err := doServeAdmitFunc(w, r, admit, clientMgr); err != nil {
		log.Printf("Error handling webhook request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr = w.Write([]byte(err.Error()))
	} else {
		log.Print("Webhook request handled successfully")
		_, writeErr = w.Write(bytes)
	}

	if writeErr != nil {
		log.Printf("Could not write response: %v", writeErr)
	}
}

// AdmitFuncHandler takes an admitFunc and wraps it into a http.Handler by means of calling serveAdmitFunc.
func AdmitFuncHandler(admit admitFunc, clientMgr ClientManagerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveAdmitFunc(w, r, admit, clientMgr)
	})
}

func allowedResponse(uid types.UID, patchBytes []byte) []byte {
	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID: uid,
		},
	}
	admissionReviewResponse.Response.Allowed = true
	admissionReviewResponse.Response.Patch = patchBytes

	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		return errorResponse(uid, err)
	}
	return bytes
}

func errorResponse(uid types.UID, err error) []byte {
	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID: uid,
		},
	}
	admissionReviewResponse.Response.Allowed = false
	admissionReviewResponse.Response.Result = &metav1.Status{
		Message: err.Error(),
	}
	bytes, _ := json.Marshal(&admissionReviewResponse)
	return bytes
}
