// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentialrotation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestSecretPatchPreservesUnrelatedMetadata(t *testing.T) {
	kube := fake.NewClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: LegacySecret, Namespace: "test", ResourceVersion: "7", Annotations: map[string]string{"operator": "keep"}}, Data: map[string][]byte{"other": []byte("keep")}})
	c := &Client{Clientset: kube, Namespace: "test"}
	updated, err := c.PatchSecret(context.Background(), LegacySecret, "7", map[string][]byte{"accesskey": []byte("synthetic")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Annotations["operator"] != "keep" || string(updated.Data["other"]) != "keep" {
		t.Fatal("patch removed unrelated fields")
	}
	var patch map[string]interface{}
	if err := json.Unmarshal(kube.Actions()[0].(clienttesting.PatchAction).GetPatch(), &patch); err != nil {
		t.Fatal(err)
	}
	metadata := patch["metadata"].(map[string]interface{})
	if metadata["resourceVersion"] != "7" {
		t.Fatal("missing concurrency precondition")
	}
	if _, exists := metadata["annotations"]; exists {
		t.Fatal("data-only update owns annotations")
	}
}

func TestRemoteStorageRetriesStaleIAMRevocation(t *testing.T) {
	iamAccepted := true
	deletes := 0
	iamStatus := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		old := strings.Contains(r.Header.Get("Authorization"), "Credential="+DefaultAccessKey+"/")
		if r.Method == http.MethodHead {
			if old {
				w.WriteHeader(http.StatusForbidden)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}
		if !strings.Contains(r.Header.Get("Authorization"), "/iam/aws4_request") {
			t.Error("IAM request not signed for IAM")
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		switch r.Form.Get("Action") {
		case "GetUser":
			w.WriteHeader(http.StatusOK)
		case "ListUsers":
			switch {
			case iamStatus != 0:
				w.WriteHeader(iamStatus)
			case old && iamAccepted:
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusForbidden)
			}
		case "DeleteAccessKey":
			if r.Form.Get("AccessKeyId") != DefaultAccessKey || old {
				t.Error("wrong deletion principal or key")
			}
			deletes++
			if deletes > 1 {
				iamAccepted = false
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Error("unexpected IAM operation")
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	storage, err := NewRemoteStorage(Credentials{"synthetic-replacement", "synthetic-secret"}, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	ctx := context.Background()
	if err := storage.Verify(ctx); err != nil {
		t.Fatal(err)
	}
	if err := storage.RevokeDefault(ctx); !IsWaiting(err) {
		t.Fatalf("stale IAM should wait, got %T", err)
	}
	if err := storage.RevokeDefault(ctx); err != nil {
		t.Fatal(err)
	}
	if deletes != 2 {
		t.Fatalf("got %d deletions", deletes)
	}
	iamStatus = 503
	if _, err := storage.DefaultAccepted(ctx); err == nil {
		t.Fatal("IAM failure counted as revocation")
	}
}

func TestHealthIsIndependentOfCompletion(t *testing.T) {
	health := &Health{}
	check := func(want int) {
		t.Helper()
		response := httptest.NewRecorder()
		health.ServeHTTP(response, httptest.NewRequest("GET", "/readyz", nil))
		if response.Code != want {
			t.Fatalf("readiness %d, want %d", response.Code, want)
		}
	}
	check(503)
	health.Observe(&Waiting{Message: "DrainingWorkloads"})
	check(200)
	health.Observe(errors.New("API unavailable"))
	check(503)
	health.Observe(nil)
	check(200)
	health.lastSuccess.Store(time.Now().Add(-3 * time.Minute).UnixNano())
	check(503)
}
