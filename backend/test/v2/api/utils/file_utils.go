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
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

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
func ReadYamlFile(pipelineFilePath string) interface{} {
	pipelineSpecs, err := os.ReadFile(pipelineFilePath)
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
			logger.Log("Failed to decode YAML: %s, due to %v", pipelineFilePath, err)
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
