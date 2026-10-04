// Copyright 2025 The Kubeflow Authors
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

package util

import (
	"fmt"
	"path"
	"strings"

	"github.com/golang/glog"
)

// GenerateOutputURI appends the specified paths to the pipeline root.
// It may be configured to preserve the query part of the pipeline root
// by splitting it off and appending it back to the full URI.
func GenerateOutputURI(pipelineRoot string, paths []string, preserveQueryString bool) string {
	querySplit := strings.Split(pipelineRoot, "?")
	query := ""
	if len(querySplit) == 2 {
		pipelineRoot = querySplit[0]
		if preserveQueryString {
			query = "?" + querySplit[1]
		}
	} else if len(querySplit) > 2 {
		// this should never happen, but just in case.
		glog.Warningf("Unexpected pipeline root: %v", pipelineRoot)
	}
	// we cannot path.Join(root, taskName, artifactName), because root
	// contains scheme like gs:// and path.Join cleans up scheme to gs:/
	return fmt.Sprintf("%s/%s%s", strings.TrimRight(pipelineRoot, "/"), path.Join(paths...), query)
}
