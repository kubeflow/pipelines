// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func createRuns(store *RunStore, runs ...*model.Run) error {
	for _, run := range runs {
		if _, err := store.CreateRun(run); err != nil {
			return fmt.Errorf("failed to create run: %v", err)
		}
	}
	return nil
}

func createMetrics(store *RunStore, metrics ...*model.RunMetric) error {
	for _, metric := range metrics {
		if err := store.CreateMetric(metric); err != nil {
			return fmt.Errorf("failed to create metric: %v", err)
		}
	}
	return nil
}
