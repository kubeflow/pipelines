// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

const concurrentDeliveries = 8

type concurrentWriteResult struct {
	payload  string
	artifact *model.Artifact
	link     *model.ArtifactTask
	err      error
}

// deliverConcurrently sends one write per pool at the same time; payloadFor picks each write's artifact name.
func deliverConcurrently(f productionWriteFixture, operationID string, payloadFor func(int) string) []concurrentWriteResult {
	start := make(chan struct{})
	results := make([]concurrentWriteResult, concurrentDeliveries)
	var wg sync.WaitGroup
	for i := range concurrentDeliveries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := payloadFor(i)
			<-start
			artifact, link, err := f.store(i).CreateArtifactWithTask(f.artifact(payload), f.link("model"), operationID)
			results[i] = concurrentWriteResult{payload: payload, artifact: artifact, link: link, err: err}
		}()
	}
	close(start)
	wg.Wait()
	return results
}

func TestCreateArtifactWithTask_ConcurrentSameIdentityProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			f := newProductionWriteFixture(t, dbs, d, "concurrent-identity-run")

			results := deliverConcurrently(f, "operation-concurrent", func(int) string { return "model" })

			for i, r := range results {
				require.NoError(t, r.err, "delivery %d", i)
				require.Equal(t, results[0].artifact.UUID, r.artifact.UUID, "delivery %d returned a different artifact", i)
				require.Equal(t, results[0].link.UUID, r.link.UUID, "delivery %d returned a different link", i)
			}
			artifacts, links, identities := countRetryCleanupRows(t, dbs[0])
			require.Equal(t, []int{1, 1, 1}, []int{artifacts, links, identities})
		})
	}
}

func TestCreateArtifactWithTask_ConcurrentChangedPayloadProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			f := newProductionWriteFixture(t, dbs, d, "concurrent-conflict-run")

			results := deliverConcurrently(f, "operation-conflict", func(i int) string {
				if i%2 == 0 {
					return "model"
				}
				return "different-model"
			})

			var winner *concurrentWriteResult
			for i := range results {
				if results[i].err == nil {
					winner = &results[i]
					break
				}
			}
			require.NotNil(t, winner, "one payload must win the operation identity")
			for i, r := range results {
				if r.payload == winner.payload {
					require.NoError(t, r.err, "delivery %d carries the winning payload", i)
					require.Equal(t, winner.artifact.UUID, r.artifact.UUID, "delivery %d returned a different artifact", i)
					continue
				}
				require.Equal(t, codes.InvalidArgument, statusCode(r.err), "delivery %d: %v", i, r.err)
			}
			artifacts, links, identities := countRetryCleanupRows(t, dbs[0])
			require.Equal(t, []int{1, 1, 1}, []int{artifacts, links, identities})
		})
	}
}

func TestCreateArtifactsWithTasks_PayloadConflictRollsBackBatchProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			f := newProductionWriteFixture(t, dbs, d, "batch-rollback-run")

			_, _, err := f.write(0, "operation-1")
			require.NoError(t, err)

			_, _, err = f.store(1).CreateArtifactsWithTasks(
				[]*model.Artifact{f.artifact("fresh"), f.artifact("different-model")},
				[]*model.ArtifactTask{f.link("fresh"), f.link("model")},
				[]string{"operation-fresh", "operation-1"},
			)
			require.Equal(t, codes.InvalidArgument, statusCode(err), "unexpected error: %v", err)
			artifacts, links, identities := countRetryCleanupRows(t, dbs[0])
			require.Equal(t, []int{1, 1, 1}, []int{artifacts, links, identities}, "the conflicting batch must not commit its fresh item")

			created, _, err := f.store(2).CreateArtifactsWithTasks(
				[]*model.Artifact{f.artifact("fresh")},
				[]*model.ArtifactTask{f.link("fresh")},
				[]string{"operation-fresh"},
			)
			require.NoError(t, err)
			require.Len(t, created, 1)
			artifacts, links, identities = countRetryCleanupRows(t, dbs[0])
			require.Equal(t, []int{2, 2, 2}, []int{artifacts, links, identities})
		})
	}
}
