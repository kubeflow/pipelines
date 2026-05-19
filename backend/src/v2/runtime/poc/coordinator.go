package poc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	apitemplate "github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	backendcompiler "github.com/kubeflow/pipelines/backend/src/v2/compiler"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"gocloud.dev/blob"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

type Coordinator struct {
	runStore              storage.RunStoreInterface
	taskStore             storage.TaskStoreInterface
	artifactStore         storage.ArtifactStoreInterface
	artifactTaskStore     storage.ArtifactTaskStoreInterface
	kubernetesCoreClient  apiserverclient.KubernetesCoreInterface
	defaultWorkspaceSpec  *corev1.PersistentVolumeClaimSpec
	mlPipelineTLSEnabled  bool
	time                  util.TimeInterface
	executors             map[string]WorkloadExecutorPlugin
	workspaceProvisioners map[string]WorkspaceProvisionerPlugin
	queue                 chan *queuedRun
	activeRunMu           sync.Mutex
	activeRuns            map[string]*activeRunExecution
}

type queuedRun struct {
	run             *model.Run
	job             *pipelinespec.PipelineJob
	spec            *pipelinespec.PipelineSpec
	kubernetesSpec  *pipelinespec.SinglePlatformSpec
	runtimeManifest *RuntimeManifest
	workspace       *WorkspaceHandle
}

const runtimeArtifactCustomPathMetadataKey = "_kfp_custom_path"

var sqliteLockRetryDelay = 200 * time.Millisecond

const sqliteLockRetryAttempts = 10

type activeRunExecution struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewCoordinator(
	runStore storage.RunStoreInterface,
	taskStore storage.TaskStoreInterface,
	artifactStore storage.ArtifactStoreInterface,
	artifactTaskStore storage.ArtifactTaskStoreInterface,
	time util.TimeInterface,
	kubernetesCoreClient apiserverclient.KubernetesCoreInterface,
	defaultWorkspaceSpec *corev1.PersistentVolumeClaimSpec,
	mlPipelineTLSEnabled bool,
) *Coordinator {
	return &Coordinator{
		runStore:             runStore,
		taskStore:            taskStore,
		artifactStore:        artifactStore,
		artifactTaskStore:    artifactTaskStore,
		kubernetesCoreClient: kubernetesCoreClient,
		defaultWorkspaceSpec: defaultWorkspaceSpec,
		mlPipelineTLSEnabled: mlPipelineTLSEnabled,
		time:                 time,
		executors: map[string]WorkloadExecutorPlugin{
			ExecutorDocker:     dockerExecutor{},
			ExecutorKubernetes: kubernetesExecutor{coreClient: kubernetesCoreClient},
		},
		workspaceProvisioners: map[string]WorkspaceProvisionerPlugin{
			ExecutorDocker:     hostPathWorkspaceProvisioner{},
			ExecutorKubernetes: pvcWorkspaceProvisioner{coreClient: kubernetesCoreClient, defaultWorkspaceSpec: defaultWorkspaceSpec},
		},
		queue:      make(chan *queuedRun, 32),
		activeRuns: map[string]*activeRunExecution{},
	}
}

func (c *Coordinator) Start() {
	go c.runLoop(context.Background())
}

func (c *Coordinator) RegisterExecutor(plugin WorkloadExecutorPlugin) {
	if plugin == nil {
		return
	}
	c.executors[plugin.Type()] = plugin
}

func (c *Coordinator) RegisterWorkspaceProvisioner(plugin WorkspaceProvisionerPlugin) {
	if plugin == nil {
		return
	}
	c.workspaceProvisioners[plugin.Type()] = plugin
}

func (c *Coordinator) SubmitRun(
	ctx context.Context,
	run *model.Run,
	job *pipelinespec.PipelineJob,
	kubernetesSpec *pipelinespec.SinglePlatformSpec,
) error {
	if run == nil {
		return fmt.Errorf("run is required")
	}
	if job == nil {
		return fmt.Errorf("pipeline job is required")
	}

	spec, err := backendcompiler.GetPipelineSpec(job)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline spec: %w", err)
	}
	executorType := RequestedExecutor()
	executor, ok := c.executors[executorType]
	if !ok {
		return fmt.Errorf("unsupported runtime executor %q", executorType)
	}
	workspaceProvisioner, ok := c.workspaceProvisioners[executorType]
	if !ok {
		return fmt.Errorf("no workspace provisioner registered for executor %q", executorType)
	}

	workspace, err := workspaceProvisioner.Provision(ctx, run, job)
	if err != nil {
		return fmt.Errorf("failed to provision runtime workspace: %w", err)
	}
	cleanupWorkspace := true
	defer func() {
		if cleanupWorkspace {
			_ = workspaceProvisioner.Cleanup(ctx, run, workspace)
		}
	}()
	executorInfo, err := executor.PrepareRun(ctx, run, job, workspace)
	if err != nil {
		return fmt.Errorf("failed to prepare runtime executor: %w", err)
	}

	manifest := NewRuntimeManifest(executorType)
	manifest.Workspace = workspace
	manifest.ExecutorInfo = executorInfo

	rootTask, err := c.createTask(run, nil, "ROOT", run.DisplayName, apiv2beta1.PipelineTaskDetail_ROOT, "root", nil)
	if err != nil {
		return fmt.Errorf("failed to create root task: %w", err)
	}
	manifest.Tasks = append(manifest.Tasks, RuntimeTask{
		TaskID:      rootTask.UUID,
		Name:        rootTask.Name,
		DisplayName: rootTask.DisplayName,
		Type:        apiv2beta1.PipelineTaskDetail_TaskType(rootTask.Type).String(),
		ScopePath:   rootTask.ScopePath,
	})

	if err := c.materializeComponent(run, spec, spec.GetRoot(), "root", rootTask.UUID, manifest); err != nil {
		return err
	}

	manifestJSON, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal runtime manifest: %w", err)
	}
	run.PipelineRuntimeManifest = model.LargeText(manifestJSON)
	if err := c.updateRun(run); err != nil {
		return fmt.Errorf("failed to persist runtime manifest: %w", err)
	}
	if err := c.enqueueRun(ctx, &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		kubernetesSpec:  kubernetesSpec,
		runtimeManifest: manifest,
		workspace:       workspace,
	}); err != nil {
		return err
	}
	cleanupWorkspace = false
	return nil
}

func (c *Coordinator) enqueueRun(ctx context.Context, item *queuedRun) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.queue <- item:
		return nil
	}
}

func (c *Coordinator) materializeComponent(
	run *model.Run,
	spec *pipelinespec.PipelineSpec,
	component *pipelinespec.ComponentSpec,
	parentScope string,
	parentTaskID string,
	manifest *RuntimeManifest,
) error {
	if component == nil {
		return fmt.Errorf("component is nil for scope %q", parentScope)
	}
	dag := component.GetDag()
	if dag == nil {
		return fmt.Errorf("component at scope %q is not a DAG; only DAG-backed orchestration is supported in the coordinator proof of concept", parentScope)
	}

	taskNames := make([]string, 0, len(dag.GetTasks()))
	for taskName := range dag.GetTasks() {
		taskNames = append(taskNames, taskName)
	}
	sort.Strings(taskNames)

	for _, taskName := range taskNames {
		taskSpec := dag.GetTasks()[taskName]
		if err := validateSupportedTask(taskSpec); err != nil {
			return fmt.Errorf("task %q is not supported by the coordinator proof of concept: %w", taskName, err)
		}
		scopePath := util.StringPathToDotNotation([]string{parentScope, taskName})
		refName := taskSpec.GetComponentRef().GetName()
		childComponent := spec.GetComponents()[refName]
		if childComponent == nil {
			return fmt.Errorf("component ref %q not found for task %q", refName, taskName)
		}
		importerSpec, err := importerSpecForComponent(spec, childComponent)
		if err != nil {
			return err
		}

		taskType := apiv2beta1.PipelineTaskDetail_RUNTIME
		if childComponent.GetDag() != nil {
			taskType = apiv2beta1.PipelineTaskDetail_DAG
		} else if importerSpec != nil {
			taskType = apiv2beta1.PipelineTaskDetail_IMPORTER
		}
		if taskSpec.GetTriggerPolicy().GetCondition() != "" {
			taskType = apiv2beta1.PipelineTaskDetail_CONDITION_BRANCH
		}
		typeAttrs, err := taskTypeAttributes(taskSpec, importerSpec)
		if err != nil {
			return err
		}
		createdTask, err := c.createTask(
			run,
			util.StringPointer(parentTaskID),
			taskName,
			taskDisplayName(taskSpec),
			taskType,
			scopePath,
			typeAttrs,
		)
		if err != nil {
			return fmt.Errorf("failed to create task %q: %w", taskName, err)
		}

		manifest.Tasks = append(manifest.Tasks, RuntimeTask{
			TaskID:       createdTask.UUID,
			ParentTaskID: parentTaskID,
			Name:         createdTask.Name,
			DisplayName:  createdTask.DisplayName,
			Type:         apiv2beta1.PipelineTaskDetail_TaskType(createdTask.Type).String(),
			ScopePath:    createdTask.ScopePath,
			DependsOn:    append([]string(nil), taskSpec.GetDependentTasks()...),
		})

		if childComponent.GetDag() != nil {
			if err := c.materializeComponent(run, spec, childComponent, scopePath, createdTask.UUID, manifest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Coordinator) createTask(
	run *model.Run,
	parentTaskID *string,
	name string,
	displayName string,
	taskType apiv2beta1.PipelineTaskDetail_TaskType,
	scopePath string,
	typeAttrs model.JSONData,
) (*model.Task, error) {
	task := &model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: parentTaskID,
		Name:           name,
		DisplayName:    displayName,
		Type:           model.TaskType(taskType),
		State:          model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
		ScopePath:      scopePath,
		TypeAttrs:      typeAttrs,
		CreatedAtInSec: c.time.Now().Unix(),
	}
	return c.taskStore.CreateTask(task)
}

func (c *Coordinator) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.queue:
			if item == nil {
				continue
			}
			go func(item *queuedRun) {
				if err := c.executeRun(ctx, item); err != nil {
					glog.Errorf("Coordinator run %s failed: %v", item.run.UUID, err)
				}
			}(item)
		}
	}
}

func (c *Coordinator) executeRun(ctx context.Context, item *queuedRun) error {
	run := item.run
	manifest := item.runtimeManifest
	if run == nil || manifest == nil {
		return fmt.Errorf("queued run is incomplete")
	}
	cleanupWorkspace := false
	defer func() {
		if !cleanupWorkspace {
			return
		}
		if err := c.cleanupWorkspaceForRun(context.Background(), run, manifest); err != nil {
			glog.Warningf("Failed to cleanup managed workspace for run %s: %v", run.UUID, err)
		}
	}()
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	c.setActiveRunExecution(run.UUID, cancel, done)
	defer func() {
		c.clearActiveRunExecution(run.UUID)
		close(done)
	}()
	executor, ok := c.executors[manifest.Executor]
	if !ok {
		return fmt.Errorf("no executor registered for %q", manifest.Executor)
	}
	rootTask, ok := manifest.taskByScope("root")
	if !ok {
		return fmt.Errorf("root task is missing from runtime manifest")
	}
	if cancellationRequested, err := c.runCancellationRequested(run.UUID); err != nil {
		return err
	} else if cancellationRequested {
		err := c.finalizeCanceledRun(run.UUID, "run cancellation was requested before execution started")
		if err == nil {
			cleanupWorkspace = true
		}
		return err
	}

	if err := c.updateRunState(run.UUID, model.RuntimeStateRunning, 0); err != nil {
		return err
	}
	if _, err := c.updateTaskState(rootTask.TaskID, apiv2beta1.PipelineTaskDetail_RUNNING, nil, nil); err != nil {
		return err
	}
	rootInputs, err := runtimeParameters(item.run, item.spec.GetRoot())
	if err != nil {
		return err
	}

	if err := c.executeDagComponent(runCtx, item, executor, item.spec.GetRoot(), "root", rootInputs, false); err != nil {
		if c.isCancellationError(run.UUID, err) {
			cancelErr := c.finalizeCanceledRun(run.UUID, "run execution was cancelled")
			if cancelErr == nil {
				cleanupWorkspace = true
			}
			return cancelErr
		}
		_, _ = c.updateTaskState(rootTask.TaskID, apiv2beta1.PipelineTaskDetail_FAILED, model.JSONData{
			"message": err.Error(),
		}, nil)
		cleanupWorkspace = true
		_ = c.updateRunState(run.UUID, model.RuntimeStateFailed, c.time.Now().Unix())
		return err
	}
	if cancellationRequested, err := c.runCancellationRequested(run.UUID); err != nil {
		return err
	} else if cancellationRequested {
		err := c.finalizeCanceledRun(run.UUID, "run execution was cancelled")
		if err == nil {
			cleanupWorkspace = true
		}
		return err
	}

	if _, err := c.updateTaskState(rootTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, nil, nil); err != nil {
		return err
	}
	cleanupWorkspace = true
	return c.updateRunState(run.UUID, model.RuntimeStateSucceeded, c.time.Now().Unix())
}

func (c *Coordinator) executeDagComponent(
	ctx context.Context,
	item *queuedRun,
	executor WorkloadExecutorPlugin,
	component *pipelinespec.ComponentSpec,
	parentScope string,
	scopeInputs map[string]*structpb.Value,
	forceExecute bool,
) error {
	if component == nil || component.GetDag() == nil {
		return fmt.Errorf("component at scope %q is not a DAG", parentScope)
	}

	tasks := component.GetDag().GetTasks()
	names := make([]string, 0, len(tasks))
	for name := range tasks {
		names = append(names, name)
	}
	sort.Strings(names)

	completed := map[string]struct{}{}
	for len(completed) < len(names) {
		if cancellationRequested, err := c.runCancellationRequested(item.run.UUID); err != nil {
			return err
		} else if cancellationRequested {
			return context.Canceled
		}
		progressed := false
		for _, name := range names {
			if _, done := completed[name]; done {
				continue
			}
			taskSpec := tasks[name]
			if !dependenciesSatisfied(taskSpec, completed) {
				continue
			}
			scopePath := util.StringPathToDotNotation([]string{parentScope, name})
			runtimeTask, ok := item.runtimeManifest.taskByScope(scopePath)
			if !ok {
				return fmt.Errorf("runtime task %q not found in manifest", scopePath)
			}
			existingTask, err := c.taskStore.GetTask(runtimeTask.TaskID)
			if err != nil {
				return err
			}
			if !forceExecute && isTerminalTaskState(existingTask.State) {
				completed[name] = struct{}{}
				progressed = true
				continue
			}
			childComponent := item.spec.GetComponents()[taskSpec.GetComponentRef().GetName()]
			if childComponent == nil {
				return fmt.Errorf("component ref %q not found for task %q", taskSpec.GetComponentRef().GetName(), name)
			}
			taskInputs, err := c.resolveInputParameters(item, runtimeTask, scopePath, taskSpec, childComponent, scopeInputs)
			if err != nil {
				return err
			}
			if condition := taskSpec.GetTriggerPolicy().GetCondition(); condition != "" {
				willTrigger, err := evaluateTaskCondition(taskInputs, condition)
				if err != nil {
					c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
					return err
				}
				if !willTrigger {
					if err := c.skipTaskTree(item, runtimeTask.ScopePath); err != nil {
						return err
					}
					completed[name] = struct{}{}
					progressed = true
					continue
				}
			}
			if taskSpec.GetParameterIterator() != nil {
				if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_RUNNING, nil, nil); err != nil {
					return err
				}
				if err := c.executeParameterIteratorTask(ctx, item, executor, runtimeTask, taskSpec, childComponent, scopePath, taskInputs); err != nil {
					c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
					return err
				}
				if childComponent.GetDag() != nil {
					if err := c.propagateDagOutputs(item, runtimeTask, childComponent, scopePath); err != nil {
						c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
						return err
					}
				}
				if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, nil, nil); err != nil {
					return err
				}
			} else if childComponent.GetDag() != nil {
				if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_RUNNING, nil, nil); err != nil {
					return err
				}
				if err := c.executeDagComponent(ctx, item, executor, childComponent, scopePath, taskInputs, forceExecute); err != nil {
					c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
					return err
				}
				if err := c.propagateDagOutputs(item, runtimeTask, childComponent, scopePath); err != nil {
					c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
					return err
				}
				if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, nil, nil); err != nil {
					return err
				}
			} else {
				if err := c.executeRuntimeTask(ctx, item, executor, runtimeTask, taskSpec, childComponent, taskInputs); err != nil {
					return err
				}
			}
			completed[name] = struct{}{}
			progressed = true
		}
		if !progressed {
			return fmt.Errorf("unable to make progress executing DAG at scope %q", parentScope)
		}
	}
	return nil
}

func (c *Coordinator) executeRuntimeTask(
	ctx context.Context,
	item *queuedRun,
	executor WorkloadExecutorPlugin,
	runtimeTask RuntimeTask,
	taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	parameterValues map[string]*structpb.Value,
) error {
	taskModel, err := c.taskStore.GetTask(runtimeTask.TaskID)
	if err != nil {
		return err
	}
	if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_RUNNING, nil, nil); err != nil {
		return err
	}
	importerSpec, err := importerSpecForComponent(item.spec, componentSpec)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	if importerSpec != nil {
		result, err := c.executeImporterTask(ctx, item, runtimeTask, taskSpec, componentSpec, importerSpec, parameterValues)
		if err != nil {
			c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
			return err
		}
		_, err = c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, result.StatusMetadata, nil)
		return err
	}

	executorInput, outputTypes, err := c.buildExecutorInput(item, runtimeTask.ScopePath, taskSpec, componentSpec, parameterValues)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	container, err := containerForComponent(item.spec, componentSpec)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	kubernetesExecutorConfig, err := loadKubernetesExecutorConfig(componentSpec, item.kubernetesSpec)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	pvcNames, err := resolvePVCNames(kubernetesExecutorConfig, executorInput)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	fingerprint, cachedTask, err := c.lookupCachedTask(componentSpec, taskSpec, container, executorInput, item.run.Namespace, pvcNames)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	if fingerprint != "" {
		_, _ = c.updateTask(&model.Task{UUID: runtimeTask.TaskID, Fingerprint: fingerprint})
		taskModel.Fingerprint = fingerprint
	}
	if cachedTask != nil {
		return c.applyCachedTask(item.run, runtimeTask.TaskID, taskSpec, cachedTask)
	}
	if platformResult, handled, err := c.executePlatformTask(ctx, item.run, taskModel, container, executorInput, taskSpec); handled {
		if err != nil {
			c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
			return err
		}
		_, err = c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, platformResult.StatusMetadata, outputParametersToJSON(platformResult.OutputParameters, taskSpec))
		return err
	}
	resolverRunTasks, resolverParentTask, err := c.buildResolverTaskDetails(item.runtimeManifest, runtimeTask.ParentTaskID)
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}

	result, err := executor.ExecuteTask(ctx, &TaskExecutionRequest{
		Run:                      item.run,
		Task:                     taskModel,
		Workspace:                item.workspace,
		Container:                container,
		PipelineSpec:             item.job.GetPipelineSpec(),
		ExecutorInput:            executorInput,
		OutputParameterTypes:     outputTypes,
		ComponentSpec:            componentSpec,
		TaskSpec:                 taskSpec,
		KubernetesExecutorConfig: kubernetesExecutorConfig,
		ResolverRunTasks:         resolverRunTasks,
		ResolverParentTask:       resolverParentTask,
		ParentTaskID:             runtimeTask.ParentTaskID,
		PipelineName:             item.spec.GetPipelineInfo().GetName(),
		TaskDisplayName:          taskDisplayName(taskSpec),
		ArtifactLocalRoot:        c.artifactLocalRoot(item.run, RequestedExecutor()),
		MLPipelineTLSEnabled:     c.mlPipelineTLSEnabled,
		ObjectStore:              c.newObjectStoreClient(item.run.Namespace),
		HandleRecorder: func(pods model.JSONSlice, metadata model.JSONData) error {
			update := &model.Task{UUID: runtimeTask.TaskID}
			if pods != nil {
				update.Pods = pods
			}
			if metadata != nil {
				update.StatusMetadata = mergeJSONData(taskModel.StatusMetadata, metadata)
			}
			_, recorderErr := c.updateTask(update)
			return recorderErr
		},
	})
	if err != nil {
		c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
		return err
	}
	if RequestedExecutor() == ExecutorDocker {
		if err := c.publishOutputArtifacts(ctx, item.run, runtimeTask.TaskID, taskDisplayName(taskSpec), executorInput); err != nil {
			c.recordTaskExecutionError(item.run.UUID, runtimeTask.TaskID, err)
			return err
		}
	}
	_, err = c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SUCCEEDED, result.StatusMetadata, outputParametersToJSON(result.OutputParameters, taskSpec))
	return err
}

func (c *Coordinator) executeImporterTask(
	ctx context.Context,
	item *queuedRun,
	runtimeTask RuntimeTask,
	taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	importerSpec *pipelinespec.PipelineDeploymentConfig_ImporterSpec,
	parameterValues map[string]*structpb.Value,
) (*TaskExecutionResult, error) {
	if item == nil || item.run == nil || importerSpec == nil || componentSpec == nil {
		return nil, fmt.Errorf("importer task execution request is incomplete")
	}
	if c.artifactStore == nil || c.artifactTaskStore == nil {
		return nil, fmt.Errorf("importer task execution requires artifact stores")
	}
	artifactURI, err := resolveImporterArtifactURI(importerSpec, parameterValues)
	if err != nil {
		return nil, err
	}
	artifactKey, err := importerArtifactOutputKey(componentSpec)
	if err != nil {
		return nil, err
	}
	artifactType, err := importerArtifactType(importerSpec.GetTypeSchema())
	if err != nil {
		return nil, err
	}
	artifactMetadata := importerArtifactMetadata(importerSpec)
	if importerSpec.GetDownloadToWorkspace() {
		if shouldDownloadImporterArtifactToWorkspace(item.workspace) {
			objectStoreClient := c.newObjectStoreClient(item.run.Namespace)
			if objectStoreClient == nil {
				return nil, fmt.Errorf("importer download_to_workspace requires an object store client")
			}
			localPath, err := importerWorkspaceArtifactLocalPath(item.workspace, artifactURI)
			if err != nil {
				return nil, err
			}
			if err := ensureContainerWritableDir(filepath.Dir(localPath)); err != nil {
				return nil, err
			}
			if err := objectStoreClient.DownloadArtifact(ctx, artifactURI, localPath, artifactKey); err != nil {
				return nil, fmt.Errorf("failed to download artifact to workspace: %w", err)
			}
			artifactMetadata[drivercommon.WorkspaceMetadataField] = true
		}
	}
	artifactModel := &model.Artifact{
		Namespace: item.run.Namespace,
		Type:      model.ArtifactType(artifactType),
		URI:       util.StringPointer(artifactURI),
		Name:      importerArtifactName(artifactURI),
		Metadata:  artifactMetadata,
	}
	createdArtifact, err := c.artifactStore.CreateArtifact(artifactModel)
	if err != nil {
		return nil, err
	}
	_, err = c.artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  createdArtifact.UUID,
		TaskID:      runtimeTask.TaskID,
		RunUUID:     item.run.UUID,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: artifactKey,
		Producer: model.JSONData{
			"task_name": runtimeTask.Name,
		},
	})
	if err != nil {
		return nil, err
	}
	return &TaskExecutionResult{
		StatusMetadata: customStatusMetadata("", map[string]any{
			"executor": "importer",
			"uri":      artifactURI,
		}),
	}, nil
}

func (c *Coordinator) executePlatformTask(
	ctx context.Context,
	run *model.Run,
	task *model.Task,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	executorInput *pipelinespec.ExecutorInput,
	taskSpec *pipelinespec.PipelineTaskSpec,
) (*TaskExecutionResult, bool, error) {
	if container == nil {
		return nil, false, nil
	}
	switch container.GetImage() {
	case "argostub/createpvc":
		result, err := c.createPVCPlatformTask(ctx, run, task, executorInput, taskSpec)
		return result, true, err
	default:
		return nil, false, nil
	}
}

func (c *Coordinator) createPVCPlatformTask(
	ctx context.Context,
	run *model.Run,
	task *model.Task,
	executorInput *pipelinespec.ExecutorInput,
	taskSpec *pipelinespec.PipelineTaskSpec,
) (*TaskExecutionResult, error) {
	if c.kubernetesCoreClient == nil || c.kubernetesCoreClient.GetClientSet() == nil {
		return nil, fmt.Errorf("createpvc task requires a kubernetes client")
	}
	if run == nil {
		return nil, fmt.Errorf("createpvc task requires a run")
	}
	namespace := run.Namespace
	if namespace == "" {
		namespace = common.GetPodNamespace()
	}
	if namespace == "" {
		return nil, fmt.Errorf("createpvc task requires a run namespace")
	}
	if executorInput == nil || executorInput.GetInputs() == nil {
		return nil, fmt.Errorf("createpvc task requires executor input parameters")
	}
	inputs := executorInput.GetInputs().GetParameterValues()

	accessModeInput, ok := inputs["access_modes"]
	if !ok || accessModeInput == nil {
		return nil, fmt.Errorf("failed to create pvc: parameter access_modes not provided")
	}
	var accessModes []corev1.PersistentVolumeAccessMode
	if accessModeInput.GetListValue() == nil || len(accessModeInput.GetListValue().GetValues()) == 0 {
		return nil, fmt.Errorf("failed to create pvc: parameter access_modes must be a non-empty list")
	}
	for _, value := range accessModeInput.GetListValue().GetValues() {
		accessMode, err := pvcAccessMode(value.GetStringValue())
		if err != nil {
			return nil, fmt.Errorf("failed to create pvc: %w", err)
		}
		accessModes = append(accessModes, accessMode)
	}

	pvcNameSuffixInput := inputs["pvc_name_suffix"]
	pvcNameInput := inputs["pvc_name"]
	var pvcName string
	switch {
	case pvcNameInput != nil && pvcNameInput.GetStringValue() != "" &&
		pvcNameSuffixInput != nil && pvcNameSuffixInput.GetStringValue() != "":
		return nil, fmt.Errorf("failed to create pvc: at most one of pvc_name and pvc_name_suffix can be non-empty")
	case pvcNameSuffixInput != nil && pvcNameSuffixInput.GetStringValue() != "":
		pvcName = uuid.NewString() + pvcNameSuffixInput.GetStringValue()
	case pvcNameInput != nil && pvcNameInput.GetStringValue() != "":
		pvcName = pvcNameInput.GetStringValue()
	default:
		pvcName = uuid.NewString()
	}

	volumeSizeInput, ok := inputs["size"]
	if !ok || volumeSizeInput == nil {
		return nil, fmt.Errorf("failed to create pvc: parameter size not provided")
	}
	volumeSize := strings.TrimSpace(volumeSizeInput.GetStringValue())
	if volumeSize == "" {
		return nil, fmt.Errorf("failed to create pvc: parameter size must be a non-empty string")
	}
	volumeQuantity, err := resource.ParseQuantity(volumeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create pvc: invalid size %q: %w", volumeSize, err)
	}

	var storageClassName *string
	if storageClassNameInput, ok := inputs["storage_class_name"]; ok && storageClassNameInput != nil {
		if requestedStorageClass := strings.TrimSpace(storageClassNameInput.GetStringValue()); requestedStorageClass != "" {
			storageClassName = util.StringPointer(requestedStorageClass)
		}
	}

	pvcAnnotations := map[string]string{}
	if pvcAnnotationsInput, ok := inputs["annotations"]; ok && pvcAnnotationsInput != nil && pvcAnnotationsInput.GetStructValue() != nil {
		for key, value := range pvcAnnotationsInput.GetStructValue().GetFields() {
			pvcAnnotations[key] = value.GetStringValue()
		}
	}

	var volumeName string
	if volumeNameInput, ok := inputs["volume_name"]; ok && volumeNameInput != nil {
		volumeName = volumeNameInput.GetStringValue()
	}

	var dataSource *corev1.TypedLocalObjectReference
	if dataSourceInput, ok := inputs["data_source"]; ok && dataSourceInput != nil {
		dataSource = &corev1.TypedLocalObjectReference{}
		dataSourceBytes, err := dataSourceInput.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data_source: %w", err)
		}
		if err := json.Unmarshal(dataSourceBytes, dataSource); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data_source: %w", err)
		}
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Namespace:   namespace,
			Annotations: pvcAnnotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: volumeQuantity,
				},
			},
			StorageClassName: storageClassName,
			VolumeName:       volumeName,
			DataSource:       dataSource,
		},
	}
	if _, err := c.kubernetesCoreClient.GetClientSet().CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("failed to create pvc: %w", err)
	}

	taskName := taskDisplayName(taskSpec)
	return &TaskExecutionResult{
		OutputParameters: map[string]*structpb.Value{
			"name": structpb.NewStringValue(pvcName),
		},
		StatusMetadata: customStatusMetadata("", map[string]any{
			"executor": "kubernetes-platform-op",
			"taskName": taskName,
			"pvcName":  pvcName,
		}),
	}, nil
}

func pvcAccessMode(value string) (corev1.PersistentVolumeAccessMode, error) {
	switch value {
	case "ReadOnlyMany":
		return corev1.ReadOnlyMany, nil
	case "ReadWriteMany":
		return corev1.ReadWriteMany, nil
	case "ReadWriteOncePod":
		return corev1.ReadWriteOncePod, nil
	case "ReadWriteOnce":
		return corev1.ReadWriteOnce, nil
	default:
		return "", fmt.Errorf("invalid access mode %q", value)
	}
}

func (c *Coordinator) updateRunState(runID string, state model.RuntimeState, finishedAt int64) error {
	run := &model.Run{
		UUID: runID,
		RunDetails: model.RunDetails{
			State:           state,
			Conditions:      string(state.ToV1()),
			FinishedAtInSec: finishedAt,
		},
	}
	run.State = state
	run.Conditions = string(state.ToV1())
	run.FinishedAtInSec = finishedAt
	return c.updateRun(run)
}

func (c *Coordinator) setActiveRunExecution(runID string, cancel context.CancelFunc, done chan struct{}) {
	c.activeRunMu.Lock()
	defer c.activeRunMu.Unlock()
	c.activeRuns[runID] = &activeRunExecution{
		cancel: cancel,
		done:   done,
	}
}

func (c *Coordinator) clearActiveRunExecution(runID string) {
	c.activeRunMu.Lock()
	defer c.activeRunMu.Unlock()
	delete(c.activeRuns, runID)
}

func (c *Coordinator) cancelActiveRun(runID string) <-chan struct{} {
	c.activeRunMu.Lock()
	defer c.activeRunMu.Unlock()
	activeRun := c.activeRuns[runID]
	if activeRun == nil {
		return nil
	}
	if activeRun.cancel != nil {
		activeRun.cancel()
	}
	return activeRun.done
}

func (c *Coordinator) runCancellationRequested(runID string) (bool, error) {
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return true, nil
		}
		return false, err
	}
	return run.State == model.RuntimeStateCancelling || run.State == model.RuntimeStateCanceled, nil
}

func (c *Coordinator) isCancellationError(runID string, err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	cancelRequested, cancelErr := c.runCancellationRequested(runID)
	return cancelErr == nil && cancelRequested
}

func (c *Coordinator) recordTaskExecutionError(runID string, taskID string, err error) {
	if taskID == "" || err == nil {
		return
	}
	state := apiv2beta1.PipelineTaskDetail_FAILED
	message := err.Error()
	if c.isCancellationError(runID, err) {
		state = apiv2beta1.PipelineTaskDetail_SKIPPED
		message = "run execution was cancelled"
	}
	_, _ = c.updateTaskState(taskID, state, model.JSONData{"message": message}, nil)
}

func (c *Coordinator) markRunTasksForState(runID string, state model.RuntimeState, message string) error {
	tasks, _, _, err := c.taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID},
	}, list.EmptyOptions())
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if isTerminalTaskState(task.State) {
			continue
		}
		if _, err := c.updateTaskState(task.UUID, taskStateForRunState(state), model.JSONData{
			"message": message,
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) finalizeCanceledRun(runID string, message string) error {
	if err := c.markRunTasksForState(runID, model.RuntimeStateCanceled, message); err != nil {
		return err
	}
	return c.updateRunState(runID, model.RuntimeStateCanceled, c.time.Now().Unix())
}

func (c *Coordinator) updateTaskState(
	taskID string,
	state apiv2beta1.PipelineTaskDetail_TaskState,
	statusMetadata model.JSONData,
	outputs model.JSONSlice,
) (*model.Task, error) {
	update := &model.Task{
		UUID:             taskID,
		State:            model.TaskStatus(state),
		StatusMetadata:   statusMetadata,
		OutputParameters: outputs,
	}
	if state == apiv2beta1.PipelineTaskDetail_SUCCEEDED || state == apiv2beta1.PipelineTaskDetail_FAILED || state == apiv2beta1.PipelineTaskDetail_SKIPPED {
		update.FinishedInSec = c.time.Now().Unix()
	}
	return c.updateTask(update)
}

func (c *Coordinator) updateRun(run *model.Run) error {
	return retrySQLiteLockedError(func() error {
		return c.runStore.UpdateRun(run)
	})
}

func (c *Coordinator) updateTask(task *model.Task) (*model.Task, error) {
	var updatedTask *model.Task
	err := retrySQLiteLockedError(func() error {
		var updateErr error
		updatedTask, updateErr = c.taskStore.UpdateTask(task)
		return updateErr
	})
	return updatedTask, err
}

func retrySQLiteLockedError(operation func() error) error {
	var lastErr error
	for attempt := 0; attempt < sqliteLockRetryAttempts; attempt++ {
		lastErr = operation()
		if lastErr == nil || !isSQLiteLockedError(lastErr) {
			return lastErr
		}
		time.Sleep(sqliteLockRetryDelay)
	}
	return lastErr
}

func isSQLiteLockedError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked")
}

func (c *Coordinator) ReconcileManagedRuns(ctx context.Context) error {
	runs, _, _, err := c.runStore.ListRuns(&model.FilterContext{}, list.EmptyOptions(), false)
	if err != nil {
		return err
	}
	for _, run := range runs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !IsManagedRun(run) {
			continue
		}
		switch run.State {
		case model.RuntimeStatePending, model.RuntimeStateRunning, model.RuntimeStateUnspecified, model.RuntimeStateCancelling:
		default:
			continue
		}
		if run.State == model.RuntimeStateCancelling {
			if err := c.deleteManagedRunPods(ctx, run.UUID, run.Namespace); err != nil {
				return err
			}
			if err := c.finalizeCanceledRun(run.UUID, "coordinator run was cancelling during API server restart"); err != nil {
				return err
			}
			if cleanupErr := c.CleanupManagedRun(ctx, run); cleanupErr != nil {
				glog.Warningf("Failed to cleanup managed run %s after restart cancellation: %v", run.UUID, cleanupErr)
			}
			continue
		}
		queued, queueErr := c.rebuildQueuedRun(run)
		if queueErr != nil {
			return queueErr
		}
		if run.State == model.RuntimeStateRunning && queued.runtimeManifest != nil && queued.runtimeManifest.Executor == ExecutorDocker {
			if err := c.markRunTasksForState(run.UUID, model.RuntimeStateFailed, "local-docker managed runs cannot resume after API server restart"); err != nil {
				return err
			}
			if cleanupErr := c.cleanupWorkspaceForRun(ctx, run, queued.runtimeManifest); cleanupErr != nil {
				glog.Warningf("Failed to cleanup managed workspace for run %s after API server restart: %v", run.UUID, cleanupErr)
			}
			if err := c.updateRunState(run.UUID, model.RuntimeStateFailed, c.time.Now().Unix()); err != nil {
				return err
			}
			continue
		}
		if err := c.enqueueRun(ctx, queued); err != nil {
			return err
		}
		continue
	}
	return nil
}

func (c *Coordinator) cleanupWorkspaceForRun(ctx context.Context, run *model.Run, manifest *RuntimeManifest) error {
	if run == nil || manifest == nil || manifest.Workspace == nil {
		return nil
	}
	provisioner, ok := c.workspaceProvisioners[manifest.Executor]
	if !ok {
		return nil
	}
	return provisioner.Cleanup(ctx, run, manifest.Workspace)
}

func (c *Coordinator) CleanupManagedRun(ctx context.Context, run *model.Run) error {
	if run == nil || !IsManagedRun(run) {
		return nil
	}
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil || manifest == nil || manifest.Workspace == nil {
		return err
	}
	return c.cleanupWorkspaceForRun(ctx, run, manifest)
}

func (c *Coordinator) TerminateManagedRun(ctx context.Context, run *model.Run) error {
	if run == nil || !IsManagedRun(run) {
		return nil
	}
	done := c.cancelActiveRun(run.UUID)
	if err := c.deleteManagedRunPods(ctx, run.UUID, run.Namespace); err != nil {
		return err
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		if cancellationRequested, err := c.runCancellationRequested(run.UUID); err != nil {
			return err
		} else if cancellationRequested {
			if err := c.finalizeCanceledRun(run.UUID, "run execution was cancelled"); err != nil {
				return err
			}
			if cleanupErr := c.CleanupManagedRun(ctx, run); cleanupErr != nil {
				glog.Warningf("Failed to cleanup managed run %s after cancellation: %v", run.UUID, cleanupErr)
			}
			return nil
		}
	}
	return nil
}

func (c *Coordinator) deleteManagedRunPods(ctx context.Context, runID string, namespace string) error {
	if c.kubernetesCoreClient == nil || namespace == "" {
		return nil
	}
	tasks, _, _, err := c.taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID},
	}, list.EmptyOptions())
	if err != nil {
		return err
	}
	for _, task := range tasks {
		podName := existingExecutorPodName(task)
		if podName == "" {
			continue
		}
		if err := c.kubernetesCoreClient.PodClient(namespace).Delete(ctx, podName, metav1.DeleteOptions{}); err != nil && !util.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func SupportsPipelineJob(job *pipelinespec.PipelineJob) error {
	if job == nil {
		return fmt.Errorf("pipeline job is nil")
	}
	spec, err := backendcompiler.GetPipelineSpec(job)
	if err != nil {
		return err
	}
	return validateSupportedComponent(spec, spec.GetRoot())
}

func (c *Coordinator) rebuildQueuedRun(run *model.Run) (*queuedRun, error) {
	if run == nil {
		return nil, fmt.Errorf("run is nil")
	}
	templateBytes := []byte(run.PipelineSpecManifest)
	tmpl, err := apitemplate.New(templateBytes, apitemplate.TemplateOptions{})
	if err != nil {
		return nil, err
	}
	v2Spec, ok := tmpl.(*apitemplate.V2Spec)
	if !ok {
		return nil, fmt.Errorf("managed run %s is not backed by a v2 template", run.UUID)
	}
	job, kubernetesSpec, err := v2Spec.BuildPipelineJob(run, apitemplate.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	if err != nil {
		return nil, err
	}
	spec, err := backendcompiler.GetPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("runtime manifest is missing for managed run %s", run.UUID)
	}
	return &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		kubernetesSpec:  kubernetesSpec,
		runtimeManifest: manifest,
		workspace:       manifest.Workspace,
	}, nil
}

func validateSupportedTask(task *pipelinespec.PipelineTaskSpec) error {
	if task == nil {
		return fmt.Errorf("task spec is nil")
	}
	if strategy := task.GetTriggerPolicy().GetStrategy().String(); strategy != "" &&
		strategy != "ALL_UPSTREAM_TASKS_COMPLETED" &&
		strategy != "TRIGGER_STRATEGY_UNSPECIFIED" {
		return fmt.Errorf("trigger strategy %q is not supported", strategy)
	}
	if task.GetArtifactIterator() != nil {
		return fmt.Errorf("artifact iterator tasks are not supported")
	}
	if task.GetComponentRef().GetName() == "" {
		return fmt.Errorf("component reference is required")
	}
	return nil
}

func validateSupportedComponent(spec *pipelinespec.PipelineSpec, component *pipelinespec.ComponentSpec) error {
	if component == nil {
		return fmt.Errorf("component is nil")
	}
	dag := component.GetDag()
	if dag == nil {
		return nil
	}
	for taskName, task := range dag.GetTasks() {
		if err := validateSupportedTask(task); err != nil {
			return fmt.Errorf("task %q is not supported: %w", taskName, err)
		}
		childComponent := spec.GetComponents()[task.GetComponentRef().GetName()]
		if err := validateSupportedComponent(spec, childComponent); err != nil {
			return err
		}
	}
	return nil
}

func taskDisplayName(task *pipelinespec.PipelineTaskSpec) string {
	if task == nil {
		return ""
	}
	if name := task.GetTaskInfo().GetName(); name != "" {
		return name
	}
	return task.GetComponentRef().GetName()
}

func resolveImporterArtifactURI(
	importerSpec *pipelinespec.PipelineDeploymentConfig_ImporterSpec,
	parameterValues map[string]*structpb.Value,
) (string, error) {
	if importerSpec == nil {
		return "", fmt.Errorf("importer spec is nil")
	}
	switch {
	case importerSpec.GetArtifactUri().GetConstant() != nil:
		artifactURI := importerSpec.GetArtifactUri().GetConstant().GetStringValue()
		if artifactURI == "" {
			return "", fmt.Errorf("empty artifact URI constant value")
		}
		return artifactURI, nil
	case importerSpec.GetArtifactUri().GetRuntimeParameter() != "":
		paramName := importerSpec.GetArtifactUri().GetRuntimeParameter()
		value, ok := parameterValues[paramName]
		if !ok || value == nil {
			return "", fmt.Errorf("artifact URI runtime parameter %q not found", paramName)
		}
		artifactURI := value.GetStringValue()
		if artifactURI == "" {
			return "", fmt.Errorf("empty artifact URI runtime value for parameter %q", paramName)
		}
		return artifactURI, nil
	default:
		return "", fmt.Errorf("artifact uri not provided")
	}
}

func importerArtifactOutputKey(componentSpec *pipelinespec.ComponentSpec) (string, error) {
	if componentSpec == nil || componentSpec.GetOutputDefinitions() == nil {
		return "", fmt.Errorf("component output artifact definition is missing")
	}
	outputNames := make([]string, 0, len(componentSpec.GetOutputDefinitions().GetArtifacts()))
	for name := range componentSpec.GetOutputDefinitions().GetArtifacts() {
		outputNames = append(outputNames, name)
	}
	if len(outputNames) != 1 {
		return "", fmt.Errorf("failed to extract output artifact name from component output spec")
	}
	return outputNames[0], nil
}

func importerArtifactType(typeSchema *pipelinespec.ArtifactTypeSchema) (apiv2beta1.Artifact_ArtifactType, error) {
	if typeSchema == nil || typeSchema.GetSchemaTitle() == "" {
		return apiv2beta1.Artifact_TYPE_UNSPECIFIED, fmt.Errorf("importer type schema title is required")
	}
	switch typeSchema.GetSchemaTitle() {
	case "system.Artifact":
		return apiv2beta1.Artifact_Artifact, nil
	case "system.Dataset":
		return apiv2beta1.Artifact_Dataset, nil
	case "system.Model":
		return apiv2beta1.Artifact_Model, nil
	case "system.Metrics":
		return apiv2beta1.Artifact_Metric, nil
	case "system.ClassificationMetrics":
		return apiv2beta1.Artifact_ClassificationMetric, nil
	case "system.SlicedClassificationMetrics":
		return apiv2beta1.Artifact_SlicedClassificationMetric, nil
	case "system.HTML":
		return apiv2beta1.Artifact_HTML, nil
	case "system.Markdown":
		return apiv2beta1.Artifact_Markdown, nil
	default:
		return apiv2beta1.Artifact_TYPE_UNSPECIFIED, fmt.Errorf("unknown artifact type: %s", typeSchema.GetSchemaTitle())
	}
}

func importerArtifactMetadata(importerSpec *pipelinespec.PipelineDeploymentConfig_ImporterSpec) model.JSONData {
	metadata := model.JSONData{}
	if importerSpec != nil && importerSpec.GetMetadata() != nil {
		for key, value := range importerSpec.GetMetadata().AsMap() {
			metadata[key] = value
		}
	}
	return metadata
}

func importerArtifactName(uriValue string) string {
	parsed, err := url.Parse(uriValue)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return path.Base(parsed.Path)
	}
	if err == nil && parsed.Scheme != "" {
		return path.Base(parsed.Path)
	}
	return path.Base(strings.TrimSuffix(uriValue, "/"))
}

func importerWorkspaceArtifactLocalPath(workspace *WorkspaceHandle, artifactURI string) (string, error) {
	localPath, err := component.LocalWorkspacePathForURI(artifactURI)
	if err != nil {
		return "", err
	}
	if workspace != nil && workspace.Type == WorkspaceTypeHostPath && workspace.Metadata != nil {
		workspacePath := workspace.Metadata["path"]
		if workspacePath == "" {
			return "", fmt.Errorf("workspace host path is empty")
		}
		relativePath, err := filepath.Rel(component.WorkspaceMountPath, localPath)
		if err != nil {
			return "", err
		}
		return filepath.Join(workspacePath, relativePath), nil
	}
	return localPath, nil
}

func shouldDownloadImporterArtifactToWorkspace(workspace *WorkspaceHandle) bool {
	return workspace != nil && workspace.Type == WorkspaceTypeHostPath
}

func taskTypeAttributesDownloadToWorkspace(typeAttrs model.JSONData) bool {
	if typeAttrs == nil {
		return false
	}
	value, ok := typeAttrs["downloadToWorkspace"]
	if !ok {
		return false
	}
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func taskTypeAttributes(
	task *pipelinespec.PipelineTaskSpec,
	importerSpec *pipelinespec.PipelineDeploymentConfig_ImporterSpec,
) (model.JSONData, error) {
	typeAttrs := &apiv2beta1.PipelineTaskDetail_TypeAttributes{}
	if importerSpec != nil && importerSpec.GetDownloadToWorkspace() {
		typeAttrs.DownloadToWorkspace = util.BoolPointer(true)
	}
	return model.ProtoMessageToJSONData(typeAttrs)
}

func dependenciesSatisfied(task *pipelinespec.PipelineTaskSpec, completed map[string]struct{}) bool {
	for _, dependency := range task.GetDependentTasks() {
		if _, ok := completed[dependency]; !ok {
			return false
		}
	}
	return true
}

func (m *RuntimeManifest) taskByScope(scopePath string) (RuntimeTask, bool) {
	if m == nil {
		return RuntimeTask{}, false
	}
	for _, task := range m.Tasks {
		if task.ScopePath == scopePath {
			return task, true
		}
	}
	return RuntimeTask{}, false
}

func containerForComponent(spec *pipelinespec.PipelineSpec, componentSpec *pipelinespec.ComponentSpec) (*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if spec == nil || componentSpec == nil {
		return nil, fmt.Errorf("pipeline spec and component spec are required")
	}
	deploymentConfig, err := backendcompiler.GetDeploymentConfig(spec)
	if err != nil {
		return nil, err
	}
	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, fmt.Errorf("component executor label is empty")
	}
	executorSpec := deploymentConfig.GetExecutors()[executorLabel]
	if executorSpec == nil || executorSpec.GetContainer() == nil {
		return nil, fmt.Errorf("container executor %q not found", executorLabel)
	}
	return executorSpec.GetContainer(), nil
}

func importerSpecForComponent(spec *pipelinespec.PipelineSpec, componentSpec *pipelinespec.ComponentSpec) (*pipelinespec.PipelineDeploymentConfig_ImporterSpec, error) {
	if spec == nil || componentSpec == nil {
		return nil, fmt.Errorf("pipeline spec and component spec are required")
	}
	deploymentConfig, err := backendcompiler.GetDeploymentConfig(spec)
	if err != nil {
		return nil, err
	}
	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, nil
	}
	executorSpec := deploymentConfig.GetExecutors()[executorLabel]
	if executorSpec == nil {
		return nil, nil
	}
	return executorSpec.GetImporter(), nil
}

func (c *Coordinator) buildExecutorInput(
	item *queuedRun,
	scopePath string,
	taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	parameterValues map[string]*structpb.Value,
) (*pipelinespec.ExecutorInput, map[string]pipelinespec.ParameterType_ParameterTypeEnum, error) {
	inputArtifacts, err := c.resolveInputArtifacts(item, scopePath, taskSpec)
	if err != nil {
		return nil, nil, err
	}
	pipelineRoot, err := c.getPipelineRoot(item.run)
	if err != nil {
		return nil, nil, err
	}
	outputParameterTypes := map[string]pipelinespec.ParameterType_ParameterTypeEnum{}
	var outputs *pipelinespec.ExecutorInput_Outputs
	if err := withArtifactLocalRootEnv(c.artifactLocalRoot(item.run, RequestedExecutor()), func() error {
		outputs = driver.BuildProvisionedOutputs(
			pipelineRoot,
			path.Base(scopePath),
			componentSpec.GetOutputDefinitions(),
			uuid.NewString(),
			"true",
		)
		return nil
	}); err != nil {
		return nil, nil, err
	}
	if componentSpec != nil && componentSpec.GetOutputDefinitions() != nil {
		for name, paramSpec := range componentSpec.GetOutputDefinitions().GetParameters() {
			outputParameterTypes[name] = paramSpec.GetParameterType()
		}
	}
	return &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: parameterValues,
			Artifacts:       inputArtifacts,
		},
		Outputs: outputs,
	}, outputParameterTypes, nil
}

func loadKubernetesExecutorConfig(
	componentSpec *pipelinespec.ComponentSpec,
	kubernetesSpec *pipelinespec.SinglePlatformSpec,
) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	if componentSpec == nil || kubernetesSpec == nil || kubernetesSpec.GetDeploymentSpec() == nil {
		return nil, nil
	}
	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, nil
	}
	executor := kubernetesSpec.GetDeploymentSpec().GetExecutors()[executorLabel]
	if executor == nil {
		return nil, nil
	}
	cfg := &kubernetesplatform.KubernetesExecutorConfig{}
	jsonBytes, err := protojson.Marshal(executor)
	if err != nil {
		return nil, err
	}
	if err := protojson.Unmarshal(jsonBytes, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Coordinator) buildResolverTaskDetails(
	manifest *RuntimeManifest,
	parentTaskID string,
) ([]*apiv2beta1.PipelineTaskDetail, *apiv2beta1.PipelineTaskDetail, error) {
	if manifest == nil || len(manifest.Tasks) == 0 {
		return nil, nil, nil
	}
	apiTasks := make([]*apiv2beta1.PipelineTaskDetail, 0, len(manifest.Tasks))
	var parentTask *apiv2beta1.PipelineTaskDetail
	for _, runtimeTask := range manifest.Tasks {
		modelTask, err := c.taskStore.GetTask(runtimeTask.TaskID)
		if err != nil {
			return nil, nil, err
		}
		apiTask := &apiv2beta1.PipelineTaskDetail{
			TaskId:           modelTask.UUID,
			RunId:            modelTask.RunUUID,
			ParentTaskId:     modelTask.ParentTaskUUID,
			Name:             modelTask.Name,
			DisplayName:      modelTask.DisplayName,
			CacheFingerprint: modelTask.Fingerprint,
			State:            apiv2beta1.PipelineTaskDetail_TaskState(modelTask.State),
			Type:             apiv2beta1.PipelineTaskDetail_TaskType(modelTask.Type),
			Inputs:           &apiv2beta1.PipelineTaskDetail_InputOutputs{},
			Outputs:          &apiv2beta1.PipelineTaskDetail_InputOutputs{},
			ScopePath:        modelTask.ScopePath,
		}
		if len(modelTask.OutputParameters) > 0 {
			outputParameters, err := model.JSONSliceToProtoSlice(
				modelTask.OutputParameters,
				func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
					return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
				},
			)
			if err != nil {
				return nil, nil, err
			}
			apiTask.Outputs.Parameters = outputParameters
		}
		if modelTask.TypeAttrs != nil {
			typeAttributes, err := model.JSONDataToProtoMessage(
				modelTask.TypeAttrs,
				func() *apiv2beta1.PipelineTaskDetail_TypeAttributes {
					return &apiv2beta1.PipelineTaskDetail_TypeAttributes{}
				},
			)
			if err != nil {
				return nil, nil, err
			}
			apiTask.TypeAttributes = typeAttributes
		}
		apiTasks = append(apiTasks, apiTask)
		if modelTask.UUID == parentTaskID {
			parentTask = apiTask
		}
	}
	return apiTasks, parentTask, nil
}

func resolvePVCNames(
	cfg *kubernetesplatform.KubernetesExecutorConfig,
	executorInput *pipelinespec.ExecutorInput,
) ([]string, error) {
	if cfg == nil || cfg.GetPvcMount() == nil {
		return nil, nil
	}
	values := map[string]*structpb.Value{}
	if executorInput != nil && executorInput.GetInputs() != nil {
		values = executorInput.GetInputs().GetParameterValues()
	}
	var pvcNames []string
	for _, mount := range cfg.GetPvcMount() {
		if pvcName := resolvePVCNameForMount(mount, values); pvcName != "" {
			pvcNames = append(pvcNames, pvcName)
		}
	}
	return pvcNames, nil
}

func (c *Coordinator) lookupCachedTask(
	componentSpec *pipelinespec.ComponentSpec,
	taskSpec *pipelinespec.PipelineTaskSpec,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	executorInput *pipelinespec.ExecutorInput,
	namespace string,
	pvcNames []string,
) (string, *model.Task, error) {
	if taskSpec == nil || componentSpec == nil || container == nil || c.taskStore == nil {
		return "", nil, nil
	}
	if taskSpec.GetCachingOptions() == nil || !taskSpec.GetCachingOptions().GetEnableCache() {
		return "", nil, nil
	}
	fingerprint, err := driver.BuildCacheFingerprint(drivercommon.Options{
		Task:      taskSpec,
		Component: componentSpec,
		Container: container,
	}, executorInput, pvcNames)
	if err != nil {
		return "", nil, err
	}
	cachedTask, err := c.taskStore.FindLatestSucceededTaskByFingerprint(namespace, fingerprint)
	if err != nil {
		return "", nil, err
	}
	return fingerprint, cachedTask, nil
}

func (c *Coordinator) applyCachedTask(
	run *model.Run,
	taskID string,
	taskSpec *pipelinespec.PipelineTaskSpec,
	cachedTask *model.Task,
) error {
	if cachedTask == nil {
		return nil
	}
	if c.artifactTaskStore != nil {
		existingArtifactLinks := map[string]struct{}{}
		existingArtifactTasks, _, _, err := c.artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: taskID}},
			{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
		}, nil, list.EmptyOptions())
		if err != nil {
			return err
		}
		for _, existingArtifactTask := range existingArtifactTasks {
			linkKey := fmt.Sprintf("%s|%s|%v", existingArtifactTask.ArtifactID, existingArtifactTask.ArtifactKey, existingArtifactTask.Type)
			existingArtifactLinks[linkKey] = struct{}{}
		}

		artifactTasks, _, _, err := c.artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: cachedTask.UUID}},
		}, nil, list.EmptyOptions())
		if err != nil {
			return err
		}
		var copies []*model.ArtifactTask
		for _, artifactTask := range artifactTasks {
			linkKey := fmt.Sprintf("%s|%s|%v", artifactTask.ArtifactID, artifactTask.ArtifactKey, artifactTask.Type)
			if _, exists := existingArtifactLinks[linkKey]; exists {
				continue
			}
			copy := *artifactTask
			copy.UUID = ""
			copy.TaskID = taskID
			copy.RunUUID = run.UUID
			copies = append(copies, &copy)
			existingArtifactLinks[linkKey] = struct{}{}
		}
		if len(copies) > 0 {
			if _, err := c.artifactTaskStore.CreateArtifactTasks(copies); err != nil {
				return err
			}
		}
	}
	_, err := c.updateTaskState(taskID, apiv2beta1.PipelineTaskDetail_CACHED, customStatusMetadata("", map[string]any{
		"cachedFromTask": cachedTask.UUID,
		"taskName":       taskDisplayName(taskSpec),
	}), cachedTask.OutputParameters)
	return err
}

func (c *Coordinator) resolveInputParameters(
	item *queuedRun,
	runtimeTask RuntimeTask,
	scopePath string,
	taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	scopeInputs map[string]*structpb.Value,
) (map[string]*structpb.Value, error) {
	if taskSpec == nil || taskSpec.GetInputs() == nil {
		return cloneParameterMap(scopeInputs), nil
	}
	runParameters, err := runtimeParameters(item.run, item.spec.GetRoot())
	if err != nil {
		return nil, err
	}
	values := cloneParameterMap(scopeInputs)
	parentScope := parentScopePath(scopePath)
	for name, inputSpec := range taskSpec.GetInputs().GetParameters() {
		switch {
		case inputSpec.GetComponentInputParameter() != "":
			if value, ok := scopeInputs[inputSpec.GetComponentInputParameter()]; ok {
				values[name] = value
			} else if value, ok := runParameters[inputSpec.GetComponentInputParameter()]; ok {
				values[name] = value
			}
		case inputSpec.GetTaskOutputParameter() != nil:
			producerScope := util.StringPathToDotNotation([]string{parentScope, inputSpec.GetTaskOutputParameter().GetProducerTask()})
			producerTask, ok := item.runtimeManifest.taskByScope(producerScope)
			if !ok {
				return nil, fmt.Errorf("producer task %q not found for scope %q", inputSpec.GetTaskOutputParameter().GetProducerTask(), scopePath)
			}
			value, err := c.lookupTaskOutputParameterByRuntimeTask(
				producerTask,
				inputSpec.GetTaskOutputParameter().GetOutputParameterKey(),
			)
			if err != nil {
				return nil, err
			}
			values[name] = value
		case inputSpec.GetTaskFinalStatus() != nil:
			value, err := c.resolveTaskFinalStatusInput(item, parentScope, inputSpec)
			if err != nil {
				return nil, err
			}
			values[name] = value
		case inputSpec.GetRuntimeValue() != nil:
			runtimeValue := inputSpec.GetRuntimeValue()
			if constant := runtimeValue.GetConstant(); constant != nil {
				if stringValue := constant.GetStringValue(); strings.Contains(stringValue, "{{$.workspace_path}}") {
					values[name] = structpb.NewStringValue(strings.ReplaceAll(stringValue, "{{$.workspace_path}}", component.WorkspaceMountPath))
				} else if resolvedValue, ok := resolvePipelineRuntimePlaceholder(item, runtimeTask, stringValue); ok {
					values[name] = resolvedValue
				} else {
					values[name] = constant
					if isMatch, isPipelineChannel, paramName := drivercommon.ParsePipelineParam(stringValue); isMatch && isPipelineChannel {
						if channelSpec, ok := taskSpec.GetInputs().GetParameters()[paramName]; ok {
							if channelValue, ok := scopeInputs[channelSpec.GetComponentInputParameter()]; ok && channelSpec.GetComponentInputParameter() != "" {
								values[name] = channelValue
							}
						}
					}
				}
			} else if runtimeParameter := runtimeValue.GetRuntimeParameter(); runtimeParameter != "" {
				if value, ok := runParameters[runtimeParameter]; ok {
					values[name] = value
				}
			}
		}
	}
	if componentSpec != nil && componentSpec.GetInputDefinitions() != nil {
		for name, inputSpec := range componentSpec.GetInputDefinitions().GetParameters() {
			if _, ok := values[name]; ok {
				continue
			}
			if inputSpec == nil || !inputSpec.GetIsOptional() || inputSpec.GetDefaultValue() == nil {
				continue
			}
			values[name] = inputSpec.GetDefaultValue()
		}
	}
	return values, nil
}

func (c *Coordinator) resolveTaskFinalStatusInput(
	item *queuedRun,
	parentScope string,
	inputSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
) (*structpb.Value, error) {
	if item == nil || item.runtimeManifest == nil || inputSpec == nil || inputSpec.GetTaskFinalStatus() == nil {
		return nil, fmt.Errorf("task final status input is incomplete")
	}
	producerTaskName := inputSpec.GetTaskFinalStatus().GetProducerTask()
	if producerTaskName == "" {
		return nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerScope := util.StringPathToDotNotation([]string{parentScope, producerTaskName})
	producerTask, ok := item.runtimeManifest.taskByScope(producerScope)
	if !ok {
		return nil, fmt.Errorf("producer task %q not found for scope %q", producerTaskName, parentScope)
	}
	modelTask, err := c.taskStore.GetTask(producerTask.TaskID)
	if err != nil {
		return nil, err
	}

	finalStatus := pipelinespec.PipelineTaskFinalStatus{
		State:                   pipelineTaskFinalStatusState(modelTask.State),
		PipelineTaskName:        producerTask.Name,
		PipelineJobResourceName: pipelineJobResourceName(item),
		Error: &status.Status{
			Message: statusMetadataMessage(modelTask.StatusMetadata),
			Code:    int32(modelTask.State),
		},
	}
	finalStatusJSON, err := protojson.Marshal(&finalStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PipelineTaskFinalStatus: %w", err)
	}
	finalStatusValue := &structpb.Value{}
	if err := finalStatusValue.UnmarshalJSON(finalStatusJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PipelineTaskFinalStatus: %w", err)
	}
	return finalStatusValue, nil
}

func resolvePipelineRuntimePlaceholder(
	item *queuedRun,
	runtimeTask RuntimeTask,
	raw string,
) (*structpb.Value, bool) {
	if item == nil || item.run == nil {
		return nil, false
	}
	run := item.run
	if run == nil {
		return nil, false
	}
	switch raw {
	case "{{$.pipeline_job_name}}":
		return structpb.NewStringValue(run.DisplayName), true
	case "{{$.pipeline_job_resource_name}}":
		if item.job != nil && item.job.GetName() != "" {
			return structpb.NewStringValue(item.job.GetName()), true
		}
		if item.spec != nil && item.spec.GetPipelineInfo().GetName() != "" {
			return structpb.NewStringValue(fmt.Sprintf("%s-%s", item.spec.GetPipelineInfo().GetName(), run.UUID)), true
		}
		return structpb.NewStringValue(run.K8SName), true
	case "{{$.pipeline_job_uuid}}":
		return structpb.NewStringValue(run.UUID), true
	case "{{$.pipeline_task_name}}":
		return structpb.NewStringValue(runtimeTask.Name), true
	case "{{$.pipeline_task_uuid}}":
		return structpb.NewStringValue(runtimeTask.TaskID), true
	default:
		return nil, false
	}
}

func pipelineJobResourceName(item *queuedRun) string {
	if item == nil || item.run == nil {
		return ""
	}
	if item.job != nil && item.job.GetName() != "" {
		return item.job.GetName()
	}
	if item.spec != nil && item.spec.GetPipelineInfo().GetName() != "" {
		return fmt.Sprintf("%s-%s", item.spec.GetPipelineInfo().GetName(), item.run.UUID)
	}
	return item.run.K8SName
}

func pipelineTaskFinalStatusState(state model.TaskStatus) string {
	switch apiv2beta1.PipelineTaskDetail_TaskState(state) {
	case apiv2beta1.PipelineTaskDetail_SUCCEEDED,
		apiv2beta1.PipelineTaskDetail_SKIPPED,
		apiv2beta1.PipelineTaskDetail_CACHED:
		return "COMPLETE"
	case apiv2beta1.PipelineTaskDetail_FAILED:
		return "FAILED"
	default:
		return apiv2beta1.PipelineTaskDetail_TaskState(state).String()
	}
}

func statusMetadataMessage(metadata model.JSONData) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata["message"].(string); ok {
		return value
	}
	return ""
}

func (c *Coordinator) executeParameterIteratorTask(
	ctx context.Context,
	item *queuedRun,
	executor WorkloadExecutorPlugin,
	runtimeTask RuntimeTask,
	taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	scopePath string,
	taskInputs map[string]*structpb.Value,
) error {
	iterator := taskSpec.GetParameterIterator()
	if iterator == nil {
		return fmt.Errorf("parameter iterator is nil")
	}
	items, err := iteratorItems(iterator, taskInputs)
	if err != nil {
		return err
	}
	for _, itemValue := range items {
		iterationInputs := cloneParameterMap(taskInputs)
		iterationInputs[iterator.GetItemInput()] = itemValue
		if componentSpec.GetDag() != nil {
			if err := c.executeDagComponent(ctx, item, executor, componentSpec, scopePath, iterationInputs, true); err != nil {
				return err
			}
			continue
		}
		if err := c.executeRuntimeTask(ctx, item, executor, runtimeTask, taskSpec, componentSpec, iterationInputs); err != nil {
			return err
		}
	}
	return nil
}

func iteratorItems(
	iterator *pipelinespec.ParameterIteratorSpec,
	taskInputs map[string]*structpb.Value,
) ([]*structpb.Value, error) {
	if iterator == nil {
		return nil, fmt.Errorf("parameter iterator is nil")
	}
	switch items := iterator.GetItems().GetKind().(type) {
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
		value := &structpb.Value{}
		if err := value.UnmarshalJSON([]byte(items.Raw)); err != nil {
			return nil, err
		}
		if list := value.GetListValue(); list != nil {
			return list.GetValues(), nil
		}
		return nil, fmt.Errorf("parameter iterator raw items are not a list")
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
		value, ok := taskInputs[items.InputParameter]
		if !ok {
			return nil, fmt.Errorf("parameter iterator input %q not found", items.InputParameter)
		}
		switch typed := value.GetKind().(type) {
		case *structpb.Value_ListValue:
			return typed.ListValue.GetValues(), nil
		case *structpb.Value_StringValue:
			listValue := &structpb.Value{}
			if err := listValue.UnmarshalJSON([]byte(typed.StringValue)); err != nil {
				return nil, err
			}
			if list := listValue.GetListValue(); list != nil {
				return list.GetValues(), nil
			}
			return nil, fmt.Errorf("parameter iterator input %q is not a list", items.InputParameter)
		default:
			return nil, fmt.Errorf("parameter iterator input %q is not iterable", items.InputParameter)
		}
	default:
		return nil, fmt.Errorf("unsupported parameter iterator items kind %T", items)
	}
}

func cloneParameterMap(values map[string]*structpb.Value) map[string]*structpb.Value {
	if len(values) == 0 {
		return map[string]*structpb.Value{}
	}
	cloned := make(map[string]*structpb.Value, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func runtimeParameters(run *model.Run, componentSpec *pipelinespec.ComponentSpec) (map[string]*structpb.Value, error) {
	values := map[string]*structpb.Value{}
	if run == nil || run.RuntimeConfig.Parameters == "" {
		return mergeDefaultRuntimeParameters(values, componentSpec), nil
	}
	raw := map[string]any{}
	if err := json.Unmarshal([]byte(run.RuntimeConfig.Parameters), &raw); err != nil {
		return nil, err
	}
	for key, value := range raw {
		pbValue, err := structpb.NewValue(value)
		if err != nil {
			return nil, err
		}
		values[key] = pbValue
	}
	return mergeDefaultRuntimeParameters(values, componentSpec), nil
}

func mergeDefaultRuntimeParameters(
	values map[string]*structpb.Value,
	componentSpec *pipelinespec.ComponentSpec,
) map[string]*structpb.Value {
	if componentSpec == nil || componentSpec.GetInputDefinitions() == nil {
		return values
	}
	for name, inputSpec := range componentSpec.GetInputDefinitions().GetParameters() {
		if _, ok := values[name]; ok {
			continue
		}
		if inputSpec == nil || !inputSpec.GetIsOptional() || inputSpec.GetDefaultValue() == nil {
			continue
		}
		values[name] = inputSpec.GetDefaultValue()
	}
	return values
}

func (c *Coordinator) propagateDagOutputs(
	item *queuedRun,
	runtimeTask RuntimeTask,
	componentSpec *pipelinespec.ComponentSpec,
	scopePath string,
) error {
	if item == nil || componentSpec == nil || componentSpec.GetDag() == nil || item.run == nil {
		return nil
	}
	outputs := componentSpec.GetDag().GetOutputs()
	if outputs == nil {
		return nil
	}

	var parameterOutputs model.JSONSlice
	if len(outputs.GetParameters()) > 0 {
		parameterOutputs = model.JSONSlice{}
	}
	for outputKey, outputSpec := range outputs.GetParameters() {
		value, err := c.resolveDagOutputParameter(item, scopePath, outputSpec)
		if err != nil {
			return fmt.Errorf("failed to resolve DAG output parameter %q: %w", outputKey, err)
		}
		if value == nil {
			continue
		}
		param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			ParameterKey: outputKey,
			Value:        value,
			Type:         apiv2beta1.IOType_OUTPUT,
			Producer: &apiv2beta1.IOProducer{
				TaskName: runtimeTask.Name,
			},
		}
		jsonValue, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param})
		if err != nil {
			return err
		}
		parameterOutputs = append(parameterOutputs, jsonValue...)
	}
	if len(parameterOutputs) > 0 {
		if _, err := c.updateTask(&model.Task{
			UUID:             runtimeTask.TaskID,
			OutputParameters: parameterOutputs,
		}); err != nil {
			return err
		}
	}

	existingArtifactLinks := map[string]struct{}{}
	if c.artifactTaskStore != nil {
		existingArtifactTasks, _, _, err := c.artifactTaskStore.ListArtifactTasks(
			[]*model.FilterContext{
				{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: runtimeTask.TaskID}},
				{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: item.run.UUID}},
			},
			nil,
			list.EmptyOptions(),
		)
		if err != nil {
			return err
		}
		for _, existingArtifactTask := range existingArtifactTasks {
			linkKey := fmt.Sprintf("%s|%s|%v", existingArtifactTask.ArtifactID, existingArtifactTask.ArtifactKey, existingArtifactTask.Type)
			existingArtifactLinks[linkKey] = struct{}{}
		}
	}

	for outputKey, outputSpec := range outputs.GetArtifacts() {
		for _, selector := range outputSpec.GetArtifactSelectors() {
			producerTask, artifactTasks, err := c.resolveDagOutputArtifacts(item, scopePath, selector)
			if err != nil {
				return fmt.Errorf("failed to resolve DAG output artifact %q: %w", outputKey, err)
			}
			for _, artifactTask := range artifactTasks {
				producer := model.JSONData{
					"task_name": producerTask.Name,
				}
				linkKey := fmt.Sprintf("%s|%s|%v", artifactTask.ArtifactID, outputKey, model.IOType(apiv2beta1.IOType_OUTPUT))
				if _, exists := existingArtifactLinks[linkKey]; exists {
					continue
				}
				if _, err := c.artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
					ArtifactID:  artifactTask.ArtifactID,
					TaskID:      runtimeTask.TaskID,
					RunUUID:     item.run.UUID,
					Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
					ArtifactKey: outputKey,
					Producer:    producer,
				}); err != nil {
					return err
				}
				existingArtifactLinks[linkKey] = struct{}{}
			}
		}
	}

	return nil
}

func (c *Coordinator) resolveDagOutputParameter(
	item *queuedRun,
	scopePath string,
	outputSpec *pipelinespec.DagOutputsSpec_DagOutputParameterSpec,
) (*structpb.Value, error) {
	if outputSpec == nil {
		return nil, nil
	}
	if selector := outputSpec.GetValueFromParameter(); selector != nil {
		return c.lookupTaskOutputParameterInScope(item, scopePath, selector.GetProducerSubtask(), selector.GetOutputParameterKey())
	}
	if oneof := outputSpec.GetValueFromOneof(); oneof != nil {
		for _, selector := range oneof.GetParameterSelectors() {
			value, err := c.lookupTaskOutputParameterInScope(item, scopePath, selector.GetProducerSubtask(), selector.GetOutputParameterKey())
			if err == nil && value != nil {
				return value, nil
			}
		}
	}
	return nil, nil
}

func (c *Coordinator) lookupTaskOutputParameter(
	item *queuedRun,
	scopePath string,
	producerSubtask string,
	outputParameterKey string,
) (*structpb.Value, error) {
	producerTask, err := c.lookupRuntimeTaskBySubtask(item, scopePath, producerSubtask)
	if err != nil {
		return nil, err
	}
	return c.lookupTaskOutputParameterByRuntimeTask(producerTask, outputParameterKey)
}

func (c *Coordinator) lookupTaskOutputParameterInScope(
	item *queuedRun,
	scopePath string,
	producerSubtask string,
	outputParameterKey string,
) (*structpb.Value, error) {
	producerTask, err := c.lookupRuntimeTaskInScope(item, scopePath, producerSubtask)
	if err != nil {
		return nil, err
	}
	return c.lookupTaskOutputParameterByRuntimeTask(producerTask, outputParameterKey)
}

func (c *Coordinator) lookupTaskOutputParameterByRuntimeTask(
	producerTask RuntimeTask,
	outputParameterKey string,
) (*structpb.Value, error) {
	modelTask, err := c.taskStore.GetTask(producerTask.TaskID)
	if err != nil {
		return nil, err
	}
	outputs, err := model.JSONSliceToProtoSlice(
		modelTask.OutputParameters,
		func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
			return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
		},
	)
	if err != nil {
		return nil, err
	}
	matches := make([]*structpb.Value, 0, len(outputs))
	selfMatches := make([]*structpb.Value, 0, len(outputs))
	for _, output := range outputs {
		if output.GetParameterKey() == outputParameterKey {
			matches = append(matches, output.GetValue())
			if output.GetProducer() != nil && output.GetProducer().GetTaskName() == producerTask.Name {
				selfMatches = append(selfMatches, output.GetValue())
			}
		}
	}
	if len(selfMatches) > 0 {
		matches = selfMatches
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("output parameter %q not found on task %q", outputParameterKey, producerTask.Name)
	case 1:
		return matches[0], nil
	default:
		return &structpb.Value{
			Kind: &structpb.Value_ListValue{
				ListValue: &structpb.ListValue{Values: matches},
			},
		}, nil
	}
}

func (c *Coordinator) resolveDagOutputArtifacts(
	item *queuedRun,
	scopePath string,
	selector *pipelinespec.DagOutputsSpec_ArtifactSelectorSpec,
) (RuntimeTask, []*model.ArtifactTask, error) {
	producerTask, err := c.lookupRuntimeTaskInScope(item, scopePath, selector.GetProducerSubtask())
	if err != nil {
		return RuntimeTask{}, nil, err
	}
	artifactTasks, _, _, err := c.artifactTaskStore.ListArtifactTasks(
		[]*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: producerTask.TaskID}},
			{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: item.run.UUID}},
		},
		nil,
		list.EmptyOptions(),
	)
	if err != nil {
		return RuntimeTask{}, nil, err
	}
	filtered := make([]*model.ArtifactTask, 0, len(artifactTasks))
	for _, artifactTask := range artifactTasks {
		if artifactTask.ArtifactKey == selector.GetOutputArtifactKey() {
			filtered = append(filtered, artifactTask)
		}
	}
	return producerTask, filtered, nil
}

func (c *Coordinator) lookupRuntimeTaskBySubtask(
	item *queuedRun,
	scopePath string,
	producerSubtask string,
) (RuntimeTask, error) {
	parentScope := scopePath
	if parent := parentScopePath(scopePath); parent != "" {
		parentScope = parent
	}
	producerScope := util.StringPathToDotNotation([]string{parentScope, producerSubtask})
	runtimeTask, ok := item.runtimeManifest.taskByScope(producerScope)
	if !ok {
		return RuntimeTask{}, fmt.Errorf("producer subtask %q not found for scope %q", producerSubtask, scopePath)
	}
	return runtimeTask, nil
}

func (c *Coordinator) lookupRuntimeTaskInScope(
	item *queuedRun,
	scopePath string,
	producerSubtask string,
) (RuntimeTask, error) {
	producerScope := util.StringPathToDotNotation([]string{scopePath, producerSubtask})
	runtimeTask, ok := item.runtimeManifest.taskByScope(producerScope)
	if !ok {
		return RuntimeTask{}, fmt.Errorf("producer subtask %q not found inside scope %q", producerSubtask, scopePath)
	}
	return runtimeTask, nil
}

func (c *Coordinator) resolveInputArtifacts(
	item *queuedRun,
	scopePath string,
	taskSpec *pipelinespec.PipelineTaskSpec,
) (map[string]*pipelinespec.ArtifactList, error) {
	artifacts := map[string]*pipelinespec.ArtifactList{}
	if taskSpec == nil || taskSpec.GetInputs() == nil || c.artifactTaskStore == nil || c.artifactStore == nil {
		return artifacts, nil
	}
	parentScope := parentScopePath(scopePath)
	for name, inputSpec := range taskSpec.GetInputs().GetArtifacts() {
		taskOutput := inputSpec.GetTaskOutputArtifact()
		if taskOutput == nil {
			continue
		}
		producerScope := util.StringPathToDotNotation([]string{parentScope, taskOutput.GetProducerTask()})
		runtimeTask, ok := item.runtimeManifest.taskByScope(producerScope)
		if !ok {
			return nil, fmt.Errorf("producer task %q not found for artifact input %q", taskOutput.GetProducerTask(), name)
		}
		filterContexts := []*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: runtimeTask.TaskID}},
			{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: item.run.UUID}},
		}
		artifactTasks, _, _, err := c.artifactTaskStore.ListArtifactTasks(filterContexts, nil, list.EmptyOptions())
		if err != nil {
			return nil, err
		}
		inputArtifacts := []*pipelinespec.RuntimeArtifact{}
		for _, artifactTask := range artifactTasks {
			if artifactTask.ArtifactKey != taskOutput.GetOutputArtifactKey() {
				continue
			}
			artifact, err := c.artifactStore.GetArtifact(artifactTask.ArtifactID)
			if err != nil {
				return nil, err
			}
			runtimeArtifact := modelArtifactToRuntimeArtifact(artifact)
			inputArtifacts = append(inputArtifacts, runtimeArtifact)
		}
		if len(inputArtifacts) > 0 {
			artifacts[name] = &pipelinespec.ArtifactList{Artifacts: inputArtifacts}
		}
	}
	return artifacts, nil
}

func (c *Coordinator) getPipelineRoot(run *model.Run) (string, error) {
	if run != nil && run.RuntimeConfig.PipelineRoot != "" {
		return string(run.RuntimeConfig.PipelineRoot), nil
	}
	if run == nil {
		return "", fmt.Errorf("run is nil")
	}
	var launcherConfig *config.Config
	var err error
	if c.kubernetesCoreClient != nil && c.kubernetesCoreClient.GetClientSet() != nil {
		launcherConfig, err = config.FetchLauncherConfigMap(context.Background(), c.kubernetesCoreClient.GetClientSet(), run.Namespace)
		if err != nil {
			return "", err
		}
	}
	defaultRoot := launcherConfig.DefaultPipelineRoot()
	runRootName := run.K8SName
	if runRootName == "" {
		runRootName = run.DisplayName
	}
	return util.GenerateOutputURI(defaultRoot, []string{runRootName, run.UUID}, true), nil
}

func modelArtifactToRuntimeArtifact(artifact *model.Artifact) *pipelinespec.RuntimeArtifact {
	if artifact == nil {
		return nil
	}
	runtimeArtifact := &pipelinespec.RuntimeArtifact{
		Name: artifact.Name,
	}
	if artifact.URI != nil {
		runtimeArtifact.Uri = *artifact.URI
	}
	runtimeArtifact.Type = artifactTypeSchemaFromModelArtifactType(artifact.Type)
	if artifact.Metadata != nil {
		fields := map[string]*structpb.Value{}
		if customPathValue, ok := artifact.Metadata[runtimeArtifactCustomPathMetadataKey]; ok {
			if customPath, ok := customPathValue.(string); ok && customPath != "" {
				runtimeArtifact.CustomPath = util.StringPointer(customPath)
			}
		}
		for key, value := range artifact.Metadata {
			if key == runtimeArtifactCustomPathMetadataKey {
				continue
			}
			pbValue, err := structpb.NewValue(value)
			if err == nil {
				fields[key] = pbValue
			}
		}
		runtimeArtifact.Metadata = &structpb.Struct{Fields: fields}
	}
	return runtimeArtifact
}

func artifactTypeSchemaFromModelArtifactType(artifactType model.ArtifactType) *pipelinespec.ArtifactTypeSchema {
	switch apiv2beta1.Artifact_ArtifactType(artifactType) {
	case apiv2beta1.Artifact_Dataset:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_Model:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_Metric:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Metrics"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_ClassificationMetric:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.ClassificationMetrics"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_SlicedClassificationMetric:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.SlicedClassificationMetrics"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_HTML:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.HTML"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_Markdown:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Markdown"},
			SchemaVersion: "0.0.1",
		}
	case apiv2beta1.Artifact_Artifact, apiv2beta1.Artifact_TYPE_UNSPECIFIED:
		fallthrough
	default:
		return &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"},
			SchemaVersion: "0.0.1",
		}
	}
}

func outputParametersToJSON(values map[string]*structpb.Value, taskSpec *pipelinespec.PipelineTaskSpec) model.JSONSlice {
	if len(values) == 0 {
		return nil
	}
	parameters := make([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(values))
	taskName := ""
	if taskSpec != nil && taskSpec.GetTaskInfo() != nil {
		taskName = taskSpec.GetTaskInfo().GetName()
	}
	for key, value := range values {
		parameters = append(parameters, &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			ParameterKey: key,
			Value:        value,
			Type:         apiv2beta1.IOType_OUTPUT,
			Producer: &apiv2beta1.IOProducer{
				TaskName: taskName,
			},
		})
	}
	jsonSlice, _ := model.ProtoSliceToJSONSlice(parameters)
	return jsonSlice
}

func evaluateTaskCondition(
	parameterValues map[string]*structpb.Value,
	condition string,
) (bool, error) {
	expr, err := expression.New()
	if err != nil {
		return false, err
	}
	return expr.Condition(&pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: parameterValues,
		},
	}, condition)
}

func (c *Coordinator) skipTaskTree(item *queuedRun, rootScope string) error {
	for _, runtimeTask := range item.runtimeManifest.Tasks {
		if runtimeTask.ScopePath != rootScope && !strings.HasPrefix(runtimeTask.ScopePath, rootScope+".") {
			continue
		}
		if _, err := c.updateTaskState(runtimeTask.TaskID, apiv2beta1.PipelineTaskDetail_SKIPPED, model.JSONData{
			"message": "condition not met",
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func parentScopePath(scopePath string) string {
	segments := util.DotNotationToStringPath(scopePath)
	if len(segments) <= 1 {
		return ""
	}
	return util.StringPathToDotNotation(segments[:len(segments)-1])
}

func sanitizePath(scopePath string) string {
	return strings.ReplaceAll(scopePath, ".", "_")
}

func taskStateForRunState(runState model.RuntimeState) apiv2beta1.PipelineTaskDetail_TaskState {
	switch runState {
	case model.RuntimeStateCanceled:
		return apiv2beta1.PipelineTaskDetail_SKIPPED
	default:
		return apiv2beta1.PipelineTaskDetail_FAILED
	}
}

func mergeJSONData(base model.JSONData, overlay model.JSONData) model.JSONData {
	if base == nil && overlay == nil {
		return nil
	}
	merged := model.JSONData{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		if key == "customProperties" {
			existing, _ := merged[key].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
			}
			switch typed := value.(type) {
			case map[string]any:
				for nestedKey, nestedValue := range typed {
					existing[nestedKey] = nestedValue
				}
				merged[key] = existing
				continue
			case model.JSONData:
				for nestedKey, nestedValue := range typed {
					existing[nestedKey] = nestedValue
				}
				merged[key] = existing
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func (c *Coordinator) artifactLocalRoot(run *model.Run, executorType string) string {
	if executorType == ExecutorKubernetes {
		return ""
	}
	return filepath.Join(WorkspaceRoot(), run.UUID, "artifact-local")
}

func (c *Coordinator) newObjectStoreClient(namespace string) component.ObjectStoreClientInterface {
	if c.kubernetesCoreClient == nil {
		return nil
	}
	clientSet := c.kubernetesCoreClient.GetClientSet()
	if clientSet == nil {
		return nil
	}
	launcherConfig := &config.Config{}
	if os.Getenv("LOCAL_API_SERVER") != "true" {
		fetchedLauncherConfig, err := config.FetchLauncherConfigMap(context.Background(), clientSet, namespace)
		if err == nil && fetchedLauncherConfig != nil {
			launcherConfig = fetchedLauncherConfig
		}
	}
	if namespace == "" {
		namespace = common.GetPodNamespace()
	}
	return component.NewObjectStoreClient(&runtimeObjectStoreDeps{
		openedBucketCache: map[string]*blob.Bucket{},
		launcherConfig:    launcherConfig,
		k8sClient:         clientSet,
		namespace:         namespace,
	})
}

func (c *Coordinator) publishOutputArtifacts(
	ctx context.Context,
	run *model.Run,
	taskID string,
	taskName string,
	executorInput *pipelinespec.ExecutorInput,
) error {
	if c.artifactStore == nil || c.artifactTaskStore == nil || executorInput == nil || executorInput.GetOutputs() == nil {
		return nil
	}
	objectStoreClient := c.newObjectStoreClient(run.Namespace)
	executorOutputArtifacts, err := loadExecutorOutputArtifacts(executorInput.GetOutputs().GetOutputFile())
	if err != nil {
		return err
	}
	for artifactKey, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if updatedArtifactList, ok := executorOutputArtifacts[artifactKey]; ok {
			for index, runtimeArtifact := range artifactList.Artifacts {
				if index < len(updatedArtifactList.Artifacts) {
					mergeRuntimeArtifacts(updatedArtifactList.Artifacts[index], runtimeArtifact)
				}
			}
		}
		for _, runtimeArtifact := range artifactList.Artifacts {
			artifactLocalRoot := c.artifactLocalRoot(run, ExecutorDocker)
			localPath, err := resolvedRuntimeArtifactLocalPath(
				artifactLocalRoot,
				dockerTempMountSourceForArtifactRoot(artifactLocalRoot),
				runtimeArtifact,
			)
			if err != nil {
				return err
			}
			if _, statErr := os.Stat(localPath); statErr != nil {
				if os.IsNotExist(statErr) && artifactKey == "executor-logs" {
					continue
				}
				if os.IsNotExist(statErr) && runtimeArtifactMayOmitLocalFile(runtimeArtifact) {
					localPath = ""
				} else {
					return statErr
				}
			}
			if localPath != "" && objectStoreClient != nil && runtimeArtifact.GetUri() != "" && !strings.HasPrefix(runtimeArtifact.GetUri(), "oci://") {
				if err := objectStoreClient.UploadArtifact(ctx, localPath, runtimeArtifact.GetUri(), artifactKey); err != nil {
					return err
				}
			}
			artifactModel := &model.Artifact{
				Namespace: run.Namespace,
				Name:      runtimeArtifact.GetName(),
				Metadata:  runtimeArtifactMetadata(runtimeArtifact),
			}
			if runtimeArtifact.GetUri() != "" {
				artifactModel.URI = util.StringPointer(runtimeArtifact.GetUri())
			}
			createdArtifact, err := c.artifactStore.CreateArtifact(artifactModel)
			if err != nil {
				return err
			}
			_, err = c.artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
				ArtifactID:  createdArtifact.UUID,
				TaskID:      taskID,
				RunUUID:     run.UUID,
				Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
				ArtifactKey: artifactKey,
				Producer: model.JSONData{
					"task_name": taskName,
				},
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func loadExecutorOutputArtifacts(outputFile string) (map[string]*pipelinespec.ArtifactList, error) {
	if outputFile == "" {
		return nil, nil
	}
	content, err := os.ReadFile(outputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read executor output file %q: %w", outputFile, err)
	}
	if len(content) == 0 {
		return nil, nil
	}
	executorOutput := &pipelinespec.ExecutorOutput{}
	if err := protojson.Unmarshal(content, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to parse executor output file %q: %w", outputFile, err)
	}
	return executorOutput.GetArtifacts(), nil
}

func mergeRuntimeArtifacts(src, dst *pipelinespec.RuntimeArtifact) {
	if src == nil || dst == nil {
		return
	}
	if src.GetUri() != "" {
		if strings.HasPrefix(src.GetUri(), "/") && dst.GetUri() != "" {
			customPath := src.GetUri()
			dst.CustomPath = &customPath
		} else {
			dst.Uri = src.Uri
		}
	}
	if src.GetMetadata() != nil {
		if dst.Metadata == nil {
			dst.Metadata = src.Metadata
		} else {
			for key, value := range src.GetMetadata().GetFields() {
				dst.Metadata.Fields[key] = value
			}
		}
	}
	if src.CustomPath != nil && src.GetCustomPath() != "" {
		dst.CustomPath = src.CustomPath
	}
}

func runtimeArtifactMayOmitLocalFile(artifact *pipelinespec.RuntimeArtifact) bool {
	if artifact == nil {
		return false
	}
	if strings.HasPrefix(artifact.GetUri(), "oci://") {
		return true
	}
	if artifact.GetType() == nil {
		return false
	}
	switch artifact.GetType().GetSchemaTitle() {
	case "system.Metrics", "system.ClassificationMetrics", "system.SlicedClassificationMetrics":
		return true
	default:
		return false
	}
}

func runtimeArtifactMetadata(artifact *pipelinespec.RuntimeArtifact) model.JSONData {
	metadata := model.JSONData{}
	if artifact == nil {
		return nil
	}
	if artifact.GetMetadata() != nil {
		for key, value := range artifact.GetMetadata().GetFields() {
			metadata[key] = value.AsInterface()
		}
	}
	if artifact.GetCustomPath() != "" {
		metadata[runtimeArtifactCustomPathMetadataKey] = artifact.GetCustomPath()
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func withArtifactLocalPath(root string, fn func() (string, error)) (string, error) {
	if current, ok := os.LookupEnv("ARTIFACT_LOCAL_PATH"); ok && current == root {
		return fn()
	}
	artifactLocalPathEnvMu.Lock()
	defer artifactLocalPathEnvMu.Unlock()
	previous, hadPrevious := os.LookupEnv("ARTIFACT_LOCAL_PATH")
	if root != "" {
		if err := os.Setenv("ARTIFACT_LOCAL_PATH", root); err != nil {
			return "", err
		}
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("ARTIFACT_LOCAL_PATH", previous)
		} else {
			_ = os.Unsetenv("ARTIFACT_LOCAL_PATH")
		}
	}()
	return fn()
}

func isTerminalTaskState(state model.TaskStatus) bool {
	switch apiv2beta1.PipelineTaskDetail_TaskState(state) {
	case apiv2beta1.PipelineTaskDetail_SUCCEEDED,
		apiv2beta1.PipelineTaskDetail_FAILED,
		apiv2beta1.PipelineTaskDetail_SKIPPED,
		apiv2beta1.PipelineTaskDetail_CACHED:
		return true
	default:
		return false
	}
}
