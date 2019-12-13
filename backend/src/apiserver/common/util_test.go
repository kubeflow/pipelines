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
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/stretchr/testify/assert"
)

func TestGetNamespaceFromResourceReferences(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_NAMESPACE, Id: "ns"},
			Relationship: api.Relationship_OWNER,
		},
	}
	namespace := GetNamespaceFromResourceReferences(references)
	assert.Equal(t, "ns", namespace)

	references = []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
	}
	namespace = GetNamespaceFromResourceReferences(references)
	assert.Equal(t, "", namespace)
}
