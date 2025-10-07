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
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	matcher "github.com/kubeflow/pipelines/backend/test/compiler/matchers"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeEach(func() {
	logger.Log("Initializing proxy config...")
	proxy.InitializeConfigWithEmptyForTests()
})

var _ = Describe("Verify Spec Compilation to Workflow >", Label(POSITIVE, WorkflowCompiler), func() {
	pipelineFilePaths := testutil.GetListOfAllFilesInDir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	testParams := []struct {
		compilerOptions argocompiler.Options
		envVars         map[string]string
	}{
		{
			compilerOptions: argocompiler.Options{CacheDisabled: true},
		},
		{
			compilerOptions: argocompiler.Options{CacheDisabled: true},
			envVars:         map[string]string{"PIPELINE_RUN_AS_USER": "1001", "PIPELINE_LOG_LEVEL": "3"},
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
		Context(fmt.Sprintf("Verify compiled workflow for a pipeline with compiler options cacheDisabled '%v' and env vars %v >", param.compilerOptions.CacheDisabled, param.envVars), Ordered, func() {
			for _, pipelineSpecFilePath := range pipelineFilePaths {
				pipelineSpecFileName := filepath.Base(pipelineSpecFilePath)
				fileExtension := filepath.Ext(pipelineSpecFileName)
				fileNameWithoutExtension := strings.TrimSuffix(pipelineSpecFileName, fileExtension)
				compiledWorkflowFileName := fileNameWithoutExtension + ".yaml"
				compiledWorkflowFilePath := filepath.Join(argoYAMLDir, compiledWorkflowFileName)
				It(fmt.Sprintf("When I compile %s pipeline spec, then the compiled yaml should be %s", pipelineSpecFileName, compiledWorkflowFileName), func() {
					testutil.CheckIfSkipping(pipelineSpecFileName)
					pipelineSpecs, platformSpec := workflowutils.LoadPipelineSpecsFromIR(pipelineSpecFilePath, param.compilerOptions.CacheDisabled, nil)
					compiledWorkflow := workflowutils.GetCompiledArgoWorkflow(pipelineSpecs, platformSpec, &param.compilerOptions)
					if *createMissingGoldenFiles || *updateGoldenFiles {
						var configuredWorkflow *v1alpha1.Workflow
						if param.compilerOptions.CacheDisabled {
							configuredWorkflow = workflowutils.ConfigureCacheSettings(compiledWorkflow, true)
						} else {
							configuredWorkflow = compiledWorkflow
						}
						logger.Log("Updating/Creating Compiled Workflow File '%s'", compiledWorkflowFilePath)
						workflowutils.CreateCompiledWorkflowFile(configuredWorkflow, compiledWorkflowFilePath)
					}
					expectedWorkflow := workflowutils.UnmarshallWorkflowYAML(compiledWorkflowFilePath)
					if param.compilerOptions.CacheDisabled {
						expectedWorkflow = workflowutils.ConfigureCacheSettings(expectedWorkflow, false)
					}

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

					matcher.CompareWorkflows(compiledWorkflow, expectedWorkflow)

				})
			}
		})
	}
})
