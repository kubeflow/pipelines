// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kfpapi

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/metadata"
)

type driverRecoveryContextKey struct{}

// WithDriverRecovery requests durable replay payloads on single-task RPCs.
// Ordinary runtime reads carry only ownership; run/list/bulk reads omit both.
func WithDriverRecovery(ctx context.Context) context.Context {
	return context.WithValue(ctx, driverRecoveryContextKey{}, true)
}

func taskRecoveryContext(ctx context.Context) context.Context {
	view := util.DriverRecoveryViewOwnership
	if enabled, _ := ctx.Value(driverRecoveryContextKey{}).(bool); enabled {
		view = util.DriverRecoveryViewFull
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set(util.DriverRecoveryViewHeader, view)
	return metadata.NewOutgoingContext(ctx, md)
}
