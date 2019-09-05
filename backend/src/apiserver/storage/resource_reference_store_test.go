package storage

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
)

var testRefOne = &model.ResourceReference{
	ResourceUUID: "r1", ResourceType: common.Run,
	ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
	ReferenceType: common.Experiment, Relationship: common.Creator,
}

var testRefTwo = &model.ResourceReference{
	ResourceUUID: "j2", ResourceType: common.Job,
	ReferenceUUID: defaultFakeExpIdTwo, ReferenceName: "e2",
	ReferenceType: common.Experiment, Relationship: common.Owner,
}

var testRefThree = &model.ResourceReference{
	ResourceUUID: defaultFakeExpId, ResourceType: common.Experiment,
	ReferenceUUID: defaultFakeExpIdTwo, ReferenceName: "e2",
	ReferenceType: common.Experiment, Relationship: common.Owner,
}

func TestResourceReferenceStore(t *testing.T) {
	db := NewFakeDbOrFatal()
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
			ResourceUUID: "notexist", ResourceType: common.Job,
			ReferenceUUID: "e1", ReferenceType: common.Experiment,
			Relationship: common.Owner,
		}})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "e1 not found")
	tx.Rollback()

	// Get experiment resource reference
	experimentRef, err := store.GetResourceReference("r1", common.Run, common.Experiment)
	assert.Nil(t, err)
	payload, err := json.Marshal(testRefOne)
	assert.Equal(t, &model.ResourceReference{
		ResourceUUID: "r1", ResourceType: common.Run,
		ReferenceUUID: defaultFakeExpId, ReferenceName: "e1", ReferenceType: common.Experiment,
		Relationship: common.Creator, Payload: string(payload)}, experimentRef)

	// Delete resource references
	tx, _ = db.Begin()
	err = store.DeleteResourceReferences(tx, defaultFakeExpId, common.Experiment)
	assert.Nil(t, err)
	err = tx.Commit()
	assert.Nil(t, err)

	_, err = store.GetResourceReference("r1", common.Run, common.Experiment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
	_, err = store.GetResourceReference(defaultFakeExpId, common.Experiment, common.Experiment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
	_, err = store.GetResourceReference("j2", common.Job, common.Experiment)
	assert.Nil(t, err)
}
