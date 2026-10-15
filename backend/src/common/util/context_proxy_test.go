// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestHTTPProxyContextIsolation(t *testing.T) {
	base := context.Background()
	_, ok := util.HTTPProxyFrom(base)
	require.False(t, ok)
	expected, _ := url.Parse("http://proxy.example")
	override := util.WithHTTPProxy(base, func(*http.Request) (*url.URL, error) { return expected, nil })
	resolver, ok := util.HTTPProxyFrom(override)
	require.True(t, ok)
	actual, err := resolver(&http.Request{})
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	direct := util.WithHTTPProxy(base, nil)
	resolver, ok = util.HTTPProxyFrom(direct)
	require.True(t, ok)
	require.Nil(t, resolver)
	_, ok = util.HTTPProxyFrom(base)
	require.False(t, ok)
}
