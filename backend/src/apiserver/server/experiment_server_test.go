package server

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:           resource.DefaultFakeUUID,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Create experiment failed.")
}

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "accounts.google.com:user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	result, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:                 resource.DefaultFakeUUID,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	expectedExperiment := &api.Experiment{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get experiment failed.")
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "accounts.google.com:user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(ctx, &api.GetExperimentRequest{Id: createResult.Id})
	expectedExperiment := &api.Experiment{
		Id:                 createResult.Id,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestListExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiment(nil, &api.ListExperimentsRequest{})
	expectedExperiment := []*api.Experiment{{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}}
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed.")
}

func TestListExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "accounts.google.com:user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	tests := []struct {
		name                string
		request             *api.ListExperimentsRequest
		wantError           bool
		errorMessage        string
		expectedExperiments []*api.Experiment
	}{
		{
			"Valid",
			&api.ListExperimentsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			[]*api.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       api.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
		{
			"Valid but empty result",
			&api.ListExperimentsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "ns2",
				},
			},
			false,
			"",
			[]*api.Experiment{},
		},
		{
			"Missing resource reference key",
			&api.ListExperimentsRequest{},
			true,
			"Invalid resource references for experiment.",
			nil,
		},
		{
			"Invalid resource reference key type",
			&api.ListExperimentsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_EXPERIMENT,
					Id:   "fake_id",
				},
			},
			true,
			"Invalid resource references for experiment.",
			nil,
		},
		{
			"Empty namespace",
			&api.ListExperimentsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "",
				},
			},
			true,
			"Invalid resource references for experiment. Namespace is empty.",
			nil,
		},
	}

	for _, tc := range tests {
		response, err := server.ListExperiment(ctx, tc.request)
		if tc.wantError {
			if err == nil {
				t.Errorf("TestListExperiment_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListExperiment_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListExperiment_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedExperiments, response.Experiments) {
				t.Errorf("TestListExperiment_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedExperiments, response.Experiments)
			}
		}
	}
}

func TestValidateCreateExperimentRequest(t *testing.T) {
	tests := []struct {
		name         string
		experiment   *api.Experiment
		wantError    bool
		errorMessage string
	}{
		{
			"Valid",
			&api.Experiment{Name: "exp1", Description: "first experiment"},
			false,
			"",
		},
		{
			"Empty name",
			&api.Experiment{Description: "first experiment"},
			true,
			"name is empty",
		},
	}

	for _, tc := range tests {
		err := ValidateCreateExperimentRequest(&api.CreateExperimentRequest{Experiment: tc.experiment})
		if !tc.wantError && err != nil {
			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
		}
		if tc.wantError {
			if err == nil {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		}
	}
}

func TestValidateCreateExperimentRequest_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	tests := []struct {
		name         string
		experiment   *api.Experiment
		wantError    bool
		errorMessage string
	}{
		{
			"Valid",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			false,
			"",
		},
		{
			"Missing namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
			},
			true,
			"Invalid resource references for experiment.",
		},
		{
			"Empty namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: ""},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment. Namespace is empty.",
		},
		{
			"Multiple namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: api.Relationship_OWNER,
					},
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns2"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment.",
		},
		{
			"Invalid resource type",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "exp2"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment.",
		},
	}

	for _, tc := range tests {
		err := ValidateCreateExperimentRequest(&api.CreateExperimentRequest{Experiment: tc.experiment})
		if !tc.wantError && err != nil {
			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
		}
		if tc.wantError {
			if err == nil {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		}
	}
}

func TestArchiveAndUnarchiveExperiment(t *testing.T) {
	// Create experiment and runs under it.
	clients, manager, experiment := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager)
	run1 := &api.Run{
		Name:               "123",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := runServer.validateCreateRunRequest(&api.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	_, err = runServer.CreateRun(nil, &api.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(resource.FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients)
	runServer = NewRunServer(manager)
	run2 := &api.Run{
		Name:               "456",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err = runServer.validateCreateRunRequest(&api.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	_, err = runServer.CreateRun(nil, &api.CreateRunRequest{Run: run2})
	assert.Nil(t, err)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager)
	_, err = experimentServer.ArchiveExperiment(nil, &api.ArchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperiment(nil, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.Equal(t, result.StorageState, api.Experiment_STORAGESTATE_ARCHIVED)
	runs, err := runServer.ListRuns(nil, &api.ListRunsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, runs.Runs[0].StorageState, api.Run_STORAGESTATE_ARCHIVED)
	assert.Equal(t, runs.Runs[1].StorageState, api.Run_STORAGESTATE_ARCHIVED)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperiment(nil, &api.UnarchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperiment(nil, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.Equal(t, result.StorageState, api.Experiment_STORAGESTATE_AVAILABLE)
	runs, err = runServer.ListRuns(nil, &api.ListRunsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, runs.Runs[0].StorageState, api.Run_STORAGESTATE_AVAILABLE)
	assert.Equal(t, runs.Runs[1].StorageState, api.Run_STORAGESTATE_AVAILABLE)
}
