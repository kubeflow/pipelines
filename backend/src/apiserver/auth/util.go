// Copyright 2021 Arrikto Inc.
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

package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

// singleHeaderFromMetadata tries to get a header from the grpc request
// metadata. If the header doesn't exist OR appears multiple times, it will
// return an error.
func singleHeaderFromMetadata(ctx context.Context, header string) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values, ok := md[strings.ToLower(header)]
	if !ok {
		return "", IdentityHeaderMissingError
	}
	if len(values) != 1 {
		msg := fmt.Sprintf("Request header error: unexpected number of '%s' headers. Expect 1 got %d", header, len(values))
		return "", util.NewBadRequestError(errors.New(msg), msg)
	}
	return values[0], nil

}

// singlePrefixedHeaderFromMetadata tries to get a header from the grpc request
// metadata. If the header doesn't exist OR appears multiple times OR doesn't
// start with the specified prefix, it will return an error.
func singlePrefixedHeaderFromMetadata(ctx context.Context, header string, prefix string) (string, error) {
	val, err := singleHeaderFromMetadata(ctx, header)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(val, prefix) {
		msg := fmt.Sprintf("Header '%s' is incorrectly formatted. Expected prefix '%s'", header, prefix)
		return "", util.NewBadRequestError(errors.New(msg), msg)
	}
	return strings.TrimPrefix(val, prefix), nil
}
