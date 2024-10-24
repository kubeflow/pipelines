// Copyright 2021-2023 The Kubeflow Authors
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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"

	"github.com/golang/glog"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

const (
	pipelineContextTypeName    = "system.Pipeline"
	pipelineRunContextTypeName = "system.PipelineRun"
	ImporterExecutionTypeName  = "system.ImporterExecution"
	mlmdClientSideMaxRetries   = 3
)

type ExecutionType string

const (
	ContainerExecutionTypeName ExecutionType = "system.ContainerExecution"
	DagExecutionTypeName       ExecutionType = "system.DAGExecution"
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
		Name: proto.String(string(DagExecutionTypeName)),
	}

	containerExecutionType = &pb.ExecutionType{
		Name: proto.String(string(ContainerExecutionTypeName)),
	}
	importerExecutionType = &pb.ExecutionType{
		Name: proto.String(ImporterExecutionTypeName),
	}
)

type ClientInterface interface {
	GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*Pipeline, error)
	GetDAG(ctx context.Context, executionID int64) (*DAG, error)
	PublishExecution(ctx context.Context, execution *Execution, outputParameters map[string]*structpb.Value, outputArtifacts []*OutputArtifact, state pb.Execution_State) error
	CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error)
	PrePublishExecution(ctx context.Context, execution *Execution, config *ExecutionConfig) (*Execution, error)
	GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error)
	GetExecution(ctx context.Context, id int64) (*Execution, error)
	GetPipelineFromExecution(ctx context.Context, id int64) (*Pipeline, error)
	GetExecutionsInDAG(ctx context.Context, dag *DAG, pipeline *Pipeline) (executionsMap map[string]*Execution, err error)
	GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error)
	GetArtifactName(ctx context.Context, artifactId int64) (string, error)
	GetArtifacts(ctx context.Context, maxResultSize int32, orderByAscending bool, orderByField, filterQuery, nextPageToken string) ([]*pb.Artifact, *string, error)
	GetContexts(ctx context.Context, maxResultSize int32, orderByAscending bool, orderByField, filterQuery, nextPageToken string) ([]*pb.Context, *string, error)
	GetArtifactsByID(ctx context.Context, ids []int64) ([]*pb.Artifact, error)
	GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*OutputArtifact, error)
	RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*OutputArtifact, error)
	GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error)
	FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error)
	GetContextByArtifactID(ctx context.Context, id int64) (*pb.Context, error)
}

// Client is an MLMD service client.
type Client struct {
	svc          pb.MetadataStoreServiceClient
	ctxTypeCache sync.Map
}

// NewClient creates a Client given the MLMD server address and port.
func NewClient(serverAddress, serverPort string, tlsEnabled bool) (*Client, error) {
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(mlmdClientSideMaxRetries),
		grpc_retry.WithBackoff(grpc_retry.BackoffExponentialWithJitter(300*time.Millisecond, 0.20)),
		grpc_retry.WithCodes(codes.Aborted),
	}

	creds := insecure.NewCredentials()
	if tlsEnabled {
		config := &tls.Config{
			InsecureSkipVerify: true, // This should be removed by https://issues.redhat.com/browse/RHOAIENG-13871
		}
		creds = credentials.NewTLS(config)
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", serverAddress, serverPort),
		grpc.WithTransportCredentials(creds),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("metadata.NewClient() failed: %w", err)
	}

	return &Client{
		svc:          pb.NewMetadataStoreServiceClient(conn),
		ctxTypeCache: sync.Map{},
	}, nil
}

// ExecutionConfig represents the input parameters and artifacts to an Execution.
type ExecutionConfig struct {
	TaskName         string
	Name             string // optional, MLMD execution name. When provided, this needs to be unique among all MLMD executions.
	ExecutionType    ExecutionType
	NotTriggered     bool  // optional, not triggered executions will have CANCELED state.
	ParentDagID      int64 // parent DAG execution ID. Only the root DAG does not have a parent DAG.
	InputParameters  map[string]*structpb.Value
	InputArtifactIDs map[string][]int64
	IterationIndex   *int // Index of the iteration.

	// ContainerExecution custom properties
	Image, CachedMLMDExecutionID, FingerPrint string
	PodName, PodUID, Namespace                string

	// DAGExecution custom properties
	IterationCount *int // Number of iterations for an iterator DAG.
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

func (oa *OutputArtifact) ToRuntimeArtifact() (*pipelinespec.RuntimeArtifact, error) {
	if oa == nil {
		return nil, nil
	}
	ra, err := toRuntimeArtifact(oa.Artifact)
	if err != nil {
		return nil, err
	}
	if ra.Type == nil {
		ra.Type = &pipelinespec.ArtifactTypeSchema{}
	}
	ra.Type.Kind = &pipelinespec.ArtifactTypeSchema_InstanceSchema{
		InstanceSchema: oa.Schema,
	}
	return ra, nil
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

func (p *Pipeline) GetCtxID() int64 {
	if p == nil {
		return 0
	}
	return p.pipelineCtx.GetId()
}

func (p *Pipeline) GetStoreSessionInfo() string {
	if p == nil {
		return ""
	}
	props := p.pipelineRunCtx.GetCustomProperties()
	storeSessionInfo, ok := props[keyStoreSessionInfo]
	if !ok {
		return ""
	}
	return storeSessionInfo.GetStringValue()
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

func (e *Execution) GetPipeline() *Pipeline {
	if e == nil {
		return nil
	}
	return e.pipeline
}

func (e *Execution) GetExecution() *pb.Execution {
	if e == nil {
		return nil
	}
	return e.execution
}

func (e *Execution) TaskName() string {
	if e == nil {
		return ""
	}
	return e.execution.GetCustomProperties()[keyTaskName].GetStringValue()
}

func (e *Execution) FingerPrint() string {
	if e == nil {
		return ""
	}
	return e.execution.GetCustomProperties()[keyCacheFingerPrint].GetStringValue()
}

// GenerateOutputURI appends the specified paths to the pipeline root.
// It may be configured to preserve the query part of the pipeline root
// by splitting it off and appending it back to the full URI.
func GenerateOutputURI(pipelineRoot string, paths []string, preserveQueryString bool) string {
	querySplit := strings.Split(pipelineRoot, "?")
	query := ""
	if len(querySplit) == 2 {
		pipelineRoot = querySplit[0]
		if preserveQueryString {
			query = "?" + querySplit[1]
		}
	} else if len(querySplit) > 2 {
		// this should never happen, but just in case.
		glog.Warningf("Unexpected pipeline root: %v", pipelineRoot)
	}
	// we cannot path.Join(root, taskName, artifactName), because root
	// contains scheme like gs:// and path.Join cleans up scheme to gs:/
	return fmt.Sprintf("%s/%s%s", strings.TrimRight(pipelineRoot, "/"), path.Join(paths...), query)
}

// GetPipeline returns the current pipeline represented by the specified
// pipeline name and run ID.
func (c *Client) GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*Pipeline, error) {
	pipelineContext, err := c.getOrInsertContext(ctx, pipelineName, pipelineContextType, nil)
	if err != nil {
		return nil, err
	}
	glog.Infof("Pipeline Context: %+v", pipelineContext)
	metadata := map[string]*pb.Value{
		keyNamespace:    stringValue(namespace),
		keyResourceName: stringValue(runResource),
		// pipeline root of this run
		keyPipelineRoot:     stringValue(GenerateOutputURI(pipelineRoot, []string{pipelineName, runID}, true)),
		keyStoreSessionInfo: stringValue(storeSessionInfo),
	}
	runContext, err := c.getOrInsertContext(ctx, runID, pipelineRunContextType, metadata)
	glog.Infof("Pipeline Run Context: %+v", runContext)
	if err != nil {
		return nil, err
	}

	// Detect whether such parent-child relationship exists.
	resParents, err := c.svc.GetParentContextsByContext(ctx, &pb.GetParentContextsByContextRequest{
		ContextId: runContext.Id,
	})
	if err != nil {
		return nil, err
	}
	parents := resParents.GetContexts()
	if len(parents) > 1 {
		return nil, fmt.Errorf("Current run context has more than 1 parent context: %v", parents)
	}
	if len(parents) == 1 {
		// Parent-child context alredy exists.
		if parents[0].GetId() != pipelineContext.GetId() {
			return nil, fmt.Errorf("Parent context ID %d of current run is different from expected: %d",
				parents[0].GetId(),
				pipelineContext.GetId())
		}
		return &Pipeline{
			pipelineCtx:    pipelineContext,
			pipelineRunCtx: runContext,
		}, nil
	}

	// Insert ParentContext relationship if doesn't exist.
	err = c.putParentContexts(ctx, &pb.PutParentContextsRequest{
		ParentContexts: []*pb.ParentContext{{
			ChildId:  runContext.Id,
			ParentId: pipelineContext.Id,
		}},
	})
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		pipelineCtx:    pipelineContext,
		pipelineRunCtx: runContext,
	}, nil
}

// a Kubeflow Pipelines DAG
type DAG struct {
	Execution *Execution
}

// identifier info for error message purposes
func (d *DAG) Info() string {
	return fmt.Sprintf("DAG(executionID=%v)", d.Execution.GetID())
}

func (c *Client) GetDAG(ctx context.Context, executionID int64) (*DAG, error) {
	dagError := func(err error) error {
		return fmt.Errorf("failed to get DAG executionID=%v: %w", executionID, err)
	}
	res, err := c.GetExecution(ctx, executionID)
	if err != nil {
		return nil, dagError(err)
	}
	execution := res.GetExecution()
	// TODO(Bobgy): verify execution type is system.DAGExecution
	return &DAG{Execution: &Execution{execution: execution}}, nil
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

func (c *Client) getExecutionTypeID(ctx context.Context, executionType *pb.ExecutionType) (int64, error) {
	eType, err := c.svc.PutExecutionType(ctx, &pb.PutExecutionTypeRequest{
		ExecutionType: executionType,
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
func (c *Client) PublishExecution(ctx context.Context, execution *Execution, outputParameters map[string]*structpb.Value, outputArtifacts []*OutputArtifact, state pb.Execution_State) error {
	e := execution.execution
	e.LastKnownState = state.Enum()

	if outputParameters != nil {
		// Record output parameters.
		outputs := &pb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: make(map[string]*structpb.Value),
			},
		}
		for n, p := range outputParameters {
			outputs.StructValue.Fields[n] = p
		}
		e.CustomProperties[keyOutputs] = &pb.Value{Value: outputs}
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
		aePair := &pb.PutExecutionRequest_ArtifactAndEvent{}
		if oa.Artifact.GetId() == 0 {
			glog.Infof("the id of output artifact is not set, will create new artifact when publishing execution")
			aePair = &pb.PutExecutionRequest_ArtifactAndEvent{
				Artifact: oa.Artifact,
				Event: &pb.Event{
					Type: pb.Event_OUTPUT.Enum(),
					Path: eventPath(oa.Name),
				},
			}
		} else {
			aePair = &pb.PutExecutionRequest_ArtifactAndEvent{
				Event: &pb.Event{
					Type:       pb.Event_OUTPUT.Enum(),
					Path:       eventPath(oa.Name),
					ArtifactId: oa.Artifact.Id,
				},
			}
		}
		req.ArtifactEventPairs = append(req.ArtifactEventPairs, aePair)
	}

	_, err := c.svc.PutExecution(ctx, req)
	return err
}

// metadata keys
const (
	keyDisplayName       = "display_name"
	keyTaskName          = "task_name"
	keyImage             = "image"
	keyPodName           = "pod_name"
	keyPodUID            = "pod_uid"
	keyNamespace         = "namespace"
	keyResourceName      = "resource_name"
	keyPipelineRoot      = "pipeline_root"
	keyStoreSessionInfo  = "store_session_info"
	keyCacheFingerPrint  = "cache_fingerprint"
	keyCachedExecutionID = "cached_execution_id"
	keyInputs            = "inputs"
	keyOutputs           = "outputs"
	keyParentDagID       = "parent_dag_id" // Parent DAG Execution ID.
	keyIterationIndex    = "iteration_index"
	keyIterationCount    = "iteration_count"
)

// CreateExecution creates a new MLMD execution under the specified Pipeline.
func (c *Client) CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error) {
	if config == nil {
		return nil, fmt.Errorf("metadata.CreateExecution got config == nil")
	}
	typeID, err := c.getExecutionTypeID(ctx, &pb.ExecutionType{
		Name: proto.String(string(config.ExecutionType)),
	})
	if err != nil {
		return nil, err
	}

	e := &pb.Execution{
		TypeId: &typeID,
		CustomProperties: map[string]*pb.Value{
			// We should support overriding display name in the future, for now it defaults to task name.
			keyDisplayName: stringValue(config.TaskName),
			keyTaskName:    stringValue(config.TaskName),
		},
	}
	if config.Name != "" {
		e.Name = &config.Name
	}
	e.LastKnownState = pb.Execution_RUNNING.Enum()
	if config.NotTriggered {
		// Note, in MLMD, CANCELED state means exactly as what we call
		// not triggered.
		// Reference: https://github.com/google/ml-metadata/blob/3434ebaf36db54a7e67dbb0793980a74ec0c5d50/ml_metadata/proto/metadata_store.proto#L251-L254
		e.LastKnownState = pb.Execution_CANCELED.Enum()
	}
	if config.ParentDagID != 0 {
		e.CustomProperties[keyParentDagID] = intValue(config.ParentDagID)
	}
	if config.IterationIndex != nil {
		e.CustomProperties[keyIterationIndex] = intValue(int64(*config.IterationIndex))
	}
	if config.IterationCount != nil {
		e.CustomProperties[keyIterationCount] = intValue(int64(*config.IterationCount))
	}
	if config.ExecutionType == ContainerExecutionTypeName {
		e.CustomProperties[keyPodName] = stringValue(config.PodName)
		e.CustomProperties[keyPodUID] = stringValue(config.PodUID)
		e.CustomProperties[keyNamespace] = stringValue(config.Namespace)
		e.CustomProperties[keyImage] = stringValue(config.Image)
		if config.CachedMLMDExecutionID != "" {
			e.CustomProperties[keyCachedExecutionID] = stringValue(config.CachedMLMDExecutionID)
		}
		if config.FingerPrint != "" {
			e.CustomProperties[keyCacheFingerPrint] = stringValue(config.FingerPrint)
		}
	}
	if config.InputParameters != nil {
		e.CustomProperties[keyInputs] = &pb.Value{Value: &pb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: config.InputParameters,
			},
		}}
	}

	req := &pb.PutExecutionRequest{
		Execution: e,
		Contexts:  []*pb.Context{pipeline.pipelineCtx, pipeline.pipelineRunCtx},
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
	execution := executions[0]
	pipeline, err := c.GetPipelineFromExecution(ctx, execution.GetId())
	if err != nil {
		return nil, err
	}
	return &Execution{execution: execution, pipeline: pipeline}, nil
}

func (c *Client) GetPipelineFromExecution(ctx context.Context, id int64) (*Pipeline, error) {
	pipelineCtxTypeID, err := c.getContextTypeID(ctx, pipelineContextType)
	if err != nil {
		return nil, err
	}
	runCtxTypeID, err := c.getContextTypeID(ctx, pipelineRunContextType)
	if err != nil {
		return nil, err
	}
	res, err := c.svc.GetContextsByExecution(ctx, &pb.GetContextsByExecutionRequest{
		ExecutionId: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get contexts of execution ID=%v: %w", id, err)
	}
	pipeline := &Pipeline{}
	for _, context := range res.GetContexts() {
		if context.GetTypeId() == pipelineCtxTypeID {
			if pipeline.pipelineCtx != nil {
				return nil, fmt.Errorf("multiple pipeline contexts found")
			}
			pipeline.pipelineCtx = context
		}
		if context.GetTypeId() == runCtxTypeID {
			if pipeline.pipelineRunCtx != nil {
				return nil, fmt.Errorf("multiple run contexts found")
			}
			pipeline.pipelineRunCtx = context
		}
	}
	return pipeline, nil
}

// GetExecutionsInDAG gets all executions in the DAG, and organize them
// into a map, keyed by task name.
func (c *Client) GetExecutionsInDAG(ctx context.Context, dag *DAG, pipeline *Pipeline) (executionsMap map[string]*Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get executions in %s: %w", dag.Info(), err)
		}
	}()
	executionsMap = make(map[string]*Execution)
	// Documentation on query syntax:
	// https://github.com/google/ml-metadata/blob/839c3501a195d340d2855b6ffdb2c4b0b49862c9/ml_metadata/proto/metadata_store.proto#L831
	parentDAGFilter := fmt.Sprintf("custom_properties.parent_dag_id.int_value = %v", dag.Execution.GetID())
	// Note, because MLMD does not have index on custom properties right now, we
	// take a pipeline run context to limit the number of executions the DB needs to
	// iterate through to find sub-executions.

	nextPageToken := ""
	for {
		res, err := c.svc.GetExecutionsByContext(ctx, &pb.GetExecutionsByContextRequest{
			ContextId: pipeline.pipelineRunCtx.Id,
			Options: &pb.ListOperationOptions{
				FilterQuery:   &parentDAGFilter,
				NextPageToken: &nextPageToken,
			},
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

		nextPageToken = res.GetNextPageToken()

		if nextPageToken == "" {
			break
		}
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

// GetArtifactsByID ...
func (c *Client) GetArtifactsByID(ctx context.Context, ids []int64) ([]*pb.Artifact, error) {
	req := &pb.GetArtifactsByIDRequest{ArtifactIds: ids}
	res, err := c.svc.GetArtifactsByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Artifacts, nil
}

// GetArtifacts ...
func (c *Client) GetArtifacts(ctx context.Context, maxResultSize int32, orderByAscending bool, orderByField, filterQuery, nextPageToken string) ([]*pb.Artifact, *string, error) {
	opts := buildListOpts(maxResultSize, orderByAscending, orderByField, filterQuery, nextPageToken)
	req := &pb.GetArtifactsRequest{Options: opts}
	res, err := c.svc.GetArtifacts(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return res.Artifacts, res.NextPageToken, nil
}

// GetOutputArtifactsByExecutionId ...
// TODO: Support multiple artifacts someday, probably through the v2 engine.
func (c *Client) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*OutputArtifact, error) {
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
	outputArtifacts, err := c.GetArtifactsByID(ctx, outputArtifactsIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get output artifacts: %w", err)
	}
	outputArtifactsByName := make(map[string]*OutputArtifact)
	for _, outputArtifact := range outputArtifacts {
		name, ok := outputArtifactNamesById[outputArtifact.GetId()]
		if !ok {
			return nil, fmt.Errorf("failed to get name of artifact with id %v", outputArtifact.GetId())
		}
		outputArtifactsByName[name] = &OutputArtifact{
			Name:     name,
			Artifact: outputArtifact,
			Schema:   "", // TODO(Bobgy): figure out how to get schema
		}
	}
	return outputArtifactsByName, nil
}

func (c *Client) GetInputArtifactsByExecutionID(ctx context.Context, executionID int64) (inputs map[string]*pipelinespec.ArtifactList, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("GetInputArtifactsByExecution(id=%v) failed: %w", executionID, err)
		}
	}()
	eventsReq := &pb.GetEventsByExecutionIDsRequest{ExecutionIds: []int64{executionID}}
	eventsRes, err := c.svc.GetEventsByExecutionIDs(ctx, eventsReq)
	if err != nil {
		return nil, err
	}
	var artifactIDs []int64
	nameByID := make(map[int64]string)
	for _, event := range eventsRes.Events {
		if *event.Type == pb.Event_INPUT {
			artifactIDs = append(artifactIDs, event.GetArtifactId())
			name, err := getArtifactName(event.Path)
			if err != nil {
				return nil, err
			}
			nameByID[event.GetArtifactId()] = name
		}
	}
	artifacts, err := c.GetArtifactsByID(ctx, artifactIDs)
	if err != nil {
		return nil, err
	}
	inputs = make(map[string]*pipelinespec.ArtifactList)
	for _, artifact := range artifacts {
		name, ok := nameByID[artifact.GetId()]
		if !ok {
			return nil, fmt.Errorf("failed to get name of artifact with id %v", artifact.GetId())
		}
		runtimeArtifact, err := toRuntimeArtifact(artifact)
		if err != nil {
			return nil, err
		}
		inputs[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
		}
	}
	return inputs, nil
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
func (c *Client) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*OutputArtifact, error) {
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

	// An artifact can belong to an external store specified via kfp-launcher
	// or via executor environment (e.g. IRSA)
	// This allows us to easily identify where to locate the artifact both
	// in user executor environment as well as in kfp ui
	if _, ok := artifact.CustomProperties["store_session_info"]; !ok {
		storeSessionInfoJSON, err1 := json.Marshal(bucketConfig.SessionInfo)
		if err1 != nil {
			return nil, err1
		}
		storeSessionInfoStr := string(storeSessionInfoJSON)
		artifact.CustomProperties["store_session_info"] = stringValue(storeSessionInfoStr)
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

// TODO consider batching these requests
// TODO(lingqinggan): need to create artifact types during initiation, and only allow these types.
// Currently we allow users to create any artifact type.
func (c *Client) GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("getOrInsertArtifactType(schema=%q) failed: %w", schema, err)
		}
	}()
	at, err := SchemaToArtifactType(schema)
	if err != nil {
		return 0, fmt.Errorf("converting schema to artifact type failed: %w", err)
	}
	getTypesRes, err := c.svc.GetArtifactType(ctx, &pb.GetArtifactTypeRequest{TypeName: at.Name})
	if err == nil {
		if getTypesRes.GetArtifactType() != nil {
			return getTypesRes.GetArtifactType().GetId(), nil
		}
	}
	// If artifact type is empty, create one
	putTypeRes, err := c.svc.PutArtifactType(ctx, &pb.PutArtifactTypeRequest{ArtifactType: at})
	if err != nil {
		return 0, fmt.Errorf("PutArtifactType failed: %w", err)
	}
	return putTypeRes.GetTypeId(), err
}

func (c *Client) FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("FindMatchedArtifact(artifact=%q) failed: %w", artifactToMatch, err)
		}
	}()
	uris := []string{artifactToMatch.GetUri()}
	getArtifactsByUriRes, err := c.svc.GetArtifactsByURI(ctx, &pb.GetArtifactsByURIRequest{Uris: uris})
	if err != nil {
		return nil, err
	}
	for _, candidateArtifact := range getArtifactsByUriRes.GetArtifacts() {
		matched, err := c.matchedArtifactOrNot(ctx, artifactToMatch, candidateArtifact, pipelineContextId)
		if err != nil {
			return nil, err
		}
		if matched {
			return candidateArtifact, nil
		}
	}
	return nil, nil

}

func (c *Client) matchedArtifactOrNot(ctx context.Context, target *pb.Artifact, candidate *pb.Artifact, pipelineContextId int64) (bool, error) {
	if target.GetTypeId() != candidate.GetTypeId() || target.GetState() != candidate.GetState() || target.GetUri() != candidate.GetUri() {
		return false, nil
	}
	for target_k, target_v := range target.GetCustomProperties() {
		val, ok := candidate.GetCustomProperties()[target_k]
		if !ok || !proto.Equal(target_v, val) {
			return false, nil
		}
	}
	res, err := c.svc.GetContextsByArtifact(ctx, &pb.GetContextsByArtifactRequest{ArtifactId: candidate.Id})
	if err != nil {
		return false, fmt.Errorf("failed to get contextsByArtifact with artifactID=%q: %w", candidate.GetId(), err)
	}
	for _, c := range res.GetContexts() {
		if c.GetId() == pipelineContextId {
			return true, nil
		}
	}
	return false, nil
}

// GetContexts ...
func (c *Client) GetContexts(ctx context.Context, maxResultSize int32, orderByAscending bool, orderByField, filterQuery, nextPageToken string) ([]*pb.Context, *string, error) {
	opts := buildListOpts(maxResultSize, orderByAscending, orderByField, filterQuery, nextPageToken)
	req := &pb.GetContextsRequest{Options: opts}
	res, err := c.svc.GetContexts(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return res.Contexts, res.NextPageToken, nil
}

// GetContextByArtifactID fetches the system.PipelineRun context for this Artifact ID
func (c *Client) GetContextByArtifactID(ctx context.Context, id int64) (*pb.Context, error) {
	res, err := c.svc.GetContextsByArtifact(ctx, &pb.GetContextsByArtifactRequest{ArtifactId: &id})
	if err != nil {
		return nil, fmt.Errorf("getContext(id=%v): %w", id, err)
	}
	contexts := res.GetContexts()

	if len(contexts) == 0 {
		return nil, fmt.Errorf("getContext(id=%v): not found", id)
	}
	if contexts[0] == nil {
		return nil, fmt.Errorf("getContext(id=%v): got nil context", id)
	}

	for _, artifactContext := range contexts {
		if *artifactContext.Type == "system.PipelineRun" {
			return artifactContext, nil
		}
	}
	return nil, fmt.Errorf("unable to find artifact context")
}

func (c *Client) getContextTypeID(ctx context.Context, contextType *pb.ContextType) (typeID int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("getContextTypeID(name=%q) failed: %w", contextType.GetName(), err)
		}
	}()
	cached, ok := c.ctxTypeCache.Load(contextType.GetName())
	if ok {
		typeID, ok = cached.(int64)
		if !ok {
			return 0, fmt.Errorf("bug: incorrect value type cached")
		}
		return typeID, nil
	}
	res, err := c.svc.GetContextType(ctx, &pb.GetContextTypeRequest{TypeName: contextType.Name})
	if err == nil { // no error
		c.ctxTypeCache.Store(contextType.GetName(), res.GetContextType().GetId())
		return res.GetContextType().GetId(), nil
	}
	if status.Convert(err).Code() != codes.NotFound {
		return 0, err
	}
	// only not found error is expected
	putRes, err := c.svc.PutContextType(ctx, &pb.PutContextTypeRequest{ContextType: contextType})
	if err == nil { // no error
		c.ctxTypeCache.Store(contextType.GetName(), putRes.GetTypeId())
		return putRes.GetTypeId(), nil
	}
	if status.Convert(err).Code() != codes.AlreadyExists {
		return 0, err
	}
	// It's expected other tasks may try to create the context type at the same time.
	// Handle codes.AlreadyExists:
	res, err = c.svc.GetContextType(ctx, &pb.GetContextTypeRequest{TypeName: contextType.Name})
	if err != nil {
		return 0, err
	}
	c.ctxTypeCache.Store(contextType.GetName(), res.GetContextType().GetId())
	return res.GetContextType().GetId(), nil
}

func (c *Client) getOrInsertContext(ctx context.Context, name string, contextType *pb.ContextType, customProps map[string]*pb.Value) (*pb.Context, error) {
	// The most common case -- the context is already created by upstream tasks.
	// So we try to get the context first.
	getCtxRes, err := c.svc.GetContextByTypeAndName(ctx, &pb.GetContextByTypeAndNameRequest{TypeName: contextType.Name, ContextName: proto.String(name)})

	if err != nil {
		return nil, fmt.Errorf("Failed GetContextByTypeAndName(type=%q, name=%q)", contextType.GetName(), name)
	}
	// Bug in MLMD GetContextsByTypeAndName? It doesn't return error even when no
	// context was found.
	if getCtxRes.Context != nil {
		return getCtxRes.Context, nil
	}

	// Get the ContextType ID.
	typeID, err := c.getContextTypeID(ctx, contextType)
	if err != nil {
		return nil, err
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
	_, err = c.svc.PutContexts(ctx, putReq)
	// It's expected other tasks may try to create the context at the same time,
	// so ignore AlreadyExists error.
	if err != nil && status.Convert(err).Code() != codes.AlreadyExists {
		return nil, fmt.Errorf("Failed PutContext(name=%q, type=%q, typeid=%v): %w", name, contextType.GetName(), typeID, err)
	}

	// Get the created context.
	getCtxRes, err = c.svc.GetContextByTypeAndName(ctx, &pb.GetContextByTypeAndNameRequest{TypeName: contextType.Name, ContextName: proto.String(name)})
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

	ecfg.InputParameters = executorInput.Inputs.ParameterValues
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

func buildListOpts(maxResultSize int32, orderByAscending bool, orderByField, filterQuery, nextPageToken string) *pb.ListOperationOptions {
	orderByFieldOpt := &pb.ListOperationOptions_OrderByField{
		IsAsc: &orderByAscending,
	}
	// Get artifacts with unspecified field will result is an error response from mlmd
	// so we omit it in this case.
	field := pb.ListOperationOptions_OrderByField_Field(pb.ListOperationOptions_OrderByField_Field_value[orderByField])
	unspecifiedField := pb.ListOperationOptions_OrderByField_Field_name[int32(pb.ListOperationOptions_OrderByField_FIELD_UNSPECIFIED)]
	if orderByField != "" && orderByField != unspecifiedField {
		orderByFieldOpt.Field = &field
	}

	opts := &pb.ListOperationOptions{
		MaxResultSize: &maxResultSize,
		OrderByField:  orderByFieldOpt,
		FilterQuery:   &filterQuery,
		NextPageToken: &nextPageToken,
	}
	return opts
}
