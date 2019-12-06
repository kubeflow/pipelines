// Copyright 2019 Google LLC
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

package resource

import (
	"github.com/pkg/errors"
)

const (
	InternalError string = "InternalError"
	Unauthorized  string = "Unauthorized"
	Authorized    string = "Authorized"
)

type FakeKFAMClient struct {
	mode string
}

func NewKFAMClientFake() *FakeKFAMClient {
	return &FakeKFAMClient{
		Authorized,
	}
}

func (c *FakeKFAMClient) SetMode(mode string) {
	c.mode = mode
}

func (c *FakeKFAMClient) IsAuthorized(userIdentity string, namespace string) (bool, error) {
	if c.mode == InternalError {
		return false, errors.New("Failed to connect to the KFAM service.")
	} else if c.mode == Unauthorized {
		return false, nil
	}
	return true, nil

}
