// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/stretchr/testify/require"
)

func TestLegacyTransferDatabaseEngines(t *testing.T) {
	for _, tc := range []struct{ driver, env string }{{"mysql", "KFP_HISTORY_MYSQL_TEST_DSN"}, {"postgres", "KFP_HISTORY_POSTGRES_TEST_DSN"}} {
		t.Run(tc.driver, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if dsn == "" {
				t.Skip("set " + tc.env + " to run database integration tests")
			}
			dest := externalDatabase(t, tc.driver, dsn)
			create(t, dest, &model.Experiment{UUID: "native", Name: "Default", Namespace: "team"})
			archive, err := os.ReadFile("testdata/legacy-218-mlmd-v2-export.json")
			require.NoError(t, err)
			ctx := context.Background()
			apply := func(dry bool) *TransferPlan {
				b, _, err := DecodeNamespaceArchive(dest, archive, "team", "team", false)
				require.NoError(t, err)
				plan, err := PrepareTransfer(ctx, dest, b, transfer.ImportOptions{NamePrefix: "old-", DryRun: dry})
				require.NoError(t, err)
				require.NoError(t, CommitTransfer(ctx, dest, plan, dry))
				return plan
			}
			apply(true)
			require.Zero(t, count(t, dest, &model.Run{}))
			require.Zero(t, count(t, dest, &model.TransferReceipt{}))
			apply(false)
			require.Zero(t, apply(false).Summary.Imported)
			require.EqualValues(t, 1, count(t, dest, &model.Run{}))
			require.EqualValues(t, 3, count(t, dest, &model.Task{}))
			require.EqualValues(t, 1, count(t, dest, &model.Artifact{}))
			require.EqualValues(t, 2, count(t, dest, &model.ArtifactTask{}))
			var job model.Job
			require.NoError(t, dest.Take(&job).Error)
			require.False(t, job.Enabled)
			require.True(t, job.NoCatchup)
			require.JSONEq(t, `{"text":"schedule override"}`, string(job.RuntimeConfig.Parameters))
		})
	}
}
