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
	"strconv"
	"strings"
	"testing"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "gocloud.dev/blob/memblob"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

// Supply a pipeline root through the real metadata reader without a server.
type finalizationMetadataStore struct {
	pb.MetadataStoreServiceClient
}

func (*finalizationMetadataStore) GetExecutionsByID(context.Context, *pb.GetExecutionsByIDRequest, ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
	return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{
		Id: proto.Int64(10),
		CustomProperties: map[string]*pb.Value{
			"parent_dag_id": {Value: &pb.Value_IntValue{IntValue: 20}},
		},
	}}}, nil
}

func (*finalizationMetadataStore) GetContextType(_ context.Context, request *pb.GetContextTypeRequest, _ ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
	identifier := int64(1)
	if request.GetTypeName() == "system.PipelineRun" {
		identifier = 2
	}
	return &pb.GetContextTypeResponse{ContextType: &pb.ContextType{Id: &identifier}}, nil
}

func (*finalizationMetadataStore) GetContextsByExecution(context.Context, *pb.GetContextsByExecutionRequest, ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	return &pb.GetContextsByExecutionResponse{Contexts: []*pb.Context{
		{Id: proto.Int64(1), TypeId: proto.Int64(1)},
		{Id: proto.Int64(2), TypeId: proto.Int64(2), CustomProperties: map[string]*pb.Value{
			"pipeline_root": {Value: &pb.Value_StringValue{StringValue: "mem://finalization-test/root"}},
		}},
	}}, nil
}

type finalizationMetadataClient struct {
	metadata.ClientInterface
	execution           *metadata.Execution
	publicationError    error
	dagLookupError      error
	pipelineLookupError error
	updateError         error
	calls               []string
	publishedState      pb.Execution_State
}

func (client *finalizationMetadataClient) GetExecution(context.Context, int64) (*metadata.Execution, error) {
	return client.execution, nil
}

func (client *finalizationMetadataClient) PrePublishExecution(context.Context, *metadata.Execution, *metadata.ExecutionConfig) (*metadata.Execution, error) {
	return client.execution, nil
}

func (client *finalizationMetadataClient) PublishExecution(_ context.Context, _ *metadata.Execution, _ map[string]*structpb.Value, _ []*metadata.OutputArtifact, state pb.Execution_State) error {
	client.calls = append(client.calls, "publish")
	client.publishedState = state
	return client.publicationError
}

func (client *finalizationMetadataClient) GetDAG(context.Context, int64) (*metadata.DAG, error) {
	client.calls = append(client.calls, "get DAG")
	if client.dagLookupError != nil {
		return nil, client.dagLookupError
	}
	return &metadata.DAG{Execution: metadata.NewExecution(&pb.Execution{Id: proto.Int64(20)})}, nil
}

func (client *finalizationMetadataClient) GetPipelineFromExecution(context.Context, int64) (*metadata.Pipeline, error) {
	client.calls = append(client.calls, "get pipeline")
	if client.pipelineLookupError != nil {
		return nil, client.pipelineLookupError
	}
	return client.execution.GetPipeline(), nil
}

func (client *finalizationMetadataClient) UpdateDAGExecutionsState(ctx context.Context, dag *metadata.DAG, pipeline *metadata.Pipeline) error {
	client.calls = append(client.calls, "update DAG")
	if client.updateError != nil {
		return client.updateError
	}
	// The real method skips root DAGs and reproduces the nil-DAG panic.
	return metadata.NewTestClient(nil).UpdateDAGExecutionsState(ctx, dag, pipeline)
}

func TestLauncherV2ExecuteFinalization(t *testing.T) {
	publicationFailure := errors.New("publication unavailable")
	dagLookupFailure := errors.New("DAG lookup unavailable")
	pipelineLookupFailure := errors.New("pipeline lookup unavailable")
	updateFailure := errors.New("DAG update unavailable")

	for _, scenario := range []struct {
		name                string
		applicationExitCode int
		publicationError    error
		dagLookupError      error
		pipelineLookupError error
		updateError         error
	}{
		{name: "successful execution"},
		{name: "application failure", applicationExitCode: 7},
		{name: "publication failure", publicationError: publicationFailure},
		{name: "DAG lookup failure", dagLookupError: dagLookupFailure},
		{name: "pipeline lookup failure", pipelineLookupError: pipelineLookupFailure},
		{name: "DAG update failure", updateError: updateFailure},
		{name: "application and publication failures", applicationExitCode: 7, publicationError: publicationFailure},
		{name: "application publication and DAG lookup failures", applicationExitCode: 7, publicationError: publicationFailure, dagLookupError: dagLookupFailure},
		{name: "application and pipeline lookup failures", applicationExitCode: 7, pipelineLookupError: pipelineLookupFailure},
		{name: "application and DAG update failures", applicationExitCode: 7, updateError: updateFailure},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx := context.Background()
			execution, executionError := metadata.NewTestClient(&finalizationMetadataStore{}).GetExecution(ctx, 10)
			require.NoError(t, executionError)
			client := &finalizationMetadataClient{
				execution:           execution,
				publicationError:    scenario.publicationError,
				dagLookupError:      scenario.dagLookupError,
				pipelineLookupError: scenario.pipelineLookupError,
				updateError:         scenario.updateError,
			}
			directory := t.TempDir()
			outputPath := filepath.Join(directory, "executor-output.json")
			invocationsPath := filepath.Join(directory, "invocations")
			executorInput := &pipelinespec.ExecutorInput{
				Inputs:  &pipelinespec.ExecutorInput_Inputs{},
				Outputs: &pipelinespec.ExecutorInput_Outputs{OutputFile: outputPath},
			}
			executorInputJSON, marshalError := protojson.Marshal(executorInput)
			require.NoError(t, marshalError)
			launcher, creationError := component.NewLauncherV2(
				ctx, 10, string(executorInputJSON), "{}",
				[]string{"sh", "-c", `printf '{}\n' > "$1"; printf 'executed\n' >> "$2"; exit "$3"`, "application", outputPath, invocationsPath, strconv.Itoa(scenario.applicationExitCode)},
				&component.LauncherV2Options{
					Namespace: "test", PodName: "test-launcher", PodUID: "test-pod",
					MLMDServerAddress: "unused", MLMDServerPort: "8080", PublishLogs: "false",
				},
				client_manager.NewFakeClientManager(fake.NewSimpleClientset(), client, nil),
			)
			require.NoError(t, creationError)

			// Capture publication logging as well as the returned error.
			originalStderr := os.Stderr
			originalLogToStderr := flag.Lookup("logtostderr").Value.String()
			require.NoError(t, flag.Set("logtostderr", "true"))
			logPath := filepath.Join(directory, "launcher.log")
			logFile, openError := os.Create(logPath)
			require.NoError(t, openError)
			os.Stderr = logFile
			t.Cleanup(func() {
				os.Stderr = originalStderr
				_ = flag.Set("logtostderr", originalLogToStderr)
				_ = logFile.Close()
			})
			var executionResult error
			require.NotPanics(t, func() { executionResult = launcher.Execute(ctx) })
			glog.Flush()
			os.Stderr = originalStderr
			logs, readError := os.ReadFile(logPath)
			require.NoError(t, readError)
			assert.Equal(t, scenario.publicationError == nil, strings.Contains(string(logs), "publish success."), string(logs))

			expectedCalls := []string{"publish", "get DAG"}
			if scenario.dagLookupError == nil {
				expectedCalls = append(expectedCalls, "get pipeline")
				if scenario.pipelineLookupError == nil {
					expectedCalls = append(expectedCalls, "update DAG")
				}
			}
			assert.Equal(t, expectedCalls, client.calls)
			invocations, readError := os.ReadFile(invocationsPath)
			require.NoError(t, readError)
			assert.Equal(t, "executed\n", string(invocations))

			expectedState := pb.Execution_COMPLETE
			if scenario.applicationExitCode != 0 {
				expectedState = pb.Execution_FAILED
				var applicationError *exec.ExitError
				if assert.ErrorAs(t, executionResult, &applicationError) {
					assert.Equal(t, scenario.applicationExitCode, applicationError.ExitCode())
				}
			}
			assert.Equal(t, expectedState, client.publishedState)
			hasFailure := scenario.applicationExitCode != 0
			for _, expectedError := range []error{scenario.publicationError, scenario.dagLookupError, scenario.pipelineLookupError, scenario.updateError} {
				if expectedError != nil {
					hasFailure = true
					assert.ErrorIs(t, executionResult, expectedError)
				}
			}
			if hasFailure {
				assert.Error(t, executionResult)
			} else {
				assert.NoError(t, executionResult)
			}
		})
	}
}
