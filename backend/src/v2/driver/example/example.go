// Copyright 2021 Google LLC
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
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
)

const (
	outputPath = "./example"
)

func exampleTaskSpec_DAG() {
	taskSpec := &pb.PipelineTaskSpec{}
	taskSpec.TaskInfo = &pb.PipelineTaskInfo{
		Name: "hello-world-dag",
	}
	taskInputsSpec := &pb.TaskInputsSpec{}
	taskSpec.Inputs = taskInputsSpec
	value := &pb.TaskInputsSpec_InputParameterSpec{}
	// This wrapper hell might be improved when issues like https://github.com/golang/protobuf/issues/283
	// is resolved.
	// TODO(Bobgy): investigate if there are better workarounds we can use now.
	value.Kind = &pb.TaskInputsSpec_InputParameterSpec_RuntimeValue{
		RuntimeValue: &pb.ValueOrRuntimeParameter{
			Value: &pb.ValueOrRuntimeParameter_ConstantValue{
				ConstantValue: &pb.Value{Value: &pb.Value_StringValue{StringValue: "Hello, World!"}},
			}}}
	taskInputsSpec.Parameters = make(map[string]*pb.TaskInputsSpec_InputParameterSpec)
	taskInputsSpec.Parameters["text"] = value

	marshaler := &jsonpb.Marshaler{}
	jsonSpec, _ := marshaler.MarshalToString(taskSpec)
	// jsonSpec, _ := json.MarshalIndent(taskInputsSpec, "", "  ")
	err := ioutil.WriteFile(outputPath+"/task_spec_dag.json", []byte(jsonSpec), 0644)
	if err != nil {
		glog.Fatal(err)
	}
	// fmt.Printf("%s\n", jsonSpec)
}

func exampleTaskSpec_HelloWorld() {
	taskSpec := &pb.PipelineTaskSpec{}
	taskSpec.TaskInfo = &pb.PipelineTaskInfo{
		Name: "hello-world-task",
	}
	taskInputsSpec := &pb.TaskInputsSpec{}
	taskSpec.Inputs = taskInputsSpec
	value := &pb.TaskInputsSpec_InputParameterSpec{}
	// This wrapper hell might be improved when issues like https://github.com/golang/protobuf/issues/283
	// is resolved.
	// TODO(Bobgy): investigate if there are better workarounds we can use now.
	value.Kind = &pb.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
		ComponentInputParameter: "text",
	}
	taskInputsSpec.Parameters = make(map[string]*pb.TaskInputsSpec_InputParameterSpec)
	taskInputsSpec.Parameters["text"] = value

	marshaler := &jsonpb.Marshaler{}
	jsonSpec, _ := marshaler.MarshalToString(taskSpec)
	// jsonSpec, _ := json.MarshalIndent(taskInputsSpec, "", "  ")
	err := ioutil.WriteFile(outputPath+"/task_spec_hw.json", []byte(jsonSpec), 0644)
	if err != nil {
		glog.Fatal(err)
	}
	// fmt.Printf("%s\n", jsonSpec)
}

func exampleExecutorSpec_HelloWorld() {
	executorSpecJson := `
	{
		"container": {
		    "image": "python:3.7",
		    "args": ["--text", "{{$.inputs.parameters['text']}}"],
		    "command": [
				"python3",
				"-u",
				"-c",
				"def hello_world(text):\n    print(text)\n    return text\n\nimport argparse\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\n_parser.add_argument(\"--text\", dest=\"text\", type=str, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = hello_world(**_parsed_args)\n"
		    ]
		}
	}`
	var executorSpec pb.PipelineDeploymentConfig_ExecutorSpec
	// verify the json string has correct format
	err := jsonpb.UnmarshalString(executorSpecJson, &executorSpec)
	if err != nil {
		glog.Fatal(err)
	}
	marshaler := &jsonpb.Marshaler{}
	jsonSpec, err := marshaler.MarshalToString(&executorSpec)
	if err != nil {
		glog.Fatal(err)
	}
	err = ioutil.WriteFile(outputPath+"/executor_spec_hw.json", []byte(jsonSpec), 0644)
	if err != nil {
		glog.Fatal(err)
	}
}

func main() {
	exampleTaskSpec_DAG()
	exampleTaskSpec_HelloWorld()
	exampleExecutorSpec_HelloWorld()
}
