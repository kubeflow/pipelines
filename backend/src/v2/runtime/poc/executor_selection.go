package poc

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func unfinishedRun(state model.RuntimeState) bool {
	switch state {
	case model.RuntimeStatePending, model.RuntimeStateRunning, model.RuntimeStateUnspecified, model.RuntimeStatePaused:
		return true
	default:
		return false
	}
}

func (c *Coordinator) reconcileExecutorSelections(ctx context.Context, configured string) error {
	runs, _, _, err := c.runStore.ListRuns(&model.FilterContext{}, list.EmptyOptions(), false)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if !IsManagedRun(run) || !unfinishedRun(run.State) || run.ExecutorType == nil || *run.ExecutorType == configured {
			continue
		}
		fail := func(owned *Coordinator) error {
			return owned.failExecutorMismatch(run.UUID, configured)
		}
		if c.leaseStore != nil && *run.ExecutorType == ExecutorKubernetes {
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = c.runOwnedMaintenance(bounded, run.UUID, fail)
			cancel()
		} else {
			err = fail(c)
		}
		if err != nil {
			return fmt.Errorf("executor-mismatch for run %s: recorded %q, configured %q; restore the recorded executor or resolve its ownership: %w", run.UUID, *run.ExecutorType, configured, err)
		}
	}
	return nil
}

func (c *Coordinator) failExecutorMismatch(runID, configured string) error {
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if !unfinishedRun(run.State) || run.ExecutorType == nil || *run.ExecutorType == configured {
		return nil
	}
	message := fmt.Sprintf("executor-mismatch: run requires %q but the worker is configured with %q; computation was not resubmitted", *run.ExecutorType, configured)
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil {
		return err
	}
	if manifest == nil {
		return fmt.Errorf("run %s has no durable runtime manifest", run.UUID)
	}
	return c.failBeforeDispatch(run, manifest, message)
}
