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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// fakeRunStore records calls to the two GC methods while satisfying the full
// storage.RunStoreInterface. Only ArchiveExpiredRuns and DeleteExpiredArchivedRuns
// are exercised by collect(); the remaining methods are no-op stubs.
type fakeRunStore struct {
	archiveCalled bool
	archiveCutoff int64
	archiveBatch  int
	archiveReturn int64
	archiveErr    error

	deleteCalled bool
	deleteCutoff int64
	deleteBatch  int
	deleteReturn int64
	deleteErr    error
}

func (f *fakeRunStore) ArchiveExpiredRuns(cutoff int64, batchSize int) (int64, error) {
	f.archiveCalled = true
	f.archiveCutoff = cutoff
	f.archiveBatch = batchSize
	return f.archiveReturn, f.archiveErr
}

func (f *fakeRunStore) DeleteExpiredArchivedRuns(cutoff int64, batchSize int) (int64, error) {
	f.deleteCalled = true
	f.deleteCutoff = cutoff
	f.deleteBatch = batchSize
	return f.deleteReturn, f.deleteErr
}

// Stubs for the remaining RunStoreInterface methods.
func (f *fakeRunStore) CreateRun(_ *model.Run) (*model.Run, error)                      { return nil, nil }
func (f *fakeRunStore) GetRun(_ string) (*model.Run, error)                             { return nil, nil }
func (f *fakeRunStore) ListRuns(_ *model.FilterContext, _ *list.Options) ([]*model.Run, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeRunStore) UpdateRun(_ *model.Run) error                                    { return nil }
func (f *fakeRunStore) UpdateRunPluginsOutput(_ string, _ *model.LargeText) error        { return nil }
func (f *fakeRunStore) ArchiveRun(_ string) error                                        { return nil }
func (f *fakeRunStore) UnarchiveRun(_ string) error                                      { return nil }
func (f *fakeRunStore) DeleteRun(_ string) error                                         { return nil }
func (f *fakeRunStore) CreateMetric(_ *model.RunMetric) error                            { return nil }
func (f *fakeRunStore) TerminateRun(_ string) error                                      { return nil }
func (f *fakeRunStore) GetRunByRecurringRunIDAndDisplayName(_, _ string) (string, error)  { return "", nil }

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

	gc.collect()

	assert.False(t, fake.archiveCalled, "archive should not be called when RunsRetentionTime is 0")
	assert.False(t, fake.deleteCalled, "delete should not be called when ArchivedRunsRetentionTime is 0")
}

func TestCollect_ArchiveOnlyEnabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	// 720h = 30 days = 2592000 seconds.
	viper.Set(common.RunsRetentionTime, "720h")

	fake := &fakeRunStore{archiveReturn: 5}
	now := int64(3000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect()

	assert.True(t, fake.archiveCalled, "archive pass should be invoked")
	assert.Equal(t, now-2592000, fake.archiveCutoff, "archive cutoff = now minus 720h in seconds")
	assert.Equal(t, 100, fake.archiveBatch, "default batch size is 100")
	assert.False(t, fake.deleteCalled, "delete should not be called when ArchivedRunsRetentionTime is 0")
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

	gc.collect()

	assert.False(t, fake.archiveCalled, "archive should not be called when RunsRetentionTime is 0")
	assert.True(t, fake.deleteCalled, "delete pass should be invoked")
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

	gc.collect()

	assert.True(t, fake.archiveCalled)
	assert.Equal(t, now-2592000, fake.archiveCutoff)
	assert.Equal(t, 50, fake.archiveBatch, "custom batch size should be respected")

	assert.True(t, fake.deleteCalled)
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
		archiveErr:    util.NewInternalServerError(nil, "db connection lost"),
		deleteReturn:  4,
	}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 10000000 },
	}

	gc.collect()

	assert.True(t, fake.archiveCalled, "archive pass should be attempted even if it will fail")
	assert.True(t, fake.deleteCalled, "delete pass must still run after an archive error")
}
