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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"

	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// GetProjectRoot Get project root directory
func GetProjectRoot() string {
	dirFiles := make([]string, 0)
	dir, _ := os.Getwd()
	for !slices.Contains(dirFiles, "backend") && !slices.Contains(dirFiles, "data") {
		dirFiles = make([]string, 0)
		dir = filepath.Join(dir, "..")
		files, err := os.ReadDir(dir)
		if err != nil {
			logger.Log("Failed to read directory '%s', due to %s", err.Error())
		}
		for _, file := range files {
			dirFiles = append(dirFiles, file.Name())
		}
	}
	return dir
}

// GetProjectDataDir Get the directory location of the data directory
func GetProjectDataDir() string {
	projectRootDir := GetProjectRoot()
	return filepath.Join(projectRootDir, "data")
}

// GetPipelineFilesDir Get the directory location of the main list of pipeline files
func GetPipelineFilesDir() string {
	projectRootDir := GetProjectRoot()
	return filepath.Join(projectRootDir, "data", "pipeline_files")
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

// ReadYamlFile - Read a YAML file and unmarshall it into a map
func ReadYamlFile(filePath string) interface{} {
	pipelineSpecs, err := os.ReadFile(filePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	decoder := yaml.NewDecoder(bytes.NewReader(pipelineSpecs))
	var finalYamlData map[string]interface{}
	for {
		var yamlData map[string]interface{}
		err := decoder.Decode(&yamlData)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Log("Failed to decode YAML: %s, due to %v", filePath, err)
		}
		_, exists := yamlData["platforms"]
		if exists {
			yamlDataToReturn := make(map[string]interface{})
			yamlDataToReturn["pipeline_spec"] = finalYamlData
			yamlDataToReturn["platform_spec"] = yamlData
			return yamlDataToReturn
		} else {
			finalYamlData = yamlData
		}
	}
	return finalYamlData
}

func PipelineSpecFromFile(pipelineFilesRootDir string, pipelineDir string, pipelineFileName string) map[string]interface{} {
	pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFileName)
	logger.Log("Unmarshalling %s spec file", pipelineSpecFilePath)
	var unmarshalledPipelineSpec map[string]interface{}
	switch filepath.Ext(pipelineFileName) {
	case ".yaml":
		{
			unmarshalledPipelineSpec = ReadYamlFile(pipelineSpecFilePath).(map[string]interface{})
		}
	case ".json":
		{
			specFromFile, err := os.ReadFile(pipelineSpecFilePath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			err = json.Unmarshal(specFromFile, &unmarshalledPipelineSpec)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	default:
		{
			logger.Log("Unknown pipeline file format, supported format: yaml, json")
			return map[string]interface{}{}
		}
	}
	return unmarshalledPipelineSpec
}

func DeserializeSpecs(specs map[string]interface{}) (*pipelinespec.PipelineSpec, *pipelinespec.PlatformSpec) {
	logger.Log("Unmarshalling %v specs to pipeline specs and platform specs", specs)
	var unmarshalledPipelineSpec pipelinespec.PipelineSpec
	var unmarshalledPlatformSpec pipelinespec.PlatformSpec

	marshalledPipelineSpec, marshallPipelineSpecErr := yaml.Marshal(specs["pipeline_spec"])
	gomega.Expect(marshallPipelineSpecErr).NotTo(gomega.HaveOccurred(), "Failed to marshall pipeline specs")
	unmarshalledPipelineSpecErr := yaml.Unmarshal(marshalledPipelineSpec, &unmarshalledPipelineSpec)
	gomega.Expect(unmarshalledPipelineSpecErr).NotTo(gomega.HaveOccurred(), "Failed to unmarshall pipeline specs")
	marshalledPlatformSpecs, marshalledPlatformSpecErr := yaml.Marshal(specs["platform_spec"])
	gomega.Expect(marshalledPlatformSpecErr).NotTo(gomega.HaveOccurred(), "Failed to marshall platform specs")
	unmarshalledPlatformSpecErr := yaml.Unmarshal(marshalledPlatformSpecs, &unmarshalledPlatformSpec)
	gomega.Expect(unmarshalledPlatformSpecErr).NotTo(gomega.HaveOccurred(), "Failed to unmarshall platform specs")
	return &unmarshalledPipelineSpec, &unmarshalledPlatformSpec
}

func CreateFile(filePath string, fileContents [][]byte) *os.File {
	file, err := os.Create(filePath)
	if err != nil {
		logger.Log("Failed to create file &s due to %s", filePath, err.Error())
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

func CreateTempFile(fileContents [][]byte) *os.File {
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
	for _, content := range fileContents {
		_, err = tmpFile.Write(content)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to write contents to a temporary file")
	}
	return tmpFile
}
