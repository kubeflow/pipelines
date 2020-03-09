package storage

import (
	"testing"

	"fmt"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	fakeID      = "123e4567-e89b-12d3-a456-426655440000"
	fakeIDTwo   = "123e4567-e89b-12d3-a456-426655440001"
	fakeIDThree = "123e4567-e89b-12d3-a456-426655440002"
	fakeIDFour  = "123e4567-e89b-12d3-a456-426655440003"
)

func createExperiment(name string) *model.Experiment {
	return createExperimentInNamespace(name, "")
}

func createExperimentInNamespace(name string, namespace string) *model.Experiment {
	return &model.Experiment{
		Name:        name,
		Description: fmt.Sprintf("My name is %s", name),
		Namespace:   namespace,
	}
}

func TestListExperiments_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))
	expectedExperiment1 := &model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
	}
	expectedExperiment4 := &model.Experiment{
		UUID:           fakeIDFour,
		CreatedAtInSec: 4,
		Name:           "experiment2",
		Description:    "My name is experiment2",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment1, expectedExperiment4}
	opts, err := list.NewOptions(&model.Experiment{}, 2, "name", nil)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(opts)

	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, experimentsExpected, experiments)
	assert.Equal(t, 4, total_size)

	expectedExperiment2 := &model.Experiment{
		UUID:           fakeIDTwo,
		CreatedAtInSec: 2,
		Name:           "experiment3",
		Description:    "My name is experiment3",
	}
	expectedExperiment3 := &model.Experiment{
		UUID:           fakeIDThree,
		CreatedAtInSec: 3,
		Name:           "experiment4",
		Description:    "My name is experiment4",
	}
	experimentsExpected2 := []*model.Experiment{expectedExperiment2, expectedExperiment3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected2, experiments)
}

func TestListExperiments_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))

	expectedExperiment2 := &model.Experiment{
		UUID:           fakeIDTwo,
		CreatedAtInSec: 2,
		Name:           "experiment3",
		Description:    "My name is experiment3",
	}
	expectedExperiment3 := &model.Experiment{
		UUID:           fakeIDThree,
		CreatedAtInSec: 3,
		Name:           "experiment4",
		Description:    "My name is experiment4",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment3, expectedExperiment2}

	opts, err := list.NewOptions(&model.Experiment{}, 2, "name desc", nil)
	assert.Nil(t, err)
	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(opts)

	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected, experiments)

	expectedExperiment1 := &model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
	}
	expectedExperiment4 := &model.Experiment{
		UUID:           fakeIDFour,
		CreatedAtInSec: 4,
		Name:           "experiment2",
		Description:    "My name is experiment2",
	}
	experimentsExpected2 := []*model.Experiment{expectedExperiment4, expectedExperiment1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected2, experiments)
}

func TestListExperiments_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	expectedExperiment1 := &model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment1}

	opts, err := list.NewOptions(&model.Experiment{}, 2, "", nil)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, experimentsExpected, experiments)
}

func TestListExperimentsError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()

	opts, err := list.NewOptions(&model.Experiment{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = experimentStore.ListExperiments(opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetExperiment(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
	}

	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment.")
}

func TestGetExperiment_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))

	_, err := experimentStore.GetExperiment(fakeID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get experiment to return not found")
}

func TestGetExperiment_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()
	_, err := experimentStore.GetExperiment("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get experiment to return internal error")
}

func TestCreateExperiment(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
	}

	experiment := createExperiment("experiment1")
	experiment, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment.")
}

func TestCreateExperiment_DifferentNamespaces(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		Namespace:      "namespace1",
	}

	experiment := createExperimentInNamespace("experiment1", "namespace1")
	experiment, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment.")

	experimentStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil))
	experiment = createExperimentInNamespace("experiment1", "namespace2")
	experimentExpected = model.Experiment{
		UUID:           fakeIDTwo,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		Namespace:      "namespace2",
	}

	experiment, err = experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment.")
}

func TestCreateExperiment_DuplicatedKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experiment := createExperiment("experiment1")
	_, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)

	experimentStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil))
	_, err = experimentStore.CreateExperiment(experiment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name experiment1 already exist")
}

func TestCreateExperiment_InternalServerError(t *testing.T) {
	experiment := &model.Experiment{Name: "Experiment123"}
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()

	_, err := experimentStore.CreateExperiment(experiment)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create experiment to return error")
}

func TestCreateExperiment_CreateUUIDFailure(t *testing.T) {
	experiment := &model.Experiment{Name: "Experiment123"}
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, errors.New("error")))
	db.Close()

	_, err := experimentStore.CreateExperiment(experiment)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Failed to create an experiment id")
}

func TestDeleteExperiment(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.NotNil(t, experiment)

	err = experimentStore.DeleteExperiment(fakeID)
	assert.Nil(t, err)
	_, err = experimentStore.GetExperiment(fakeID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.NotNil(t, experiment)

	db.Close()
	err = experimentStore.DeleteExperiment(fakeID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete experiment to return internal error")
}

func TestListExperiments_Filtering(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key: "name",
				Op:  api.Predicate_IN,
				Value: &api.Predicate_StringValues{
					StringValues: &api.StringValues{
						Values: []string{"experiment2", "experiment4", "experiment3"},
					},
				},
			},
		},
	}

	opts, err := list.NewOptions(&model.Experiment{}, 2, "id", filterProto)
	assert.Nil(t, err)
	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(opts)

	expected := []*model.Experiment{
		&model.Experiment{
			UUID:           fakeIDTwo,
			CreatedAtInSec: 2,
			Name:           "experiment2",
			Description:    "My name is experiment2",
		},
		&model.Experiment{
			UUID:           fakeIDThree,
			CreatedAtInSec: 3,
			Name:           "experiment3",
			Description:    "My name is experiment3",
		},
	}

	assert.Nil(t, err)
	assert.NotEqual(t, "", nextPageToken)
	assert.Equal(t, expected, experiments)
	assert.Equal(t, 3, total_size)

	// Next page should give experiment4.
	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(opts)

	expected = []*model.Experiment{
		&model.Experiment{
			UUID:           fakeIDFour,
			CreatedAtInSec: 4,
			Name:           "experiment4",
			Description:    "My name is experiment4",
		},
	}

	assert.Nil(t, err)
	// No more pages.
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, expected, experiments)
	assert.Equal(t, 3, total_size)
}
