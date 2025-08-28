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
	"fmt"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/test_utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	matcher "github.com/kubeflow/pipelines/backend/test/compiler/matchers"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeEach(func() {
	logger.Log("Initializing proxy...")
	proxy.InitializeConfigWithEmptyForTests()
})

var _ = Describe("Verify Spec Compilation to Workflow >", Label("Positive", "WorkflowCompiler"), func() {
	pipelineFilePaths := test_utils.GetListOfAllFilesInDir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

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
		Context(fmt.Sprintf("Verify compiled workflow for a pipeline with compiler options cacheDisabled '%v' and env vars %v >", param.compilerOptions.CacheDisabled, param.envVars), testRunType, func() {
			for _, pipelineSpecFilePath := range pipelineFilePaths {
				pipelineSpecFileName := filepath.Base(pipelineSpecFilePath)
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
				It(fmt.Sprintf("When I compile %s pipeline spec, then the compiled yaml should be %s", pipelineSpecFileName, pipelineSpecFileName), func() {
					if strings.Contains(pipelineSpecFileName, "pipeline_with_placeholders") {
						Skip("Arguments.Parameter.Name and Arguments.Parameter.Value keeps on changing, so skipping until we figure it out")
					}
					test_utils.SkipTest(pipelineSpecFileName)
					pipelineSpecs, platformSpec := workflowutils.LoadSpecsFromIR(pipelineSpecFilePath, param.compilerOptions.CacheDisabled, nil)
					compiledWorflow := workflowutils.GetCompiledArgoWorkflow(pipelineSpecs, platformSpec, &argocompiler.Options{})
					if *createMissingGoldenFiles {
						_, err := os.Stat(compiledWorkflowFilePath)
						if err != nil {
							logger.Log("Creating Compiled Workflow File '%s'", compiledWorkflowFilePath)
							workflowutils.CreateCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						} else {
							logger.Log("Compiled Workflow File '%s' already exists", compiledWorkflowFilePath)
						}
					}
					if *updateGoldenFiles {
						logger.Log("Updating golden file %s", compiledWorkflowFilePath)
						_, err := os.Stat(compiledWorkflowFilePath)
						if err != nil {
							logger.Log("File %s does not exist, but if you want to create the missing workflow file, please set 'createGoldenFiles' flag to true", compiledWorkflowFilePath)
						} else {
							workflowutils.CreateCompiledWorkflowFile(compiledWorflow, compiledWorkflowFilePath)
						}
					}
					expectedWorkflow := workflowutils.UnmarshallWorkflowYAML(compiledWorkflowFilePath)
					matcher.CompareWorkflows(compiledWorflow, expectedWorkflow)

				})
			}
		})
	}
})
