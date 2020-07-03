// Copyright 2018 Google LLC
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

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
)

// Resource reference table models the relationship between resources in a loosely coupled way.
type ResourceReference struct {
	// ID of the resource object
	ResourceUUID string `gorm:"column:ResourceUUID; not null; primary_key"`

	// The type of the resource object
	ResourceType common.ResourceType `gorm:"column:ResourceType; not null; primary_key; index:referencefilter"`

	// The ID of the resource that been referenced to.
	ReferenceUUID string `gorm:"column:ReferenceUUID; not null; index:referencefilter"`

	// The name of the resource that been referenced to.
	ReferenceName string `gorm:"column:ReferenceName; not null; "`

	// The type of the resource that been referenced to.
	ReferenceType common.ResourceType `gorm:"column:ReferenceType; not null; primary_key; index:referencefilter"`

	// The relationship between the resource object and the resource that been referenced to.
	Relationship common.Relationship `gorm:"column:Relationship; not null; "`

	// The json formatted blob of the resource reference.
	Payload string `gorm:"column:Payload; not null; size:65535 "`
}

func GetNamespaceFromModelResourceReferences(resourceRefs []*ResourceReference) string {
	namespace := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.ReferenceType == common.Namespace {
			namespace = resourceRef.ReferenceUUID
			break
		}
	}
	return namespace
}
