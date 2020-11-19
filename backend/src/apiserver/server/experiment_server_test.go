package server

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Create experiment failed.")
}

func TestCreateExperiment_SingleUser_NamespaceNotAllowed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "In single-user mode, CreateExperimentRequest shouldn't contain resource references.")
}

func TestCreateExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{
		Name:        "exp1",
		Description: "first experiment",
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
				Relationship: api.Relationship_OWNER,
			},
		}}

	_, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeExperiments,
		Name:      experiment.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(ctx, resourceAttributes))).Error(),
	)
}

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get experiment failed.")
}

func TestGetExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.GetExperiment(ctx, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeExperiments,
		Name:      "exp1",
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(ctx, resourceAttributes))).Error(),
	)
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed.")
}

func TestListExperiment_SingleUser_NamespaceNotAllowed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "In single-user mode, ListExperiment cannot filter by namespace.")
}

func TestListExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.ListExperiment(ctx, &api.ListExperimentsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbList,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeExperiments,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzApiResourcesError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(ctx, resourceAttributes))).Error(),
	)
}

func TestListExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
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
			} else if !cmp.Equal(tc.expectedExperiments, response.Experiments, cmpopts.IgnoreFields(api.Experiment{}, "CreatedAt")) {
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
	// Create experiment and runs/jobs under it.
	clients, manager, experiment := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run1 := &api.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := runServer.validateCreateRunRequest(&api.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	_, err = runServer.CreateRun(nil, &api.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(resource.FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients)
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run2 := &api.Run{
		Name:               "run2",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err = runServer.validateCreateRunRequest(&api.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	_, err = runServer.CreateRun(nil, &api.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(resource.DefaultFakeUUID, nil))
	manager = resource.NewResourceManager(clients)
	jobServer := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job1 := &api.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err = jobServer.validateCreateJobRequest(&api.CreateJobRequest{Job: job1})
	assert.Nil(t, err)
	_, err = jobServer.CreateJob(nil, &api.CreateJobRequest{Job: job1})
	assert.Nil(t, err)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager, &ExperimentServerOptions{CollectMetrics: false})
	_, err = experimentServer.ArchiveExperiment(nil, &api.ArchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperiment(nil, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.Equal(t, api.Experiment_STORAGESTATE_ARCHIVED, result.StorageState)
	runs, err := runServer.ListRuns(nil, &api.ListRunsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, api.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, api.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err := jobServer.ListJobs(nil, &api.ListJobsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperiment(nil, &api.UnarchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperiment(nil, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.Equal(t, api.Experiment_STORAGESTATE_AVAILABLE, result.StorageState)
	runs, err = runServer.ListRuns(nil, &api.ListRunsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, api.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, api.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &api.ListJobsRequest{ResourceReferenceKey: &api.ResourceKey{Id: experiment.UUID, Type: api.ResourceType_EXPERIMENT}})
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)
}
