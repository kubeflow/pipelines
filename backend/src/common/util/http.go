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

package util

import (
	"net/http"
	"net/url"
)

type HTTPInterface interface {
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

type HTTP struct{}

func NewRealHTTP() HTTPInterface {
	return &HTTP{}
}

func (r *HTTP) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return http.PostForm(url, data)
}

type FakeHTTP struct {
	postForm *FakeHTTPPostForm
}

func NewFakeHTTP(resp *http.Response, err error) HTTPInterface {
	return &FakeHTTP{postForm: &FakeHTTPPostForm{resp: resp, err: err}}
}

// FakeHTTPPostForm is a fake implementation of the HTTPPostFormInterface used for testing.
// It always generates the http.Response and error provided during instantiation.
type FakeHTTPPostForm struct {
	resp *http.Response
	err  error
}

func (f *FakeHTTP) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return f.postForm.resp, f.postForm.err
}