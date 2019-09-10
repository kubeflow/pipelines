// Copyright 2018 Google LLC
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

package storage

import (
	"path"
)

// Interface for getting path for different objects in the object store.
type ObjectPathsInterface interface {
	GetPipelinePath(pipelineID string) string
}

// how to store various objects in the object store
type ObjectPaths struct {
	pipelineFolder string
}

// GetPipelinePath return the object store path to a pipeline spec.
func (p *ObjectPaths) GetPipelinePath(pipelineID string) string {
	return path.Join(p.pipelineFolder, pipelineID)
}

func NewObjectPaths(pipelineFolder string) *ObjectPaths {
	return &ObjectPaths{pipelineFolder: pipelineFolder}
}
