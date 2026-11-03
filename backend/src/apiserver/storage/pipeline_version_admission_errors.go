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

package storage

import (
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func pipelineVersionAdmissionError(err error) error {
	if cause, ok := k8serrors.StatusCause(err, v2beta1.PipelineVersionRejectedCause); ok {
		// The cause carries the webhook message without Kubernetes' admission-denial prefix.
		return util.NewInvalidInputError("%s", cause.Message)
	}
	return nil
}
