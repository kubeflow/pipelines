// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package transfer defines the self-service namespace transfer contract.
package transfer

// MaxArchiveBytes bounds metadata archives; artifact files are never included.
const MaxArchiveBytes = 256 << 20

// ExportOptions limits completed history while retaining the full catalog.
type ExportOptions struct {
	CompletedAfter  int64 `json:"completed_after,omitempty"`
	CompletedBefore int64 `json:"completed_before,omitempty"`
}

// ImportOptions controls destination names and validation without publication.
type ImportOptions struct {
	NamePrefix string
	DryRun     bool
}

// Counts describes the resources in an archive.
type Counts struct {
	Experiments      int `json:"experiments"`
	Pipelines        int `json:"pipelines"`
	PipelineVersions int `json:"pipeline_versions"`
	Runs             int `json:"runs"`
	Schedules        int `json:"schedules"`
}

// Summary is returned after validating or applying an archive.
type Summary struct {
	Counts   Counts   `json:"counts"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	DryRun   bool     `json:"dry_run"`
	Warnings []string `json:"warnings"`
}
