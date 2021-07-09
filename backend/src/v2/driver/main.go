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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/backend/src/v2/common"
	"github.com/kubeflow/pipelines/backend/src/v2/common/mlmd"
	pb "github.com/kubeflow/pipelines/backend/src/v2/third_party/pipeline_spec"
	mlmdPb "github.com/kubeflow/pipelines/third_party/ml-metadata/go_client/ml_metadata/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	k8sv1 "k8s.io/api/core/v1"
)

const (
	kfpV2ExecutionType  = "pipelines.kubeflow.org/v2alpha1/task"
	kfpV2ContextTypeDag = "pipelines.kubeflow.org/v2alpha1/dag"
)

// execution custom parameters
const (
	executionParamPrefixInputProperty = "input:"
	executionParamContextName         = "contextName"
	// (context name, task name) is a unique key to identify a KFP execution.
	// task name is unique among the context DAG.
	executionParamTaskName = "taskName"
)

// context custom parameters
const (
	contextParamDagExecutionId = "dagExecutionId"
)

func main() {
	initFlags()
	validateFlagsOrFatal()

	taskSpec, err := unmarshalTaskSpec(taskSpecJson)
	if err != nil {
		glog.Fatalln(err)
	}
	glog.Infof("taskSpec: %s", taskSpec.String())
	var executorSpec *pb.PipelineDeploymentConfig_ExecutorSpec
	if executorSpecJson != "" {
		executorSpec, err = unmarshalExecutorSpec(executorSpecJson)
		if err != nil {
			glog.Fatalln(err)
		}
		glog.Infof("executorSpec: %s", executorSpec.String())
	}

	// Initialize MLMD connection, client and context
	conn, err := grpc.Dial(mlmdUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := mlmdPb.NewMetadataStoreServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	execution, err := putExecutionAndContext(ctx, c, taskSpec, driverType == driverTypeDag /** createContext*/)
	if err != nil {
		glog.Fatal(err)
	}
	if driverType == driverTypeExecutor {
		err = driveExecutor(executorSpec, taskSpec.GetOutputs(), execution, outputPathPodSpecPatch)
		if err != nil {
			glog.Fatal(err)
		}
	}
	glog.Flush()
}

func putExecutionAndContext(
	ctx context.Context,
	c mlmdPb.MetadataStoreServiceClient,
	taskSpec *pb.PipelineTaskSpec,
	createContext bool,
) (*mlmdPb.Execution, error) {
	// put execution type
	executionTypeName := kfpV2ExecutionType
	putExecutionTypeResponse, err := c.PutExecutionType(ctx,
		&mlmdPb.PutExecutionTypeRequest{
			ExecutionType: &mlmdPb.ExecutionType{
				Name: &executionTypeName,
			},
		})
	if err != nil {
		return nil, err
	}
	executionTypeId := putExecutionTypeResponse.GetTypeId()
	if executionTypeId == 0 {
		return nil, errors.Errorf("Execution type ID is 0, %v", putExecutionTypeResponse)
	}

	// put context type
	contextTypeName := kfpV2ContextTypeDag
	putContextTypeResponse, err := c.PutContextType(ctx,
		&mlmdPb.PutContextTypeRequest{
			ContextType: &mlmdPb.ContextType{
				Name: &contextTypeName,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	contextTypeId := putContextTypeResponse.GetTypeId()
	if contextTypeId == 0 {
		return nil, errors.Errorf("Context type ID is 0, %v", putContextTypeResponse)
	}

	// put execution
	contextName := ""
	if createContext {
		contextName = executionName
	}
	var parentContext *mlmdPb.Context
	if parentContextName != "" {
		// get parent context
		contextTypeName := kfpV2ContextTypeDag
		parentContextResponse, err := c.GetContextByTypeAndName(ctx, &mlmdPb.GetContextByTypeAndNameRequest{
			TypeName:    &contextTypeName,
			ContextName: &parentContextName,
		})
		if err != nil {
			return nil, err
		}
		parentContext = parentContextResponse.GetContext()
		if parentContext == nil {
			return nil, errors.Errorf("parentContext is nil. response: %s", parentContextResponse.String())
		}
	}
	execution, err := initExecution(ctx, c, executionName, executionTypeId, taskSpec, contextName, parentContext, outputPathParameters)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to init execution")
	}
	glog.Infof("initExecution: %s\n", execution.String())
	var contexts []*mlmdPb.Context
	if parentContext != nil {
		contexts = []*mlmdPb.Context{
			parentContext,
		}
	}
	putExecutionResponse, err := c.PutExecution(
		ctx,
		&mlmdPb.PutExecutionRequest{
			Execution: execution,
			Contexts:  contexts,
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to put execution")
	}
	executionId := putExecutionResponse.GetExecutionId()
	if executionId == 0 {
		return nil, errors.Errorf("Failed to put execution, execution ID is 0: %v", putExecutionResponse.String())
	}
	glog.Infof("PutExecutionResponse: %s\n", putExecutionResponse.String())
	execution.Id = &executionId

	err = os.MkdirAll(filepath.Dir(outputPathExecutionId), os.ModePerm)
	if err != nil {
		glog.Fatal(err)
	}
	// TODO(Bobgy): figure out an explanation of 0644
	err = ioutil.WriteFile(outputPathExecutionId, []byte(fmt.Sprint(executionId)), 0644)
	if err != nil {
		glog.Fatal(err)
	}

	if createContext {
		// put context
		putContextResponse, err := c.PutContexts(ctx, &mlmdPb.PutContextsRequest{
			Contexts: []*mlmdPb.Context{
				{
					Name:   &contextName,
					TypeId: &contextTypeId,
					CustomProperties: map[string]*mlmdPb.Value{
						contextParamDagExecutionId: {
							Value: &mlmdPb.Value_IntValue{
								IntValue: executionId,
							},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to put context")
		}
		contextIds := putContextResponse.GetContextIds()
		if len(contextIds) != 1 {
			return nil, errors.Errorf("Failed to put context: unexpected length of context IDs in put context response: %v", contextIds)
		}
		contextId := contextIds[0]
		if contextId == 0 {
			return nil, errors.Errorf("Failed to put context: context ID in response is 0.")
		}
		glog.Infof("PutContextResponse: %s\n", putContextResponse.String())

		// Example: we can find this context like this.
		// foundContext, err := c.GetContextByTypeAndName(ctx, &mlmdPb.GetContextByTypeAndNameRequest{
		// 	TypeName:    &contextTypeName,
		// 	ContextName: &contextName,
		// })
		// glog.Infof("%v", foundContext.String())

		err = os.MkdirAll(filepath.Dir(outputPathContextName), os.ModePerm)
		if err != nil {
			glog.Fatal(err)
		}
		err = ioutil.WriteFile(outputPathContextName, []byte(contextName), 0644)
		if err != nil {
			glog.Fatal(err)
		}
	}

	return execution, nil
}

func initExecution(
	ctx context.Context,
	c mlmdPb.MetadataStoreServiceClient,
	name string, typeId int64, taskSpec *pb.PipelineTaskSpec, contextName string,
	parentContext *mlmdPb.Context,
	outputPathParameters string,
) (*mlmdPb.Execution, error) {
	var parentExecution *mlmdPb.Execution
	executionByTaskName := make(map[string]*mlmdPb.Execution)

	// There is only one case that parent context does not exist: root DAG.
	if parentContext != nil {
		// get parent execution
		parentExecutionId := parentContext.CustomProperties[contextParamDagExecutionId].GetIntValue()
		if parentExecutionId == 0 {
			return nil, errors.Errorf("Cannot get parent execution ID from parent context: %v", parentContext.String())
		}
		parentExecutionResponse, err := c.GetExecutionsByID(ctx, &mlmdPb.GetExecutionsByIDRequest{
			ExecutionIds: []int64{parentExecutionId},
		})
		if err != nil {
			return nil, err
		}
		if len(parentExecutionResponse.GetExecutions()) != 1 {
			return nil, errors.Errorf("Unexpected number of executions in get executions by ID response: (ID=%v response: %v)", parentExecutionId, parentExecutionResponse)
		}
		parentExecution = parentExecutionResponse.GetExecutions()[0]
		if parentExecution == nil {
			return nil, errors.Errorf("Parent execution is nil: (ID=%v response: %v)", parentExecutionId, parentExecutionResponse)
		}
		glog.Infof("Parent execution: %v", parentExecution.String())

		// get all executions in the same context
		executionsResponse, err := c.GetExecutionsByContext(ctx, &mlmdPb.GetExecutionsByContextRequest{
			ContextId: parentContext.Id,
			Options:   &mlmdPb.ListOperationOptions{},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed getting executions by context: context_name=%v, context_id=%v", parentContextName, parentContext.Id)
		}
		// TODO(Bobgy): support pagination and more executions in the same context
		if executionsResponse.GetNextPageToken() != "" {
			return nil, errors.Errorf("Too many (>%v) executions in the same context, when getting executions by context: context_name=%v, context_id=%v.", len(executionsResponse.GetExecutions()), parentContextName, parentContext.Id)
		}
		executions := executionsResponse.GetExecutions()

		// Convert execution list to a map keyed by DAG task name.
		for _, execution := range executions {
			customProperties := execution.GetCustomProperties()
			if customProperties == nil {
				glog.Warningf("Execution id=%v name=%v does not have custom properties", execution.GetId(), execution.GetName())
				continue
			}
			taskNameProp := customProperties[executionParamTaskName]
			if taskNameProp == nil {
				glog.Warningf("Execution id=%v name=%v does not have %s custom property", execution.GetId(), execution.GetName(), executionParamTaskName)
				continue
			}
			switch value := taskNameProp.Value.(type) {
			case *mlmdPb.Value_StringValue:
				executionByTaskName[value.StringValue] = execution
			default:
				glog.Warningf("Unexpected execution task name type: expected string, but got unrecognized type: %T.", taskNameProp.Value)
			}
		}
	}

	// initialize execution
	var execution mlmdPb.Execution
	execution.Name = &name
	execution.TypeId = &typeId
	state := mlmdPb.Execution_RUNNING
	execution.LastKnownState = &state
	execution.CustomProperties = make(map[string]*mlmdPb.Value)
	taskName := taskSpec.GetTaskInfo().GetName()
	if taskName != "" {
		execution.CustomProperties[executionParamTaskName] = &mlmdPb.Value{
			Value: &mlmdPb.Value_StringValue{
				StringValue: taskName,
			},
		}
	}
	if contextName != "" {
		// Note, the context can be queried via GetContextByTypeAndName.
		execution.CustomProperties[executionParamContextName] =
			&mlmdPb.Value{Value: &mlmdPb.Value_StringValue{StringValue: contextName}}
	}
	// resolve parameters
	if outputPathParameters != "" {
		// When driving a task, we need to expose parameters as argo parameters.
		err := os.MkdirAll(outputPathParameters, os.ModePerm)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to make all parent dirs for -%s=%v", argumentOutputPathParameters, outputPathParameters)
		}
	}
	for name, parameter := range taskSpec.GetInputs().GetParameters() {
		// TODO(Bobgy): handle other parameter types

		// resolve parameter value
		var value string

		if parameter.GetRuntimeValue() != nil {
			runtimeValue := parameter.GetRuntimeValue()
			if runtimeValue.GetConstantValue() != nil {
				// TODO(Bobgy): build conversion helpers between MLMD Value and KFP Pipeline Spec Value, and use them here.
				value = runtimeValue.GetConstantValue().GetStringValue()
				execution.CustomProperties[executionParamPrefixInputProperty+name] =
					&mlmdPb.Value{Value: &mlmdPb.Value_StringValue{
						StringValue: value}}
			} else {
				return nil, errors.Errorf("Unsupported parameter type: %T", parameter.GetRuntimeValue().Value)
			}
		} else if parameter.GetTaskOutputParameter() != nil {
			spec := parameter.GetTaskOutputParameter()
			if spec.ProducerTask == "" {
				return nil, errors.Errorf("Task output parameter's producer task is empty: %v", parameter.String())
			}
			producerExecution := executionByTaskName[spec.ProducerTask]
			if producerExecution == nil {
				return nil, errors.Errorf("Cannot find producer execution for parameter '%v' in parent context with name '%v'.", parameter.String(), parentContextName)
			}
			producerKfpExecution := mlmd.NewKfpExecution(producerExecution)
			if spec.GetOutputParameterKey() == "" {
				return nil, errors.Errorf("Output parameter key is empty for parameter '%v'.", parameter.String())
			}
			value := producerKfpExecution.GetOutputParameter(spec.GetOutputParameterKey())
			// All parameters are required for KFP.
			if value == nil {
				return nil, errors.Errorf("Producer execution does not have parameter '%v'. Producer: %v", parameter.String(), producerKfpExecution.String())
			}
			execution.CustomProperties[executionParamPrefixInputProperty+name] = value
			// TODO(Bobgy): type check using component inputs spec.
		} else if parameter.GetComponentInputParameter() != "" {
			if parentExecution == nil || parentContext == nil {
				return nil, errors.Errorf("ComponentInputParameter should not be specified in root DAG.")
			}
			customProperties := parentExecution.GetCustomProperties()
			if customProperties == nil {
				return nil, errors.Errorf("ComponentInputParameter '%s' specified, but parent execution does not have any parameters", parameter.GetComponentInputParameter())
			}
			parentInputPropertyName := executionParamPrefixInputProperty + parameter.GetComponentInputParameter()
			parentInputProperty := customProperties[parentInputPropertyName]
			if parentInputProperty == nil {
				return nil, errors.Errorf("ComponentInputParameter '%s' specified, but parent execution does not have '%s' custom property", parameter.GetComponentInputParameter(), parentInputPropertyName)
			}
			value = parentInputProperty.GetStringValue()
			// TODO(Bobgy): verify custom property type matches and support other types
			execution.CustomProperties[executionParamPrefixInputProperty+name] =
				&mlmdPb.Value{Value: &mlmdPb.Value_StringValue{
					StringValue: value}}
		}

		if outputPathParameters != "" {
			// write parameter value to output path
			filePath := filepath.Join(outputPathParameters, name)
			err := ioutil.WriteFile(filePath, []byte(value), 0644)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to write parameter '%s' to file '%s'", name, filePath)
			}
		}
	}
	return &execution, nil
}

func driveExecutor(
	executorSpec *pb.PipelineDeploymentConfig_ExecutorSpec,
	outputsSpec *pb.TaskOutputsSpec,
	execution *mlmdPb.Execution,
	outputPathPodSpecPatch string,
) error {
	var podSpec k8sv1.PodSpec
	podSpec.Containers = []k8sv1.Container{
		{
			Name:  "main",
			Image: executorSpec.GetContainer().GetImage(),
		},
	}
	mainContainer := &podSpec.Containers[0]

	mainContainer.Image = fillPlaceholders(mainContainer.Image, execution, outputsSpec)
	marshaler := jsonpb.Marshaler{}
	if outputsSpec == nil {
		outputsSpec = &pb.TaskOutputsSpec{}
	}
	outputsSpecJson, err := marshaler.MarshalToString(outputsSpec)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal outputs spec")
	}
	mainContainer.Command = []string{
		common.ExecutorEntrypointPath,
		"--logtostderr",
		"--mlmd_url=metadata-grpc-service.kubeflow.svc.cluster.local:8080",
		fmt.Sprintf("--component_outputs_spec=%s", outputsSpecJson),
		fmt.Sprintf("--execution_id=%v", execution.GetId()),
		"--publisher_type=EXECUTOR",
		"--input_path_parameters=" + common.ExecutorOutputPathParameters,
		"--", // append "--", so that all the following arguments will not be parsed as flags.
	}
	mainContainer.Args = append(executorSpec.GetContainer().GetCommand(), executorSpec.GetContainer().GetArgs()...)
	fillPlaceholdersForArray(&mainContainer.Args, execution, outputsSpec)

	// write pod spec patch to output path
	podSpecPatchJsonBytes, err := json.Marshal(podSpec)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal pod spec to JSON. PodSpec: %v", podSpec)
	}
	err = ioutil.WriteFile(outputPathPodSpecPatch, podSpecPatchJsonBytes, 0644)
	if err != nil {
		return errors.Wrapf(err, "Failed to pod spec patch to file '%s'", outputPathPodSpecPatch)
	}
	return nil
}

func fillPlaceholdersForArray(s *[]string, execution *mlmdPb.Execution, outputsSpec *pb.TaskOutputsSpec) {
	for index, item := range *s {
		(*s)[index] = fillPlaceholders(item, execution, outputsSpec)
	}
}

func fillPlaceholders(s string, execution *mlmdPb.Execution, outputsSpec *pb.TaskOutputsSpec) string {
	properties := &execution.CustomProperties
	if properties != nil {
		for propertyName, property := range *properties {
			if strings.HasPrefix(propertyName, executionParamPrefixInputProperty) {
				name := strings.TrimPrefix(propertyName, executionParamPrefixInputProperty)
				// TODO(Bobgy): handle other value types
				value := property.GetStringValue()
				// TODO(Bobgy): figure out a more stable way to replace placeholders
				s = strings.ReplaceAll(s, fmt.Sprintf("{{$.inputs.parameters['%s']}}", name), value)
			}
		}
	}
	parameters := outputsSpec.GetParameters()
	if parameters != nil {
		for name := range parameters {
			s = strings.ReplaceAll(
				s,
				fmt.Sprintf("{{$.outputs.parameters['%s'].output_file}}", name),
				filepath.Join(common.ExecutorOutputPathParameters, name),
			)
		}
	}
	return s
}

func unmarshalTaskSpec(json string) (*pb.PipelineTaskSpec, error) {
	taskSpec := &pb.PipelineTaskSpec{}
	if err := jsonpb.UnmarshalString(json, taskSpec); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse task spec: %v", json)
	}
	return taskSpec, nil
}

func unmarshalExecutorSpec(json string) (*pb.PipelineDeploymentConfig_ExecutorSpec, error) {
	executorSpec := &pb.PipelineDeploymentConfig_ExecutorSpec{}
	if err := jsonpb.UnmarshalString(json, executorSpec); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse executor spec: %v", json)
	}
	return executorSpec, nil
}
