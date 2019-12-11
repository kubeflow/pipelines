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
import(
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAuthorize(t *testing.T) {
	expect_response := []byte (`{"bindings":[{"user": {"kind": "User","name": "userA@google.com"},"referredNamespace": "nsA","RoleRef": {"apiGroup": "","kind": "ClusterRole", "name":"edit"}},{"user": {"kind": "User","name": "userA@google.com"},"referredNamespace": "nsB","RoleRef": {"apiGroup": "","kind": "ClusterRole", "name":"admin"}}]}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(expect_response)
	}))
	defer srv.Close()
	fmt.Println(srv.URL)
	kfam_client := NewKFAMClient("","")
	kfam_client.kfamServiceUrl = srv.URL
	authorized, err := kfam_client.IsAuthorized("user", "nsA")
	assert.Nil(t, err)
	assert.True(t, authorized)

	authorized, err = kfam_client.IsAuthorized("user", "nsB")
	assert.Nil(t, err)
	assert.True(t, authorized)

	authorized, err = kfam_client.IsAuthorized("user", "nsC")
	assert.Nil(t, err)
	assert.False(t, authorized)
}