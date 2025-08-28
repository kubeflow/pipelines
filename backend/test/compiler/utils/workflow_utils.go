// Package utils defines custom utility methods for compiled workflows
/*
Copyright 2018-2023 The Kubeflow Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	"github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

// LoadSpecsFromIR - Unmarshall Pipeline Spec IR into a tuple of (pipelinespec.PipelineJob, pipelinespec.SinglePlatformSpec)
func LoadSpecsFromIR(pipelineIRRootDir, pipelineIRDirectory, pipelineIRFileName string) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	pipelineSpecsFromFile := utils.PipelineSpecFromFile(pipelineIRRootDir, pipelineIRDirectory, pipelineIRFileName)
	var singlePlatformSpec *pipelinespec.SinglePlatformSpec = nil
	pipelineSpecsYaml := make(map[string]interface{})
	if _, platformSpecExists := pipelineSpecsFromFile["platform_spec"]; platformSpecExists {
		pipelineSpecsYaml["pipelineSpec"] = pipelineSpecsFromFile["pipeline_spec"]
		platformSpecBytes, platformMarshallingError := json.Marshal(pipelineSpecsFromFile["platform_spec"])
		gomega.Expect(platformMarshallingError).NotTo(gomega.HaveOccurred(), "Failed to marshall platform spec map")
		platformSpecs := &pipelinespec.PlatformSpec{}
		err := protojson.Unmarshal(platformSpecBytes, platformSpecs)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Failed to unmarshall platform spec %s", string(platformSpecBytes)))
		for _, spec := range platformSpecs.Platforms {
			singlePlatformSpec = spec
		}
	} else {
		pipelineSpecsYaml["pipelineSpec"] = pipelineSpecsFromFile
	}
	pipelineSpecBytes, marshallingError := json.Marshal(pipelineSpecsYaml)
	gomega.Expect(marshallingError).NotTo(gomega.HaveOccurred(), "Failed to marshall pipeline spec map")
	pipelineSpecs := &pipelinespec.PipelineJob{}
	err := protojson.Unmarshal(pipelineSpecBytes, pipelineSpecs)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Failed to unmarshal pipeline spec\n %s", string(pipelineSpecBytes)))
	return pipelineSpecs, singlePlatformSpec
}

// GetCompiledArgoWorkflow - Compile pipeline and platform specs into a workflow and return an instance of v1alpha1.Workflow
func GetCompiledArgoWorkflow(pipelineSpecs *pipelinespec.PipelineJob, platformSpec *pipelinespec.SinglePlatformSpec, compilerOptions *argocompiler.Options) *v1alpha1.Workflow {
	logger.Log("Compiling Argo Workflow for provided pipeline job and platform spec")
	compiledWorflow, err := argocompiler.Compile(pipelineSpecs, platformSpec, compilerOptions)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to compile Argo workflow")
	return compiledWorflow
}

// UnmarshallWorkflowYAML - Unmarshall compiler workflow YAML into a v1alpha1.Workflow object
func UnmarshallWorkflowYAML(filePath string) *v1alpha1.Workflow {
	logger.Log("Unmarshalling Expected Workflow YAML")
	workflowFromFile := utils.ReadYamlFile(filePath)
	workflowFromFileBytes, marshallingError := json.Marshal(workflowFromFile)
	gomega.Expect(marshallingError).NotTo(gomega.HaveOccurred(), "Failed to marshall workflow map")
	workflow := v1alpha1.Workflow{}
	err := yaml.Unmarshal(workflowFromFileBytes, &workflow)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to unmarshall workflow map")
	logger.Log("Unmarshalled Expected Workflow YAML")
	return &workflow
}

// CreateCompiledWorkflowFile - Marshall v1alpha1.Workflow into a yaml file and save the file to the path provided as `compiledWorkflowFilePath`
func CreateCompiledWorkflowFile(compiledWorflow *v1alpha1.Workflow, compiledWorkflowFilePath string) *os.File {
	fileContents, err := yaml.Marshal(compiledWorflow)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return utils.CreateFile(compiledWorkflowFilePath, [][]byte{fileContents})
}
