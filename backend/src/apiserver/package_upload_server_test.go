package main

import (
	"bytes"
	"mime/multipart"
	"ml/backend/src/model"
	"ml/backend/src/resource"
	"ml/backend/src/storage"
	"ml/backend/src/util"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	defaultUUID = "123e4567-e89b-12d3-a456-426655440000"
)

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) ListPackages(pageToken string, pageSize int, sortByFieldName string) ([]model.Package, string, error) {
	return nil, "", util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) GetPackage(packageId uint32) (*model.Package, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) CreatePackage(*model.Package) (*model.Package, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad package store")
}
func (s *FakeBadPackageStore) DeletePackage(packageId uint32) error {
	return util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func (s *FakeBadPackageStore) UpdatePackageStatus(uint32, model.PackageStatus) error {
	return util.NewInternalServerError(errors.New("Error"), "bad package store")
}

func initResourceManager(
	ps storage.PackageStoreInterface,
	js storage.JobStoreInterface,
	pls storage.PipelineStoreInterface,
	fm storage.ObjectStoreInterface) *resource.ResourceManager {
	clientManager := &ClientManager{
		packageStore:  ps,
		jobStore:      js,
		pipelineStore: pls,
		objectStore:   fm,
		time:          util.NewFakeTimeForEpoch(),
		uuid:          util.NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)}
	return resource.NewResourceManager(clientManager)
}

func TestUploadPackage(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	fm := storage.NewFakeObjectStore()

	server := PackageUploadServer{resourceManager: initResourceManager(store.PackageStore(), nil, nil, fm)}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha1/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	template, err := fm.GetFile(storage.PackageFolder, "1")
	assert.Nil(t, err)
	assert.NotNil(t, template)

	// Verify metadata in db
	pkgsExpect := []model.Package{
		{
			ID:             1,
			CreatedAtInSec: 1,
			Name:           "hello-world",
			Parameters:     []model.Parameter{},
			Status:         model.PackageReady}}
	pkg, str, err := store.PackageStore().ListPackages("" /*pageToken*/, 2 /*pageSize*/, "id" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, pkg, pkgsExpect)
}

func TestUploadPackage_GetFormFileError(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	server := PackageUploadServer{resourceManager: initResourceManager(store.PackageStore(), nil, nil, storage.NewFakeObjectStore())}
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha1/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Failed to read package")
}

func TestUploadPackage_CreatePackageError(t *testing.T) {
	server := PackageUploadServer{resourceManager: initResourceManager(&FakeBadPackageStore{}, nil, nil, storage.NewFakeObjectStore())}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1alpha1/packages/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPackage)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 500, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "bad package store")
}
