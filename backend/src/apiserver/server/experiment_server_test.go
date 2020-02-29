package server

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
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
		Id:          resource.DefaultFakeUUID,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
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
		Id:          createResult.Id,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
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
		Id:          createResult.Id,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
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
	referenceKey := &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          referenceKey,
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

	result, err := server.ListExperiment(ctx, &api.ListExperimentsRequest{ResourceReferenceKey: referenceKey})
	assert.Nil(t, err)
	expectedExperiment := []*api.Experiment{{
		Id:                 createResult.Id,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
	}}
	assert.Equal(t, expectedExperiment, result.Experiments)

	result, err = server.ListExperiment(ctx, &api.ListExperimentsRequest{ResourceReferenceKey: &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns2"}})
	assert.Nil(t, err)
	expectedExperiment = []*api.Experiment{}
	assert.Equal(t, expectedExperiment, result.Experiments)
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
				Name:               "exp1",
				Description:        "first experiment",
				ResourceReferences: []*api.ResourceReference{},
			},
			true,
			"Experiment should specify one and only one resource reference",
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
			"Experiment should specify one and only one resource reference",
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
			"Experiment should specify namespace",
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
