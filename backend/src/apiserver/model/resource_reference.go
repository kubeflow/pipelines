// Copyright 2018 The Kubeflow Authors
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

const (
	NamespaceResourceType       ResourceType = "Namespace"
	ExperimentResourceType      ResourceType = "Experiment"
	JobResourceType             ResourceType = "Job"
	RecurringRunResourceType    ResourceType = "RecurringRun"
	RunResourceType             ResourceType = "Run"
	PipelineResourceType        ResourceType = "pipeline"
	PipelineVersionResourceType ResourceType = "PipelineVersion"
	ArtifactResourceType        ResourceType = "Artifact"
	TaskResourceType            ResourceType = "Task"
)

const (
	OwnerRelationship   Relationship = "Owner"
	CreatorRelationship Relationship = "Creator"
)

// ResourceType The type of a resource object.
type ResourceType string

// Relationship The relationship between two resource objects.
type Relationship string

// ResourceReference Resource reference table models the relationship between resources in a loosely coupled way.
// This model has a composite primary key consisting of ResourceUUID, ResourceType, and ReferenceType.
type ResourceReference struct {
	// ID of the resource object
	ResourceUUID string `gorm:"column:ResourceUUID; not null; primaryKey; type:varchar(191);"`

	// The type of the resource object
	ResourceType ResourceType `gorm:"column:ResourceType; not null; primaryKey; index:referencefilter;"`

	// The ID of the referenced resource.
	ReferenceUUID string `gorm:"column:ReferenceUUID; not null; index:referencefilter; type:varchar(191);"`

	// The name of the referenced resource.
	ReferenceName string `gorm:"column:ReferenceName; not null;"`

	// The type of the referenced resource.
	ReferenceType ResourceType `gorm:"column:ReferenceType; not null; primaryKey; index:referencefilter;"`

	// The relationship between the resource object and the referenced resource.
	Relationship Relationship `gorm:"column:Relationship; not null;"`

	// JSON-encoded metadata blob about the reference
	Payload LargeText `gorm:"column:Payload; not null;"`
}

type ReferenceKey struct {
	Type ResourceType
	ID   string
}

type FilterContext struct {
	// Filter by a specific reference key
	*ReferenceKey
}

// GetRefIdFromResourceReferences fetches the first reference id of a given type
func GetRefIdFromResourceReferences(resRefs []*ResourceReference, refType ResourceType) string {
	for _, ref := range resRefs {
		if ref.ReferenceUUID != "" && ref.ReferenceType == refType {
			return ref.ReferenceUUID
		}
	}
	return ""
}
