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

package component

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"google.golang.org/protobuf/types/known/structpb"
)

// DebugPauseBarrier identifies which point in a task's lifecycle the launcher
// parks at. Pause is an overlay on the task's real state, never written into
// PipelineTask.State.
type DebugPauseBarrier string

const (
	DebugPauseBarrierBefore  DebugPauseBarrier = "before"
	DebugPauseBarrierAfter   DebugPauseBarrier = "after"
	DebugPauseBarrierOnError DebugPauseBarrier = "on_error"
	debugPauseBarrierNone    DebugPauseBarrier = ""
)

// Env vars set by the SDK's set_debug_pause(). KFP-owned names, distinct from
// Argo's ARGO_DEBUG_PAUSE_*: the launcher, not the emissary, owns pausing.
const (
	envKFPDebugPauseBefore  = "KFP_DEBUG_PAUSE_BEFORE"
	envKFPDebugPauseAfter   = "KFP_DEBUG_PAUSE_AFTER"
	envKFPDebugPauseOnError = "KFP_DEBUG_PAUSE_ON_ERROR"

	// envKFPDebugPauseMaxDuration optionally overrides the safety-valve
	// duration (Go duration string, e.g. "2h"). If unset or unparseable,
	// defaultDebugPauseMaxDuration is used.
	envKFPDebugPauseMaxDuration = "KFP_DEBUG_PAUSE_MAX_DURATION"
)

// custom_properties keys on PipelineTask.StatusMetadata - pause is a
// side-channel on the task record, not a PipelineTask_TaskState, so it never
// competes with the task's real outcome.
const (
	customPropDebugPauseBarrier         = "debug_pause_barrier"
	customPropDebugPauseResumeRequested = "debug_pause_resume_requested"
	// customPropDebugPauseInstance is a unique ID per Pause() call,
	// published with the barrier. It lets a publish retry distinguish
	// "my write committed" from an earlier pause epoch's residue.
	customPropDebugPauseInstance = "debug_pause_instance"
)

// defaultDebugPauseMaxDuration bounds the parked wait before the task fails;
// without it a missed resume would hold the pod (and node resources) forever.
// Override via envKFPDebugPauseMaxDuration.
const defaultDebugPauseMaxDuration = 1 * time.Hour

// defaultDebugPausePollInterval is the resume poll period while parked.
const defaultDebugPausePollInterval = 5 * time.Second

// DebugPauseConfig is parsed once from the task's env vars.
type DebugPauseConfig struct {
	Before  bool
	After   bool
	OnError bool

	// MaxDuration bounds the parked wait; PollInterval is the resume poll period.
	MaxDuration  time.Duration
	PollInterval time.Duration
}

// NewDebugPauseConfigFromEnv reads the KFP_DEBUG_PAUSE_* env vars set by
// set_debug_pause(). Absent or non-"true" values mean no pause.
func NewDebugPauseConfigFromEnv() DebugPauseConfig {
	cfg := DebugPauseConfig{
		Before:       os.Getenv(envKFPDebugPauseBefore) == "true",
		After:        os.Getenv(envKFPDebugPauseAfter) == "true",
		OnError:      os.Getenv(envKFPDebugPauseOnError) == "true",
		MaxDuration:  defaultDebugPauseMaxDuration,
		PollInterval: defaultDebugPausePollInterval,
	}
	if raw := os.Getenv(envKFPDebugPauseMaxDuration); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.MaxDuration = d
		} else {
			glog.Warningf("debug pause: invalid %s=%q, using default %s", envKFPDebugPauseMaxDuration, raw, defaultDebugPauseMaxDuration)
		}
	}
	return cfg
}

// Enabled reports whether any pause barrier is configured.
func (c DebugPauseConfig) Enabled() bool {
	return c.Before || c.After || c.OnError
}

// PauseState is the pause overlay persisted on the task record.
type PauseState struct {
	Barrier         DebugPauseBarrier
	Instance        string
	ResumeRequested bool
}

// PauseSignaler abstracts pause persistence so Pause can be unit-tested.
type PauseSignaler interface {
	// PublishBarrier records that the task is parked at the given barrier
	// by this pause instance. Implementations must merge into existing
	// StatusMetadata.CustomProperties: UpdateTask replaces StatusMetadata
	// wholesale.
	PublishBarrier(ctx context.Context, barrier DebugPauseBarrier, instance string) error

	// ReadPauseState returns the persisted pause overlay. ResumeRequested
	// is trustworthy only once the caller confirms Barrier and Instance
	// match its own pause epoch.
	ReadPauseState(ctx context.Context) (PauseState, error)

	// ClearBarrier removes the pause markers, best effort.
	ClearBarrier(ctx context.Context) error
}

// kfpAPIPauseSignaler implements PauseSignaler via the kfpapi.API client.
type kfpAPIPauseSignaler struct {
	client kfpapi.API
	runID  string
	taskID string
}

// NewKFPAPIPauseSignaler builds the production PauseSignaler for one task.
func NewKFPAPIPauseSignaler(client kfpapi.API, runID, taskID string) PauseSignaler {
	return &kfpAPIPauseSignaler{client: client, runID: runID, taskID: taskID}
}

func (s *kfpAPIPauseSignaler) PublishBarrier(ctx context.Context, barrier DebugPauseBarrier, instance string) error {
	// Reset the resume flag on every publish: ClearBarrier is best effort,
	// so a stale "true" could otherwise release this barrier on its first poll.
	return s.mergeCustomProperties(ctx, map[string]*structpb.Value{
		customPropDebugPauseBarrier:         structpb.NewStringValue(string(barrier)),
		customPropDebugPauseResumeRequested: structpb.NewStringValue("false"),
		customPropDebugPauseInstance:        structpb.NewStringValue(instance),
	})
}

func (s *kfpAPIPauseSignaler) ReadPauseState(ctx context.Context) (PauseState, error) {
	task, err := s.client.GetTask(ctx, &apiV2.GetTaskRequest{
		TaskId: s.taskID,
		RunId:  s.runID,
	})
	if err != nil {
		return PauseState{}, err
	}
	props := task.GetStatusMetadata().GetCustomProperties()
	return PauseState{
		Barrier:         DebugPauseBarrier(props[customPropDebugPauseBarrier].GetStringValue()),
		Instance:        props[customPropDebugPauseInstance].GetStringValue(),
		ResumeRequested: props[customPropDebugPauseResumeRequested].GetStringValue() == "true",
	}, nil
}

func (s *kfpAPIPauseSignaler) ClearBarrier(ctx context.Context) error {
	return s.mergeCustomProperties(ctx, map[string]*structpb.Value{
		customPropDebugPauseBarrier:         structpb.NewStringValue(string(debugPauseBarrierNone)),
		customPropDebugPauseResumeRequested: structpb.NewStringValue("false"),
		customPropDebugPauseInstance:        structpb.NewStringValue(""),
	})
}

// mergeCustomProperties does a read-merge-write because UpdateTask replaces
// StatusMetadata wholesale rather than merging per-key; writing without the
// read would erase unrelated custom properties.
// TODO: UpdateTask has no optimistic concurrency, so a concurrent writer to
// other custom_properties keys could still lose updates between our read and
// write. The pause keys are fenced by the instance check, but other keys are
// not.
func (s *kfpAPIPauseSignaler) mergeCustomProperties(ctx context.Context, updates map[string]*structpb.Value) error {
	current, err := s.client.GetTask(ctx, &apiV2.GetTaskRequest{
		TaskId: s.taskID,
		RunId:  s.runID,
	})
	if err != nil {
		return err
	}

	statusMetadata := current.GetStatusMetadata()
	if statusMetadata == nil {
		statusMetadata = &apiV2.PipelineTask_StatusMetadata{}
	}
	props := statusMetadata.GetCustomProperties()
	if props == nil {
		props = map[string]*structpb.Value{}
	} else {
		// Copy rather than mutate the response in place.

		merged := make(map[string]*structpb.Value, len(props)+len(updates))
		for k, v := range props {
			merged[k] = v
		}
		props = merged
	}
	for k, v := range updates {
		props[k] = v
	}
	statusMetadata.CustomProperties = props

	_, err = s.client.UpdateTask(ctx, &apiV2.UpdateTaskRequest{
		TaskId: s.taskID,
		RunId:  s.runID,
		Task: &apiV2.PipelineTask{
			TaskId:         s.taskID,
			RunId:          s.runID,
			StatusMetadata: statusMetadata,
		},
	})
	return err
}

// Pause parks the launcher at the barrier: publishes it, then polls for a
// resume until one arrives, MaxDuration elapses, or ctx is canceled. The
// barrier is cleared best-effort on return.
//
// Publish and poll failures are logged and retried rather than releasing or
// failing the pause; a timeout is the only error returned.
func Pause(ctx context.Context, signaler PauseSignaler, barrier DebugPauseBarrier, cfg DebugPauseConfig) error {
	if barrier == debugPauseBarrierNone {
		return nil
	}

	// Bound all parked RPCs by the safety valve - a per-tick deadline check
	// cannot fire while an RPC is stalled.
	waitCtx, waitCancel := context.WithTimeout(ctx, cfg.MaxDuration)
	defer waitCancel()

	instance := fmt.Sprintf("pause-%d", time.Now().UnixNano())

	// Resume signals are ignored until the barrier publish is confirmed.
	published := false
	if err := signaler.PublishBarrier(waitCtx, barrier, instance); err != nil {
		glog.Warningf("debug pause: failed to publish barrier %q, parking anyway and retrying: %v", barrier, err)
	} else {
		published = true
	}

	defer func() {
		// Clear on a fresh context: ctx may already be canceled.
		clearCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := signaler.ClearBarrier(clearCtx); err != nil {
			glog.Warningf("debug pause: failed to clear barrier %q after resume: %v", barrier, err)
		}
	}()

	glog.Infof("debug pause: parked at barrier %q, polling every %s (max wait %s)", barrier, cfg.PollInterval, cfg.MaxDuration)

	consecutivePollErrors := 0

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return newDebugPauseTimeoutError(barrier, cfg.MaxDuration)
			}
			glog.Warningf("debug pause: context canceled while parked at barrier %q: %v", barrier, waitCtx.Err())
			return waitCtx.Err()

		case <-ticker.C:
			state, err := signaler.ReadPauseState(waitCtx)
			if err != nil {
				consecutivePollErrors++
				logDebugPausePollError(barrier, consecutivePollErrors, err)
				continue
			}
			consecutivePollErrors = 0

			if state.Barrier != barrier || state.Instance != instance {
				// The row belongs to another epoch - either the initial
				// publish never committed, or a stale write rewound it.
				// Republish to fence the stale barrier and reset any
				// resume flag not meant for this pause.
				if err := signaler.PublishBarrier(waitCtx, barrier, instance); err != nil {
					consecutivePollErrors++
					logDebugPausePollError(barrier, consecutivePollErrors, err)
					continue
				}
				published = true
				glog.Infof("debug pause: barrier %q published after reconciling row {barrier: %q, instance: %q}", barrier, state.Barrier, state.Instance)
				// The publish reset the resume flag; get a fresh read next tick.
				continue
			}

			if !published {
				// The earlier publish committed but its response was lost;
				// republishing would reset an already-accepted resume.
				published = true
				glog.Infof("debug pause: barrier %q publish confirmed by reconcile", barrier)
			}

			if state.ResumeRequested {
				glog.Infof("debug pause: resume observed at barrier %q, continuing", barrier)
				return nil
			}
		}
	}
}

// logDebugPausePollError escalates severity with sustained failures; the
// pause itself never gives up on its own.
func logDebugPausePollError(barrier DebugPauseBarrier, consecutiveFailures int, err error) {
	switch {
	case consecutiveFailures >= 12: // ~1 minute of failures at the default poll interval
		glog.Errorf("debug pause: %d consecutive poll failures at barrier %q, still waiting: %v", consecutiveFailures, barrier, err)
	case consecutiveFailures >= 3:
		glog.Warningf("debug pause: %d consecutive poll failures at barrier %q: %v", consecutiveFailures, barrier, err)
	default:
		glog.V(2).Infof("debug pause: poll failure at barrier %q: %v", barrier, err)
	}
}

// debugPauseTimeoutError marks a safety-valve timeout distinctly from other
// execution failures.
type debugPauseTimeoutError struct {
	barrier     DebugPauseBarrier
	maxDuration time.Duration
}

func newDebugPauseTimeoutError(barrier DebugPauseBarrier, maxDuration time.Duration) error {
	return &debugPauseTimeoutError{barrier: barrier, maxDuration: maxDuration}
}

func (e *debugPauseTimeoutError) Error() string {
	return "debug pause timed out after " + e.maxDuration.String() + " waiting at barrier " + strconv.Quote(string(e.barrier))
}

// BarrierForError resolves which barrier applies once the command finishes.
// The SDK never sets both after and on_error; on_error is checked first
// defensively in case the env vars are set by hand.
func BarrierForError(cfg DebugPauseConfig, commandFailed bool) DebugPauseBarrier {
	if commandFailed && cfg.OnError {
		return DebugPauseBarrierOnError
	}
	if cfg.After {
		return DebugPauseBarrierAfter
	}
	return debugPauseBarrierNone
}

// clearDebugPauseProperties returns StatusMetadata with the pause keys
// explicitly cleared. The final task write replaces StatusMetadata wholesale
// and nil metadata means "leave unchanged", so the keys must be written
// cleared, not merely deleted.
func clearDebugPauseProperties(metadata *apiV2.PipelineTask_StatusMetadata) *apiV2.PipelineTask_StatusMetadata {
	if metadata == nil {
		metadata = &apiV2.PipelineTask_StatusMetadata{}
	}
	props := metadata.GetCustomProperties()
	if props == nil {
		props = map[string]*structpb.Value{}
	}
	props[customPropDebugPauseBarrier] = structpb.NewStringValue(string(debugPauseBarrierNone))
	props[customPropDebugPauseResumeRequested] = structpb.NewStringValue("false")
	props[customPropDebugPauseInstance] = structpb.NewStringValue("")
	metadata.CustomProperties = props
	return metadata
}
