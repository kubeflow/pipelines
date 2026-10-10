// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestRecurringRunOperationErrors(t *testing.T) {
	resource := schema.GroupResource{Group: "kubeflow.org", Resource: "scheduledworkflows"}
	for _, tc := range []struct {
		name    string
		err     error
		code    codes.Code
		message string
	}{
		{"missing CR", apierrors.NewNotFound(resource, "schedule"), codes.NotFound, "delete and recreate this recurring run through the KFP API"},
		{"changed identity", recurringRunIdentityError("schedule"), codes.FailedPrecondition, "use the original resource or delete and recreate"},
		{"conflict", apierrors.NewConflict(resource, "schedule", errors.New("resource version changed")), codes.Unavailable, "retry shortly"},
		{"Kubernetes timeout", apierrors.NewTimeoutError("timeout", 1), codes.Unavailable, "retry shortly"},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded, "retry the operation"},
		{"canceled", context.Canceled, codes.Canceled, "retry the operation if still needed"},
		{"database", errors.New("database connection failed"), codes.Internal, "inspect the API-server logs"},
		{"deleted job", gorm.ErrRecordNotFound, codes.NotFound, "refresh the recurring-run list"},
		{"forbidden", apierrors.NewForbidden(resource, "schedule", errors.New("forbidden")), codes.PermissionDenied, "restore its Kubernetes permissions"},
		{"existing not found", util.NewNotFoundError(errors.New("missing"), "Recurring run was deleted"), codes.NotFound, "Recurring run was deleted"},
		{"existing authorization", util.NewPermissionDeniedError(errors.New("denied"), "Account is not authorized"), codes.PermissionDenied, "Account is not authorized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := recurringRunOperationError(tc.err, "schedule")
			require.Equal(t, tc.code, status.Code(err))
			require.Contains(t, status.Convert(err).Message(), tc.message)
		})
	}
	require.NoError(t, recurringRunOperationError(nil, "schedule"))
}

type recurringRunErrorCR struct {
	swfclient.ScheduledWorkflowInterface
	readError      error
	replacementUID bool
}

func (c *recurringRunErrorCR) ScheduledWorkflow(string) swfclient.ScheduledWorkflowInterface {
	return c
}
func (c *recurringRunErrorCR) Get(ctx context.Context, name string, options metav1.GetOptions) (*swfapi.ScheduledWorkflow, error) {
	if c.readError != nil {
		return nil, c.readError
	}
	live, err := c.ScheduledWorkflowInterface.Get(ctx, name, options)
	if err == nil && c.replacementUID {
		live = live.DeepCopy()
		live.UID = types.UID("replacement")
	}
	return live, err
}

func TestRecurringRunModeErrorsHaveActionableStatus(t *testing.T) {
	for _, tc := range []struct {
		name           string
		readError      error
		replacementUID bool
		code           codes.Code
		message        string
	}{
		{name: "missing original CR", readError: apierrors.NewNotFound(schema.GroupResource{Resource: "scheduledworkflows"}, "schedule"), code: codes.NotFound, message: "delete and recreate this recurring run through the KFP API"},
		{name: "replaced CR", replacementUID: true, code: codes.FailedPrecondition, message: "use the original resource or delete and recreate"},
		{name: "Kubernetes request expired", readError: context.DeadlineExceeded, code: codes.DeadlineExceeded, message: "retry the operation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, clients := onlineAdoptionManager(t)
			job := onlineLegacyJob(t, clients, false)
			manager.swfClient = &recurringRunErrorCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), readError: tc.readError, replacementUID: tc.replacementUID}
			err := manager.changeAdoptableJobMode(context.Background(), job, true)
			require.Equal(t, tc.code, status.Code(err))
			require.Contains(t, status.Convert(err).Message(), tc.message)
			stored, err := clients.JobStore().GetJob(job.UUID)
			require.NoError(t, err)
			require.False(t, stored.Enabled)
		})
	}
}

func TestRecurringRunModeDatabaseFailureIsInternal(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, false)
	db, err := manager.recurringRunAdoptionDB(context.Background())
	require.NoError(t, err)
	require.NoError(t, db.Migrator().DropTable("jobs"))
	err = manager.changeAdoptableJobMode(context.Background(), job, true)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "inspect the API-server logs")
}
