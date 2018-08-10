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

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) GetPackageWithStatus(id string, status model.PackageStatus) (*model.Package, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) ListPackages(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Package, string, error) {
	return nil, "", util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) GetPackage(packageId string) (*model.Package, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) CreatePackage(*model.Package) (*model.Package, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad package store")
}
func (s *FakeBadPackageStore) DeletePackage(packageId string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) UpdatePackageStatus(string, model.PackageStatus) error {
	return util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func initResourceManager(
	ps storage.PackageStoreInterface,
	fm storage.ObjectStoreInterface) *resource.ResourceManager {
	clientManager := &ClientManager{
		packageStore: ps,
		objectStore:  fm,
		time:         util.NewFakeTimeForEpoch(),
		uuid:         util.NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)}
	return resource.NewResourceManager(clientManager)
}

func TestUploadPackage(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	fm := storage.NewFakeObjectStore()

	server := PackageUploadServer{resourceManager: initResourceManager(store.PackageStore(), fm)}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	template, err := fm.GetFile(storage.PackageFolder, resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.NotNil(t, template)

	// Verify metadata in db
	pkgsExpect := []model.Package{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "hello-world",
			Parameters:     "[]",
			Status:         model.PackageReady}}
	pkg, str, err := store.PackageStore().ListPackages("" /*pageToken*/, 2 /*pageSize*/, "UUID" /*sortByFieldName*/, false /*isDesc*/)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)
}

func TestUploadPackage_GetFormFileError(t *testing.T) {
	store := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	server := PackageUploadServer{resourceManager: initResourceManager(store.PackageStore(), storage.NewFakeObjectStore())}
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Failed to read package")
}

func TestUploadPackage_CreatePackageError(t *testing.T) {
	server := PackageUploadServer{resourceManager: initResourceManager(&FakeBadPackageStore{}, storage.NewFakeObjectStore())}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha2/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 500, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "bad package store")
}
