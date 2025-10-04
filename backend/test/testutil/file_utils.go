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

// Package testutil provides helpers for working with test data files.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	testconstants "github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

// GetProjectRoot Get project root directory
// GetProjectRoot Get project root directory
func GetProjectRoot() string {
	dirFiles := make([]string, 0)
	dir, _ := os.Getwd()
	for !slices.Contains(dirFiles, testconstants.ParentDirectory) && !slices.Contains(dirFiles, testconstants.TestDataDir) {
		dirFiles = make([]string, 0)
		dir = filepath.Join(dir, "..")
		files, err := os.ReadDir(dir)
		if err != nil {
			logger.Log("Failed to read directory, due to %s", err.Error())
		}
		for _, file := range files {
			dirFiles = append(dirFiles, file.Name())
		}
	}
	return dir
}

// GetTestDataDir Get the directory location for all the test data
func GetTestDataDir() string {
	return filepath.Join(GetProjectRoot(), testconstants.TestDataDir)
}

// GetPipelineFilesDir Get the directory location of the main list of pipeline files
func GetPipelineFilesDir() string {
	return filepath.Join(GetTestDataDir(), testconstants.PipelineFilesDir)
}

// GetValidPipelineFilesDir Get the directory location of the main list of pipeline files
func GetValidPipelineFilesDir() string {
	return filepath.Join(GetPipelineFilesDir(), testconstants.ValidPipelineFilesDir)
}

// GetCompiledWorkflowsFilesDir Get the directory location of the main list of pipeline files
func GetCompiledWorkflowsFilesDir() string {
	return filepath.Join(GetTestDataDir(), testconstants.CompiledPipelineFilesDir)
}

// GetListOfFilesInADir - Get list of files in a dir (not nested)
func GetListOfFilesInADir(directoryPath string) []string {
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

// GetListOfAllFilesInDir - Iterator over a directory and even nested directories and fetch the list of files
func GetListOfAllFilesInDir(directoryPath string) []string {
	var filePaths []string
	err := filepath.Walk(directoryPath, func(path string, d os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}

		// Check if it's a regular file (not a directory)
		if !d.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the directory tree: %v\n", err)
	}
	return filePaths
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
func ParseFileToSpecs(pipelineFilePath string, cacheDisabled bool, defaultWorkspace *v1.PersistentVolumeClaimSpec) *template.V2Spec {

	specFromFile, err := os.OpenFile(pipelineFilePath, os.O_RDWR, 0644)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read pipeline file")
	defer specFromFile.Close()
	pipelineSpecBytes, pipelineUnmarshallError := server.ReadPipelineFile(pipelineFilePath, specFromFile, common.MaxFileLength)
	gomega.Expect(pipelineUnmarshallError).To(gomega.BeNil(), "Failed to read pipeline spec")
	specs, templateErr := template.NewV2SpecTemplate(pipelineSpecBytes, cacheDisabled, defaultWorkspace)
	gomega.Expect(templateErr).To(gomega.BeNil(), "Failed to parse spec bytes into a spec object")
	return specs
}

func CreateFile(filePath string, fileContents [][]byte) *os.File {
	file, err := os.Create(filePath)
	if err != nil {
		logger.Log("Failed to create file %s due to %s", filePath, err.Error())
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Log("Failed to close file: %s", err.Error())
		}
	}(file)
	for _, content := range fileContents {
		_, err = file.Write(content)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to write contents to a file")
	}
	return file
}

func CreateTempFile(fileContents []byte) *os.File {
	tmpFile, err := os.CreateTemp("", "pipeline-*.yaml")
	if err != nil {
		logger.Log("Failed to create temporary file: %s", err.Error())
	}
	defer func(tmpFile *os.File) {
		err := tmpFile.Close()
		if err != nil {
			logger.Log("Failed to close temporary file: %s", err.Error())
		}
	}(tmpFile)
	_, _ = tmpFile.Write(fileContents)
	return tmpFile
}
