// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
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
	"sync"
	"testing"
	"time"

	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// errLostResponse simulates a publish whose write committed on the server
// but whose response never reached the caller - the failure mode that makes
// a blind republish dangerous (it would reset a resume flag that may
// already have been accepted for this pause).
var errLostResponse = errors.New("lost response after commit")

// mockPauseSignaler is a hand-rolled PauseSignaler for testing Pause()'s
// control flow without any real network call. It simulates the "server"
// side of the task record: a failed publish does not commit, a successful
// (or lost-response) publish does. This lets tests exercise the reconcile
// path - distinguishing "my publish already landed" from "the row holds an
// older epoch".
type mockPauseSignaler struct {
	mu sync.Mutex

	// server is the pause state the task record currently holds.
	server PauseState

	// publishErrs is consumed one error per PublishBarrier call; publishErr
	// applies once the sequence is exhausted. errLostResponse commits the
	// write but reports failure; any other error fails before committing.
	publishErrs []error
	publishErr  error
	publishes   []DebugPauseBarrier

	// readErrs is consumed one error per ReadPauseState call; readErr
	// applies once exhausted.
	readErrs  []error
	readErr   error
	readCount int

	// resumeAfter: on the Nth successful read (0-indexed) the server flips
	// ResumeRequested to true - modelling the user's click landing at a
	// particular moment. -1 means never.
	resumeAfter int

	clearErr    error
	clearCalled bool
}

func (m *mockPauseSignaler) PublishBarrier(ctx context.Context, barrier DebugPauseBarrier, instance string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := len(m.publishes)
	m.publishes = append(m.publishes, barrier)

	var err error
	if idx < len(m.publishErrs) {
		err = m.publishErrs[idx]
	} else {
		err = m.publishErr
	}
	if err != nil && !errors.Is(err, errLostResponse) {
		return err
	}
	// The write commits: barrier + instance + resume flag reset, atomically.
	m.server = PauseState{Barrier: barrier, Instance: instance, ResumeRequested: false}
	return err
}

func (m *mockPauseSignaler) ReadPauseState(ctx context.Context) (PauseState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.readCount
	m.readCount++

	if idx < len(m.readErrs) {
		if err := m.readErrs[idx]; err != nil {
			return PauseState{}, err
		}
	} else if m.readErr != nil {
		return PauseState{}, m.readErr
	}

	if m.resumeAfter >= 0 && idx >= m.resumeAfter {
		m.server.ResumeRequested = true
	}
	return m.server, nil
}

func (m *mockPauseSignaler) ClearBarrier(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCalled = true
	return m.clearErr
}

func (m *mockPauseSignaler) readCountSafe() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readCount
}

// stallingPauseSignaler simulates an API server whose calls accept the
// publish but then stall forever on the read - the request only returns
// when the call's context expires. It proves the safety valve bounds
// in-flight RPCs, not just idle ticks between polls.
type stallingPauseSignaler struct {
	mockPauseSignaler
}

func (m *stallingPauseSignaler) ReadPauseState(ctx context.Context) (PauseState, error) {
	m.mu.Lock()
	m.readCount++
	m.mu.Unlock()
	<-ctx.Done()
	return PauseState{}, ctx.Err()
}

func fastTestConfig() DebugPauseConfig {
	return DebugPauseConfig{
		Before:       true,
		MaxDuration:  200 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}
}

func TestPause_ResumesOnFirstPollThatSaysYes(t *testing.T) {
	signaler := &mockPauseSignaler{resumeAfter: 2}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err)
	require.Equal(t, []DebugPauseBarrier{DebugPauseBarrierBefore}, signaler.publishes)
	require.True(t, signaler.clearCalled, "barrier must be cleared once resumed")
	require.GreaterOrEqual(t, signaler.readCountSafe(), 3)
}

func TestPause_NoOpWhenBarrierIsNone(t *testing.T) {
	signaler := &mockPauseSignaler{resumeAfter: -1}
	err := Pause(context.Background(), signaler, debugPauseBarrierNone, fastTestConfig())
	require.NoError(t, err)
	require.Empty(t, signaler.publishes, "must not publish or poll when no barrier applies")
	require.False(t, signaler.clearCalled)
}

// TestPause_PublishFailuresStillParks verifies that a failure to report "I
// am paused" does not abort the wait - the debugging session must not be
// lost to a transient reporting error. The barrier is republished on every
// poll tick until it lands, and a resume signal is honored only after the
// barrier is actually visible (which also resets any stale resume flag).
func TestPause_PublishFailuresStillParks(t *testing.T) {
	signaler := &mockPauseSignaler{
		publishErrs: []error{errors.New("api server unreachable"), errors.New("still down")},
		resumeAfter: 4,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err, "a publish failure must not prevent parking or resuming")
	require.True(t, signaler.clearCalled)
	require.Len(t, signaler.publishes, 3, "publish must be retried until it succeeds")
}

// TestPause_ResumeNotHonoredBeforePublishSucceeds covers the stale-flag
// hazard: a leftover resume_requested=true on the record must not release
// a barrier that was never published - otherwise a new barrier could
// "resume" itself immediately. The poll loop must reconcile publication
// first; publishing resets the flag, after which a resume read is trusted.
func TestPause_ResumeNotHonoredBeforePublishSucceeds(t *testing.T) {
	signaler := &mockPauseSignaler{
		// The record already reads resume_requested=true (stale). If Pause
		// honored it before republishing, it would return after a single
		// publish attempt.
		server:      PauseState{ResumeRequested: true},
		publishErrs: []error{errors.New("api server unreachable")},
		resumeAfter: 2,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err)
	require.Len(t, signaler.publishes, 2,
		"resume must not release the barrier until a republish lands and resets the flag")
	require.Equal(t, 3, signaler.readCountSafe(),
		"the stale flag must never be trusted while the barrier is unpublished")
}

// TestPause_PermanentPublishFailureTimesOut covers a publish that never
// lands: the launcher keeps the task parked (the session is worth keeping
// while the API may recover) and the safety valve is what eventually
// surfaces the failure.
func TestPause_PermanentPublishFailureTimesOut(t *testing.T) {
	signaler := &mockPauseSignaler{
		publishErr:  errors.New("api server unreachable"),
		resumeAfter: -1,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.Error(t, err)
	require.True(t, IsDebugPauseTimeout(err))
	require.GreaterOrEqual(t, len(signaler.publishes), 2, "publish must keep retrying until the safety valve")
	require.True(t, signaler.clearCalled, "best-effort clear still runs on timeout")
}

// TestPause_LostPublishResponsePreservesResume reproduces the
// committed-write / lost-response race: the publish lands on the server but
// its response never arrives, so the launcher believes it failed. If a user
// resumes before the retry, republishing would erase that accepted flag and
// the task would sit until timeout. Reconciling on the published instance
// ID lets the loop detect its own committed write and never republish.
func TestPause_LostPublishResponsePreservesResume(t *testing.T) {
	signaler := &mockPauseSignaler{
		publishErrs: []error{errLostResponse}, // commits, reports failure
		resumeAfter: 0,                        // resume already requested on first read
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err, "a resume accepted for this pause must release it")
	require.Len(t, signaler.publishes, 1,
		"a committed publish must never be republished - republishing would erase the accepted resume")
	require.True(t, signaler.clearCalled)
}

// TestPause_StaleEpochDoesNotBlockRepublish is the other side of the fence:
// the row holds a barrier at our value but published by a dead epoch (e.g.
// a restarted pod's earlier pause). Barrier match alone is not proof our
// write committed - the instance ID must also match, or the stale resume
// flag would be honored without a republish/reset.
func TestPause_StaleEpochDoesNotBlockRepublish(t *testing.T) {
	signaler := &mockPauseSignaler{
		server:      PauseState{Barrier: DebugPauseBarrierBefore, Instance: "dead-epoch", ResumeRequested: true},
		publishErrs: []error{errors.New("api server unreachable")}, // ours never commits
		resumeAfter: 2,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err)
	require.Len(t, signaler.publishes, 2,
		"a barrier published by a dead epoch must be republished, resetting its stale resume flag")
}

// TestPause_PollErrorsDoNotAbortWait verifies that transient poll errors are
// tolerated - the launcher keeps waiting and retrying rather than giving up
// (which would either silently release or wedge the pause).
func TestPause_PollErrorsDoNotAbortWait(t *testing.T) {
	flaky := errors.New("transient network error")
	signaler := &mockPauseSignaler{
		readErrs:    []error{flaky, flaky, flaky},
		resumeAfter: 4,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierAfter, fastTestConfig())
	require.NoError(t, err, "poll errors must not cause Pause to give up early")
	require.True(t, signaler.clearCalled)
}

// TestPause_SafetyValveTimesOut verifies the one case where Pause is allowed
// to return an error on its own: nobody ever resumes it, and the max
// duration elapses. This must surface as a real, identifiable failure -
// not a silent hang.
func TestPause_SafetyValveTimesOut(t *testing.T) {
	signaler := &mockPauseSignaler{resumeAfter: -1} // never resumes
	cfg := DebugPauseConfig{
		Before:       true,
		MaxDuration:  30 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, cfg)
	require.Error(t, err)
	require.True(t, IsDebugPauseTimeout(err), "timeout must be identifiable via IsDebugPauseTimeout")
	require.True(t, signaler.clearCalled, "barrier must still be cleared, best effort, after a timeout")
}

// TestPause_StalledPollIsBoundedBySafetyValve reproduces the hung-RPC hole:
// a resume poll that never returns must still be cut off by MaxDuration -
// the wait context covers in-flight RPCs, not just the gap between ticks.
func TestPause_StalledPollIsBoundedBySafetyValve(t *testing.T) {
	signaler := &stallingPauseSignaler{}
	cfg := DebugPauseConfig{
		Before:       true,
		MaxDuration:  50 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}

	done := make(chan error, 1)
	go func() { done <- Pause(context.Background(), signaler, DebugPauseBarrierBefore, cfg) }()

	select {
	case err := <-done:
		require.Error(t, err)
		require.True(t, IsDebugPauseTimeout(err), "a stalled RPC must surface as the safety-valve timeout")
		require.True(t, signaler.clearCalled, "best-effort clear still runs after a stalled-RPC timeout")
	case <-time.After(10 * time.Second):
		t.Fatal("Pause did not return - a stalled poll escaped the safety valve")
	}
}

// TestPause_ContextCancellationClearsBestEffort verifies that canceling the
// context (e.g. the launcher process shutting down) causes Pause to return
// promptly with the context's error, while still attempting a best-effort
// clear using a fresh, uncanceled context for cleanup.
func TestPause_ContextCancellationClearsBestEffort(t *testing.T) {
	signaler := &mockPauseSignaler{resumeAfter: -1} // never resumes
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Pause(ctx, signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, signaler.clearCalled, "clear must still be attempted after cancellation, via a fresh context")
}

// seedPausedTask creates a task in the mock API carrying the given custom
// properties, simulating leftover state from a previous barrier.
func seedPausedTask(t *testing.T, api *kfpapi.MockAPI, runID, taskID string, props map[string]*structpb.Value) {
	t.Helper()
	_, err := api.CreateTask(context.Background(), &apiV2.CreateTaskRequest{
		RunId: runID,
		Task: &apiV2.PipelineTask{
			TaskId: taskID,
			RunId:  runID,
			StatusMetadata: &apiV2.PipelineTask_StatusMetadata{
				CustomProperties: props,
			},
		},
	})
	require.NoError(t, err)
}

// TestPublishBarrier_ResetsStaleResumeFlag verifies that publishing a barrier
// also resets debug_pause_resume_requested. ClearBarrier is best effort, so
// a resume flag from a previous barrier can survive a failed UpdateTask and
// would otherwise release the next barrier on its first poll.
func TestPublishBarrier_ResetsStaleResumeFlag(t *testing.T) {
	api := kfpapi.NewMockAPI()
	seedPausedTask(t, api, "run-1", "task-1", map[string]*structpb.Value{
		customPropDebugPauseBarrier:         structpb.NewStringValue("none"),
		customPropDebugPauseResumeRequested: structpb.NewStringValue("true"),
	})

	signaler := NewKFPAPIPauseSignaler(api, "run-1", "task-1")
	require.NoError(t, signaler.PublishBarrier(context.Background(), DebugPauseBarrierAfter, "instance-1"))

	state, err := signaler.ReadPauseState(context.Background())
	require.NoError(t, err)
	require.False(t, state.ResumeRequested, "publishing a new barrier must reset a stale resume flag")
	require.Equal(t, DebugPauseBarrierAfter, state.Barrier)
	require.Equal(t, "instance-1", state.Instance, "the publish must stamp its instance ID on the record")
}

// TestPause_StaleResumeFlagDoesNotReleaseBarrier reproduces the full scenario
// end to end: a stale resume flag from a previous barrier sits on the task
// when Pause() parks at a new barrier. PublishBarrier resets it, so the wait
// continues to the safety valve instead of releasing on the first poll.
func TestPause_StaleResumeFlagDoesNotReleaseBarrier(t *testing.T) {
	api := kfpapi.NewMockAPI()
	seedPausedTask(t, api, "run-1", "task-1", map[string]*structpb.Value{
		customPropDebugPauseResumeRequested: structpb.NewStringValue("true"),
	})

	signaler := NewKFPAPIPauseSignaler(api, "run-1", "task-1")
	err := Pause(context.Background(), signaler, DebugPauseBarrierAfter, fastTestConfig())
	require.Error(t, err)
	require.True(t, IsDebugPauseTimeout(err), "stale resume flag must not release the barrier; expected timeout")
}

func TestBarrierForError(t *testing.T) {
	tests := []struct {
		name          string
		cfg           DebugPauseConfig
		commandFailed bool
		want          DebugPauseBarrier
	}{
		{"after only, succeeded", DebugPauseConfig{After: true}, false, DebugPauseBarrierAfter},
		{"after only, failed", DebugPauseConfig{After: true}, true, DebugPauseBarrierAfter},
		{"after+on_error, succeeded", DebugPauseConfig{After: true, OnError: true}, false, DebugPauseBarrierAfter},
		{"after+on_error, failed", DebugPauseConfig{After: true, OnError: true}, true, DebugPauseBarrierOnError},
		{"neither configured", DebugPauseConfig{}, true, debugPauseBarrierNone},
		{"before only, ignored post-execution", DebugPauseConfig{Before: true}, true, debugPauseBarrierNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, BarrierForError(tt.cfg, tt.commandFailed))
		})
	}
}

func TestNewDebugPauseConfigFromEnv(t *testing.T) {
	t.Setenv(envKFPDebugPauseBefore, "true")
	t.Setenv(envKFPDebugPauseAfter, "")
	t.Setenv(envKFPDebugPauseOnError, "")
	t.Setenv(envKFPDebugPauseMaxDuration, "")

	cfg := NewDebugPauseConfigFromEnv()
	require.True(t, cfg.Before)
	require.False(t, cfg.After)
	require.False(t, cfg.OnError)
	require.True(t, cfg.Enabled())
	require.Equal(t, defaultDebugPauseMaxDuration, cfg.MaxDuration)
}

func TestNewDebugPauseConfigFromEnv_Disabled(t *testing.T) {
	t.Setenv(envKFPDebugPauseBefore, "")
	t.Setenv(envKFPDebugPauseAfter, "")
	t.Setenv(envKFPDebugPauseOnError, "")

	cfg := NewDebugPauseConfigFromEnv()
	require.False(t, cfg.Enabled(), "a task that never called set_debug_pause() must be a no-op")
}

func TestNewDebugPauseConfigFromEnv_InvalidMaxDurationFallsBackToDefault(t *testing.T) {
	t.Setenv(envKFPDebugPauseBefore, "true")
	t.Setenv(envKFPDebugPauseMaxDuration, "not-a-duration")

	cfg := NewDebugPauseConfigFromEnv()
	require.Equal(t, defaultDebugPauseMaxDuration, cfg.MaxDuration)
}

func TestNewDebugPauseConfigFromEnv_ValidMaxDurationOverride(t *testing.T) {
	t.Setenv(envKFPDebugPauseBefore, "true")
	t.Setenv(envKFPDebugPauseMaxDuration, "45m")

	cfg := NewDebugPauseConfigFromEnv()
	require.Equal(t, 45*time.Minute, cfg.MaxDuration)
}
