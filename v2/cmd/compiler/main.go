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
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/compiler"
)

var (
	spec         = flag.String("spec", "", "path to pipeline spec file")
	launcher     = flag.String("launcher", "", "v2 launcher image")
	driver       = flag.String("driver", "", "v2 driver image")
	pipelineRoot = flag.String("pipeline_root", "", "pipeline root")
)

func main() {
	flag.Parse()
	if spec == nil || *spec == "" {
		glog.Exitf("spec must be specified")
	}
	err := compile(*spec)
	if err != nil {
		glog.Exitf("Failed to compile: %v", err)
	}
}

func compile(specPath string) error {
	job, err := load(specPath)
	if err != nil {
		return err
	}
	wf, err := compiler.Compile(job, &compiler.Options{
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

func load(path string) (*pipelinespec.PipelineJob, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	json := string(content)
	job := &pipelinespec.PipelineJob{}
	if err := jsonpb.UnmarshalString(json, job); err != nil {
		return nil, fmt.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, json)
	}
	return job, nil
}
