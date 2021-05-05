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

package cmd

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
	mlmdPb "github.com/kubeflow/pipelines/third_party/ml-metadata/go_client/ml_metadata/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	executionParamPrefixOutputProperty = "output:"
)

type publishArgs struct {
	outputsSpec         *pb.ComponentOutputsSpec
	executionId         int64
	mlmdUrl             string
	publisherType       string
	inputPathParameters string
}

type mlmdClientHelper struct {
	client mlmdPb.MetadataStoreServiceClient
	ctx    context.Context
}

// TODO(Bobgy): refactor driver to also use this client
func (c *mlmdClientHelper) getExecutionByID(id int64) (*mlmdPb.Execution, error) {
	response, err := c.client.GetExecutionsByID(c.ctx, &mlmdPb.GetExecutionsByIDRequest{
		ExecutionIds: []int64{id},
	})
	if err != nil {
		return nil, err
	}
	if len(response.Executions) != 1 {
		return nil, errors.Errorf("Unexpected number of executions returned: '%v',when fetching execution with ID %v", len(response.Executions), id)
	}
	return response.Executions[0], nil
}

func publishImp(args *publishArgs) error {
	// Initialize MLMD connection, client and context
	conn, err := grpc.Dial(mlmdUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrapf(err, "did not connect")
	}
	defer conn.Close()
	c := mlmdPb.NewMetadataStoreServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mlmdClient := mlmdClientHelper{client: c, ctx: ctx}

	execution, err := mlmdClient.getExecutionByID(args.executionId)
	if err != nil {
		return errors.Wrapf(err, "Failed to get execution.")
	}
	glog.Infof("Execution before update: %v", execution.String())
	execution.Id = &args.executionId
	if execution.CustomProperties == nil {
		execution.CustomProperties = make(map[string]*mlmdPb.Value)
	}
	for name := range args.outputsSpec.Parameters {
		content, err := ioutil.ReadFile(filepath.Join(inputPathParameters, name))
		if err != nil {
			return errors.Wrapf(err, "Failed to read content of parameter %s.", name)
		}
		// TODO(Bobgy): support other types and validate parameter type
		execution.CustomProperties[executionParamPrefixOutputProperty+name] = &mlmdPb.Value{
			Value: &mlmdPb.Value_StringValue{
				StringValue: string(content),
			},
		}
	}
	state := mlmdPb.Execution_COMPLETE
	execution.LastKnownState = &state
	_, err = c.PutExecution(ctx, &mlmdPb.PutExecutionRequest{Execution: execution})
	if err != nil {
		return errors.Wrapf(err, "Failed to put execution")
	}
	execution2, err := mlmdClient.getExecutionByID(args.executionId)
	if err != nil {
		return errors.Wrapf(err, "Failed to get execution.")
	}
	glog.Infof("Execution after update: %v", execution2.String())
	return nil
}

func unmarshalOutputsSpec(json string) (*pb.ComponentOutputsSpec, error) {
	outputsSpec := &pb.ComponentOutputsSpec{}
	if err := jsonpb.UnmarshalString(json, outputsSpec); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse component outputs spec: %v", json)
	}
	return outputsSpec, nil
}

func Publish() error {
	outputsSpec, err := unmarshalOutputsSpec(componentOutputsSpecJson)
	if err != nil {
		return err
	}
	return publishImp(&publishArgs{
		outputsSpec:         outputsSpec,
		executionId:         executionId,
		mlmdUrl:             mlmdUrl,
		publisherType:       publisherType,
		inputPathParameters: inputPathParameters,
	})
}
