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
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestSingleHeaderFromMetadata(t *testing.T) {
	header := "expected-header"
	expectedValue := "user"

	md := metadata.New(map[string]string{header: expectedValue})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	value, err := singleHeaderFromMetadata(ctx, header)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestSingleHeaderFromMetadataErrorNone(t *testing.T) {
	header := "expected-header"

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := singleHeaderFromMetadata(ctx, header)
	assert.NotNil(t, err)
	assert.Equal(t, IdentityHeaderMissingError, err)
}

func TestSingleHeaderFromMetadataErrorMultiple(t *testing.T) {
	header := "expected-header"

	md := metadata.New(map[string]string{header: "user1"})
	md.Append(header, "user2")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := singleHeaderFromMetadata(ctx, header)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		fmt.Sprintf("Request header error: unexpected number of '%s' headers. Expect 1 got 2", header),
	)
}

func TestSinglePrefixedHeaderFromMetadata(t *testing.T) {
	header := "expected-header"
	expectedPrefix := "expected-prefix"
	expectedValue := "user"

	md := metadata.New(map[string]string{header: expectedPrefix + expectedValue})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	value, err := singlePrefixedHeaderFromMetadata(ctx, header, expectedPrefix)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestSinglePrefixedHeaderFromMetadataError(t *testing.T) {
	header := "expected-header"
	expectedPrefix := "expected-prefix"
	actualPrefix := "actual-prefix"

	md := metadata.New(map[string]string{header: actualPrefix + "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := singlePrefixedHeaderFromMetadata(ctx, header, expectedPrefix)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		fmt.Sprintf("Header '%s' is incorrectly formatted. Expected prefix '%s'", header, expectedPrefix),
	)
}
