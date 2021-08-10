// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package metadata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"

	"github.com/golang/glog"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
)

const (
	pipelineContextTypeName    = "system.Pipeline"
	pipelineRunContextTypeName = "system.PipelineRun"
	containerExecutionTypeName = "system.ContainerExecution"
	dagExecutionTypeName       = "system.DAGExecution"
	mlmdClientSideMaxRetries   = 3
)

var (
	// Note: All types are schemaless so we can easily evolve the types as needed.
	pipelineContextType = &pb.ContextType{
		Name: proto.String(pipelineContextTypeName),
	}

	pipelineRunContextType = &pb.ContextType{
		Name: proto.String(pipelineRunContextTypeName),
	}

	dagExecutionType = &pb.ExecutionType{
		Name: proto.String(dagExecutionTypeName),
	}

	containerExecutionType = &pb.ExecutionType{
		Name: proto.String(containerExecutionTypeName),
	}
)

// Client is an MLMD service client.
type Client struct {
	svc pb.MetadataStoreServiceClient
}

// NewClient creates a Client given the MLMD server address and port.
func NewClient(serverAddress, serverPort string) (*Client, error) {
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(mlmdClientSideMaxRetries),
		grpc_retry.WithBackoff(grpc_retry.BackoffExponentialWithJitter(300*time.Millisecond, 0.20)),
		grpc_retry.WithCodes(codes.Aborted),
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", serverAddress, serverPort),
		grpc.WithInsecure(),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("metadata.NewClient() failed: %w", err)
	}

	return &Client{
		svc: pb.NewMetadataStoreServiceClient(conn),
	}, nil
}

// Parameters is used to represent input or output parameters (which are scalar
// values) from pipeline components.
type Parameters struct {
	IntParameters    map[string]int64
	StringParameters map[string]string
	DoubleParameters map[string]float64
}

func NewParameters(params map[string]*pipelinespec.Value) (*Parameters, error) {
	result := &Parameters{
		IntParameters:    make(map[string]int64),
		StringParameters: make(map[string]string),
		DoubleParameters: make(map[string]float64),
	}
	for name, parameter := range params {
		switch t := parameter.Value.(type) {
		case *pipelinespec.Value_StringValue:
			result.StringParameters[name] = parameter.GetStringValue()
		case *pipelinespec.Value_IntValue:
			result.IntParameters[name] = parameter.GetIntValue()
		case *pipelinespec.Value_DoubleValue:
			result.DoubleParameters[name] = parameter.GetDoubleValue()
		default:
			return nil, fmt.Errorf("failed to convert from map[string]*pipelinespec.Value to metadata.Parameters: unknown parameter type for parameter name=%q: %T", name, t)
		}
	}
	return result, nil
}

// ExecutionConfig represents the input parameters and artifacts to an Execution.
type ExecutionConfig struct {
	InputParameters  *Parameters
	InputArtifactIDs map[string][]int64
	TaskName, PodName, PodUID, Namespace,
	Image, CachedMLMDExecutionID string
	// a temporary flag to special case some logic for root DAG
	IsRootDAG bool
}

// InputArtifact is a wrapper around an MLMD artifact used as component inputs.
type InputArtifact struct {
	Artifact *pb.Artifact
}

// OutputArtifact represents a schema and an MLMD artifact for output artifacts
// produced by a component.
type OutputArtifact struct {
	Name     string
	Artifact *pb.Artifact
	Schema   string
}

func (oa *OutputArtifact) Marshal() ([]byte, error) {
	b, err := protojson.Marshal(oa.Artifact)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Pipeline is a handle for the current pipeline.
type Pipeline struct {
	pipelineCtx    *pb.Context
	pipelineRunCtx *pb.Context
}

func (p *Pipeline) GetRunCtxID() int64 {
	if p == nil {
		return 0
	}
	return p.pipelineRunCtx.GetId()
}

func (p *Pipeline) GetPipelineRoot() string {
	if p == nil {
		return ""
	}
	props := p.pipelineRunCtx.GetCustomProperties()
	root, ok := props[keyPipelineRoot]
	if !ok {
		return ""
	}
	return root.GetStringValue()
}

// Execution is a handle for the current execution.
type Execution struct {
	execution *pb.Execution
	pipeline  *Pipeline
}

func (e *Execution) GetID() int64 {
	if e == nil {
		return 0
	}
	return e.execution.GetId()
}

func (e *Execution) String() string {
	if e == nil {
		return ""
	}
	return e.execution.String()
}

func (e *Execution) TaskName() string {
	if e == nil {
		return ""
	}
	return e.execution.GetCustomProperties()[keyTaskName].GetStringValue()
}

// GetPipeline returns the current pipeline represented by the specified
// pipeline name and run ID.
func (c *Client) GetPipeline(ctx context.Context, pipelineName, pipelineRunID, namespace, runResource, pipelineRoot string) (*Pipeline, error) {
	pipelineContext, err := getOrInsertContext(ctx, c.svc, pipelineName, pipelineContextType, nil)
	if err != nil {
		return nil, err
	}
	runMetadata := map[string]*pb.Value{
		keyNamespace:    stringValue(namespace),
		keyResourceName: stringValue(runResource),
		keyPipelineRoot: stringValue(pipelineRoot),
	}
	pipelineRunContext, err := getOrInsertContext(ctx, c.svc, pipelineRunID, pipelineRunContextType, runMetadata)
	if err != nil {
		return nil, err
	}

	err = c.putParentContexts(ctx, &pb.PutParentContextsRequest{
		ParentContexts: []*pb.ParentContext{{
			ChildId:  pipelineRunContext.Id,
			ParentId: pipelineContext.Id,
		}},
	})
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		pipelineCtx:    pipelineContext,
		pipelineRunCtx: pipelineRunContext,
	}, nil
}

// a Kubeflow Pipelines DAG
type DAG struct {
	Execution *Execution
	context   *pb.Context
}

// identifier info for error message purposes
func (d *DAG) Info() string {
	return fmt.Sprintf("DAG(executionID=%v, contextID=%v)", d.Execution.GetID(), d.context.GetId())
}

func (c *Client) GetDAG(ctx context.Context, executionID int64, contextID int64) (*DAG, error) {
	dagError := func(err error) error {
		return fmt.Errorf("failed to get DAG executionID=%v contextID=%v: %w", executionID, contextID, err)
	}
	executions, err := c.GetExecutions(ctx, []int64{executionID})
	if err != nil {
		return nil, dagError(err)
	}
	if len(executions) != 1 {
		return nil, dagError(fmt.Errorf("got %v executions, expect 1", len(executions)))
	}
	execution := executions[0]
	context, err := c.getContextByID(ctx, contextID)
	if err != nil {
		return nil, dagError(err)
	}
	if context == nil {
		return nil, dagError(fmt.Errorf("context not found"))
	}
	// TODO(Bobgy): verify execution type is system.DAGExecution & context type is system.PipelineRun or system.DAGExecution
	return &DAG{Execution: &Execution{execution: execution}, context: context}, nil
}

func (c *Client) putParentContexts(ctx context.Context, req *pb.PutParentContextsRequest) error {
	_, err := c.svc.PutParentContexts(ctx, req)
	if err != nil {
		if status.Convert(err).Code() == codes.AlreadyExists {
			// already exists code is expected when multiple requests are sent in parallel
		} else {
			return fmt.Errorf("Failed PutParentContexts(%v): %w", req.String(), err)
		}
	}
	return nil
}

func (c *Client) getContainerExecutionTypeID(ctx context.Context) (int64, error) {
	eType, err := c.svc.PutExecutionType(ctx, &pb.PutExecutionTypeRequest{
		ExecutionType: containerExecutionType,
	})

	if err != nil {
		return 0, err
	}

	return eType.GetTypeId(), nil
}

func stringValue(s string) *pb.Value {
	return &pb.Value{Value: &pb.Value_StringValue{StringValue: s}}
}

func intValue(i int64) *pb.Value {
	return &pb.Value{Value: &pb.Value_IntValue{IntValue: i}}
}

func doubleValue(f float64) *pb.Value {
	return &pb.Value{Value: &pb.Value_DoubleValue{DoubleValue: f}}
}

// Event path is conceptually artifact name for the execution.
// We cannot store the name as a property of artifact "a", because for example:
// 1. In first task "preprocess", there's an output artifact "processed_data".
// 2. In second task "train", there's an input artifact "dataset" passed from "preprocess"
// task's "processed_data" output.
//
// Now the same artifact is called "processed_data" in "preprocess" task, but "dataset" in
// "train" task, because artifact name is related to the context it's used.
// Therefore, we should store artifact name as a property of the artifact's events
// (connects artifact and execution) instead of the artifact's property.
func eventPath(artifactName string) *pb.Event_Path {
	return &pb.Event_Path{
		Steps: []*pb.Event_Path_Step{{
			Value: &pb.Event_Path_Step_Key{
				Key: artifactName,
			},
		}},
	}
}

func getArtifactName(eventPath *pb.Event_Path) (string, error) {
	if eventPath == nil || len(eventPath.Steps) == 0 {
		return "", fmt.Errorf("failed to get artifact name from eventPath")
	}
	return eventPath.Steps[0].GetKey(), nil
}

// PublishExecution publishes the specified execution with the given output
// parameters, artifacts and state.
func (c *Client) PublishExecution(ctx context.Context, execution *Execution, outputParameters *Parameters, outputArtifacts []*OutputArtifact, state pb.Execution_State) error {
	e := execution.execution
	e.LastKnownState = state.Enum()

	if outputParameters != nil {
		// Record output parameters.
		for n, p := range outputParameters.IntParameters {
			e.CustomProperties["output:"+n] = intValue(p)
		}
		for n, p := range outputParameters.DoubleParameters {
			e.CustomProperties["output:"+n] = doubleValue(p)
		}
		for n, p := range outputParameters.StringParameters {
			e.CustomProperties["output:"+n] = stringValue(p)
		}
	}

	contexts := []*pb.Context{}
	if execution.pipeline != nil {
		contexts = append(contexts, execution.pipeline.pipelineCtx, execution.pipeline.pipelineRunCtx)
	}
	req := &pb.PutExecutionRequest{
		Execution: e,
		Contexts:  contexts,
	}

	for _, oa := range outputArtifacts {
		aePair := &pb.PutExecutionRequest_ArtifactAndEvent{
			Event: &pb.Event{
				Type:       pb.Event_OUTPUT.Enum(),
				ArtifactId: oa.Artifact.Id,
				Path:       eventPath(oa.Name),
			},
		}
		req.ArtifactEventPairs = append(req.ArtifactEventPairs, aePair)
	}

	_, err := c.svc.PutExecution(ctx, req)
	return err
}

// metadata keys
const (
	keyDisplayName  = "display_name"
	keyTaskName     = "task_name"
	keyImage        = "image"
	keyPodName      = "pod_name"
	keyPodUID       = "pod_uid"
	keyNamespace    = "namespace"
	keyResourceName = "resource_name"
	keyPipelineRoot = "pipeline_root"
)

// CreateExecution creates a new MLMD execution under the specified Pipeline.
func (c *Client) CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error) {
	typeID, err := c.getContainerExecutionTypeID(ctx)
	if err != nil {
		return nil, err
	}

	e := &pb.Execution{
		TypeId: &typeID,
		CustomProperties: map[string]*pb.Value{
			// We should support overriding display name in the future, for now it defaults to task name.
			keyDisplayName: stringValue(config.TaskName),
			keyTaskName:    stringValue(config.TaskName),
			keyPodName:     stringValue(config.PodName),
			keyPodUID:      stringValue(config.PodUID),
			keyNamespace:   stringValue(config.Namespace),
			keyImage:       stringValue(config.Image),
		},
		LastKnownState: pb.Execution_RUNNING.Enum(),
	}
	if config.CachedMLMDExecutionID != "" {
		e.CustomProperties["cached_execution_id"] = stringValue(config.CachedMLMDExecutionID)
	}

	if config.InputParameters != nil {
		for k, v := range config.InputParameters.StringParameters {
			e.CustomProperties["input:"+k] = stringValue(v)
		}
		for k, v := range config.InputParameters.IntParameters {
			e.CustomProperties["input:"+k] = intValue(v)
		}
		for k, v := range config.InputParameters.DoubleParameters {
			e.CustomProperties["input:"+k] = doubleValue(v)
		}
	}

	req := &pb.PutExecutionRequest{
		Execution: e,
		Contexts:  []*pb.Context{pipeline.pipelineCtx},
	}
	if !config.IsRootDAG {
		// For root DAG execution, it should not be part of the pipeline run context,
		// because corresponds to the pipeline run.
		// TODO(Bobgy): how do we record relationship between pipeilne run context and pipeline run execution?
		req.Contexts = append(req.Contexts, pipeline.pipelineRunCtx)
	}

	for name, ids := range config.InputArtifactIDs {
		for _, id := range ids {
			thisId := id // thisId will be referenced below, so we need a local immutable var
			aePair := &pb.PutExecutionRequest_ArtifactAndEvent{
				Event: &pb.Event{
					ArtifactId: &thisId,
					Path:       eventPath(name),
					Type:       pb.Event_INPUT.Enum(),
				},
			}
			req.ArtifactEventPairs = append(req.ArtifactEventPairs, aePair)
		}
	}

	res, err := c.svc.PutExecution(ctx, req)
	if err != nil {
		return nil, err
	}

	getReq := &pb.GetExecutionsByIDRequest{
		ExecutionIds: []int64{res.GetExecutionId()},
	}

	getRes, err := c.svc.GetExecutionsByID(ctx, getReq)
	if err != nil {
		return nil, err
	}

	if len(getRes.Executions) != 1 {
		return nil, fmt.Errorf("Expected to get one Execution, got %d instead. Request: %v", len(getRes.Executions), getReq)
	}

	return &Execution{
		pipeline:  pipeline,
		execution: getRes.Executions[0],
	}, nil
}

// PrePublishExecution updates an existing MLMD execution with Pod info.
func (c *Client) PrePublishExecution(ctx context.Context, execution *Execution, config *ExecutionConfig) (*Execution, error) {
	e := execution.execution
	if e.CustomProperties == nil {
		e.CustomProperties = make(map[string]*pb.Value)
	}
	e.CustomProperties[keyPodName] = stringValue(config.PodName)
	e.CustomProperties[keyPodUID] = stringValue(config.PodUID)
	e.CustomProperties[keyNamespace] = stringValue(config.Namespace)
	e.LastKnownState = pb.Execution_RUNNING.Enum()

	_, err := c.svc.PutExecution(ctx, &pb.PutExecutionRequest{
		Execution: e,
	})
	if err != nil {
		return nil, err
	}
	return execution, nil
}

// GetExecutions ...
func (c *Client) GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error) {
	req := &pb.GetExecutionsByIDRequest{ExecutionIds: ids}
	res, err := c.svc.GetExecutionsByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Executions, nil
}

func (c *Client) GetExecution(ctx context.Context, id int64) (*Execution, error) {
	executions, err := c.GetExecutions(ctx, []int64{id})
	if err != nil {
		return nil, fmt.Errorf("get execution ID=%v: %w", id, err)
	}
	if len(executions) == 0 {
		return nil, fmt.Errorf("execution ID=%v not found", id)
	}
	if len(executions) > 1 {
		return nil, fmt.Errorf("got %v executions with ID=%v", len(executions), id)
	}
	return &Execution{execution: executions[0]}, nil
}

// GetExecutionsInDAG gets all executions in the DAG context, and organize them
// into a map, keyed by task name.
func (c *Client) GetExecutionsInDAG(ctx context.Context, dag *DAG) (executionsMap map[string]*Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get executions in %s: %w", dag.Info(), err)
		}
	}()
	executionsMap = make(map[string]*Execution)
	res, err := c.svc.GetExecutionsByContext(ctx, &pb.GetExecutionsByContextRequest{
		ContextId: dag.context.Id,
	})
	if err != nil {
		return nil, err
	}
	execs := res.GetExecutions()
	for _, e := range execs {
		execution := &Execution{execution: e}
		taskName := execution.TaskName()
		if taskName == "" {
			return nil, fmt.Errorf("empty task name for execution ID: %v", execution.GetID())
		}
		existing, ok := executionsMap[taskName]
		if ok {
			// TODO(Bobgy): to support retry, we need to handle multiple tasks with the same task name.
			return nil, fmt.Errorf("two tasks have the same task name %q, id1=%v id2=%v", taskName, existing.GetID(), execution.GetID())
		}
		executionsMap[taskName] = execution
	}
	return executionsMap, nil
}

// GetEventsByArtifactIDs ...
func (c *Client) GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error) {
	req := &pb.GetEventsByArtifactIDsRequest{ArtifactIds: artifactIds}
	res, err := c.svc.GetEventsByArtifactIDs(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Events, nil
}

func (c *Client) GetArtifactName(ctx context.Context, artifactId int64) (string, error) {
	mlmdEvents, err := c.GetEventsByArtifactIDs(ctx, []int64{artifactId})
	if err != nil {
		return "", fmt.Errorf("faild when getting events with artifact id %v: %w", artifactId, err)
	}
	if len(mlmdEvents) == 0 {
		glog.Infof("can't find any events with artifact id %v", artifactId)
		return "", nil
	}
	event := mlmdEvents[0]
	return getArtifactName(event.Path)
}

// GetArtifacts ...
func (c *Client) GetArtifacts(ctx context.Context, ids []int64) ([]*pb.Artifact, error) {
	req := &pb.GetArtifactsByIDRequest{ArtifactIds: ids}
	res, err := c.svc.GetArtifactsByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Artifacts, nil
}

// GetOutputArtifactsByExecutionId ...
func (c *Client) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string][]*pb.Artifact, error) {
	getEventsByExecutionIDsReq := &pb.GetEventsByExecutionIDsRequest{ExecutionIds: []int64{executionId}}
	getEventsByExecutionIDsRes, err := c.svc.GetEventsByExecutionIDs(ctx, getEventsByExecutionIDsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get events with execution id %v: %w", executionId, err)
	}
	var outputArtifactsIDs []int64
	outputArtifactNamesById := make(map[int64]string)
	for _, event := range getEventsByExecutionIDsRes.Events {
		if *event.Type == pb.Event_OUTPUT {
			outputArtifactsIDs = append(outputArtifactsIDs, event.GetArtifactId())
			artifactName, err := getArtifactName(event.Path)
			if err != nil {
				return nil, err
			}
			outputArtifactNamesById[event.GetArtifactId()] = artifactName
		}
	}
	outputArtifacts, err := c.GetArtifacts(ctx, outputArtifactsIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get output artifacts: %w", err)
	}
	outputArtifactsByName := make(map[string][]*pb.Artifact)
	for _, outputArtifact := range outputArtifacts {
		name, ok := outputArtifactNamesById[outputArtifact.GetId()]
		if !ok {
			return nil, fmt.Errorf("failed to get name of artifact with id %v", outputArtifact.GetId())
		}
		outputArtifactsByName[name] = append(outputArtifactsByName[name], outputArtifact)

	}

	return outputArtifactsByName, nil
}

// Only supports schema titles for now.
type schemaObject struct {
	Title string `yaml:"title"`
}

func SchemaToArtifactType(schema string) (*pb.ArtifactType, error) {
	so := &schemaObject{}
	if err := yaml.Unmarshal([]byte(schema), so); err != nil {
		return nil, err
	}

	// TODO: Also parse properties.
	if so.Title == "" {
		glog.Fatal("No title specified")
	}
	at := &pb.ArtifactType{Name: proto.String(so.Title)}
	return at, nil
}

// RecordArtifact ...
func (c *Client) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State) (*OutputArtifact, error) {
	artifact, err := toMLMDArtifact(runtimeArtifact)
	if err != nil {
		return nil, err
	}
	at, err := SchemaToArtifactType(schema)
	if err != nil {
		return nil, err
	}
	putTypeRes, err := c.svc.PutArtifactType(ctx, &pb.PutArtifactTypeRequest{ArtifactType: at})
	if err != nil {
		return nil, err
	}
	at.Id = putTypeRes.TypeId

	artifact.TypeId = at.Id
	artifact.State = &state
	if artifact.CustomProperties == nil {
		artifact.CustomProperties = make(map[string]*pb.Value)
	}
	if _, ok := artifact.CustomProperties["display_name"]; !ok {
		// display name default value
		artifact.CustomProperties["display_name"] = stringValue(outputName)
	}

	res, err := c.svc.PutArtifacts(ctx, &pb.PutArtifactsRequest{
		Artifacts: []*pb.Artifact{artifact},
	})
	if err != nil {
		return nil, err
	}
	if len(res.ArtifactIds) != 1 {
		return nil, errors.New("Failed to insert exactly one artifact")
	}

	getRes, err := c.svc.GetArtifactsByID(ctx, &pb.GetArtifactsByIDRequest{ArtifactIds: res.ArtifactIds})
	if err != nil {
		return nil, err
	}
	if len(getRes.Artifacts) != 1 {
		return nil, errors.New("Failed to retrieve exactly one artifact")
	}
	return &OutputArtifact{
		Artifact: getRes.Artifacts[0],
		Name:     outputName, // runtimeArtifact.Name is in fact artifact ID, we need to pass name separately
		Schema:   runtimeArtifact.GetType().GetInstanceSchema(),
	}, nil
}

func getOrInsertContext(ctx context.Context, svc pb.MetadataStoreServiceClient, name string, contextType *pb.ContextType, customProps map[string]*pb.Value) (*pb.Context, error) {
	// The most common case -- the context is already created by upstream tasks.
	// So we try to get the context first.
	getCtxRes, err := svc.GetContextByTypeAndName(ctx, &pb.GetContextByTypeAndNameRequest{TypeName: contextType.Name, ContextName: proto.String(name)})

	if err != nil {
		return nil, fmt.Errorf("Failed GetContextByTypeAndName(type=%q, name=%q)", contextType.GetName(), name)
	}
	// Bug in MLMD GetContextsByTypeAndName? It doesn't return error even when no
	// context was found.
	if getCtxRes.Context != nil {
		return getCtxRes.Context, nil
	}

	// Get the ContextType ID.
	var typeID int64
	putTypeRes, err := svc.PutContextType(ctx, &pb.PutContextTypeRequest{ContextType: contextType})
	if err == nil {
		typeID = putTypeRes.GetTypeId()
	} else {
		if status.Convert(err).Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("Failed PutContextType(type=%q): %w", contextType.GetName(), err)
		}
		// It's expected other tasks may try to create the context type at the same time.
		// Handle codes.AlreadyExists:
		getTypeRes, err := svc.GetContextType(ctx, &pb.GetContextTypeRequest{TypeName: contextType.Name})
		if err != nil {
			return nil, fmt.Errorf("Failed GetContextType(type=%q): %w", contextType.GetName(), err)
		}
		typeID = getTypeRes.ContextType.GetId()
	}

	// Next, create the Context.
	putReq := &pb.PutContextsRequest{
		Contexts: []*pb.Context{
			{
				Name:             proto.String(name),
				TypeId:           proto.Int64(typeID),
				CustomProperties: customProps,
			},
		},
	}
	_, err = svc.PutContexts(ctx, putReq)
	// It's expected other tasks may try to create the context at the same time,
	// so ignore AlreadyExists error.
	if err != nil && status.Convert(err).Code() != codes.AlreadyExists {
		return nil, fmt.Errorf("Failed PutContext(name=%q, type=%q, typeid=%v): %w", name, contextType.GetName(), typeID, err)
	}

	// Get the created context.
	getCtxRes, err = svc.GetContextByTypeAndName(ctx, &pb.GetContextByTypeAndNameRequest{TypeName: contextType.Name, ContextName: proto.String(name)})
	if err != nil {
		return nil, fmt.Errorf("Failed GetContext(name=%q, type=%q): %w", name, contextType.GetName(), err)
	}
	return getCtxRes.GetContext(), nil
}

func GenerateExecutionConfig(executorInput *pipelinespec.ExecutorInput) (*ExecutionConfig, error) {
	ecfg := &ExecutionConfig{
		InputArtifactIDs: make(map[string][]int64),
	}

	for name, artifactList := range executorInput.Inputs.Artifacts {
		for _, artifact := range artifactList.Artifacts {
			id, err := strconv.ParseInt(artifact.Name, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("unable to parse input artifact id from %q: %w", id, err)
			}
			ecfg.InputArtifactIDs[name] = append(ecfg.InputArtifactIDs[name], id)
		}
	}

	parameters, err := NewParameters(executorInput.Inputs.Parameters)
	if err != nil {
		return nil, err
	}
	ecfg.InputParameters = parameters
	return ecfg, nil
}

func (c *Client) getContextByID(ctx context.Context, id int64) (*pb.Context, error) {
	res, err := c.svc.GetContextsByID(ctx, &pb.GetContextsByIDRequest{ContextIds: []int64{id}})
	if err != nil {
		return nil, fmt.Errorf("getContext(id=%v): %w", id, err)
	}
	contexts := res.GetContexts()
	if len(contexts) > 1 {
		return nil, fmt.Errorf("getContext(id=%v): got %v contexts, expect 1", id, len(contexts))
	}
	if len(contexts) == 0 {
		return nil, fmt.Errorf("getContext(id=%v): not found", id)
	}
	if contexts[0] == nil {
		return nil, fmt.Errorf("getContext(id=%v): got nil context", id)
	}
	return contexts[0], nil
}
