// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"errors"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

type recordingStoppedDriverFinalizer struct {
	calls             int
	run, task, parent string
	generation        int64
	iteration         *int64
	err               error
}

func (f *recordingStoppedDriverFinalizer) FinalizeStoppedDriver(_ context.Context, run string, generation int64, task, parent string, iteration *int64) error {
	f.calls++
	f.run, f.generation, f.task, f.parent, f.iteration = run, generation, task, parent, iteration
	return f.err
}
func finalizerArgs() driverapi.DriverPluginArgs {
	return driverapi.DriverPluginArgs{DriverRetryEnabled: "true", DriverRetryFinalize: true, DriverRetryStatus: "Failed", DriverRetryGeneration: "4", IterationIndex: "-1", RunID: "run", TaskName: "task", ParentTaskID: "parent", LogLevel: "1"}
}
func TestFinalizeStoppedDriver(t *testing.T) {
	for _, phase := range []string{"Failed", "Error"} {
		for _, iteration := range []int{-1, 0, 7} {
			args := finalizerArgs()
			args.DriverRetryStatus = phase
			args.IterationIndex = strconv.Itoa(iteration)
			args.DriverRetryMaxCount = "10"
			api := &recordingStoppedDriverFinalizer{}
			require.NoError(t, finalizeStoppedDriver(context.Background(), api, args, 4))
			require.Equal(t, 1, api.calls)
			require.Equal(t, "run", api.run)
			require.Equal(t, "task", api.task)
			require.Equal(t, "parent", api.parent)
			require.Equal(t, int64(4), api.generation)
			if iteration < 0 {
				require.Nil(t, api.iteration)
			} else {
				require.NotNil(t, api.iteration)
				require.Equal(t, int64(iteration), *api.iteration)
			}
		}
	}
}
func TestFinalizeStoppedDriverRejectsInvalidIdentityOrPhase(t *testing.T) {
	for name, change := range map[string]func(*driverapi.DriverPluginArgs){
		"succeeded":         func(a *driverapi.DriverPluginArgs) { a.DriverRetryStatus = "Succeeded" },
		"running":           func(a *driverapi.DriverPluginArgs) { a.DriverRetryStatus = "Running" },
		"unknown":           func(a *driverapi.DriverPluginArgs) { a.DriverRetryStatus = "" },
		"missing run":       func(a *driverapi.DriverPluginArgs) { a.RunID = "" },
		"missing task":      func(a *driverapi.DriverPluginArgs) { a.TaskName = "" },
		"missing parent":    func(a *driverapi.DriverPluginArgs) { a.ParentTaskID = "" },
		"invalid iteration": func(a *driverapi.DriverPluginArgs) { a.IterationIndex = "-2" },
	} {
		t.Run(name, func(t *testing.T) {
			args := finalizerArgs()
			change(&args)
			api := &recordingStoppedDriverFinalizer{}
			require.Error(t, finalizeStoppedDriver(context.Background(), api, args, 4))
			require.Zero(t, api.calls)
		})
	}
	api := &recordingStoppedDriverFinalizer{}
	require.Error(t, finalizeStoppedDriver(context.Background(), api, finalizerArgs(), -1))
	require.Zero(t, api.calls)
}
func TestFinalizeStoppedDriverReturnsAPIError(t *testing.T) {
	expected := errors.New("unavailable")
	api := &recordingStoppedDriverFinalizer{err: expected}
	require.ErrorIs(t, finalizeStoppedDriver(context.Background(), api, finalizerArgs(), 4), expected)
}
func TestDisabledFinalizerDoesNotInitializeOrExecuteDriver(t *testing.T) {
	args := finalizerArgs()
	args.DriverRetryEnabled = "false"
	args.RunID = ""
	runner := &driverRunner{logDirectory: t.TempDir()}
	result, err := runner.drive(context.Background(), args)
	require.NoError(t, err)
	require.NotNil(t, result)
}
func TestDriverRetryArgumentParsing(t *testing.T) {
	args := finalizerArgs()
	args.DriverRetryAttempt = "2"
	args.DriverRetryMaxCount = "10"
	parsed, err := parseDriverRetry(args)
	require.NoError(t, err)
	require.True(t, parsed.enabled)
	require.Equal(t, 2, parsed.attempt)
	require.Equal(t, 10, parsed.maxCount)
	require.Equal(t, int64(4), parsed.generation)
	for _, change := range []func(*driverapi.DriverPluginArgs){func(a *driverapi.DriverPluginArgs) { a.DriverRetryEnabled = "invalid" }, func(a *driverapi.DriverPluginArgs) { a.DriverRetryAttempt = "-1" }, func(a *driverapi.DriverPluginArgs) { a.DriverRetryMaxCount = "not-a-number" }, func(a *driverapi.DriverPluginArgs) { a.DriverRetryGeneration = "-1" }} {
		bad := args
		change(&bad)
		_, err := parseDriverRetry(bad)
		require.Error(t, err)
	}
}

type finalizingInvocationAPI struct {
	*invocationAPI
	finalizer recordingStoppedDriverFinalizer
}

func (a *finalizingInvocationAPI) FinalizeStoppedDriver(ctx context.Context, run string, generation int64, task, parent string, iteration *int64) error {
	return a.finalizer.FinalizeStoppedDriver(ctx, run, generation, task, parent, iteration)
}
func TestFinalizerInvocationBypassesTaskExecutionAndClosesClients(t *testing.T) {
	runner, args, api, manager := invocationFixture(t, driverapi.ContainerDriver)
	args.DriverRetryFinalize = true
	args.DriverRetryEnabled = "true"
	args.DriverRetryStatus = "Error"
	args.DriverRetryGeneration = "4"
	api.getRunErr = errors.New("finalizer must not fetch run or resolve specs")
	wrapped := &finalizingInvocationAPI{invocationAPI: api}
	manager.api = wrapped
	closed := false
	manager.close = func() error { closed = true; return nil }
	runner.execute = nil
	result, err := runner.drive(context.Background(), args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, wrapped.finalizer.calls)
	require.True(t, closed)
}
func TestRetryInvocationPreservesRetryAndExitOptions(t *testing.T) {
	runner, args, _, _ := invocationFixture(t, driverapi.ContainerDriver)
	args.DriverRetryEnabled = "true"
	args.DriverRetryAttempt = "2"
	args.DriverRetryMaxCount = "10"
	args.DriverRetryGeneration = "4"
	args.ExitTaskName = "upstream"
	args.ExitTaskStatus = "Failed"
	args.PublishLogs = "false"
	called := false
	runner.execute = func(_ context.Context, options drivercommon.Options, _ client_manager.ClientManagerInterface) (*driver.Execution, error) {
		called = true
		require.True(t, options.DriverRetryEnabled)
		require.Equal(t, 2, options.DriverRetryAttempt)
		require.Equal(t, 10, options.DriverRetryMaxCount)
		require.Equal(t, int64(4), options.DriverRetryGeneration)
		require.Equal(t, "upstream", options.ExitTaskName)
		require.Equal(t, "Failed", options.ExitTaskStatus)
		return &driver.Execution{}, nil
	}
	_, err := runner.drive(context.Background(), args)
	require.NoError(t, err)
	require.True(t, called)
}
