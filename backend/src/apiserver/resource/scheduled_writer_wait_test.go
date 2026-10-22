// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestScheduleWriterWaitFailureDoesNotMutate(t *testing.T) {
	cases := []struct {
		name  string
		code  codes.Code
		setup func() (context.Context, context.CancelFunc, func(context.Context) error)
	}{
		{"canceled", codes.Canceled, func() (context.Context, context.CancelFunc, func(context.Context) error) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel, func(context.Context) error { cancel(); return context.Canceled }
		}},
		{"deadline", codes.DeadlineExceeded, func() (context.Context, context.CancelFunc, func(context.Context) error) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			return ctx, cancel, func(context.Context) error { return context.DeadlineExceeded }
		}},
		{"success_after_cancel", codes.Canceled, func() (context.Context, context.CancelFunc, func(context.Context) error) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel, func(context.Context) error { cancel(); return nil }
		}},
		{"wait_budget_exhausted", codes.Unavailable, func() (context.Context, context.CancelFunc, func(context.Context) error) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel, func(context.Context) error { return context.DeadlineExceeded }
		}},
		{"forbidden_configuration", codes.FailedPrecondition, func() (context.Context, context.CancelFunc, func(context.Context) error) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel, func(context.Context) error {
				return apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "writer", errors.New("missing RBAC"))
			}
		}},
	}
	for _, tc := range cases {
		for _, operation := range []string{"create", "enable", "disable", "transfer"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				manager, clients := onlineAdoptionManager(t)
				job := onlineLegacyJob(t, clients, operation != "enable")
				before, err := clients.JobStore().GetJob(job.UUID)
				require.NoError(t, err)
				live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(context.Background(), job.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				liveBefore := live.DeepCopy()
				db, err := clients.TransferDB()
				require.NoError(t, err)
				var jobsBefore int64
				require.NoError(t, db.Model(&model.Job{}).Count(&jobsBefore).Error)
				swfClient := newTransferSwfClient()
				manager.swfClient = swfClient
				ctx, cancel, check := tc.setup()
				defer cancel()
				manager.options.ScheduleWritersReady = check
				switch operation {
				case "create":
					created, createErr := manager.CreateJob(ctx, &model.Job{Namespace: "ns1"})
					require.Nil(t, created)
					err = createErr
				case "enable", "disable":
					err = manager.ChangeJobMode(ctx, job.UUID, operation == "enable")
				case "transfer":
					staged, stageErr := (transferSchedules{r: manager}).Prepare(ctx, "source", job, "digest", nil, false)
					require.Nil(t, staged, "failed waiting must not return a candidate for SQL finalization")
					err = stageErr
				}
				require.Error(t, err)
				status := util.ToGRPCStatus(err)
				require.Equal(t, tc.code, status.Code())
				require.Equal(t, tc.code, grpcstatus.Code(util.ToGRPCError(err)))
				marker := false
				for _, detail := range status.Details() {
					if info, ok := detail.(*errdetails.ErrorInfo); ok && info.Reason == "SCHEDULE_HANDOFF_IN_PROGRESS" {
						marker = true
					}
				}
				require.Equal(t, tc.code == codes.Unavailable, marker)
				require.Empty(t, swfClient.client.Actions(), "waiting must precede any schedule Kubernetes access")
				after, err := clients.JobStore().GetJob(job.UUID)
				require.NoError(t, err)
				require.Equal(t, before, after)
				liveAfter, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(context.Background(), job.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				require.Equal(t, liveBefore, liveAfter)
				var jobsAfter, states, receipts int64
				require.NoError(t, db.Model(&model.Job{}).Count(&jobsAfter).Error)
				require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&states).Error)
				require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Count(&receipts).Error)
				require.Equal(t, jobsBefore, jobsAfter)
				require.Zero(t, states)
				require.Zero(t, receipts)
			})
		}
	}
}

func TestScheduleWriterWaitCompletesBeforeDisableMutation(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	entered := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.options.ScheduleWritersReady = util.NewScheduleWriterHandoffWaiter(func(ctx context.Context) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	result := make(chan error, 1)
	go func() { result <- manager.ChangeJobMode(ctx, job.UUID, false) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("mode change did not enter handoff wait")
	}
	before, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.True(t, before.Enabled, "an outstanding disable must not mutate before handoff")
	close(release)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("mode change did not resume after handoff")
	}
	after, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, after.Enabled)
}
