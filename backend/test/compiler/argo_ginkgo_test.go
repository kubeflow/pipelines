// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	matcher "github.com/kubeflow/pipelines/backend/test/compiler/matchers"
	workflow_utils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

// var update = flag.Bool("update", false, "update golden files")
var projectDataDir = utils.GetProjectDataDir()
var pipelineFilesRootDir = utils.GetPipelineFilesDir()
var pipelineDirectory = "valid"
var argoYAMLDir = filepath.Join(projectDataDir, "compiled-workflows")
var testReportDirectory = "reports"
var junitReportFilename = "compiler.xml"
var jsonReportFilename = "compiler.json"
var updateGoldenFiles = flag.Bool("updateCompiledFiles", false, "update golden/expected compiled workflow files")
var createMissingGoldenFiles = flag.Bool("createGoldenFiles", false, "create missing golden/expected compiled workflow files")

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing test logs")
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
	}
})

func TestCompilation(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.FailFast = false
	reporterConfig.GithubOutput = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "API Tests Suite", suiteConfig, reporterConfig)
}

var _ = BeforeEach(func() {
	logger.Log("Initializing proxy...")
	proxy.InitializeConfigWithEmptyForTests()
})

var _ = Describe("Verify Spec Compilation to Workflow >", Label("Positive", "WorkflowCompiler"), func() {
	pipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	testParams := []struct {
		compilerOptions argocompiler.Options
		envVars         map[string]string
	}{
		{
			compilerOptions: argocompiler.Options{CacheDisabled: true},
		},
		{
			compilerOptions: argocompiler.Options{CacheDisabled: false},
		},
		{
			compilerOptions: argocompiler.Options{CacheDisabled: false},
			envVars:         map[string]string{"PIPELINE_RUN_AS_USER": "1001", "PIPELINE_LOG_LEVEL": "3"},
		},
	}
	for _, param := range testParams {
		var testRunType interface{}
		if param.envVars == nil {
			testRunType = Serial
		} else {
			testRunType = Ordered
		}
		Context(fmt.Sprintf("Verify compiled workflow for a pipeline with compiler options '%v' and env vars %v >", param.compilerOptions, param.envVars), testRunType, func() {
			for _, pipelineSpecFileName := range pipelineFiles {
				compiledWorkflowFilePath := filepath.Join(argoYAMLDir, pipelineSpecFileName)
				// Set provided env variables
				for envVarName, envVarValue := range param.envVars {
					err := os.Setenv(envVarName, envVarValue)
					Expect(err).To(BeNil(), "Could not set env var %s", envVarName)
				}

				// Defer UnSetting the set env variables at the end of the test
				defer func() {
					for envVarName := range param.envVars {
						err := os.Unsetenv(envVarName)
						Expect(err).To(BeNil(), "Could not unset env var %s", envVarName)
					}
				}()
				It(fmt.Sprintf("When I compile %s pipeline spec, then the compiled yaml should be =%s", pipelineSpecFileName, compiledWorkflowFilePath), func() {
					pipelineSpecs, platformSpec := workflow_utils.LoadSpecsFromIR(pipelineFilesRootDir, pipelineDirectory, pipelineSpecFileName)
					compiledWorflow := workflow_utils.GetCompiledArgoWorkflow(pipelineSpecs, platformSpec, &argocompiler.Options{})
					if *updateGoldenFiles {
						logger.Log(fmt.Sprintf("Updating golden file %s", compiledWorkflowFilePath))
						_, err := os.Stat(compiledWorkflowFilePath)
						if err != nil {
							logger.Log("File %s does not exist, but if you want to create the missing workflow file, please set 'createGoldenFiles' flag to true", compiledWorkflowFilePath)
						} else {
							workflow_utils.CreateCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						}
					} else if *createMissingGoldenFiles {
						_, err := os.Stat(compiledWorkflowFilePath)
						if err == nil {
							logger.Log("Creating Compiled Workflow File '%s'", compiledWorkflowFilePath)
							workflow_utils.CreateCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						} else {
							logger.Log("Compiled Workflow File '%s' already exists", compiledWorkflowFilePath)
						}
					} else {
						expectedWorkflow := workflow_utils.UnmarshallWorkflowYAML(compiledWorkflowFilePath)
						matcher.CompareWorkflows(compiledWorflow, expectedWorkflow)
					}

				})
			}
		})
	}
})
