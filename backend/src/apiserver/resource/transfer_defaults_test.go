// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	kubernetesmodel "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type transferDefaultStore struct {
	storage.PipelineStoreInterface
	version  *model.PipelineVersion
	err      error
	calls    int
	unpinned bool
}

func (s *transferDefaultStore) TransferDefaultPinned(string) (bool, error) { return !s.unpinned, nil }

func (s *transferDefaultStore) GetDefaultPipelineVersion(string) (*model.PipelineVersion, error) {
	s.calls++
	return s.version, s.err
}

func transferDefaultPlan(external, existing bool) *history.TransferPlan {
	return &history.TransferPlan{
		ExternalCatalog: external,
		Existing:        map[string]bool{"receipt": existing},
		Receipts:        []model.TransferReceipt{{Key: "receipt", Kind: "pipeline", TargetID: "pipeline"}},
		Bundle: &history.NamespaceBundle{Bundle: history.Bundle{
			Pipelines: []model.Pipeline{{UUID: "pipeline", DefaultVersionId: "source-old"}},
			Versions: []model.PipelineVersion{
				{UUID: "source-old", PipelineId: "pipeline", CreatedAtInSec: 10, PipelineSpec: model.LargeText(v2SpecHelloWorld)},
				{UUID: "source-new", PipelineId: "pipeline", CreatedAtInSec: 20, PipelineSpec: model.LargeText(v2SpecHelloWorld)},
			},
		}},
	}
}

func TestTransferScheduleVersionDefaults(t *testing.T) {
	job := &model.Job{PipelineSpec: model.PipelineSpec{PipelineId: "pipeline"}}
	for _, tc := range []struct {
		name               string
		external, existing bool
		destination        *model.PipelineVersion
		expected           string
	}{
		{name: "new Kubernetes pipeline uses archived pin", external: true, expected: "source-old"},
		{name: "existing Kubernetes pipeline retains destination pin outside archive", external: true, existing: true, destination: &model.PipelineVersion{UUID: "destination", PipelineId: "pipeline", CreatedAtInSec: 1}, expected: "destination"},
		{name: "new SQL pipeline ignores deprecated pin", expected: "source-new"},
		{name: "existing SQL pipeline retains newer destination version", existing: true, destination: &model.PipelineVersion{UUID: "destination", PipelineId: "pipeline", CreatedAtInSec: 30}, expected: "destination"},
		{name: "SQL selects newly imported newer version", existing: true, destination: &model.PipelineVersion{UUID: "destination", PipelineId: "pipeline", CreatedAtInSec: 1}, expected: "source-new"},
		{name: "SQL same-second tie uses UUID", existing: true, destination: &model.PipelineVersion{UUID: "z-destination", PipelineId: "pipeline", CreatedAtInSec: 20}, expected: "z-destination"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &transferDefaultStore{version: tc.destination}
			r := &ResourceManager{pipelineStore: store}
			selected, err := r.transferScheduleVersion(transferDefaultPlan(tc.external, tc.existing), job)
			require.NoError(t, err)
			require.Equal(t, tc.expected, selected.UUID)
			require.Empty(t, job.PipelineVersionId, "validation must preserve unpinned execution")
		})
	}
}

func TestTransferScheduleRetainedDefaultErrorsAndOwnership(t *testing.T) {
	job := &model.Job{PipelineSpec: model.PipelineSpec{PipelineId: "pipeline"}}
	missing := util.NewResourceNotFoundError("PipelineVersion", "default")
	for _, tc := range []struct {
		name    string
		version *model.PipelineVersion
		err     error
	}{
		{name: "missing default", err: missing},
		{name: "failed lookup", err: errors.New("catalog unavailable")},
		{name: "foreign owner", version: &model.PipelineVersion{UUID: "foreign", PipelineId: "other"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &ResourceManager{pipelineStore: &transferDefaultStore{version: tc.version, err: tc.err}}
			selected, err := r.transferScheduleVersion(transferDefaultPlan(true, true), job)
			require.Error(t, err)
			require.Nil(t, selected, "never fall back to archive when destination default cannot be used")
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.ErrorContains(t, err, "different pipeline")
			}
		})
	}
}

func TestTransferScheduleExplicitPinAndInlineRemainUnchanged(t *testing.T) {
	store := &transferDefaultStore{err: errors.New("must not query destination default")}
	r := &ResourceManager{pipelineStore: store}
	plan := transferDefaultPlan(true, true)
	job := &model.Job{PipelineSpec: model.PipelineSpec{PipelineId: "pipeline", PipelineVersionId: "source-old"}}
	selected, err := r.transferScheduleVersion(plan, job)
	require.NoError(t, err)
	require.Equal(t, "source-old", selected.UUID)
	require.Zero(t, store.calls)
	job.PipelineId = "other"
	_, err = r.transferScheduleVersion(plan, job)
	require.ErrorContains(t, err, "archived pipeline")
	selected, err = r.transferScheduleVersion(plan, &model.Job{PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld)}})
	require.NoError(t, err)
	require.Nil(t, selected)
}

func TestTransferScheduleValidatesRetainedDestinationInputs(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	r, client := transferTestManager(t)
	// The archived definition accepts a string; the retained destination
	// version requires a number. Validate the version the controller will use.
	destinationSpec := strings.ReplaceAll(v2SpecHelloWorld, "parameterType: STRING", "parameterType: NUMBER_INTEGER")
	destinationSpec = strings.ReplaceAll(destinationSpec, "defaultValue: world", "defaultValue: 1")
	r.pipelineStore = &transferDefaultStore{PipelineStoreInterface: r.pipelineStore, version: &model.PipelineVersion{UUID: "destination", PipelineId: "pipeline", PipelineSpec: model.LargeText(destinationSpec)}}
	plan := transferDefaultPlan(true, true)
	job := &model.Job{Namespace: "team", K8SName: "test-schedule", PipelineSpec: model.PipelineSpec{PipelineId: "pipeline", RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"world"}`}}}
	_, err := r.transferScheduleWorkflow(context.Background(), plan, job)
	require.Error(t, err, "string input must fail validation against the retained number input")
	require.Empty(t, client.Actions(), "validation must not create a schedule")
	job.RuntimeConfig.Parameters = `{"text":1}`
	workflow, err := r.transferScheduleWorkflow(context.Background(), plan, job)
	require.NoError(t, err)
	require.False(t, workflow.Spec.Enabled)
	require.Empty(t, workflow.Spec.PipelineVersionId, "validation must not pin the schedule")
}

func TestTransferImportUsesRetainedKubernetesDefaultBeforeWrites(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	old, oldNamespace := viper.Get("MULTIUSER"), viper.Get("POD_NAMESPACE")
	viper.Set("MULTIUSER", false)
	viper.Set("POD_NAMESPACE", "team")
	t.Cleanup(func() { viper.Set("MULTIUSER", old); viper.Set("POD_NAMESPACE", oldNamespace) })
	ctx := context.Background()
	var archive history.NamespaceBundle
	require.NoError(t, json.Unmarshal(transferTestArchive(t), &archive))
	schedules := archive.Schedules
	scheduleParameters := archive.RuntimeParameters.Schedules
	archive.RuntimeParameters.Schedules = map[string]string{}
	archive.Schedules = nil
	catalogOnly, err := json.Marshal(archive)
	require.NoError(t, err)
	r, workflows := transferTestManager(t)
	scheme := runtime.NewScheme()
	require.NoError(t, kubernetesmodel.AddToScheme(scheme))
	creates := 0
	catalog := ctrlfake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{Create: func(ctx context.Context, client ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
		creates++
		obj.SetUID(types.UID(uuid.NewString()))
		obj.SetCreationTimestamp(metav1.Now())
		return client.Create(ctx, obj, opts...)
	}}).Build()
	store := storage.NewPipelineStoreKubernetes(catalog, catalog)
	r.pipelineStore = store
	_, err = r.ImportTransfer(ctx, "team", catalogOnly, transfer.ImportOptions{})
	require.NoError(t, err)
	pipeline, err := store.GetPipelineByNameAndNamespace("training", "team")
	require.NoError(t, err)
	destinationSpec := strings.ReplaceAll(v2SpecHelloWorld, "parameterType: STRING", "parameterType: NUMBER_INTEGER")
	destinationSpec = strings.ReplaceAll(destinationSpec, "        defaultValue: world\n", "")
	destination, err := store.CreatePipelineVersion(&model.PipelineVersion{Name: "destination-number", PipelineId: pipeline.UUID, PipelineSpec: model.LargeText(destinationSpec), Status: model.PipelineVersionReady})
	require.NoError(t, err)
	require.NoError(t, store.SetTransferDefaultVersion(pipeline.UUID, destination.UUID))
	archive.Schedules = schedules
	archive.RuntimeParameters.Schedules = scheduleParameters
	archive.Schedules[0].PipelineVersionId = ""
	input, err := json.Marshal(archive)
	require.NoError(t, err)
	before := creates
	for _, dry := range []bool{true, false} {
		_, err = r.ImportTransfer(ctx, "team", input, transfer.ImportOptions{DryRun: dry})
		require.Error(t, err, "source defaults cannot satisfy the retained destination required number input")
		require.Equal(t, before, creates, "reject before catalog writes")
		require.Empty(t, workflows.Actions(), "reject before workflow access or writes")
	}
	db, err := r.transferDB()
	require.NoError(t, err)
	var jobCount int64
	require.NoError(t, db.Model(&model.Job{}).Count(&jobCount).Error)
	require.Zero(t, jobCount)

	// A cleared pin plus new versions cannot be predicted from source dates:
	// the Kubernetes API assigns their destination timestamps and UIDs.
	var current kubernetesmodel.Pipeline
	require.NoError(t, catalog.Get(ctx, types.NamespacedName{Namespace: "team", Name: "training"}, &current))
	current.Spec.DefaultVersionName = ""
	require.NoError(t, catalog.Update(ctx, &current))
	archive.Versions = append(archive.Versions, model.PipelineVersion{UUID: "new-source-version", Name: "new-version", PipelineId: "source-pipeline", PipelineSpec: model.LargeText(v2SpecHelloWorld), Status: model.PipelineVersionReady, CreatedAtInSec: 30})
	input, err = json.Marshal(archive)
	require.NoError(t, err)
	for _, dry := range []bool{true, false} {
		_, err = r.ImportTransfer(ctx, "team", input, transfer.ImportOptions{DryRun: dry})
		require.ErrorContains(t, err, "pin the destination pipeline default and retry")
		require.Equal(t, before, creates, "reject before catalog writes")
		require.Empty(t, workflows.Actions())
	}
	var receiptCount int64
	require.NoError(t, db.Model(&model.TransferReceipt{}).Where(clause.Eq{Column: "SourceID", Value: "new-source-version"}).Count(&receiptCount).Error)
	require.Zero(t, receiptCount)
}
