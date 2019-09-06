// Copyright 2019 Google LLC
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

type PipelineVersionSpec struct {
	// PipelineVersionID will be optional. It's available only if the resource
	// is created through a pipeline version ID.
	PipelineVersionId string `gorm:"column:PipelineVersionId;not null;"`

	// PipelineVersionName will be required if ID is not empty.
	PipelineVersionName string `gorm:"column:PipelineVersionName;not null;"`
}
