// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiclient

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const runtimeTimingMetadataKey = "x-kfp-runtime-profile-id"

type runtimeTimingRecord struct {
	CorrelationID string  `json:"correlation_id"`
	Method        string  `json:"method"`
	TotalMS       float64 `json:"total_ms"`
	Status        string  `json:"status"`
}

func runtimeTimingDialOptions() []grpc.DialOption {
	if os.Getenv("KFP_RUNTIME_PROFILE_TIMINGS") != "true" {
		return nil
	}
	return []grpc.DialOption{grpc.WithUnaryInterceptor(runtimeTimingInterceptor(func(record runtimeTimingRecord) {
		data, err := json.Marshal(record)
		if err == nil {
			glog.Infof("KFP_RUNTIME_CLIENT_TIMING %s", data)
		}
	}))}
}

func runtimeTimingInterceptor(emit func(runtimeTimingRecord)) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		id, err := uuid.NewRandom()
		if err != nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		// Replace any caller-provided profiling ID; never copy request/token data.
		outgoing, _ := metadata.FromOutgoingContext(ctx)
		outgoing = outgoing.Copy()
		outgoing.Set(runtimeTimingMetadataKey, id.String())
		ctx = metadata.NewOutgoingContext(ctx, outgoing)
		started := time.Now()
		err = invoker(ctx, method, req, reply, cc, opts...)
		emit(runtimeTimingRecord{CorrelationID: id.String(), Method: method, TotalMS: float64(time.Since(started)) / float64(time.Millisecond), Status: status.Code(err).String()})
		return err
	}
}
