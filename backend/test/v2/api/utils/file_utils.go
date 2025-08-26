// Package test
// Copyright 2018-2023 The Kubeflow Authors
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
package test

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	v1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"

	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	"github.com/onsi/gomega"
)

// GetProjectRoot Get project root directory
func GetProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		logger.Log("Failed to get current directory, due to %s", err.Error())
	}
	return filepath.Join(dir, "..", "..", "..", "../")
}

// GetTestDataDir Get the directory location for all the test data
func GetTestDataDir() string {
	projectRootDir := GetProjectRoot()
	return filepath.Join(projectRootDir, "test_data")
}

// GetPipelineFilesDir Get the directory location of the main list of pipeline files
func GetPipelineFilesDir() string {
	return filepath.Join(GetTestDataDir(), "pipeline_files")
}

// GetListOfFileInADir - Get list of files in a dir (not nested)
func GetListOfFileInADir(directoryPath string) []string {
	var fileNames []string
	files, err := os.ReadDir(directoryPath)
	if err != nil {
		logger.Log("Could not fetch files in directory %s, due to: %s", directoryPath, err.Error())
	}

	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	return fileNames
}

func IsYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

func IsJSONFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".json")
}

func ProtoToBytes(objectToConvert proto.Message) []byte {
	bytesConfig, err := protojson.Marshal(objectToConvert)
	if err != nil {
		// this is unexpected, cannot convert proto message to JSON
		return nil
	}
	bytesConfigYaml, err := yaml.JSONToYAML(bytesConfig)
	if err != nil {
		// this is unexpected, cannot convert JSON to YAML
		return nil
	}
	return bytesConfigYaml
}

func ToBytes(objectToConvert any) []byte {
	objectInBytes, marshallingErr := yaml.Marshal(objectToConvert)
	gomega.Expect(marshallingErr).NotTo(gomega.HaveOccurred(), "Failed to marshal object to bytes")
	return objectInBytes
}

// ParseFileToSpecs - Read a file and unmarshall it into a template.V2Spec
func ParseFileToSpecs(pipelineFilePath string, cacheEnabled bool, defaultWorkspace *v1.PersistentVolumeClaimSpec) *template.V2Spec {

	specFromFile, err := os.OpenFile(pipelineFilePath, os.O_RDWR, 0644)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read pipeline file")
	defer specFromFile.Close()
	pipelineSpecBytes, pipelineUnmarshallError := server.ReadPipelineFile(pipelineFilePath, specFromFile, common.MaxFileLength)
	gomega.Expect(pipelineUnmarshallError).To(gomega.BeNil(), "Failed to read pipeline spec")
	specs, templateErr := template.NewV2SpecTemplate(pipelineSpecBytes, cacheEnabled, defaultWorkspace)
	gomega.Expect(templateErr).To(gomega.BeNil(), "Failed to parse spec bytes into a spec object")
	return specs
}
