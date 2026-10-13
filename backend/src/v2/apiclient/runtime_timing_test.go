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
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRuntimeTimingDisabledByDefault(t *testing.T) {
	for _, value := range []string{"", "false", "TRUE"} {
		t.Setenv("KFP_RUNTIME_PROFILE_TIMINGS", value)
		if len(runtimeTimingDialOptions()) != 0 {
			t.Fatal("profiling enabled without explicit true")
		}
	}
	t.Setenv("KFP_RUNTIME_PROFILE_TIMINGS", "true")
	if len(runtimeTimingDialOptions()) != 1 {
		t.Fatal("profiling interceptor missing")
	}
}

func TestRuntimeTimingPreservesRPCAndOnlyLogsSafeFields(t *testing.T) {
	var records []runtimeTimingRecord
	interceptor := runtimeTimingInterceptor(func(record runtimeTimingRecord) { records = append(records, record) })
	source := metadata.Pairs("authorization", "secret-token", runtimeTimingMetadataKey, "caller-value")
	ctx := metadata.NewOutgoingContext(context.Background(), source)
	failure := status.Error(codes.PermissionDenied, "secret error detail")
	request, reply := new(int), new(int)
	var ids []string
	for i := 0; i < 2; i++ {
		err := interceptor(ctx, "/RunService/UpdateTask", request, reply, nil, func(callctx context.Context, method string, req, res any, _ *grpc.ClientConn, opts ...grpc.CallOption) error {
			if req != request || res != reply || method != "/RunService/UpdateTask" || len(opts) != 1 {
				t.Fatal("changed RPC")
			}
			md, _ := metadata.FromOutgoingContext(callctx)
			if md.Get("authorization")[0] != "secret-token" {
				t.Fatal("lost authorization")
			}
			id := md.Get(runtimeTimingMetadataKey)
			if len(id) != 1 {
				t.Fatal(id)
			}
			parsed, parseErr := uuid.Parse(id[0])
			if parseErr != nil || parsed.String() != id[0] {
				t.Fatal(id)
			}
			ids = append(ids, id[0])
			return failure
		}, grpc.WaitForReady(true))
		if err != failure {
			t.Fatal("changed error")
		}
	}
	if ids[0] == ids[1] || source.Get(runtimeTimingMetadataKey)[0] != "caller-value" {
		t.Fatal("IDs reused or original context mutated")
	}
	for i, record := range records {
		if record.CorrelationID != ids[i] || record.Status != "PermissionDenied" || record.TotalMS < 0 {
			t.Fatal(record)
		}
		data, _ := json.Marshal(record)
		if strings.Contains(string(data), "secret") || strings.Contains(string(data), "authorization") {
			t.Fatal("sensitive diagnostic")
		}
	}
}
