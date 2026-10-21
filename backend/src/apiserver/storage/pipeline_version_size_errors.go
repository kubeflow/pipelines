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
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	pipelineVersionCreateSizeMessage = "Kubernetes rejected the pipeline version because the request is too large. " +
		"Reduce the compiled pipeline size, for example by moving embedded files or inline Python code into a container image or object storage."
	pipelineVersionUpdateSizeMessage = "Kubernetes rejected the pipeline version update because the request is too large. " +
		"Reduce the display name or the number or length of tags."
)

func pipelineVersionSizeError(err error, message string) error {
	// Unwrap an already translated create error before adding first-version recovery advice.
	// This keeps the Kubernetes error once in the status message.
	err = errors.Cause(err)
	if k8serrors.IsRequestEntityTooLargeError(err) {
		return util.NewInvalidInputErrorWithDetails(err, message)
	}
	// etcd request and client-send limits arrive as Kubernetes 500 responses without a size-specific reason.
	if status, ok := err.(k8serrors.APIStatus); ok && status.Status().Code == 500 &&
		(strings.Contains(err.Error(), "etcdserver: request is too large") ||
			strings.Contains(err.Error(), "trying to send message larger than max")) {
		return util.NewInvalidInputErrorWithDetails(err, message)
	}
	return nil
}
