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
)

type HTTPHeaderAuthenticator struct {
	userIDHeader string
	userIDPrefix string
}

func NewHTTPHeaderAuthenticator(header, prefix string) *HTTPHeaderAuthenticator {
	return &HTTPHeaderAuthenticator{userIDHeader: header, userIDPrefix: prefix}
}

func (ha *HTTPHeaderAuthenticator) GetUserIdentity(ctx context.Context) (string, error) {
	userID, err := singlePrefixedHeaderFromMetadata(ctx, ha.userIDHeader, ha.userIDPrefix)
	if err != nil {
		return "", err
	}
	return userID, nil
}
