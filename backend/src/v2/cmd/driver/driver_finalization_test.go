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

package main

import (
	"context"
	"errors"
	"flag"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/driver/driverflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingStoppedDriverFinalizer struct {
	calls                  int
	runID                  string
	generation             int64
	taskName, parentTaskID string
	iteration              *int64
	err                    error
}

func (f *recordingStoppedDriverFinalizer) FinalizeStoppedDriver(_ context.Context, runID string, generation int64, taskName, parentTaskID string, iteration *int64) error {
	f.calls++
	f.runID, f.generation, f.taskName, f.parentTaskID, f.iteration = runID, generation, taskName, parentTaskID, iteration
	return f.err
}

func finalizerFlagValues(t *testing.T) *driverflags.Values {
	t.Helper()
	fs := flag.NewFlagSet("finalizer", flag.ContinueOnError)
	values := driverflags.RegisterDriverFlags(fs)
	require.NoError(t, fs.Parse([]string{
		"--driver_retry_enabled=true", "--driver_retry_finalize=true", "--driver_retry_status=Failed",
		"--run_id=run", "--task_name=task", "--parent_task_id=parent", "--driver_retry_generation=4",
	}))
	return values
}

func TestFinalizeStoppedDriver(t *testing.T) {
	for _, phase := range []string{"Failed", "Error"} {
		for _, iteration := range []int{-1, 0, 7} {
			values := finalizerFlagValues(t)
			*values.DriverRetryStatus, *values.IterationIndex = phase, iteration
			// Argo can stop before the attempt limit due to policy or deadline.
			*values.DriverRetryAttempt, *values.DriverRetryMaxCount = 0, 10
			api := &recordingStoppedDriverFinalizer{}
			require.NoError(t, finalizeStoppedDriver(context.Background(), api, values))
			assert.Equal(t, 1, api.calls)
			assert.Equal(t, "run", api.runID)
			assert.Equal(t, int64(4), api.generation)
			assert.Equal(t, "task", api.taskName)
			assert.Equal(t, "parent", api.parentTaskID)
			if iteration < 0 {
				assert.Nil(t, api.iteration)
			} else {
				require.NotNil(t, api.iteration)
				assert.Equal(t, int64(iteration), *api.iteration)
			}
		}
	}
}

func TestFinalizeStoppedDriverRejectsInvalidIdentityOrPhase(t *testing.T) {
	for name, change := range map[string]func(*driverflags.Values){
		"succeeded":           func(v *driverflags.Values) { *v.DriverRetryStatus = "Succeeded" },
		"running":             func(v *driverflags.Values) { *v.DriverRetryStatus = "Running" },
		"unknown":             func(v *driverflags.Values) { *v.DriverRetryStatus = "" },
		"missing run":         func(v *driverflags.Values) { *v.RunID = "" },
		"missing task":        func(v *driverflags.Values) { *v.TaskName = "" },
		"missing parent":      func(v *driverflags.Values) { *v.ParentTaskID = "" },
		"negative generation": func(v *driverflags.Values) { *v.DriverRetryGeneration = -1 },
		"invalid iteration":   func(v *driverflags.Values) { *v.IterationIndex = -2 },
	} {
		t.Run(name, func(t *testing.T) {
			values := finalizerFlagValues(t)
			change(values)
			api := &recordingStoppedDriverFinalizer{}
			require.Error(t, finalizeStoppedDriver(context.Background(), api, values))
			assert.Zero(t, api.calls)
		})
	}
}

func TestFinalizeStoppedDriverReturnsAPIError(t *testing.T) {
	expected := errors.New("unavailable")
	api := &recordingStoppedDriverFinalizer{err: expected}
	assert.ErrorIs(t, finalizeStoppedDriver(context.Background(), api, finalizerFlagValues(t)), expected)
}

func TestDisabledFinalizerDoesNotInitializeOrExecuteDriver(t *testing.T) {
	previous := driverFlagValues
	t.Cleanup(func() { driverFlagValues = previous })
	driverFlagValues = finalizerFlagValues(t)
	*driverFlagValues.DriverRetryEnabled = false
	// A disabled hook needs neither a valid task nor any initialized client.
	*driverFlagValues.RunID = ""
	require.NoError(t, finalizeStoppedDriver(context.Background(), nil, driverFlagValues))
	require.NoError(t, drive())
}
