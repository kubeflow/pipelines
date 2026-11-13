// Copyright 2026 The Kubeflow Authors
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestArchiveKeyPolicy(t *testing.T) {
	identity := &RunLogArchiveContext{RunID: "run-a", Namespace: "tenant-a", WorkflowName: "workflow-a"}
	good := "private-artifacts/tenant-a/workflow-a/2026/11/12/pod-a/main.log"
	for _, tc := range []struct {
		name, key, format string
		allowed           bool
	}{
		{"standard", good, "", true},
		{"other namespace", strings.Replace(good, "tenant-a", "tenant-b", 1), "", false},
		{"other workflow", strings.Replace(good, "workflow-a", "workflow-b", 1), "", false},
		{"retry child pod", strings.Replace(good, "pod-a", "pod-b", 1), "", true},
		{"escaped segment", strings.Replace(good, "pod-a", "%70od-a", 1), "", false},
		{"traversal", strings.Replace(good, "pod-a", "../pod-a", 1), "", false},
		{"invalid date", strings.Replace(good, "/11/12/", "/02/30/", 1), "", false},
		{"custom root", "logs/tenant-a/workflow-a/pod-a/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}", true},
		{"partial date", "logs/tenant-a/workflow-a/2026/pod-a/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{pod.name}}", true},
		{"invalid partial date", "logs/tenant-a/workflow-a/13/pod-a/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/{{workflow.creationTimestamp.m}}/{{pod.name}}", false},
		{"repeated pod", "logs/tenant-a/workflow-a/pod-b/pod-b/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}/{{pod.name}}", true},
		{"inconsistent repeated pod", "logs/tenant-a/workflow-a/pod-b/pod-c/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}/{{pod.name}}", false},
		{"missing namespace", "logs/workflow-a/pod-a/main.log", "logs/{{workflow.name}}/{{pod.name}}", false},
		{"missing workflow", "logs/tenant-a/pod-a/main.log", "logs/{{workflow.namespace}}/{{pod.name}}", false},
		{"literal pod", "logs/tenant-a/workflow-a/pod-a/main.log", "logs/{{workflow.namespace}}/{{workflow.name}}/pod-a", false},
		{"partial placeholder", "logs/prefix-tenant-a/workflow-a/pod-a/main.log", "logs/prefix-{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}", false},
		{"unknown placeholder", good, strings.Replace(defaultLogArchiveKeyFormat, "{{workflow.name}}", "{{workflow.uid}}", 1), false},
		{"query", good + "?x", "", false},
		{"control", strings.Replace(good, "pod-a", "pod-\ta", 1), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KFP_LOG_ARCHIVE_KEY_FORMAT", tc.format)
			err := authorizeArchiveKey(tc.key, "pod-a", identity)
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestArchiveNativeContextValidation(t *testing.T) {
	manager := &ResourceManager{}
	for _, tc := range []struct {
		name, namespace, workflow string
		imported, allowed         bool
	}{
		{"native", "tenant-a", "workflow-a", false, true},
		{"imported", "tenant-a", "workflow-a", true, false},
		{"invalid namespace", "tenant/a", "workflow-a", false, false},
		{"missing workflow", "tenant-a", "", false, false},
		{"invalid workflow", "tenant-a", "../workflow", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := &model.Run{UUID: "run-a", Namespace: tc.namespace, K8SName: tc.workflow}
			if tc.imported {
				source := "source"
				run.ImportedFrom = &source
			}
			identity, err := manager.runLogArchiveContext(run)
			if tc.allowed {
				require.NoError(t, err)
				require.Equal(t, &RunLogArchiveContext{RunID: "run-a", Namespace: "tenant-a", WorkflowName: "workflow-a"}, identity)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMultiUserArchiveUsesNativeIdentity(t *testing.T) {
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	t.Setenv("KFP_LOG_ARCHIVE_KEY_FORMAT", "")
	identity := &RunLogArchiveContext{RunID: "run-a", Namespace: "tenant-a", WorkflowName: "workflow-a"}
	good := "private-artifacts/tenant-a/workflow-a/2026/11/12/pod-a/main.log"
	for _, key := range []string{good, strings.Replace(good, "tenant-a", "tenant-b", 1), strings.Replace(good, "workflow-a", "workflow-b", 1), strings.Replace(good, "pod-a", "pod-b", 1)} {
		t.Run(key, func(t *testing.T) {
			// A forged namespace/name and bucket are not authorization inputs. API
			// storage reads use its configured bucket, not this repository claim.
			manifest := fmt.Sprintf(`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"forged","namespace":"forged"},"status":{"nodes":{"pod-a":{"outputs":{"artifacts":[{"name":"main-logs","s3":{"bucket":"forged-bucket","key":%q}}]}}}}}`, key)
			objects := &readerOnlyObjectStore{files: map[string][]byte{key: []byte("synthetic log")}}
			manager := &ResourceManager{objectStore: objects, logArchive: archive.NewLogArchive("/logs", "main.log")}
			var dst bytes.Buffer
			err := manager.readRunLogFromArchive(context.Background(), manifest, "pod-a", &dst, identity)
			if key == good || key == strings.Replace(good, "pod-a", "pod-b", 1) {
				require.NoError(t, err)
				require.Equal(t, "synthetic log", dst.String())
				require.Equal(t, []string{key}, objects.getFileReaderPaths)
			} else {
				require.Error(t, err)
				require.Empty(t, dst.String())
				require.Empty(t, objects.getFileReaderPaths)
			}
		})
	}
}

func TestArchiveHintsDoNotReplaceNativeIdentity(t *testing.T) {
	identity := &RunLogArchiveContext{RunID: "run-a", Namespace: "tenant-a", WorkflowName: "workflow-a"}
	addArchiveHints(identity, `{"metadata":{"namespace":"forged","name":"forged","creationTimestamp":"2026-11-12T12:00:00Z"},"status":{"artifactRepositoryRef":{"artifactRepository":{"s3":{"bucket":"foreign-bucket","endpoint":"foreign.example","insecure":true,"secretKeySecret":{"key":"not-returned"}}}},"nodes":{"pod-a":{"outputs":{"artifacts":[{"name":"main-logs","s3":{"key":"foreign-key"}}]}}}}`, "pod-a")
	require.Equal(t, "tenant-a", identity.Namespace)
	require.Equal(t, "workflow-a", identity.WorkflowName)
	require.Equal(t, "foreign-bucket", identity.ArchiveBucket)
	require.Equal(t, "foreign.example", identity.ArchiveEndpoint)
	require.Equal(t, "foreign-key", identity.ArchiveKey)
	require.Equal(t, "2026-11-12T12:00:00Z", identity.WorkflowCreationTimestamp)
	require.NotNil(t, identity.ArchiveInsecure)
	require.True(t, *identity.ArchiveInsecure)
	addArchiveHints(identity, "invalid", "pod-a")
	require.Equal(t, "workflow-a", identity.WorkflowName)
}

func TestArchiveHintsResolveRetryChild(t *testing.T) {
	identity := &RunLogArchiveContext{RunID: "run-a", Namespace: "tenant-a", WorkflowName: "workflow-a"}
	key := "private-artifacts/tenant-a/workflow-a/2026/11/12/child-pod/main.log"
	manifest := fmt.Sprintf(`{"status":{"nodes":{"retry-parent":{"id":"retry-parent","type":"Retry","children":["child"]},"child":{"id":"child","outputs":{"artifacts":[{"name":"main-logs","s3":{"key":%q}}]}}}}}`, key)
	addArchiveHints(identity, manifest, "retry-parent")
	require.Equal(t, key, identity.ArchiveKey)
	t.Setenv("KFP_LOG_ARCHIVE_KEY_FORMAT", "")
	require.NoError(t, authorizeArchiveKey(identity.ArchiveKey, "retry-parent", identity))
}

func TestArchiveContextResolvesLegacyNamespace(t *testing.T) {
	for _, mode := range []string{"run reference", "experiment column", "experiment reference", "unresolved"} {
		t.Run(mode, func(t *testing.T) {
			clients, manager, run := initWithOneTimeRun(t)
			defer clients.Close()
			previous := viper.Get(common.MultiUserMode)
			viper.Set(common.MultiUserMode, true)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
			_, err := clients.DB().Exec(`UPDATE run_details SET Namespace = '', K8SName = 'workflow-a' WHERE UUID = ?`, run.UUID)
			require.NoError(t, err)
			_, err = clients.DB().Exec(`UPDATE experiments SET Namespace = '' WHERE UUID = ?`, run.ExperimentId)
			require.NoError(t, err)
			_, err = clients.DB().Exec(`DELETE FROM resource_references WHERE (ResourceUUID = ? OR ResourceUUID = ?) AND ReferenceType = ?`, run.UUID, run.ExperimentId, model.NamespaceResourceType)
			require.NoError(t, err)
			switch mode {
			case "experiment column":
				_, err = clients.DB().Exec(`UPDATE experiments SET Namespace = 'team-a' WHERE UUID = ?`, run.ExperimentId)
				require.NoError(t, err)
			case "run reference", "experiment reference":
				ref := model.ResourceReference{ResourceUUID: run.UUID, ResourceType: model.RunResourceType, ReferenceType: model.NamespaceResourceType, ReferenceUUID: "team-a", Relationship: model.OwnerRelationship}
				if mode == "experiment reference" {
					ref.ResourceUUID, ref.ResourceType = run.ExperimentId, model.ExperimentResourceType
				}
				payload, err := json.Marshal(ref)
				require.NoError(t, err)
				_, err = clients.DB().Exec(`INSERT INTO resource_references (ResourceUUID,ResourceType,ReferenceUUID,ReferenceName,ReferenceType,Relationship,Payload) VALUES (?,?,?,?,?,?,?)`, ref.ResourceUUID, ref.ResourceType, ref.ReferenceUUID, "", ref.ReferenceType, ref.Relationship, string(payload))
				require.NoError(t, err)
			}
			identity, err := manager.GetRunLogArchiveContext(run.UUID, "pod-a")
			if mode == "unresolved" {
				require.Error(t, err)
				require.Nil(t, identity)
			} else {
				require.NoError(t, err)
				require.Equal(t, "team-a", identity.Namespace)
				require.Equal(t, "workflow-a", identity.WorkflowName)
				require.Equal(t, run.UUID, identity.RunID)
			}
		})
	}
}
