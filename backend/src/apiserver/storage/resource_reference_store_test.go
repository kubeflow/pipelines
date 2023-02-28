// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
)

var testRefOne = &model.ResourceReference{
	ResourceUUID: "r1", ResourceType: model.RunResourceType,
	ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
	ReferenceType: model.ExperimentResourceType, Relationship: model.CreatorRelationship,
}

var testRefTwo = &model.ResourceReference{
	ResourceUUID: "j2", ResourceType: model.JobResourceType,
	ReferenceUUID: defaultFakeExpIdTwo, ReferenceName: "e2",
	ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
}

var testRefThree = &model.ResourceReference{
	ResourceUUID: defaultFakeExpId, ResourceType: model.ExperimentResourceType,
	ReferenceUUID: defaultFakeExpIdTwo, ReferenceName: "e2",
	ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
}

func TestResourceReferenceStore(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "e1"})
	expStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "e2"})
	store := NewResourceReferenceStore(db)

	// Create resource reference
	tx, _ := db.Begin()
	err := store.CreateResourceReferences(tx, []*model.ResourceReference{testRefOne, testRefTwo, testRefThree})
	assert.Nil(t, err)
	err = tx.Commit()
	assert.Nil(t, err)

	// Create resource reference - reference not exist
	tx, _ = db.Begin()
	err = store.CreateResourceReferences(tx, []*model.ResourceReference{
		{
			ResourceUUID: "notexist", ResourceType: model.JobResourceType,
			ReferenceUUID: "e1", ReferenceType: model.ExperimentResourceType,
			Relationship: model.OwnerRelationship,
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "e1 not found")
	tx.Rollback()

	// Get experiment resource reference
	experimentRef, err := store.GetResourceReference("r1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	payload, err := json.Marshal(testRefOne)
	assert.Equal(t, &model.ResourceReference{
		ResourceUUID: "r1", ResourceType: model.RunResourceType,
		ReferenceUUID: defaultFakeExpId, ReferenceName: "e1", ReferenceType: model.ExperimentResourceType,
		Relationship: model.CreatorRelationship, Payload: string(payload),
	}, experimentRef)

	// Delete resource references
	tx, _ = db.Begin()
	err = store.DeleteResourceReferences(tx, defaultFakeExpId, model.ExperimentResourceType)
	assert.Nil(t, err)
	err = tx.Commit()
	assert.Nil(t, err)

	_, err = store.GetResourceReference("r1", model.RunResourceType, model.ExperimentResourceType)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
	_, err = store.GetResourceReference(defaultFakeExpId, model.ExperimentResourceType, model.ExperimentResourceType)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
	_, err = store.GetResourceReference("j2", model.JobResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
}
