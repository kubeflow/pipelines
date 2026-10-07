// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	kubernetesmodel "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swffake "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/fake"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type transferFakeSWF struct{ client *swffake.Clientset }

func (c transferFakeSWF) ScheduledWorkflow(ns string) swfclient.ScheduledWorkflowInterface {
	return c.client.ScheduledworkflowV1beta1().ScheduledWorkflows(ns)
}
func transferTestManager(t *testing.T) (*ResourceManager, *swffake.Clientset) {
	t.Helper()
	f := NewFakeClientManagerOrFatalV2()
	t.Cleanup(func() { f.Close() })
	r := NewResourceManager(f, &ResourceManagerOptions{})
	cs := swffake.NewSimpleClientset()
	cs.PrependReactor("create", "scheduledworkflows", func(a ktesting.Action) (bool, runtime.Object, error) {
		obj := a.(ktesting.CreateAction).GetObject().(*scheduledworkflow.ScheduledWorkflow)
		obj.UID = types.UID(uuid.NewString())
		return false, nil, nil
	})
	cs.PrependReactor("patch", "scheduledworkflows", func(a ktesting.Action) (bool, runtime.Object, error) {
		patch := a.(ktesting.PatchAction)
		obj, err := cs.Tracker().Get(a.GetResource(), a.GetNamespace(), patch.GetName())
		if err != nil {
			return true, nil, err
		}
		swf := obj.(*scheduledworkflow.ScheduledWorkflow).DeepCopy()
		var payload struct {
			Spec struct {
				Enabled bool `json:"enabled"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(patch.GetPatch(), &payload); err != nil {
			return true, nil, err
		}
		swf.Spec.Enabled = payload.Spec.Enabled
		return true, swf, cs.Tracker().Update(a.GetResource(), swf, a.GetNamespace())
	})
	r.swfClient = transferFakeSWF{cs}
	return r, cs
}
func transferTestArchive(t *testing.T) []byte {
	t.Helper()
	source, _ := transferTestManager(t)
	db, err := source.transferDB()
	require.NoError(t, err)
	interval := int64(3600)
	values := []any{
		&model.Experiment{UUID: "source-exp", Name: "source", Namespace: "team"},
		&model.Experiment{UUID: "empty-exp", Name: "empty", Namespace: "team"},
		&model.Pipeline{UUID: "source-pipeline", Name: "training", Namespace: "team", Status: model.PipelineReady},
		&model.PipelineVersion{UUID: "source-version", Name: "v1", PipelineId: "source-pipeline", PipelineSpec: model.LargeText(v2SpecHelloWorld), Status: model.PipelineVersionReady, CreatedAtInSec: 20},
		&model.PipelineVersion{UUID: "unused-version", Name: "unused", PipelineId: "source-pipeline", PipelineSpec: model.LargeText(v2SpecHelloWorld), Status: model.PipelineVersionReady, CreatedAtInSec: 10},
		&model.Job{UUID: "source-job", DisplayName: "nightly", Namespace: "team", ExperimentId: "source-exp", Enabled: true, MaxConcurrency: 1, Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{IntervalSecond: &interval}}, PipelineSpec: model.PipelineSpec{PipelineId: "source-pipeline", PipelineVersionId: "source-version", RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"world"}`, PipelineRoot: "s3://existing/root"}}},
	}
	for _, v := range values {
		require.NoError(t, db.Omit(clause.Associations).Create(v).Error)
	}
	archive, err := source.ExportTransfer(context.Background(), "team", transfer.ExportOptions{})
	require.NoError(t, err)
	return archive
}
func TestTransferScheduleLifecycleAndCatalog(t *testing.T) {
	pod := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "team")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", pod) })
	old := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", false)
	t.Cleanup(func() { viper.Set("MULTIUSER", old) })
	for _, kubernetes := range []bool{false, true} {
		t.Run(fmt.Sprint("kubernetes=", kubernetes), func(t *testing.T) {
			ctx := context.Background()
			archive := transferTestArchive(t)
			r, cs := transferTestManager(t)
			var catalog ctrlclient.Client
			if kubernetes {
				scheme := runtime.NewScheme()
				require.NoError(t, kubernetesmodel.AddToScheme(scheme))
				catalog = ctrlfake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{Create: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
					obj.SetUID(types.UID(uuid.NewString()))
					obj.SetCreationTimestamp(metav1.Now())
					return c.Create(ctx, obj, opts...)
				}}).Build()
				r.pipelineStore = storage.NewPipelineStoreKubernetes(catalog, catalog)
			}
			summary, err := r.ImportTransfer(ctx, "team", archive, transfer.ImportOptions{DryRun: true})
			require.NoError(t, err)
			require.Equal(t, 2, summary.Counts.Experiments)
			require.Equal(t, 2, summary.Counts.PipelineVersions)
			for _, a := range cs.Actions() {
				require.NotEqual(t, "create", a.GetVerb())
			}
			if catalog != nil {
				var ps kubernetesmodel.PipelineList
				require.NoError(t, catalog.List(ctx, &ps))
				require.Empty(t, ps.Items)
			}
			_, err = r.ImportTransfer(ctx, "team", archive, transfer.ImportOptions{})
			require.NoError(t, err)
			db, err := r.transferDB()
			require.NoError(t, err)
			var jobs []model.Job
			require.NoError(t, db.Find(&jobs).Error)
			require.Len(t, jobs, 1)
			job := jobs[0]
			swf, err := cs.ScheduledworkflowV1beta1().ScheduledWorkflows("team").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, job.UUID, string(swf.UID))
			require.False(t, swf.Spec.Enabled)
			require.True(t, *swf.Spec.NoCatchup)
			require.NoError(t, r.ChangeJobMode(ctx, job.UUID, true))
			swf, err = cs.ScheduledworkflowV1beta1().ScheduledWorkflows("team").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.True(t, swf.Spec.Enabled)
			repeat, err := r.ImportTransfer(ctx, "team", archive, transfer.ImportOptions{})
			require.NoError(t, err)
			require.Zero(t, repeat.Imported)
			require.Equal(t, summary.Imported, repeat.Skipped)
			swf, err = cs.ScheduledworkflowV1beta1().ScheduledWorkflows("team").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.True(t, swf.Spec.Enabled, "repeat import must preserve a user-enabled schedule")
			if kubernetes {
				v, err := r.pipelineStore.GetDefaultPipelineVersion(job.PipelineId)
				require.NoError(t, err)
				require.Equal(t, job.PipelineVersionId, v.UUID)
			}
			require.NoError(t, r.DeleteJob(ctx, job.UUID))
		})
	}
}
func TestTransferRejectsTamperedOwnershipBeforeStaging(t *testing.T) {
	pod := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "team")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", pod) })
	old := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", false)
	t.Cleanup(func() { viper.Set("MULTIUSER", old) })
	archive := transferTestArchive(t)
	var b history.NamespaceBundle
	require.NoError(t, json.Unmarshal(archive, &b))
	b.RuntimeNamespace = "other"
	b.Schedules[0].Namespace = "other"
	raw, err := json.Marshal(b)
	require.NoError(t, err)
	r, cs := transferTestManager(t)
	_, err = r.ImportTransfer(context.Background(), "team", raw, transfer.ImportOptions{})
	require.ErrorContains(t, err, "runtime namespace")
	for _, a := range cs.Actions() {
		require.NotEqual(t, "create", a.GetVerb())
	}
}

func TestTransferDeletedCatalogAndCrossNamespaceReferences(t *testing.T) {
	old := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", false)
	t.Cleanup(func() { viper.Set("MULTIUSER", old) })
	r, _ := transferTestManager(t)
	db, err := r.transferDB()
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "exp", Namespace: "team"}).Error)
	require.NoError(t, db.Create(&model.Run{UUID: "history", Namespace: "team", ExperimentId: "exp", PipelineSpec: model.PipelineSpec{PipelineId: "missing", PipelineSpecManifest: model.LargeText(v2SpecHelloWorld)}, RunDetails: model.RunDetails{FinishedAtInSec: 10, State: model.RuntimeStateSucceeded}}).Error)
	data, err := r.ExportTransfer(context.Background(), "team", transfer.ExportOptions{})
	require.NoError(t, err)
	var b history.NamespaceBundle
	require.NoError(t, json.Unmarshal(data, &b))
	require.Len(t, b.Entries, 1)
	require.Empty(t, b.Entries[0].Run.PipelineId)
	require.NotEmpty(t, b.Entries[0].Run.PipelineSpecManifest)
	require.NoError(t, db.Create(&model.Pipeline{UUID: "missing", Name: "private", Namespace: "other", Status: model.PipelineReady}).Error)
	_, err = r.ExportTransfer(context.Background(), "team", transfer.ExportOptions{})
	require.ErrorContains(t, err, "outside the selected catalog")
}

func TestTransferInlineOnlyScheduleRetainsExecutableWorkflow(t *testing.T) {
	old := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", false)
	t.Cleanup(func() { viper.Set("MULTIUSER", old) })
	data := transferTestArchive(t)
	var b history.NamespaceBundle
	require.NoError(t, json.Unmarshal(data, &b))
	b.Schedules[0].PipelineId = ""
	b.Schedules[0].PipelineVersionId = ""
	b.Schedules[0].PipelineSpecManifest = model.LargeText(v2SpecHelloWorld)
	data, err := json.Marshal(b)
	require.NoError(t, err)
	r, cs := transferTestManager(t)
	_, err = r.ImportTransfer(context.Background(), "team", data, transfer.ImportOptions{})
	require.NoError(t, err)
	db, err := r.transferDB()
	require.NoError(t, err)
	var job model.Job
	require.NoError(t, db.Take(&job).Error)
	swf, err := cs.ScheduledworkflowV1beta1().ScheduledWorkflows("team").Get(context.Background(), job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, swf.Spec.Workflow.Spec)
	require.False(t, swf.Spec.Enabled)
	require.NoError(t, r.ChangeJobMode(context.Background(), job.UUID, true))
}

func TestTransferCatalogBudgetStopsKubernetesPagination(t *testing.T) {
	ctx := context.Background()
	p := model.Pipeline{UUID: "pipeline", Name: "training", Namespace: "team", Status: model.PipelineReady}
	kp := kubernetesmodel.FromPipelineModel(p)
	kp.Spec.DefaultVersionName = "a"
	kp.CreationTimestamp = metav1.NewTime(metav1.Now().Truncate(1e9))
	var pages []kubernetesmodel.PipelineVersion
	for _, name := range []string{"a", "b", "c"} {
		row := model.PipelineVersion{UUID: name, Name: name, PipelineId: p.UUID, PipelineSpec: model.LargeText(v2SpecHelloWorld), Status: model.PipelineVersionReady}
		v, err := kubernetesmodel.FromPipelineVersionModel(p, row)
		require.NoError(t, err)
		v.CreationTimestamp = kp.CreationTimestamp
		pages = append(pages, *v)
	}
	// The pin must win over newer versions; a broken mutable label must not hide an owned version.
	pages[2].CreationTimestamp = metav1.NewTime(kp.CreationTimestamp.Add(1e9))
	pages[0].Labels = map[string]string{"pipelines.kubeflow.org/pipeline-id": "stale"}
	scheme := runtime.NewScheme()
	require.NoError(t, kubernetesmodel.AddToScheme(scheme))
	reads := 0
	catalog := ctrlfake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{List: func(_ context.Context, _ ctrlclient.WithWatch, obj ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
		options := (&ctrlclient.ListOptions{}).ApplyOptions(opts)
		require.Equal(t, "team", options.Namespace)
		require.Equal(t, int64(1), options.Limit)
		switch page := obj.(type) {
		case *kubernetesmodel.PipelineList:
			page.Items = []kubernetesmodel.Pipeline{kp}
		case *kubernetesmodel.PipelineVersionList:
			i := 0
			if options.Continue != "" {
				var err error
				i, err = strconv.Atoi(options.Continue)
				require.NoError(t, err)
			}
			reads++
			page.Items = []kubernetesmodel.PipelineVersion{pages[i]}
			if i+1 < len(pages) {
				page.Continue = strconv.Itoa(i + 1)
			}
		default:
			t.Fatalf("unexpected catalog list %T", obj)
		}
		return nil
	}}).Build()
	store := storage.NewPipelineStoreKubernetes(catalog, catalog)
	r := &ResourceManager{pipelineStore: store}
	pipelines, versions, err := r.exportTransferCatalog(ctx, "team", transfer.NewExportBudget(transfer.MaxArchiveBytes))
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	require.Len(t, versions, 3)
	require.Equal(t, "a", pipelines[0].DefaultVersionId)
	require.Equal(t, 3, reads)
	pipelines[0].DefaultVersionId = ""
	pipelineBytes, err := json.Marshal(pipelines[0])
	require.NoError(t, err)
	versionBytes, err := json.Marshal(versions[0])
	require.NoError(t, err)
	reads = 0
	_, _, err = r.exportTransferCatalog(ctx, "team", transfer.NewExportBudget(len(pipelineBytes)+len(versionBytes)+2))
	require.ErrorContains(t, err, "transfer byte limit")
	require.Equal(t, 2, reads, "stop before fetching the third spec")
	kp.Spec.DefaultVersionName = ""
	pipelines, _, err = r.exportTransferCatalog(ctx, "team", transfer.NewExportBudget(transfer.MaxArchiveBytes))
	require.NoError(t, err)
	require.Equal(t, "c", pipelines[0].DefaultVersionId, "an unpinned pipeline uses its newest owned version")
}
