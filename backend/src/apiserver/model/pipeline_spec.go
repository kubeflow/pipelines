// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

type PipelineSpec struct {
	// Pipeline ID will be optional. It's available only if the resource is created through
	// a pipeline ID.
	PipelineId string `gorm:"column:PipelineId; not null"`

	// Pipeline Name will be required if ID is not empty.
	PipelineName string `gorm:"column:PipelineName; not null"`

	// Pipeline YAML definition. This is the pipeline interface for creating a pipeline.
	// Set size to 65535 so it will be stored as longtext.
	// https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html
	PipelineSpecManifest string `gorm:"column:PipelineSpecManifest; size:65535"`

	// Argo workflow YAML definition. This is the Argo Spec converted from Pipeline YAML.
	WorkflowSpecManifest string `gorm:"column:WorkflowSpecManifest; not null; size:65535"`

	// Store parameters key-value pairs as serialized string.
	Parameters string `gorm:"column:Parameters; size:65535"`
}
