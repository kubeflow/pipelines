// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main provides a standalone CLI tool to compile KFP v2 IR YAML
// into Argo Workflow YAML. This is useful for automating pipeline
// deployment with ArgoCD or running workflows outside KFP.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"
)
var (
	inputFile    = flag.String("input", "", "Path to IR YAML/JSON file (PipelineSpec)")
	outputFile   = flag.String("output", "-", "Path to output Argo Workflow YAML (default: stdout)")
	pipelineRoot = flag.String("pipeline-root", "", "GCS or S3 root directory for pipeline outputs")
	disabled     = flag.Bool("cache-disabled", false, "Disable caching for the pipeline")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <ir-file.yaml> [-output <argo.yaml>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A standalone CLI tool to compile KFP v2 IR YAML into Argo Workflow YAML.\n")
		fmt.Fprintf(os.Stderr, "This tool uses the same backend compiler as KFP v2 engine.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -input flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read input file
	data, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Parse as protobuf (try YAML first, then JSON)
	var job pipelinespec.PipelineJob
	if err := parseInput(data, &job); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	// Compile to Argo Workflow
	workflow, err := argocompiler.Compile(&job, nil, &argocompiler.Options{
		PipelineRoot:  *pipelineRoot,
		CacheDisabled: *disabled,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling to Argo Workflow: %v\n", err)
		os.Exit(1)
	}

	// Marshal to YAML
	out, err := yaml.Marshal(workflow)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling workflow to YAML: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if *outputFile == "-" {
		os.Stdout.Write(out)
	} else {
		if err := ioutil.WriteFile(*outputFile, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Argo Workflow YAML written to: %s\n", *outputFile)
	}
}

func parseInput(data []byte, job *pipelinespec.PipelineJob) error {
	// Try YAML first
	if err := yaml.Unmarshal(data, job); err == nil {
		return nil
	}
	// Try JSON (protobuf JSON format)
	if err := protojson.Unmarshal(data, job); err == nil {
		return nil
	}
	// Try binary protobuf
	if err := proto.Unmarshal(data, job); err == nil {
		return nil
	}
	return fmt.Errorf("unable to parse input as YAML, JSON, or protobuf")
}
