// Copyright 2018 The Kubeflow Authors
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
	// Parent pipeline's id.
	PipelineId string `gorm:"column:PipelineId; not null;"`

	// Pipeline version's id.
	PipelineVersionId string `gorm:"column:PipelineVersionId; default:null;"`

	// TODO(gkcalat): consider adding PipelineVersionName to avoid confusion.
	// Pipeline versions's Name will be required if ID is not empty.
	// This carries the name of the pipeline version in v2beta1.
	PipelineName string `gorm:"column:PipelineName; not null;"`

	// Pipeline YAML definition. This is the pipeline interface for creating a pipeline.
	// Stored as longtext to support large manifests (up to 4GB in MySQL).
	// https://dev.mysql.com/doc/refman/8.0/en/blob.html
	// TODO(kaikaila): consider enforcing a soft limit if needed for performance.
	PipelineSpecManifest string `gorm:"column:PipelineSpecManifest; type:longtext;"`

	// Argo workflow YAML definition. This is the Argo Spec converted from Pipeline YAML.
	// This is deprecated. Use the pipeline ID, pipeline version ID, or pipeline spec manifest.
	WorkflowSpecManifest string `gorm:"column:WorkflowSpecManifest; type:longtext;"`

	// Store parameters key-value pairs as serialized string.
	// This field is only used for V1 API. For V2, use the `Parameters` field in RuntimeConfig.
	// At most one of the fields `Parameters` and `RuntimeConfig` can be non-empty
	// This string stores an array of map[string]value. For example:
	//  {"param1": Value1} will be stored as [{"name": "param1", "value":"value1"}].
	Parameters string `gorm:"column:Parameters; type:longtext;"`

	// Runtime config of the pipeline, only used for v2 template in API v1beta1 API.
	RuntimeConfig
}
