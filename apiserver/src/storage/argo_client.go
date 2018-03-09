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

package storage

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

const (
	argoURLPackage = "https://%v:%v/apis/argoproj.io/v1alpha1/namespaces/default/%v"
)

type ArgoClientInterface interface {
	Request(method string, api string, body []byte) ([]byte, error)
}

type ArgoClient struct {
	K8ServiceHost string
	K8TCPPort     string
	K8Token       string
}

func initClient() http.Client {
	// TODO(yangpa): Enable SSL/TLS protection.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return http.Client{Transport: tr}
}

func (ac *ArgoClient) Request(method string, api string, body []byte) ([]byte, error) {
	client := initClient()

	requestUrl := fmt.Sprintf(argoURLPackage, ac.K8ServiceHost, ac.K8TCPPort, api)
	req, err := http.NewRequest(method, requestUrl, bytes.NewBuffer(body))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", ac.K8Token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		glog.Fatalf(err.Error())
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatalf(err.Error())
		return nil, err
	}
	return bodyBytes, nil
}
