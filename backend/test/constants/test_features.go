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

// Package constants
package constants

const (
	// Smoke - quality gate tag for smoke testing
	Smoke string = "smoke"
	// Sanity - quality gate tag for sanity level testing
	Sanity string = "sanity"
	// Integration - quality gate tag for integration testing with other components
	Integration = "integration"
	// E2eEssential - For pipelines that are essential for regression testing
	E2eEssential string = "E2EEssential"
	// E2eFailed - For expectedly failing pipelines
	E2eFailed string = "E2EFailure"
	// E2eCritical - For pipelines that verify the critical functionality of the system
	E2eCritical string = "E2ECritical"
	// E2eProxy - For pipeline that runs with a proxy
	E2eProxy string = "E2EProxy"

	WorkflowCompiler       string = "WorkflowCompiler"
	WorkflowCompilerVisits string = "WorkflowCompilerVisits"

	APIServerTests string = "ApiServerTests"

	Experiment           string = "Experiment"
	Pipeline             string = "Pipeline"
	PipelineRun          string = "PipelineRun"
	PipelineScheduledRun string = "PipelineRecurringRun"
	PipelineUpload       string = "PipelineUpload"
	ReportTests          string = "Report"

	UpgradePreparation  string = "UpgradePreparation"
	UpgradeVerification string = "UpgradeVerification"
)
