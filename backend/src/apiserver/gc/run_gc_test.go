// Copyright 2026 The Kubeflow Authors
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

package gc

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// fakeRunStore records calls to the two GC methods while satisfying the full
// storage.RunStoreInterface. When archiveReturnSequence or deleteReturnSequence
// is set, each successive call returns the next value in the slice (to test
// the drain loop's multi-batch behavior). Otherwise the scalar archiveReturn
// or deleteReturn is used for backward compatibility.
type fakeRunStore struct {
	archiveCalls          int
	archiveCutoff         int64
	archiveBatch          int
	archiveReturn         int64
	archiveReturnSequence []int64
	archiveErr            error

	deleteCalls          int
	deleteCutoff         int64
	deleteBatch          int
	deleteReturn         int64
	deleteReturnSequence []int64
	deleteErr            error
}

func (f *fakeRunStore) ArchiveExpiredRuns(cutoff int64, batchSize int) (int64, error) {
	callIndex := f.archiveCalls
	f.archiveCalls++
	f.archiveCutoff = cutoff
	f.archiveBatch = batchSize
	if f.archiveReturnSequence != nil && callIndex < len(f.archiveReturnSequence) {
		return f.archiveReturnSequence[callIndex], f.archiveErr
	}
	return f.archiveReturn, f.archiveErr
}

func (f *fakeRunStore) DeleteExpiredArchivedRuns(cutoff int64, batchSize int) (int64, error) {
	callIndex := f.deleteCalls
	f.deleteCalls++
	f.deleteCutoff = cutoff
	f.deleteBatch = batchSize
	if f.deleteReturnSequence != nil && callIndex < len(f.deleteReturnSequence) {
		return f.deleteReturnSequence[callIndex], f.deleteErr
	}
	return f.deleteReturn, f.deleteErr
}

// Stubs for the remaining RunStoreInterface methods.
func (f *fakeRunStore) CreateRun(_ *model.Run) (*model.Run, error) { return nil, nil }
func (f *fakeRunStore) GetRun(_ string) (*model.Run, error)        { return nil, nil }
func (f *fakeRunStore) ListRuns(_ *model.FilterContext, _ *list.Options) ([]*model.Run, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeRunStore) UpdateRun(_ *model.Run) error                              { return nil }
func (f *fakeRunStore) UpdateRunPluginsOutput(_ string, _ *model.LargeText) error { return nil }
func (f *fakeRunStore) ArchiveRun(_ string) error                                 { return nil }
func (f *fakeRunStore) UnarchiveRun(_ string) error                               { return nil }
func (f *fakeRunStore) DeleteRun(_ string) error                                  { return nil }
func (f *fakeRunStore) CreateMetric(_ *model.RunMetric) error                     { return nil }
func (f *fakeRunStore) TerminateRun(_ string) error                               { return nil }
func (f *fakeRunStore) GetRunByRecurringRunIDAndDisplayName(_, _ string) (string, error) {
	return "", nil
}

func resetGCConfig() {
	viper.Set(common.RunsRetentionTime, "")
	viper.Set(common.ArchivedRunsRetentionTime, "")
	viper.Set(common.RunsGCBatchSize, "")
}

func TestCollect_BothDisabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	fake := &fakeRunStore{}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 1000000 },
	}

	gc.collect(context.Background())

	assert.Equal(t, 0, fake.archiveCalls, "archive should not be called when RUNS_RETENTION_TIME is empty")
	assert.Equal(t, 0, fake.deleteCalls, "delete should not be called when ARCHIVED_RUNS_RETENTION_TIME is empty")
}

func TestCollect_ArchiveOnlyEnabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	// 720h = 30 days = 2592000 seconds.
	viper.Set(common.RunsRetentionTime, "720h")

	// Return < batchSize so the drain loop exits after one call.
	fake := &fakeRunStore{archiveReturn: 5}
	now := int64(3000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls, "archive pass should be invoked once")
	assert.Equal(t, now-2592000, fake.archiveCutoff, "archive cutoff = now minus 720h in seconds")
	assert.Equal(t, 100, fake.archiveBatch, "default batch size is 100")
	assert.Equal(t, 0, fake.deleteCalls, "delete should not be called when ARCHIVED_RUNS_RETENTION_TIME is empty")
}

func TestCollect_DeleteOnlyEnabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	// 2160h = 90 days = 7776000 seconds.
	viper.Set(common.ArchivedRunsRetentionTime, "2160h")

	fake := &fakeRunStore{deleteReturn: 3}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 0, fake.archiveCalls, "archive should not be called when RUNS_RETENTION_TIME is empty")
	assert.Equal(t, 1, fake.deleteCalls, "delete pass should be invoked once")
	assert.Equal(t, now-7776000, fake.deleteCutoff, "delete cutoff = now minus 2160h in seconds")
	assert.Equal(t, 100, fake.deleteBatch, "default batch size is 100")
}

func TestCollect_BothEnabled_CustomBatchSize(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.ArchivedRunsRetentionTime, "2160h")
	viper.Set(common.RunsGCBatchSize, "50")

	fake := &fakeRunStore{archiveReturn: 2, deleteReturn: 1}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls)
	assert.Equal(t, now-2592000, fake.archiveCutoff)
	assert.Equal(t, 50, fake.archiveBatch, "custom batch size should be respected")

	assert.Equal(t, 1, fake.deleteCalls)
	assert.Equal(t, now-7776000, fake.deleteCutoff)
	assert.Equal(t, 50, fake.deleteBatch, "custom batch size should be respected")
}

func TestCollect_ArchiveErrorDoesNotBlockDeletePass(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.ArchivedRunsRetentionTime, "2160h")

	fake := &fakeRunStore{
		archiveReturn: 0,
		archiveErr:    fmt.Errorf("db connection lost"),
		deleteReturn:  4,
	}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 10000000 },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls, "archive pass should be attempted even if it will fail")
	assert.Equal(t, 1, fake.deleteCalls, "delete pass must still run after an archive error")
}

func TestCollect_CanceledContextExitsEarly(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")

	fake := &fakeRunStore{archiveReturn: 5}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 10000000 },
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gc.collect(ctx)

	assert.Equal(t, 0, fake.archiveCalls, "canceled context should skip archive pass")
}

func TestCollect_ArchiveDrainsMultipleBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.RunsGCBatchSize, "100")

	// First call returns exactly batchSize (100) → drain loop continues.
	// Second call returns 30 (< batchSize) → drain loop exits.
	fake := &fakeRunStore{
		archiveReturnSequence: []int64{100, 30},
	}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 2, fake.archiveCalls, "drain loop should call archive twice: full batch then partial batch")
}

func TestCollect_DeleteDrainsMultipleBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.ArchivedRunsRetentionTime, "2160h")
	viper.Set(common.RunsGCBatchSize, "100")

	// First call returns exactly batchSize (100) → drain loop continues.
	// Second call returns 15 (< batchSize) → drain loop exits.
	fake := &fakeRunStore{
		deleteReturnSequence: []int64{100, 15},
	}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 2, fake.deleteCalls, "drain loop should call delete twice: full batch then partial batch")
}

func TestCollect_ContextCancelStopsDrainBetweenBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.RunsGCBatchSize, "100")

	ctx, cancel := context.WithCancel(context.Background())

	// cancellingFakeRunStore cancels the context after the first archive call
	// returns batchSize (100). The drain loop checks ctx.Err() at the top of
	// the next iteration and exits before making a second call.
	cancelOnFirstCall := &cancellingFakeRunStore{
		fakeRunStore: fakeRunStore{
			archiveReturnSequence: []int64{100, 0},
		},
		cancelAfterArchiveN: 1,
		cancelFunc:          cancel,
	}
	gc := &RunGarbageCollector{
		runStore: cancelOnFirstCall,
		nowFunc:  func() int64 { return 10000000 },
	}

	gc.collect(ctx)

	assert.Equal(t, 1, cancelOnFirstCall.archiveCalls,
		"drain loop should stop after first batch when context is canceled between iterations")
}

// cancellingFakeRunStore wraps fakeRunStore and cancels a context after N archive calls.
type cancellingFakeRunStore struct {
	fakeRunStore
	cancelAfterArchiveN int
	cancelFunc          context.CancelFunc
}

func (f *cancellingFakeRunStore) ArchiveExpiredRuns(cutoff int64, batchSize int) (int64, error) {
	result, err := f.fakeRunStore.ArchiveExpiredRuns(cutoff, batchSize)
	if f.archiveCalls >= f.cancelAfterArchiveN && f.cancelFunc != nil {
		f.cancelFunc()
	}
	return result, err
}
