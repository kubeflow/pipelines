// Copyright 2018 The Kubeflow Authors
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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/file"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	apiservermlflow "github.com/kubeflow/pipelines/backend/src/apiserver/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// v1AllowedNamespaces mirrors the unexported constant in backend/src/common/util/v1_support.go.
const v1AllowedNamespaces = "V1_ALLOWED_NAMESPACES"

type duplicateRecurringRunStore struct {
	storage.RunStoreInterface
	firstGet    bool
	existingRun *model.Run
}

func (s *duplicateRecurringRunStore) GetRun(string) (*model.Run, error) {
	if s.firstGet {
		s.firstGet = false
		return nil, util.NewResourceNotFoundError("run", "concurrent-run")
	}
	return s.existingRun, nil
}

func (s *duplicateRecurringRunStore) CreateRun(*model.Run) (*model.Run, error) {
	return s.existingRun, nil
}

func initEnvVars() {
	viper.Set(common.PodNamespace, "ns1")
	proxy.InitializeConfigWithEmptyForTests()
}

// setupTestSAToken writes a temp kubeconfig with the given bearer token and
// sets the KUBECONFIG env var so util.GetKubernetesConfig() picks it up.
func setupTestSAToken(t *testing.T, token string) {
	t.Helper()
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: %s
`, token)
	p := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(p, []byte(kubeconfig), 0600))
	t.Setenv("KUBECONFIG", p)
}

// setupMLflowViperConfig sets plugins.mlflow in Viper and restores the original
// value when the test completes.
func setupMLflowViperConfig(t *testing.T, endpoint string) {
	t.Helper()
	origConfig := viper.Get("plugins.mlflow")
	hadConfig := viper.IsSet("plugins.mlflow")
	viper.Set("plugins.mlflow", map[string]interface{}{
		"endpoint": endpoint,
		"timeout":  "10s",
	})
	t.Cleanup(func() {
		if hadConfig {
			viper.Set("plugins.mlflow", origConfig)
		} else {
			viper.Set("plugins.mlflow", nil)
		}
	})
}

type FakeBadObjectStore struct{}

func (m *FakeBadObjectStore) GetPipelineKey(pipelineID string) string {
	return pipelineID
}

func (m *FakeBadObjectStore) AddFile(ctx context.Context, template []byte, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) DeleteFile(ctx context.Context, filePath string) error {
	return errors.New("Not implemented")
}

func (m *FakeBadObjectStore) AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetFileReader(context.Context, string) (io.ReadCloser, error) {
	return nil, util.NewInternalServerError(errors.New("Error"), "bad object store")
}

type readerOnlyObjectStore struct {
	files              map[string][]byte
	getFileReaderPaths []string
}

func (m *readerOnlyObjectStore) GetPipelineKey(pipelineID string) string {
	return "pipelines/" + pipelineID
}

func (m *readerOnlyObjectStore) AddFile(ctx context.Context, template []byte, filePath string) error {
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[filePath] = template
	return nil
}

func (m *readerOnlyObjectStore) DeleteFile(ctx context.Context, filePath string) error {
	delete(m.files, filePath)
	return nil
}

func (m *readerOnlyObjectStore) AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return nil
}

func (m *readerOnlyObjectStore) GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return nil
}

func (m *readerOnlyObjectStore) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	m.getFileReaderPaths = append(m.getFileReaderPaths, filePath)
	content, ok := m.files[filePath]
	if !ok {
		return nil, util.NewInternalServerError(errors.New("not found"), "file not found")
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

func createPipelineV1(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:   name,
		Status: model.PipelineReady,
	}
}

func createPipeline(name string, description string, namespace string) *model.Pipeline {
	return &model.Pipeline{
		Name:        name,
		Description: model.LargeText(description),
		Status:      model.PipelineReady,
		Namespace:   namespace,
	}
}

func createPipelineVersion(pipelineId string, name string, description string, url string, pipelineSpec string, pipelineSpecURI string, namespace string) *model.PipelineVersion {
	if namespace == "" {
		namespace = "default"
	}
	paramsJSON := "[{\"name\":\"param1\"}]"
	spec := pipelineSpec
	tmpl, err := template.New([]byte(pipelineSpec), template.TemplateOptions{})
	if err != nil {
		spec = pipelineSpec
	} else {
		paramsJSON, _ = tmpl.ParametersJSON()
		spec = string(tmpl.Bytes())
	}
	return &model.PipelineVersion{
		Name:            name,
		Parameters:      model.LargeText(paramsJSON),
		PipelineId:      pipelineId,
		CodeSourceUrl:   url,
		Description:     model.LargeText(description),
		Status:          model.PipelineVersionReady,
		PipelineSpec:    model.LargeText(spec),
		PipelineSpecURI: model.LargeText(pipelineSpecURI),
	}
}

var testWorkflow = util.NewWorkflow(&v1alpha1.Workflow{
	TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
	Spec: v1alpha1.WorkflowSpec{
		Entrypoint: "testy",
		Templates: []v1alpha1.Template{{
			Name: "testy",
			Container: &corev1.Container{
				Image:   "docker/whalesay",
				Command: []string{"cowsay"},
				Args:    []string{"hello world"},
			},
		}},
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}},
	},
	Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
})

type retryDuringTerminalReportDispatcher struct {
	manager  *ResourceManager
	runID    string
	retryErr error
}

func (d *retryDuringTerminalReportDispatcher) OnBeforeRunCreation(context.Context, *apiserverPlugins.PendingRun, util.ExecutionSpec) error {
	return nil
}

func (d *retryDuringTerminalReportDispatcher) OnRunEnd(ctx context.Context, _ *apiserverPlugins.PersistedRun) bool {
	d.retryErr = d.manager.RetryRun(ctx, d.runID)
	return true
}

func (d *retryDuringTerminalReportDispatcher) OnRunRetry(context.Context, *apiserverPlugins.PersistedRun) error {
	return nil
}

func (d *retryDuringTerminalReportDispatcher) PluginsRegistered() bool {
	return true
}

type countingTerminalReportDispatcher struct {
	onRunEndCalls int
}

func (d *countingTerminalReportDispatcher) OnBeforeRunCreation(context.Context, *apiserverPlugins.PendingRun, util.ExecutionSpec) error {
	return nil
}

func (d *countingTerminalReportDispatcher) OnRunEnd(context.Context, *apiserverPlugins.PersistedRun) bool {
	d.onRunEndCalls++
	return true
}

func (d *countingTerminalReportDispatcher) OnRunRetry(context.Context, *apiserverPlugins.PersistedRun) error {
	return nil
}

func (d *countingTerminalReportDispatcher) PluginsRegistered() bool {
	return true
}

func TestReadRunLogFromArchiveStreamsObjectStoreFile(t *testing.T) {
	logArchive := archive.NewLogArchive("/logs", "main.log")
	execSpec, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(testWorkflow.ToStringForStore()))
	require.NoError(t, err)
	logPath, err := logArchive.GetLogObjectKey(execSpec, "node-id")
	require.NoError(t, err)

	objectStore := &readerOnlyObjectStore{
		files: map[string][]byte{
			logPath: []byte("archived log line\n"),
		},
	}
	manager := &ResourceManager{
		objectStore: objectStore,
		logArchive:  logArchive,
	}

	var dst bytes.Buffer
	err = manager.readRunLogFromArchive(context.Background(), testWorkflow.ToStringForStore(), "node-id", &dst)

	require.NoError(t, err)
	assert.Equal(t, []string{logPath}, objectStore.getFileReaderPaths)
	assert.Equal(t, "archived log line\n", dst.String())
}

// cancelAwareObjectStore wraps readerOnlyObjectStore but actually checks the
// context it's given, the way a real network-backed object store would.
type cancelAwareObjectStore struct {
	*readerOnlyObjectStore
}

func (m *cancelAwareObjectStore) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.readerOnlyObjectStore.GetFileReader(ctx, filePath)
}

func TestReadRunLogFromArchivePropagatesCanceledContext(t *testing.T) {
	logArchive := archive.NewLogArchive("/logs", "main.log")
	execSpec, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(testWorkflow.ToStringForStore()))
	require.NoError(t, err)
	logPath, err := logArchive.GetLogObjectKey(execSpec, "node-id")
	require.NoError(t, err)

	objectStore := &cancelAwareObjectStore{
		readerOnlyObjectStore: &readerOnlyObjectStore{
			files: map[string][]byte{
				logPath: []byte("archived log line\n"),
			},
		},
	}
	manager := &ResourceManager{
		objectStore: objectStore,
		logArchive:  logArchive,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var dst bytes.Buffer
	err = manager.readRunLogFromArchive(ctx, testWorkflow.ToStringForStore(), "node-id", &dst)

	require.Error(t, err)
	assert.Contains(t, err.Error(), context.Canceled.Error())
}

func TestReadPipelineSpecFromObjectStoreUsesReaderAndLimit(t *testing.T) {
	objectStore := &readerOnlyObjectStore{
		files: map[string][]byte{
			"pipeline-spec.yaml": []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow\n"),
		},
	}
	manager := &ResourceManager{objectStore: objectStore}

	pipelineSpec, err := manager.readPipelineSpecFromObjectStore(context.Background(), "pipeline-spec.yaml")

	require.NoError(t, err)
	assert.Equal(t, []string{"pipeline-spec.yaml"}, objectStore.getFileReaderPaths)
	assert.Equal(t, "apiVersion: argoproj.io/v1alpha1\nkind: Workflow\n", string(pipelineSpec))
}

func TestReadPipelineSpecFromObjectStoreRejectsOversizedFile(t *testing.T) {
	objectStore := &readerOnlyObjectStore{
		files: map[string][]byte{
			"pipeline-spec.yaml": bytes.Repeat([]byte("x"), common.MaxFileLength+1),
		},
	}
	manager := &ResourceManager{objectStore: objectStore}

	_, err := manager.readPipelineSpecFromObjectStore(context.Background(), "pipeline-spec.yaml")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Pipeline spec file size too large")
	assert.Equal(t, []string{"pipeline-spec.yaml"}, objectStore.getFileReaderPaths)
}

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Pipeline, *model.PipelineVersion) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	p1 := createPipeline("p1", "", "ns1")
	p, _ := manager.CreatePipeline(p1)
	pv1 := createPipelineVersion(
		p.UUID,
		"p1/v1",
		"v1",
		"url://namespaces/ns1/pipelines/p1/versions/v1",
		testWorkflow.ToStringForStore(),
		"uri://namespaces/ns1/pipelines/p1/versions/v1/p1v1.yaml",
		"ns1",
	)
	pv, err := manager.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	return store, manager, p, pv
}

func initWithExperiment(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1", Namespace: "ns1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return store, manager, experiment
}

func initWithExperimentAndPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.PipelineVersion) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	p1 := createPipeline("p1", "", "ns1")
	p, _ := manager.CreatePipeline(p1)
	pv1 := createPipelineVersion(
		p.UUID,
		"p1/v1",
		"v1",
		"url://namespaces/ns1/pipelines/p1/versions/v1",
		testWorkflow.ToStringForStore(),
		"uri://namespaces/ns1/pipelines/p1/versions/v1/p1v1.yaml",
		"ns1",
	)
	pv, err := manager.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	return store, manager, experiment, p, pv
}

func initWithExperimentAndPipelineAndRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.PipelineVersion, *model.Run) {
	store, manager, exp, pipeline, version := initWithExperimentAndPipeline(t)
	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: exp.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	run, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, exp, pipeline, version, run
}

// Util function to create an initial state with pipeline uploaded
func initWithJob(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId: exp.UUID,
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

// Util function to create an initial state with pipeline uploaded
func initWithJobV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		ExperimentId: exp.UUID,
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

func initWithOneTimeRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func syncWorkflowReportWithFakeCluster(t *testing.T, store *FakeClientManager, workflow *util.Workflow) {
	t.Helper()
	ctx := context.Background()
	workflowClient := store.ExecClient().Execution(workflow.ExecutionNamespace())
	liveWorkflow, err := workflowClient.Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	if util.IsNotFound(err) {
		_, err = workflowClient.Create(ctx, workflow, v1.CreateOptions{})
		require.NoError(t, err)
		return
	}
	require.NoError(t, err)
	workflow.ExecutionObjectMeta().UID = liveWorkflow.ExecutionObjectMeta().UID
	workflow.SetVersion(liveWorkflow.Version())
	_, err = workflowClient.Update(ctx, workflow, v1.UpdateOptions{})
	require.NoError(t, err)
}

func storedWorkflowUID(t *testing.T, run *model.Run) types.UID {
	t.Helper()
	manifest := run.WorkflowRuntimeManifest
	if manifest == "" {
		manifest = run.PipelineRuntimeManifest
	}
	workflow, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(manifest))
	require.NoError(t, err)
	require.NoError(t, workflow.Decompress())
	return workflow.ExecutionObjectMeta().UID
}

func initWithOneTimeRunV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithPatchedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),

			Parameters: "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
		},
		ExperimentId: exp.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.Nodes = map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	syncWorkflowReportWithFakeCluster(t, store, updatedWorkflow)
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunCompressed(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	nodes := map[string]v1alpha1.NodeStatus{"node1": {Name: "pod1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed}}
	nodeData, err := json.Marshal(nodes)
	assert.Nil(t, err)
	updatedWorkflow.Status.CompressedNodes = file.CompressEncodeString(ctx, string(nodeData))
	syncWorkflowReportWithFakeCluster(t, store, updatedWorkflow)
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

func initWithOneTimeFailedRunOffloaded(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Run) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	ctx := context.Background()
	runDetail, err := manager.CreateRun(ctx, apiRun)
	assert.Nil(t, err)
	updatedWorkflow := util.NewWorkflow(testWorkflow.DeepCopy())
	updatedWorkflow.SetLabels(util.LabelKeyWorkflowRunId, runDetail.UUID)
	updatedWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	updatedWorkflow.Status.OffloadNodeStatusVersion = "offload-hash"
	syncWorkflowReportWithFakeCluster(t, store, updatedWorkflow)
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	assert.Nil(t, err)
	return store, manager, runDetail
}

type retryWorkflowExecClient struct {
	workflowClient util.ExecutionInterface
}

func (c *retryWorkflowExecClient) Execution(namespace string) util.ExecutionInterface {
	return c.workflowClient
}

func (c *retryWorkflowExecClient) Compare(old, new interface{}) bool {
	return false
}

type updateConflictWorkflowClient struct {
	*client.FakeWorkflowClient
	updateConflictsRemaining int
	createCalls              int
}

func (c *updateConflictWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	if c.updateConflictsRemaining > 0 {
		c.updateConflictsRemaining--
		return nil, apierrors.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName(), errors.New("stale workflow"))
	}
	return c.FakeWorkflowClient.Update(ctx, execSpec, opts)
}

func (c *updateConflictWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	c.createCalls++
	return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName())
}

type persistentConflictWorkflowClient struct {
	*client.FakeWorkflowClient
	updateCalls int
	createCalls int
}

func (c *persistentConflictWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	c.updateCalls++
	return nil, apierrors.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName(), errors.New("stale workflow"))
}

func (c *persistentConflictWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	c.createCalls++
	return c.FakeWorkflowClient.Create(ctx, execSpec, opts)
}

type createAlreadyExistsWorkflowClient struct {
	*client.FakeWorkflowClient
	updateNotFoundRemaining      int
	createAlreadyExistsRemaining int
}

func (c *createAlreadyExistsWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	if c.updateNotFoundRemaining > 0 {
		c.updateNotFoundRemaining--
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName())
	}
	return c.FakeWorkflowClient.Update(ctx, execSpec, opts)
}

func (c *createAlreadyExistsWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	if c.createAlreadyExistsRemaining > 0 {
		c.createAlreadyExistsRemaining--
		return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName())
	}
	return c.FakeWorkflowClient.Create(ctx, execSpec, opts)
}

type retryableGetFailureWorkflowClient struct {
	*client.FakeWorkflowClient
	getFailuresRemaining int
	createCalls          int
}

func (c *retryableGetFailureWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	if c.getFailuresRemaining > 0 {
		c.getFailuresRemaining--
		return nil, apierrors.NewServiceUnavailable("apiserver temporarily unavailable")
	}
	return c.FakeWorkflowClient.Get(ctx, name, options)
}

func (c *retryableGetFailureWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	c.createCalls++
	return c.FakeWorkflowClient.Create(ctx, execSpec, opts)
}

type retryableUpdateFailureWorkflowClient struct {
	*client.FakeWorkflowClient
	updateFailuresRemaining int
	createCalls             int
}

func (c *retryableUpdateFailureWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	if c.updateFailuresRemaining > 0 {
		c.updateFailuresRemaining--
		return nil, apierrors.NewServiceUnavailable("apiserver temporarily unavailable")
	}
	return c.FakeWorkflowClient.Update(ctx, execSpec, opts)
}

func (c *retryableUpdateFailureWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	c.createCalls++
	return c.FakeWorkflowClient.Create(ctx, execSpec, opts)
}

type genericUpdateFailureWorkflowClient struct {
	*client.FakeWorkflowClient
	createCalls int
}

type deleteFailureWorkflowClient struct {
	*client.FakeWorkflowClient
}

func (c *deleteFailureWorkflowClient) Delete(context.Context, string, v1.DeleteOptions) error {
	return errors.New("failed to delete workflow")
}

type countingWorkflowClient struct {
	*client.FakeWorkflowClient
	getCalls int
}

func (c *countingWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	c.getCalls++
	return c.FakeWorkflowClient.Get(ctx, name, options)
}

type transientDeleteFailureWorkflowClient struct {
	*client.FakeWorkflowClient
	deleteFailuresRemaining int
	deleteCalls             int
}

func (c *transientDeleteFailureWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	c.deleteCalls++
	if c.deleteFailuresRemaining > 0 {
		c.deleteFailuresRemaining--
		return apierrors.NewServiceUnavailable("apiserver temporarily unavailable")
	}
	return c.FakeWorkflowClient.Delete(ctx, name, options)
}

type disappearBeforeDeleteLookupWorkflowClient struct {
	*client.FakeWorkflowClient
	getCalls int
}

func (c *disappearBeforeDeleteLookupWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	c.getCalls++
	if c.getCalls == 2 {
		if err := c.Delete(ctx, name, v1.DeleteOptions{}); err != nil {
			return nil, err
		}
	}
	return c.FakeWorkflowClient.Get(ctx, name, options)
}

type notFoundOnOrphanDeleteWorkflowClient struct {
	*client.FakeWorkflowClient
	getCalls    int
	deleteCalls int
}

func (c *notFoundOnOrphanDeleteWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	c.getCalls++
	return c.FakeWorkflowClient.Get(ctx, name, options)
}

func (c *notFoundOnOrphanDeleteWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	c.deleteCalls++
	if err := c.FakeWorkflowClient.Delete(ctx, name, options); err != nil {
		return err
	}
	return apierrors.NewNotFound(
		schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, name)
}

type updateBeforeFirstDeleteWorkflowClient struct {
	*client.FakeWorkflowClient
	deleteCalls int
}

func (c *updateBeforeFirstDeleteWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	c.deleteCalls++
	if c.deleteCalls == 1 {
		workflow, err := c.Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return err
		}
		if _, err := c.Update(ctx, workflow, v1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return c.FakeWorkflowClient.Delete(ctx, name, options)
}

type replaceBeforeOrphanDeleteLookupWorkflowClient struct {
	*client.FakeWorkflowClient
	getCalls int
}

func (c *replaceBeforeOrphanDeleteLookupWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	c.getCalls++
	if c.getCalls != 2 {
		return c.FakeWorkflowClient.Get(ctx, name, options)
	}

	current, err := c.FakeWorkflowClient.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	metadata := current.ExecutionObjectMeta()
	if err := c.Delete(ctx, name, v1.DeleteOptions{}); err != nil {
		return nil, err
	}
	replacement := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              name,
			Namespace:         current.ExecutionNamespace(),
			Labels:            metadata.Labels,
			CreationTimestamp: metadata.CreationTimestamp,
		},
	})
	return c.Create(ctx, replacement, v1.CreateOptions{})
}

type retryBeforeDeleteWorkflowClient struct {
	*client.FakeWorkflowClient
	manager        *ResourceManager
	runID          string
	retryAttempted bool
	retryErr       error
}

func (c *retryBeforeDeleteWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	if !c.retryAttempted {
		c.retryAttempted = true
		c.retryErr = c.manager.RetryRun(ctx, c.runID)
	}
	return c.FakeWorkflowClient.Delete(ctx, name, options)
}

func (c *genericUpdateFailureWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	return nil, errors.New("transient update failure")
}

func (c *genericUpdateFailureWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	c.createCalls++
	return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execSpec.ExecutionName())
}

func seedRetryWorkflow(t *testing.T, manager *ResourceManager, runID string, workflowClient util.ExecutionInterface) {
	t.Helper()
	run, err := manager.GetRun(runID)
	require.NoError(t, err)
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.NoError(t, execSpec.Decompress())
	retryExecSpec, _, err := execSpec.GenerateRetryExecution()
	require.NoError(t, err)
	_, err = workflowClient.Create(context.Background(), retryExecSpec, v1.CreateOptions{})
	require.NoError(t, err)
}

// Tests CreatePipeline and CreatePipelineVersion
func TestCreatePipeline(t *testing.T) {
	tt := []struct {
		msg            string
		name           string // optional
		description    string // optional
		template       string // pipeline template
		badObjectStore bool   // optional, object requests always fail
		badDB          bool   // optional, DB request always fail
		// The following are expected results.
		model        *model.Pipeline        // optional, expected pipeline model when success
		modelVersion *model.PipelineVersion // optional, expected pipeline model when success
		// To verify an error, set the errorCode and
		// optionally set errorMsg and errorIs based on the test's needs.
		errorCode codes.Code
		errorMsg  string // error message
		errorIs   error  // verify a wrapped error is specific instance
	}{
		{
			msg:         "HappyCase",
			template:    testWorkflow.ToStringForStore(),
			name:        "p_v",
			description: "test",
			model:       createPipeline("p_v", "test", "user1"),
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			name:     "complex",
			model:    createPipeline("complex", "", "user1"),
		},
		{
			msg:       "InvalidTemplate",
			template:  "I am invalid yaml",
			model:     createPipeline("InvalidYAML", "", "user1"),
			errorCode: codes.InvalidArgument,
			errorIs:   template.ErrorInvalidPipelineSpec,
		},
		{
			msg:       "BadDB",
			template:  testWorkflow.ToStringForStore(),
			badDB:     true,
			errorCode: codes.Internal,
			errorMsg:  "database is closed",
			model:     createPipeline("BadDB", "", "user1"),
		},
		{
			msg:      "V2PipelineSpec",
			template: v2SpecHelloWorld,
			name:     "v2spec",
			model:    createPipeline("v2 spec", "", "user1"),
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			var pipelineVersion, pv *model.PipelineVersion
			// setup
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			if test.badObjectStore {
				manager.objectStore = &FakeBadObjectStore{}
			}
			if test.badDB {
				store.DB().Close()
			}

			// start test
			if test.name == "" {
				test.name = "my_pipeline_name"
			}
			pipeline, err := manager.CreatePipeline(
				test.model,
			)
			if err == nil {
				pv = createPipelineVersion(
					pipeline.UUID,
					pipeline.Name,
					string(pipeline.Description),
					fmt.Sprintf("url://%v", pipeline.Name),
					test.template,
					fmt.Sprintf("uri://pipelines/%v/versions/v1/spec.yaml", pipeline.Name),
					pipeline.Namespace,
				)
				pipelineVersion, err = manager.CreatePipelineVersion(
					pv,
				)
			}

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				if test.errorIs != nil {
					assert.ErrorIs(t, err, test.errorIs)
				}
				return
			}
			require.Nil(t, err)

			test.model.CreatedAtInSec = 1
			test.model.Status = "READY"
			test.model.UUID = pipeline.UUID
			assert.Equal(t, test.model, pipeline)

			pv.UUID = pipelineVersion.UUID
			pv.PipelineId = pipelineVersion.PipelineId
			pv.CreatedAtInSec = 2
			pv.Status = "READY"
			pv.Parameters = pipelineVersion.Parameters
			assert.Equal(t, pv, pipelineVersion)
		})
	}
}

// Tests CreatePipelineVersion
func TestCreatePipelineVersion(t *testing.T) {
	initEnvVars()
	tt := []struct {
		msg            string
		template       string                 // pipeline template
		version        *model.PipelineVersion // optional
		badObjectStore bool                   // optional, object requests always fail
		badDB          bool                   // optional, DB request always fail
		// The following are expected results.
		model *model.PipelineVersion // optional, expected version model when success
		// To verify an error, set the errorCode and
		// optionally set errorMsg and errorIs based on the test's needs.
		errorCode codes.Code
		errorMsg  string // error message
		errorIs   error  // verify a wrapped error is specific instance
	}{
		{
			msg:      "HappyCase",
			template: testWorkflow.ToStringForStore(),
			version: &model.PipelineVersion{
				Name:        "p_v",
				Description: model.LargeText("test"),
			},
			model: &model.PipelineVersion{
				Name:         "p_v",
				Parameters:   "[{\"name\":\"param1\"}]",
				Description:  model.LargeText("test"),
				PipelineSpec: model.LargeText(testWorkflow.ToStringForStore()),
			},
		},
		{
			msg:      "ComplexPipeline",
			template: complexPipeline,
			version: &model.PipelineVersion{
				Name: "complex",
			},
			model: &model.PipelineVersion{
				Name:         "complex",
				Parameters:   "[{\"name\":\"output\"},{\"name\":\"project\"},{\"name\":\"schema\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json\"},{\"name\":\"train\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv\"},{\"name\":\"evaluation\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv\"},{\"name\":\"preprocess-mode\",\"value\":\"local\"},{\"name\":\"preprocess-module\",\"value\":\"gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py\"},{\"name\":\"target\",\"value\":\"tips\"},{\"name\":\"learning-rate\",\"value\":\"0.1\"},{\"name\":\"hidden-layer-size\",\"value\":\"1500\"},{\"name\":\"steps\",\"value\":\"3000\"},{\"name\":\"workers\",\"value\":\"0\"},{\"name\":\"pss\",\"value\":\"0\"},{\"name\":\"predict-mode\",\"value\":\"local\"},{\"name\":\"analyze-mode\",\"value\":\"local\"},{\"name\":\"analyze-slice-column\",\"value\":\"trip_start_hour\"}]",
				PipelineSpec: complexPipeline,
			},
		},
		{
			msg:       "InvalidTemplate",
			template:  "I am invalid yaml",
			errorCode: codes.InvalidArgument,
			errorIs:   template.ErrorInvalidPipelineSpec,
		},
		{
			msg:       "BadDB",
			template:  testWorkflow.ToStringForStore(),
			badDB:     true,
			errorCode: codes.Internal,
			errorMsg:  "database is closed",
		},
		{
			msg:      "V2PipelineSpec",
			template: v2SpecHelloWorld,
			version: &model.PipelineVersion{
				Name: "v2spec",
			},
			model: &model.PipelineVersion{
				Name: "v2spec",
				// TODO(v2): when parameter extraction is implemented, this won't be empty.
				Parameters:   "[{\"name\":\"param1\"}]",
				PipelineSpec: model.LargeText(testWorkflow.ToStringForStore()),
			},
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

			// Create a pipeline before versions.
			p0 := createPipelineV1("my_pipeline")
			pv0 := createPipelineVersion(
				"",
				"my_pipeline",
				"",
				"",
				testWorkflow.ToStringForStore(),
				"",
				"",
			)
			pipeline, err := manager.CreatePipeline(p0)
			require.Nil(t, err)
			pv0.PipelineId = pipeline.UUID
			_, err = manager.CreatePipelineVersion(pv0)
			require.Nil(t, err)

			// Override bad dependencies after create pipeline request succeeds.
			if test.badObjectStore {
				manager.objectStore = &FakeBadObjectStore{}
			}
			if test.badDB {
				store.DB().Close()
			}
			// Create a version under the above pipeline.
			var pv *model.PipelineVersion
			if test.model == nil {
				pv = createPipelineVersion(
					pipeline.UUID,
					"my_pipeline_version_name",
					"",
					"",
					test.template,
					"",
					"",
				)
			} else {
				pv = test.model
				pv.PipelineId = pipeline.UUID
			}
			version, err := manager.CreatePipelineVersion(pv)
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				if test.errorIs != nil {
					assert.ErrorIs(t, err, test.errorIs)
				}
				return
			}
			require.Nil(t, err)

			version.UUID = ""
			test.model.PipelineId = pipeline.UUID
			test.model.Status = model.PipelineVersionReady
			test.model.CreatedAtInSec = 3
			test.model.PipelineSpec = version.PipelineSpec
			assert.Equal(t, test.model, version)
		})
	}
}

// Tests CreatePipelineVersion, GetPipelineVersionTemplate and GetPipelineLatestTemplate
func TestCreatePipelineOrVersion_V2PipelineName(t *testing.T) {
	initEnvVars()
	tests := []struct {
		// inputs
		name      string
		namespace string
		template  string // template to upload
		// expected
		pipelineName string
	}{
		{name: "v2-compat", namespace: "", pipelineName: "two-step-pipeline"},
		{name: "pipe3", namespace: "", pipelineName: "two-step-pipeline"},
		{name: "pipeline2", namespace: "kubeflow", pipelineName: "two-step-pipeline"},
		{name: "abcd", namespace: "user", pipelineName: "two-step-pipeline"},
		{name: "v2-spec1", namespace: "", template: v2SpecHelloWorld, pipelineName: "hello-world"},
		{name: "v2-spec2", namespace: "user", template: v2SpecHelloWorld, pipelineName: "hello-world"},
	}
	for _, test := range tests {
		testClone := test
		testClone.template = "" // template is too long for the message
		t.Run(fmt.Sprintf("%+v", testClone), func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

			if test.template == "" {
				test.template = strings.TrimSpace(v2compatPipeline)
			}

			// Verify v2 pipeline name of CreatePipeline template.
			p := createPipeline(
				test.name,
				"",
				test.namespace,
			)
			createdPipeline, err := manager.CreatePipeline(p)
			require.Nil(t, err)

			// Verify v2 pipeline name of CreatePipelineVersion template.
			pv := createPipelineVersion(
				createdPipeline.UUID,
				"pipeline_version",
				"",
				"",
				test.template,
				"",
				"",
			)
			if pv.PipelineSpec == "" {
				pv.PipelineSpec = v2compatPipeline
			}
			version, err := manager.CreatePipelineVersion(pv)
			require.Nil(t, err)
			bytes, err := manager.GetPipelineVersionTemplate(version.UUID)
			require.Nil(t, err)
			tmpl, err := template.New(bytes, template.TemplateOptions{CacheDisabled: true})
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())

			bytes, err = manager.GetPipelineLatestTemplate(createdPipeline.UUID)
			require.Nil(t, err)
			tmpl, err = template.New(bytes, template.TemplateOptions{CacheDisabled: true})
			require.Nil(t, err)
			assert.Equal(t, test.pipelineName, tmpl.V2PipelineName())
		})
	}
}

func TestResourceManager_CreatePipelineAndPipelineVersion(t *testing.T) {
	tests := []struct {
		name         string
		p            *model.Pipeline
		pv           *model.PipelineVersion
		wantPipeline *model.Pipeline
		wantVersion  *model.PipelineVersion
		wantErr      bool
		errorMsg     string
	}{
		{
			"Valid - pipeline v2",
			&model.Pipeline{
				Name:        "pipeline v2",
				Description: model.LargeText("pipeline two"),
				Namespace:   "user1",
			},
			&model.PipelineVersion{
				Name:            "pipeline v2 version 1",
				Description:     model.LargeText("pipeline v2 version description"),
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    model.LargeText(v2SpecHelloWorld),
				PipelineSpecURI: model.LargeText("pipeline_version_two.yaml"),
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 1,
				Name:           "pipeline v2",
				DisplayName:    "pipeline v2",
				Description:    model.LargeText("pipeline two"),
				Namespace:      "user1",
				Status:         model.PipelineReady,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  2,
				Name:            "pipeline v2 version 1",
				DisplayName:     "pipeline v2 version 1",
				Description:     model.LargeText("pipeline v2 version description"),
				PipelineId:      DefaultFakePipelineIdTwo,
				Status:          model.PipelineVersionReady,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    model.LargeText(v2SpecHelloWorld),
				PipelineSpecURI: model.LargeText("pipeline_version_two.yaml"),
				Parameters:      "[]",
			},
			false,
			"",
		},
		{
			"Valid - pipeline v2 (with name and display name)",
			&model.Pipeline{
				Name:        "pipeline v2",
				DisplayName: "pipeline v2 display name",
				Description: model.LargeText("pipeline two"),
				Namespace:   "user1",
			},
			&model.PipelineVersion{
				Name:            "pipeline v2 version 1",
				DisplayName:     "pipeline v2 version 1 display name",
				Description:     model.LargeText("pipeline v2 version description"),
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    model.LargeText(v2SpecHelloWorld),
				PipelineSpecURI: model.LargeText("pipeline_version_two.yaml"),
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 1,
				Name:           "pipeline v2",
				DisplayName:    "pipeline v2 display name",
				Description:    model.LargeText("pipeline two"),
				Namespace:      "user1",
				Status:         model.PipelineReady,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  2,
				Name:            "pipeline v2 version 1",
				DisplayName:     "pipeline v2 version 1 display name",
				Description:     model.LargeText("pipeline v2 version description"),
				PipelineId:      DefaultFakePipelineIdTwo,
				Status:          model.PipelineVersionReady,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    model.LargeText(v2SpecHelloWorld),
				PipelineSpecURI: model.LargeText("pipeline_version_two.yaml"),
				Parameters:      "[]",
			},
			false,
			"",
		},
		{
			"Valid - pipeline v1",
			&model.Pipeline{
				Name:        "pipeline v1",
				Description: model.LargeText("pipeline one"),
				Parameters:  `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
			},
			&model.PipelineVersion{
				Name:            "pipeline v1 version 1",
				Description:     model.LargeText("pipeline v1 version description"),
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    model.LargeText(complexPipeline),
				PipelineSpecURI: model.LargeText("pipeline_version_one.yaml"),
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 1,
				Name:           "pipeline v1",
				DisplayName:    "pipeline v1",
				Description:    model.LargeText("pipeline one"),
				Parameters:     `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
				Status:         model.PipelineReady,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  2,
				PipelineId:      DefaultFakePipelineIdTwo,
				Name:            "pipeline v1 version 1",
				DisplayName:     "pipeline v1 version 1",
				Description:     model.LargeText("pipeline v1 version description"),
				Status:          model.PipelineVersionReady,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    model.LargeText(complexPipeline),
				PipelineSpecURI: model.LargeText("pipeline_version_one.yaml"),
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
			assert.True(t, ok)
			pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))

			gotPipeline, gotVersion, err := manager.CreatePipelineAndPipelineVersion(tt.p, tt.pv)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, gotPipeline)
				assert.Nil(t, gotVersion)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.wantPipeline, gotPipeline)
				tt.wantVersion.PipelineSpec = gotVersion.PipelineSpec
				tt.wantVersion.Parameters = gotVersion.Parameters
				assert.Equal(t, tt.wantVersion, gotVersion)
			}
		})
	}
}

func TestCreatePipelineAndPipelineVersion_V1Blocked(t *testing.T) {
	viper.Set(util.BlockV1Pipelines, "true")
	viper.Set(v1AllowedNamespaces, "ns1")
	viper.Set(common.PodNamespace, "ns1")
	defer func() {
		viper.Set(util.BlockV1Pipelines, nil)
		viper.Set(v1AllowedNamespaces, nil)
		viper.Set(common.PodNamespace, nil)
	}()

	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	_, _, err := manager.CreatePipelineAndPipelineVersion(
		&model.Pipeline{Name: "v1-pipeline", Namespace: "blocked-ns"},
		&model.PipelineVersion{
			Name:         "v1-version",
			PipelineSpec: complexPipeline,
		},
	)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "V1 pipeline specs are not allowed")
}

func TestCreatePipelineAndPipelineVersion_V1Blocked_PodNamespaceFallback(t *testing.T) {
	viper.Set(util.BlockV1Pipelines, "true")
	viper.Set(v1AllowedNamespaces, "ns1")
	viper.Set(common.PodNamespace, "other-ns")
	defer func() {
		viper.Set(util.BlockV1Pipelines, nil)
		viper.Set(v1AllowedNamespaces, nil)
		viper.Set(common.PodNamespace, nil)
	}()

	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	_, _, err := manager.CreatePipelineAndPipelineVersion(
		&model.Pipeline{Name: "v1-pipeline"},
		&model.PipelineVersion{
			Name:         "v1-version",
			PipelineSpec: complexPipeline,
		},
	)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "V1 pipeline specs are not allowed")
}

func TestCreatePipelineVersion_V1Blocked(t *testing.T) {
	viper.Set(util.BlockV1Pipelines, "true")
	viper.Set(v1AllowedNamespaces, "ns1")
	viper.Set(common.PodNamespace, "ns1")
	defer func() {
		viper.Set(util.BlockV1Pipelines, nil)
		viper.Set(v1AllowedNamespaces, nil)
		viper.Set(common.PodNamespace, nil)
	}()

	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, err := manager.CreatePipeline(&model.Pipeline{Name: "test-pipeline", Namespace: "blocked-ns"})
	require.Nil(t, err)

	_, err = manager.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1-version",
		PipelineId:   p.UUID,
		PipelineSpec: complexPipeline,
	})
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "V1 pipeline specs are not allowed")
}

// Tests GetPipelineByNameAndNamespace
func TestGetPipelineByNameAndNamespace(t *testing.T) {
	tt := []struct {
		msg          string
		pipelineName string
		namespace    string
		badDB        bool
		errorCode    codes.Code
		errMsg       string
	}{
		{
			msg:          "OK",
			pipelineName: "p1",
			namespace:    "ns1",
			errorCode:    codes.OK,
		},
		{
			msg:          "NotFount",
			pipelineName: "doesNotExists",
			namespace:    "ns1",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "SharedPipelineNotFound",
			pipelineName: "p1",
			namespace:    "wrongNamespace",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "BadDB",
			pipelineName: "p1",
			namespace:    "ns1",
			badDB:        true,
			errorCode:    codes.Internal,
			errMsg:       "database is closed",
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store, manager, p, _ := initWithPipeline(t)
			if test.badDB {
				store.Close()
			}

			result, err := manager.GetPipelineByNameAndNamespace(
				test.pipelineName,
				test.namespace,
			)

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
				return
			}
			require.Nil(t, err)
			assert.Equal(t, result, p)
		})
	}
}

// Tests GetPipelineByNameAndNamespaceV1
func TestGetPipelineByNameAndNamespaceV1(t *testing.T) {
	tt := []struct {
		msg          string
		pipelineName string
		namespace    string
		badDB        bool
		errorCode    codes.Code
		errMsg       string
	}{
		{
			msg:          "OK",
			pipelineName: "p1",
			namespace:    "ns1",
			errorCode:    codes.OK,
		},
		{
			msg:          "NotFount",
			pipelineName: "doesNotExists",
			namespace:    "ns1",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "SharedPipelineNotFound",
			pipelineName: "p1",
			namespace:    "wrongNamespace",
			errorCode:    codes.NotFound,
		},
		{
			msg:          "BadDB",
			pipelineName: "p1",
			namespace:    "ns1",
			badDB:        true,
			errorCode:    codes.Internal,
			errMsg:       "database is closed",
		},
	}
	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			store, manager, p, pv := initWithPipeline(t)
			if test.badDB {
				store.Close()
			}

			resp, respv, err := manager.GetPipelineByNameAndNamespaceV1(
				test.pipelineName,
				test.namespace,
			)

			// verify result
			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
				return
			}
			require.Nil(t, err)
			assert.Equal(t, p, resp)
			assert.Equal(t, pv, respv)
		})
	}
}

// Tests GetPipelineLatestTemplate (from PipelineSpec)
func TestGetLatestPipelineVersion(t *testing.T) {
	store, manager, p, pv := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetLatestPipelineVersion(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, pv, actualTemplate)

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	assert.True(t, ok)
	pv2 := createPipelineVersion(
		p.UUID,
		"new version",
		"new version desc",
		"url://pipelines/p1/versions/v2",
		testWorkflow.ToStringForStore(),
		"uri://pipelines/p1/versions/v2/spec.yaml",
		p.Namespace,
	)
	pv2expected, _ := manager.CreatePipelineVersion(pv2)
	pv2.UUID = pv2expected.UUID
	pv2.CreatedAtInSec = pv2expected.CreatedAtInSec
	pv2.Status = model.PipelineVersionReady
	actualTemplate2, err := manager.GetLatestPipelineVersion(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, pv2, actualTemplate2)
}

// Tests GetPipelineLatestTemplate (from PipelineSpec)
func TestGetPipelineTemplate(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, []byte(testWorkflow.ToStringForStore()), actualTemplate)
}

// Tests GetPipelineLatestTemplate (from PipelineSpecURI)
func TestGetPipelineTemplate_FromPipelineURI(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	manager.objectStore.AddFile(context.TODO(), []byte(testWorkflow.ToStringForStore()), p.UUID)
	pv := &model.PipelineVersion{
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: model.LargeText(p.UUID),
	}
	_, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (from PipelineVersionId)
func TestGetPipelineTemplate_FromPipelineVersionId(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	pv := &model.PipelineVersion{
		UUID:            "1000",
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: model.LargeText(manager.objectStore.GetPipelineKey(p.UUID)),
	}

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	assert.True(t, ok)

	manager.objectStore.AddFile(context.TODO(), []byte(testWorkflow.ToStringForStore()), manager.objectStore.GetPipelineKey(p.UUID))
	pv2, err := manager.CreatePipelineVersion(pv)
	require.Nil(t, err, "CreatePipelineVersion failed: %v", err)
	assert.NotEqual(t, p.UUID, pv2.UUID)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (from PipelineId)
func TestGetPipelineTemplate_FromPipelineId(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	p, _ := manager.CreatePipeline(createPipelineV1("new_pipeline"))
	pv := &model.PipelineVersion{
		PipelineId:      p.UUID,
		Name:            "new_version",
		PipelineSpecURI: model.LargeText(manager.objectStore.GetPipelineKey(p.UUID)),
	}

	manager.objectStore.AddFile(context.TODO(), []byte(testWorkflow.ToStringForStore()), manager.objectStore.GetPipelineKey(p.UUID))

	pipelineStore, ok := manager.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv2, err := manager.CreatePipelineVersion(pv)
	require.Nil(t, err, "CreatePipelineVersion failed: %v", err)
	assert.NotEqual(t, p.UUID, pv2.UUID)

	tmpl, err := manager.GetPipelineLatestTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(tmpl), "argoproj.io/v1alpha1")
}

// Tests GetPipelineLatestTemplate (NotFound)
func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.objectStore.AddFile(context.TODO(), template, store.objectStore.GetPipelineKey(fmt.Sprint(1)))
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	_, err := manager.GetPipelineLatestTemplate("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

// Tests GetPipelineLatestTemplate (pipelineSpec NotFound)
func TestGetPipelineTemplate_PipelineFileNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipelineV1("pipeline1"))
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	_, err := manager.GetPipelineLatestTemplate(pipeline.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// Tests ListPipelines
func TestListPipelines(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	assert.Nil(t, err)

	_, nTotal, _, err := manager.ListPipelines(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID, false)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelines(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests ListPipelinesV1
func TestListPipelinesV1(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	assert.Nil(t, err)

	_, _, nTotal, _, err := manager.ListPipelinesV1(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID, false)
	assert.Nil(t, err)

	_, _, nTotal, _, err = manager.ListPipelinesV1(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""}},
		opts,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests ListPipelineVersions
func TestListPipelineVersions(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew1.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "", nil)
	assert.Nil(t, err)

	_, nTotal, _, err := manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID, false)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, nTotal)

	// Delete a pipeline version
	err = manager.DeletePipelineVersion(FakeUUIDOne)
	assert.Nil(t, err)

	_, nTotal, _, err = manager.ListPipelineVersions(
		pnew1.UUID,
		opts,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, nTotal)
}

// Tests UpdatePipelineStatus
func TestUpdatePipelineStatus(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)
	p2 := createPipelineV1(
		"pipeline2",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew2.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	p1retrieved, err := manager.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineReady, p1retrieved.Status)

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineCreating)
	assert.Nil(t, err)
	_, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineDeleting)
	assert.Nil(t, err)
	_, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineReady)
	assert.Nil(t, err)
	p1retrieved, err = manager.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineReady, p1retrieved.Status)
}

// Tests that the go-swagger UpdatePipeline request body correctly serializes
// empty tags as {"tags":{}} (not omitted), so the server can distinguish
// "clear all tags" from "don't change tags".
func TestUpdatePipelineBody_EmptyTagsSerialization(t *testing.T) {
	type UpdateBody struct {
		Tags map[string]string `json:"tags"`
	}

	// Empty map must serialize to {"tags":{}}
	body := UpdateBody{Tags: map[string]string{}}
	data, err := json.Marshal(body)
	assert.Nil(t, err)
	assert.Contains(t, string(data), `"tags":{}`, "Empty tags map must be present in serialized JSON")

	// Nil map must serialize to {"tags":null}
	bodyNil := UpdateBody{Tags: nil}
	dataNil, err := json.Marshal(bodyNil)
	assert.Nil(t, err)
	assert.Contains(t, string(dataNil), `"tags":null`, "Nil tags must serialize as null")

	// Verify server-side: unmarshal null back to nil
	var decoded UpdateBody
	err = json.Unmarshal(dataNil, &decoded)
	assert.Nil(t, err)
	assert.Nil(t, decoded.Tags, "Null tags should unmarshal to nil")

	// Verify server-side: unmarshal {} back to empty (non-nil) map
	var decodedEmpty UpdateBody
	err = json.Unmarshal(data, &decodedEmpty)
	assert.Nil(t, err)
	assert.NotNil(t, decodedEmpty.Tags, "Empty tags object should unmarshal to non-nil map")
	assert.Empty(t, decodedEmpty.Tags, "Empty tags object should unmarshal to empty map")
}

// Tests UpdatePipeline clears tags when an empty map is passed.
func TestUpdatePipeline_ClearTags(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline with tags.
	p := &model.Pipeline{
		Name:   "pipeline-with-tags",
		Status: model.PipelineReady,
		Tags:   map[string]string{"team": "ml-ops", "env": "prod"},
	}
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	createdPipeline, err := manager.CreatePipeline(p)
	assert.Nil(t, err)

	// Verify tags are set.
	retrieved, err := manager.GetPipeline(createdPipeline.UUID)
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"team": "ml-ops", "env": "prod"}, retrieved.Tags)

	// Clear tags by passing an empty map (not nil).
	updated, err := manager.UpdatePipeline(createdPipeline.UUID, "", map[string]string{})
	assert.Nil(t, err)
	assert.Empty(t, updated.Tags, "Tags should be empty after clearing with empty map")

	// Verify via GetPipeline.
	retrieved, err = manager.GetPipeline(createdPipeline.UUID)
	assert.Nil(t, err)
	assert.Empty(t, retrieved.Tags, "Tags should be empty after clearing with empty map")
}

// Tests UpdatePipeline does not modify tags when nil is passed.
func TestUpdatePipeline_NilTagsNoChange(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline with tags.
	p := &model.Pipeline{
		Name:   "pipeline-with-tags",
		Status: model.PipelineReady,
		Tags:   map[string]string{"team": "ml-ops"},
	}
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	createdPipeline, err := manager.CreatePipeline(p)
	assert.Nil(t, err)

	// Update with nil tags should not change existing tags.
	updated, err := manager.UpdatePipeline(createdPipeline.UUID, "new-name", nil)
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"team": "ml-ops"}, updated.Tags, "Tags should remain unchanged when nil is passed")
	assert.Equal(t, "new-name", updated.DisplayName)
}

// Tests UpdatePipelineVersionStatus
func TestUpdatePipelineVersionStatus(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)
	p2 := createPipelineV1(
		"pipeline2",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pv2 := createPipelineVersion(
		pnew2.UUID,
		"pipelinev2",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))
	_, err = manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	p1retrieved, err := manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineVersionReady, p1retrieved.Status)

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionCreating)
	assert.Nil(t, err)
	_, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	_, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	err = manager.UpdatePipelineVersionStatus(DefaultFakePipelineIdTwo, model.PipelineVersionReady)
	assert.Nil(t, err)
	p1retrieved, err = manager.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(t, model.PipelineVersionReady, p1retrieved.Status)
}

func TestDeletePipelineVersion(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p := createPipelineV1(
		"pipeline",
	)
	pnew, err := manager.CreatePipeline(p)
	assert.Nil(t, err)
	// Create a version under the above pipeline.
	pv := createPipelineVersion(
		pnew.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	// Create a version under the above pipeline.
	pv2 := createPipelineVersion(
		pnew.UUID,
		"pipeline_version",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pnew2, err := manager.CreatePipelineVersion(pv2)
	assert.Nil(t, err)

	// Delete the above pipeline_version.
	err = manager.DeletePipelineVersion(pnew2.UUID)
	assert.Nil(t, err)

	// Verify the version doesn't exist.
	_, err = manager.GetPipelineVersion(FakeUUIDOne)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Verify the first version exists.
	_, err = manager.GetPipelineVersion(DefaultFakeUUID)
	assert.Nil(t, err)

	// Verify the latest version
	pvLatestTeplate, err := manager.GetPipelineLatestTemplate(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}", string(pvLatestTeplate))
}

// Tests DeletePipelineVersion (NotFound)
func TestDeletePipelineVersion_FileError(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p := createPipelineV1(
		"pipeline",
	)
	pnew, err := manager.CreatePipeline(p)
	assert.Nil(t, err)
	// Create a version under the above pipeline.
	pv := createPipelineVersion(
		pnew.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager.CreatePipelineVersion(pv)

	// Switch to a bad object store
	manager.objectStore = &FakeBadObjectStore{}

	// Delete the above pipeline_version.
	err = manager.DeletePipelineVersion(FakeUUIDOne)
	assert.Nil(t, err)

	// Verify the version in deleting status.
	version, err := manager.pipelineStore.GetPipelineVersionWithStatus(FakeUUIDOne, model.PipelineVersionDeleting)
	assert.NotNil(t, err)
	assert.Nil(t, version)
}

// Tests DeletePipeline
func TestDeletePipeline(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	// Create a pipeline.
	p1 := createPipelineV1(
		"pipeline1",
	)
	pnew1, err := manager.CreatePipeline(p1)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pnew1.UUID,
		"pipeline",
		"",
		"",
		"apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
		"",
		"",
	)

	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	p2 := createPipelineV1(
		"pipeline2",
	)
	pnew2, err := manager.CreatePipeline(p2)
	assert.Nil(t, err)

	// Delete the above pipeline.
	err = manager.DeletePipeline(pnew2.UUID, false)
	assert.Nil(t, err)

	// Verify the pipeline doesn't exist.
	_, err = manager.GetPipeline(FakeUUIDOne)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Verify the first pipeline exists.
	_, err = manager.GetPipeline(DefaultFakeUUID)
	assert.Nil(t, err)

	// Must fail due to active pipeline versions
	err = manager.DeletePipeline(pnew1.UUID, false)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("as it has existing pipeline versions (e.g. %v)", FakeUUIDOne))
}

func TestCreateRun_BlockV1Pipelines(t *testing.T) {
	tt := []struct {
		msg               string
		blockV1           bool
		allowedNamespaces string
		namespace         string
		useV2Spec         bool
		errorCode         codes.Code
		errorMsg          string
	}{
		{
			msg:               "BlockV1_NamespaceNotAllowed",
			blockV1:           true,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         false,
			errorCode:         codes.InvalidArgument,
			errorMsg:          "not allowed to run v1 pipelines",
		},
		{
			msg:               "BlockV1_NamespaceAllowed",
			blockV1:           true,
			allowedNamespaces: "ns1",
			namespace:         "ns1",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_NamespaceAllowed_MultipleNamespaces",
			blockV1:           true,
			allowedNamespaces: "ns1,ns2,ns3",
			namespace:         "ns2",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_Disabled_AnyNamespaceAllowed",
			blockV1:           false,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_V2PipelineNotBlocked",
			blockV1:           true,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         true,
		},
		{
			msg:               "BlockV1_NamespaceNotInAllowedList",
			blockV1:           true,
			allowedNamespaces: "ns2,ns3",
			namespace:         "ns1",
			useV2Spec:         false,
			errorCode:         codes.InvalidArgument,
			errorMsg:          "Namespace ns1 is not allowed to run v1 pipelines",
		},
		{
			msg:               "BlockV1_CaseInsensitiveNamespaceMatch",
			blockV1:           true,
			allowedNamespaces: "NS1",
			namespace:         "ns1",
			useV2Spec:         false,
		},
	}

	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			viper.Set(util.BlockV1Pipelines, test.blockV1)
			viper.Set(v1AllowedNamespaces, test.allowedNamespaces)
			defer func() {
				viper.Set(util.BlockV1Pipelines, nil)
				viper.Set(v1AllowedNamespaces, nil)
			}()

			store, manager, exp := initWithExperiment(t)
			defer store.Close()

			var apiRun *model.Run
			if test.useV2Spec {
				apiRun = &model.Run{
					DisplayName:  "run1",
					ExperimentId: exp.UUID,
					Namespace:    test.namespace,
					PipelineSpec: model.PipelineSpec{
						PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
						RuntimeConfig: model.RuntimeConfig{
							Parameters: `{"text":"world"}`,
						},
					},
				}
			} else {
				apiRun = &model.Run{
					DisplayName:  "run1",
					ExperimentId: exp.UUID,
					Namespace:    test.namespace,
					PipelineSpec: model.PipelineSpec{
						WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
						Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
					},
				}
			}

			_, err := manager.CreateRun(context.Background(), apiRun)

			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				return
			}
			assert.Nil(t, err)
		})
	}
}

// TODO: use table driven test to test CreateRun api
func TestCreateRun_ThroughPipelineID(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(p.UUID, "version_for_run", "", "", testWorkflow.ToStringForStore(), "", "")
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.ResourceVersion = "1"
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.ObjectMeta.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}
	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId:    version.UUID,
			PipelineId:           p.UUID,
			PipelineName:         "version_for_run",
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			WorkflowRuntimeManifest: model.LargeText(util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore()),
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRunV2(t)
	expectedExperimentUUID := runDetail.ExperimentId

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "hello-world-0",
		ServiceAccount: "pipeline-runner",
		Namespace:      runDetail.Namespace,
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	expectedRunDetail.PipelineSpec.PipelineSpecManifest = runDetail.PipelineSpec.PipelineSpecManifest
	expectedRunDetail.RunDetails.PipelineRuntimeManifest = runDetail.RunDetails.PipelineRuntimeManifest
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpec(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	expectedExperimentUUID := runDetail.ExperimentId
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.ResourceVersion = "1"
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			State:            "PENDING",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
			WorkflowRuntimeManifest: model.LargeText(util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore()),
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecWithPatch(t *testing.T) {
	viper.Set(common.HasDefaultBucketEnvVar, "true")
	viper.Set(common.ProjectIDEnvVar, "test-project-id")
	viper.Set(common.DefaultBucketNameEnvVar, "test-default-bucket")
	store, manager, runDetail := initWithPatchedRun(t)
	expectedExperimentUUID := runDetail.ExperimentId
	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.ResourceVersion = "1"
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("test-default-bucket")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
			},
			WorkflowRuntimeManifest: model.LargeText(util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore()),
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
		},
	}
	expectedRunDetail.PipelineSpec.PipelineName = runDetail.PipelineSpec.PipelineName
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughWorkflowSpecSameManifest(t *testing.T) {
	viper.Set(common.HasDefaultBucketEnvVar, "true")
	viper.Set(common.ProjectIDEnvVar, "test-project-id")
	viper.Set(common.DefaultBucketNameEnvVar, "test-default-bucket")
	_, manager, runDetail := initWithPatchedRun(t)

	manager.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore, _ := manager.pipelineStore.(*storage.PipelineStore)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil))

	newRun, err := manager.CreateRun(
		context.Background(),
		&model.Run{
			DisplayName: "run1",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
				Parameters:           "[{\"name\":\"param1\",\"value\":\"{{kfp-default-bucket}}\"}]",
			},
			ExperimentId: runDetail.ExperimentId,
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, "run1", newRun.DisplayName)
	assert.Empty(t, newRun.PipelineId)
	assert.Empty(t, newRun.PipelineVersionId)
	assert.NotEqual(t, runDetail.WorkflowRuntimeManifest, newRun.WorkflowRuntimeManifest)
	assert.Equal(t, runDetail.WorkflowSpecManifest, newRun.WorkflowSpecManifest)
	assert.Empty(t, newRun.PipelineSpecManifest)
}

func TestCreateRun_ThroughPipelineVersion(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "sa1")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_run",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
			PipelineVersionId: version.UUID,
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.ResourceVersion = "1"
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "sa1",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId:    version.UUID,
			PipelineId:           version.PipelineId,
			PipelineName:         version.Name,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: model.LargeText(util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore()),
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
	}
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "sa1")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")
	// Create experiment, pipeline, and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_run",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:        pipeline.UUID,
			PipelineVersionId: version.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ServiceAccount: "sa1",
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.ResourceVersion = "1"
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "sa1"
	expectedRuntimeWorkflow.Namespace = "ns1"
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: DefaultFakeUUID,
		},
	}

	expectedRunDetail := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "sa1",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: model.LargeText(util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore()),
			CreatedAtInSec:          5,
			ScheduledAtInSec:        5,
			Conditions:              "Pending",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 6,
					State:           model.RuntimeStatePending,
				},
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipeline.UUID,
			PipelineVersionId:    version.UUID,
			PipelineName:         version.Name,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	expectedRunDetail = expectedRunDetail.ToV2().ToV1()
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "The CreateRun return has unexpected value")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "Workflow CRD is not created")
	runDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1(), "CreateRun stored invalid data in database")
}

func TestCreateRun_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch a template with an empty pipeline spec manifest")
}

func TestCreateRun_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText("I am invalid"),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRun_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "null", // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRun_OverrideParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unrecognized input parameter")
}

func TestCreateRun_CreateWorkflowError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a workflow")
}

func TestCreateRun_StoreRunMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	store.DB().Close()
	apiRun := &model.Run{
		DisplayName:  "run1",
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "database is closed")
}

func TestCreateRun_WithMLflowPlugin(t *testing.T) {
	// Set up a fake MLflow server that handles experiment lookup and run creation.
	// Tags are passed inline in the CreateRun body (atomic tagging).
	var (
		experimentGetCalled bool
		runCreateCalled     bool
		createRunBody       string
	)
	mlflowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			experimentGetCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"mlflow-exp-1","name":"Default"}}`))
		case "/api/2.0/mlflow/runs/create":
			runCreateCalled = true
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			createRunBody = string(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-parent-run-1"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mlflowServer.Close()

	setupTestSAToken(t, "test-sa-token")
	setupMLflowViperConfig(t, mlflowServer.URL)

	store, manager, exp := initWithExperiment(t)
	defer store.Close()

	// Build a run with plugins_input that triggers MLflow integration.
	pluginsInput := `{"mlflow":{"experiment_name":"Default"}}`
	pluginsInputLT := model.LargeText(pluginsInput)
	apiRun := &model.Run{
		DisplayName: "mlflow-test-run",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
		RunDetails: model.RunDetails{
			PluginsInputString: &pluginsInputLT,
		},
	}

	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	require.NoError(t, err)
	require.NotNil(t, runDetail)

	// Verify MLflow API calls were made.
	assert.True(t, experimentGetCalled, "MLflow experiment lookup should have been called")
	assert.True(t, runCreateCalled, "MLflow run creation should have been called")
	assert.Contains(t, createRunBody, "kfp.pipeline_run_id", "CreateRun body should contain KFP tags")

	// Verify plugins_output is persisted on the run.
	storedRun, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	require.NotNil(t, storedRun.PluginsOutputString, "PluginsOutputString should be set")
	assert.Contains(t, string(*storedRun.PluginsOutputString), "mlflow-parent-run-1")
	assert.Contains(t, string(*storedRun.PluginsOutputString), "mlflow-exp-1")

	// Parse and verify the plugin output structure.
	outputs, err := apiserverPlugins.DeserializePluginsOutput(storedRun.PluginsOutputString)
	require.NoError(t, err)
	output := outputs["mlflow"]
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Equal(t, "mlflow-exp-1", output.Entries["experiment_id"].Value.GetStringValue())
	assert.Equal(t, "mlflow-parent-run-1", output.Entries[apiserverPlugins.EntryRootRunID].Value.GetStringValue())
	assert.Contains(t, output.Entries[apiserverPlugins.EntryRunURL].Value.GetStringValue(), "mlflow-parent-run-1")

}

// TestCreateRun_NoMLflowConfig verifies that run creation succeeds without
// error when no MLflow plugin config is set at either the global or namespace
// level.  The unconditional MLflow dispatcher must short-circuit cleanly.
func TestCreateRun_NoMLflowConfig(t *testing.T) {
	// Ensure no global MLflow config.
	origConfig := viper.Get("plugins.mlflow")
	viper.Set("plugins.mlflow", nil)
	t.Cleanup(func() {
		viper.Set("plugins.mlflow", origConfig)
	})

	store, manager, exp := initWithExperiment(t)
	defer store.Close()

	apiRun := &model.Run{
		DisplayName: "no-mlflow-run",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}

	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	require.NoError(t, err)
	require.NotNil(t, runDetail)

	// Verify plugins_output is not set (no plugin ran).
	storedRun, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	assert.True(t,
		storedRun.PluginsOutputString == nil || *storedRun.PluginsOutputString == "",
		"PluginsOutputString should be empty when MLflow is not configured",
	)
}

func TestDeleteRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	manager.storedWorkflowIdentities.loadOrStore(runDetail.UUID, storedWorkflowIdentity{
		name: runDetail.K8SName,
		uid:  types.UID(runDetail.UUID),
	})
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	_, err = manager.GetRun(runDetail.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
	_, found := manager.storedWorkflowIdentities.load(runDetail.UUID)
	assert.False(t, found)
}

func TestStoredWorkflowIdentityCacheIsBounded(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	for index := 0; index < storedWorkflowIdentityCacheCapacity; index++ {
		runID := fmt.Sprintf("run-%d", index)
		cache.loadOrStore(runID, storedWorkflowIdentity{name: runID, uid: types.UID(runID)})
	}

	_, found := cache.load("run-0")
	require.True(t, found)
	cache.loadOrStore("overflow", storedWorkflowIdentity{name: "overflow", uid: "overflow"})

	cache.mu.Lock()
	assert.Len(t, cache.entries, storedWorkflowIdentityCacheCapacity)
	cache.mu.Unlock()
	_, found = cache.load("run-0")
	assert.True(t, found, "recently used identity should remain cached")
	_, found = cache.load("run-1")
	assert.False(t, found, "least recently used identity should be evicted")
}

func TestStoredWorkflowIdentityCacheRejectsOlderRetryGenerationReplacement(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	current := storedWorkflowIdentity{name: "workflow", uid: "new-uid", retryGeneration: 2}
	stale := storedWorkflowIdentity{name: "workflow", uid: "old-uid", retryGeneration: 1}

	cache.loadOrStore("run", current)
	actual := cache.loadOrStore("run", stale)

	assert.Equal(t, current, actual)
	cached, found := cache.load("run")
	require.True(t, found)
	assert.Equal(t, current, cached)
}

func TestStoredWorkflowIdentityCacheRefreshesSameGenerationManifest(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	stale := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "old-uid",
		retryGeneration: 1,
		manifestDigest:  sha256.Sum256([]byte("old-manifest")),
	}
	current := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "new-uid",
		retryGeneration: 1,
		manifestDigest:  sha256.Sum256([]byte("new-manifest")),
	}

	cache.loadOrStore("run", stale)
	actual := cache.loadOrStore("run", current)

	assert.Equal(t, current, actual)
	cached, found := cache.load("run")
	require.True(t, found)
	assert.Equal(t, current, cached)
}

func TestStoredWorkflowIdentityCacheRefreshesAfterPersistedAdoption(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	oldIdentity := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "old-uid",
		retryGeneration: 1,
		manifestDigest:  sha256.Sum256([]byte("old-manifest")),
	}
	adoptedIdentity := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "new-uid",
		retryGeneration: 1,
		manifestDigest:  sha256.Sum256([]byte("adopted-manifest")),
	}

	cache.loadOrStore("run", oldIdentity)
	cache.replaceAfterPersist("run", oldIdentity.manifestDigest, adoptedIdentity)

	actual, found := cache.load("run")
	require.True(t, found)
	assert.Equal(t, adoptedIdentity, actual)
}

func TestStoredWorkflowIdentityCacheReplaceRejectsOlderRetryGeneration(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	current := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "new-uid",
		retryGeneration: 2,
		manifestDigest:  sha256.Sum256([]byte("current-manifest")),
	}
	stale := storedWorkflowIdentity{
		name:            "workflow",
		uid:             "old-uid",
		retryGeneration: 1,
		manifestDigest:  sha256.Sum256([]byte("stale-manifest")),
	}

	cache.replaceAfterPersist("run", sha256.Sum256([]byte("initial-manifest")), current)
	cache.replaceAfterPersist("run", sha256.Sum256([]byte("initial-manifest")), stale)

	cached, found := cache.load("run")
	require.True(t, found)
	assert.Equal(t, current, cached)
}

func TestStoredWorkflowIdentityCacheDelayedRefreshDoesNotReplaceNewerManifest(t *testing.T) {
	cache := storedWorkflowIdentityCache{}
	initial := storedWorkflowIdentity{retryGeneration: 1, manifestDigest: sha256.Sum256([]byte("manifest-0"))}
	first := storedWorkflowIdentity{retryGeneration: 1, manifestDigest: sha256.Sum256([]byte("manifest-1"))}
	second := storedWorkflowIdentity{retryGeneration: 1, manifestDigest: sha256.Sum256([]byte("manifest-2"))}

	cache.loadOrStore("run", initial)
	cache.replaceAfterPersist("run", initial.manifestDigest, first)
	cache.replaceAfterPersist("run", first.manifestDigest, second)
	cache.replaceAfterPersist("run", initial.manifestDigest, first)

	cached, found := cache.load("run")
	require.True(t, found)
	assert.Equal(t, second, cached)
}

func TestDeleteRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.DeleteRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_CrdFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	// assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	// assert.Contains(t, err.Error(), "some error")
	// TODO(IronPan) This should return error if swf CRD doesn't cascade delete runs.
	assert.Nil(t, err)
}

func TestDeleteRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteRun(context.Background(), runDetail.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)

	_, err = manager.GetExperiment(experiment.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_ClearsDefaultExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	// Set default experiment ID. This is not normally done manually
	err := manager.SetDefaultExperimentId(experiment.UUID)
	assert.Nil(t, err)
	// Verify that default experiment ID is set
	defaultExperimentId, err := manager.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, experiment.UUID, defaultExperimentId)

	err = manager.DeleteExperiment(experiment.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Experiment id cannot be equal to the default id")
}

func TestDeleteExperiment_ExperimentNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.DeleteExperiment("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_CrdFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)
}

func TestDeleteExperiment_DbFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestTerminateRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	err := manager.TerminateRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, "Terminating", actualRunDetail.Conditions)

	workflowNamespace := runDetail.Namespace
	if manager.IsEmptyNamespace(workflowNamespace) {
		workflowNamespace = common.GetPodNamespace()
	}
	isTerminated, err := store.ExecClientFake.IsTerminatedInNamespace(workflowNamespace, runDetail.K8SName)
	assert.Nil(t, err)
	assert.True(t, isTerminated)
}

func TestTerminateRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.TerminateRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestTerminateRun_DbFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()

	store.DB().Close()
	err := manager.TerminateRun(context.Background(), runDetail.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestRetryRun(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	actualRunDetail, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(actualRunDetail.WorkflowRuntimeManifest), "Failed")

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)

	actualRunDetail, err = manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Contains(t, string(actualRunDetail.WorkflowRuntimeManifest), "Running")
	assert.Equal(t, actualRunDetail.RunDetails.State, model.RuntimeStateRunning)
}

func TestRetryRun_RefreshesDivergentWorkflowName(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()
	expectedWorkflowName := runDetail.K8SName

	storedRun, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	storedRun.K8SName = "stale-workflow-name"
	require.NoError(t, manager.runStore.UpdateRun(storedRun))

	require.NoError(t, manager.RetryRun(context.Background(), runDetail.UUID))
	retriedRun, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	assert.Equal(t, expectedWorkflowName, retriedRun.K8SName)
}

func TestRetryRun_RetriesWorkflowUpdateConflict(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := client.NewWorkflowClientFake()
	seedRetryWorkflow(t, manager, runDetail.UUID, workflowClient)
	conflictWorkflowClient := &updateConflictWorkflowClient{
		FakeWorkflowClient:       workflowClient,
		updateConflictsRemaining: 1,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: conflictWorkflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.NoError(t, err)
	assert.Equal(t, 0, conflictWorkflowClient.updateConflictsRemaining)
	assert.Equal(t, 0, conflictWorkflowClient.createCalls)
}

func TestRetryRun_ReturnsUnavailableWhenWorkflowUpdateConflictPersists(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := client.NewWorkflowClientFake()
	seedRetryWorkflow(t, manager, runDetail.UUID, workflowClient)
	conflictWorkflowClient := &persistentConflictWorkflowClient{
		FakeWorkflowClient: workflowClient,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: conflictWorkflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable), "expected retryable error after persistent workflow update conflict, got: %v", err)
	assert.Greater(t, conflictWorkflowClient.updateCalls, 1)
	assert.Equal(t, 0, conflictWorkflowClient.createCalls)
}

func TestRetryRun_RetriesWorkflowUpdateAfterCreateAlreadyExists(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := client.NewWorkflowClientFake()
	seedRetryWorkflow(t, manager, runDetail.UUID, workflowClient)
	alreadyExistsWorkflowClient := &createAlreadyExistsWorkflowClient{
		FakeWorkflowClient:           workflowClient,
		updateNotFoundRemaining:      1,
		createAlreadyExistsRemaining: 1,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: alreadyExistsWorkflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.NoError(t, err)
	assert.Equal(t, 0, alreadyExistsWorkflowClient.updateNotFoundRemaining)
	assert.Equal(t, 0, alreadyExistsWorkflowClient.createAlreadyExistsRemaining)
}

func TestRetryRun_CreatesWorkflowAfterRetryableGetFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := &retryableGetFailureWorkflowClient{
		FakeWorkflowClient:   client.NewWorkflowClientFake(),
		getFailuresRemaining: 10,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.NoError(t, err)
	assert.Equal(t, 9, workflowClient.getFailuresRemaining)
	assert.Equal(t, 1, workflowClient.createCalls)
}

func TestRetryRun_RetriesRetryableWorkflowUpdateFailure(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := client.NewWorkflowClientFake()
	seedRetryWorkflow(t, manager, runDetail.UUID, workflowClient)
	retryableFailureWorkflowClient := &retryableUpdateFailureWorkflowClient{
		FakeWorkflowClient:      workflowClient,
		updateFailuresRemaining: 1,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: retryableFailureWorkflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.NoError(t, err)
	assert.Equal(t, 0, retryableFailureWorkflowClient.updateFailuresRemaining)
	assert.Equal(t, 0, retryableFailureWorkflowClient.createCalls)
}

func TestRetryRun_GenericWorkflowUpdateFailureDoesNotCreate(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	workflowClient := client.NewWorkflowClientFake()
	seedRetryWorkflow(t, manager, runDetail.UUID, workflowClient)
	genericFailureWorkflowClient := &genericUpdateFailureWorkflowClient{
		FakeWorkflowClient: workflowClient,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: genericFailureWorkflowClient}

	err := manager.RetryRun(context.Background(), runDetail.UUID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error updating workflow")
	assert.Equal(t, 0, genericFailureWorkflowClient.createCalls)
}

func TestRetryRun_PreservesRunFields(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	originalRun, err := manager.GetRun(runDetail.UUID)
	require.Nil(t, err)
	assert.Equal(t, string(v1alpha1.WorkflowFailed), string(originalRun.Conditions))

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	require.Nil(t, err)

	retriedRun, err := manager.GetRun(runDetail.UUID)
	require.Nil(t, err)

	// Core identifying fields must be preserved after retry
	assert.Equal(t, originalRun.UUID, retriedRun.UUID)
	assert.Equal(t, originalRun.DisplayName, retriedRun.DisplayName)
	assert.Equal(t, originalRun.ExperimentId, retriedRun.ExperimentId)
	// FinishedAtInSec must be reset to 0 on retry
	assert.Equal(t, int64(0), retriedRun.FinishedAtInSec)
	// State must be updated to Running
	assert.Equal(t, model.RuntimeStateRunning, retriedRun.State)

	// StateHistory must preserve pre-retry entries and append a new RUNNING entry.
	// With the old sparse-struct UpdateRun call, StateHistory would be overwritten
	// to a single entry; passing the full run object preserves history.
	originalHistoryLen := len(originalRun.StateHistory)
	assert.Greater(t, originalHistoryLen, 0)
	require.Greater(t, len(retriedRun.StateHistory), originalHistoryLen)
	lastEntry := retriedRun.StateHistory[len(retriedRun.StateHistory)-1]
	assert.Equal(t, model.RuntimeStateRunning, lastEntry.State)
}

func TestRetryRun_ReopensMLflowParentAndFailedNestedRuns(t *testing.T) {
	type updateCall struct {
		RunID  string
		Status string
	}
	var updateCalls []updateCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			// Return experiment for initial run creation
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"exp-1","name":"Default"}}`))
		case "/api/2.0/mlflow/runs/create":
			// Return a temporary parent run ID for initial run creation
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"temp-parent-run"}}}`))
		case "/api/2.0/mlflow/runs/update":
			defer r.Body.Close()
			var payload struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			updateCalls = append(updateCalls, updateCall{RunID: payload.RunID, Status: payload.Status})
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			body, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if strings.Contains(string(body), "parent-run-1") {
				_, _ = w.Write([]byte(`{
				"runs": [
					{"info":{"run_id":"nested-failed","status":"FAILED"}},
					{"info":{"run_id":"nested-killed","status":"KILLED"}},
					{"info":{"run_id":"nested-finished","status":"FINISHED"}}
				]
			}`))
			} else {
				_, _ = w.Write([]byte(`{"runs":[]}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setupTestSAToken(t, "retry-token")
	setupMLflowViperConfig(t, server.URL)

	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	runWithPluginOutput, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	mlflowOutput := apiservermlflow.SuccessfulPluginOutput("exp-1", "exp-1", "parent-run-1", server.URL+"/runs/parent-run-1")
	lt, err := apiserverPlugins.SerializePluginsOutput(map[string]*apiv2beta1.PluginOutput{apiservermlflow.PluginName: mlflowOutput})
	require.NoError(t, err)
	runWithPluginOutput.PluginsOutputString = lt
	require.NoError(t, manager.runStore.UpdateRun(runWithPluginOutput))

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	require.NoError(t, err)

	assert.Contains(t, updateCalls, updateCall{RunID: "parent-run-1", Status: "RUNNING"})
	assert.Contains(t, updateCalls, updateCall{RunID: "nested-failed", Status: "RUNNING"})
	assert.Contains(t, updateCalls, updateCall{RunID: "nested-killed", Status: "RUNNING"})
	assert.NotContains(t, updateCalls, updateCall{RunID: "nested-finished", Status: "RUNNING"})

	updatedRun, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	updatedOutputs, err := apiserverPlugins.DeserializePluginsOutput(updatedRun.PluginsOutputString)
	require.NoError(t, err)
	updatedOutput := updatedOutputs["mlflow"]
	require.NotNil(t, updatedOutput)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, updatedOutput.State)
	assert.Equal(t, "", updatedOutput.StateMessage)
}

func TestRetryRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.RetryRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryRun_FailedDeletePods(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestRetryRun_FailedDeletePodsCompressed(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRunCompressed(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestRetryRun_FailedOffloadNodeStatus(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRunOffloaded(t)
	defer store.Close()

	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Cannot retry workflow with offloaded node status")
}

func TestRetryRun_UpdateAndCreateFailed(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.execClient = client.NewFakeExecClientWithBadWorkflow()
	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error getting workflow")
}

func TestRetryRun_Failed_RunArchived(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	err := manager.ArchiveRun(runDetail.UUID)
	assert.Nil(t, err)
	before, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	userError := err.(*util.UserError)
	assert.Equal(t, codes.FailedPrecondition, userError.ExternalStatusCode())
	assert.Equal(t,
		fmt.Sprintf("Failed to retry run %s as it is archived. Unarchive the run first to allow it to be retried", runDetail.UUID),
		userError.ExternalMessage())

	after, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, model.StorageStateArchived, after.StorageState.ToV2())
	assert.Equal(t, before.State, after.State)
	assert.Equal(t, before.RetryGeneration, after.RetryGeneration)
	assert.Equal(t, before.RetryClaimedAtInSec, after.RetryClaimedAtInSec)
	assert.Equal(t, before.WorkflowRuntimeManifest, after.WorkflowRuntimeManifest)
}

func TestRetryRun_UnarchivedRunStillRetries(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	err := manager.ArchiveRun(runDetail.UUID)
	assert.Nil(t, err)
	err = manager.UnarchiveRun(runDetail.UUID)
	assert.Nil(t, err)

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	assert.Nil(t, err)
}

// archiveOnClaimRunStore archives the run inside the window RetryRun leaves
// between its GetRun pre-check and the claim taking the row lock, which is what
// ArchiveExpiredRuns does when it commits concurrently.
type archiveOnClaimRunStore struct {
	storage.RunStoreInterface
	runID string
}

func (s *archiveOnClaimRunStore) ClaimRunForRetry(runID string, takeoverExpiredClaim bool) (string, string, int64, int64, error) {
	if runID == s.runID {
		if err := s.RunStoreInterface.ArchiveRun(runID); err != nil {
			return "", "", 0, 0, err
		}
	}
	return s.RunStoreInterface.ClaimRunForRetry(runID, takeoverExpiredClaim)
}

func TestRetryRun_Failed_RunArchivedDuringClaim(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	manager.runStore = &archiveOnClaimRunStore{RunStoreInterface: manager.runStore, runID: runDetail.UUID}

	err := manager.RetryRun(context.Background(), runDetail.UUID)
	assert.NotNil(t, err)
	userError := err.(*util.UserError)
	assert.Equal(t, codes.FailedPrecondition, userError.ExternalStatusCode())
	assert.Equal(t,
		fmt.Sprintf("Failed to retry run %s as it is archived. Unarchive the run first to allow it to be retried", runDetail.UUID),
		userError.ExternalMessage())

	after, err := manager.GetRun(runDetail.UUID)
	assert.Nil(t, err)
	assert.Equal(t, model.StorageStateArchived, after.StorageState.ToV2())
	assert.Equal(t, model.RuntimeStateFailed, after.State)
	assert.Equal(t, int64(0), after.RetryGeneration)
	assert.Equal(t, int64(0), after.RetryClaimedAtInSec)
}

// The node id in a log request is caller controlled, so a run must not act as
// a namespace selector for arbitrary pods: a pod whose run id label does not
// match the requested run is rejected before any logs are streamed.
func TestReadRunLogFromPod_PodNotFromRun(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	foreignPod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "victim-pod",
			Namespace: "ns1",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "some-other-run"},
		},
	}
	manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithPod(foreignPod)

	var buf bytes.Buffer
	err := manager.readRunLogFromPod(context.Background(), "run-1", "ns1", "victim-pod", false, &buf)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "does not belong to run")
	assert.Empty(t, buf.String())
}

func TestUnarchiveRun_OK(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.UnarchiveRun(runDetail.UUID)
	assert.Nil(t, err)
}

func TestUnarchiveRun_Failed_ExperimentArchived(t *testing.T) {
	store, manager, runDetail := initWithOneTimeRun(t)
	defer store.Close()
	err := manager.ArchiveExperiment(context.Background(), runDetail.ExperimentId)
	assert.Nil(t, err)
	err = manager.UnarchiveRun(runDetail.UUID)
	assert.NotNil(t, err)
	assert.Equal(t, codes.FailedPrecondition, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unarchive the experiment first to allow")
}

func TestUnarchiveRun_Failed_ResourceNotFound(t *testing.T) {
	store, manager, _ := initWithExperiment(t)
	defer store.Close()
	err := manager.UnarchiveRun(FakeUUIDOne)
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateJob_BlocksV1Pipelines(t *testing.T) {
	tt := []struct {
		msg               string
		blockV1           bool
		allowedNamespaces string
		namespace         string
		useV2Spec         bool
		errorCode         codes.Code
		errorMsg          string
	}{
		{
			msg:               "BlockV1_NamespaceNotAllowed",
			blockV1:           true,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         false,
			errorCode:         codes.InvalidArgument,
			errorMsg:          "not allowed to run v1 pipelines",
		},
		{
			msg:               "BlockV1_NamespaceAllowed",
			blockV1:           true,
			allowedNamespaces: "ns1",
			namespace:         "ns1",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_NamespaceAllowed_MultipleNamespaces",
			blockV1:           true,
			allowedNamespaces: "ns1,ns2,ns3",
			namespace:         "ns2",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_Disabled_AnyNamespaceAllowed",
			blockV1:           false,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         false,
		},
		{
			msg:               "BlockV1_V2PipelineNotBlocked",
			blockV1:           true,
			allowedNamespaces: "",
			namespace:         "ns1",
			useV2Spec:         true,
		},
		{
			msg:               "BlockV1_NamespaceNotInAllowedList",
			blockV1:           true,
			allowedNamespaces: "ns2,ns3",
			namespace:         "ns1",
			useV2Spec:         false,
			errorCode:         codes.InvalidArgument,
			errorMsg:          "Namespace ns1 is not allowed to run v1 pipelines",
		},
		{
			msg:               "BlockV1_CaseInsensitiveNamespaceMatch",
			blockV1:           true,
			allowedNamespaces: "NS1",
			namespace:         "ns1",
			useV2Spec:         false,
		},
	}

	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			viper.Set(util.BlockV1Pipelines, test.blockV1)
			viper.Set(v1AllowedNamespaces, test.allowedNamespaces)
			defer func() {
				viper.Set(util.BlockV1Pipelines, nil)
				viper.Set(v1AllowedNamespaces, nil)
			}()

			store, manager, exp := initWithExperiment(t)
			defer store.Close()

			job := &model.Job{
				DisplayName:  "j1",
				Enabled:      true,
				ExperimentId: exp.UUID,
				Namespace:    test.namespace,
			}
			if test.useV2Spec {
				job.PipelineSpec = model.PipelineSpec{
					PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
					RuntimeConfig: model.RuntimeConfig{
						Parameters:   "{\"text\":\"world\"}",
						PipelineRoot: "job-1-root",
					},
				}
			} else {
				job.PipelineSpec = model.PipelineSpec{
					WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
				}
			}

			_, err := manager.CreateJob(context.Background(), job)

			if test.errorCode != 0 {
				require.NotNil(t, err)
				assert.Equal(t, test.errorCode, err.(*util.UserError).ExternalStatusCode())
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
				return
			}
			assert.Nil(t, err)
		})
	}
}

// TODO Use table driven to write UT to test CreateJob
func TestCreateJob_ThroughWorkflowSpec(t *testing.T) {
	store, _, job := initWithJob(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		ExperimentId:   DefaultFakeUUID,
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
	}
	expectedJob.PipelineSpec.PipelineName = job.PipelineSpec.PipelineName
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
}

func TestCreateJob_ThroughWorkflowSpecV2(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		ExperimentId:   DefaultFakeUUID,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
	}
	expectedJob.PipelineSpec.PipelineName = job.PipelineSpec.PipelineName
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
	fetchedJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), fetchedJob.ToV1(), "CreateJob stored invalid data in database")
}

func TestCreateJobDifferentDefaultServiceAccountName_ThroughWorkflowSpecV2(t *testing.T) {
	originalDefaultServiceAccount := viper.Get(common.DefaultPipelineRunnerServiceAccountFlag)

	viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "my-service-account")
	defer viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, originalDefaultServiceAccount)

	store, manager, job := initWithJobV2(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "my-service-account",
		Enabled:        true,
		ExperimentId:   DefaultFakeUUID,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
	}
	expectedJob.PipelineSpec.PipelineName = job.PipelineSpec.PipelineName
	require.Equal(t, expectedJob.ToV1(), job.ToV1())
	fetchedJob, err := manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.Equal(t, expectedJob.ToV1(), fetchedJob.ToV1(), "CreateJob stored invalid data in database")
}

func TestCreateJob_ThroughPipelineID(t *testing.T) {
	store, manager, pipeline, _ := initWithPipeline(t)
	defer store.Close()
	apiExperiment := &model.Experiment{Name: "e1"}
	experiment, _ := manager.CreateExperiment(apiExperiment)
	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}

	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:        "123e4567-e89b-12d3-a456-426655440000",
		DisplayName: "j1",
		K8SName:     "job-",
		Namespace:   "ns1",
		// Since there is no pipeline version or service account specified, the API server will select the service
		// account when compiling the run, not within the ScheduledWorkflow.
		ServiceAccount: "",
		Enabled:        true,
		CreatedAtInSec: 4,
		UpdatedAtInSec: 4,
		Conditions:     "STATUS_UNSPECIFIED",
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_ThroughPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_job",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId: version.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 5,
		UpdatedAtInSec: 5,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           version.PipelineId,
			PipelineName:         version.Name,
			PipelineVersionId:    version.UUID,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_ThroughPipelineIdAndPipelineVersion(t *testing.T) {
	// Create experiment, pipeline and pipeline version.
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	pv := createPipelineVersion(
		pipeline.UUID,
		"version_for_job",
		"",
		"",
		testWorkflow.ToStringForStore(),
		"",
		"",
	)
	version, err := manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	job := &model.Job{
		DisplayName:  "j1",
		Enabled:      true,
		ExperimentId: experiment.UUID,

		PipelineSpec: model.PipelineSpec{
			PipelineId:        pipeline.UUID,
			Parameters:        "[{\"name\":\"param1\",\"value\":\"world\"}]",
			PipelineVersionId: version.UUID,
		},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 5,
		UpdatedAtInSec: 5,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   experiment.UUID,

		PipelineSpec: model.PipelineSpec{
			PipelineName:         version.Name,
			PipelineId:           pipeline.UUID,
			PipelineVersionId:    version.UUID,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), newJob.ToV1())
}

func TestCreateJob_EmptyPipelineSpec(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		DisplayName:  "pp 1",
		Enabled:      true,
		ExperimentId: experimentID,
		PipelineSpec: model.PipelineSpec{
			Parameters: "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	assert.Contains(t, errMsg, "Cannot create a job with an empty pipeline ID")
}

func TestCreateJob_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText("I am invalid"),
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateJob_NullWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText("null"), // this situation occurs for real when the manifest file disappears from object store in some way due to retention policy or manual deletion.
			Parameters:           "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateJob_ExtraInputParameterError(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp 1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			Parameters: "[{\"name\":\"param2\",\"value\":\"world\"}]",
		},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unrecognized input parameter: param2")
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store, manager, p, _ := initWithPipeline(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	experimentID, _ := manager.CreateDefaultExperiment("")
	job := &model.Job{
		K8SName:      "pp1",
		ExperimentId: experimentID,
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{PipelineId: p.UUID},
	}
	_, err := manager.CreateJob(context.Background(), job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a recurring run during scheduling a workflow")
}

func TestEnableJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.ChangeJobMode(context.Background(), job.UUID, false)
	job, err := manager.GetJob(job.UUID)
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		K8SName:        "job-",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
		Conditions:     "STATUS_UNSPECIFIED",
		ExperimentId:   DefaultFakeUUID,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           job.PipelineSpec.PipelineId,
			PipelineName:         job.PipelineSpec.PipelineName,
			PipelineVersionId:    job.PipelineSpec.PipelineVersionId,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob.ToV1(), job.ToV1())
}

func TestEnableJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.ChangeJobMode(context.Background(), "1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.ChangeJobMode(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
}

func TestEnableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})
	// When swf CR is missing, enabling the job needs to fail.
	err := manager.ChangeJobMode(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
	assert.Contains(t, err.Error(), "not found")
}

func TestDisableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	require.Equal(t, job.Enabled, true)

	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})
	err := manager.ChangeJobMode(context.Background(), job.UUID, false)
	require.Nil(t, err, "Disabling the job should succeed even when the custom resource is missing")
	job, err = manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.Equal(t, job.Enabled, false)
}

func TestEnableJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.DB().Close()
	err := manager.ChangeJobMode(context.Background(), job.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_WithForegroundPolicy(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_FOREGROUND)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_WithBackgroundPolicy(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_BACKGROUND)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_WithOrphanPolicy(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_ORPHAN)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.DeleteJob(context.Background(), "1", apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestDeleteJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check if the scheduled workflow exists")
}

func TestDeleteJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.K8SName, &v1.DeleteOptions{})

	// Now deleting job should still succeed when the swf CR is already deleted.
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	assert.Nil(t, err)

	// And verify Job has been deleted from DB too.
	_, err = manager.GetJob(job.UUID)
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteJob(context.Background(), job.UUID, apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestReportWorkflowResource_ScheduledWorkflowIDEmpty_Success(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	expectedExperimentUUID := run.ExperimentId
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			Namespace: "ns1",
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)
	run, err = manager.GetRun(run.UUID)
	assert.Nil(t, err)
	expectedRun := &model.Run{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   expectedExperimentUUID,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Running",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStatePending,
				},
				{
					UpdateTimeInSec: 4,
					State:           model.RuntimeStateRunning,
				},
			},
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
	}
	expectedRun.PipelineSpec.PipelineName = run.PipelineSpec.PipelineName
	expectedRun.RunDetails.WorkflowRuntimeManifest = run.RunDetails.WorkflowRuntimeManifest
	assert.Equal(t, expectedRun.ToV1(), run.ToV1())
}

type runStoreWithBeforeWorkflowUpdateHook struct {
	storage.RunStoreInterface
	beforeUpdate func()
}

func (s *runStoreWithBeforeWorkflowUpdateHook) UpdateRunFromWorkflow(
	run *model.Run,
	expectedState model.RuntimeState,
	expectedWorkflowRuntimeManifest model.LargeText,
	expectedPipelineRuntimeManifest model.LargeText,
) (bool, error) {
	if s.beforeUpdate != nil {
		beforeUpdate := s.beforeUpdate
		s.beforeUpdate = nil
		beforeUpdate()
	}
	return s.RunStoreInterface.UpdateRunFromWorkflow(
		run,
		expectedState,
		expectedWorkflowRuntimeManifest,
		expectedPipelineRuntimeManifest,
	)
}

func TestReportWorkflowResource_DoesNotOverwriteConcurrentTermination(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	liveWorkflow, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	beforeTermination, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	runStore := manager.runStore.(*storage.RunStore)
	manager.runStore = &runStoreWithBeforeWorkflowUpdateHook{
		RunStoreInterface: runStore,
		beforeUpdate: func() {
			require.NoError(t, runStore.TerminateRun(run.UUID))
		},
	}

	_, err = manager.ReportWorkflowResource(ctx, liveWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable))
	assert.Contains(t, err.Error(), "stored run changed concurrently")

	persistedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateCancelling, persistedRun.State)
	assert.Equal(t, "Terminating", persistedRun.Conditions)
	assert.Equal(t, beforeTermination.WorkflowRuntimeManifest, persistedRun.WorkflowRuntimeManifest)
	assert.Equal(t, beforeTermination.StateHistory, persistedRun.StateHistory)

	// Model a retry after the first request committed CANCELING but exited
	// before patching the Workflow. The repeated termination must finish it.
	require.NoError(t, manager.TerminateRun(ctx, run.UUID))
	isTerminated, err := store.ExecClientFake.IsTerminatedInNamespace(run.Namespace, run.K8SName)
	require.NoError(t, err)
	assert.True(t, isTerminated)
}

func TestReportWorkflowResource_DoesNotOverwriteExistingCancellation(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	liveWorkflow, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	beforeTermination, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.NoError(t, manager.runStore.TerminateRun(run.UUID))

	_, err = manager.ReportWorkflowResource(ctx, liveWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable))
	assert.Contains(t, err.Error(), "stored run changed concurrently")

	persistedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateCancelling, persistedRun.State)
	assert.Equal(t, "Terminating", persistedRun.Conditions)
	assert.Equal(t, beforeTermination.WorkflowRuntimeManifest, persistedRun.WorkflowRuntimeManifest)
	assert.Equal(t, beforeTermination.StateHistory, persistedRun.StateHistory)
}

func TestReportWorkflowResource_UsesLiveWorkflowState(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	_, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	forgedReport := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	forgedReport.Status.Phase = v1alpha1.WorkflowFailed
	_, err = manager.ReportWorkflowResource(ctx, forgedReport)
	require.NoError(t, err)

	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, updatedRun.State,
		"request-controlled status must not override the live Workflow")
}

func TestReportWorkflowResource_AdoptsLegacyEmptyWorkflowName(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	_, err := store.DB().Exec(`UPDATE run_details SET Name = '' WHERE UUID = ?`, run.UUID)
	require.NoError(t, err)
	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	_, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, liveWorkflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, run.K8SName, updatedRun.K8SName)
}

func TestReportWorkflowResource_RepairsDivergentWorkflowName(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	_, err := store.DB().Exec(`UPDATE run_details SET Name = ? WHERE UUID = ?`, "stale-workflow-name", run.UUID)
	require.NoError(t, err)
	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	liveWorkflow, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, liveWorkflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, run.K8SName, updatedRun.K8SName)
	assert.Equal(t, model.RuntimeStateRunning, updatedRun.State)
}

func TestReportWorkflowResource_NamespaceMismatch_Rejected(t *testing.T) {
	store, manager, exp := initWithExperiment(t)
	defer store.Close()
	apiRun := &model.Run{
		DisplayName: "run1",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
	}
	run, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	// The run must have a namespace for the cross-namespace guard to apply.
	run, err = manager.GetRun(run.UUID)
	assert.Nil(t, err)
	assert.NotEmpty(t, run.Namespace)
	runBeforeReport := run.ToV1()

	// A workflow reported from a different namespace must not be allowed to
	// overwrite this run, even though it carries the run's ID label.
	spoofed := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: run.K8SName,
			UID:  types.UID(run.UUID),
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
			Namespace: "attacker-ns",
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err = manager.ReportWorkflowResource(context.Background(), spoofed)
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported namespace does not match owning resource", err.(*util.UserError).ExternalMessage())
	assert.NotContains(t, err.Error(), run.Namespace)
	runAfterReport, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, runBeforeReport, runAfterReport.ToV1())

	// A workflow reported from the run's own namespace still succeeds.
	legit := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			Namespace: run.Namespace,
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, legit)
	_, err = manager.ReportWorkflowResource(context.Background(), legit)
	assert.Nil(t, err)
}

func TestReportWorkflowResource_PersistedRunNamespaceInMultiUserMode(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	experiment, err := manager.GetExperiment(run.ExperimentId)
	require.NoError(t, err)
	assert.Equal(t, experiment.Namespace, run.Namespace)
	persistedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, experiment.Namespace, persistedRun.Namespace)
	require.Equal(t, "ns1", experiment.Namespace)
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	spoofed := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			Namespace: "attacker-ns",
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})

	_, err = manager.ReportWorkflowResource(context.Background(), spoofed)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported namespace does not match owning resource", err.(*util.UserError).ExternalMessage())
	assert.NotContains(t, err.Error(), "ns1")

	legitimate := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: experiment.Namespace,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, legitimate)
	_, err = manager.ReportWorkflowResource(context.Background(), legitimate)
	require.NoError(t, err)
}

func TestReportWorkflowResource_NoNamespaceRunUsesStoredWorkflowNamespaceInSingleUserMode(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// Simulate a legacy row which used the namespace sentinel before the
	// Kubernetes execution namespace was persisted on runs.
	_, err := store.DB().Exec(`UPDATE run_details SET Namespace = ? WHERE UUID = ?`, model.NoNamespace, run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)
	// The API server may have moved since this legacy workflow was submitted.
	// The created runtime manifest remains the authoritative mapping and avoids
	// both stranding the run and accepting a replacement in another namespace.
	previousPodNamespace := viper.GetString(common.PodNamespace)
	viper.Set(common.PodNamespace, "new-api-server-namespace")
	t.Cleanup(func() { viper.Set(common.PodNamespace, previousPodNamespace) })

	workflow, err := store.ExecClient().Execution(run.Namespace).Get(
		context.Background(), run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	workflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	workflow, err = store.ExecClient().Execution(run.Namespace).Update(
		context.Background(), workflow, v1.UpdateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, run.Namespace, updatedRun.Namespace)
}

func TestReportWorkflowResource_NoNamespaceRunAdoptsMissingStoredUID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	legacyManifest := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	legacyManifest.UID = ""
	legacyManifest.Namespace = ""
	_, err = store.DB().Exec(
		`UPDATE run_details SET Namespace = ?, WorkflowRuntimeManifest = ? WHERE UUID = ?`,
		model.NoNamespace, legacyManifest.ToStringForStore(), run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, liveWorkflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, run.Namespace, updatedRun.Namespace)
	assert.Equal(t, liveWorkflow.ExecutionObjectMeta().UID, storedWorkflowUID(t, updatedRun))
}

func TestReportWorkflowResource_RefreshesIdentityCacheAfterAnotherReplicaRepairsLegacyRun(t *testing.T) {
	store, firstManager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	secondManager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	liveWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	legacyManifest := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	legacyManifest.UID = ""
	legacyManifest.Namespace = ""
	_, err = store.DB().Exec(
		`UPDATE run_details SET Namespace = ?, WorkflowRuntimeManifest = ? WHERE UUID = ?`,
		model.NoNamespace, legacyManifest.ToStringForStore(), run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	legacyRun, err := firstManager.GetRun(run.UUID)
	require.NoError(t, err)
	legacyIdentity, err := firstManager.storedWorkflowIdentityForRun(legacyRun)
	require.NoError(t, err)
	require.Empty(t, legacyIdentity.uid)

	_, err = secondManager.ReportWorkflowResource(ctx, liveWorkflow)
	require.NoError(t, err, "a second replica should repair the persisted workflow identity")
	repairedRun, err := firstManager.GetRun(run.UUID)
	require.NoError(t, err)
	require.Equal(t, run.Namespace, repairedRun.Namespace)
	require.Equal(t, liveWorkflow.ExecutionObjectMeta().UID, storedWorkflowUID(t, repairedRun))

	_, err = firstManager.ReportWorkflowResource(ctx, liveWorkflow)
	require.NoError(t, err, "the first replica must refresh its cache from the repaired manifest")
	refreshedIdentity, found := firstManager.storedWorkflowIdentities.load(run.UUID)
	require.True(t, found)
	assert.Equal(t, liveWorkflow.ExecutionObjectMeta().UID, refreshedIdentity.uid)
	assert.NotEqual(t, legacyIdentity.manifestDigest, refreshedIdentity.manifestDigest)
}

func TestReportWorkflowResource_RejectsStaleLegacyIdentityAfterConcurrentRepair(t *testing.T) {
	store, staleManager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	repairManager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	workflowClient := store.ExecClient().Execution(run.Namespace)
	original, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	legacyManifest := util.NewWorkflow(original.(*util.Workflow).DeepCopy())
	legacyManifest.UID = ""
	legacyManifest.Namespace = ""
	_, err = store.DB().Exec(
		`UPDATE run_details SET Namespace = ?, WorkflowRuntimeManifest = ? WHERE UUID = ?`,
		model.NoNamespace, legacyManifest.ToStringForStore(), run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	staleRun, err := staleManager.GetRun(run.UUID)
	require.NoError(t, err)
	require.Empty(t, storedWorkflowUID(t, staleRun))

	_, err = repairManager.ReportWorkflowResource(ctx, original)
	require.NoError(t, err)
	repairedRun, err := staleManager.GetRun(run.UUID)
	require.NoError(t, err)
	originalUID := original.ExecutionObjectMeta().UID
	require.Equal(t, originalUID, storedWorkflowUID(t, repairedRun))
	repairedIdentity, err := staleManager.storedWorkflowIdentityForRun(repairedRun)
	require.NoError(t, err)
	require.Equal(t, originalUID, repairedIdentity.uid)

	require.NoError(t, workflowClient.Delete(ctx, run.K8SName, v1.DeleteOptions{}))
	replacement, err := workflowClient.Create(ctx, util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	}), v1.CreateOptions{})
	require.NoError(t, err)
	require.NotEqual(t, originalUID, replacement.ExecutionObjectMeta().UID)

	_, err = staleManager.ReportWorkflowResourceWithRun(ctx, replacement, staleRun)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable), "got %v", err)
	afterConflict, err := staleManager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, originalUID, storedWorkflowUID(t, afterConflict))
	refreshedIdentity, found := staleManager.storedWorkflowIdentities.load(run.UUID)
	require.True(t, found)
	assert.Equal(t, originalUID, refreshedIdentity.uid)

	_, err = staleManager.ReportWorkflowResource(ctx, replacement)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "got %v", err)
	unchangedRun, err := staleManager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, originalUID, storedWorkflowUID(t, unchangedRun))
}

func TestReportWorkflowResource_RefreshesIdentityCacheAfterOrdinaryUpdate(t *testing.T) {
	store, manager, run := initWithOneTimeRunV2(t)
	defer store.Close()
	ctx := context.Background()
	workflowClient := store.ExecClient().Execution(run.Namespace)
	liveWorkflow, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	storedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.Empty(t, storedRun.WorkflowRuntimeManifest)
	require.NotEmpty(t, storedRun.PipelineRuntimeManifest)
	initialIdentity, err := manager.storedWorkflowIdentityForRun(storedRun)
	require.NoError(t, err)

	if liveWorkflow.ExecutionObjectMeta().Annotations == nil {
		liveWorkflow.ExecutionObjectMeta().Annotations = map[string]string{}
	}
	liveWorkflow.ExecutionObjectMeta().Annotations["cache-refresh"] = "ordinary-report"
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	updatedWorkflow, err := workflowClient.Update(ctx, liveWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)
	_, err = manager.ReportWorkflowResource(ctx, updatedWorkflow)
	require.NoError(t, err)

	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedRun.WorkflowRuntimeManifest)
	updatedIdentity, found := manager.storedWorkflowIdentities.load(run.UUID)
	require.True(t, found)
	assert.Equal(t, sha256.Sum256([]byte(updatedRun.WorkflowRuntimeManifest)), updatedIdentity.manifestDigest)
	assert.NotEqual(t, initialIdentity.manifestDigest, updatedIdentity.manifestDigest)
	assert.Equal(t, updatedWorkflow.ExecutionName(), updatedIdentity.name)
	assert.Equal(t, updatedWorkflow.ExecutionNamespace(), updatedIdentity.namespace)
	assert.Equal(t, updatedWorkflow.ExecutionObjectMeta().UID, updatedIdentity.uid)
	assert.Equal(t, updatedRun.RetryGeneration, updatedIdentity.retryGeneration)
}

func TestReportWorkflowResource_RejectsStaleReportAfterRunIDRecreation(t *testing.T) {
	store, manager, originalRun := initWithOneTimeRunV2(t)
	defer store.Close()
	ctx := context.Background()
	workflowClient := store.ExecClient().Execution(originalRun.Namespace)
	originalLiveWorkflow, err := workflowClient.Get(ctx, originalRun.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	staleReport := util.NewWorkflow(originalLiveWorkflow.(*util.Workflow).DeepCopy())
	if staleReport.Labels == nil {
		staleReport.Labels = map[string]string{}
	}
	staleReport.Labels[util.LabelKeyWorkflowPersistedFinalState] = "true"
	staleReport.Status.Phase = v1alpha1.WorkflowSucceeded
	staleReport.Status.FinishedAt = v1.NewTime(time.Unix(123, 0))

	staleRun, err := manager.GetRun(originalRun.UUID)
	require.NoError(t, err)
	require.Empty(t, staleRun.WorkflowRuntimeManifest)
	require.NotEmpty(t, staleRun.PipelineRuntimeManifest)
	require.NoError(t, manager.DeleteRun(ctx, originalRun.UUID))

	replacementRun, err := manager.CreateRun(ctx, &model.Run{
		UUID:         originalRun.UUID,
		DisplayName:  originalRun.DisplayName,
		ExperimentId: originalRun.ExperimentId,
		Namespace:    originalRun.Namespace,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
	})
	require.NoError(t, err)
	require.Empty(t, replacementRun.WorkflowRuntimeManifest)
	require.NotEmpty(t, replacementRun.PipelineRuntimeManifest)
	require.NotEqual(t, staleRun.PipelineRuntimeManifest, replacementRun.PipelineRuntimeManifest)
	require.NotEqual(t, storedWorkflowUID(t, staleRun), storedWorkflowUID(t, replacementRun))
	replacementBeforeReport, err := manager.GetRun(replacementRun.UUID)
	require.NoError(t, err)
	replacementLiveWorkflow, err := workflowClient.Get(ctx, replacementRun.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResourceWithRun(ctx, staleReport, staleRun)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable), "got %v", err)

	replacementAfterReport, err := manager.GetRun(replacementRun.UUID)
	require.NoError(t, err)
	assert.Equal(t, replacementBeforeReport, replacementAfterReport)
	stillLive, err := workflowClient.Get(ctx, replacementRun.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, replacementLiveWorkflow.ExecutionObjectMeta().UID, stillLive.ExecutionObjectMeta().UID)
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(
		replacementRun.Namespace,
		replacementRun.K8SName,
	))
}

func TestReportWorkflowResource_NoNamespaceRunRejectsSameNameReplacementUID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	_, err := store.DB().Exec(`UPDATE run_details SET Namespace = ? WHERE UUID = ?`, model.NoNamespace, run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	workflowClient := store.ExecClient().Execution(run.Namespace)
	original, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, workflowClient.Delete(ctx, run.K8SName, v1.DeleteOptions{}))
	replacement, err := workflowClient.Create(ctx, util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	}), v1.CreateOptions{})
	require.NoError(t, err)
	require.NotEqual(t, original.ExecutionObjectMeta().UID, replacement.ExecutionObjectMeta().UID)

	_, err = manager.ReportWorkflowResource(ctx, replacement)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported identity does not match the stored workflow", err.(*util.UserError).ExternalMessage())
}

func TestReportWorkflowResource_NoNamespaceRunRejectsDifferentWorkflowNamespace(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	_, err := store.DB().Exec(`UPDATE run_details SET Namespace = ? WHERE UUID = ?`, model.NoNamespace, run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	replacement := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "replacement-namespace",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, replacement)

	_, err = manager.ReportWorkflowResource(context.Background(), replacement)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported namespace does not match owning resource", err.(*util.UserError).ExternalMessage())
}

func TestGetNamespaceFromRunID_PrefersPersistedRunNamespace(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	_, err := store.DB().Exec(`UPDATE run_details SET Namespace = ? WHERE UUID = ?`, "run-namespace", run.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		run.UUID,
		model.RunResourceType,
		model.NamespaceResourceType,
	)
	require.NoError(t, err)

	namespace, err := manager.getNamespaceFromRunId(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, "run-namespace", namespace)
}

func TestValidateWorkflowReportNamespace_EmptyNamespaceFailsPermanently(t *testing.T) {
	manager := &ResourceManager{}
	err := manager.validateWorkflowReportNamespace("recurring run", "job-id", "", "attacker-ns", "workflow")
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: owning namespace cannot be determined", err.(*util.UserError).ExternalMessage())
}

func TestResolveWorkflowReportNamespace_MultiUserEmptyExperimentIsRetryable(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	store, manager, _ := initWithExperiment(t)
	defer store.Close()

	_, err := manager.resolveWorkflowReportNamespace("run", "run-id", "", "", "attacker-ns")
	require.Error(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to determine namespace")

	_, err = manager.resolveWorkflowReportNamespace("run", "run-id", "", "missing-experiment", "attacker-ns")
	require.Error(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestReportWorkflowResource_ScheduledWorkflowIDNotEmpty_Success(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: job.Namespace,
			UID:       "WORKFLOW_1",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "WORKFLOW_1"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
		},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("WORKFLOW_1")
	assert.Nil(t, err)

	expectedRunDetail := &model.Run{
		UUID:           "WORKFLOW_1",
		ExperimentId:   job.ExperimentId,
		DisplayName:    "MY_NAME",
		StorageState:   model.StorageStateAvailable,
		K8SName:        "MY_NAME",
		Namespace:      job.Namespace,
		RecurringRunId: job.UUID,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(workflow.GetExecutionSpec().ToStringForStore()),
			PipelineSpecManifest: job.PipelineSpecManifest,
			PipelineId:           job.PipelineId,
			PipelineName:         job.PipelineName,
			PipelineVersionId:    job.PipelineVersionId,
		},
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: model.LargeText(workflow.ToStringForStore()),
			CreatedAtInSec:          11,
			ScheduledAtInSec:        11,
			FinishedAtInSec:         0,
			Conditions:              "Error",
			State:                   model.RuntimeStateUnspecified,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 3,
					State:           model.RuntimeStateUnspecified,
				},
			},
		},
	}
	assert.Equal(t, expectedRunDetail.ToV1(), runDetail.ToV1())
}

func TestReportWorkflowResource_ScheduledWorkflowNamespaceMismatch_Rejected(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "spoofed-run",
			Namespace: "attacker-ns",
			UID:       "spoofed-run-id",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "spoofed-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "spoofed-schedule",
				UID:        types.UID(job.UUID),
			}},
		},
	})

	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported namespace does not match owning resource", err.(*util.UserError).ExternalMessage())
	assert.NotContains(t, err.Error(), job.Namespace)
	_, err = manager.GetRun("spoofed-run-id")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReportWorkflowResource_ExistingRunNameMismatchDoesNotDeleteWorkflow(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	workflowNamespace := run.Namespace
	if manager.IsEmptyNamespace(workflowNamespace) {
		workflowNamespace = common.GetPodNamespace()
	}

	decoy := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "same-namespace-decoy",
			Namespace: workflowNamespace,
			UID:       "same-namespace-decoy-id",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := store.ExecClient().Execution(workflowNamespace).Create(ctx, decoy, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, decoy)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported name does not match owning run", err.(*util.UserError).ExternalMessage())
	_, err = store.ExecClient().Execution(workflowNamespace).Get(ctx, decoy.ExecutionName(), v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflowNamespace, decoy.ExecutionName()))
}

func TestReportWorkflowResource_DuplicateRecurringRunOwnershipMismatchRejected(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.runStore = &duplicateRecurringRunStore{
		RunStoreInterface: store.runStore,
		firstGet:          true,
		existingRun: &model.Run{
			UUID:           "shared-run-id",
			ExperimentId:   "other-experiment",
			RecurringRunId: "other-recurring-run",
			K8SName:        "other-workflow",
			Namespace:      "other-namespace",
		},
	}
	manager.runStore = store.runStore

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "legitimate-workflow",
			Namespace: job.Namespace,
			UID:       "shared-run-id",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "shared-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)

	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: persisted run ownership does not match recurring run", err.(*util.UserError).ExternalMessage())
}

func TestReportWorkflowResource_DuplicateRecurringRunIdentityMismatchRejected(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()

	storedWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "legitimate-workflow",
			Namespace: job.Namespace,
			UID:       "persisted-workflow-uid",
		},
	})
	store.runStore = &duplicateRecurringRunStore{
		RunStoreInterface: store.runStore,
		firstGet:          true,
		existingRun: &model.Run{
			UUID:           "shared-run-id",
			ExperimentId:   job.ExperimentId,
			RecurringRunId: job.UUID,
			K8SName:        storedWorkflow.ExecutionName(),
			Namespace:      job.Namespace,
			RunDetails: model.RunDetails{
				WorkflowRuntimeManifest: model.LargeText(storedWorkflow.ToStringForStore()),
			},
		},
	}
	manager.runStore = store.runStore

	replacement := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      storedWorkflow.ExecutionName(),
			Namespace: job.Namespace,
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "shared-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	syncWorkflowReportWithFakeCluster(t, store, replacement)
	require.NotEqual(t, storedWorkflow.ExecutionObjectMeta().UID, replacement.ExecutionObjectMeta().UID)

	_, err := manager.ReportWorkflowResource(ctx, replacement)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported identity does not match the stored workflow", err.(*util.UserError).ExternalMessage())
	_, getErr := store.ExecClient().Execution(job.Namespace).Get(ctx, replacement.ExecutionName(), v1.GetOptions{})
	require.NoError(t, getErr, "identity rejection must not delete the replacement workflow")
}

func TestReportWorkflowResource_ExistingRecurringRunOwnershipMismatchRejected(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.runStore = &duplicateRecurringRunStore{
		RunStoreInterface: store.runStore,
		existingRun: &model.Run{
			UUID:           "shared-run-id",
			ExperimentId:   job.ExperimentId,
			RecurringRunId: "other-recurring-run",
			K8SName:        "legitimate-workflow",
			Namespace:      job.Namespace,
		},
	}
	manager.runStore = store.runStore

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "legitimate-workflow",
			Namespace: job.Namespace,
			UID:       "shared-run-id",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "shared-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})

	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported owner does not match owning run", err.(*util.UserError).ExternalMessage())
}

func TestReportWorkflowResource_OrphanedRecurringWorkflowSucceeds(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName:    "orphaned-recurring-workflow",
		ExperimentId:   job.ExperimentId,
		RecurringRunId: job.UUID,
		Namespace:      job.Namespace,
		PipelineSpec:   job.PipelineSpec,
	})
	require.NoError(t, err)
	require.NoError(t, manager.DeleteJob(ctx, job.UUID, apiv2beta1.DeletePropagationPolicy_ORPHAN))

	orphanedWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	orphanedWorkflow.ExecutionObjectMeta().OwnerReferences = nil
	orphanedWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowSucceeded
	_, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, orphanedWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, orphanedWorkflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateSucceeded, updatedRun.State)
}

func TestReportWorkflowResource_MissingRecurringOwnerWithExistingJobIsPermanent(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()
	manager.options.CollectMetrics = true

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName:    "owner-stripped-recurring-workflow",
		ExperimentId:   job.ExperimentId,
		RecurringRunId: job.UUID,
		Namespace:      job.Namespace,
		PipelineSpec:   job.PipelineSpec,
	})
	require.NoError(t, err)

	workflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	workflow.ExecutionObjectMeta().OwnerReferences = nil
	workflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowSucceeded
	workflow, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, workflow, v1.UpdateOptions{})
	require.NoError(t, err)

	rejectionCounter := workflowReportRejectedCounter.WithLabelValues(workflowReportRejectionIdentityMismatch)
	metric := &dto.Metric{}
	require.NoError(t, rejectionCounter.Write(metric))
	rejectionsBefore := metric.GetCounter().GetValue()

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "got %v", err)
	assert.Contains(t, err.Error(), "recurring-run owner is missing")

	metric.Reset()
	require.NoError(t, rejectionCounter.Write(metric))
	assert.Equal(t, rejectionsBefore+1, metric.GetCounter().GetValue())
}

func TestReportWorkflowResource_ExistingRunRejectsSameNameReplacementUID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	originalManifest := run.WorkflowRuntimeManifest
	originalUID := storedWorkflowUID(t, run)

	require.NoError(t, store.ExecClient().Execution(run.Namespace).Delete(
		ctx, run.K8SName, v1.DeleteOptions{}))
	replacement := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			UID:       "same-name-replacement-uid",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := store.ExecClient().Execution(run.Namespace).Create(
		ctx, replacement, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, replacement)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	assert.Contains(t, err.Error(), "reported identity does not match the stored workflow")

	unchangedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, originalManifest, unchangedRun.WorkflowRuntimeManifest)
	assert.Equal(t, originalUID, storedWorkflowUID(t, unchangedRun))
	assert.NotEqual(t, replacement.ExecutionObjectMeta().UID, storedWorkflowUID(t, unchangedRun))
}

func TestReportWorkflowResource_AdoptsRecreatedWorkflowForActiveRetryClaim(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()
	ctx := context.Background()

	run, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	originalUID := storedWorkflowUID(t, run)
	_, _, _, claimGeneration, err := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), claimGeneration)

	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.NoError(t, execSpec.Decompress())
	retryExecSpec, _, err := execSpec.GenerateRetryExecution()
	require.NoError(t, err)
	retryExecSpec.SetAnnotations(util.AnnotationKeyRetryGeneration, strconv.FormatInt(claimGeneration, 10))
	retryExecSpec.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning

	workflowClient := store.ExecClient().Execution(run.Namespace)
	require.NoError(t, workflowClient.Delete(ctx, run.K8SName, v1.DeleteOptions{}))
	recreatedWorkflow, err := workflowClient.Create(ctx, retryExecSpec, v1.CreateOptions{})
	require.NoError(t, err)
	require.NotEqual(t, originalUID, recreatedWorkflow.ExecutionObjectMeta().UID)

	_, err = manager.ReportWorkflowResource(ctx, recreatedWorkflow)
	require.NoError(t, err)
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, updatedRun.State)
	assert.Equal(t, recreatedWorkflow.ExecutionObjectMeta().UID, storedWorkflowUID(t, updatedRun))

	_, err = manager.ReportWorkflowResource(ctx, recreatedWorkflow)
	require.NoError(t, err, "the adopted identity must remain valid for later workflow reports")
}

func TestReportWorkflowResource_FinalizesDeletedOrphanedRecurringWorkflow(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName:    "deleted-orphaned-recurring-workflow",
		ExperimentId:   job.ExperimentId,
		RecurringRunId: job.UUID,
		Namespace:      job.Namespace,
		PipelineSpec:   job.PipelineSpec,
	})
	require.NoError(t, err)
	require.NoError(t, manager.DeleteJob(ctx, job.UUID, apiv2beta1.DeletePropagationPolicy_ORPHAN))

	orphanedWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	orphanedWorkflow.ExecutionObjectMeta().OwnerReferences = nil
	orphanedWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowSucceeded
	orphanedWorkflow.(*util.Workflow).Status.FinishedAt = v1.NewTime(time.Unix(456, 0))
	_, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, orphanedWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, store.ExecClient().Execution(run.Namespace).Delete(
		ctx, run.K8SName, v1.DeleteOptions{}))

	_, err = manager.ReportWorkflowResource(ctx, orphanedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "got %v", err)

	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateSucceeded, updatedRun.State)
	assert.Equal(t, int64(456), updatedRun.FinishedAtInSec)
}

func TestReportWorkflowResource_PersistedRecurringWorkflowCreatesMissingRunBeforeDelete(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "persisted-recurring-workflow",
			Namespace: job.Namespace,
			UID:       "persisted-recurring-run-id",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "persisted-recurring-run-id",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowSucceeded},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)

	reportedWorkflow, err := manager.ReportWorkflowResource(ctx, workflow)
	require.NoError(t, err)
	assert.Equal(
		t,
		"persisted-recurring-run-id",
		reportedWorkflow.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowRunId],
	)
	createdRun, err := manager.GetRun("persisted-recurring-run-id")
	require.NoError(t, err)
	assert.Equal(t, job.UUID, createdRun.RecurringRunId)
	assert.Equal(t, job.Namespace, createdRun.Namespace)
	_, err = store.ExecClient().Execution(job.Namespace).Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	assert.True(t, util.IsNotFound(err))
}

func TestReportWorkflowResource_RecurringRunRejectsStaleWorkflowUID(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()
	const runID = "replacement-recurring-run-id"

	liveWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "replacement-recurring-workflow",
			Namespace: job.Namespace,
			UID:       "replacement-workflow-uid",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: runID},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := store.ExecClient().Execution(job.Namespace).Create(
		ctx, liveWorkflow, v1.CreateOptions{})
	require.NoError(t, err)
	staleReport := util.NewWorkflow(liveWorkflow.DeepCopy())
	staleReport.UID = "stale-workflow-uid"

	_, err = manager.ReportWorkflowResource(ctx, staleReport)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	_, err = manager.GetRun(runID)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	_, err = store.ExecClient().Execution(job.Namespace).Get(
		ctx, liveWorkflow.ExecutionName(), v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(
		job.Namespace, liveWorkflow.ExecutionName()))
}

func TestCreateOrUpdateTasks_RejectsWorkflowNamespaceMismatch(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	_, err := manager.CreateOrUpdateTasks(
		[]*model.Task{{RunID: run.UUID, Namespace: "attacker-ns", PodName: "attacker-task"}},
		run.UUID,
		"attacker-ns",
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestCreateOrUpdateTasks_RejectsTaskRunIDMismatch(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	_, err := manager.CreateOrUpdateTasks(
		[]*model.Task{{RunID: "another-run", Namespace: run.Namespace, PodName: "mismatched-task"}},
		run.UUID,
		run.Namespace,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "does not match owning run")
}

func TestCreateOrUpdateTasks_RejectsTaskNamespaceMismatch(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	_, err := manager.CreateOrUpdateTasks(
		[]*model.Task{{RunID: run.UUID, Namespace: "attacker-ns", PodName: "mismatched-task"}},
		run.UUID,
		run.Namespace,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "task namespace does not match owning run")
}

func TestCreateOrUpdateTasksForRun_RejectsTasksAfterRunIDRecreation(t *testing.T) {
	store, manager, originalRun := initWithOneTimeRunV2(t)
	defer store.Close()
	ctx := context.Background()
	workflowClient := store.ExecClient().Execution(originalRun.Namespace)
	originalWorkflow, err := workflowClient.Get(ctx, originalRun.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	staleRun, err := manager.GetRun(originalRun.UUID)
	require.NoError(t, err)
	_, err = manager.ReportWorkflowResourceWithRun(ctx, originalWorkflow, staleRun)
	require.NoError(t, err)
	require.NotEmpty(t, staleRun.WorkflowRuntimeManifest)
	staleIdentity, found := manager.storedWorkflowIdentities.load(originalRun.UUID)
	require.True(t, found)
	assert.Equal(t, storedWorkflowUID(t, staleRun), staleIdentity.uid)

	// Recreate the run through another manager so this manager retains A's
	// cached identity until the guarded task write detects B and refreshes it.
	replacementManager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	require.NoError(t, replacementManager.DeleteRun(ctx, originalRun.UUID))

	// Fake workflow clients allocate UIDs per namespace, while Kubernetes UIDs
	// are cluster-wide. Advance the replacement namespace once so this fixture
	// preserves the production invariant that recreated objects have new UIDs.
	_, err = store.ExecClient().Execution("ns2").Create(ctx, util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "uid-seed"},
	}), v1.CreateOptions{})
	require.NoError(t, err)
	replacementRun, err := replacementManager.CreateRun(ctx, &model.Run{
		UUID:         originalRun.UUID,
		DisplayName:  originalRun.DisplayName,
		ExperimentId: originalRun.ExperimentId,
		Namespace:    "ns2",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
	})
	require.NoError(t, err)
	require.NotEqual(t, storedWorkflowUID(t, staleRun), storedWorkflowUID(t, replacementRun))
	replacementBeforeTasks, err := replacementManager.GetRun(replacementRun.UUID)
	require.NoError(t, err)
	replacementWorkflow, err := store.ExecClient().Execution(replacementRun.Namespace).Get(
		ctx,
		replacementRun.K8SName,
		v1.GetOptions{},
	)
	require.NoError(t, err)

	staleTask := &model.Task{
		RunID:     staleRun.UUID,
		Namespace: staleRun.Namespace,
		PodName:   "stale-run-task",
		State:     model.RuntimeStateRunning,
	}
	_, err = manager.CreateOrUpdateTasksForRun(
		[]*model.Task{staleTask},
		staleRun,
		staleRun.Namespace,
	)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable), "got %v", err)
	assert.Empty(t, staleTask.UUID, "a rejected task report must not mutate task identity")

	var taskCount int
	require.NoError(t, store.DB().QueryRow(
		"SELECT COUNT(*) FROM tasks WHERE RunUUID = ?",
		replacementRun.UUID,
	).Scan(&taskCount))
	assert.Zero(t, taskCount)
	replacementAfterTasks, err := replacementManager.GetRun(replacementRun.UUID)
	require.NoError(t, err)
	assert.Equal(t, replacementBeforeTasks, replacementAfterTasks)
	stillLive, err := store.ExecClient().Execution(replacementRun.Namespace).Get(
		ctx,
		replacementRun.K8SName,
		v1.GetOptions{},
	)
	require.NoError(t, err)
	assert.Equal(t, replacementWorkflow.ExecutionObjectMeta().UID, stillLive.ExecutionObjectMeta().UID)
	refreshedIdentity, found := manager.storedWorkflowIdentities.load(replacementRun.UUID)
	require.True(t, found)
	assert.Equal(t, replacementWorkflow.ExecutionObjectMeta().UID, refreshedIdentity.uid)
	assert.Equal(t, replacementRun.Namespace, refreshedIdentity.namespace)
	assert.Equal(t,
		sha256.Sum256([]byte(replacementBeforeTasks.PipelineRuntimeManifest)),
		refreshedIdentity.manifestDigest,
	)
}

func TestReportWorkflowResource_ScheduledWorkflowNamespaceMismatchDoesNotDeletePersistedWorkflow(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "spoofed-persisted-run",
			Namespace: "attacker-ns",
			UID:       "spoofed-persisted-run-id",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "spoofed-persisted-run-id",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "spoofed-schedule",
				UID:        types.UID(job.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := store.ExecClient().Execution("attacker-ns").Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Equal(t, "Failed to report workflow: reported namespace does not match owning resource", err.(*util.UserError).ExternalMessage())
	assert.NotContains(t, err.Error(), job.Namespace)

	storedWorkflow, err := store.ExecClient().Execution("attacker-ns").Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, storedWorkflow)
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflow.ExecutionNamespace(), workflow.ExecutionName()))
	_, err = manager.GetRun("spoofed-persisted-run-id")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReportWorkflowResource_MissingRecurringExperimentReferenceDoesNotDeleteWorkflow(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()

	// Simulate an inconsistent legacy row: the recurring run still exists, but
	// neither its denormalized experiment column nor its experiment ownership
	// reference can identify the tenant. This is not proof that the workflow is
	// orphaned and therefore must never enter the orphan-GC path.
	_, err := store.DB().Exec(`UPDATE jobs SET ExperimentUUID = ? WHERE UUID = ?`, "", job.UUID)
	require.NoError(t, err)
	_, err = store.DB().Exec(
		`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`,
		job.UUID,
		model.JobResourceType,
		model.ExperimentResourceType,
	)
	require.NoError(t, err)

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "inconsistent-owner-run",
			Namespace: "ns1",
			UID:       "inconsistent-owner-run-id",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "inconsistent-owner-run-id",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "inconsistent-schedule",
				UID:        types.UID(job.UUID),
			}},
		},
	})
	_, err = store.ExecClient().Execution("ns1").Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Internal))
	assert.Contains(t, err.Error(), "Failed to retrieve the experiment ID")
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflow.ExecutionNamespace(), workflow.ExecutionName()))
	_, err = store.ExecClient().Execution("ns1").Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	require.NoError(t, err)
}

func TestReportWorkflowResource_MissingRecurringOwnerDoesNotDeletePersistedWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	defer store.Close()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "orphaned-persisted-run",
			Namespace: "ns1",
			UID:       "orphaned-persisted-run-id",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "orphaned-persisted-run-id",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "missing-schedule",
				UID:        "missing-job-id",
			}},
		},
	})
	_, err := store.ExecClient().Execution("ns1").Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflow.ExecutionNamespace(), workflow.ExecutionName()))
}

func TestReportWorkflowResource_MissingRecurringOwnerDoesNotDeleteYoungWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	defer store.Close()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "young-orphaned-run",
			Namespace:         "ns1",
			UID:               "young-orphaned-run-id",
			CreationTimestamp: v1.NewTime(store.Time().Now()),
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "young-orphaned-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "missing-schedule",
				UID:        "missing-job-id",
			}},
		},
	})
	_, err := store.ExecClient().Execution("ns1").Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflow.ExecutionNamespace(), workflow.ExecutionName()))
}

func TestReportWorkflowResource_ScheduledWorkflowNoNamespaceResolvedFromExperiment(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// Simulate a legacy job row which stored the namespace sentinel and must
	// recover its Kubernetes namespace from the owning experiment.
	_, err := store.DB().Exec(`UPDATE jobs SET Namespace = ? WHERE UUID = ?`, model.NoNamespace, job.UUID)
	require.NoError(t, err)
	experiment, err := manager.GetExperiment(job.ExperimentId)
	require.NoError(t, err)
	require.Equal(t, "ns1", experiment.Namespace)
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "legacy-run",
			Namespace: "ns1",
			UID:       "legacy-run-id",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "legacy-run-id"},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       job.K8SName,
				UID:        types.UID(job.UUID),
			}},
		},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)

	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)
	run, err := manager.GetRun("legacy-run-id")
	require.NoError(t, err)
	assert.Equal(t, "ns1", run.Namespace)
}

func TestReportWorkflowResource_WorkflowMissingRunID(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: run.K8SName,
		},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow[workflow-name] missing the Run ID label")
}

func TestReportWorkflowResource_RunNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()
	// Set CreationTimestamp far in the past so the workflow is beyond the GC grace period.
	// Use the injected fake time to ensure consistency with manager's time.
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete",
			Namespace:         "kubeflow",
			UID:               "obsolete-workflow-uid",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "run-id-not-exist"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	_, err := store.ExecClient().Execution("kubeflow").Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.NotNil(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Deleted orphaned workflow")
	assert.Equal(t, 1, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(workflow.ExecutionNamespace(), workflow.ExecutionName()))
	_, getErr := store.ExecClient().Execution("kubeflow").Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	assert.True(t, util.IsNotFound(getErr))
}

func TestReportWorkflowResource_RunNotFoundRetriesTransientDeleteFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete-after-transient-delete-failure",
			Namespace:         "kubeflow",
			UID:               "obsolete-after-transient-delete-failure-uid",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "missing-run-id"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	workflowClient := &transientDeleteFailureWorkflowClient{
		FakeWorkflowClient:      client.NewWorkflowClientFake(),
		deleteFailuresRemaining: 1,
	}
	_, err := workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Deleted orphaned workflow")
	assert.Equal(t, 2, workflowClient.deleteCalls)
	_, getErr := workflowClient.Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	assert.True(t, util.IsNotFound(getErr))
}

func TestReportWorkflowResource_RunNotFoundTreatsMissingDeleteLookupAsSuccess(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete-before-delete-lookup",
			Namespace:         "kubeflow",
			UID:               "obsolete-before-delete-lookup-uid",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "missing-run-id"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	workflowClient := &disappearBeforeDeleteLookupWorkflowClient{
		FakeWorkflowClient: client.NewWorkflowClientFake(),
	}
	_, err := workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Deleted orphaned workflow")
	assert.Equal(t, 2, workflowClient.getCalls, "a missing workflow must not consume the delete backoff budget")
}

func TestReportWorkflowResource_RunNotFoundTreatsDeleteNotFoundAsSuccess(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete-during-delete",
			Namespace:         "kubeflow",
			UID:               "obsolete-during-delete-uid",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "missing-run-id"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	workflowClient := &notFoundOnOrphanDeleteWorkflowClient{
		FakeWorkflowClient: client.NewWorkflowClientFake(),
	}
	_, err := workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Deleted orphaned workflow")
	assert.Equal(t, 2, workflowClient.getCalls, "a NotFound delete response must not be retried")
	assert.Equal(t, 1, workflowClient.deleteCalls)
}

func TestReportWorkflowResource_RunNotFoundRefreshesWorkflowBeforeDeleteRetry(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete-after-concurrent-update",
			Namespace:         "kubeflow",
			UID:               "obsolete-after-concurrent-update-uid",
			ResourceVersion:   "initial",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "missing-run-id"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	workflowClient := &updateBeforeFirstDeleteWorkflowClient{
		FakeWorkflowClient: client.NewWorkflowClientFake(),
	}
	_, err := workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "Deleted orphaned workflow")
	assert.Equal(t, 2, workflowClient.deleteCalls)
	_, getErr := workflowClient.Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	assert.True(t, util.IsNotFound(getErr))
}

func TestReportWorkflowResource_RunNotFoundStopsRetryingReplacementIdentityMismatch(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "obsolete-before-replacement",
			Namespace:         "kubeflow",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "missing-run-id"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
		},
	})
	workflowClient := &replaceBeforeOrphanDeleteLookupWorkflowClient{
		FakeWorkflowClient: client.NewWorkflowClientFake(),
	}
	reportedWorkflow, err := workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	reportedUID := reportedWorkflow.ExecutionObjectMeta().UID
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, reportedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "got %v", err)
	assert.Equal(t, 2, workflowClient.getCalls,
		"a permanent replacement-identity mismatch must not consume the delete backoff budget")

	replacement, getErr := workflowClient.Get(ctx, workflow.ExecutionName(), v1.GetOptions{})
	require.NoError(t, getErr)
	assert.NotEqual(t, reportedUID, replacement.ExecutionObjectMeta().UID)
}

func TestReportWorkflowResource_MissingOneTimeRunRejectsStaleWorkflowUID(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	liveWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "unowned-persisted-workflow",
			Namespace:         "attacker-ns",
			UID:               "replacement-workflow-uid",
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Minute)),
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "missing-run-id",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
	})
	_, err := store.ExecClient().Execution("attacker-ns").Create(ctx, liveWorkflow, v1.CreateOptions{})
	require.NoError(t, err)
	staleWorkflow := util.NewWorkflow(liveWorkflow.DeepCopy())
	staleWorkflow.ExecutionObjectMeta().UID = "stale-workflow-uid"

	_, err = manager.ReportWorkflowResource(ctx, staleWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowDeleteCountInNamespace(liveWorkflow.ExecutionNamespace(), liveWorkflow.ExecutionName()))
	storedWorkflow, getErr := store.ExecClient().Execution("attacker-ns").Get(ctx, liveWorkflow.ExecutionName(), v1.GetOptions{})
	require.NoError(t, getErr)
	assert.NotNil(t, storedWorkflow)
}

func TestReportWorkflowResource_RunNotFound_WithinGracePeriod(t *testing.T) {
	// When a workflow is young (within the GC grace period) and its run is not
	// found in the DB, it should NOT be deleted. Instead, a retryable
	// UNAVAILABLE error should be returned so the persistence agent retries.
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	defer store.Close()

	// Set CreationTimestamp to a recent time so the workflow is within the grace period.
	// Use the injected fake time to ensure consistency with manager's time.
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "young-workflow",
			Namespace:         "kubeflow",
			UID:               "young-workflow-uid",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: "run-id-not-exist"},
			CreationTimestamp: v1.NewTime(store.Time().Now().Add(-10 * time.Second)),
		},
	})
	store.ExecClient().Execution("kubeflow").Create(ctx, workflow, v1.CreateOptions{})
	_, err := manager.ReportWorkflowResource(ctx, workflow)
	require.NotNil(t, err)
	// Should be UNAVAILABLE (retryable), not NOT_FOUND (permanent).
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable),
		"Expected Unavailable error for young workflow within grace period, got: %v", err)
	assert.Contains(t, err.Error(), "run-creation grace period")

	// Verify the workflow was NOT deleted by checking it's still accessible.
	wf, getErr := store.ExecClient().Execution("kubeflow").Get(ctx, "young-workflow", v1.GetOptions{})
	assert.Nil(t, getErr, "Workflow should still exist after grace period skip")
	assert.NotNil(t, wf)
}

func TestReportWorkflowResource_WorkflowCompleted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := common.GetPodNamespace()
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: namespace,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	wf, err := store.ExecClientFake.Execution(namespace).Get(context.Background(), run.K8SName, v1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowPersistedFinalState], "true")
}

func TestAddWorkflowLabelIfWorkflowUnchanged_SkipsWhenWorkflowWasRetried(t *testing.T) {
	wfClient := client.NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:            "workflow-name",
			Namespace:       "ns1",
			ResourceVersion: "retry-version",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: "run-id"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})
	_, err := wfClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)

	labelAdded, err := addWorkflowLabelIfWorkflowUnchanged(
		ctx,
		wfClient,
		"workflow-name",
		"terminal-version",
		util.LabelKeyWorkflowPersistedFinalState,
		"true",
	)
	require.NoError(t, err)
	assert.False(t, labelAdded)

	updatedWorkflow, err := wfClient.Get(ctx, "workflow-name", v1.GetOptions{})
	require.NoError(t, err)
	_, hasFinalStateLabel := updatedWorkflow.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowPersistedFinalState]
	assert.False(t, hasFinalStateLabel)
}

func TestReportWorkflowResource_SkipsTerminalPluginSyncWhenReportedWorkflowIsStale(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := "ns1"
	ctx := context.Background()
	defer store.Close()

	runWithPluginOutput, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	mlflowOutput := apiservermlflow.SuccessfulPluginOutput("exp-1", "exp-1", "parent-run-1", "https://mlflow.example/runs/parent-run-1")
	pluginsOutput, err := apiserverPlugins.SerializePluginsOutput(map[string]*apiv2beta1.PluginOutput{apiservermlflow.PluginName: mlflowOutput})
	require.NoError(t, err)
	runWithPluginOutput.State = model.RuntimeStateRunning
	runWithPluginOutput.Conditions = string(model.RuntimeStateRunning.ToV1())
	runWithPluginOutput.FinishedAtInSec = 0
	runWithPluginOutput.PluginsOutputString = pluginsOutput
	require.NoError(t, manager.runStore.UpdateRun(runWithPluginOutput))

	dispatcher := &countingTerminalReportDispatcher{}
	manager.pluginDispatcher = dispatcher

	currentWorkflow, err := store.ExecClientFake.Execution(namespace).Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	currentWorkflow.SetVersion("retry-version")
	_, err = store.ExecClientFake.Execution(namespace).Update(ctx, currentWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)

	staleTerminalWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:            run.K8SName,
			Namespace:       namespace,
			UID:             types.UID(run.UUID),
			ResourceVersion: "terminal-version",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.NewTime(time.Unix(123, 0)),
		},
	})

	reportedWorkflow, err := manager.ReportWorkflowResource(ctx, staleTerminalWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable))
	assert.Nil(t, reportedWorkflow)
	assert.Equal(t, 0, dispatcher.onRunEndCalls)

	currentRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, currentRun.State)
	assert.Equal(t, int64(0), currentRun.FinishedAtInSec)
}

func TestReportWorkflowResource_FinalizesRunWhenWorkflowDeletedBeforeTerminalReport(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := "ns1"
	ctx := context.Background()
	defer store.Close()
	require.NoError(t, store.ExecClient().Execution(namespace).Delete(ctx, run.K8SName, v1.DeleteOptions{}))

	// Report a terminal workflow whose CR no longer exists, simulating a
	// deletion between the persistence agent's read and this report. The run
	// row must still be finalized before the caller receives the NotFound
	// signal that stops further retries.
	deletedWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:            run.K8SName,
			Namespace:       namespace,
			UID:             storedWorkflowUID(t, run),
			ResourceVersion: "terminal-version",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.NewTime(time.Unix(123, 0)),
		},
	})

	reportedWorkflow, err := manager.ReportWorkflowResource(ctx, deletedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound),
		"caller should receive the NotFound signal so the persistence agent stops retrying, got %v", err)
	assert.Nil(t, reportedWorkflow)

	currentRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, currentRun.State)
	assert.Equal(t, int64(123), currentRun.FinishedAtInSec)
}

func TestReportWorkflowResource_FinalizesLegacyEmptyNameRunWhenWorkflowDeletedBeforeTerminalReport(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := "ns1"
	ctx := context.Background()
	defer store.Close()
	reportedName := run.K8SName
	reportedUID := storedWorkflowUID(t, run)
	require.NoError(t, store.ExecClient().Execution(namespace).Delete(ctx, reportedName, v1.DeleteOptions{}))
	_, err := store.DB().Exec(
		`UPDATE run_details SET Name = '', Namespace = '' WHERE UUID = ?`, run.UUID)
	require.NoError(t, err)

	deletedWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:            reportedName,
			Namespace:       namespace,
			UID:             reportedUID,
			ResourceVersion: "terminal-version",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.NewTime(time.Unix(123, 0)),
		},
	})

	reportedWorkflow, err := manager.ReportWorkflowResource(ctx, deletedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound),
		"caller should receive the NotFound signal after the legacy run is finalized, got %v", err)
	assert.Nil(t, reportedWorkflow)

	currentRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, currentRun.State)
	assert.Equal(t, int64(123), currentRun.FinishedAtInSec)
	assert.Equal(t, reportedName, currentRun.K8SName)
	assert.Equal(t, namespace, currentRun.Namespace)
}

func TestReportWorkflowResource_FinalizesV2RunWhenWorkflowDeletedBeforeFirstReport(t *testing.T) {
	store, manager, run := initWithOneTimeRunV2(t)
	namespace := "ns1"
	ctx := context.Background()
	defer store.Close()
	require.Empty(t, run.WorkflowRuntimeManifest)
	require.NotEmpty(t, run.PipelineRuntimeManifest)
	require.NoError(t, store.ExecClient().Execution(namespace).Delete(ctx, run.K8SName, v1.DeleteOptions{}))
	manager.options.CollectMetrics = true
	rejectionCounter := workflowReportRejectedCounter.WithLabelValues(workflowReportRejectionOwnershipUnresolved)
	counterValue := func() float64 {
		metric := &dto.Metric{}
		require.NoError(t, rejectionCounter.Write(metric))
		return metric.GetCounter().GetValue()
	}
	rejectionsBefore := counterValue()

	deletedWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:            run.K8SName,
			Namespace:       namespace,
			UID:             storedWorkflowUID(t, run),
			ResourceVersion: "terminal-version",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.NewTime(time.Unix(123, 0)),
		},
	})

	reportedWorkflow, err := manager.ReportWorkflowResource(ctx, deletedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound),
		"caller should receive the NotFound signal so the persistence agent stops retrying, got %v", err)
	assert.Nil(t, reportedWorkflow)

	currentRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, currentRun.State)
	assert.Equal(t, int64(123), currentRun.FinishedAtInSec)
	assert.NotEmpty(t, currentRun.WorkflowRuntimeManifest)
	assert.Equal(t, rejectionsBefore, counterValue(),
		"an accepted stored-identity fallback must not be counted as a rejected report")
}

func TestRecordWorkflowReportLiveLookupRejectionIgnoresRetryableErrors(t *testing.T) {
	manager := &ResourceManager{options: &ResourceManagerOptions{CollectMetrics: true}}
	counter := workflowReportRejectedCounter.WithLabelValues(workflowReportRejectionOwnershipUnresolved)
	counterValue := func() float64 {
		metric := &dto.Metric{}
		require.NoError(t, counter.Write(metric))
		return metric.GetCounter().GetValue()
	}
	before := counterValue()

	manager.recordWorkflowReportLiveLookupRejection(util.NewUnavailableServerError(
		errors.New("temporary Kubernetes outage"), "will retry"))
	assert.Equal(t, before, counterValue())

	manager.recordWorkflowReportLiveLookupRejection(util.NewNotFoundError(
		errors.New("workflow missing"), "cannot apply report"))
	assert.Equal(t, before+1, counterValue())
}

func TestReportWorkflowResource_SkipsPersistedFinalStateLabelWhenRunRetriedDuringPluginSync(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	namespace := "ns1"
	defer store.Close()

	runWithPluginOutput, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	mlflowOutput := apiservermlflow.SuccessfulPluginOutput("exp-1", "exp-1", "parent-run-1", "https://mlflow.example/runs/parent-run-1")
	pluginsOutput, err := apiserverPlugins.SerializePluginsOutput(map[string]*apiv2beta1.PluginOutput{apiservermlflow.PluginName: mlflowOutput})
	require.NoError(t, err)
	runWithPluginOutput.PluginsOutputString = pluginsOutput
	require.NoError(t, manager.runStore.UpdateRun(runWithPluginOutput))

	dispatcher := &retryDuringTerminalReportDispatcher{
		manager: manager,
		runID:   run.UUID,
	}
	manager.pluginDispatcher = dispatcher

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: namespace,
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.NewTime(time.Unix(123, 0)),
		},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)

	reportedWorkflow, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable))
	assert.Nil(t, reportedWorkflow)
	require.NoError(t, dispatcher.retryErr)

	retriedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, retriedRun.State)
	assert.Equal(t, int64(0), retriedRun.FinishedAtInSec)

	retriedWorkflow, err := store.ExecClientFake.Execution(namespace).Get(context.Background(), run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(v1alpha1.WorkflowRunning), string(retriedWorkflow.ExecutionStatus().Condition()))
	_, hasFinalStateLabel := retriedWorkflow.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowPersistedFinalState]
	assert.False(t, hasFinalStateLabel)
}

// TestReportWorkflow_WithMLflowOnRunEnd verifies that when a run has PluginsOutputString
// set, reporting a terminal workflow triggers the plugin dispatcher's
// OnRunEnd, which updates the plugin output state.
func TestReportWorkflow_WithMLflowOnRunEnd(t *testing.T) {
	// Set a dummy MLflow config so the manager creates a real MLflow dispatcher,
	// then clear it before the run lifecycle to simulate config being unavailable
	// at OnRunEnd time. This verifies that OnRunEnd still fires and sets
	// PLUGIN_FAILED because config is unavailable.
	setupMLflowViperConfig(t, "http://dummy-mlflow:5000")

	store, manager, exp := initWithExperiment(t)

	// Now clear MLflow config to simulate it being unavailable at runtime.
	viper.Set("plugins", nil)
	t.Cleanup(func() {
		viper.Set("plugins", nil)
	})
	defer store.Close()

	// Pre-populate PluginsOutputString to simulate a prior OnBeforeRunCreation success.
	pluginsOutputJSON := `{"mlflow":{"entries":{"experiment_id":{"value":"exp-1"},"root_run_id":{"value":"parent-run-1"}},"state":"PLUGIN_SUCCEEDED"}}`
	pluginsOutput := model.LargeText(pluginsOutputJSON)
	apiRun := &model.Run{
		DisplayName: "mlflow-run",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: exp.UUID,
		RunDetails: model.RunDetails{
			PluginsOutputString: &pluginsOutput,
		},
	}
	run, err := manager.CreateRun(context.Background(), apiRun)
	require.NoError(t, err)

	// Verify PluginsOutputString was persisted at creation time.
	createdRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.NotNil(t, createdRun.PluginsOutputString)
	assert.Contains(t, string(*createdRun.PluginsOutputString), "parent-run-1")

	// Report a terminal (failed) workflow.
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err,
		"an unavailable MLflow config is a permanent plugin failure and must not block run finalization")

	// After terminal report, the plugin dispatcher's OnRunEnd should have fired.
	// Without MLflow config in Viper, the handler sets PLUGIN_FAILED, and the
	// run still reaches its terminal state.
	updatedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.NotNil(t, updatedRun.PluginsOutputString, "PluginsOutputString should be updated after terminal report")
	assert.Contains(t, string(*updatedRun.PluginsOutputString), "PLUGIN_FAILED")
	assert.Contains(t, string(*updatedRun.PluginsOutputString), "config unavailable")
	assert.Equal(t, model.RuntimeStateFailed, updatedRun.State)
}

func TestReportWorkflowResource_WorkflowCompleted_WorkflowNotFound(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	require.NoError(t, store.ExecClient().Execution(common.GetPodNamespace()).Delete(
		context.Background(), run.K8SName, v1.DeleteOptions{}))
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: common.GetPodNamespace(),
			UID:       storedWorkflowUID(t, run),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NotNil(t, err)
	assert.Equalf(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), "Expected not found error, but got %s", err.Error())
	assert.Contains(t, err.Error(), "Failed to add PersistedFinalState label")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)
	_, err = store.ExecClient().Execution("ns1").Get(context.Background(), workflow.ExecutionName(), v1.GetOptions{})
	assert.True(t, util.IsNotFound(err))
}

func TestReportWorkflowResource_StaleNonterminalReportDoesNotPoisonRetriedWorkflowIdentity(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()
	workflowClient := store.ExecClient().Execution(run.Namespace)

	liveWorkflow, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	staleNonterminal := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	require.False(t, staleNonterminal.ExecutionStatus().IsInFinalState())

	terminalWorkflow := util.NewWorkflow(liveWorkflow.(*util.Workflow).DeepCopy())
	terminalWorkflow.Status.Phase = v1alpha1.WorkflowFailed
	terminalWorkflow.Status.FinishedAt = v1.NewTime(time.Unix(123, 0))
	terminalWorkflow.SetLabels(util.LabelKeyWorkflowPersistedFinalState, "true")
	updatedTerminal, err := workflowClient.Update(ctx, terminalWorkflow, v1.UpdateOptions{})
	require.NoError(t, err)
	oldUID := updatedTerminal.ExecutionObjectMeta().UID

	// The submitted snapshot is stale and non-terminal, but identity resolution
	// replaces it with the live terminal workflow and deletes that workflow.
	_, err = manager.ReportWorkflowResource(ctx, staleNonterminal)
	require.NoError(t, err)
	_, err = workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.True(t, util.IsNotFound(err))
	_, found := manager.storedWorkflowIdentities.load(run.UUID)
	assert.False(t, found, "effective terminal cleanup must invalidate the stored identity cache")

	// RetryRun must also invalidate any entry that predates the recreated
	// workflow identity, including one left by an overlapping report.
	manager.storedWorkflowIdentities.loadOrStore(run.UUID, storedWorkflowIdentity{
		name: run.K8SName,
		uid:  oldUID,
	})
	require.NoError(t, manager.RetryRun(ctx, run.UUID))
	_, found = manager.storedWorkflowIdentities.load(run.UUID)
	assert.False(t, found, "retry persistence must invalidate the previous workflow identity")
	// Model an overlapping report that loaded the pre-retry row and reaches the
	// cache after RetryRun invalidated it. Its generation-zero identity must not
	// poison reports for the generation-one workflow.
	manager.storedWorkflowIdentities.loadOrStore(run.UUID, storedWorkflowIdentity{
		name:            run.K8SName,
		uid:             oldUID,
		retryGeneration: 0,
	})

	retriedWorkflow, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	require.NotEqual(t, oldUID, retriedWorkflow.ExecutionObjectMeta().UID)
	_, err = manager.ReportWorkflowResource(ctx, retriedWorkflow)
	require.NoError(t, err, "reports from the recreated retry must validate against its new UID")
	cachedIdentity, found := manager.storedWorkflowIdentities.load(run.UUID)
	require.True(t, found)
	assert.Equal(t, retriedWorkflow.ExecutionObjectMeta().UID, cachedIdentity.uid)
	assert.Equal(t, int64(1), cachedIdentity.retryGeneration)
}

func TestReportWorkflowResource_PersistedFinalStateDoesNotDeleteConcurrentRetry(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	workflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	workflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowFailed
	workflow.(*util.Workflow).Status.FinishedAt = v1.NewTime(time.Unix(123, 0))
	workflow, err = store.ExecClient().Execution(run.Namespace).Update(
		ctx, workflow, v1.UpdateOptions{})
	require.NoError(t, err)
	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.NoError(t, err)

	persistedWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	require.True(t, persistedWorkflow.PersistedFinalState())
	workflowClient, ok := store.ExecClientFake.Execution(run.Namespace).(*client.FakeWorkflowClient)
	require.True(t, ok)
	interleavingClient := &retryBeforeDeleteWorkflowClient{
		FakeWorkflowClient: workflowClient,
		manager:            manager,
		runID:              run.UUID,
	}
	manager.execClient = &retryWorkflowExecClient{workflowClient: interleavingClient}

	_, err = manager.ReportWorkflowResource(ctx, persistedWorkflow)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.Unavailable), "got %v", err)
	assert.Contains(t, err.Error(), "workflow changed before persisted-final-state cleanup")
	require.NoError(t, interleavingClient.retryErr)

	retriedWorkflow, err := workflowClient.Get(ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(v1alpha1.WorkflowRunning), string(retriedWorkflow.ExecutionStatus().Condition()))
	assert.False(t, retriedWorkflow.PersistedFinalState())
	retriedRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, retriedRun.State)
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersistedReusesLiveLookup(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	createdWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	workflow := util.NewWorkflow(createdWorkflow.(*util.Workflow).DeepCopy())
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.SetLabels(util.LabelKeyWorkflowPersistedFinalState, "true")
	workflow.SetVersion("current-version")

	workflowClient := &countingWorkflowClient{FakeWorkflowClient: client.NewWorkflowClientFake()}
	_, err = workflowClient.Create(ctx, workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}

	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.NoError(t, err)
	assert.Equal(t, 1, workflowClient.getCalls,
		"version, identity, and persisted-final-state checks should share one live lookup")
}

func TestReportWorkflowResource_PersistedFinalStateRestoresAbandonedRetryBeforeDelete(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	ctx := context.Background()

	terminal := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			FinishedAt: v1.Time{Time: time.Unix(500, 0)},
		},
	})
	syncWorkflowReportWithFakeCluster(t, store, terminal)
	_, err := manager.ReportWorkflowResource(ctx, terminal)
	require.NoError(t, err)
	_, _, _, generation, err := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), generation)
	_, err = store.DB().Exec(
		`UPDATE run_details SET RetryClaimedAtInSec = 0 WHERE UUID = ?`, run.UUID)
	require.NoError(t, err)

	persistedWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	require.True(t, persistedWorkflow.PersistedFinalState())
	_, err = manager.ReportWorkflowResource(ctx, persistedWorkflow)
	require.NoError(t, err)

	recoveredRun, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, recoveredRun.State,
		"persisted final state must repair an abandoned retry claim before cleanup")
	assert.NotZero(t, recoveredRun.FinishedAtInSec)
	_, err = store.ExecClient().Execution(run.Namespace).Get(
		ctx, run.K8SName, v1.GetOptions{})
	assert.True(t, util.IsNotFound(err))
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_WorkflowNotFound(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	require.NoError(t, store.ExecClient().Execution(common.GetPodNamespace()).Delete(
		context.Background(), run.K8SName, v1.DeleteOptions{}))
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: common.GetPodNamespace(),
			UID:       storedWorkflowUID(t, run),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NotNil(t, err)
	assert.Equalf(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), "Expected not found error, but got %s", err.Error())
	assert.Contains(t, err.Error(), "Failed to delete the completed workflow")
}

func TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_DeleteFailed(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()
	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID, util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	workflowClient := &deleteFailureWorkflowClient{FakeWorkflowClient: client.NewWorkflowClientFake()}
	_, err := workflowClient.Create(context.Background(), workflow, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete workflow")
}

func TestReportScheduledWorkflowResource_Success(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(job.UUID),
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		K8SName:        "MY_NAME",
		DisplayName:    "j1",
		Namespace:      actualJob.Namespace,
		ExperimentId:   actualJob.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			PipelineSpecManifest: actualJob.PipelineSpec.PipelineSpecManifest,
			PipelineName:         actualJob.PipelineSpec.PipelineName,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReportScheduledWorkflowResource_Success_withParamsV1(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(job.UUID),
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{
						Name:  "param_v1",
						Value: "value_v1",
					},
				},
			},
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		K8SName:        "MY_NAME",
		DisplayName:    "j1",
		Namespace:      actualJob.Namespace,
		ExperimentId:   actualJob.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			Parameters:           `[{"name":"param_v1","value":"value_v1"}]`,
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			PipelineSpecManifest: actualJob.PipelineSpec.PipelineSpecManifest,
			PipelineName:         actualJob.PipelineSpec.PipelineName,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReportScheduledWorkflowResource_Success_withRuntimeParamsV2(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()
	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "updated_name",
			Namespace: "ns1",
			UID:       types.UID(job.UUID),
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{
						Name:  "param1",
						Value: "\"world-updated\"",
					},
				},
			},
		},
	})
	err := manager.ReportScheduledWorkflowResource(swf)
	assert.Nil(t, err)

	actualJob, err := manager.GetJob(job.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		K8SName:        "updated_name",
		DisplayName:    "j1",
		Namespace:      "ns1",
		ExperimentId:   job.ExperimentId,
		ServiceAccount: "pipeline-runner",
		Enabled:        false,
		UUID:           actualJob.UUID,
		Conditions:     "STATUS_UNSPECIFIED",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			PipelineName:         actualJob.PipelineSpec.PipelineName,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `{"param1":"world-updated"}`,
				PipelineRoot: "job-1-root",
			},
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 3,
	}
	expectedJob.Conditions = "STATUS_UNSPECIFIED"
	assert.Equal(t, expectedJob.ToV1(), actualJob.ToV1())
}

func TestReconcileSwfCrs(t *testing.T) {
	store, manager, job := initWithJobV2(t)
	defer store.Close()

	fetchedJob, err := manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.NotNil(t, fetchedJob)

	swfClient := store.SwfClient().ScheduledWorkflow("ns1")

	options := v1.GetOptions{}
	ctx := context.Background()

	swf, err := swfClient.Get(ctx, "job-", options)
	require.Nil(t, err)

	// emulates an invalid/outdated spec
	swf.Spec.Workflow.Spec = nil
	swf, err = swfClient.Update(ctx, swf)
	require.Nil(t, swf.Spec.Workflow.Spec)

	err = manager.ReconcileSwfCrs(ctx)
	require.Nil(t, err)

	swf, err = swfClient.Get(ctx, "job-", options)
	require.Nil(t, err)
	require.NotNil(t, swf.Spec.Workflow.Spec)
}

func TestReportScheduledWorkflowResource_Error(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.CreateDefaultExperiment("")
	// Create pipeline
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
	})
	p := createPipelineV1("1")
	pipeline, err := manager.CreatePipeline(p)
	assert.Nil(t, err)

	pv := createPipelineVersion(
		pipeline.UUID,
		"1",
		"",
		"",
		workflow.ToStringForStore(),
		"",
		pipeline.Namespace,
	)
	_, err = manager.CreatePipelineVersion(pv)
	assert.Nil(t, err)

	// Create job
	job := &model.Job{
		K8SName:      "pp1",
		Enabled:      true,
		PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID},
	}
	newJob, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	store.Close()

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newJob.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf)
	assert.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.(*util.UserError).String(), "database is closed")
}

const (
	v2compatPipeline = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: two-step-pipeline-
  annotations:
    pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
    pipelines.kubeflow.org/pipeline_compilation_time: '2021-07-14T06:59:20.208189'
    pipelines.kubeflow.org/pipeline_spec: '{"inputs": [{"default": "", "name": "pipeline-root"},
      {"default": "pipeline/two_step_pipeline", "name": "pipeline-name"}], "name":
      "two_step_pipeline"}'
    pipelines.kubeflow.org/v2_pipeline: "true"
  labels:
    pipelines.kubeflow.org/v2_pipeline: "true"
    pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
spec:
  entrypoint: two-step-pipeline
  templates:
  - name: preprocess
    container:
      args:
      - sh
      - -ec
      - |
        program_path=$(mktemp)
        printf "%s" "$0" > "$program_path"
        python3 -u "$program_path" "$@"
      - |
        def _make_parent_dirs_and_return_path(file_path: str):
            import os
            os.makedirs(os.path.dirname(file_path), exist_ok=True)
            return file_path

        def preprocess(
            uri, some_int, output_parameter_one,
            output_dataset_one
        ):
            '''Dummy Preprocess Step.'''
            with open(output_dataset_one, 'w') as f:
                f.write('Output dataset')
            with open(output_parameter_one, 'w') as f:
                f.write("{}".format(1234))

        import argparse
        _parser = argparse.ArgumentParser(prog='Preprocess', description='Dummy Preprocess Step.')
        _parser.add_argument("--uri", dest="uri", type=str, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--some-int", dest="some_int", type=int, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--output-parameter-one", dest="output_parameter_one", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--output-dataset-one", dest="output_dataset_one", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parsed_args = vars(_parser.parse_args())

        _outputs = preprocess(**_parsed_args)
      - --uri
      - '{{$.inputs.parameters[''uri'']}}'
      - --some-int
      - '{{$.inputs.parameters[''some_int'']}}'
      - --output-parameter-one
      - '{{$.outputs.parameters[''output_parameter_one''].output_file}}'
      - --output-dataset-one
      - '{{$.outputs.artifacts[''output_dataset_one''].path}}'
      command: [/kfp-launcher/launch, --mlmd_server_address, $(METADATA_GRPC_SERVICE_HOST),
        --mlmd_server_port, $(METADATA_GRPC_SERVICE_PORT), --runtime_info_json, $(KFP_V2_RUNTIME_INFO),
        --container_image, $(KFP_V2_IMAGE), --task_name, preprocess, --pipeline_name,
        '{{inputs.parameters.pipeline-name}}', --pipeline_run_id, $(WORKFLOW_ID),
        --pipeline_task_id, $(KFP_POD_NAME), --pipeline_root, '{{inputs.parameters.pipeline-root}}',
        --, some_int=12, uri=uri-to-import, --]
      env:
      - name: KFP_POD_NAME
        valueFrom:
          fieldRef: {fieldPath: metadata.name}
      - name: KFP_NAMESPACE
        valueFrom:
          fieldRef: {fieldPath: metadata.namespace}
      - name: WORKFLOW_ID
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''workflows.argoproj.io/workflow'']'}
      - name: ENABLE_CACHING
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''pipelines.kubeflow.org/enable_caching'']'}
      - {name: KFP_V2_IMAGE, value: 'python:3.11'}
      - {name: KFP_V2_RUNTIME_INFO, value: '{"inputParameters": {"some_int": {"type":
          "INT"}, "uri": {"type": "STRING"}}, "inputArtifacts": {}, "outputParameters":
          {"output_parameter_one": {"type": "INT", "path": "/tmp/outputs/output_parameter_one/data"}},
          "outputArtifacts": {"output_dataset_one": {"schemaTitle": "system.Dataset",
          "instanceSchema": "", "metadataPath": "/tmp/outputs/output_dataset_one/data"}}}'}
      envFrom:
      - configMapRef: {name: metadata-grpc-configmap, optional: true}
      image: python:3.11
      volumeMounts:
      - {mountPath: /kfp-launcher, name: kfp-launcher}
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
    outputs:
      parameters:
      - name: preprocess-output_parameter_one
        valueFrom: {path: /tmp/outputs/output_parameter_one/data}
      artifacts:
      - {name: preprocess-output_dataset_one, path: /tmp/outputs/output_dataset_one/data}
      - {name: preprocess-output_parameter_one, path: /tmp/outputs/output_parameter_one/data}
    metadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/component_ref: '{}'
        pipelines.kubeflow.org/arguments.parameters: '{"some_int": "12", "uri": "uri-to-import"}'
      labels:
        pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
        pipelines.kubeflow.org/pipeline-sdk-type: kfp
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/enable_caching: "true"
    initContainers:
    - command: [/bin/mount_launcher.sh]
      image: gcr.io/ml-pipeline/kfp-launcher:1.6.4
      name: kfp-launcher
      mirrorVolumeMounts: true
    volumes:
    - {name: kfp-launcher}
  - name: train-op
    container:
      args:
      - sh
      - -ec
      - |
        program_path=$(mktemp)
        printf "%s" "$0" > "$program_path"
        python3 -u "$program_path" "$@"
      - |
        def _make_parent_dirs_and_return_path(file_path: str):
            import os
            os.makedirs(os.path.dirname(file_path), exist_ok=True)
            return file_path

        def train_op(
            dataset,
            model,
            num_steps = 100
        ):
            '''Dummy Training Step.'''

            with open(dataset, 'r') as input_file:
                input_string = input_file.read()
                with open(model, 'w') as output_file:
                    for i in range(num_steps):
                        output_file.write(
                            "Step {}\n{}\n=====\n".format(i, input_string)
                        )

        import argparse
        _parser = argparse.ArgumentParser(prog='Train op', description='Dummy Training Step.')
        _parser.add_argument("--dataset", dest="dataset", type=str, required=True, default=argparse.SUPPRESS)
        _parser.add_argument("--num-steps", dest="num_steps", type=int, required=False, default=argparse.SUPPRESS)
        _parser.add_argument("--model", dest="model", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
        _parsed_args = vars(_parser.parse_args())

        _outputs = train_op(**_parsed_args)
      - --dataset
      - '{{$.inputs.artifacts[''dataset''].path}}'
      - --num-steps
      - '{{$.inputs.parameters[''num_steps'']}}'
      - --model
      - '{{$.outputs.artifacts[''model''].path}}'
      command: [/kfp-launcher/launch, --mlmd_server_address, $(METADATA_GRPC_SERVICE_HOST),
        --mlmd_server_port, $(METADATA_GRPC_SERVICE_PORT), --runtime_info_json, $(KFP_V2_RUNTIME_INFO),
        --container_image, $(KFP_V2_IMAGE), --task_name, train-op, --pipeline_name,
        '{{inputs.parameters.pipeline-name}}', --pipeline_run_id, $(WORKFLOW_ID),
        --pipeline_task_id, $(KFP_POD_NAME), --pipeline_root, '{{inputs.parameters.pipeline-root}}',
        --, 'num_steps={{inputs.parameters.preprocess-output_parameter_one}}', --]
      env:
      - name: KFP_POD_NAME
        valueFrom:
          fieldRef: {fieldPath: metadata.name}
      - name: KFP_NAMESPACE
        valueFrom:
          fieldRef: {fieldPath: metadata.namespace}
      - name: WORKFLOW_ID
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''workflows.argoproj.io/workflow'']'}
      - name: ENABLE_CACHING
        valueFrom:
          fieldRef: {fieldPath: 'metadata.labels[''pipelines.kubeflow.org/enable_caching'']'}
      - {name: KFP_V2_IMAGE, value: 'python:3.11'}
      - {name: KFP_V2_RUNTIME_INFO, value: '{"inputParameters": {"num_steps": {"type":
          "INT"}}, "inputArtifacts": {"dataset": {"metadataPath": "/tmp/inputs/dataset/data",
          "schemaTitle": "system.Dataset", "instanceSchema": ""}}, "outputParameters":
          {}, "outputArtifacts": {"model": {"schemaTitle": "system.Model", "instanceSchema":
          "", "metadataPath": "/tmp/outputs/model/data"}}}'}
      envFrom:
      - configMapRef: {name: metadata-grpc-configmap, optional: true}
      image: python:3.11
      volumeMounts:
      - {mountPath: /kfp-launcher, name: kfp-launcher}
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
      - {name: preprocess-output_parameter_one}
      artifacts:
      - {name: preprocess-output_dataset_one, path: /tmp/inputs/dataset/data}
    outputs:
      artifacts:
      - {name: train-op-model, path: /tmp/outputs/model/data}
    metadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/component_ref: '{}'
        pipelines.kubeflow.org/arguments.parameters: '{"num_steps": "{{inputs.parameters.preprocess-output_parameter_one}}"}'
      labels:
        pipelines.kubeflow.org/kfp_sdk_version: 1.6.4
        pipelines.kubeflow.org/pipeline-sdk-type: kfp
        pipelines.kubeflow.org/v2_component: "true"
        pipelines.kubeflow.org/enable_caching: "true"
    initContainers:
    - command: [/bin/mount_launcher.sh]
      image: gcr.io/ml-pipeline/kfp-launcher:1.6.4
      name: kfp-launcher
      mirrorVolumeMounts: true
    volumes:
    - {name: kfp-launcher}
  - name: two-step-pipeline
    inputs:
      parameters:
      - {name: pipeline-name}
      - {name: pipeline-root}
    dag:
      tasks:
      - name: preprocess
        template: preprocess
        arguments:
          parameters:
          - {name: pipeline-name, value: '{{inputs.parameters.pipeline-name}}'}
          - {name: pipeline-root, value: '{{inputs.parameters.pipeline-root}}'}
      - name: train-op
        template: train-op
        dependencies: [preprocess]
        arguments:
          parameters:
          - {name: pipeline-name, value: '{{inputs.parameters.pipeline-name}}'}
          - {name: pipeline-root, value: '{{inputs.parameters.pipeline-root}}'}
          - {name: preprocess-output_parameter_one, value: '{{tasks.preprocess.outputs.parameters.preprocess-output_parameter_one}}'}
          artifacts:
          - {name: preprocess-output_dataset_one, from: '{{tasks.preprocess.outputs.artifacts.preprocess-output_dataset_one}}'}
  arguments:
    parameters:
    - {name: pipeline-root, value: ''}
    - {name: pipeline-name, value: two-step-pipeline}
  serviceAccountName: pipeline-runner
`

	complexPipeline = `
# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: tfmataxicabclassificationpipelineexample-
spec:
  arguments:
    parameters:
    - name: output
    - name: project
    - name: schema
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json
    - name: train
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv
    - name: evaluation
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv
    - name: preprocess-mode
      value: local
    - name: preprocess-module
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py
    - name: target
      value: tips
    - name: learning-rate
      value: '0.1'
    - name: hidden-layer-size
      value: '1500'
    - name: steps
      value: '3000'
    - name: workers
      value: '0'
    - name: pss
      value: '0'
    - name: predict-mode
      value: local
    - name: analyze-mode
      value: local
    - name: analyze-slice-column
      value: trip_start_hour
  entrypoint: tfmataxicabclassificationpipelineexample
  templates:
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/analysis'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.analyze-mode}}'
      - --slice-columns
      - '{{inputs.parameters.analyze-slice-column}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: output
      - name: project
      - name: schema
      - name: training-train
    name: analysis
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: seaweedfs.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: analysis-analysis
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/predict'
      - --data
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --mode
      - '{{inputs.parameters.predict-mode}}'
      - --project
      - '{{inputs.parameters.project}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: predict-mode
      - name: project
      - name: schema
      - name: target
      - name: training-train
    name: prediction
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: seaweedfs.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: prediction-predict
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --train
      - '{{inputs.parameters.train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/transformed'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.preprocess-mode}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tft
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: schema
      - name: train
    name: preprocess
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: seaweedfs.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: preprocess-transformed
        valueFrom:
          path: /output.txt
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: analyze-mode
            value: '{{inputs.parameters.analyze-mode}}'
          - name: analyze-slice-column
            value: '{{inputs.parameters.analyze-slice-column}}'
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: analysis
        template: analysis
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: predict-mode
            value: '{{inputs.parameters.predict-mode}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: prediction
        template: prediction
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-mode
            value: '{{inputs.parameters.preprocess-mode}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: train
            value: '{{inputs.parameters.train}}'
        name: preprocess
        template: preprocess
      - arguments:
          parameters:
          - name: hidden-layer-size
            value: '{{inputs.parameters.hidden-layer-size}}'
          - name: learning-rate
            value: '{{inputs.parameters.learning-rate}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: preprocess-transformed
            value: '{{tasks.preprocess.outputs.parameters.preprocess-transformed}}'
          - name: pss
            value: '{{inputs.parameters.pss}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: steps
            value: '{{inputs.parameters.steps}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: workers
            value: '{{inputs.parameters.workers}}'
        dependencies:
        - preprocess
        name: training
        template: training
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: predict-mode
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: train
      - name: workers
    name: tfmataxicabclassificationpipelineexample
  - container:
      args:
      - --job-dir
      - '{{inputs.parameters.output}}/{{workflow.name}}/train'
      - --transformed-data-dir
      - '{{inputs.parameters.preprocess-transformed}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --learning-rate
      - '{{inputs.parameters.learning-rate}}'
      - --hidden-layer-size
      - '{{inputs.parameters.hidden-layer-size}}'
      - --steps
      - '{{inputs.parameters.steps}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --workers
      - '{{inputs.parameters.workers}}'
      - --pss
      - '{{inputs.parameters.pss}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      - --tfjob-timeout-minutes
      - '60'
      image: gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf
    inputs:
      parameters:
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: preprocess-module
      - name: preprocess-transformed
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: workers
    name: training
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: seaweedfs.kubeflow:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: training-train
        valueFrom:
          path: /output.txt`
)

func TestCreateDefaultExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experimentID, err := manager.CreateDefaultExperiment("")
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateDefaultExperiment_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experimentID, err := manager.CreateDefaultExperiment("multi-user")
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "multi-user",
		StorageState:   "AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateTask(t *testing.T) {
	_, manager, _, _, _, runDetail := initWithExperimentAndPipelineAndRun(t)
	task := &model.Task{
		Namespace:         "",
		PipelineName:      "pipeline/my-pipeline",
		RunID:             runDetail.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1462875553,
		FinishedTimestamp: 1462875663,
		Fingerprint:       "123",
	}

	expectedTask := &model.Task{
		UUID:              DefaultFakeUUID,
		PipelineName:      "pipeline/my-pipeline",
		RunID:             runDetail.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1462875553,
		FinishedTimestamp: 1462875663,
		Fingerprint:       "123",
	}
	createdTask, err := manager.CreateTask(task)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, createdTask, "The CreateTask return has unexpected value")

	// Verify the T in DB is in status PipelineVersionCreating.
	storedTask, err := manager.taskStore.GetTask(DefaultFakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, expectedTask, storedTask, "The StoredTask return has unexpected value")
}

var v2SpecHelloWorld = `
components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        text:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - "--text"
        - "{{$.inputs.parameters['text']}}"
        command:
        - sh
        - "-ec"
        - |
          program_path=$(mktemp)
          printf "%s" "$0" > "$program_path"
          python3 -u "$program_path" "$@"
        - |
          def hello_world(text):
              print(text)
              return text

          import argparse
          _parser = argparse.ArgumentParser(prog='Hello world', description='')
          _parser.add_argument("--text", dest="text", type=str, required=True, default=argparse.SUPPRESS)
          _parsed_args = vars(_parser.parse_args())

          _outputs = hello_world(**_parsed_args)
        image: python:3.11
pipelineInfo:
  name: hello-world
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            text:
              componentInputParameter: text
        taskInfo:
          name: hello-world
  inputDefinitions:
    parameters:
      text:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

var v2SpecHelloWorldMutated = `
components:
  comp-hello-world:
    executorLabel: exec-hello-world
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        image: python:3.11
pipelineInfo:
  name: pipelines/p1/versions/v1
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        taskInfo:
          name: hello-world
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

// v2SpecWithLiterals is a v2 pipeline spec with literal parameter constraints for testing.
var v2SpecWithLiterals = `
components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        environment:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - "--env"
        - "{{$.inputs.parameters['environment']}}"
        command:
        - echo
        image: python:3.11
pipelineInfo:
  name: hello-world-with-literals
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            environment:
              componentInputParameter: environment
        taskInfo:
          name: hello-world
  inputDefinitions:
    parameters:
      environment:
        parameterType: STRING
        literals:
        - "dev"
        - "staging"
        - "prod"
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

// v2SpecWithIntLiterals is a v2 pipeline spec with integer literal parameter constraints for testing.
var v2SpecWithIntLiterals = `
components:
  comp-test:
    executorLabel: exec-test
    inputDefinitions:
      parameters:
        replicas:
          parameterType: NUMBER_INTEGER
deploymentSpec:
  executors:
    exec-test:
      container:
        image: python:3.11
pipelineInfo:
  name: test-int-literals
root:
  dag:
    tasks:
      test-task:
        componentRef:
          name: comp-test
        inputs:
          parameters:
            replicas:
              componentInputParameter: replicas
        taskInfo:
          name: test-task
  inputDefinitions:
    parameters:
      replicas:
        parameterType: NUMBER_INTEGER
        literals:
        - 1
        - 3
        - 5
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

// v2SpecWithFloatLiterals is a v2 pipeline spec with float literal parameter constraints for testing.
var v2SpecWithFloatLiterals = `
components:
  comp-test:
    executorLabel: exec-test
    inputDefinitions:
      parameters:
        threshold:
          parameterType: NUMBER_DOUBLE
deploymentSpec:
  executors:
    exec-test:
      container:
        image: python:3.11
pipelineInfo:
  name: test-float-literals
root:
  dag:
    tasks:
      test-task:
        componentRef:
          name: comp-test
        inputs:
          parameters:
            threshold:
              componentInputParameter: threshold
        taskInfo:
          name: test-task
  inputDefinitions:
    parameters:
      threshold:
        parameterType: NUMBER_DOUBLE
        literals:
        - 0.1
        - 0.5
        - 0.9
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

// v2SpecWithBoolLiterals is a v2 pipeline spec with boolean literal parameter constraints for testing.
var v2SpecWithBoolLiterals = `
components:
  comp-test:
    executorLabel: exec-test
    inputDefinitions:
      parameters:
        enable_feature:
          parameterType: BOOLEAN
deploymentSpec:
  executors:
    exec-test:
      container:
        image: python:3.11
pipelineInfo:
  name: test-bool-literals
root:
  dag:
    tasks:
      test-task:
        componentRef:
          name: comp-test
        inputs:
          parameters:
            enable_feature:
              componentInputParameter: enable_feature
        taskInfo:
          name: test-task
  inputDefinitions:
    parameters:
      enable_feature:
        parameterType: BOOLEAN
        literals:
        - true
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`

func TestCreateRun_LiteralParameterValidation(t *testing.T) {
	tests := []struct {
		name          string
		pipelineSpec  string
		runtimeParams string
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid input - string literal",
			pipelineSpec:  v2SpecWithLiterals,
			runtimeParams: `{"environment":"dev"}`,
			expectError:   false,
		},
		{
			name:          "invalid input - string literal",
			pipelineSpec:  v2SpecWithLiterals,
			runtimeParams: `{"environment":"test"}`,
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name:          "valid input - int literal",
			pipelineSpec:  v2SpecWithIntLiterals,
			runtimeParams: `{"replicas":3}`,
			expectError:   false,
		},
		{
			name:          "invalid input - int literal",
			pipelineSpec:  v2SpecWithIntLiterals,
			runtimeParams: `{"replicas":2}`,
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name:          "valid input - float literal",
			pipelineSpec:  v2SpecWithFloatLiterals,
			runtimeParams: `{"threshold":0.5}`,
			expectError:   false,
		},
		{
			name:          "invalid input - float literal",
			pipelineSpec:  v2SpecWithFloatLiterals,
			runtimeParams: `{"threshold":0.3}`,
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name:          "valid input - boolean literal",
			pipelineSpec:  v2SpecWithBoolLiterals,
			runtimeParams: `{"enable_feature":true}`,
			expectError:   false,
		},
		{
			name:          "invalid input - boolean literal",
			pipelineSpec:  v2SpecWithBoolLiterals,
			runtimeParams: `{"enable_feature":false}`,
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name:          "valid input - nil literals field",
			pipelineSpec:  v2SpecHelloWorld, // No literals field
			runtimeParams: `{"text":"any-value-is-fine"}`,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, manager, exp := initWithExperiment(t)
			defer store.Close()
			apiRun := &model.Run{
				DisplayName:  "run1",
				ExperimentId: exp.UUID,
				PipelineSpec: model.PipelineSpec{
					PipelineSpecManifest: model.LargeText(tt.pipelineSpec),
					RuntimeConfig: model.RuntimeConfig{
						Parameters: model.LargeText(tt.runtimeParams),
					},
				},
			}
			_, err := manager.CreateRun(context.Background(), apiRun)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.ErrorContains(t, err, tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name    string
		tags    map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil tags",
			tags:    nil,
			wantErr: false,
		},
		{
			name:    "empty tags",
			tags:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "valid single tag",
			tags:    map[string]string{"env": "prod"},
			wantErr: false,
		},
		{
			name: "valid max tags",
			tags: func() map[string]string {
				m := make(map[string]string)
				for i := 0; i < MaxTagsPerEntity; i++ {
					m[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
				}
				return m
			}(),
			wantErr: false,
		},
		{
			name: "exceeds max tags",
			tags: func() map[string]string {
				m := make(map[string]string)
				for i := 0; i <= MaxTagsPerEntity; i++ {
					m[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
				}
				return m
			}(),
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name:    "empty key",
			tags:    map[string]string{"": "value"},
			wantErr: true,
			errMsg:  "tag key cannot be empty",
		},
		{
			name:    "key with dot",
			tags:    map[string]string{"team.name": "ml"},
			wantErr: true,
			errMsg:  "must not contain '.'",
		},
		{
			name:    "key too long",
			tags:    map[string]string{strings.Repeat("k", MaxTagKeyLength+1): "v"},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "key at max length",
			tags:    map[string]string{strings.Repeat("k", MaxTagKeyLength): "v"},
			wantErr: false,
		},
		{
			name:    "value too long",
			tags:    map[string]string{"key": strings.Repeat("v", MaxTagValueLength+1)},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "value at max length",
			tags:    map[string]string{"key": strings.Repeat("v", MaxTagValueLength)},
			wantErr: false,
		},
		{
			name:    "unicode key at max rune length",
			tags:    map[string]string{strings.Repeat("日", MaxTagKeyLength): "v"},
			wantErr: false,
		},
		{
			name: "unicode key exceeds max rune length",
			tags: func() map[string]string {
				k := strings.Repeat("日", MaxTagKeyLength+1)
				assert.Greater(t, utf8.RuneCountInString(k), MaxTagKeyLength)
				return map[string]string{k: "v"}
			}(),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "empty value is valid",
			tags:    map[string]string{"key": ""},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.ValidateTags(tt.tags)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCreateRun_IdempotentFromRecurringRun(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	// Pre-create a run as if it was already submitted for this recurring run trigger.
	// This simulates a race where one replica already persisted the run.
	preExistingRun := &model.Run{
		UUID:           "pre-existing-run-uuid",
		DisplayName:    "scheduled-run-trigger-1",
		RecurringRunId: job.UUID,
		ExperimentId:   job.ExperimentId,
		K8SName:        "pre-existing-k8s-name",
		StorageState:   model.StorageStateAvailable,
		PipelineSpec:   job.PipelineSpec,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			State:                   model.RuntimeStatePending,
			WorkflowRuntimeManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
	}
	_, err := manager.runStore.CreateRun(preExistingRun)
	require.Nil(t, err)

	// A second CreateRun with the same RecurringRunId + DisplayName should return
	// the existing run without submitting any new Argo Workflow.
	duplicateRun := &model.Run{
		DisplayName:    "scheduled-run-trigger-1",
		RecurringRunId: job.UUID,
		ExperimentId:   job.ExperimentId,
		PipelineSpec:   job.PipelineSpec,
	}
	returned, err := manager.CreateRun(context.Background(), duplicateRun)
	assert.Nil(t, err)
	assert.Equal(t, "pre-existing-run-uuid", returned.UUID, "should return existing run, not create a new one")
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowCount(), "no new Argo Workflow should be submitted")
}

func TestCreateRun_DeterministicUUIDFromRecurringRun(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	run := &model.Run{
		DisplayName:    "scheduled-run-trigger-1",
		RecurringRunId: job.UUID,
		ExperimentId:   job.ExperimentId,
		PipelineSpec:   job.PipelineSpec,
	}
	created, err := manager.CreateRun(context.Background(), run)
	require.Nil(t, err)

	// The run ID is derived deterministically from the recurring run ID and display
	// name, so concurrent triggers converge on the same primary key.
	wantUUID := util.NewDeterministicUUID(job.UUID + "/scheduled-run-trigger-1")
	assert.Equal(t, wantUUID, created.UUID)
}

// A terminal report carrying no (or an older) retry-generation annotation is a
// snapshot of the pre-retry workflow. While the claim is fresh it must be
// skipped as a successful no-op: not an error (the persistence agent would
// requeue forever) and not a write (it would restore a GC-eligible
// FinishedAtInSec while the retry is running).
func TestReportWorkflowResource_SkipsStaleTerminalReportDuringRetryClaim(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	staleWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, staleWorkflow)

	// Drive the run terminal, then claim it for retry.
	_, err := manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	require.Nil(t, err)
	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.Nil(t, claimErr)
	require.Equal(t, int64(1), claimGeneration)

	// The same terminal snapshot arrives again (requeued by the persistence
	// agent). It carries no retry-generation annotation.
	_, err = manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	assert.Nil(t, err, "stale terminal report during a fresh claim must be a successful no-op")

	claimed, err := manager.GetRun(run.UUID)
	require.Nil(t, err)
	assert.Equal(t, model.RuntimeStatePending, claimed.State, "stale report must not overwrite the claimed row")
	assert.Equal(t, int64(0), claimed.FinishedAtInSec)
}

// A report from the retried workflow itself carries the claim's generation in
// its annotation and must pass the fence.
func TestReportWorkflowResource_AcceptsRetriedWorkflowReport(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	terminal := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, terminal)
	_, err := manager.ReportWorkflowResource(context.Background(), terminal)
	require.Nil(t, err)
	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.Nil(t, claimErr)

	retried := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			Annotations: map[string]string{
				util.AnnotationKeyRetryGeneration: strconv.FormatInt(claimGeneration, 10),
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowSucceeded},
	})
	syncWorkflowReportWithFakeCluster(t, store, retried)
	_, err = manager.ReportWorkflowResource(context.Background(), retried)
	assert.Nil(t, err)

	updated, err := manager.GetRun(run.UUID)
	require.Nil(t, err)
	assert.Equal(t, model.RuntimeStateSucceeded, updated.State, "post-retry completion must pass the generation fence")
}

// A claim with no claim timestamp (cleared by rollback, or orphaned by a crash
// past the grace period) must not fence terminal reports forever: the reporter
// accepts the terminal state so the run self-heals instead of staying PENDING.
func TestReportWorkflowResource_RecoversOrphanedRetryClaim(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	staleWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, staleWorkflow)
	_, err := manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	require.Nil(t, err)
	_, _, _, _, claimErr := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.Nil(t, claimErr)

	// Simulate an orphaned claim: the claim timestamp is gone (rollback) or
	// far in the past (crash between claim and workflow update).
	_, err = store.DB().Exec(`UPDATE run_details SET RetryClaimedAtInSec = 0 WHERE UUID = ?`, run.UUID)
	require.Nil(t, err)

	_, err = manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	assert.Nil(t, err)

	recovered, err := manager.GetRun(run.UUID)
	require.Nil(t, err)
	assert.Equal(t, model.RuntimeStateFailed, recovered.State, "orphaned claim must self-heal to the last real terminal state")
}

// RetryRun must stamp the claim's RetryGeneration on the retried workflow so
// ReportWorkflowResource can tell its reports apart from stale snapshots of
// the pre-retry workflow.
func TestRetryRun_StampsRetryGenerationAnnotation(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	err := manager.RetryRun(context.Background(), runDetail.UUID)
	require.Nil(t, err)

	retried, err := manager.GetRun(runDetail.UUID)
	require.Nil(t, err)
	assert.Equal(t, int64(1), retried.RetryGeneration)
	assert.Contains(t, string(retried.WorkflowRuntimeManifest), util.AnnotationKeyRetryGeneration,
		"retried workflow manifest must carry the retry-generation annotation")
}

// Regression: when the workflow mutation errors but was actually applied (the
// live workflow carries this claim's retry-generation annotation), RetryRun
// must adopt the live workflow — even if it already reached a terminal state —
// instead of rolling back the claim, which would restore a GC-eligible
// FinishedAtInSec under a live retried workflow and permit a duplicate retry.
func TestRetryRun_AdoptsAppliedWorkflowInsteadOfRollingBack(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	// Seed the retried workflow as already terminal and carrying the
	// generation the upcoming claim will produce (1), simulating a timed-out
	// update that was applied and a retry that finished quickly.
	run, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.NoError(t, execSpec.Decompress())
	retryExecSpec, _, err := execSpec.GenerateRetryExecution()
	require.NoError(t, err)
	retryExecSpec.SetAnnotations(util.AnnotationKeyRetryGeneration, "1")
	appliedWorkflow := retryExecSpec.(*util.Workflow)
	appliedWorkflow.Status.Phase = v1alpha1.WorkflowSucceeded
	appliedWorkflow.Status.FinishedAt = v1.Time{Time: time.Unix(500, 0)}

	workflowClient := client.NewWorkflowClientFake()
	_, err = workflowClient.Create(context.Background(), appliedWorkflow, v1.CreateOptions{})
	require.NoError(t, err)
	conflictClient := &persistentConflictWorkflowClient{FakeWorkflowClient: workflowClient}
	manager.execClient = &retryWorkflowExecClient{workflowClient: conflictClient}

	err = manager.RetryRun(context.Background(), runDetail.UUID)
	require.NoError(t, err, "an applied retry must be adopted, not treated as failed")

	adopted, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateSucceeded, adopted.State, "run must reflect the adopted live workflow")
	assert.Equal(t, int64(1), adopted.RetryGeneration, "claim must not be rolled back")
	assert.Equal(t, int64(500), adopted.FinishedAtInSec, "adopted terminal workflow's finish time must be persisted")
}

// Regression: an expired retry claim is not necessarily abandoned. When the
// previous claim's workflow is live (the earlier API server crashed after
// applying the mutation but before persisting), a new RetryRun must adopt it
// rather than take over the claim and restart in-flight work.
func TestRetryRun_ExpiredClaimWithLiveWorkflowIsAdoptedNotTakenOver(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()
	expectedWorkflowName := runDetail.K8SName

	// Simulate a retry that crashed after claiming generation 1 and creating
	// the workflow, but before persisting the run row: claim via the store
	// (leaves the terminal manifest untouched), seed the live generation-1
	// workflow, and age out the claim timestamp.
	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(runDetail.UUID, false)
	require.NoError(t, claimErr)
	require.Equal(t, int64(1), claimGeneration)
	_, err := store.DB().Exec(
		`UPDATE run_details SET RetryClaimedAtInSec = 0, Name = ? WHERE UUID = ?`,
		"stale-workflow-name", runDetail.UUID)
	require.NoError(t, err)

	run, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.NoError(t, execSpec.Decompress())
	retryExecSpec, _, err := execSpec.GenerateRetryExecution()
	require.NoError(t, err)
	retryExecSpec.SetAnnotations(util.AnnotationKeyRetryGeneration, "1")
	workflowClient := client.NewWorkflowClientFake()
	_, err = workflowClient.Create(context.Background(), retryExecSpec, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}
	manager.storedWorkflowIdentities.loadOrStore(runDetail.UUID, storedWorkflowIdentity{
		name: runDetail.K8SName,
		uid:  storedWorkflowUID(t, run),
	})

	require.NoError(t, manager.RetryRun(context.Background(), runDetail.UUID))
	_, found := manager.storedWorkflowIdentities.load(runDetail.UUID)
	assert.False(t, found, "adopting a live retry must invalidate the previous workflow identity")

	adopted, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), adopted.RetryGeneration,
		"live generation-1 workflow must be adopted; a takeover to generation 2 would restart in-flight work")
	assert.NotEqual(t, model.RuntimeStatePending, adopted.State, "adoption must persist the live workflow's state")
	assert.Equal(t, expectedWorkflowName, adopted.K8SName)

	liveWorkflow, err := workflowClient.Get(context.Background(), expectedWorkflowName, v1.GetOptions{})
	require.NoError(t, err)
	_, err = manager.ReportWorkflowResource(context.Background(), liveWorkflow)
	require.NoError(t, err, "reports for the adopted workflow must use the repaired run name")
}

// Regression: takeover happens only when the previous claim's workflow is
// definitively absent.
func TestRetryRun_ExpiredClaimWithoutWorkflowIsTakenOver(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	// Claim generation 1 but never create the workflow (crash before create).
	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(runDetail.UUID, false)
	require.NoError(t, claimErr)
	require.Equal(t, int64(1), claimGeneration)
	_, err := store.DB().Exec(`UPDATE run_details SET RetryClaimedAtInSec = 0 WHERE UUID = ?`, runDetail.UUID)
	require.NoError(t, err)

	// No workflow exists in the fake client: absence is definitive.
	manager.execClient = &retryWorkflowExecClient{workflowClient: client.NewWorkflowClientFake()}
	require.NoError(t, manager.RetryRun(context.Background(), runDetail.UUID))

	recovered, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recovered.RetryGeneration, "definitive absence must allow the takeover claim")
}

type retryHookCountingDispatcher struct {
	onRunRetryCalls int
}

func (d *retryHookCountingDispatcher) OnBeforeRunCreation(context.Context, *apiserverPlugins.PendingRun, util.ExecutionSpec) error {
	return nil
}

func (d *retryHookCountingDispatcher) OnRunEnd(context.Context, *apiserverPlugins.PersistedRun) bool {
	return true
}

func (d *retryHookCountingDispatcher) OnRunRetry(context.Context, *apiserverPlugins.PersistedRun) error {
	d.onRunRetryCalls++
	return nil
}

func (d *retryHookCountingDispatcher) PluginsRegistered() bool {
	return true
}

// Regression: the resource-version check passes vacuously for reports without
// a resourceVersion, so the age-based accept path must never accept a
// lower-generation terminal report — or delete the workflow via the
// persisted-final-state path — while the claimed generation is live.
func TestReportWorkflowResource_NeverAgeAcceptsWhileClaimedGenerationLive(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	staleWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	syncWorkflowReportWithFakeCluster(t, store, staleWorkflow)
	_, err := manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	require.Nil(t, err)
	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(run.UUID, false)
	require.Nil(t, claimErr)
	require.Equal(t, int64(1), claimGeneration)

	// Age out the claim, and make the live workflow carry the claimed
	// generation (the retried workflow was applied by a crashed retry).
	_, err = store.DB().Exec(`UPDATE run_details SET RetryClaimedAtInSec = 0 WHERE UUID = ?`, run.UUID)
	require.Nil(t, err)
	wfClient := store.ExecClientFake.Execution("ns1")
	liveWorkflow, err := wfClient.Get(context.Background(), run.K8SName, v1.GetOptions{})
	require.Nil(t, err)
	liveWorkflow.SetAnnotations(util.AnnotationKeyRetryGeneration, "1")
	delete(liveWorkflow.ExecutionObjectMeta().Labels, util.LabelKeyWorkflowPersistedFinalState)
	liveWorkflow.(*util.Workflow).Status.Phase = v1alpha1.WorkflowRunning
	_, err = wfClient.Update(context.Background(), liveWorkflow, v1.UpdateOptions{})
	require.Nil(t, err)

	// The stale snapshot (no annotation, empty resourceVersion) arrives with
	// an aged-out claim: it must be skipped, not age-accepted.
	_, err = manager.ReportWorkflowResource(context.Background(), staleWorkflow)
	assert.Nil(t, err)
	claimed, err := manager.GetRun(run.UUID)
	require.Nil(t, err)
	assert.Equal(t, model.RuntimeStateRunning, claimed.State,
		"the live claimed generation must win over the stale terminal report")

	// A stale snapshot carrying persistedFinalState must not delete the
	// live retried workflow either: the fence runs before the deletion.
	staleFinal := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       liveWorkflow.ExecutionObjectMeta().UID,
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})
	_, err = manager.ReportWorkflowResource(context.Background(), staleFinal)
	assert.Nil(t, err)
	_, err = wfClient.Get(context.Background(), run.K8SName, v1.GetOptions{})
	assert.Nil(t, err, "the live retried workflow must not be deleted by a stale persisted-final-state snapshot")
}

// Regression: expired-claim adoption must fire the plugin retry hook; the
// crashed retry never reached plugin notification, and skipping it leaves
// plugin-side (e.g. MLflow) runs terminal while the KFP retry runs.
func TestRetryRun_AdoptionFiresPluginRetryHook(t *testing.T) {
	store, manager, runDetail := initWithOneTimeFailedRun(t)
	defer store.Close()

	_, _, _, claimGeneration, claimErr := store.RunStore().ClaimRunForRetry(runDetail.UUID, false)
	require.NoError(t, claimErr)
	require.Equal(t, int64(1), claimGeneration)
	_, err := store.DB().Exec(`UPDATE run_details SET RetryClaimedAtInSec = 0, PluginsOutput = ? WHERE UUID = ?`,
		`{"mlflow":{"runId":"abc"}}`, runDetail.UUID)
	require.NoError(t, err)

	run, err := manager.GetRun(runDetail.UUID)
	require.NoError(t, err)
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.NoError(t, execSpec.Decompress())
	retryExecSpec, _, err := execSpec.GenerateRetryExecution()
	require.NoError(t, err)
	retryExecSpec.SetAnnotations(util.AnnotationKeyRetryGeneration, "1")
	workflowClient := client.NewWorkflowClientFake()
	_, err = workflowClient.Create(context.Background(), retryExecSpec, v1.CreateOptions{})
	require.NoError(t, err)
	manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}
	dispatcher := &retryHookCountingDispatcher{}
	manager.pluginDispatcher = dispatcher

	require.NoError(t, manager.RetryRun(context.Background(), runDetail.UUID))
	assert.Equal(t, 1, dispatcher.onRunRetryCalls, "adoption must notify plugins exactly like a normal retry")
}

// Regression: a transient run-store read failure must fail closed — it must
// not skip the generation fence and fall through into persisted-final-state
// workflow deletion, which would delete the live retried workflow on the
// strength of a stale snapshot.
func TestReportWorkflowResource_RunStoreErrorFailsClosedBeforeDeletion(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	staleFinal := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: "ns1",
			UID:       types.UID(run.UUID),
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowFailed},
	})

	// Force GetRun to fail with a non-NotFound error.
	store.Close()

	_, err := manager.ReportWorkflowResource(context.Background(), staleFinal)
	require.Error(t, err)
	assert.False(t, util.IsUserErrorCodeMatch(err, codes.NotFound),
		"a run-store read failure must surface as an error, not be treated as run-not-found")
	assert.Contains(t, err.Error(), "before applying workflow report",
		"the error must come from the fail-closed guard, before the deletion path")

	// The decisive assertion: the workflow must still exist. The fake exec
	// client is independent of the closed database, so a deletion would
	// have gone through and be visible here.
	_, err = store.ExecClientFake.Execution("ns1").Get(context.Background(), run.K8SName, v1.GetOptions{})
	assert.Nil(t, err, "the workflow must not be deleted when the run-store read fails")
}

// --- ServiceAccount SAR authorization tests ---

func multiUserContext() context.Context {
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	return metadata.NewIncomingContext(context.Background(), md)
}

func initWithExperimentAndUnauthorizedSAR(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	store.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1", Namespace: "ns1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	require.Nil(t, err)
	return store, manager, experiment
}

func TestCreateRun_ServiceAccountSAR_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestCreateRun_ServiceAccountSAR_MultiUserAuthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperiment(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	run, err := manager.CreateRun(multiUserContext(), apiRun)
	require.Nil(t, err)
	assert.Equal(t, "custom-sa", run.ServiceAccount)
}

func TestCreateRun_ServiceAccountSAR_SingleUserSkipped(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	run, err := manager.CreateRun(context.Background(), apiRun)
	require.Nil(t, err)
	assert.Equal(t, "custom-sa", run.ServiceAccount)
}

func TestCreateRun_ServiceAccountSAR_DefaultSASkipped(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	run, err := manager.CreateRun(multiUserContext(), apiRun)
	require.Nil(t, err)
	assert.Equal(t, common.DefaultPipelineRunnerServiceAccount, run.ServiceAccount)
}

func TestCreateJob_ServiceAccountSAR_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err := manager.CreateJob(multiUserContext(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestCreateJob_ServiceAccountSAR_MultiUserAuthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperiment(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	createdJob, err := manager.CreateJob(multiUserContext(), job)
	require.Nil(t, err)
	assert.Equal(t, "custom-sa", createdJob.ServiceAccount)
}

func TestCreateJob_ServiceAccountSAR_SingleUserSkipped(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	createdJob, err := manager.CreateJob(context.Background(), job)
	require.Nil(t, err)
	assert.Equal(t, "custom-sa", createdJob.ServiceAccount)
}

func TestCreateJob_ServiceAccountSAR_DefaultSASkipped(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId: experiment.UUID,
	}
	createdJob, err := manager.CreateJob(multiUserContext(), job)
	assert.Nil(t, err)
	assert.NotNil(t, createdJob)
}

// --- V2 pipeline spec SAR tests ---

func TestCreateRun_ServiceAccountSAR_V2Spec_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestCreateJob_ServiceAccountSAR_V2Spec_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err := manager.CreateJob(multiUserContext(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

// --- Confused deputy: privileged SA name ---

func TestCreateRun_ServiceAccountSAR_ConfusedDeputy_PrivilegedSA(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "ds-pipeline-dspa",
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestCreateJob_ServiceAccountSAR_ConfusedDeputy_PrivilegedSA(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "ds-pipeline-dspa",
	}
	_, err := manager.CreateJob(multiUserContext(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

// --- SAR ResourceAttributes verification ---

type capturingSARClient struct {
	lastReview *authzv1.SubjectAccessReview
}

func (c *capturingSARClient) Create(_ context.Context, sar *authzv1.SubjectAccessReview, _ v1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	c.lastReview = sar
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: true}}, nil
}

func initWithExperimentAndCapturingSAR(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *capturingSARClient) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	capturingClient := &capturingSARClient{}
	store.SubjectAccessReviewClientFake = capturingClient
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	apiExperiment := &model.Experiment{Name: "e1", Namespace: "ns1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	require.Nil(t, err)
	return store, manager, experiment, capturingClient
}

func TestCreateRun_ServiceAccountSAR_CorrectResourceAttributes(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "my-special-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment, capturingClient := initWithExperimentAndCapturingSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "my-special-sa",
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.Nil(t, err)

	require.NotNil(t, capturingClient.lastReview)
	attrs := capturingClient.lastReview.Spec.ResourceAttributes
	assert.Equal(t, common.RbacResourceVerbUse, attrs.Verb)
	assert.Equal(t, "serviceaccounts", attrs.Resource)
	assert.Equal(t, "my-special-sa", attrs.Name)
	assert.Equal(t, "ns1", attrs.Namespace)
}

// --- CreateJob allowlist bypass tests ---

func TestCreateJob_PipelineIdOnly_DisallowedSA_Rejected(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	require.Nil(t, err)

	p, _ := manager.CreatePipeline(createPipeline("p1", "", "ns1"))
	pv := createPipelineVersion(
		p.UUID, "p1/v1", "v1", "",
		v2SpecHelloWorld,
		"", "ns1",
	)
	_, err = manager.CreatePipelineVersion(pv)
	require.Nil(t, err)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "disallowed-sa",
	}
	_, err = manager.CreateJob(context.Background(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestCreateJob_PipelineIdOnly_AllowedSA_Succeeds(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	require.Nil(t, err)

	p, _ := manager.CreatePipeline(createPipeline("p1", "", "ns1"))
	pv := createPipelineVersion(p.UUID, "p1/v1", "v1", "", v2SpecHelloWorld, "", "ns1")
	_, err = manager.CreatePipelineVersion(pv)
	require.Nil(t, err)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	createdJob, err := manager.CreateJob(context.Background(), job)
	require.Nil(t, err)
	assert.Equal(t, "custom-sa", createdJob.ServiceAccount)
}

func TestCreateJob_PipelineIdOnly_SAR_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	store.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	require.Nil(t, err)

	p, _ := manager.CreatePipeline(createPipeline("p1", "", "ns1"))
	pv := createPipelineVersion(p.UUID, "p1/v1", "v1", "", v2SpecHelloWorld, "", "ns1")
	_, err = manager.CreatePipelineVersion(pv)
	require.Nil(t, err)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineId: p.UUID,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"text\":\"world\"}",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err = manager.CreateJob(multiUserContext(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestCreateJob_ServiceAccountSAR_CorrectResourceAttributes(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "my-special-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	_, manager, experiment, capturingClient := initWithExperimentAndCapturingSAR(t)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "my-special-sa",
	}
	_, err := manager.CreateJob(multiUserContext(), job)
	require.Nil(t, err)

	require.NotNil(t, capturingClient.lastReview)
	attrs := capturingClient.lastReview.Spec.ResourceAttributes
	assert.Equal(t, common.RbacResourceVerbUse, attrs.Verb)
	assert.Equal(t, "serviceaccounts", attrs.Resource)
	assert.Equal(t, "my-special-sa", attrs.Name)
	assert.Equal(t, "ns1", attrs.Namespace)
}

func TestCreateRun_ServiceAccountSAR_DefaultSA_NotCalled(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment, capturingClient := initWithExperimentAndCapturingSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	run, err := manager.CreateRun(multiUserContext(), apiRun)
	require.Nil(t, err)
	assert.Equal(t, common.DefaultPipelineRunnerServiceAccount, run.ServiceAccount)
	assert.Nil(t, capturingClient.lastReview)
}

// --- PipelineVersionId path SA authorization ---

func TestCreateJob_PipelineVersionId_SAR_MultiUserUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	store.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})

	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	require.Nil(t, err)

	p, _ := manager.CreatePipeline(createPipeline("p1", "", "ns1"))
	pv := createPipelineVersion(p.UUID, "p1/v1", "v1", "", v2SpecHelloWorld, "", "ns1")
	version, err := manager.CreatePipelineVersion(pv)
	require.Nil(t, err)

	job := &model.Job{
		DisplayName: "j1",
		Enabled:     true,
		PipelineSpec: model.PipelineSpec{
			PipelineVersionId: version.UUID,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"text\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err = manager.CreateJob(multiUserContext(), job)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

// --- Unauthorized request must not create k8s resources ---

func TestCreateRun_ServiceAccountSAR_Unauthorized_NoWorkflowCreated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	defer viper.Set(common.AllowedServiceAccountsFlag, "")

	store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId:   experiment.UUID,
		ServiceAccount: "custom-sa",
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.NotNil(t, err)
	assert.Equal(t, 0, store.ExecClientFake.GetWorkflowCount(), "no Workflow CRD should be created when SA authorization fails")
}

// --- SA embedded in workflow spec ---

func TestCreateRun_ServiceAccountSAR_EmbeddedSA_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	_, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)

	workflowWithEmbeddedSA := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint:         "testy",
			ServiceAccountName: "evil-sa",
			Templates: []v1alpha1.Template{{
				Name: "testy",
				Container: &corev1.Container{
					Image:   "docker/whalesay",
					Command: []string{"cowsay"},
					Args:    []string{"hello world"},
				},
			}},
			Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{Name: "param1"}}},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning},
	})

	apiRun := &model.Run{
		DisplayName: "run1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(workflowWithEmbeddedSA.ToStringForStore()),
			Parameters:           "[{\"name\":\"param1\",\"value\":\"world\"}]",
		},
		ExperimentId: experiment.UUID,
	}
	_, err := manager.CreateRun(multiUserContext(), apiRun)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}
