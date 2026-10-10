// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// recurringRunOperationError translates boundary failures after Kubernetes
// conflict retries have finished. Already typed authorization errors survive.
func recurringRunOperationError(err error, id string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Errorf(codes.DeadlineExceeded, "Synchronization of recurring run %s timed out; retry the operation", id)
	}
	if errors.Is(err, context.Canceled) {
		return status.Errorf(codes.Canceled, "Synchronization of recurring run %s was canceled; retry the operation if still needed", id)
	}
	var userError *util.UserError
	if errors.As(err, &userError) {
		return err
	}
	if apierrors.IsNotFound(err) {
		return util.NewNotFoundError(err, "The backing ScheduledWorkflow for recurring run %s is missing; refresh the recurring-run list and, if its backing resource was deleted, delete and recreate this recurring run through the KFP API", id)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return util.NewNotFoundError(err, "Recurring run %s no longer exists; refresh the recurring-run list before retrying", id)
	}
	if apierrors.IsConflict(err) || apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) || apierrors.IsServiceUnavailable(err) || apierrors.IsTooManyRequests(err) {
		return util.NewUnavailableServerError(err, "Recurring run %s could not be synchronized; retry shortly while automatic recovery continues", id)
	}
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
		return util.NewPermissionDeniedError(err, "The API server cannot access the ScheduledWorkflow for recurring run %s; restore its Kubernetes permissions before retrying", id)
	}
	if apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) {
		return util.NewFailedPreconditionError(err, "The ScheduledWorkflow for recurring run %s cannot be synchronized; correct the invalid Kubernetes resource before retrying", id)
	}
	if code := status.Code(err); code != codes.Unknown {
		return err
	}
	return util.NewInternalServerError(err, "Cannot synchronize recurring run %s; retry and inspect the API-server logs if the problem persists", id)
}

func recurringRunIdentityError(id string) error {
	return util.NewFailedPreconditionError(fmt.Errorf("ScheduledWorkflow identity differs from its job"), "Recurring run %s no longer matches its backing ScheduledWorkflow; use the original resource or delete and recreate the recurring run through the KFP API", id)
}
