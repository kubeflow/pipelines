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
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/v2/driver/driverflags"
)

type stoppedDriverFinalizer interface {
	FinalizeStoppedDriver(context.Context, string, int64, string, string, *int64) error
}

func validateDriverFinalization(values *driverflags.Values) error {
	if *values.DriverRetryStatus != "Failed" && *values.DriverRetryStatus != "Error" {
		return fmt.Errorf("driver finalization requires a completed Failed or Error retry node")
	}
	if *values.RunID == "" || *values.TaskName == "" || *values.ParentTaskID == "" {
		return fmt.Errorf("driver finalization requires run, task name, and parent task IDs")
	}
	if *values.DriverRetryGeneration < 0 || *values.IterationIndex < -1 {
		return fmt.Errorf("driver finalization requires a non-negative generation and iteration index of at least -1")
	}
	return nil
}

func finalizeStoppedDriver(ctx context.Context, api stoppedDriverFinalizer, values *driverflags.Values) error {
	if !*values.DriverRetryEnabled {
		return nil
	}
	if err := validateDriverFinalization(values); err != nil {
		return err
	}
	var iteration *int64
	if *values.IterationIndex >= 0 {
		index := int64(*values.IterationIndex)
		iteration = &index
	}
	return api.FinalizeStoppedDriver(ctx, *values.RunID, *values.DriverRetryGeneration, *values.TaskName, *values.ParentTaskID, iteration)
}
