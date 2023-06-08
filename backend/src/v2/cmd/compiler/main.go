// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

var (
	// The spec flag is added to make running a pipeline with default parameters easier.
	// Backend compiler should only accept PipelineJob.
	specPath     = flag.String("spec", "", "path to pipeline spec file")
	jobPath      = flag.String("job", "", "path to pipeline job file")
	launcher     = flag.String("launcher", "", "v2 launcher image")
	driver       = flag.String("driver", "", "v2 driver image")
	pipelineRoot = flag.String("pipeline_root", "", "pipeline root")
)

func main() {
	flag.Parse()
	noSpec := specPath == nil || *specPath == ""
	noJob := jobPath == nil || *jobPath == ""
	if noSpec && noJob {
		glog.Exitf("spec or job must be specified")
	}
	if !noSpec && !noJob {
		glog.Exitf("spec and job cannot be specified at the same time")
	}
	var job *pipelinespec.PipelineJob
	var err error
	if !noSpec {
		job, err = loadSpec(*specPath)
	} else {
		// !noJob
		job, err = loadJob(*jobPath)
	}
	if err != nil {
		glog.Exitf("Failed to load: %v", err)
	}
	if err := compile(job); err != nil {
		glog.Exitf("Failed to compile: %v", err)
	}
}

func compile(job *pipelinespec.PipelineJob) error {
	wf, err := argocompiler.Compile(job, nil, &argocompiler.Options{
		DriverImage:   *driver,
		LauncherImage: *launcher,
		PipelineRoot:  *pipelineRoot,
	})
	if err != nil {
		return err
	}
	payload, err := yaml.Marshal(wf)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(payload)
	if err != nil {
		return err
	}
	return nil
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}

func loadJob(path string) (*pipelinespec.PipelineJob, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	job := &pipelinespec.PipelineJob{}
	if err := protojson.Unmarshal(bytes, job); err != nil {
		return nil, fmt.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, string(bytes))
	}
	return job, nil
}

func loadSpec(path string) (*pipelinespec.PipelineJob, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	spec := &pipelinespec.PipelineSpec{}
	specJson, err := yaml.YAMLToJSON(bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert pipeline spec from yaml to json, error: %s, spec: %v", err, string(bytes))
	}
	if err := protojson.Unmarshal(specJson, spec); err != nil {
		return nil, fmt.Errorf("Failed to parse pipeline spec, error: %s, spec: %v", err, string(specJson))
	}
	return jobFromSpec(spec)
}

func jobFromSpec(spec *pipelinespec.PipelineSpec) (*pipelinespec.PipelineJob, error) {
	specStruct, err := toStruct(spec)
	if err != nil {
		return nil, err
	}
	job := &pipelinespec.PipelineJob{}
	job.Name = spec.GetPipelineInfo().GetName()
	job.PipelineSpec = specStruct
	job.RuntimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{},
	}
	return job, nil
}

func toStruct(msg proto.Message) (*structpb.Struct, error) {
	specStr, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	res := &structpb.Struct{}
	err = protojson.Unmarshal(specStr, res)
	return res, err
}
