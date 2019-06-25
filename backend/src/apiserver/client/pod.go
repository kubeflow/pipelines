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
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/restclient"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

func CreatePodClient() (*kubeclient.Client, error) {
	client, err := kubeclient.New(&restclient.Config{

	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CreatePodClientOrFatal() *kubeclient.Client{
	var podClient *kubeclient.Client
	var err error
	var operation = func() error {
		podClient, err = CreatePodClient()
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	if err := backoff.Retry(operation, b); err != nil {
		glog.Fatalf("Failed to create Pod client. Error: %v", err)
	}
	return podClient
}
