// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"errors"
	"fmt"
)

// DeleteRun prevents a user retry from reopening a run between cleanup and deletion.
func (s *runtimeStore) DeleteRun(runID string, expectedGeneration int64) error {
	if runID == "" || expectedGeneration < 0 {
		return errors.New("managed run deletion requires a run ID and nonnegative retry generation")
	}
	if err := checkLeaseRun(s.tasks.lease, runID); err != nil {
		return err
	}
	tx, err := beginStoreMutation(s.tasks.db, s.tasks.lease)
	if err != nil {
		return fmt.Errorf("begin atomic managed run deletion: %w", err)
	}
	defer tx.Rollback()
	current, err := s.lockManagedRunLifecycle(tx, runID)
	if err != nil {
		return err
	}
	if !current.executor.Valid || current.executor.String == "" ||
		current.generation.Int64 != expectedGeneration || !terminalRunStateValues[string(current.effectiveState())] {
		return fmt.Errorf("%w: deletion requires a terminal managed run at generation %d; reload the run", ErrRuntimeTaskConflict, expectedGeneration)
	}
	if err := s.checkRunRuntimeCleanup(tx, runID); err != nil {
		return err
	}
	runs := NewRunStore(s.tasks.db, s.tasks.time, s.tasks.dbDialect)
	runs.lease, runs.taskStore = s.tasks.lease, s.tasks
	if err := runs.deleteRunInTransaction(tx, runID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit atomic managed run deletion: %w", err)
	}
	return nil
}
