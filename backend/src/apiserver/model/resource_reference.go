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
)

const (
	OwnerRelationship   Relationship = "Owner"
	CreatorRelationship Relationship = "Creator"
)

type ResourceReferenceRelationshipTriplet struct {
	ResourceType     ResourceType
	ReferenceType    ResourceType
	RelationshipType Relationship
}

// Describes all possible relationship types between resources.
var validResourceReferenceRelationship map[ResourceReferenceRelationshipTriplet]bool = map[ResourceReferenceRelationshipTriplet]bool{
	// Experiment
	{
		ResourceType:     ExperimentResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,

	// Pipeline
	{
		ResourceType:     PipelineResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,

	// PipelineVersion
	{
		ResourceType:     PipelineVersionResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     PipelineVersionResourceType,
		ReferenceType:    PipelineResourceType,
		RelationshipType: OwnerRelationship,
	}: true,

	// Job
	{
		ResourceType:     JobResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     JobResourceType,
		ReferenceType:    ExperimentResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     JobResourceType,
		ReferenceType:    PipelineResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
	{
		ResourceType:     JobResourceType,
		ReferenceType:    PipelineVersionResourceType,
		RelationshipType: CreatorRelationship,
	}: true,

	// RecurringRon
	{
		ResourceType:     RecurringRunResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     RecurringRunResourceType,
		ReferenceType:    ExperimentResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     RecurringRunResourceType,
		ReferenceType:    PipelineResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
	{
		ResourceType:     RecurringRunResourceType,
		ReferenceType:    PipelineVersionResourceType,
		RelationshipType: CreatorRelationship,
	}: true,

	// Run
	{
		ResourceType:     RunResourceType,
		ReferenceType:    NamespaceResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     RunResourceType,
		ReferenceType:    ExperimentResourceType,
		RelationshipType: OwnerRelationship,
	}: true,
	{
		ResourceType:     RunResourceType,
		ReferenceType:    JobResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
	{
		ResourceType:     RunResourceType,
		ReferenceType:    RecurringRunResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
	{
		ResourceType:     RunResourceType,
		ReferenceType:    PipelineResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
	{
		ResourceType:     RunResourceType,
		ReferenceType:    PipelineVersionResourceType,
		RelationshipType: CreatorRelationship,
	}: true,
}

// The type of a resource object.
type ResourceType string

// The relationship between two resource objects.
type Relationship string

// Resource reference table models the relationship between resources in a loosely coupled way.
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
	Payload string `gorm:"column:Payload; not null; type: longtext"`
}

type ReferenceKey struct {
	Type ResourceType
	ID   string
}

type FilterContext struct {
	// Filter by a specific reference key
	*ReferenceKey
}

// Checks whether the resource-reference relationship combination is valid.
func ValidateResourceReferenceRelationship(resType ResourceType, refType ResourceType, relType Relationship) bool {
	check, err := validResourceReferenceRelationship[ResourceReferenceRelationshipTriplet{
		ResourceType:     resType,
		ReferenceType:    refType,
		RelationshipType: relType,
	}]
	if err {
		return check
	}
	return false
}

// Fetches the first reference id of a given type
func GetRefIdFromResourceReferences(resRefs []*ResourceReference, refType ResourceType) string {
	for _, ref := range resRefs {
		if ref.ReferenceUUID != "" && ref.ReferenceType == refType {
			return ref.ReferenceUUID
		}
	}
	return ""
}
