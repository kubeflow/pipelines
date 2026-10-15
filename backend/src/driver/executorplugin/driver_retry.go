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
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
)

type driverRetryOptions struct {
	enabled           bool
	attempt, maxCount int
	generation        int64
}

func parseDriverRetry(args driverapi.DriverPluginArgs) (driverRetryOptions, error) {
	var result driverRetryOptions
	var err error
	if args.DriverRetryEnabled != "" {
		result.enabled, err = strconv.ParseBool(args.DriverRetryEnabled)
		if err != nil {
			return result, fmt.Errorf("invalid driver_retry_enabled: %w", err)
		}
	}
	for _, field := range []struct {
		name, value string
		target      *int
	}{{"driver_retry_attempt", args.DriverRetryAttempt, &result.attempt}, {"driver_retry_max_count", args.DriverRetryMaxCount, &result.maxCount}} {
		if field.value == "" {
			continue
		}
		*field.target, err = strconv.Atoi(field.value)
		if err != nil || *field.target < 0 {
			return result, fmt.Errorf("%s must be a nonnegative integer", field.name)
		}
	}
	if args.DriverRetryGeneration != "" {
		result.generation, err = strconv.ParseInt(args.DriverRetryGeneration, 10, 64)
		if err != nil || result.generation < 0 {
			return result, fmt.Errorf("driver_retry_generation must be a nonnegative integer")
		}
	}
	return result, nil
}

func pluginDispatcherForDriver(runtimeArgs driverapi.RuntimeArgs, retryEnabled bool) (plugins.TaskPluginDispatcher, error) {
	dispatcher, err := plugins.GetPluginDispatcherWithRuntimeArgs(runtimeArgs)
	if err != nil {
		if retryEnabled {
			return nil, fmt.Errorf("cannot initialize plugins for driver retries: %w", err)
		}
		glog.Errorf("Failed to initialize plugin dispatcher: %v", err)
		return plugins.NoOpDispatcher{}, nil
	}
	if retryEnabled {
		if err := plugins.ValidateDriverRetry(dispatcher); err != nil {
			return nil, err
		}
	}
	return dispatcher, nil
}

type stoppedDriverFinalizer interface {
	FinalizeStoppedDriver(context.Context, string, int64, string, string, *int64) error
}

func validateDriverFinalization(args driverapi.DriverPluginArgs, generation int64) error {
	if args.DriverRetryStatus != "Failed" && args.DriverRetryStatus != "Error" {
		return fmt.Errorf("driver finalization requires a completed Failed or Error retry node")
	}
	if args.RunID == "" || args.TaskName == "" || args.ParentTaskID == "" {
		return fmt.Errorf("driver finalization requires run, task name, and parent task IDs")
	}
	iteration, err := strconv.Atoi(args.IterationIndex)
	if err != nil || generation < 0 || iteration < -1 {
		return fmt.Errorf("driver finalization requires a non-negative generation and iteration index of at least -1")
	}
	return nil
}

func finalizeStoppedDriver(ctx context.Context, api stoppedDriverFinalizer, args driverapi.DriverPluginArgs, generation int64) error {
	if err := validateDriverFinalization(args, generation); err != nil {
		return err
	}
	index, _ := strconv.ParseInt(args.IterationIndex, 10, 64)
	var iteration *int64
	if index >= 0 {
		iteration = &index
	}
	return api.FinalizeStoppedDriver(ctx, args.RunID, generation, args.TaskName, args.ParentTaskID, iteration)
}
