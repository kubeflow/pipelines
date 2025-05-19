// Copyright 2022 The Kubeflow Authors
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

type RuntimeConfig struct {
	// Store parameters key-value pairs as serialized string.
	Parameters string `gorm:"column:RuntimeParameters; type:longtext;"`

	// A path in a object store bucket which will be treated as the root
	// output directory of the pipeline. It is used by the system to
	// generate the paths of output artifacts. Ref:(https://www.kubeflow.org/docs/components/pipelines/pipeline-root/)
	PipelineRoot string `gorm:"column:PipelineRoot; type:longtext;"`
}
