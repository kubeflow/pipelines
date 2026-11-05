// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestMySQLDrainUsesVerifiedHistoricalWorkflowNamespace(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	const runID = "11111111-1111-4111-8111-111111111111"
	const workflowNS = "migration-fixture"
	labels := map[string]string{util.LabelKeyWorkflowRunId: runID, util.LabelKeyWorkflowPersistedFinalState: "true"}
	manifest, err := json.Marshal(map[string]any{"metadata": metav1.ObjectMeta{
		Name: "original", Namespace: workflowNS, UID: "original-uid", Labels: labels,
	}})
	require.NoError(t, err)
	_, err = db.Exec("UPDATE run_details SET Name='original', Namespace='-', State='SUCCEEDED', FinishedAtInSec=1700000003, WorkflowRuntimeManifest=? WHERE UUID=?", string(manifest), runID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE experiments SET Namespace='-'")
	require.NoError(t, err)

	const deploymentNS = "kubeflow"
	kube := kubefake.NewClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "ml-pipeline", Namespace: deploymentNS},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "ml-pipeline-api-server", EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-mode"},
			}}},
		}}}}},
	}, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "api-mode", Namespace: deploymentNS},
		Data: map[string]string{"MULTIUSER": "false"}})
	workflow := func(namespace, uid, phase string, persisted bool) *unstructured.Unstructured {
		wf := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow", "status": map[string]any{"phase": phase},
		}}
		wf.SetName("original")
		wf.SetNamespace(namespace)
		wf.SetUID(types.UID(uid))
		wf.SetLabels(map[string]string{util.LabelKeyWorkflowRunId: runID})
		if persisted {
			wf.SetLabels(labels)
		}
		return wf
	}
	gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		singleUser bool
		workflows  []runtime.Object
		want       string
	}{
		{name: "historical", singleUser: true, workflows: []runtime.Object{workflow(workflowNS, "original-uid", "Succeeded", true)}},
		{name: "strict", workflows: []runtime.Object{workflow(workflowNS, "original-uid", "Succeeded", true)}, want: "lacks a namespace"},
		{name: "active", singleUser: true, workflows: []runtime.Object{workflow(workflowNS, "original-uid", "Running", true)}, want: "is not terminal"},
		{name: "unreported", singleUser: true, workflows: []runtime.Object{workflow(workflowNS, "original-uid", "Succeeded", false)}, want: "no stored terminal report"},
		{name: "foreign namespace", singleUser: true, workflows: []runtime.Object{workflow("other", "other-uid", "Running", false)}},
		{name: "replacement UID", singleUser: true, workflows: []runtime.Object{workflow(workflowNS, "other-uid", "Succeeded", true)}, want: "conflicts with the persisted UID"},
		{name: "deleted", singleUser: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dynamic := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{gvr: "WorkflowList"}, tc.workflows...)
			gate := mlmd.DrainGate{Database: db, Kubernetes: kube, Workflows: dynamic, Namespace: deploymentNS, Deployment: "ml-pipeline", SourceSingleUser: tc.singleUser}
			err := gate.CheckDrained(ctx)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
			} else {
				require.NoError(t, err)
			}
		})
	}
	config, err := kube.CoreV1().ConfigMaps(deploymentNS).Get(ctx, "api-mode", metav1.GetOptions{})
	require.NoError(t, err)
	config.Data["MULTIUSER"] = "true"
	_, err = kube.CoreV1().ConfigMaps(deploymentNS).Update(ctx, config, metav1.UpdateOptions{})
	require.NoError(t, err)
	gate := mlmd.DrainGate{Database: db, Kubernetes: kube, Workflows: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		Namespace: deploymentNS, Deployment: "ml-pipeline", SourceSingleUser: true}
	_, err = gate.Blockers(ctx)
	require.ErrorContains(t, err, "requires a single-user API Deployment")
}

type namespaceFixtureSource struct{ fixtureClient }

func (namespaceFixtureSource) GetExecutionTypes(context.Context, *pb.GetExecutionTypesRequest, ...grpc.CallOption) (*pb.GetExecutionTypesResponse, error) {
	return &pb.GetExecutionTypesResponse{}, nil
}

func TestMySQLMigrationChecksNamespaceAfterEligibility(t *testing.T) {
	const runID = "11111111-1111-4111-8111-111111111111"
	const workflowNamespace = "migration-fixture"
	const deploymentNamespace = "kubeflow"
	for _, tc := range []struct {
		name, executionType   string
		singleUser, wantError bool
	}{
		{name: "unsupported execution skips", executionType: "custom.Unsupported"},
		{name: "supported execution needs a verified namespace", executionType: "system.ContainerExecution", wantError: true},
		{name: "supported execution adopts the saved Workflow namespace", executionType: "system.ContainerExecution", singleUser: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, _, _, _ := releasedDatabase(t)
			manifest, err := json.Marshal(map[string]any{"metadata": metav1.ObjectMeta{
				Name: "original", Namespace: workflowNamespace, UID: "original-uid",
				Labels: map[string]string{util.LabelKeyWorkflowRunId: runID},
			}})
			require.NoError(t, err)
			_, err = db.Exec("UPDATE run_details SET Namespace='-', Name='original', WorkflowRuntimeManifest=? WHERE UUID=?", string(manifest), runID)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE experiments SET Namespace='-'")
			require.NoError(t, err)
			deployment := func(name string) *appsv1.Deployment {
				return &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: deploymentNamespace, Generation: 1},
					Spec: appsv1.DeploymentSpec{
						Replicas: proto.Int32(0),
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
						Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "ml-pipeline-api-server"}},
						}},
					},
					Status: appsv1.DeploymentStatus{ObservedGeneration: 1},
				}
			}
			gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}
			gate := mlmd.DrainGate{
				Database: db, Namespace: deploymentNamespace, Deployment: "ml-pipeline", SourceSingleUser: tc.singleUser,
				Kubernetes: kubefake.NewClientset(deployment("ml-pipeline"), deployment("ml-pipeline-scheduledworkflow")),
				Workflows:  dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{gvr: "WorkflowList"}),
			}
			source := namespaceFixtureSource{fixtureClient{
				executions: &pb.GetExecutionsResponse{Executions: []*pb.Execution{{
					Id: proto.Int64(1), Type: proto.String(tc.executionType),
					CustomProperties: map[string]*pb.Value{"namespace": stringValue(workflowNamespace)},
				}}},
				contexts: map[int64]*pb.GetContextsByExecutionResponse{1: {
					Contexts: []*pb.Context{{Type: proto.String("system.PipelineRun"), Name: proto.String(runID)}},
				}},
				artifacts: &pb.GetArtifactsResponse{Artifacts: []*pb.Artifact{{Id: proto.Int64(7), Type: proto.String("system.Dataset")}}},
				events: &pb.GetEventsByExecutionIDsResponse{Events: []*pb.Event{{
					ExecutionId: proto.Int64(1), ArtifactId: proto.Int64(7), Type: pb.Event_OUTPUT.Enum(),
					Path: &pb.Event_Path{Steps: []*pb.Event_Path_Step{{Value: &pb.Event_Path_Step_Key{Key: "dataset"}}}},
				}}},
			}}
			require.NoError(t, mlmd.ValidateSource(ctx, db, source))
			require.NoError(t, gate.Check(ctx))
			report, err := mlmd.MigrateFenced(ctx, source, db, fixtureDiskBudget, gate)
			if tc.wantError {
				require.ErrorContains(t, err, "cannot verify the persisted namespace")
				var writes int
				require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()
					AND table_name IN ('tasks_pre_migration','tasks_migration_tmp','mlmd_migration_state','artifacts','artifact_tasks')`).Scan(&writes))
				require.Zero(t, writes)
				return
			}
			require.NoError(t, err)
			if !tc.singleUser {
				require.Zero(t, report.Tasks)
				require.Equal(t, 1, report.SkippedByKind["execution"])
				require.Contains(t, report.Skipped, mlmd.Skip{
					Kind: "execution", ID: 1, Reason: `execution 1 has unsupported type "custom.Unsupported"`,
				})
				return
			}
			require.Equal(t, 1, report.Tasks)
			require.Equal(t, 1, report.Links)
			var taskNamespace, artifactNamespace string
			require.NoError(t, db.QueryRow("SELECT t.Namespace,a.Namespace FROM artifact_tasks l JOIN tasks t ON t.UUID=l.TaskID JOIN artifacts a ON a.UUID=l.ArtifactID WHERE l.TaskID=? AND l.ArtifactKey='dataset'", mlmd.ExecutionUUID(1)).Scan(&taskNamespace, &artifactNamespace))
			require.Equal(t, workflowNamespace, taskNamespace)
			require.Equal(t, workflowNamespace, artifactNamespace)
		})
	}
}
