// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const transferredStateHistory = `[{"UpdateTimeInSec":1700000000,"State":"FAILED","Error":{"code":13,"message":"source error"}}]`

func TestStateHistorySurvivesTransfer(t *testing.T) {
	source := fixture(t, testDB(t))
	require.NoError(t, source.DB.Table("run_details").Where(equal("UUID", "run")).Update("StateHistory", transferredStateHistory).Error)
	require.NoError(t, source.DB.Table("tasks").Where(equal("UUID", "task")).Update("StateHistory", transferredStateHistory).Error)
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = destination.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	var run model.Run
	require.NoError(t, destination.DB.Take(&run).Error)
	require.Equal(t, model.LargeText(transferredStateHistory), run.StateHistoryString)
	var task model.Task
	require.NoError(t, destination.DB.Take(&task).Error)
	require.Equal(t, model.LargeText(transferredStateHistory), task.StateHistoryString)
	sqlDB, err := destination.DB.DB()
	require.NoError(t, err)
	hydrated, err := storage.NewRunStore(sqlDB, util.NewFakeTimeForEpoch(), dialect.NewDBDialect("sqlite")).GetRun(run.UUID)
	require.NoError(t, err)
	require.Len(t, hydrated.StateHistory, 1)
	require.Equal(t, codes.Internal, status.Code(hydrated.StateHistory[0].Error))
	require.ErrorContains(t, hydrated.StateHistory[0].Error, "source error")
}

func TestStateHistoryRejectsUnreadableArchiveBeforeWrites(t *testing.T) {
	for _, task := range []bool{false, true} {
		source := fixture(t, testDB(t))
		data, err := source.Export(context.Background(), "team", ExportOptions{})
		require.NoError(t, err)
		var archive Bundle
		require.NoError(t, json.Unmarshal(data, &archive))
		if task {
			archive.Runs[0].Tasks[0].StateHistoryString = `[{"Error":{"unknown":true}}]`
		} else {
			archive.Runs[0].Run.StateHistoryString = `[{"Error":{"unknown":true}}]`
		}
		destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
		for _, dry := range []bool{true, false} {
			_, err = destination.Import(context.Background(), "team", signRuntimeArchive(t, archive), ImportOptions{DryRun: dry})
			require.ErrorContains(t, err, "state history")
		}
		require.Zero(t, destination.Schedules.(*fakeSchedules).writes)
		require.Zero(t, destination.Metadata.RPC.(*emptyRPC).writes)
		var count int64
		require.NoError(t, destination.DB.Model(&model.Experiment{}).Count(&count).Error)
		require.Zero(t, count)
	}
}
