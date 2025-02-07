package cacheutils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/cachekey"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
	// The endpoint uses Kubernetes service DNS name with namespace:
	//https://kubernetes.io/docs/concepts/services-networking/service/#dns
	defaultKfpApiEndpoint = "ml-pipeline.kubeflow:8887"
)

func GenerateFingerPrint(cacheKey *cachekey.CacheKey) (string, error) {
	cacheKeyJsonBytes, err := protojson.Marshal(cacheKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key with protojson: %w", err)
	}
	// This json unmarshal and marshal is to use encoding/json formatter to format the bytes[] returned by protojson
	// Do the json formatter because of https://developers.google.com/protocol-buffers/docs/reference/go/faq#unstable-json
	var v interface{}
	if err := json.Unmarshal(cacheKeyJsonBytes, &v); err != nil {
		return "", fmt.Errorf("failed to unmarshall cache key json bytes array: %w", err)
	}
	formattedCacheKeyBytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshall cache key with golang encoding/json : %w", err)
	}
	hash := sha256.New()
	hash.Write(formattedCacheKeyBytes)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)
	return executionHashKey, nil
}

func GenerateCacheKey(
	inputs *pipelinespec.ExecutorInput_Inputs,
	outputs *pipelinespec.ExecutorInput_Outputs,
	outputParametersTypeMap map[string]string,
	cmdArgs []string, image string) (*cachekey.CacheKey, error) {

	cacheKey := cachekey.CacheKey{
		InputArtifactNames:   make(map[string]*cachekey.ArtifactNameList),
		InputParameterValues: make(map[string]*structpb.Value),
		OutputArtifactsSpec:  make(map[string]*pipelinespec.RuntimeArtifact),
		OutputParametersSpec: make(map[string]string),
	}

	for inputArtifactName, inputArtifactList := range inputs.GetArtifacts() {
		inputArtifactNameList := cachekey.ArtifactNameList{ArtifactNames: make([]string, 0)}
		for _, artifact := range inputArtifactList.Artifacts {
			inputArtifactNameList.ArtifactNames = append(inputArtifactNameList.ArtifactNames, artifact.GetName())
		}
		cacheKey.InputArtifactNames[inputArtifactName] = &inputArtifactNameList
	}

	for inputParameterName, inputParameterValue := range inputs.GetParameterValues() {
		cacheKey.InputParameterValues[inputParameterName] = inputParameterValue
	}

	for outputArtifactName, outputArtifactList := range outputs.GetArtifacts() {
		if len(outputArtifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := outputArtifactList.Artifacts[0]
		outputArtifactWithUriWiped := pipelinespec.RuntimeArtifact{
			Name:     outputArtifact.GetName(),
			Type:     outputArtifact.GetType(),
			Metadata: outputArtifact.GetMetadata(),
		}
		cacheKey.OutputArtifactsSpec[outputArtifactName] = &outputArtifactWithUriWiped
	}

	for outputParameterName, _ := range outputs.GetParameters() {
		outputParameterType, ok := outputParametersTypeMap[outputParameterName]
		if !ok {
			return nil, fmt.Errorf("unknown parameter %q found in ExecutorInput_Outputs", outputParameterName)
		}

		cacheKey.OutputParametersSpec[outputParameterName] = outputParameterType
	}

	cacheKey.ContainerSpec = &cachekey.ContainerSpec{
		Image:   image,
		CmdArgs: cmdArgs,
	}

	return &cacheKey, nil

}

// Client is an KFP service client.
type Client struct {
	svc api.TaskServiceClient
}

// NewClient creates a Client.
func NewClient(mlPipelineServiceTLSEnabled bool) (*Client, error) {
	creds := insecure.NewCredentials()
	if mlPipelineServiceTLSEnabled {
		config := &tls.Config{
			InsecureSkipVerify: false,
		}
		creds = credentials.NewTLS(config)
	}
	cacheEndPoint := cacheDefaultEndpoint()
	glog.Infof("Connecting to cache endpoint %s", cacheEndPoint)
	conn, err := grpc.Dial(
		cacheEndPoint,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("metadata.NewClient() failed: %w", err)
	}

	return &Client{
		svc: api.NewTaskServiceClient(conn),
	}, nil
}

func cacheDefaultEndpoint() string {
	// Discover ml-pipeline in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	cacheHost := os.Getenv("ML_PIPELINE_SERVICE_HOST")
	cachePort := os.Getenv("ML_PIPELINE_SERVICE_PORT_GRPC")
	if cacheHost != "" && cachePort != "" {
		// If there is a ml-pipeline Kubernetes service in the same namespace,
		// ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT env vars should
		// exist by default, so we use it as default.
		return cacheHost + ":" + cachePort
	}
	// If the env vars do not exist, use default ml-pipeline grpc endpoint `ml-pipeline.kubeflow:8887`.
	glog.Infof("Cannot detect ml-pipeline in the same namespace, default to %s as KFP endpoint.", defaultKfpApiEndpoint)
	return defaultKfpApiEndpoint
}

func (c *Client) GetExecutionCache(fingerPrint, pipelineName, namespace string) (string, error) {
	fingerPrintPredicate := &api.Predicate{
		Op:    api.Predicate_EQUALS,
		Key:   "fingerprint",
		Value: &api.Predicate_StringValue{StringValue: fingerPrint},
	}
	pipelineNamePredicate := &api.Predicate{
		Op:    api.Predicate_EQUALS,
		Key:   "pipelineName",
		Value: &api.Predicate_StringValue{StringValue: pipelineName},
	}
	namespacePredicate := &api.Predicate{
		Op:    api.Predicate_EQUALS,
		Key:   "namespace",
		Value: &api.Predicate_StringValue{StringValue: namespace},
	}
	filter := api.Filter{Predicates: []*api.Predicate{fingerPrintPredicate, pipelineNamePredicate, namespacePredicate}}

	taskFilterJson, err := protojson.Marshal(&filter)
	if err != nil {
		return "", fmt.Errorf("failed to convert filter into JSON: %w", err)
	}
	listTasksReuqest := &api.ListTasksRequest{Filter: string(taskFilterJson), SortBy: "created_at desc", PageSize: 1}
	listTasksResponse, err := c.svc.ListTasksV1(context.Background(), listTasksReuqest)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}
	tasks := listTasksResponse.Tasks
	if len(tasks) == 0 {
		return "", nil
	} else {
		return tasks[0].GetMlmdExecutionID(), nil
	}
}

func (c *Client) CreateExecutionCache(ctx context.Context, task *api.Task) error {
	req := &api.CreateTaskRequest{
		Task: task,
	}
	_, err := c.svc.CreateTaskV1(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}
