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

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// mockPauseSignaler is a hand-rolled PauseSignaler for testing Pause()'s
// control flow without any real network call. Each behavior is independently
// configurable so tests can exercise the specific failure modes Pause() is
// required to handle.
type mockPauseSignaler struct {
	mu sync.Mutex

	publishErr error
	publishes  []DebugPauseBarrier

	// resumeSequence is consumed one value per IsResumeRequested call; the
	// last value is reused once exhausted. This lets a test express "not
	// resumed for the first N polls, then resumed" concisely.
	resumeSequence []bool
	resumeErrs     []error
	pollCount      int

	clearErr    error
	clearCalled bool
}

func (m *mockPauseSignaler) PublishBarrier(ctx context.Context, barrier DebugPauseBarrier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishes = append(m.publishes, barrier)
	return m.publishErr
}

func (m *mockPauseSignaler) IsResumeRequested(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.pollCount
	m.pollCount++

	var err error
	if idx < len(m.resumeErrs) {
		err = m.resumeErrs[idx]
	}
	if err != nil {
		return false, err
	}

	if len(m.resumeSequence) == 0 {
		return false, nil
	}
	if idx < len(m.resumeSequence) {
		return m.resumeSequence[idx], nil
	}
	return m.resumeSequence[len(m.resumeSequence)-1], nil
}

func (m *mockPauseSignaler) ClearBarrier(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCalled = true
	return m.clearErr
}

func (m *mockPauseSignaler) pollCountSafe() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pollCount
}

func fastTestConfig() DebugPauseConfig {
	return DebugPauseConfig{
		Before:       true,
		MaxDuration:  200 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}
}

func TestPause_ResumesOnFirstPollThatSaysYes(t *testing.T) {
	signaler := &mockPauseSignaler{resumeSequence: []bool{false, false, true}}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err)
	require.Equal(t, []DebugPauseBarrier{DebugPauseBarrierBefore}, signaler.publishes)
	require.True(t, signaler.clearCalled, "barrier must be cleared once resumed")
	require.GreaterOrEqual(t, signaler.pollCountSafe(), 3)
}

func TestPause_NoOpWhenBarrierIsNone(t *testing.T) {
	signaler := &mockPauseSignaler{}
	err := Pause(context.Background(), signaler, debugPauseBarrierNone, fastTestConfig())
	require.NoError(t, err)
	require.Empty(t, signaler.publishes, "must not publish or poll when no barrier applies")
	require.False(t, signaler.clearCalled)
}

// TestPause_PublishFailuresStillParks verifies that a failure to report "I am
// paused" does not prevent the actual pause from happening - the debugging
// session must not be lost to a transient reporting error. Confirmed by the
// loop still requiring a real resume signal before returning
func TestPause_PublishFailuresStillParks(t *testing.T) {
	signaler := &mockPauseSignaler{
		publishErr:     errors.New("api server unreachable"),
		resumeSequence: []bool{false, true},
	}
	err := Pause(context.Background(), signaler, DebugPauseBarrierBefore, fastTestConfig())
	require.NoError(t, err, "a publish failure must not prevent parking or resuming")
	require.True(t, signaler.clearCalled)
}

// TestPause_PollErrorsDoNotAbortWait verifies that transient poll errors are
// tolerated - the launcher keeps waiting and retrying rather than giving up
// (which would either silently release or wedge the pause).
func TestPause_PollErrorsDoNotAbortWait(t *testing.T) {
	flaky := errors.New("transient network error")
	signaler := &mockPauseSignaler{
		resumeErrs:     []error{flaky, flaky, flaky, nil, nil},
		resumeSequence: []bool{false, false, false, false, true},
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
	signaler := &mockPauseSignaler{} // never resumes
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

// TestPause_ContextCancellationClearsBestEffort verifies that canceling the
// context (e.g. the launcher process shutting down) causes Pause to return
// promptly with the context's error, while still attempting a best-effort
// clear using a fresh, uncanceled context for cleanup.
func TestPause_ContextCancellationClearsBestEffort(t *testing.T) {
	signaler := &mockPauseSignaler{} // never resumes
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
	_, err := api.CreateTask(context.Background(), &apiV2beta1.CreateTaskRequest{
		RunId: runID,
		Task: &apiV2beta1.PipelineTask{
			TaskId: taskID,
			RunId:  runID,
			StatusMetadata: &apiV2beta1.PipelineTask_StatusMetadata{
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
	require.NoError(t, signaler.PublishBarrier(context.Background(), DebugPauseBarrierAfter))

	resumed, err := signaler.IsResumeRequested(context.Background())
	require.NoError(t, err)
	require.False(t, resumed, "publishing a new barrier must reset a stale resume flag")
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
