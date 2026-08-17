// Copyright 2026 The Kubeflow Authors
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

import apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"

// OutputIOTypeForIteration returns ITERATOR_OUTPUT when an iteration index is
// present so hydration can group loop outputs by iteration; otherwise OUTPUT.
func OutputIOTypeForIteration(iterationIndex *int64) apiv2beta1.IOType {
	if iterationIndex != nil {
		return apiv2beta1.IOType_ITERATOR_OUTPUT
	}
	return apiv2beta1.IOType_OUTPUT
}
