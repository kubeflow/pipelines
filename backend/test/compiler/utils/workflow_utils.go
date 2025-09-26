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
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"sigs.k8s.io/yaml"
	"slices"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/test_utils"
	utils "github.com/kubeflow/pipelines/backend/test/test_utils"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

// LoadPipelineSpecsFromIR - Unmarshall Pipeline Spec IR into a tuple of (pipelinespec.PipelineJob, pipelinespec.SinglePlatformSpec)
func LoadPipelineSpecsFromIR(pipelineIRFilePath string, cacheDisabled bool, defaultWorkspace *v1.PersistentVolumeClaimSpec) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	pipelineSpecsFromFile := utils.ParseFileToSpecs(pipelineIRFilePath, cacheDisabled, defaultWorkspace)
	platformSpec := pipelineSpecsFromFile.PlatformSpec()
	var singlePlatformSpec *pipelinespec.SinglePlatformSpec = nil
	if platformSpec != nil {
		singlePlatformSpec = platformSpec.Platforms["kubernetes"]
	}
	pipelineSpecMap := make(map[string]interface{})
	pipelineSpecBytes, marshallingError := protojson.Marshal(pipelineSpecsFromFile.PipelineSpec())
	gomega.Expect(marshallingError).NotTo(gomega.HaveOccurred(), "Failed to marshall pipeline spec")
	err := json.Unmarshal(pipelineSpecBytes, &pipelineSpecMap)
	pipelineSpecMapNew := make(map[string]interface{})
	pipelineSpecMapNew["pipelineSpec"] = pipelineSpecMap
	pipelineSpecBytes, marshallingError = json.Marshal(pipelineSpecMapNew)
	gomega.Expect(marshallingError).NotTo(gomega.HaveOccurred(), "Failed to marshall pipeline spec map")
	pipelineJob := &pipelinespec.PipelineJob{}
	err = protojson.Unmarshal(pipelineSpecBytes, pipelineJob)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Failed to unmarshal pipeline spec\n %s", string(pipelineSpecBytes)))
	return pipelineJob, singlePlatformSpec
}

// GetCompiledArgoWorkflow - Compile pipeline and platform specs into a workflow and return an instance of v1alpha1.Workflow
func GetCompiledArgoWorkflow(pipelineSpecs *pipelinespec.PipelineJob, platformSpec *pipelinespec.SinglePlatformSpec, compilerOptions *argocompiler.Options) *v1alpha1.Workflow {
	ginkgo.GinkgoHelper()
	logger.Log("Compiling Argo Workflow for provided pipeline job and platform spec")
	compiledWorflow, err := argocompiler.Compile(pipelineSpecs, platformSpec, compilerOptions)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to compile Argo workflow")
	return compiledWorflow
}

// UnmarshallWorkflowYAML - Unmarshall compiler workflow YAML into a v1alpha1.Workflow object
func UnmarshallWorkflowYAML(filePath string) *v1alpha1.Workflow {
	ginkgo.GinkgoHelper()
	logger.Log("Unmarshalling Expected Workflow YAML")
	workflowFromFileBytes, err := os.ReadFile(filePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read workflow yaml file")
	workflow := v1alpha1.Workflow{}
	err = yaml.Unmarshal(workflowFromFileBytes, &workflow)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to unmarshall workflow")
	logger.Log("Unmarshalled Expected Workflow YAML")
	return &workflow
}

// CreateCompiledWorkflowFile - Marshall v1alpha1.Workflow into a yaml file and save the file to the path provided as `compiledWorkflowFilePath`
func CreateCompiledWorkflowFile(compiledWorflow *v1alpha1.Workflow, compiledWorkflowFilePath string) *os.File {
	ginkgo.GinkgoHelper()
	fileContents, err := yaml.Marshal(compiledWorflow)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return test_utils.CreateFile(compiledWorkflowFilePath, [][]byte{fileContents})
}

// ConfigureCacheSettings - Add/Remove cache_disabled args in the workflow
func ConfigureCacheSettings(workflow *v1alpha1.Workflow, remove bool) *v1alpha1.Workflow {
	cacheDisabledArg := "--cache_disabled"
	configuredWorkflow := workflow.DeepCopy()
	for _, template := range configuredWorkflow.Spec.Templates {
		if template.Container != nil {
			if len(template.Container.Args) > 0 {
				if remove && slices.Contains(template.Container.Args, cacheDisabledArg) {
					containerArgs := make([]string, len(template.Container.Args)-1)
					for index, arg := range template.Container.Args {
						if arg == cacheDisabledArg {
							containerArgs = append(template.Container.Args[:index], template.Container.Args[index+1:]...)
							break
						}
					}
					template.Container.Args = containerArgs
				} else {
					if slices.Contains(template.Container.Args, "--run_id") {
						template.Container.Args = append(template.Container.Args, cacheDisabledArg)
					}
				}
			}
			for index, userContainer := range template.InitContainers {
				if remove {
					userArgs := make([]string, len(userContainer.Args)-1)
					for userArgsIndex, arg := range userContainer.Args {
						if arg == cacheDisabledArg {
							userArgs = append(userContainer.Args[:userArgsIndex], userContainer.Args[userArgsIndex+1:]...)
							break
						}
					}
					userContainer.Args = userArgs
				} else {
					if len(userContainer.Args) > 0 {
						userContainer.Args = append(userContainer.Args, cacheDisabledArg)
					} else {
						userContainer.Args = []string{cacheDisabledArg}
					}
				}
				template.InitContainers[index].Args = userContainer.Args
			}
		}
	}
	return configuredWorkflow
}
