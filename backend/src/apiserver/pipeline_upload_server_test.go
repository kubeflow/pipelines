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
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/apiserver/storage"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	defaultUUID = "123e4567-e89b-12d3-a456-426655440000"
)

type FakeBadPipelineStore struct{}

func (s *FakeBadPipelineStore) GetPipelineWithStatus(id string, status model.PipelineStatus) (*model.Pipeline, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}

func (s *FakeBadPipelineStore) ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Pipeline, string, error) {
	return nil, "", util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}

func (s *FakeBadPipelineStore) GetPipeline(pipelineId string) (*model.Pipeline, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}

func (s *FakeBadPipelineStore) CreatePipeline(*model.Pipeline) (*model.Pipeline, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}
func (s *FakeBadPipelineStore) DeletePipeline(pipelineId string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}

func (s *FakeBadPipelineStore) UpdatePipelineStatus(string, model.PipelineStatus) error {
	return util.NewInternalServerError(errors.New("Error"), "bad pipeline store")
}

func initResourceManager(
	ps storage.PipelineStoreInterface,
	fm storage.ObjectStoreInterface) *resource.ResourceManager {
	clientManager := &ClientManager{
		pipelineStore: ps,
		objectStore:   fm,
		time:          util.NewFakeTimeForEpoch(),
		uuid:          util.NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)}
	return resource.NewResourceManager(clientManager)
}

func TestUploadPipeline(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	fm := storage.NewFakeObjectStore()

	server := PipelineUploadServer{resourceManager: initResourceManager(store.PipelineStore(), fm)}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	template, err := fm.GetFile(storage.JobFolder, resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.NotNil(t, template)

	// Verify metadata in db
	pkgsExpect := []model.Pipeline{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "hello-world",
			Parameters:     "[]",
			Status:         model.PipelineReady}}
	pkg, str, err := store.PipelineStore().ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "UUID" /*sortByFieldName*/, false /*isDesc*/)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)
}

func TestUploadPipeline_SpecifyFileName(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	fm := storage.NewFakeObjectStore()

	server := PipelineUploadServer{resourceManager: initResourceManager(store.PipelineStore(), fm)}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	encodedName := base64.StdEncoding.EncodeToString([]byte("foobar"))
	req, _ := http.NewRequest("POST", fmt.Sprintf("/apis/v1alpha2/pipelines/upload?name=%s", encodedName), bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	template, err := fm.GetFile(storage.JobFolder, resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.NotNil(t, template)

	// Verify metadata in db
	pkgsExpect := []model.Pipeline{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "foobar",
			Parameters:     "[]",
			Status:         model.PipelineReady}}
	pkg, str, err := store.PipelineStore().ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "UUID" /*sortByFieldName*/, false /*isDesc*/)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)
}

func TestUploadPipeline_FileNameTooLong(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	fm := storage.NewFakeObjectStore()

	server := PipelineUploadServer{resourceManager: initResourceManager(store.PipelineStore(), fm)}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	encodedName := base64.StdEncoding.EncodeToString([]byte(
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name"))
	req, _ := http.NewRequest("POST", fmt.Sprintf("/apis/v1alpha2/pipelines/upload?name=%s", encodedName), bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "File name too long")
}

func TestUploadPipeline_GetFormFileError(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	server := PipelineUploadServer{resourceManager: initResourceManager(store.PipelineStore(), storage.NewFakeObjectStore())}
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/pipeline/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Failed to read pipeline")
}

func TestUploadPipeline_CreatePipelineError(t *testing.T) {
	server := PipelineUploadServer{resourceManager: initResourceManager(&FakeBadPipelineStore{}, storage.NewFakeObjectStore())}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 500, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "bad pipeline store")
}
