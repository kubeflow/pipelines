// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component_test

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "gocloud.dev/blob/memblob"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes/fake"
)

// Clone at the service boundary so failed writes cannot change persisted state.
type finalizationMetadataStore struct {
	pb.MetadataStoreServiceClient
	executions              map[int64]*pb.Execution
	publicationFailures     []error
	dagLookupFailures       []error
	ancestorUpdateFailures  []error
	publicationResponseLost bool
	publicationRequests     []*pb.PutExecutionRequest
	taskPipelineReads       int
	artifacts               []*pb.Artifact
	artifactEvents          []*pb.Event
	reverseResults          bool
	panicRecordingArtifact  bool
}

func nextFinalizationFailure(failures *[]error) error {
	if len(*failures) == 0 {
		return nil
	}
	failure := (*failures)[0]
	*failures = (*failures)[1:]
	return failure
}

func (store *finalizationMetadataStore) GetExecutionsByID(_ context.Context, request *pb.GetExecutionsByIDRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
	response := &pb.GetExecutionsByIDResponse{}
	for _, identifier := range request.GetExecutionIds() {
		if identifier == 20 {
			if lookupError := nextFinalizationFailure(&store.dagLookupFailures); lookupError != nil {
				return nil, lookupError
			}
		}
		response.Executions = append(response.Executions, proto.Clone(store.executions[identifier]).(*pb.Execution))
	}
	return response, nil
}

func (*finalizationMetadataStore) GetContextType(_ context.Context, request *pb.GetContextTypeRequest, _ ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
	identifier := int64(1)
	if request.GetTypeName() == "system.PipelineRun" {
		identifier = 2
	}
	return &pb.GetContextTypeResponse{ContextType: &pb.ContextType{Id: &identifier}}, nil
}

func (store *finalizationMetadataStore) GetContextsByExecution(_ context.Context, request *pb.GetContextsByExecutionRequest, _ ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	if request.GetExecutionId() == 10 {
		store.taskPipelineReads++
	}
	return &pb.GetContextsByExecutionResponse{Contexts: []*pb.Context{
		{Id: proto.Int64(1), TypeId: proto.Int64(1)},
		{Id: proto.Int64(2), TypeId: proto.Int64(2), CustomProperties: map[string]*pb.Value{
			"pipeline_root": metadata.StringValue("mem://finalization-test/root"),
		}},
	}}, nil
}

func (store *finalizationMetadataStore) GetExecutionsByContext(_ context.Context, request *pb.GetExecutionsByContextRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error) {
	parentIdentifier, parseError := strconv.ParseInt(strings.TrimPrefix(request.GetOptions().GetFilterQuery(), "custom_properties.parent_dag_id.int_value = "), 10, 64)
	if parseError != nil {
		return nil, parseError
	}
	response := &pb.GetExecutionsByContextResponse{}
	for _, execution := range store.executions {
		if execution.GetCustomProperties()["parent_dag_id"].GetIntValue() == parentIdentifier {
			response.Executions = append(response.Executions, proto.Clone(execution).(*pb.Execution))
		}
	}
	return response, nil
}

func (store *finalizationMetadataStore) PutExecution(_ context.Context, request *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
	execution := request.GetExecution()
	var writeError error
	if execution.GetId() == 10 && execution.GetLastKnownState() != pb.Execution_RUNNING {
		store.publicationRequests = append(store.publicationRequests, proto.Clone(request).(*pb.PutExecutionRequest))
		writeError = nextFinalizationFailure(&store.publicationFailures)
	} else if execution.GetId() == 30 {
		writeError = nextFinalizationFailure(&store.ancestorUpdateFailures)
	}
	if writeError == nil || store.publicationResponseLost && execution.GetId() == 10 {
		store.executions[execution.GetId()] = proto.Clone(execution).(*pb.Execution)
		for _, pair := range request.GetArtifactEventPairs() {
			store.artifactEvents = append(store.artifactEvents, proto.Clone(pair.GetEvent()).(*pb.Event))
		}
	}
	return &pb.PutExecutionResponse{ExecutionId: execution.Id}, writeError
}

func (*finalizationMetadataStore) PutArtifactType(context.Context, *pb.PutArtifactTypeRequest, ...grpc.CallOption) (*pb.PutArtifactTypeResponse, error) {
	return &pb.PutArtifactTypeResponse{TypeId: proto.Int64(1)}, nil
}

func (store *finalizationMetadataStore) PutArtifacts(_ context.Context, request *pb.PutArtifactsRequest, _ ...grpc.CallOption) (*pb.PutArtifactsResponse, error) {
	if store.panicRecordingArtifact {
		panic("record artifact")
	}
	response := &pb.PutArtifactsResponse{}
	for _, artifact := range request.GetArtifacts() {
		storedArtifact := proto.Clone(artifact).(*pb.Artifact)
		storedArtifact.Id = proto.Int64(int64(len(store.artifacts) + 1))
		store.artifacts = append(store.artifacts, storedArtifact)
		response.ArtifactIds = append(response.ArtifactIds, storedArtifact.GetId())
	}
	return response, nil
}

func (store *finalizationMetadataStore) GetArtifactsByID(_ context.Context, request *pb.GetArtifactsByIDRequest, _ ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error) {
	response := &pb.GetArtifactsByIDResponse{}
	for _, identifier := range request.GetArtifactIds() {
		response.Artifacts = append(response.Artifacts, proto.Clone(store.artifacts[identifier-1]).(*pb.Artifact))
	}
	if store.reverseResults {
		slices.Reverse(response.Artifacts)
	}
	return response, nil
}

func (store *finalizationMetadataStore) GetEventsByExecutionIDs(context.Context, *pb.GetEventsByExecutionIDsRequest, ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
	events := slices.Clone(store.artifactEvents)
	if store.reverseResults {
		slices.Reverse(events)
	}
	return &pb.GetEventsByExecutionIDsResponse{Events: events}, nil
}

func (store *finalizationMetadataStore) states() []pb.Execution_State {
	return []pb.Execution_State{
		store.executions[10].GetLastKnownState(),
		store.executions[20].GetLastKnownState(),
		store.executions[30].GetLastKnownState(),
	}
}

type finalizationCache struct {
	cacheutils.Client
	store        *finalizationMetadataStore
	tasks        []*api.Task
	statesAtCall [][]pb.Execution_State
	writeError   error
	responseLost bool
}

func (cache *finalizationCache) CreateExecutionCache(_ context.Context, task *api.Task) error {
	cache.statesAtCall = append(cache.statesAtCall, cache.store.states())
	if cache.writeError == nil || cache.responseLost {
		cache.tasks = append(cache.tasks, proto.Clone(task).(*api.Task))
	}
	return cache.writeError
}

type finalizationPlugin struct {
	plugins.TaskPluginHandler
	statuses []string
}

func (*finalizationPlugin) Name() string { return "finalization-test" }
func (plugin *finalizationPlugin) OnTaskEnd(_ context.Context, information *plugins.TaskInfo) error {
	plugin.statuses = append(plugin.statuses, information.RunStatus)
	return nil
}

type finalizationPluginFactory struct {
	plugin *finalizationPlugin
}

func (*finalizationPluginFactory) Name() string    { return "finalization-test" }
func (*finalizationPluginFactory) IsEnabled() bool { return true }
func (factory *finalizationPluginFactory) Create() (plugins.TaskPluginHandler, error) {
	return factory.plugin, nil
}

type finalizationFixture struct {
	attempt         int
	store           *finalizationMetadataStore
	cache           *finalizationCache
	plugin          *finalizationPlugin
	directory       string
	invocationsPath string
	artifactURI     string
}

func newFinalizationFixture(testingContext *testing.T, cacheEnabled bool) *finalizationFixture {
	testingContext.Helper()
	store := &finalizationMetadataStore{executions: map[int64]*pb.Execution{}}
	for _, identifier := range []int64{10, 20, 30, 40} {
		properties := map[string]*pb.Value{"task_name": metadata.StringValue(strconv.FormatInt(identifier, 10))}
		if identifier != 40 {
			properties["parent_dag_id"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: identifier + 10}}
		}
		if identifier == 20 || identifier == 30 {
			properties["total_dag_tasks"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: 1}}
		}
		store.executions[identifier] = &pb.Execution{Id: proto.Int64(identifier), LastKnownState: pb.Execution_RUNNING.Enum(), CustomProperties: properties}
	}
	if cacheEnabled {
		store.executions[10].CustomProperties["cache_fingerprint"] = metadata.StringValue("finalization-fingerprint")
	}
	plugin := &finalizationPlugin{}
	originalFactories := plugins.RegisteredFactories()
	plugins.ResetRegistry()
	plugins.RegisterHandlerFactory(&finalizationPluginFactory{plugin: plugin})
	testingContext.Cleanup(func() {
		plugins.ResetRegistry()
		for _, factory := range originalFactories {
			plugins.RegisterHandlerFactory(factory)
		}
	})
	directory := testingContext.TempDir()
	return &finalizationFixture{
		store: store, cache: &finalizationCache{store: store}, plugin: plugin,
		directory: directory, invocationsPath: filepath.Join(directory, "invocations"),
	}
}

func (fixture *finalizationFixture) execute(testingContext *testing.T, applicationExitCode int) (string, error) {
	testingContext.Helper()
	fixture.attempt++
	podUID := "test-pod-" + strconv.Itoa(fixture.attempt)
	outputPath := filepath.Join(fixture.directory, "executor-output.json")
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{}, Outputs: &pipelinespec.ExecutorInput_Outputs{OutputFile: outputPath},
	}
	artifactPath := filepath.Join(fixture.directory, "artifact")
	if fixture.artifactURI != "" {
		executorInput.Outputs.Artifacts = map[string]*pipelinespec.ArtifactList{
			"result": {Artifacts: []*pipelinespec.RuntimeArtifact{{
				Uri: fixture.artifactURI, CustomPath: &artifactPath,
				Type: &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"}},
			}}},
		}
	}
	executorInputJSON, marshalError := protojson.Marshal(executorInput)
	require.NoError(testingContext, marshalError)
	launcher, creationError := component.NewLauncherV2(
		context.Background(), 10, string(executorInputJSON), "{}",
		[]string{"sh", "-c", `printf '{"parameterValues":{"result":"written"}}\n' > "$1"; printf 'executed\n' >> "$2"; printf 'artifact\n' > "$4"; exit "$3"`, "application", outputPath, fixture.invocationsPath, strconv.Itoa(applicationExitCode), artifactPath},
		&component.LauncherV2Options{
			Namespace: "test", PodName: "test-launcher", PodUID: podUID, PipelineName: "test-pipeline", RunID: "test-run",
			MLMDServerAddress: "unused", MLMDServerPort: "8080", PublishLogs: "false",
		},
		client_manager.NewFakeClientManager(fake.NewSimpleClientset(), metadata.NewTestClient(fixture.store), fixture.cache),
	)
	require.NoError(testingContext, creationError)
	originalStderr := os.Stderr
	originalLogToStderr := flag.Lookup("logtostderr").Value.String()
	require.NoError(testingContext, flag.Set("logtostderr", "true"))
	logPath := filepath.Join(fixture.directory, "launcher.log")
	logFile, openError := os.Create(logPath)
	require.NoError(testingContext, openError)
	os.Stderr = logFile
	defer func() {
		os.Stderr = originalStderr
		_ = flag.Set("logtostderr", originalLogToStderr)
		_ = logFile.Close()
	}()
	executionResult := launcher.Execute(context.Background())
	glog.Flush()
	logs, readError := os.ReadFile(logPath)
	require.NoError(testingContext, readError)
	return string(logs), executionResult
}

func TestLauncherV2ExecuteFinalizationPanicSkipsCache(t *testing.T) {
	fixture := newFinalizationFixture(t, true)
	fixture.artifactURI = "mem://finalization-test/root/result"
	fixture.store.panicRecordingArtifact = true
	require.PanicsWithValue(t, "record artifact", func() { fixture.execute(t, 0) })
	assert.Equal(t, []pb.Execution_State{pb.Execution_FAILED, pb.Execution_FAILED, pb.Execution_FAILED}, fixture.store.states())
	assert.Empty(t, fixture.cache.statesAtCall)
}

func TestLauncherV2ExecuteFinalizationPreservesOutputs(testingContext *testing.T) {
	for _, publicationResponseLost := range []bool{false, true} {
		testingContext.Run("publication response lost="+strconv.FormatBool(publicationResponseLost), func(testingContext *testing.T) {
			fixture := newFinalizationFixture(testingContext, true)
			fixture.artifactURI = "mem://finalization-test/root/result"
			finalizationFailure := errors.New("metadata unavailable")
			if publicationResponseLost {
				fixture.store.publicationFailures = []error{finalizationFailure}
				fixture.store.publicationResponseLost = true
			} else {
				fixture.store.ancestorUpdateFailures = []error{finalizationFailure}
			}
			_, executionResult := fixture.execute(testingContext, 0)
			require.ErrorIs(testingContext, executionResult, finalizationFailure)
			assert.Equal(testingContext, []pb.Execution_State{pb.Execution_FAILED, pb.Execution_FAILED, pb.Execution_FAILED}, fixture.store.states())
			require.Len(testingContext, fixture.store.publicationRequests, 2)
			assert.Len(testingContext, fixture.store.publicationRequests[0].GetArtifactEventPairs(), 1)
			assert.Empty(testingContext, fixture.store.publicationRequests[1].GetArtifactEventPairs(), "state reconciliation must not replay artifact events")
			assert.Len(testingContext, fixture.store.artifacts, 1)
			assert.Len(testingContext, fixture.store.artifactEvents, 1)
			assert.Equal(testingContext, "written", fixture.store.executions[10].GetCustomProperties()["outputs"].GetStructValue().GetFields()["result"].GetStringValue())
			assert.Empty(testingContext, fixture.cache.tasks)
		})
	}
}

func TestLauncherV2ExecuteFinalization(testingContext *testing.T) {
	publicationFailure := errors.New("publication unavailable")
	lookupFailure := errors.New("DAG lookup unavailable")
	updateFailure := errors.New("ancestor update unavailable")
	reconciliationFailure := errors.New("reconciliation unavailable")
	cacheFailure := errors.New("cache unavailable")

	for _, scenario := range []struct {
		name                    string
		applicationExitCode     int
		publicationFailures     []error
		dagLookupFailures       []error
		ancestorUpdateFailures  []error
		publicationResponseLost bool
		cacheDisabled           bool
		cacheFailure            error
		cacheResponseLost       bool
		expectedErrors          []error
		expectedStates          []pb.Execution_State
	}{
		{name: "successful execution"},
		{name: "cache disabled", cacheDisabled: true},
		{name: "application failure", applicationExitCode: 7},
		{name: "publication failure", publicationFailures: []error{publicationFailure}, expectedErrors: []error{publicationFailure}},
		{name: "DAG lookup failure", dagLookupFailures: []error{lookupFailure}, expectedErrors: []error{lookupFailure}},
		{name: "ancestor update failure", ancestorUpdateFailures: []error{updateFailure}, expectedErrors: []error{updateFailure}},
		{name: "application and publication failures", applicationExitCode: 7, publicationFailures: []error{publicationFailure}, expectedErrors: []error{publicationFailure}},
		{name: "application and DAG lookup failures", applicationExitCode: 7, dagLookupFailures: []error{lookupFailure}, expectedErrors: []error{lookupFailure}},
		{name: "application and ancestor update failures", applicationExitCode: 7, ancestorUpdateFailures: []error{updateFailure}, expectedErrors: []error{updateFailure}},
		{name: "publication response lost", publicationFailures: []error{publicationFailure}, publicationResponseLost: true, expectedErrors: []error{publicationFailure}},
		{name: "failed repair response lost", publicationFailures: []error{nil, reconciliationFailure}, publicationResponseLost: true, ancestorUpdateFailures: []error{updateFailure}, expectedErrors: []error{updateFailure, reconciliationFailure}},
		{name: "failure publication unavailable", publicationFailures: []error{publicationFailure, reconciliationFailure}, expectedErrors: []error{publicationFailure, reconciliationFailure}, expectedStates: []pb.Execution_State{pb.Execution_RUNNING, pb.Execution_RUNNING, pb.Execution_RUNNING}},
		{name: "DAG recovery unavailable", dagLookupFailures: []error{lookupFailure, reconciliationFailure}, expectedErrors: []error{lookupFailure, reconciliationFailure}, expectedStates: []pb.Execution_State{pb.Execution_FAILED, pb.Execution_RUNNING, pb.Execution_RUNNING}},
		{name: "ancestor recovery unavailable", ancestorUpdateFailures: []error{updateFailure, reconciliationFailure}, expectedErrors: []error{updateFailure, reconciliationFailure}, expectedStates: []pb.Execution_State{pb.Execution_FAILED, pb.Execution_FAILED, pb.Execution_RUNNING}},
		{name: "cache registration failure", cacheFailure: cacheFailure},
		{name: "cache response lost", cacheFailure: cacheFailure, cacheResponseLost: true},
	} {
		testingContext.Run(scenario.name, func(testingContext *testing.T) {
			fixture := newFinalizationFixture(testingContext, !scenario.cacheDisabled)
			fixture.store.publicationFailures = append([]error(nil), scenario.publicationFailures...)
			fixture.store.dagLookupFailures = append([]error(nil), scenario.dagLookupFailures...)
			fixture.store.ancestorUpdateFailures = append([]error(nil), scenario.ancestorUpdateFailures...)
			fixture.store.publicationResponseLost = scenario.publicationResponseLost
			fixture.cache.writeError = scenario.cacheFailure
			fixture.cache.responseLost = scenario.cacheResponseLost
			logs, executionResult := fixture.execute(testingContext, scenario.applicationExitCode)
			expectedStatus := pb.Execution_COMPLETE
			if scenario.applicationExitCode != 0 || len(scenario.expectedErrors) != 0 {
				expectedStatus = pb.Execution_FAILED
				assert.Error(testingContext, executionResult)
			} else {
				assert.NoError(testingContext, executionResult)
			}
			for _, expectedError := range scenario.expectedErrors {
				assert.ErrorIs(testingContext, executionResult, expectedError)
			}
			if scenario.applicationExitCode != 0 {
				var applicationError *exec.ExitError
				if assert.ErrorAs(testingContext, executionResult, &applicationError) {
					assert.Equal(testingContext, scenario.applicationExitCode, applicationError.ExitCode())
				}
			}
			expectedStates := scenario.expectedStates
			if expectedStates == nil {
				expectedStates = []pb.Execution_State{expectedStatus, expectedStatus, expectedStatus}
			}
			assert.Equal(testingContext, expectedStates, fixture.store.states(), "task, parent DAG, ancestor DAG")
			assert.Equal(testingContext, pb.Execution_RUNNING, fixture.store.executions[40].GetLastKnownState(), "root DAG remains owned by the workflow")
			assert.Equal(testingContext, []string{expectedStatus.String()}, fixture.plugin.statuses)
			assert.Equal(testingContext, 1, fixture.store.taskPipelineReads, "reuse the pipeline already loaded by GetExecution")
			assert.Equal(testingContext, (len(scenario.publicationFailures) == 0 || scenario.publicationFailures[0] == nil), strings.Contains(logs, "publish success."), logs)
			invocations, readError := os.ReadFile(fixture.invocationsPath)
			require.NoError(testingContext, readError)
			assert.Equal(testingContext, "executed\n", string(invocations))
			if expectedStatus == pb.Execution_FAILED || scenario.cacheDisabled {
				assert.Empty(testingContext, fixture.cache.statesAtCall, "failed finalization must never register cache entries")
			} else {
				assert.Equal(testingContext, [][]pb.Execution_State{{pb.Execution_COMPLETE, pb.Execution_COMPLETE, pb.Execution_COMPLETE}}, fixture.cache.statesAtCall)
			}
			if expectedStatus == pb.Execution_COMPLETE && !scenario.cacheDisabled && (scenario.cacheFailure == nil || scenario.cacheResponseLost) {
				require.Len(testingContext, fixture.cache.tasks, 1)
				assert.Equal(testingContext, "10", fixture.cache.tasks[0].GetMlmdExecutionID())
			} else {
				assert.Empty(testingContext, fixture.cache.tasks)
			}
		})
	}
}

func TestLauncherV2ExecuteFinalizationRetry(testingContext *testing.T) {
	for _, scenario := range []struct {
		name          string
		exitCode      int
		expectedState pb.Execution_State
	}{
		{name: "successful retry", expectedState: pb.Execution_COMPLETE},
		{name: "failed retry", exitCode: 7, expectedState: pb.Execution_FAILED},
	} {
		testingContext.Run(scenario.name, func(testingContext *testing.T) {
			fixture := newFinalizationFixture(testingContext, true)
			updateFailure := errors.New("ancestor update unavailable")
			fixture.store.ancestorUpdateFailures = []error{updateFailure}
			fixture.artifactURI = "mem://finalization-test/root/first"
			_, firstResult := fixture.execute(testingContext, 0)
			require.ErrorIs(testingContext, firstResult, updateFailure)
			assert.Empty(testingContext, fixture.cache.tasks)
			assert.Equal(testingContext, []pb.Execution_State{pb.Execution_FAILED, pb.Execution_FAILED, pb.Execution_FAILED}, fixture.store.states())

			fixture.artifactURI = "mem://finalization-test/root/retry"
			_, retryResult := fixture.execute(testingContext, scenario.exitCode)
			if scenario.exitCode == 0 {
				require.NoError(testingContext, retryResult)
				require.Len(testingContext, fixture.cache.tasks, 1)
				require.Len(testingContext, fixture.store.artifactEvents, 2)
				for _, reverse := range []bool{false, true} {
					fixture.store.reverseResults = reverse
					outputs, readErr := metadata.NewTestClient(fixture.store).GetOutputArtifactsByExecutionId(context.Background(), 10)
					require.NoError(testingContext, readErr)
					require.Contains(testingContext, outputs, "result")
					assert.Equal(testingContext, fixture.artifactURI, outputs["result"].Artifact.GetUri())
				}
				assert.Equal(testingContext, [][]pb.Execution_State{{pb.Execution_COMPLETE, pb.Execution_COMPLETE, pb.Execution_COMPLETE}}, fixture.cache.statesAtCall)
			} else {
				require.Error(testingContext, retryResult)
				assert.Empty(testingContext, fixture.cache.tasks)
			}
			assert.Equal(testingContext, []pb.Execution_State{scenario.expectedState, scenario.expectedState, scenario.expectedState}, fixture.store.states())
			assert.Equal(testingContext, []string{"FAILED", scenario.expectedState.String()}, fixture.plugin.statuses)
			invocations, readError := os.ReadFile(fixture.invocationsPath)
			require.NoError(testingContext, readError)
			assert.Equal(testingContext, "executed\nexecuted\n", string(invocations), "one application invocation per workflow attempt")
		})
	}
}
