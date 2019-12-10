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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

type KFAMClientInterface interface {
	IsAuthorized(userIdentity string, namespace string) (bool, error)
}

type KFAMClient struct {
	kfamServiceUrl string
}

type User struct {
	Kind string
	Name string
}

type RoleRef struct {
	ApiGroup string
	Kind     string
	Name     string
}

type Binding struct {
	User              User
	ReferredNamespace string
	RoleRef           RoleRef
}

type Bindings struct {
	Bindings []Binding
}

const (
	HTTP_TIMEOUT_SECONDS = 10
)

func (c *KFAMClient) IsAuthorized(userIdentity string, namespace string) (bool, error) {
	req, err := http.NewRequest("GET", c.kfamServiceUrl, nil)
	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to create a KFAM http request.")
	}
	q := req.URL.Query()
	q.Add("user", userIdentity)
	req.URL.RawQuery = q.Encode()

	var httpClient = &http.Client{Timeout: HTTP_TIMEOUT_SECONDS * time.Second}

	resp, err := httpClient.Get(req.URL.String())
	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to connect to the KFAM service.")
	}
	if resp.StatusCode != http.StatusOK {
		return false, util.NewInternalServerError(errors.New("Requests to the KFAM service failed."), resp.Status)
	}
	defer resp.Body.Close()

	jsonBindings := new(Bindings)
	err = json.NewDecoder(resp.Body).Decode(jsonBindings)

	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to parse KFAM response.")
	}

	nsFound := false
	for _, jsonBinding := range jsonBindings.Bindings {
		if jsonBinding.ReferredNamespace == namespace {
			nsFound = true
			break
		}
	}
	return nsFound, nil
}

func NewKFAMClient(kfamServiceHost string, kfamServicePort string) *KFAMClient {
	kfamServiceUrl := fmt.Sprintf("http://%s:%s/kfam/v1/bindings", kfamServiceHost, kfamServicePort)
	return &KFAMClient{kfamServiceUrl}
}
