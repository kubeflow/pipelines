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
	"fmt"
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

func TestRemoteStorageRevocationRequiresAuthenticationEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, s3Code, iamCode string
		wantError, accepted   bool
	}{
		{"removed", "InvalidAccessKeyId", "InvalidAccessKeyId", false, false},
		{"changed-secret", "SignatureDoesNotMatch", "SignatureDoesNotMatch", false, false},
		{"permission-denied", "AccessDenied", "AccessDenied", true, false},
		{"s3-ambiguous", "AccessDenied", "InvalidAccessKeyId", true, false},
		{"iam-ambiguous", "InvalidAccessKeyId", "AccessDenied", true, false},
		{"iam-still-accepts", "InvalidAccessKeyId", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprintf(w, "<Error><Code>%s</Code></Error>", tc.s3Code)
					return
				}
				if tc.iamCode == "" {
					fmt.Fprint(w, "<ListUsersResponse/>")
					return
				}
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintf(w, "<ErrorResponse><Error><Code>%s</Code></Error></ErrorResponse>", tc.iamCode)
			}))
			defer server.Close()
			storage, err := NewRemoteStorage(testPair, server.URL, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			accepted, err := storage.DefaultAccepted(context.Background())
			if (err != nil) != tc.wantError || accepted != tc.accepted {
				t.Fatalf("accepted=%v err=%v", accepted, err)
			}
		})
	}
}

func TestRemoteStorageDiscoversOwnerAndRefusesAmbiguousDeletion(t *testing.T) {
	for _, owners := range [][]string{{"minio"}, {"legacy-fixture"}, {"kubeflow-admin", "operator-custom"}} {
		t.Run(strings.Join(owners, "-"), func(t *testing.T) {
			removed := false
			var deleted []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				old := strings.Contains(r.Header.Get("Authorization"), "Credential="+DefaultAccessKey+"/")
				if r.Method == http.MethodGet {
					if !removed {
						fmt.Fprint(w, "<LocationConstraint/>")
						return
					}
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, "<Error><Code>InvalidAccessKeyId</Code></Error>")
					return
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
					return
				}
				if old && removed {
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, "<Error><Code>InvalidAccessKeyId</Code></Error>")
					return
				}
				switch r.Form.Get("Action") {
				case "ListUsers":
					fmt.Fprint(w, "<ListUsersResponse><ListUsersResult><Users>")
					for _, owner := range owners {
						fmt.Fprintf(w, "<member><UserName>%s</UserName></member>", owner)
					}
					fmt.Fprint(w, "</Users><IsTruncated>false</IsTruncated></ListUsersResult></ListUsersResponse>")
				case "ListAccessKeys":
					fmt.Fprintf(w, "<ListAccessKeysResponse><ListAccessKeysResult><AccessKeyMetadata><member><AccessKeyId>%s</AccessKeyId></member></AccessKeyMetadata></ListAccessKeysResult></ListAccessKeysResponse>", DefaultAccessKey)
				case "DeleteAccessKey":
					if old || r.Form.Get("AccessKeyId") != DefaultAccessKey {
						t.Error("wrong deletion identity")
					}
					deleted = append(deleted, r.Form.Get("UserName"))
					removed = true
					fmt.Fprint(w, "<DeleteAccessKeyResponse/>")
				default:
					t.Error("unexpected IAM operation")
				}
			}))
			defer server.Close()
			storage, err := NewRemoteStorage(testPair, server.URL, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			err = storage.RevokeDefault(context.Background())
			if len(owners) == 1 {
				if err != nil || len(deleted) != 1 || deleted[0] != owners[0] {
					t.Fatalf("deletion=%v err=%v", deleted, err)
				}
			} else if err == nil || len(deleted) != 0 {
				t.Fatalf("ambiguous deletion=%v err=%v", deleted, err)
			}
		})
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

func TestRevocationDoesNotDeleteFromStaleS3Acceptance(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, "<LocationConstraint/>")
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("Action") == "DeleteAccessKey" {
			deleted = true
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "<Error><Code>SignatureDoesNotMatch</Code></Error>")
	}))
	defer server.Close()
	storage, err := NewRemoteStorage(testPair, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if err := storage.RevokeDefault(context.Background()); err == nil || deleted {
		t.Fatalf("deleted=%v err=%v", deleted, err)
	}
}

func TestRevocationRejectsIAMErrorBodyOnSuccessfulHTTP(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, "<LocationConstraint/>")
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		if r.Form.Get("Action") == "DeleteAccessKey" {
			deleted = true
		}
		fmt.Fprint(w, "<ErrorResponse><Error><Code>AccessDenied</Code></Error></ErrorResponse>")
	}))
	defer server.Close()
	storage, err := NewRemoteStorage(testPair, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	err = storage.RevokeDefault(context.Background())
	if err == nil || deleted || !strings.Contains(err.Error(), "S3 and IAM disagree") {
		t.Fatalf("deleted=%v error=%v", deleted, err)
	}
}
