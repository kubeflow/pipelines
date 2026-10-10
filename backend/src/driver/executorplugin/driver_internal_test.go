// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type invocationAPI struct {
	kfpapi.API
	run                            *gc.Run
	spec                           *structpb.Struct
	getRunErr, getTaskErr, specErr error
}

func (a *invocationAPI) GetRun(_ context.Context, request *gc.GetRunRequest) (*gc.Run, error) {
	return a.run, a.getRunErr
}
func (a *invocationAPI) GetTask(context.Context, *gc.GetTaskRequest) (*gc.PipelineTask, error) {
	return &gc.PipelineTask{ScopePath: "root"}, a.getTaskErr
}
func (a *invocationAPI) FetchPipelineSpecFromRun(context.Context, *gc.Run) (*structpb.Struct, error) {
	return a.spec, a.specErr
}

type invocationManager struct {
	api   kfpapi.API
	k8s   kubernetes.Interface
	close func() error
}

func (m *invocationManager) KFPAPIClient() kfpapi.API        { return m.api }
func (m *invocationManager) K8sClient() kubernetes.Interface { return m.k8s }
func (m *invocationManager) Close() error {
	if m.close != nil {
		return m.close()
	}
	return nil
}

func invocationFixture(t *testing.T, kind string) (*driverRunner, driverapi.DriverPluginArgs, *invocationAPI, *invocationManager) {
	t.Helper()
	spec := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
  "root":{"dag":{"tasks":{"task":{"componentRef":{"name":"container"}},"group":{"componentRef":{"name":"group"}}}}},
  "components":{"container":{"executorLabel":"exec"},"group":{"dag":{}}},
  "deploymentSpec":{"executors":{"exec":{"container":{"image":"test-image"}}}}
 }`), spec))
	api := &invocationAPI{run: &gc.Run{RunId: "run-id", RuntimeConfig: &gc.RuntimeConfig{PipelineRoot: "s3://bucket/root"}}, spec: spec}
	manager := &invocationManager{api: api, k8s: fake.NewSimpleClientset()}
	args := driverapi.DriverPluginArgs{
		Type: kind, Namespace: "ns", PipelineName: "pipeline", RunID: "run-id", RunName: "workflow", IterationIndex: "-1", LogLevel: "1",
	}
	if kind != rootDAG {
		args.ParentTaskID = "parent"
		args.TaskName = "task"
	}
	if kind == dagDriver {
		args.TaskName = "group"
	}
	runner := newDriverRunner()
	runner.logDirectory = t.TempDir()
	runner.newClients = func(context.Context, driverapi.DriverPluginArgs, string) (*invocationClients, error) {
		return &invocationClients{manager: manager, pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "agent", UID: "agent-uid"}},
			workflowMetadata: func(context.Context, string, string) (*metav1.ObjectMeta, error) {
				t.Fatal("unnecessary workflow lookup")
				return nil, nil
			},
		}, nil
	}
	return runner, args, api, manager
}

func TestDriveSetupAndExecution(t *testing.T) {
	for _, kind := range []string{rootDAG, dagDriver, containerDriver} {
		for _, executionErr := range []error{nil, assert.AnError} {
			t.Run(kind+"/"+map[bool]string{true: "failure", false: "success"}[executionErr != nil], func(t *testing.T) {
				runner, args, _, manager := invocationFixture(t, kind)
				t.Setenv(caCertPathEnvVar, "/cert/path")
				args.RuntimeConfig = `{"parameterValues":{"input":"value"}}`
				args.DefaultRunAsNonRoot = "false"
				args.DefaultHostUsers = "true"
				args.KubernetesConfig = `{"imagePullSecret":[{"secretName":"pull-secret"}]}`
				execution := &driver.Execution{TaskID: "task-id"}
				var events []string
				runner.execute = func(ctx context.Context, options drivercommon.Options, clients client_manager.ClientManagerInterface) (*driver.Execution, error) {
					events = append(events, "execute")
					assert.Same(t, manager, clients)
					assert.Equal(t, kind, options.DriverType)
					assert.Equal(t, "agent", options.PodName)
					assert.Equal(t, "agent-uid", options.PodUID)
					assert.Equal(t, -1, options.IterationIndex)
					assert.Equal(t, "/cert/path", options.CaCertPath)
					assert.NotNil(t, options.Component)
					if kind == rootDAG {
						require.NotNil(t, options.RuntimeConfig)
						assert.Equal(t, "value", options.RuntimeConfig.ParameterValues["input"].GetStringValue())
					}
					if kind == containerDriver {
						assert.Equal(t, "test-image", options.Container.Image)
						assert.Equal(t, "pull-secret", options.KubernetesExecutorConfig.ImagePullSecret[0].SecretName)
						require.NotNil(t, options.DefaultRunAsNonRoot)
						assert.False(t, *options.DefaultRunAsNonRoot)
						require.NotNil(t, options.DefaultHostUsers)
						assert.True(t, *options.DefaultHostUsers)
						assert.NotEmpty(t, options.OutputPathPrefix)
					}
					util.GetLoggerFrom(ctx).Info("execution log line")
					return execution, executionErr
				}
				runner.uploadLogs = func(ctx context.Context, log *driverLogArtifactContext, client kubernetes.Interface) error {
					events = append(events, "upload")
					assert.Same(t, manager.k8s, client)
					assert.Same(t, manager.api, log.KFPAPI)
					assert.Same(t, execution, log.Execution)
					assert.Equal(t, "s3://bucket/root/pipeline/run-id", log.PipelineRoot)
					deadline, ok := ctx.Deadline()
					require.True(t, ok)
					assert.LessOrEqual(t, time.Until(deadline), driverLogPublicationTimeout)
					contents, err := os.ReadFile(log.LocalPath)
					require.NoError(t, err)
					assert.Contains(t, string(contents), "execution log line")
					if executionErr != nil {
						assert.Contains(t, string(contents), "driver execution failed")
					}
					return assert.AnError // publication must never replace execution outcome
				}
				manager.close = func() error {
					events = append(events, "close clients")
					files, err := os.ReadDir(runner.logDirectory)
					require.NoError(t, err)
					assert.Empty(t, files)
					return assert.AnError
				}
				actual, err := runner.drive(context.Background(), args)
				assert.Same(t, execution, actual)
				if executionErr != nil {
					require.ErrorIs(t, err, executionErr)
					assert.Contains(t, err.Error(), "KFP driver")
				} else {
					require.NoError(t, err)
				}
				assert.Equal(t, []string{"execute", "upload", "close clients"}, events)
			})
		}
	}
}

func TestDriveSetupFailuresCleanUp(t *testing.T) {
	for _, tc := range []struct {
		name, message          string
		mutate                 func(*driverRunner, *driverapi.DriverPluginArgs, *invocationAPI)
		clientsCreated, upload bool
		cause                  error
	}{
		{"namespace", "namespace", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) { args.Namespace = "" }, false, false, nil},
		{"iteration", "iteration index", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) {
			args.IterationIndex = "invalid"
		}, false, false, nil},
		{"client setup", "identity failure", func(r *driverRunner, _ *driverapi.DriverPluginArgs, _ *invocationAPI) {
			r.newClients = func(context.Context, driverapi.DriverPluginArgs, string) (*invocationClients, error) {
				return nil, errors.New("identity failure")
			}
		}, false, false, nil},
		{"runtime config", "runtime config", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) { args.RuntimeConfig = "{" }, true, false, nil},
		{"kubernetes config", "Kubernetes config", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) { args.KubernetesConfig = "{" }, true, false, nil},
		{"get run", "failed to get run", func(_ *driverRunner, _ *driverapi.DriverPluginArgs, api *invocationAPI) {
			api.getRunErr = assert.AnError
		}, true, false, assert.AnError},
		{"get parent task", "failed to get parent task", func(_ *driverRunner, _ *driverapi.DriverPluginArgs, api *invocationAPI) {
			api.getTaskErr = assert.AnError
		}, true, false, assert.AnError},
		{"fetch spec", "failed to build scope path", func(_ *driverRunner, _ *driverapi.DriverPluginArgs, api *invocationAPI) { api.specErr = assert.AnError }, true, false, assert.AnError},
		{"resolve spec", "failed to resolve specs", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) { args.TaskName = "group" }, true, false, nil},
		{"run as non root", "default_run_as_non_root", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) {
			args.DefaultRunAsNonRoot = "invalid"
		}, true, true, nil},
		{"host users", "default_host_users", func(_ *driverRunner, args *driverapi.DriverPluginArgs, _ *invocationAPI) {
			args.DefaultHostUsers = "invalid"
		}, true, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, args, api, manager := invocationFixture(t, containerDriver)
			closed, uploaded := false, false
			manager.close = func() error { closed = true; return nil }
			runner.execute = func(context.Context, drivercommon.Options, client_manager.ClientManagerInterface) (*driver.Execution, error) {
				t.Fatal("must not execute after setup failure")
				return nil, nil
			}
			runner.uploadLogs = func(_ context.Context, log *driverLogArtifactContext, _ kubernetes.Interface) error {
				uploaded = true
				assert.Nil(t, log.Execution)
				return nil
			}
			tc.mutate(runner, &args, api)
			execution, err := runner.drive(context.Background(), args)
			assert.Nil(t, execution)
			require.ErrorContains(t, err, tc.message)
			if tc.cause != nil {
				assert.ErrorIs(t, err, tc.cause)
			}
			if tc.name == "runtime config" {
				assert.NotNil(t, errors.Unwrap(errors.Unwrap(err)), "retain the parser error")
			}
			assert.Equal(t, tc.clientsCreated, closed)
			assert.Equal(t, tc.upload, uploaded)
			files, err := os.ReadDir(runner.logDirectory)
			require.NoError(t, err)
			assert.Empty(t, files)
		})
	}
}

func TestDriveLoggerCreationFailure(t *testing.T) {
	runner := newDriverRunner()
	runner.logDirectory = filepath.Join(t.TempDir(), "missing")
	runner.newClients = func(context.Context, driverapi.DriverPluginArgs, string) (*invocationClients, error) {
		t.Fatal("no client setup without logger")
		return nil, nil
	}
	execution, err := runner.drive(context.Background(), driverapi.DriverPluginArgs{LogLevel: "1"})
	assert.Nil(t, execution)
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, err.Error(), "check directory permissions")
}

func TestDriveLogStorageSetupFailureIsBestEffort(t *testing.T) {
	for _, mode := range []string{"pipeline root", "launcher config", "session"} {
		t.Run(mode, func(t *testing.T) {
			runner, args, api, manager := invocationFixture(t, rootDAG)
			if mode == "pipeline root" {
				api.run.RuntimeConfig = nil
			}
			client := manager.k8s.(*fake.Clientset)
			client.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				if mode == "session" {
					return true, &corev1.ConfigMap{Data: map[string]string{"providers": "[invalid"}}, nil
				}
				return true, nil, assert.AnError
			})
			runner.execute = func(context.Context, drivercommon.Options, client_manager.ClientManagerInterface) (*driver.Execution, error) {
				return &driver.Execution{TaskID: "task"}, nil
			}
			runner.uploadLogs = func(context.Context, *driverLogArtifactContext, kubernetes.Interface) error {
				t.Fatal("storage setup failed")
				return nil
			}
			execution, err := runner.drive(context.Background(), args)
			require.NoError(t, err)
			assert.Equal(t, "task", execution.TaskID)
		})
	}
}

type closeFunc func() error

func (f closeFunc) Close() error { return f() }

func TestPublishAndCloseOrderingAndFailures(t *testing.T) {
	for _, cleanupFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup success", true: "cleanup failure"}[cleanupFails], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "log")
			file, err := os.Create(path)
			require.NoError(t, err)
			var events []string
			manager := &invocationManager{k8s: fake.NewSimpleClientset()}
			invocation := &driverInvocation{
				clients:    &invocationClients{manager: manager},
				logContext: driverLogArtifactContext{PipelineRoot: "s3://bucket", LocalPath: path},
				logCloser: closeFunc(func() error {
					events = append(events, "close log")
					require.NoError(t, file.Close())
					return assert.AnError
				}),
			}
			var publicationContext context.Context
			manager.close = func() error {
				events = append(events, "close clients")
				assert.ErrorIs(t, publicationContext.Err(), context.Canceled)
				_, err := os.Stat(path)
				if cleanupFails {
					require.NoError(t, err)
				} else {
					assert.ErrorIs(t, err, os.ErrNotExist)
				}
				return assert.AnError
			}
			invocation.publishAndClose(context.Background(), func(ctx context.Context, log *driverLogArtifactContext, _ kubernetes.Interface) error {
				publicationContext = ctx
				events = append(events, "upload")
				_, err := file.WriteString("too late")
				assert.ErrorIs(t, err, os.ErrClosed)
				require.FileExists(t, path)
				if cleanupFails {
					require.NoError(t, os.Remove(path))
					require.NoError(t, os.Mkdir(path, 0700))
					require.NoError(t, os.WriteFile(filepath.Join(path, "child"), nil, 0600))
				}
				return assert.AnError
			})
			assert.Equal(t, []string{"close log", "upload", "close clients"}, events)
		})
	}
}

func TestPluginDispatcherWithoutRuntimeArgsReturnsNonNil(t *testing.T) {
	dispatcher, err := plugins.GetPluginDispatcherWithRuntimeArgs(nil)
	require.NoError(t, err)
	require.NotNil(t, dispatcher)
	// With no plugins enabled in unit tests, this should be a usable no-op dispatcher.
	_, err = dispatcher.OnTaskStart(context.Background(), &plugins.TaskInfo{Name: "unit-test"})
	assert.NoError(t, err)
}
