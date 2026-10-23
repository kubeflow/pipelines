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

package util

import (
	"context"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// NewScheduleWriterHandoffWaiter absorbs short rollout pauses before API mutations.
// Each request waits at most 30 seconds, including time behind another caller.
// One caller polls; canceling it lets another caller take over without canceling
// peers. No polling goroutine outlives its request. Background adoption and
// controller startup continue using the underlying handoff check directly.
func NewScheduleWriterHandoffWaiter(check func(context.Context) error) func(context.Context) error {
	return newScheduleWriterHandoffWaiter(check, 30*time.Second, time.Second)
}

func newScheduleWriterHandoffWaiter(check func(context.Context) error, timeout, interval time.Duration) func(context.Context) error {
	var complete atomic.Bool
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return func(ctx context.Context) error {
		waitCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := waitCtx.Err(); err != nil {
			return err
		}
		if complete.Load() {
			return waitCtx.Err()
		}
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		case <-gate:
		}
		defer func() { gate <- struct{}{} }()
		for {
			if err := waitCtx.Err(); err != nil {
				return err
			}
			if complete.Load() {
				return waitCtx.Err()
			}
			err := check(waitCtx)
			if err == nil {
				complete.Store(true)
				return waitCtx.Err()
			}
			if IsScheduleWriterConfigurationError(err) {
				return err
			}
			timer := time.NewTimer(interval)
			select {
			case <-waitCtx.Done():
				timer.Stop()
				return waitCtx.Err()
			case <-timer.C:
			}
		}
	}
}

// IsScheduleWriterConfigurationError identifies failures that waiting cannot fix.
// Missing resources can occur during an apply and remain retryable.
func IsScheduleWriterConfigurationError(err error) bool {
	return apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) || apierrors.IsBadRequest(err) || apierrors.IsInvalid(err)
}
