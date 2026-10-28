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
// should park at. This is intentionally a small, closed set matching the SDK's
// set_debug_pause() surface (before/after/on_error) - it is never written into
// PipelineTask.State. Debug pause is a separate overlay value on top of a
// task's normal RUNNING/SUCCEEDED/FAILED lifecycle, never a competing state.
type DebugPauseBarrier string

const (
	DebugPauseBarrierBefore  DebugPauseBarrier = "before"
	DebugPauseBarrierAfter   DebugPauseBarrier = "after"
	DebugPauseBarrierOnError DebugPauseBarrier = "on_error"
	debugPauseBarrierNone    DebugPauseBarrier = ""
)

// Env vars set by the SDK's set_debug_pause() (sdk/python/kfp/dsl/pipeline_task.py).
// These are KFP-owned names, deliberately distinct from Argo's ARGO_DEBUG_PAUSE_*
// vars: Argo's emissary no longer participates in pausing at all - the launcher
// is the sole owner of this behavior.
const (
	envKFPDebugPauseBefore  = "KFP_DEBUG_PAUSE_BEFORE"
	envKFPDebugPauseAfter   = "KFP_DEBUG_PAUSE_AFTER"
	envKFPDebugPauseOnError = "KFP_DEBUG_PAUSE_ON_ERROR"

	// envKFPDebugPauseMaxDuration optionally overrides the safety-valve
	// duration (Go duration string, e.g. "2h"). If unset or unparseable,
	// defaultDebugPauseMaxDuration is used.
	envKFPDebugPauseMaxDuration = "KFP_DEBUG_PAUSE_MAX_DURATION"
)

// custom_properties keys on PipelineTask.StatusMetadata. Chosen over adding a
// PAUSED value to PipelineTask_TaskState so pause never competes with the
// task's real RUNNING/SUCCEEDED/FAILED outcome - it is a side-channel on the
// same record, not a fourth state.
const (
	customPropDebugPauseBarrier         = "debug_pause_barrier"
	customPropDebugPauseResumeRequested = "debug_pause_resume_requested"
	// customPropDebugPauseInstance fences each pause attempt: a unique ID
	// generated per Pause() call and published alongside the barrier. It
	// lets a publish retry tell "my write already committed" apart from
	// "residue of an earlier pause epoch" (e.g. a dead pod's barrier at the
	// same value), without which a retry could either erase an accepted
	// resume or honor a stale one.
	customPropDebugPauseInstance = "debug_pause_instance"
)

// defaultDebugPauseMaxDuration bounds how long a launcher will wait at a
// barrier before giving up and failing the task. Without a bound, a missed
// resume signal (UI bug, network blip, or a person simply walking away) would
// leave the pod running forever, silently holding its node's resources and
// scheduler slot with no recovery path. Operators needing longer debugging
// sessions can override via envKFPDebugPauseMaxDuration.
const defaultDebugPauseMaxDuration = 1 * time.Hour

// defaultDebugPausePollInterval is how often the launcher asks the API server
// "has resume been requested yet?" while parked at a barrier.
const defaultDebugPausePollInterval = 5 * time.Second

// DebugPauseConfig is parsed once from the task's env vars.
type DebugPauseConfig struct {
	Before  bool
	After   bool
	OnError bool

	// MaxDuration is the safety-valve bound; PollInterval is how often the
	// launcher polls for a resume request while parked.
	MaxDuration  time.Duration
	PollInterval time.Duration
}

// NewDebugPauseConfigFromEnv reads the KFP_DEBUG_PAUSE_* env vars set by the
// SDK's set_debug_pause(). Absent or non-"true" values mean "do not pause" -
// this mirrors set_env_variable's own "true"/unset convention, so a launcher
// running a task that never called set_debug_pause() sees an all-false,
// no-op config.
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

// Enabled reports whether any pause barrier is configured at all, so callers
// can skip the feature entirely for the common case of a task that never
// called set_debug_pause().
func (c DebugPauseConfig) Enabled() bool {
	return c.Before || c.After || c.OnError
}

// PauseState is the pause overlay as currently persisted on the task
// record: which barrier is parked, which pause instance published it, and
// whether resume has been requested.
type PauseState struct {
	Barrier         DebugPauseBarrier
	Instance        string
	ResumeRequested bool
}

// PauseSignaler is the seam between the pause loop and however it talks to
// the KFP API server. Abstracting this out lets debugPauseLoop be unit
// tested with a mock, without a real network call or a running API server.
type PauseSignaler interface {
	// PublishBarrier records that the task is now parked at the given
	// barrier by this pause instance. Implementations must merge into the
	// task's existing StatusMetadata.CustomProperties rather than overwrite
	// it wholesale - UpdateTask replaces the whole StatusMetadata field, so
	// a naive overwrite here would silently erase any other custom
	// properties already recorded on the task (e.g. a prior failure
	// message).
	PublishBarrier(ctx context.Context, barrier DebugPauseBarrier, instance string) error

	// ReadPauseState returns the pause overlay currently persisted on the
	// task record.
	ReadPauseState(ctx context.Context) (PauseState, error)

	// ClearBarrier removes the pause, instance, and resume-requested
	// markers, best effort, once the launcher is done waiting (whether it
	// was released normally or is exiting due to cancellation/timeout).
	ClearBarrier(ctx context.Context) error
}

// kfpAPIPauseSignaler is the real PauseSignaler, backed by the same
// kfpapi.API client the launcher already uses for every other task update
// (task creation, output reporting, etc).
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
	// The resume flag must be reset at every publish, not just by
	// ClearBarrier: ClearBarrier is best effort, so a stale "true" from a
	// previous barrier can survive if its UpdateTask failed - and would
	// otherwise release this barrier on the very first poll. The instance
	// ID makes this write identifiable so a retry can reconcile "did my
	// publish commit?" instead of blindly rewriting (which would erase a
	// resume accepted between a committed write and its lost response).
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

// mergeCustomProperties performs a read-merge-write against the task's
// current StatusMetadata. This exists specifically because UpdateTask (and
// the launcher's own BatchUpdater) treat StatusMetadata as a whole-object
// overwrite, not a per-key merge, unlike InputParameters/OutputParameters
// which are explicitly merged server-side. Without this read first, setting
// the pause barrier would risk erasing an unrelated custom property (e.g. a
// failure message) that was already recorded on the task.
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

// Pause parks the launcher at the given barrier: it publishes that it has
// paused, then repeatedly polls for a resume request until one arrives, the
// safety-valve duration elapses, or ctx is canceled. It always attempts to
// clear the barrier before returning, best effort.
//
// Failure handling is deliberate, not incidental:
//   - A failure to publish the barrier does not prevent parking - the pause
//     itself must still happen even if the API server is briefly unreachable,
//     so a debugging session isn't lost to a transient reporting error.
//   - Poll errors do not abort the wait - they are logged with escalating
//     severity and polling continues, since a flaky API server should not
//     silently release (or permanently wedge) a paused task.
//   - The safety valve is the only way this function returns an error to the
//     caller: it is treated as a real, visible failure ("debug pause timed
//     out"), not a silent hang.
func Pause(ctx context.Context, signaler PauseSignaler, barrier DebugPauseBarrier, cfg DebugPauseConfig) error {
	if barrier == debugPauseBarrierNone {
		return nil
	}

	// Bound the entire parked wait - including in-flight publish and poll
	// RPCs - by the safety-valve duration. A between-polls deadline check
	// alone cannot fire while an RPC is stalled, so a hung call on the
	// caller's context could otherwise keep the pod alive indefinitely.
	waitCtx, waitCancel := context.WithTimeout(ctx, cfg.MaxDuration)
	defer waitCancel()

	// A unique ID per Pause() call, published alongside the barrier. It
	// lets a retry distinguish "my earlier publish actually committed and
	// only its response was lost" (barrier AND instance match ours - safe
	// to stop republishing without erasing a resume accepted meanwhile)
	// from "the row holds an older epoch's pause" (instance differs -
	// republish to fence it and reset its stale resume flag).
	instance := fmt.Sprintf("pause-%d", time.Now().UnixNano())

	// Publishing the barrier is what makes the pause visible and resumable,
	// and it resets any stale resume flag - so a failed publish is retried
	// on every poll tick, and resume signals are not honored until the
	// barrier has actually been published.
	published := false
	if err := signaler.PublishBarrier(waitCtx, barrier, instance); err != nil {
		glog.Warningf("debug pause: failed to publish barrier %q, parking anyway and retrying: %v", barrier, err)
	} else {
		published = true
	}

	defer func() {
		// Use a fresh, short-lived context for cleanup: ctx may already be
		// canceled (timeout or caller cancellation) by the time we get here,
		// and clearing the barrier is a best-effort courtesy, not something
		// worth blocking shutdown on.
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

			if !published {
				if state.Barrier == barrier && state.Instance == instance {
					// The earlier publish did commit - only its response was
					// lost. Republishing would reset resume_requested and
					// could erase a resume already accepted for this pause.
					published = true
					glog.Infof("debug pause: barrier %q publish confirmed by reconcile", barrier)
				} else {
					if err := signaler.PublishBarrier(waitCtx, barrier, instance); err != nil {
						consecutivePollErrors++
						logDebugPausePollError(barrier, consecutivePollErrors, err)
						continue
					}
					published = true
					glog.Infof("debug pause: barrier %q published after retry", barrier)
					// The state read predates this write (which reset the
					// resume flag); get a fresh read next tick.
					continue
				}
			}

			if state.ResumeRequested {
				glog.Infof("debug pause: resume observed at barrier %q, continuing", barrier)
				return nil
			}
		}
	}
}

// logDebugPausePollError escalates log severity the longer polling has been
// failing, so a transient blip stays quiet but a sustained outage is loud -
// without ever giving up and releasing (or failing) the pause on its own.
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

// debugPauseTimeoutError is returned when the safety valve trips. It is a
// distinct type so callers (and tests) can identify a timeout specifically,
// rather than treating it the same as an arbitrary execution failure.
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

// IsDebugPauseTimeout reports whether err is a debug pause safety-valve
// timeout, so callers can distinguish "nobody resumed in time" from any
// other execution failure.
func IsDebugPauseTimeout(err error) bool {
	_, ok := err.(*debugPauseTimeoutError)
	return ok
}

// BarrierForError resolves which barrier applies once the user's command has
// finished, given whether it failed. The SDK's set_debug_pause() only ever
// sets one of after/on_error (never both - see pipeline_task.py), so in
// practice these are mutually exclusive; on_error is checked first purely as
// a defensive ordering in case env vars are ever set by hand rather than
// through the SDK.
func BarrierForError(cfg DebugPauseConfig, commandFailed bool) DebugPauseBarrier {
	if commandFailed && cfg.OnError {
		return DebugPauseBarrierOnError
	}
	if cfg.After {
		return DebugPauseBarrierAfter
	}
	return debugPauseBarrierNone
}

// clearDebugPauseProperties returns a StatusMetadata with the debug-pause
// overlay keys explicitly cleared (barrier "none", resume "false"),
// preserving any other custom properties. The launcher applies this to its
// in-memory task before the final status write (finalizeExecution): that
// write replaces StatusMetadata wholesale and a nil metadata means "leave
// unchanged" to the task store - so the keys must be written cleared, not
// merely deleted, or a stale barrier published before a lost best-effort
// ClearBarrier would persist on a finished task.
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
