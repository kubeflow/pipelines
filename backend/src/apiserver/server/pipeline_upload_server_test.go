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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	fakeVersionUUID = "123e4567-e89b-12d3-a456-526655440000"
	fakeVersionName = "a_fake_version_name"
)

func TestUploadPipeline_YAML(t *testing.T) {
	clientManager, resourceManager, server, b, w, _ := setupCreateFromFile()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	// Verify time format is RFC3339.
	parsedResponse := struct {
		CreatedAt      string `json:"created_at"`
		DefaultVersion struct {
			CreatedAt string `json:"created_at"`
		} `json:"default_version"`
	}{}
	json.Unmarshal(rr.Body.Bytes(), &parsedResponse)
	assert.Equal(t, "1970-01-01T00:00:01Z", parsedResponse.CreatedAt)
	assert.Equal(t, "1970-01-01T00:00:01Z", parsedResponse.DefaultVersion.CreatedAt)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(resource.DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)

	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:             resource.DefaultFakeUUID,
			CreatedAtInSec:   1,
			Name:             "hello-world.yaml",
			Parameters:       "[]",
			Status:           model.PipelineReady,
			DefaultVersionId: resource.DefaultFakeUUID,
			DefaultVersion: &model.PipelineVersion{
				UUID:           resource.DefaultFakeUUID,
				CreatedAtInSec: 1,
				Name:           "hello-world.yaml",
				Parameters:     "[]",
				Status:         model.PipelineVersionReady,
				PipelineId:     resource.DefaultFakeUUID,
			}}}
	pkg, total_size, str, err := clientManager.PipelineStore().ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pkgsExpect, pkg)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?name="+fakeVersionName+"&pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"created_at":"1970-01-01T00:00:02Z"`)

	// Verify stored in object store
	objStore = clientManager.ObjectStore()
	template, err = objStore.GetFile(objStore.GetPipelineKey(fakeVersionUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)
	opts, err = list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	versionsExpect := []*model.PipelineVersion{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "hello-world.yaml",
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
		{
			UUID:           fakeVersionUUID,
			CreatedAtInSec: 2,
			Name:           fakeVersionName,
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
	}
	// Expect 2 versions, one is created by default when creating pipeline and the other is what we manually created
	versions, total_size, str, err := clientManager.PipelineStore().ListPipelineVersions(resource.DefaultFakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 2, total_size)
	assert.Equal(t, versionsExpect, versions)
}

func TestUploadPipeline_Tarball(t *testing.T) {
	clientManager, resourceManager, server, b, w, part, req, fileReader := setupCreateFromTarBall()
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	// Verify time format is RFC3339
	assert.Contains(t, rr.Body.String(), `"created_at":"1970-01-01T00:00:01Z"`)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(resource.DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:             resource.DefaultFakeUUID,
			CreatedAtInSec:   1,
			Name:             "arguments.tar.gz",
			Parameters:       "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:           model.PipelineReady,
			DefaultVersionId: resource.DefaultFakeUUID,
			DefaultVersion: &model.PipelineVersion{
				UUID:           resource.DefaultFakeUUID,
				CreatedAtInSec: 1,
				Name:           "arguments.tar.gz",
				Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
				Status:         model.PipelineVersionReady,
				PipelineId:     resource.DefaultFakeUUID,
			}}}
	pkg, total_size, str, err := clientManager.PipelineStore().ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pkgsExpect, pkg)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	b = &bytes.Buffer{}
	w = multipart.NewWriter(b)
	part, _ = w.CreateFormFile("uploadfile", "arguments-version.tar.gz")
	fileReader, _ = os.Open("test/arguments_tarball/arguments-version.tar.gz")
	io.Copy(part, fileReader)
	w.Close()
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"created_at":"1970-01-01T00:00:02Z"`)

	// Verify stored in object store
	objStore = clientManager.ObjectStore()
	template, err = objStore.GetFile(objStore.GetPipelineKey(fakeVersionUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)
	opts, err = list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	versionsExpect := []*model.PipelineVersion{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "arguments.tar.gz",
			Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
		{
			UUID:           fakeVersionUUID,
			CreatedAtInSec: 2,
			Name:           "arguments-version.tar.gz",
			Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
	}
	// Expect 2 versions, one is created by default when creating pipeline and the other is what we manually created
	versions, total_size, str, err := clientManager.PipelineStore().ListPipelineVersions(resource.DefaultFakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 2, total_size)
	assert.Equal(t, versionsExpect, versions)
}

func TestUploadPipeline_GetFormFileError(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	server := PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipeline/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Failed to read pipeline")
}

func TestUploadPipeline_SpecifyFileName(t *testing.T) {
	clientManager, _, server, b, w, _ := setupCreateFromFile()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s", url.PathEscape("foo bar")), bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(resource.DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:             resource.DefaultFakeUUID,
			CreatedAtInSec:   1,
			Name:             "foo bar",
			Parameters:       "[]",
			Status:           model.PipelineReady,
			DefaultVersionId: resource.DefaultFakeUUID,
			DefaultVersion: &model.PipelineVersion{
				UUID:           resource.DefaultFakeUUID,
				CreatedAtInSec: 1,
				Name:           "foo bar",
				Parameters:     "[]",
				Status:         model.PipelineVersionReady,
				PipelineId:     resource.DefaultFakeUUID,
			}}}
	pkg, total_size, str, err := clientManager.PipelineStore().ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)
}

func TestUploadPipeline_FileNameTooLong(t *testing.T) {
	_, _, server, b, w, _ := setupCreateFromFile()
	encodedName := url.PathEscape(
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	req, _ := http.NewRequest("POST", fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s", encodedName), bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Pipeline name too long")
}

func TestUploadPipeline_SpecifyFileDescription(t *testing.T) {
	clientManager, _, server, b, w, _ := setupCreateFromFile()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s&description=%s", url.PathEscape("foo bar"), url.PathEscape("description of foo bar")), bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(resource.DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:             resource.DefaultFakeUUID,
			CreatedAtInSec:   1,
			Name:             "foo bar",
			Parameters:       "[]",
			Status:           model.PipelineReady,
			DefaultVersionId: resource.DefaultFakeUUID,
			DefaultVersion: &model.PipelineVersion{
				UUID:           resource.DefaultFakeUUID,
				CreatedAtInSec: 1,
				Name:           "foo bar",
				Parameters:     "[]",
				Status:         model.PipelineVersionReady,
				PipelineId:     resource.DefaultFakeUUID,
			},
			Description: "description of foo bar",
		}}
	pkg, total_size, str, err := clientManager.PipelineStore().ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)
}

func TestUploadPipelineVersion_GetFromFileError(t *testing.T) {
	clientManager, resourceManager, server, b, w, _ := setupCreateFromFile()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	b = &bytes.Buffer{}
	b.WriteString("I am invalid file")
	w = multipart.NewWriter(b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?name="+fakeVersionName+"&pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Failed to read pipeline version")
}

func TestUploadPipelineVersion_FileNameTooLong(t *testing.T) {
	clientManager, resourceManager, server, b, w, _ := setupCreateFromFile()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	encodedName := url.PathEscape(
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?name="+encodedName+"&pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
	assert.Contains(t, string(rr.Body.Bytes()), "Pipeline name too long")
}

func TestDefaultNotUpdatedPipelineVersion(t *testing.T) {
	viper.Set(common.UpdatePipelineVersionByDefault, "false")
	defer viper.Set(common.UpdatePipelineVersionByDefault, "true")

	clientManager, resourceManager, server, b, w, part, req, fileReader := setupCreateFromTarBall()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)

	pipelineVersion, err := clientManager.PipelineStore().GetPipelineVersion(resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion.PipelineId, resource.DefaultFakeUUID)

	// Upload a new version under this pipeline and check that the default version is not updated

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	b = &bytes.Buffer{}
	w = multipart.NewWriter(b)
	part, _ = w.CreateFormFile("uploadfile", "arguments-version.tar.gz")
	fileReader, _ = os.Open("test/arguments_tarball/arguments-version.tar.gz")
	io.Copy(part, fileReader)
	w.Close()
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)

	pipeline, err := clientManager.PipelineStore().GetPipeline(resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, resource.DefaultFakeUUID)
	assert.NotEqual(t, pipeline.DefaultVersionId, fakeVersionUUID)
}

func TestDefaultUpdatedPipelineVersion(t *testing.T) {
	clientManager, resourceManager, server, b, w, part, req, fileReader := setupCreateFromTarBall()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)

	pipelineVersion, err := clientManager.PipelineStore().GetPipelineVersion(resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion.PipelineId, resource.DefaultFakeUUID)

	// Upload a new version under this pipeline and check that the default version is not updated

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	resourceManager = resource.NewResourceManager(clientManager)
	server = PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	b = &bytes.Buffer{}
	w = multipart.NewWriter(b)
	part, _ = w.CreateFormFile("uploadfile", "arguments-version.tar.gz")
	fileReader, _ = os.Open("test/arguments_tarball/arguments-version.tar.gz")
	io.Copy(part, fileReader)
	w.Close()
	req, _ = http.NewRequest("POST", "/apis/v1beta1/pipelines/upload_version?pipelineid="+resource.DefaultFakeUUID, bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(server.UploadPipelineVersion)
	handler.ServeHTTP(rr, req)

	pipeline, err := clientManager.PipelineStore().GetPipeline(resource.DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, fakeVersionUUID)
}

func setupCreateFromFile() (*resource.FakeClientManager, *resource.ResourceManager, PipelineUploadServer, *bytes.Buffer, *multipart.Writer, io.Writer) {
	clientManager, resourceManager, server, b, w := setupBase()
	part, _ := w.CreateFormFile("uploadfile", "hello-world.yaml")
	io.Copy(part, bytes.NewBufferString("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	w.Close()
	return clientManager, resourceManager, server, b, w, part
}

func setupCreateFromTarBall() (*resource.FakeClientManager, *resource.ResourceManager, PipelineUploadServer, *bytes.Buffer, *multipart.Writer, io.Writer, *http.Request, *os.File) {
	clientManager, resourceManager, server, b, w := setupBase()
	part, _ := w.CreateFormFile("uploadfile", "arguments.tar.gz")
	fileReader, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	io.Copy(part, fileReader)
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	return clientManager, resourceManager, server, b, w, part, req, fileReader
}

func setupBase() (*resource.FakeClientManager, *resource.ResourceManager, PipelineUploadServer, *bytes.Buffer, *multipart.Writer) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	return clientManager, resourceManager, server, b, w
}
