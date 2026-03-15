// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pipeline_upload_service provides helpers for the pipeline upload API client.
package pipeline_upload_service //nolint:staticcheck // ST1003: package name uses underscores (generated code convention)

import "encoding/json"

// SetTagsMap is a convenience method that serializes a map[string]string
// to JSON and sets it on the Tags field of UploadPipelineParams.
func (o *UploadPipelineParams) SetTagsMap(tags map[string]string) {
	if tags == nil {
		o.Tags = nil
		return
	}
	data, err := json.Marshal(tags)
	if err != nil {
		return
	}
	s := string(data)
	o.Tags = &s
}

// SetTagsMap is a convenience method that serializes a map[string]string
// to JSON and sets it on the Tags field of UploadPipelineVersionParams.
func (o *UploadPipelineVersionParams) SetTagsMap(tags map[string]string) {
	if tags == nil {
		o.Tags = nil
		return
	}
	data, err := json.Marshal(tags)
	if err != nil {
		return
	}
	s := string(data)
	o.Tags = &s
}
