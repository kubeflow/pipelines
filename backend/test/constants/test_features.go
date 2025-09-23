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

package api

const (
	E2E_NON_CRITICAL string = "E2ENonCritical"
	E2E_CRITICAL     string = "E2ECritical"
	E2E_PROXY        string = "E2ECritical"

	WORKFLOW_COMPILER        string = "WorkflowCompiler"
	WORKFLOW_COMPILER_VISITS string = "WorkflowCompilerVisits"

	API_SERVER_TESTS string = "ApiServerTests"

	API_EXPERIMENT             string = "Experiment"
	API_PIPELINE               string = "Pipeline"
	API_PIPELINE_RUN           string = "PipelineRun"
	API_PIPELINE_SCHEDULED_RUN string = "PipelineRecurringRun"
	API_PIPELINE_UPLOAD        string = "PipelineUpload"
	API_REPORT                 string = "Report"

	UPGRADE_PREPARATION  string = "UpgradePreparation"
	UPGRADE_VERIFICATION string = "UpgradeVerification"
)
