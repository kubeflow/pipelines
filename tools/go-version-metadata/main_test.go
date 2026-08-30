// Copyright 2026 The Kubeflow Authors
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
	"reflect"
	"testing"
)

func TestYAMLSemantics(t *testing.T) {
	contents := `
goImage: &goImage "gol\u0061ng"
action: &action |
  actions/setup-go@v7
container: {credentials: {user: test}, image: *goImage}
steps:
  - uses: *action
  - run: |
      uses: actions/setup-go@stale
literal: image:golang
`
	metadata, err := inspect(request{Path: "workflow.yaml", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.YAMLValues["image"], []string{"golang"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image values = %q, want %q", got, want)
	}
	if got, want := metadata.YAMLValues["uses"], []string{"actions/setup-go@v7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("uses values = %q, want %q", got, want)
	}
}

func TestDockerfileSemantics(t *testing.T) {
	contents := "# escape=`\n" +
		"ARG REPOSITORY=docker.io/library/golang\n" +
		"ARG BASE=${REPOSITORY}:1.27.0\n" +
		"FROM alpine AS golang\n" +
		"FROM golang AS stage-alias\n" +
		"FROM `\n  ${BASE} AS builder\n"
	metadata, err := inspect(request{Path: "Containerfile.worker", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.DockerGoStages, []string{"${BASE}"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Go stages = %q, want %q", got, want)
	}
	if got, want := metadata.DockerRepositoryArgs, []string{"BASE"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("repository args = %q, want %q", got, want)
	}
}

func TestYAMLAliasTraversalIsLinear(t *testing.T) {
	contents := `a: &a [{image: golang}]
b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]
d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]
e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]
f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]
g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]
h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]
i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]
j: &j [*i,*i,*i,*i,*i,*i,*i,*i,*i]
root: *j
`
	metadata, err := inspect(request{Path: "workflow.yaml", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.YAMLValues["image"], []string{"golang"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image values = %q, want %q", got, want)
	}
}

func TestDockerfileDiscoversAllExternalImageSources(t *testing.T) {
	contents := "FROM alpine AS final\n" +
		"COPY --exclude=ignored --from=golang:1.27.0 /go/bin/go /usr/bin/go\n" +
		"RUN --mount=type=bind,from=golang:1.26.0,target=/go true\n"
	metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.DockerGoStages, []string{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Go stages = %q, want %q", got, want)
	}
	if got, want := metadata.DockerGoSources, []string{"golang:1.27.0", "golang:1.26.0"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Go sources = %q, want %q", got, want)
	}
}

func TestDockerfileArgExpandedStageAliasesAreNotExternal(t *testing.T) {
	contents := "ARG SOURCE=golang\n" +
		"ARG SOURCE\n" +
		"FROM alpine AS golang\n" +
		"FROM ${SOURCE} AS final\n"
	metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.DockerGoStages) != 0 || len(metadata.DockerGoSources) != 0 {
		t.Fatalf("stage alias classified as external: stages=%q sources=%q", metadata.DockerGoStages, metadata.DockerGoSources)
	}
}

func TestDockerfileValuelessArgRedeclarationPreservesDefault(t *testing.T) {
	contents := "ARG IMAGE=golang:1.27.0\n" +
		"ARG IMAGE\n" +
		"FROM ${IMAGE} AS builder\n"
	metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.DockerGoStages, []string{"${IMAGE}"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Go stages = %q, want %q", got, want)
	}
	if got, want := metadata.DockerRepositoryArgs, []string{"IMAGE"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("repository args = %q, want %q", got, want)
	}
}

func TestMalformedModuleBlocksAreRejected(t *testing.T) {
	for _, contents := range []string{
		"module example.com/test\n\ngo (\n  1.27.0\n)\n",
		"module example.com/test\n\ngo 1.27.0\n\ntoolchain (\n  go1.27.1\n)\n",
	} {
		if _, err := inspect(request{Path: "go.mod", Contents: contents}); err == nil {
			t.Fatalf("inspect accepted malformed module:\n%s", contents)
		}
	}
}
