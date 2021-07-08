package cacheutils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/v2/third_party/kfp_api"
	"github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
	// The endpoint uses Kubernetes service DNS name with namespace:
	//https://kubernetes.io/docs/concepts/services-networking/service/#dns
	defaultKfpApiEndpoint = "ml-pipeline.kubeflow:8887"
)

func GenerateFingerPrint(cacheKey *pipeline_spec.CacheKey) (string, error) {
	b, err := protojson.Marshal(cacheKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key: %w", err)
	}
	hash := sha256.New()
	hash.Write(b)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)
	return executionHashKey, nil

}

func GenerateCacheKey(
	inputs *pipeline_spec.ExecutorInput_Inputs,
	outputs *pipeline_spec.ExecutorInput_Outputs,
	outputParametersTypeMap map[string]string,
	cmdArgs []string, image string) (*pipeline_spec.CacheKey, error) {

	cacheKey := pipeline_spec.CacheKey{
		InputArtifactNames:   make(map[string]*pipeline_spec.ArtifactNameList),
		InputParameters:      make(map[string]*pipeline_spec.Value),
		OutputArtifactsSpec:  make(map[string]*pipeline_spec.RuntimeArtifact),
		OutputParametersSpec: make(map[string]string),
	}

	for inputArtifactName, inputArtifactList := range inputs.GetArtifacts() {
		inputArtifactNameList := pipeline_spec.ArtifactNameList{ArtifactNames: make([]string, 0)}
		for _, artifact := range inputArtifactList.Artifacts {
			inputArtifactNameList.ArtifactNames = append(inputArtifactNameList.ArtifactNames, artifact.GetName())
		}
		cacheKey.InputArtifactNames[inputArtifactName] = &inputArtifactNameList
	}

	for inputParameterName, inputParameterValue := range inputs.GetParameters() {
		cacheKey.InputParameters[inputParameterName] = &pipeline_spec.Value{
			Value: inputParameterValue.Value,
		}
	}

	for outputArtifactName, outputArtifactList := range outputs.GetArtifacts() {
		if len(outputArtifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := outputArtifactList.Artifacts[0]
		outputArtifactWithUriWiped := pipeline_spec.RuntimeArtifact{
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

	cacheKey.ContainerSpec = &pipeline_spec.ContainerSpec{
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
func NewClient() (*Client, error) {
	cacheEndPoint := cacheDefaultEndpoint()
	glog.Infof("Connecting to cache endpoint %s", cacheEndPoint)
	conn, err := grpc.Dial(cacheEndPoint, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)), grpc.WithInsecure())
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

func (c *Client) GetExecutionCache(fingerPrint, pipelineName string) (string, error) {
	fingerPrintPredicate := &api.Predicate{
		Op:    api.Predicate_EQUALS,
		Key:   "fingerprint",
		Value: &api.Predicate_StringValue{StringValue: fingerPrint},
	}
	pipelineNamePredicate := &api.Predicate{
		Op:    api.Predicate_EQUALS,
		Key:   "pipelineName",
		Value: &api.Predicate_StringValue{StringValue: fmt.Sprintf("pipeline/%s", pipelineName)},
	}
	filter := api.Filter{Predicates: []*api.Predicate{fingerPrintPredicate, pipelineNamePredicate}}

	taskFilterJson, err := protojson.Marshal(&filter)
	if err != nil {
		return "", fmt.Errorf("failed to convert filter into JSON: %w", err)
	}
	listTasksReuqest := &api.ListTasksRequest{Filter: string(taskFilterJson), SortBy: "created_at desc", PageSize: 1}
	listTasksResponse, err := c.svc.ListTasks(context.Background(), listTasksReuqest)
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
	_, err := c.svc.CreateTask(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func GetOutputParamsFromCachedExecution(cachedExecution *ml_metadata.Execution) (map[string]string, error) {
	mlmdOutputParameters := make(map[string]string)
	for customPropName, customPropValue := range cachedExecution.CustomProperties {
		if strings.HasPrefix(customPropName, "output:") {
			slice := strings.Split(customPropName, ":")
			if len(slice) != 2 {
				return nil, fmt.Errorf("failed to parse output parameter from MLMD execution custom property %v", customPropName)
			}
			outputParamName := slice[1]
			var outputParamValue string
			switch t := customPropValue.Value.(type) {
			case *ml_metadata.Value_StringValue:
				outputParamValue = customPropValue.GetStringValue()
			case *ml_metadata.Value_DoubleValue:
				outputParamValue = strconv.FormatFloat(customPropValue.GetDoubleValue(), 'f', -1, 64)
			case *ml_metadata.Value_IntValue:
				outputParamValue = strconv.FormatInt(customPropValue.GetIntValue(), 10)
			default:
				return nil, fmt.Errorf("unknown PipelineSpec Value type %T", t)
			}
			mlmdOutputParameters[outputParamName] = outputParamValue
		}
	}
	return mlmdOutputParameters, nil
}
