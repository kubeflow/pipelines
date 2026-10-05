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

package resource

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// AuthorizeTransfer requires namespace permissions even in shared-read mode:
// exporting a complete archive includes runtime manifests and metadata.
func (r *ResourceManager) AuthorizeTransfer(ctx context.Context, namespace string, importing bool) error {
	if namespace == "" && common.IsMultiUserMode() {
		return util.NewInvalidInputError("Choose a namespace before exporting or importing")
	}
	if namespace != "" && len(validation.IsDNS1123Label(namespace)) != 0 {
		return util.NewInvalidInputError("Namespace must be a valid Kubernetes namespace name")
	}
	verbs := []string{common.RbacResourceVerbList, common.RbacResourceVerbGet}
	if importing {
		verbs = []string{common.RbacResourceVerbList, common.RbacResourceVerbCreate}
	}
	for _, resource := range []string{common.RbacResourceTypeExperiments, common.RbacResourceTypePipelines, common.RbacResourceTypeRuns, common.RbacResourceTypeJobs} {
		for _, verb := range verbs {
			if err := r.isAuthorized(ctx, &authorizationv1.ResourceAttributes{
				Namespace: namespace, Group: common.RbacPipelinesGroup,
				Version: common.RbacPipelinesVersion, Resource: resource, Verb: verb,
			}, false); err != nil {
				return err
			}
		}
	}
	return nil
}
