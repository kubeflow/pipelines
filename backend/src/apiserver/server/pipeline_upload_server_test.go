// Copyright 2018 The Kubeflow Authors
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
	fakeDescription = "a_fake_description"
)

// TODO: move other upload pipeline tests into this table driven test
func TestUploadPipeline(t *testing.T) {
	// TODO(v2): when we add a field to distinguish between v1 and v2 template, verify it's in the response
	tt := []struct {
		name        string
		spec        []byte
		api_version string
	}{{
		name:        "upload argo workflow YAML",
		spec:        []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"),
		api_version: "v1beta1",
	}, {
		name:        "upload argo workflow YAML",
		spec:        []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"),
		api_version: "v2beta1",
	}, {
		name:        "upload pipeline v2 job in proto yaml",
		spec:        []byte(v2SpecHelloWorld),
		api_version: "v1beta1",
	}, {
		name:        "upload pipeline v2 job in proto yaml",
		spec:        []byte(v2SpecHelloWorld),
		api_version: "v2beta1",
	}}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			clientManager, server := setupClientManagerAndServer()
			bytesBuffer, writer := setupWriter("")
			setWriterWithBuffer("uploadfile", "hello-world.yaml", string(test.spec), writer)
			var response *httptest.ResponseRecorder
			if test.api_version == "v1beta1" {
				response = uploadPipeline("/apis/v1beta1/pipelines/upload",
					bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineV1)
			} else if test.api_version == "v2beta1" {
				response = uploadPipeline("/apis/v2beta1/pipelines/upload",
					bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
			}

			if response.Code != 200 {
				t.Fatalf("Upload response is not 200, message: %s", response.Body.String())
			}

			parsedResponse := struct {
				// v1 API only field
				ID string `json:"id"`
				// v2 API only field
				PipelineID string `json:"pipeline_id"`
				// v1 API and v2 API shared field
				CreatedAt string `json:"created_at"`
			}{}
			json.Unmarshal(response.Body.Bytes(), &parsedResponse)

			// Verify time format is RFC3339.
			assert.Equal(t, "1970-01-01T00:00:01Z", parsedResponse.CreatedAt)

			// Verify v1 API returns v1 object while v2 API returns v2 object.
			if test.api_version == "v1beta1" {
				assert.Equal(t, "123e4567-e89b-12d3-a456-426655440000", parsedResponse.ID)
				assert.Equal(t, "", parsedResponse.PipelineID)
			} else if test.api_version == "v2beta1" {
				assert.Equal(t, "", parsedResponse.ID)
				assert.Equal(t, "123e4567-e89b-12d3-a456-426655440000", parsedResponse.PipelineID)
			}

			// Verify stored in object store
			objStore := clientManager.ObjectStore()
			template, err := objStore.GetFile(objStore.GetPipelineKey(DefaultFakeUUID))
			assert.Nil(t, err)
			assert.NotNil(t, template)

			opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
			assert.Nil(t, err)

			// Verify metadata in db
			pkgsExpect := []*model.Pipeline{
				{
					UUID:           DefaultFakeUUID,
					CreatedAtInSec: 1,
					Name:           "hello-world.yaml",
					Status:         model.PipelineReady,
					Namespace:      "default",
				},
			}
			pkgsExpect2 := []*model.PipelineVersion{
				{
					UUID:           DefaultFakeUUID,
					CreatedAtInSec: 2,
					Name:           "hello-world.yaml",
					Parameters:     "[]",
					Status:         model.PipelineVersionReady,
					PipelineId:     DefaultFakeUUID,
				},
			}

			pkg, totalSize, str, err := clientManager.PipelineStore().ListPipelines(&model.FilterContext{}, opts)
			assert.Nil(t, err)
			assert.Equal(t, str, "")
			assert.Equal(t, 1, totalSize)
			assert.Equal(t, pkgsExpect, pkg)

			opts2, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
			pkg2, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts2)
			assert.Nil(t, err)
			assert.Equal(t, str, "")
			assert.Equal(t, 1, totalSize)
			pkgsExpect2[0].PipelineSpec = pkg2[0].PipelineSpec
			assert.Equal(t, pkgsExpect2, pkg2)

			// Upload a new version under this pipeline

			// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
			server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
			response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?name="+fakeVersionName+"&pipelineid="+DefaultFakeUUID+"&description="+fakeDescription,
				bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
			assert.Equal(t, 200, response.Code)
			assert.Contains(t, response.Body.String(), `"created_at":"1970-01-01T00:00:03Z"`)

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
					UUID:           DefaultFakeUUID,
					CreatedAtInSec: 2,
					Name:           "hello-world.yaml",
					Parameters:     "[]",
					Status:         model.PipelineVersionReady,
					PipelineId:     DefaultFakeUUID,
					PipelineSpec:   string(test.spec),
				},
				{
					UUID:           fakeVersionUUID,
					CreatedAtInSec: 3,
					Name:           fakeVersionName,
					Description:    fakeDescription,
					Parameters:     "[]",
					Status:         model.PipelineVersionReady,
					PipelineId:     DefaultFakeUUID,
					PipelineSpec:   string(test.spec),
				},
			}
			// Expect 2 versions, one is created by default when creating pipeline and the other is what we manually created
			versions, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts)
			assert.Nil(t, err)
			assert.Equal(t, str, "")
			assert.Equal(t, 2, totalSize)
			versionsExpect[0].PipelineSpec = versions[0].PipelineSpec
			versionsExpect[1].PipelineSpec = versions[1].PipelineSpec
			assert.Equal(t, versionsExpect, versions)
		})
	}
}

func TestUploadPipeline_Tarball(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterFromFile("uploadfile", "arguments.tar.gz", "test/arguments_tarball/arguments.tar.gz", writer)
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	// Verify time format is RFC3339
	assert.Contains(t, response.Body.String(), `"created_at":"1970-01-01T00:00:01Z"`)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "arguments.tar.gz",
			Status:         model.PipelineReady,
			Namespace:      "default",
		},
	}
	pkg, totalSize, str, err := clientManager.PipelineStore().ListPipelines(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pkgsExpect, pkg)

	pkgsExpect2 := []*model.PipelineVersion{
		{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 2,
			Name:           "arguments.tar.gz",
			Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
			PipelineSpec:   "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"generateName\":\"arguments-parameters-\",\"creationTimestamp\":null},\"spec\":{\"templates\":[{\"name\":\"whalesay\",\"inputs\":{\"parameters\":[{\"name\":\"param1\"},{\"name\":\"param2\"}]},\"outputs\":{},\"metadata\":{},\"container\":{\"name\":\"\",\"image\":\"docker/whalesay:latest\",\"command\":[\"cowsay\"],\"args\":[\"{{inputs.parameters.param1}}-{{inputs.parameters.param2}}\"],\"resources\":{}}}],\"entrypoint\":\"whalesay\",\"arguments\":{\"parameters\":[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
	}
	opts2, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	pkg2, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pkgsExpect2, pkg2)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	bytesBuffer, writer = setupWriter("")
	setWriterFromFile("uploadfile", "arguments-version.tar.gz", "test/arguments_tarball/arguments-version.tar.gz", writer)
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?pipelineid="+DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
	assert.Equal(t, 200, response.Code)
	assert.Contains(t, response.Body.String(), `"created_at":"1970-01-01T00:00:03Z"`)

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
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 2,
			Name:           "arguments.tar.gz",
			Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
			PipelineSpec:   "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"generateName\":\"arguments-parameters-\",\"creationTimestamp\":null},\"spec\":{\"templates\":[{\"name\":\"whalesay\",\"inputs\":{\"parameters\":[{\"name\":\"param1\"},{\"name\":\"param2\"}]},\"outputs\":{},\"metadata\":{},\"container\":{\"name\":\"\",\"image\":\"docker/whalesay:latest\",\"command\":[\"cowsay\"],\"args\":[\"{{inputs.parameters.param1}}-{{inputs.parameters.param2}}\"],\"resources\":{}}}],\"entrypoint\":\"whalesay\",\"arguments\":{\"parameters\":[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
		{
			UUID:           fakeVersionUUID,
			CreatedAtInSec: 3,
			Name:           "arguments-version.tar.gz",
			Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
			PipelineSpec:   "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"generateName\":\"arguments-parameters-\",\"creationTimestamp\":null},\"spec\":{\"templates\":[{\"name\":\"whalesay\",\"inputs\":{\"parameters\":[{\"name\":\"param1\"},{\"name\":\"param2\"}]},\"outputs\":{},\"metadata\":{},\"container\":{\"name\":\"\",\"image\":\"docker/whalesay:latest\",\"command\":[\"cowsay\"],\"args\":[\"{{inputs.parameters.param1}}-{{inputs.parameters.param2}}\"],\"resources\":{}}}],\"entrypoint\":\"whalesay\",\"arguments\":{\"parameters\":[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
	}
	// Expect 2 versions, one is created by default when creating pipeline and the other is what we manually created
	versions, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, versionsExpect, versions)
}

func TestUploadPipeline_GetFormFileError(t *testing.T) {
	_, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("I am invalid file")
	writer.CreateFormFile("uploadfile", "hello-world.yaml")
	writer.Close()
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 400, response.Code)
	assert.Contains(t, response.Body.String(), "Failed to read pipeline")
}

func TestUploadPipeline_SpecifyFileName(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterWithBuffer("uploadfile", "hello-world.yaml", "apiVersion: argoproj.io/v1alpha1\nkind: Workflow", writer)
	response := uploadPipeline(fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s", url.PathEscape("foo bar")),
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "foo bar",
			Status:         model.PipelineReady,
			Namespace:      "default",
		},
	}
	pkg, totalSize, str, err := clientManager.PipelineStore().ListPipelines(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)

	opts2, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pkgsExpect2 := []*model.PipelineVersion{
		{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 2,
			Name:           "foo bar",
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
			PipelineSpec:   "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"creationTimestamp\":null},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
	}
	pkg2, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect2, pkg2)
}

func TestUploadPipeline_FileNameTooLong(t *testing.T) {
	_, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterWithBuffer("uploadfile", "hello-world.yaml", "apiVersion: argoproj.io/v1alpha1\nkind: Workflow", writer)
	encodedName := url.PathEscape(
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	response := uploadPipeline(fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s", encodedName),
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 400, response.Code)
	assert.Contains(t, response.Body.String(), "is too long")
}

func TestUploadPipeline_SpecifyFileDescription(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterWithBuffer("uploadfile", "hello-world.yaml", "apiVersion: argoproj.io/v1alpha1\nkind: Workflow", writer)
	response := uploadPipeline(fmt.Sprintf("/apis/v1beta1/pipelines/upload?name=%s&description=%s", url.PathEscape("foo bar"),
		url.PathEscape("description of foo bar")),
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)
	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)

	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:           DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "foo bar",
			Status:         model.PipelineReady,
			Description:    "description of foo bar",
			Namespace:      "default",
		},
	}
	pkg, totalSize, str, err := clientManager.PipelineStore().ListPipelines(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect, pkg)

	opts2, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pkgsExpect2 := []*model.PipelineVersion{
		{
			UUID:           DefaultFakeUUID,
			Description:    "description of foo bar",
			CreatedAtInSec: 2,
			Name:           "foo bar",
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     DefaultFakeUUID,
			PipelineSpec:   "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"creationTimestamp\":null},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
	}
	pkg2, totalSize, str, err := clientManager.PipelineStore().ListPipelineVersions(DefaultFakeUUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, str, "")
	assert.Equal(t, pkgsExpect2, pkg2)
}

func TestUploadPipelineVersion_GetFromFileError(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterWithBuffer("uploadfile", "hello-world.yaml", "apiVersion: argoproj.io/v1alpha1\nkind: Workflow", writer)
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)
	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	bytesBuffer, writer = setupWriter("I am invalid file")
	writer.CreateFormFile("uploadfile", "hello-world.yaml")
	writer.Close()
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?name="+fakeVersionName+"&pipelineid="+DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
	assert.Equal(t, 400, response.Code)
	assert.Contains(t, response.Body.String(), "error parsing pipeline spec filename")
}

func TestUploadPipelineVersion_FileNameTooLong(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterWithBuffer("uploadfile", "hello-world.yaml", "apiVersion: argoproj.io/v1alpha1\nkind: Workflow", writer)
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	encodedName := url.PathEscape(
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?name="+encodedName+"&pipelineid="+DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 400, response.Code)
	assert.Contains(t, response.Body.String(), "is too long")
}

func TestDefaultNotUpdatedPipelineVersion(t *testing.T) {
	viper.Set(common.UpdatePipelineVersionByDefault, "false")
	defer viper.Set(common.UpdatePipelineVersionByDefault, "true")

	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterFromFile("uploadfile", "arguments.tar.gz", "test/arguments_tarball/arguments.tar.gz", writer)
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	pipelineVersion, err := clientManager.PipelineStore().GetPipelineVersion(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion.PipelineId, DefaultFakeUUID)

	// Upload a new version under this pipeline and check that the default version is not updated

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	bytesBuffer, writer = setupWriter("")
	setWriterFromFile("uploadfile", "arguments-version.tar.gz", "test/arguments_tarball/arguments.tar.gz", writer)
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?pipelineid="+DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
	assert.Equal(t, 200, response.Code)

	_, err = clientManager.PipelineStore().GetPipeline(DefaultFakeUUID)
	assert.Nil(t, err)
	// assert.Equal(t, pipeline.DefaultVersionId, DefaultFakeUUID)
	// assert.NotEqual(t, pipeline.DefaultVersionId, fakeVersionUUID)
}

func TestDefaultUpdatedPipelineVersion(t *testing.T) {
	clientManager, server := setupClientManagerAndServer()
	bytesBuffer, writer := setupWriter("")
	setWriterFromFile("uploadfile", "arguments.tar.gz", "test/arguments_tarball/arguments.tar.gz", writer)
	response := uploadPipeline("/apis/v1beta1/pipelines/upload",
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipeline)
	assert.Equal(t, 200, response.Code)

	pipelineVersion, err := clientManager.PipelineStore().GetPipelineVersion(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion.PipelineId, DefaultFakeUUID)

	// Upload a new version under this pipeline and check that the default version is not updated

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	bytesBuffer, writer = setupWriter("")
	setWriterFromFile("uploadfile", "arguments-version.tar.gz", "test/arguments_tarball/arguments-version.tar.gz", writer)
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?pipelineid="+DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
	assert.Equal(t, 200, response.Code)

	pipelineVersion2, err := clientManager.PipelineStore().GetLatestPipelineVersion(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion2.UUID, fakeVersionUUID)
}

func setWriterWithBuffer(fieldname string, filename string, buffer string, writer *multipart.Writer) {
	part, _ := writer.CreateFormFile(fieldname, filename)
	io.Copy(part, bytes.NewBufferString(buffer))
	writer.Close()
}

func setWriterFromFile(fieldname string, filename string, filepath string, writer *multipart.Writer) {
	part, _ := writer.CreateFormFile(fieldname, filename)
	fileReader, _ := os.Open(filepath)
	io.Copy(part, fileReader)
	writer.Close()
}

func setupWriter(text string) (*bytes.Buffer, *multipart.Writer) {
	bytesBuffer := &bytes.Buffer{}
	if text != "" {
		bytesBuffer.WriteString(text)
	}
	return bytesBuffer, multipart.NewWriter(bytesBuffer)
}

func setupClientManagerAndServer() (*resource.FakeClientManager, PipelineUploadServer) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	server := PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	return clientManager, server
}

func updateClientManager(clientManager *resource.FakeClientManager, uuid util.UUIDGeneratorInterface) PipelineUploadServer {
	clientManager.UpdateUUID(uuid)
	resourceManager := resource.NewResourceManager(clientManager, "default")
	server := PipelineUploadServer{resourceManager: resourceManager, options: &PipelineUploadServerOptions{CollectMetrics: false}}
	return server
}

func uploadPipeline(url string, body io.Reader, writer *multipart.Writer, uploadFunc func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(uploadFunc)
	handler.ServeHTTP(rr, req)
	return rr
}

var v2SpecHelloWorld = `components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        text:
          type: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - "--text"
        - "{{$.inputs.parameters['text']}}"
        command:
        - sh
        - "-ec"
        - |
          program_path=$(mktemp)
          printf "%s" "$0" > "$program_path"
          python3 -u "$program_path" "$@"
        - |
          def hello_world(text):
              print(text)
              return text

          import argparse
          _parser = argparse.ArgumentParser(prog='Hello world', description='')
          _parser.add_argument("--text", dest="text", type=str, required=True, default=argparse.SUPPRESS)
          _parsed_args = vars(_parser.parse_args())

          _outputs = hello_world(**_parsed_args)
        image: python:3.7
pipelineInfo:
  name: hello-world
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            text:
              componentInputParameter: text
        taskInfo:
          name: hello-world
  inputDefinitions:
    parameters:
      text:
        type: STRING
schemaVersion: 2.0.0
sdkVersion: kfp-1.6.5
`
