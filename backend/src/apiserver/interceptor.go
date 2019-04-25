// Copyright 2018 Google LLC
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

package main

import (
	"context"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc"
)

// apiServerInterceptor implements UnaryServerInterceptor that provides the common wrapping logic
// to be executed before and after all API handler calls, e.g. Logging, error handling.
// For more details, see https://github.com/grpc/grpc-go/blob/master/interceptor.go
func apiServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	glog.Infof("%v handler starting", info.FullMethod)
	resp, err = handler(ctx, req)
	if err != nil {
		util.LogError(util.Wrapf(err, "%s call failed", info.FullMethod))
		// Convert error to gRPC errors
		err = util.ToGRPCError(err)
		return
	}
	glog.Infof("%v handler finished", info.FullMethod)
	return
}
