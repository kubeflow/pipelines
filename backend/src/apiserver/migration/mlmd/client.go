// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd

import (
	"fmt"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// DialSource preserves the released MLMD client's receive limit for historical data.
func DialSource(address string, useTLS bool, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	limit := 100 * 1024 * 1024
	if raw := os.Getenv("METADATA_GRPC_MESSAGE_SIZE"); raw != "" {
		size, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("METADATA_GRPC_MESSAGE_SIZE must be a positive byte count up to %d", 1<<31-1)
		}
		limit = int(size)
	}
	transport := insecure.NewCredentials()
	if useTLS {
		transport = credentials.NewTLS(nil)
	}
	options = append(options, grpc.WithTransportCredentials(transport),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(limit)))
	conn, err := grpc.NewClient(address, options...)
	if err != nil {
		return nil, fmt.Errorf("connect to MLMD: %w", err)
	}
	return conn, nil
}
