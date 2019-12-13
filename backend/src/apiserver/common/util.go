// Copyright 2019 Google LLC
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

package common

import (
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/bazel-pipelines/backend/src/apiserver/common"
)

func GetNamespaceFromResourceReferences(resourceRefs []*api.ResourceReference) string {
	namespace := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == api.ResourceType_NAMESPACE {
			namespace = resourceRef.Key.Id
			break
		}
	}
	return namespace
}

func GetNamespaceFromResourceReferencesModel(resourceRefs []*model.ResourceReference) string {
	namespace := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.ReferenceType == Namespace {
			namespace = resourceRef.ReferenceName
			break
		}
	}
	return namespace
}
