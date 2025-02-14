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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/assert"
)

func TestLoadFile(t *testing.T) {
	file := "12345"
	bytes, err := loadFile(strings.NewReader(file), 5)
	assert.Nil(t, err)
	assert.Equal(t, []byte(file), bytes)
}

func TestLoadFile_ExceedSizeLimit(t *testing.T) {
	file := "12345"
	_, err := loadFile(strings.NewReader(file), 4)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "File size too large")
}

func TestLoadFile_LargeDoc(t *testing.T) {
	bytes, _ := os.ReadFile("test/xgboost_sample_pipeline.yaml")
	file := string(bytes)
	readBytes, err := loadFile(strings.NewReader(file), common.MaxFileLength)
	assert.Nil(t, err)
	assert.Equal(t, bytes, readBytes)
}

func TestDecompressPipelineTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := DecompressPipelineTarball(tarballByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineTarball_MalformattedTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/malformatted_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineTarball_NonYamlTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/non_yaml_tarball/non_yaml_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the tarball")
}

func TestDecompressPipelineTarball_EmptyTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/empty_tarball/empty.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := DecompressPipelineZip(zipByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineZip_MalformattedZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformatted_zip.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_MalformedZip2(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformed_zip2.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_NonYamlZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/non_yaml_zip/non_yaml_file.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the zip")
}

func TestDecompressPipelineZip_EmptyZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/empty_tarball/empty.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestReadPipelineFile_YAML(t *testing.T) {
	file, _ := os.Open("test/arguments-parameters.yaml")
	fileBytes, err := ReadPipelineFile("arguments-parameters.yaml", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_JSON(t *testing.T) {
	file, _ := os.Open("test/v2-hello-world.json")
	fileBytes, err := ReadPipelineFile("v2-hello-world.json", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := ioutil.ReadFile("test/v2-hello-world.json")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_Zip(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.zip", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Zip_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.pipeline", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_MultifileZip(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.zip")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.tar.gz", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.pipeline", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_MultifileTarball(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.tar.gz")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_UnknownFileFormat(t *testing.T) {
	file, _ := os.Open("test/unknown_format.foo")
	_, err := ReadPipelineFile("unknown_format.foo", file, common.MaxFileLength)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected pipeline file format")
}
