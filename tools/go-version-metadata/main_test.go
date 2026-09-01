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
	"errors"
	"io"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"gopkg.in/yaml.v3"
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

func TestYAMLGoDownloadUsesExactHTTPSOrigin(t *testing.T) {
	for _, test := range []struct {
		name   string
		value  string
		wantGo bool
	}{
		{name: "embedded command", value: "run: curl https://go.dev/dl/go1.27.0.tar.gz", wantGo: true},
		{name: "legacy origin", value: "url: https://dl.google.com/go/go1.27.0.tar.gz", wantGo: true},
		{name: "HTTP scheme", value: "url: http://go.dev/dl/go1.27.0.tar.gz"},
		{name: "unrelated origin", value: "url: https://example.com/go.dev/dl/go1.27.0.tar.gz"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "workflow.yaml", Contents: test.value + "\n"})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.HasGoDownload != test.wantGo {
				t.Fatalf("HasGoDownload = %t, want %t", metadata.HasGoDownload, test.wantGo)
			}
		})
	}
}

func TestDockerClassification(t *testing.T) {
	digest := strings.Repeat("a", 64)
	tests := []struct {
		name           string
		contents       string
		classification string
		candidateKinds []string
	}{
		{
			name:           "canonical managed form",
			contents:       "FROM golang:1.27.0-alpine@sha256:" + digest + " AS builder\n",
			classification: "managed",
			candidateKinds: []string{"from"},
		},
		{
			name: "unrelated modern syntax",
			contents: "# syntax=docker/dockerfile:1.19\n" +
				"FROM alpine\nCOPY --exclude=ignored source /source\n",
			classification: "irrelevant",
		},
		{
			name:           "full-line comments are irrelevant",
			contents:       "FROM alpine:3.22\n# FROM golang:latest\nRUN echo unrelated\n",
			classification: "irrelevant",
		},
		{
			name:           "literal unsupported from",
			contents:       "FROM golang:latest AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "escaped repository word is unsupported",
			contents:       "FROM g\\olang:latest AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "double-quoted repository word is unsupported",
			contents:       "FROM go\"lang\":latest AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "single-quoted repository word is unsupported",
			contents:       "FROM go'lang':latest AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "uppercase repository is unsupported",
			contents:       "FROM GOLANG:1.27.0@sha256:" + digest + " AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "uppercase digest is unsupported",
			contents:       "FROM golang:1.27.0@SHA256:" + digest + " AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "lowercase managed keywords are unsupported",
			contents:       "from golang:1.27.0@sha256:" + digest + " as builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "double-spaced managed syntax is unsupported",
			contents:       "FROM  golang:1.27.0@sha256:" + digest + " AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "tab-separated managed syntax is unsupported",
			contents:       "FROM\tgolang:1.27.0@sha256:" + digest + "\tAS\tbuilder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "leading indentation is preserved outside canonical value",
			contents:       "  FROM golang:1.27.0-alpine@sha256:" + digest + " AS builder\n",
			classification: "managed",
			candidateKinds: []string{"from"},
		},
		{
			name:           "invalid tag flavor is unsupported",
			contents:       "FROM golang:1.27.0-foo:bar@sha256:" + digest + " AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name: "arg indirection is not evaluated",
			contents: "ARG IMAGE=golang:1.27.0\nARG IMAGE\n" +
				"FROM ${IMAGE} AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"arg-default"},
		},
		{
			name: "arg identifiers are not semantic values",
			contents: "FROM alpine\n" +
				"ARG golang\nARG golang=alpine\n",
			classification: "irrelevant",
		},
		{
			name: "environment names and label metadata are not runtime sources",
			contents: "FROM alpine\n" +
				"ENV golang=alpine\n" +
				"LABEL golang=alpine toolchain=golang:latest\n",
			classification: "irrelevant",
		},
		{
			name:           "environment values are reserved source metadata",
			contents:       "FROM alpine\nENV TOOLCHAIN=golang:latest DOWNLOAD=https://go.dev/dl/go1.27.0.tar.gz\n",
			classification: "unsupported",
			candidateKinds: []string{"env-value", "env-value"},
		},
		{
			name: "download values are recognized across typed fields",
			contents: "FROM alpine\n" +
				"ARG URL=https://go.dev/dl/go1.27.0.tar.gz\n" +
				"ENV LEGACY=https://dl.google.com/go/go1.27.0.tar.gz\n" +
				"ADD https://GO.DEV/dl/go1.27.0.tar.gz /tmp/go.tgz\n" +
				"RUN wget https://go.dev/dl/go1.27.0.tar.gz\n",
			classification: "unsupported",
			candidateKinds: []string{"arg-default", "env-value", "add-download", "download"},
		},
		{
			name: "download URLs require exact scheme host and path prefix",
			contents: "FROM alpine\n" +
				"ARG A=http://go.dev/dl/go1.27.0.tar.gz\n" +
				"ARG B=https://example.com/go.dev/dl/go1.27.0.tar.gz\n" +
				"ARG C=https://go.dev.example/dl/go1.27.0.tar.gz\n",
			classification: "irrelevant",
		},
		{
			name:           "ordinary copy operands are not runtime sources",
			contents:       "FROM alpine\nCOPY golang /tmp/golang\n",
			classification: "irrelevant",
		},
		{
			name:           "arg default is lexically normalized",
			contents:       "FROM alpine\nARG IMAGE=go\"lang\":latest\n",
			classification: "unsupported",
			candidateKinds: []string{"arg-default"},
		},
		{
			name:           "literal interpolation fallback is unsupported",
			contents:       "FROM ${IMAGE:-golang} AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "literal interpolated suffix is unsupported",
			contents:       "FROM golang${TAG} AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name: "alternate interpolation operator cannot hide a second source",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"ARG IMAGE=alpine\n" +
				"FROM ${IMAGE:+golang}:1.26 AS hidden\n",
			classification: "unsupported",
			candidateKinds: []string{"from", "from"},
		},
		{
			name:           "non-colon interpolation default is unsupported",
			contents:       "FROM ${IMAGE-golang}:1.26 AS builder\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name:           "shell-delimited executable pull is unsupported",
			contents:       "FROM alpine\nRUN docker pull golang; echo done\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "BuildKit-normalized shell words are unsupported",
			contents: "FROM alpine\n" +
				"RUN docker pull g\\olang:latest\n" +
				"RUN docker pull go\"lang\":latest\n" +
				"RUN docker pull go'lang':latest\n",
			classification: "unsupported",
			candidateKinds: []string{"literal", "literal", "literal"},
		},
		{
			name:           "unbraced parameter identifier is irrelevant",
			contents:       "FROM alpine\nRUN echo $名golang\n",
			classification: "irrelevant",
		},
		{
			name: "non-POSIX runtime shell modifiers are unsupported",
			contents: "FROM alpine\n" +
				"RUN echo ${A=alpine} ${A^^}\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "run comments are not active literals",
			contents: "FROM alpine\n" +
				"RUN echo alpine # golang:latest\n",
			classification: "irrelevant",
		},
		{
			name:           "hash inside a run word remains literal",
			contents:       "FROM alpine\nRUN echo foo#golang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name:           "assignment modifier normalizes escaped literal",
			contents:       "FROM alpine\nRUN echo ${A:=g\\olang}\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name:           "word normalization errors are invalid",
			contents:       "FROM ${A AS builder\n",
			classification: "invalid",
		},
		{
			name: "image-name substrings are irrelevant",
			contents: "FROM alpine\n" +
				"RUN echo notgolang:latest golangci:latest my-golang:latest golang/tools:latest\n",
			classification: "irrelevant",
		},
		{
			name: "external copy and run mount",
			contents: "FROM alpine\n" +
				"COPY --from=golang:1.27.0 /go /go\n" +
				"RUN --mount=type=bind,from=golang:1.26.0,target=/go true\n",
			classification: "unsupported",
			candidateKinds: []string{"copy-from", "run-mount-from"},
		},
		{
			name: "invalid interpolated run mount invalidates the document",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"COPY --from=${IMAGE:-golang} /go /go\n" +
				"RUN --mount=type=bind,from=golang${TAG},target=/go true\n",
			classification: "invalid",
		},
		{
			name: "nested and executable literal image sources are unsupported",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"ONBUILD COPY --from=golang:latest /go /go\n" +
				"ONBUILD RUN --mount=from=golang:latest,target=/go true\n" +
				"RUN docker pull golang:latest\n" +
				"RUN crane export golang:latest image.tar\n",
			classification: "unsupported",
			candidateKinds: []string{"from", "copy-from", "run-mount-from", "literal", "literal"},
		},
		{
			name: "onbuild JSON payload is decoded recursively",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"ONBUILD RUN [\"docker\",\"pull\",\"gol\\u0061ng:latest\"]\n",
			classification: "unsupported",
			candidateKinds: []string{"from", "literal"},
		},
		{
			name: "exec POSIX shell receives positional arguments",
			contents: "FROM alpine\n" +
				"RUN [\"sh\",\"-c\",\"echo ${0}lang:latest\",\"go\"]\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "exec POSIX shell ignores inactive raw script text",
			contents: "FROM alpine\n" +
				"RUN [\"sh\",\"-c\",\"echo alpine # golang:latest\",\"golang\"]\n",
			classification: "irrelevant",
		},
		{
			name: "exec POSIX shell expands positional argument vector",
			contents: "FROM alpine\n" +
				"RUN [\"sh\",\"-c\",\"echo $@\",\"zero\",\"golang:latest\"]\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "nested canonical form is never managed",
			contents: "FROM alpine\nONBUILD FROM golang:1.27.0@sha256:" + digest +
				" AS hidden\n",
			classification: "invalid",
		},
		{
			name: "decoded JSON and heredoc run literals are unsupported",
			contents: "FROM alpine\n" +
				"RUN [\"docker\",\"pull\",\"gol\\u0061ng:latest\"]\n" +
				"ENV GO_BUILDER=golang:latest\n" +
				"RUN <<EOF\ndocker pull golang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal", "env-value", "literal"},
		},
		{
			name: "typed active instructions are inspected",
			contents: "FROM alpine\n" +
				"ADD https://go.dev/dl/go1.27.0.linux-amd64.tar.gz /tmp/go.tgz\n" +
				"CMD docker pull golang:latest\n" +
				"ENTRYPOINT [\"docker\",\"pull\",\"golang:latest\"]\n" +
				"HEALTHCHECK CMD curl https://go.dev/dl/go1.27.0.linux-amd64.tar.gz\n",
			classification: "unsupported",
			candidateKinds: []string{"add-download", "literal", "literal", "download"},
		},
		{
			name: "onbuild typed payload is inspected",
			contents: "FROM alpine\n" +
				"ONBUILD ADD https://go.dev/dl/go1.27.0.linux-amd64.tar.gz /tmp/go.tgz\n",
			classification: "unsupported",
			candidateKinds: []string{"add-download"},
		},
		{
			name: "local stage aliases and numeric stages are not external images",
			contents: "FROM alpine AS golang\n" +
				"FROM alpine\n" +
				"COPY --from=golang /bin/x /bin/x\n" +
				"COPY --from=0 /bin/x /bin/y\n" +
				"RUN --mount=type=bind,from=golang,target=/src true\n",
			classification: "irrelevant",
		},
		{
			name: "onbuild sources do not resolve in defining stage namespace",
			contents: "FROM alpine AS golang\n" +
				"ONBUILD COPY --from=golang /bin/x /bin/x\n",
			classification: "unsupported",
			candidateKinds: []string{"copy-from"},
		},
		{
			name: "onbuild heredoc retains defining shell state",
			contents: "FROM alpine\n" +
				"SHELL [\"fish\",\"-c\"]\n" +
				"ONBUILD RUN <<EOF\necho golang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "onbuild heredoc retains typed executable body",
			contents: "FROM alpine\n" +
				"ONBUILD RUN <<EOF\necho golang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name:           "current stage alias cannot hide its external base",
			contents:       "FROM golang:latest AS golang\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name: "heredoc comments end per line",
			contents: "FROM alpine\nRUN <<EOF\n" +
				"# golang:latest\n" +
				"docker pull golang:latest # still active\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "heredoc comments are not active",
			contents: "FROM alpine\nRUN <<EOF\n" +
				"echo alpine # golang:latest\n" +
				"# golang:latest\nEOF\n",
			classification: "irrelevant",
		},
		{
			name:           "heredoc delimiter is not an executable value",
			contents:       "FROM alpine\nRUN cat <<golang\nhello\ngolang\n",
			classification: "irrelevant",
		},
		{
			name:           "bare executable heredoc delimiter is not a value",
			contents:       "FROM alpine\nRUN <<golang\necho alpine\ngolang\n",
			classification: "irrelevant",
		},
		{
			name:           "invalid executable heredoc command fails closed",
			contents:       "FROM alpine\nRUN ( <<EOF\nhello\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "runtime escaping is independent of Dockerfile escape",
			contents: "# escape=`\nFROM alpine\n" +
				"RUN docker pull g\\olang:latest\n" +
				"RUN echo not\\golang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "runtime backslash newline is removed",
			contents: "FROM alpine\nRUN <<EOF\n" +
				"docker pull go\\\nlang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "configured PowerShell is explicitly unsupported",
			contents: "FROM alpine\nSHELL [\"pwsh\",\"-Command\"]\n" +
				"RUN docker pull go`lang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "Windows PowerShell path is explicitly unsupported",
			contents: "FROM alpine\n" +
				"SHELL [\"C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\",\"-Command\"]\n" +
				"RUN docker pull go`lang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "unknown shell fails closed only for candidate data",
			contents: "FROM alpine\nSHELL [\"fish\",\"-c\"]\n" +
				"RUN docker pull golang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name:           "unknown shell use is explicitly unsupported",
			contents:       "FROM alpine\nSHELL [\"fish\",\"-c\"]\nRUN echo alpine\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "unknown shell heredoc fails closed",
			contents: "FROM alpine\nSHELL [\"fish\",\"-c\"]\n" +
				"RUN <<EOF\necho golang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "unknown shell escaped download fails closed",
			contents: "FROM alpine\nSHELL [\"fish\",\"-c\"]\n" +
				"RUN curl https://go\\.dev/dl/go1.27.0.linux-amd64.tar.gz\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name:           "unknown candidate instruction fails closed",
			contents:       "FROM alpine\nFUTURE docker pull go\"lang\":latest\n",
			classification: "invalid",
		},
		{
			name:           "unknown unrelated instruction is invalid",
			contents:       "FROM alpine\nFUTURE echo alpine\n",
			classification: "invalid",
		},
		{
			name:           "unknown escaped download instruction fails closed",
			contents:       "FROM alpine\nFUTURE curl https://go\\.dev/dl/go1.27.0.linux-amd64.tar.gz\n",
			classification: "invalid",
		},
		{
			name: "invalid Docker parameter expressions are invalid",
			contents: "FROM alpine\n" +
				"ARG LENGTH=${#golang}\n" +
				"RUN golang=alpine echo ${#golang}\n",
			classification: "invalid",
		},
		{
			name: "assignment name does not hide literal operand",
			contents: "FROM alpine\n" +
				"RUN golang=alpine docker pull golang:latest\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name:           "assignment value remains visible",
			contents:       "FROM alpine\nRUN IMAGE=golang:latest echo ready\n",
			classification: "unsupported",
			candidateKinds: []string{"literal"},
		},
		{
			name: "tagless active literals are unsupported",
			contents: "FROM alpine\n" +
				"RUN docker pull golang\n" +
				"RUN [\"docker\",\"pull\",\"gol\\u0061ng\"]\n" +
				"RUN <<EOF\ndocker pull golang\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal", "literal", "literal"},
		},
		{
			name:           "default escape continuation is normalized before discovery",
			contents:       "FROM go\\\nlang:1.26-alpine AS stale\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name: "alternate escape continuation is normalized before discovery",
			contents: "# escape=`\nFROM alpine\n" +
				"RUN curl https://go.`\ndev/dl/go1.26.0.linux-amd64.tar.gz\n",
			classification: "unsupported",
			candidateKinds: []string{"download"},
		},
		{
			name: "download in heredoc is unsupported",
			contents: "FROM alpine\nRUN <<EOF\n" +
				"curl https://go.dev/dl/go1.27.0.linux-amd64.tar.gz\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"download"},
		},
		{
			name:           "malformed candidate-bearing file",
			contents:       "FROM golang:latest AS builder\nRUN <<EOF\n",
			classification: "invalid",
		},
		{
			name:           "empty workdir is invalid",
			contents:       "FROM alpine\nWORKDIR\nFROM golang:1.27.0@sha256:" + digest + " AS builder\n",
			classification: "invalid",
		},
		{
			name:           "run before first stage is invalid",
			contents:       "RUN true\nFROM golang:1.27.0@sha256:" + digest + " AS builder\n",
			classification: "invalid",
		},
		{
			name:           "forbidden onbuild payload is invalid",
			contents:       "FROM golang:1.27.0@sha256:" + digest + " AS builder\nONBUILD MAINTAINER example\n",
			classification: "invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Containerfile.worker", Contents: test.contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; error=%q", metadata.DockerClassification, test.classification, metadata.DockerError)
			}
			var kinds []string
			for _, candidate := range metadata.DockerCandidates {
				kinds = append(kinds, candidate.Kind)
			}
			if !reflect.DeepEqual(kinds, test.candidateKinds) {
				t.Fatalf("candidate kinds = %q, want %q", kinds, test.candidateKinds)
			}
			if test.classification == "managed" {
				candidate := metadata.DockerCandidates[0]
				if candidate.Version != "1.27.0" || candidate.Flavor != "-alpine" ||
					candidate.Digest != "sha256:"+digest || candidate.Alias != "builder" {
					t.Fatalf("managed candidate metadata = %#v", candidate)
				}
			}
		})
	}
}

func TestDockerSourceDiscoveryCrossProduct(t *testing.T) {
	tests := []struct {
		name        string
		instruction string
		kind        string
	}{
		{name: "ARG image", instruction: "ARG VALUE=golang:latest", kind: "arg-default"},
		{name: "ARG download", instruction: "ARG VALUE=https://go.dev/dl/go1.27.0.tar.gz", kind: "arg-default"},
		{name: "ENV image", instruction: "ENV VALUE=golang:latest", kind: "env-value"},
		{name: "ENV download", instruction: "ENV VALUE=https://go.dev/dl/go1.27.0.tar.gz", kind: "env-value"},
		{name: "ADD download", instruction: "ADD https://go.dev/dl/go1.27.0.tar.gz /tmp/go.tgz", kind: "add-download"},
		{name: "RUN shell image", instruction: "RUN echo golang:latest", kind: "literal"},
		{name: "RUN shell download", instruction: "RUN echo https://go.dev/dl/go1.27.0.tar.gz", kind: "download"},
		{name: "RUN exec image", instruction: `RUN ["echo","golang:latest"]`, kind: "literal"},
		{name: "RUN exec download", instruction: `RUN ["echo","https://go.dev/dl/go1.27.0.tar.gz"]`, kind: "download"},
		{name: "CMD shell", instruction: "CMD echo golang:latest", kind: "literal"},
		{name: "ENTRYPOINT exec", instruction: `ENTRYPOINT ["echo","golang:latest"]`, kind: "literal"},
		{name: "HEALTHCHECK shell", instruction: "HEALTHCHECK CMD echo golang:latest", kind: "literal"},
	}
	for _, test := range tests {
		for _, deferred := range []bool{false, true} {
			name := test.name + "/top-level"
			instruction := test.instruction
			if deferred {
				name = test.name + "/ONBUILD"
				instruction = "ONBUILD " + instruction
			}
			t.Run(name, func(t *testing.T) {
				metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\n" + instruction + "\n"})
				if err != nil {
					t.Fatal(err)
				}
				if metadata.DockerClassification != "unsupported" {
					t.Fatalf("classification = %q, want unsupported; error=%q", metadata.DockerClassification, metadata.DockerError)
				}
				if got := metadata.DockerCandidates[0].Kind; got != test.kind {
					t.Fatalf("candidate kind = %q, want %q", got, test.kind)
				}
			})
		}
	}
}

func TestDockerValidityBoundaryAdversarial(t *testing.T) {
	digest := strings.Repeat("a", 64)
	managed := "FROM golang:1.27.0@sha256:" + digest + " AS builder\n"
	for name, contents := range map[string]string{
		"empty document":               "",
		"only global ARG":              "ARG VALUE=alpine\n",
		"malformed irrelevant ARG":     managed + "ARG VALUE=${#alpine}\n",
		"malformed irrelevant ENV":     managed + "ENV VALUE=${!alpine}\n",
		"malformed irrelevant WORKDIR": managed + "WORKDIR '${ALPINE\n",
		"interpolated RUN mount":       managed + "RUN --mount=type=bind,from=${IMAGE:-alpine},target=/src true\n",
	} {
		t.Run(name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "invalid" {
				t.Fatalf("classification = %q, want invalid; error=%q", metadata.DockerClassification, metadata.DockerError)
			}
		})
	}
}

func TestOrderedDeferredShellState(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents string
		want     []string
	}{
		{
			name: "ordered trigger SHELL transitions",
			contents: "FROM alpine\n" +
				`ONBUILD SHELL ["fish","-c"]` + "\n" +
				"ONBUILD RUN echo alpine\n" +
				`ONBUILD SHELL ["sh","-c"]` + "\n" +
				"ONBUILD RUN echo golang:latest\n",
			want: []string{"unsupported-shell", "literal"},
		},
		{
			name: "final defining shell applies to earlier trigger",
			contents: "FROM alpine\n" +
				"ONBUILD RUN echo alpine\n" +
				`SHELL ["fish","-c"]` + "\n",
			want: []string{"unsupported-shell"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: test.contents})
			if err != nil {
				t.Fatal(err)
			}
			var kinds []string
			for _, candidate := range metadata.DockerCandidates {
				kinds = append(kinds, candidate.Kind)
			}
			if !reflect.DeepEqual(kinds, test.want) {
				t.Fatalf("candidate kinds = %q, want %q; error=%q", kinds, test.want, metadata.DockerError)
			}
		})
	}
}

func TestRuntimeArithmeticIdentifiersAreNotSources(t *testing.T) {
	for _, script := range []string{
		"echo $((golang + 1))",
		"echo $((1 + golang * 2))",
		"echo $((golang = 1))",
	} {
		metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\nRUN " + script + "\n"})
		if err != nil {
			t.Fatal(err)
		}
		if metadata.DockerClassification != "irrelevant" {
			t.Errorf("script %q classification = %q, want irrelevant; candidates=%#v", script, metadata.DockerClassification, metadata.DockerCandidates)
		}
	}
}

func TestExecShellArgumentSemantics(t *testing.T) {
	for _, test := range []struct {
		name           string
		argv           string
		classification string
	}{
		{name: "$0", argv: `["sh","-c","echo ${0}lang:latest","go"]`, classification: "unsupported"},
		{name: "$1", argv: `["sh","-c","echo ${1}lang:latest","zero","go"]`, classification: "unsupported"},
		{name: "$@", argv: `["sh","-c","echo $@","zero","golang:latest"]`, classification: "unsupported"},
		{name: "unused $0", argv: `["sh","-c","echo alpine","golang:latest"]`, classification: "irrelevant"},
		{name: "$# excludes $0", argv: `["sh","-c","echo $#","golang:latest"]`, classification: "irrelevant"},
		{name: "comment text", argv: `["sh","-c","echo alpine # golang:latest","zero"]`, classification: "irrelevant"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\nRUN " + test.argv + "\n"})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; candidates=%#v", metadata.DockerClassification, test.classification, metadata.DockerCandidates)
			}
		})
	}
}

func TestDockerWordAlternativesAcrossExpansionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name          string
		instruction   string
		candidateKind string
	}{
		{
			name:          "mixed set and unset image branches",
			instruction:   "ARG VALUE=${A:+go}${B:-lang}:latest",
			candidateKind: "arg-default",
		},
		{
			name:          "quoted image fragments",
			instruction:   `ENV VALUE=go"${A:-la}"ng:latest`,
			candidateKind: "env-value",
		},
		{
			name:          "download split across three variables",
			instruction:   "ADD https://${A:+go}${B:-.dev}/dl/${C:+go}1.27.0.tar.gz /tmp/go.tgz",
			candidateKind: "add-download",
		},
		{
			name:          "FROM repository split across branches",
			instruction:   "FROM ${A:+go}${B:-lang}:latest AS builder",
			candidateKind: "from",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			contents := "FROM alpine\n" + test.instruction + "\n"
			if strings.HasPrefix(test.instruction, "FROM ") {
				contents = test.instruction + "\n"
			}
			metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "unsupported" {
				t.Fatalf("classification = %q, want unsupported; error=%q", metadata.DockerClassification, metadata.DockerError)
			}
			if got := metadata.DockerCandidates[0].Kind; got != test.candidateKind {
				t.Fatalf("candidate kind = %q, want %q", got, test.candidateKind)
			}
		})
	}
}

func TestSymbolicImageRepositoryComponent(t *testing.T) {
	for _, test := range []struct {
		name           string
		image          string
		classification string
	}{
		{
			name:           "registry port before symbolic golang component",
			image:          "registry.example:5000/ns/go${VALUE}ang:latest",
			classification: "unsupported",
		},
		{
			name:           "unknown can introduce final component delimiter",
			image:          "registry.example:5000/ns/xx${VALUE}ang:latest",
			classification: "unsupported",
		},
		{
			name:           "unknown can turn colon into registry port",
			image:          "registry.example:${VALUE}ang:latest",
			classification: "unsupported",
		},
		{
			name:           "distinct unknowns compose after registry port colon",
			image:          "registry.example:${PORT}${PREFIX}ang:latest",
			classification: "unsupported",
		},
		{
			name:           "pure variable tag remains outside bounded projection",
			image:          "alpine:${TAG}",
			classification: "irrelevant",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM " + test.image + "\n"})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; candidates=%#v", metadata.DockerClassification, test.classification, metadata.DockerCandidates)
			}
		})
	}
}

func TestEscapedDollarDoesNotActivateDockerPatternOperator(t *testing.T) {
	metadata, err := inspect(request{
		Path:     "Dockerfile",
		Contents: "FROM alpine\nARG VALUE=$${X#?}${X}\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DockerClassification != "irrelevant" {
		t.Fatalf("classification = %q, want irrelevant; candidates=%#v error=%q", metadata.DockerClassification, metadata.DockerCandidates, metadata.DockerError)
	}
}

func TestDockerWordAlternativeBounds(t *testing.T) {
	supported := "FROM alpine\nARG VALUE=${A:-x}${B:-x}${C:-x}${D:-x}${E:-x}${F:-x}\n"
	metadata, err := inspect(request{Path: "Dockerfile", Contents: supported})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DockerClassification != "irrelevant" {
		t.Fatalf("six-variable boundary classification = %q, want irrelevant; error=%q", metadata.DockerClassification, metadata.DockerError)
	}

	contents := "FROM alpine\nARG VALUE=${A:-x}${B:-x}${C:-x}${D:-x}${E:-x}${F:-x}${G:-x}\n"
	metadata, err = inspect(request{Path: "Dockerfile", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DockerClassification != "invalid" || !strings.Contains(metadata.DockerError, "more than 6 variables") {
		t.Fatalf("classification = %q, error=%q, want bounded invalid", metadata.DockerClassification, metadata.DockerError)
	}

	workBudget := newDockerDiscovery('\\')
	boundaryWord := "alpine"
	workBudget.alternativeWork = maxDockerAlternativeWork - len(boundaryWord)
	if _, err := workBudget.dockerWordAlternatives(boundaryWord); err != nil {
		t.Fatalf("exact alternative-work boundary: %v", err)
	}
	if _, err := workBudget.dockerWordAlternatives("x"); !isResourceLimit(err, "alternative-expansion work limit") {
		t.Fatalf("alternative-work boundary+1 error = %v", err)
	}
}

func TestDockerSymbolicBranchesPreserveNestedStateAndDelimiters(t *testing.T) {
	for _, instruction := range []string{
		`ARG IMAGE=${A:-g${B}lang:latest}`,
		`ENV IMAGE=foo${X}go${Y}ang:latest`,
		`FROM foo${X}go${Y}ang:latest`,
	} {
		contents := "FROM alpine\n" + instruction + "\n"
		if strings.HasPrefix(instruction, "FROM ") {
			contents = instruction + "\n"
		}
		metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
		if err != nil {
			t.Fatal(err)
		}
		if metadata.DockerClassification != "unsupported" {
			t.Fatalf("%s: classification = %q, want unsupported; error=%q", instruction, metadata.DockerClassification, metadata.DockerError)
		}
	}
}

func TestRepeatedDockerVariableCanIntroducePathDelimiter(t *testing.T) {
	for name, instruction := range map[string]string{
		"docker word": "ARG IMAGE=g${X}l${X}ng:latest",
		"runtime":     "RUN echo g${X}l${X}ng:latest",
	} {
		t.Run(name, func(t *testing.T) {
			// X=/gola yields g/golal/golang:latest, whose final repository
			// component is exactly golang.
			metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\n" + instruction + "\n"})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "unsupported" {
				t.Fatalf("classification = %q, want unsupported; error=%q", metadata.DockerClassification, metadata.DockerError)
			}
		})
	}
}

func TestLiteralControlBytesAreNotSymbolicMarkers(t *testing.T) {
	for marker := byte(1); marker <= maxDockerWordVariables; marker++ {
		contents := "FROM alpine\nARG IMAGE=go" + string([]byte{marker}) + "ang:latest\n"
		metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
		if err != nil {
			t.Fatal(err)
		}
		if metadata.DockerClassification != "irrelevant" {
			t.Fatalf("byte 0x%02x classification = %q, want irrelevant; error=%q", marker, metadata.DockerClassification, metadata.DockerError)
		}
	}
}

func TestRuntimeShellNULIsInvalidAcrossForms(t *testing.T) {
	for name, instruction := range map[string]string{
		"shell":         "RUN echo \x00\n",
		"unknown shell": `SHELL ["fish","-c"]` + "\nRUN echo \x00\n",
		"exec sh":       `RUN ["sh","-c","echo \u0000"]` + "\n",
		"heredoc":       "RUN <<EOF\necho \x00\nEOF\n",
	} {
		t.Run(name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\n" + instruction})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "invalid" || !strings.Contains(metadata.DockerError, "NUL") {
				t.Fatalf("classification = %q, error=%q, want invalid NUL rejection", metadata.DockerClassification, metadata.DockerError)
			}
		})
	}
}

func TestDockerOperatorPolicyUsesExpressionProvenance(t *testing.T) {
	for _, value := range []string{`${X}\${X#?}`, `${X}'${X#?}'`} {
		metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\nARG DOC=" + value + "\n"})
		if err != nil {
			t.Fatal(err)
		}
		if metadata.DockerClassification != "irrelevant" {
			t.Fatalf("value %q classification = %q, want irrelevant; candidates=%#v error=%q", value, metadata.DockerClassification, metadata.DockerCandidates, metadata.DockerError)
		}
	}
}

func TestIdenticalRuntimePatternExpressionsShareIdentity(t *testing.T) {
	words, wordErr := parseRuntimeShellWords(`echo https://g${X#?}.dev/dl/${X#?}o1.tar.gz`, nil)
	if wordErr != nil {
		t.Fatal(wordErr)
	}
	var symbolic *symbolicValue
	for _, word := range words {
		if word.symbolic != nil {
			symbolic = word.symbolic
		}
	}
	identities := []string{}
	if symbolic != nil {
		for _, segment := range symbolic.segments {
			if segment.variable != "" {
				identities = append(identities, segment.variable)
			}
		}
	}
	if !slices.Equal(identities, []string{"R0", "R0"}) {
		t.Fatalf("identical pattern results do not share one identity: %q", identities)
	}
}

func TestDockerWordValidationAndClassificationShareAccounting(t *testing.T) {
	first := `g\olang:latest-` + strings.Repeat("a", 15000)
	second := `go"lang":latest-` + strings.Repeat("b", 15000)
	contents := "FROM alpine\nARG FIRST=" + first + "\nENV SECOND=" + second + "\n"
	metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DockerClassification != "unsupported" {
		t.Fatalf("classification = %q, want unsupported; error=%q", metadata.DockerClassification, metadata.DockerError)
	}
	var kinds []string
	for _, candidate := range metadata.DockerCandidates {
		kinds = append(kinds, candidate.Kind)
	}
	if want := []string{"arg-default", "env-value"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("candidate kinds = %q, want %q", kinds, want)
	}
}

func TestDockerStageNamespaceAndConfigTransitions(t *testing.T) {
	for _, test := range []struct {
		name           string
		contents       string
		classification string
		candidateKinds []string
	}{
		{
			name:           "duplicate alias",
			contents:       "FROM alpine AS base\nFROM alpine AS base\n",
			classification: "irrelevant",
		},
		{
			name:           "self COPY by alias",
			contents:       "FROM alpine AS base\nCOPY --from=base /x /x\n",
			classification: "invalid",
		},
		{
			name:           "numeric RUN mount is an external image name",
			contents:       "FROM alpine AS base\nRUN --mount=type=bind,from=0,target=/src true\n",
			classification: "irrelevant",
		},
		{
			name: "prior namespace and inherited config",
			contents: "FROM alpine AS base\n" +
				`SHELL ["fish","-c"]` + "\n" +
				"FROM base AS final\nRUN echo alpine\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell"},
		},
		{
			name: "consumed ONBUILD resolves in child namespace",
			contents: "FROM alpine AS golang\n" +
				"FROM alpine AS parent\n" +
				"ONBUILD COPY --from=golang /x /x\n" +
				"FROM parent AS child\n",
			classification: "irrelevant",
		},
		{
			name: "ordered ONBUILD shell persists in child",
			contents: "FROM alpine AS parent\n" +
				`ONBUILD SHELL ["fish","-c"]` + "\n" +
				"ONBUILD RUN echo alpine\n" +
				"FROM parent AS child\n" +
				"RUN echo alpine\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell", "unsupported-shell"},
		},
		{
			name: "expanded local parent joins inherited state",
			contents: "FROM alpine AS base\n" +
				`SHELL ["fish","-c"]` + "\n" +
				"ONBUILD RUN echo alpine\n" +
				"FROM ${BASE:-base} AS child\n" +
				"RUN echo alpine\n",
			classification: "unsupported",
			candidateKinds: []string{"unsupported-shell", "unsupported-shell"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: test.contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; error=%q", metadata.DockerClassification, test.classification, metadata.DockerError)
			}
			var kinds []string
			for _, candidate := range metadata.DockerCandidates {
				kinds = append(kinds, candidate.Kind)
			}
			if !reflect.DeepEqual(kinds, test.candidateKinds) {
				t.Fatalf("candidate kinds = %q, want %q", kinds, test.candidateKinds)
			}
		})
	}
}

func TestRuntimeShellContextDomains(t *testing.T) {
	for _, test := range []struct {
		name           string
		script         string
		classification string
	}{
		{name: "arithmetic identifier", script: `echo $((golang + 1))`, classification: "irrelevant"},
		{name: "command substitution body", script: `echo $(printf golang:latest)`, classification: "unsupported"},
		{name: "command substitution nested in arithmetic", script: `echo $(( $(printf 0 golang:latest) + 1 ))`, classification: "unsupported"},
		{name: "status is numeric", script: `echo ${?}golang:latest`, classification: "irrelevant"},
		{name: "pid is numeric", script: `echo ${$}golang:latest`, classification: "irrelevant"},
		{name: "parameter count is numeric", script: `echo ${#}golang:latest`, classification: "irrelevant"},
		{name: "unbraced status is numeric", script: `echo go$?lang`, classification: "irrelevant"},
		{name: "unbraced count is numeric", script: `echo go$#lang`, classification: "irrelevant"},
		{name: "unbraced pid is numeric", script: `echo go$$lang`, classification: "irrelevant"},
		{name: "shell name is not empty", script: `echo go$0lang`, classification: "irrelevant"},
		{name: "last background pid may be unset", script: `echo go$!lang`, classification: "unsupported"},
		{name: "ordinary variables vary independently", script: `echo ${A:+g}${B:-olang}:latest`, classification: "unsupported"},
		{name: "option flags are nonempty", script: `echo go${-:+l}ang:latest`, classification: "unsupported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: "FROM alpine\nRUN " + test.script + "\n"})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; candidates=%#v", metadata.DockerClassification, test.classification, metadata.DockerCandidates)
			}
		})
	}
}

func TestRuntimeExpansionAccountingPrecedesDeduplication(t *testing.T) {
	script := "echo " + strings.TrimSpace(strings.Repeat("$X ", 1000))
	accounted := 0
	_, err := parseRuntimeShellWordsWithBudget(script, nil, func(count int) error {
		accounted += count
		if accounted > 1024 {
			return resourceLimitf("test expansion budget exceeded")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "test expansion budget exceeded") {
		t.Fatalf("error = %v, want pre-dedup work rejection", err)
	}
}

func TestDeferredEvaluationUsesCentralAdmission(t *testing.T) {
	for _, test := range []struct {
		name       string
		configure  func(*dockerDiscovery)
		triggers   []*deferredDockerInstruction
		errorMatch string
	}{
		{
			name: "instruction limit",
			configure: func(discovery *dockerDiscovery) {
				discovery.instructions = maxDockerInstructions - 1
			},
			triggers: []*deferredDockerInstruction{
				{expression: "ARG A=alpine", line: 1},
				{expression: "ARG B=alpine", line: 2},
			},
			errorMatch: "instruction limit",
		},
		{
			name: "candidate limit",
			configure: func(discovery *dockerDiscovery) {
				discovery.candidates = maxDockerCandidates - 1
			},
			triggers: []*deferredDockerInstruction{
				{expression: "RUN echo golang:latest", line: 1},
				{expression: "RUN echo golang:latest", line: 2},
			},
			errorMatch: "candidate limit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			discovery := newDockerDiscovery('\\')
			test.configure(discovery)
			_, _, err := discovery.evaluateDeferred(test.triggers, dockerInstructionContext{shell: defaultRuntimeShell()})
			if err == nil || !strings.Contains(err.Error(), test.errorMatch) {
				t.Fatalf("error = %v, want %q", err, test.errorMatch)
			}
		})
	}
}

func TestBuildKitStageGraphValidity(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents string
	}{
		{
			name: "forward stage cycle",
			contents: "FROM alpine AS stage0\n" +
				"COPY --from=stage2 /x /x\n" +
				"FROM alpine AS stage1\nCOPY --from=stage0 /x /x\n" +
				"FROM alpine AS stage2\nCOPY --from=stage1 /x /x\n",
		},
		{
			name:     "expanded COPY source",
			contents: "FROM alpine\nCOPY --from=${SOURCE} /x /x\n",
		},
		{
			name:     "negative COPY stage index",
			contents: "FROM alpine\nCOPY --from=-1 /x /x\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: test.contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "invalid" {
				t.Fatalf("classification = %q, want invalid; error=%q", metadata.DockerClassification, metadata.DockerError)
			}
		})
	}
}

func TestBuildKitStageAliasCaseDomains(t *testing.T) {
	for _, test := range []struct {
		name           string
		contents       string
		classification string
	}{
		{
			name: "lowercase FROM resolves normalized alias",
			contents: "FROM alpine AS Base\n" +
				`SHELL ["fish","-c"]` + "\n" +
				"FROM base\nRUN echo alpine\n",
			classification: "unsupported",
		},
		{
			name: "uppercase FROM is external",
			contents: "FROM alpine AS Base\n" +
				`SHELL ["fish","-c"]` + "\n" +
				"FROM Base\nRUN echo alpine\n",
			classification: "irrelevant",
		},
		{
			name: "uppercase Golang FROM is not suppressed by lowercase alias",
			contents: "FROM alpine AS golang\n" +
				"FROM GOLANG\n",
			classification: "unsupported",
		},
		{
			name: "COPY alias is case insensitive",
			contents: "FROM alpine AS Base\n" +
				"FROM alpine\nCOPY --from=base /x /x\n",
			classification: "irrelevant",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := inspect(request{Path: "Dockerfile", Contents: test.contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != test.classification {
				t.Fatalf("classification = %q, want %q; candidates=%#v error=%q", metadata.DockerClassification, test.classification, metadata.DockerCandidates, metadata.DockerError)
			}
		})
	}
}

func TestDockerGoTokenBoundaries(t *testing.T) {
	for _, value := range []string{
		"golang;",
		"golang|",
		"(golang)",
		`["golang"]`,
		"registry.example/library/golang",
		"${IMAGE:+golang}:1.26",
		"${IMAGE-golang}:1.26",
	} {
		if !containsDockerGoToken(value) {
			t.Errorf("containsDockerGoToken(%q) = false, want true", value)
		}
	}
	for _, value := range []string{
		"notgolang",
		"golangci",
		"my-golang",
		"golang.foo",
		"golang_foo",
		"golang/tools",
	} {
		if containsDockerGoToken(value) {
			t.Errorf("containsDockerGoToken(%q) = true, want false", value)
		}
	}
}

func TestDockerGoTokenShellParameterNames(t *testing.T) {
	parameters := []string{
		"NAME",
		"_name2",
		"名",
		"é٢",
		"1",
		"10",
		"１２",
		"@",
		"*",
		"#",
		"?",
		"-",
		"$",
		"!",
		"0",
	}
	operators := []string{"-", ":-", "+", ":+", "?", ":?", "#", "%"}
	for _, parameter := range parameters {
		for _, operator := range operators {
			contents := "FROM ${" + parameter + operator + "golang}:1.26 AS builder\n"
			metadata, err := inspect(request{Path: "Dockerfile", Contents: contents})
			if err != nil {
				t.Fatal(err)
			}
			if metadata.DockerClassification != "unsupported" {
				t.Errorf("parameter %q with operator %q classification = %q, want unsupported", parameter, operator, metadata.DockerClassification)
			}
		}
	}
}

func TestDockerGoTokenRejectsInvalidShellParameterNames(t *testing.T) {
	for _, parameter := range []string{
		"²",
		"🙂",
		"\u0301",
		"1名",
	} {
		value := "${" + parameter + "-golang}:1.26"
		if containsDockerGoToken(value) {
			t.Errorf("containsDockerGoToken(%q) = true, want false", value)
		}
	}
}

func TestDockerGoTokenIgnoresGolangInsideParameterNames(t *testing.T) {
	for _, value := range []string{
		"${名golang}",
		"${égolang}",
		"${#golang}",
		"${!golang}",
		"$名golang",
		"$égolang",
	} {
		if containsDockerGoToken(value) {
			t.Errorf("containsDockerGoToken(%q) = true, want false", value)
		}
	}
	if value := "${名golang-golang}"; !containsDockerGoToken(value) {
		t.Errorf("containsDockerGoToken(%q) = false, want true", value)
	}
}

func TestDockerWordNormalizationMatchesBuildKit(t *testing.T) {
	lexer := shell.NewLex('\\')
	lexer.SkipUnsetEnv = true
	discovery := newDockerDiscovery('\\')
	for input, want := range map[string]string{
		`g\olang:latest`:  "golang:latest",
		`go"lang":latest`: "golang:latest",
		`go'lang':latest`: "golang:latest",
		`$名golang`:        "$名golang",
	} {
		got, err := discovery.normalizeDockerWord(input)
		if err != nil {
			t.Errorf("normalizeDockerWord(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("normalizeDockerWord(%q) = %q, want %q", input, got, want)
		}
		buildKit, _, err := lexer.ProcessWord(input, shell.EnvsFromSlice(nil))
		if err != nil {
			t.Errorf("BuildKit ProcessWord(%q): %v", input, err)
		} else if words, runtimeErr := discovery.runtimeWords(input, nil); runtimeErr != nil {
			t.Errorf("runtimeWords(%q): %v", input, runtimeErr)
		} else if len(words) != 1 || words[0].value != buildKit {
			t.Errorf("runtimeWords(%q) = %v, BuildKit = %q", input, words, buildKit)
		}
	}
}

func TestPOSIXRuntimeDiscoveryMatchesBinSh(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		wantOutput string
		wantGo     bool
	}{
		{name: "escaped word", script: `printf '%s' g\olang`, wantOutput: "golang", wantGo: true},
		{name: "joined quotes", script: `printf '%s' go"la"'ng'`, wantOutput: "golang", wantGo: true},
		{name: "continued word", script: "printf '%s' go\\\nlang", wantOutput: "golang", wantGo: true},
		{name: "comment ends at newline", script: "printf '%s' alpine # golang\nprintf '%s' golang", wantOutput: "alpinegolang", wantGo: true},
		{name: "default expansion value", script: `printf '%s' "${A:-golang}"`, wantOutput: "golang", wantGo: true},
		{name: "parameter name", script: `printf '%s' "${#golang}"`, wantOutput: "0", wantGo: false},
		{name: "assignment name", script: `golang=alpine; printf '%s' "$golang"`, wantOutput: "alpine", wantGo: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command("/bin/sh", "-c", test.script)
			command.Env = []string{}
			output, err := command.Output()
			if err != nil {
				t.Fatalf("/bin/sh rejected conformance case: %v", err)
			}
			if got := string(output); got != test.wantOutput {
				t.Fatalf("/bin/sh output = %q, want %q", got, test.wantOutput)
			}
			words, err := parseRuntimeShellWords(test.script, nil)
			if err != nil {
				t.Fatalf("POSIX parser rejected /bin/sh input: %v", err)
			}
			literal, _ := scanRuntimeWords(words)
			if literal != test.wantGo {
				t.Fatalf("Go discovery = %t, want %t; words=%v", literal, test.wantGo, words)
			}
		})
	}
}

func TestDockerWordNormalizationBudgetsAndMemoization(t *testing.T) {
	discovery := newDockerDiscovery('\\')
	value := strings.Repeat("alpine", 100)
	if _, err := discovery.normalizeDockerWord(value); err != nil {
		t.Fatal(err)
	}
	bytesAfterFirst := discovery.normalizedBytes
	workAfterFirst := discovery.wordWorkBytes
	if _, err := discovery.normalizeDockerWord(value); err != nil {
		t.Fatal(err)
	}
	if discovery.normalizedBytes != bytesAfterFirst {
		t.Fatalf("memoized normalization increased byte budget from %d to %d", bytesAfterFirst, discovery.normalizedBytes)
	}
	if discovery.wordWorkBytes != workAfterFirst {
		t.Fatalf("memoized normalization increased work budget from %d to %d", workAfterFirst, discovery.wordWorkBytes)
	}

	deep := strings.Repeat("${A:-", maxShellASTDepth+1) + "alpine" + strings.Repeat("}", maxShellASTDepth+1)
	quoted := "'" + deep + "'"
	if _, err := discovery.normalizeDockerWord(quoted); err != nil {
		t.Fatalf("quoted parameter-like text error = %v", err)
	}
	if _, err := discovery.runtimeWords(deep, nil); !isResourceLimit(err, "shell AST depth") {
		t.Fatalf("deep runtime shell word error = %v", err)
	}
	if _, err := discovery.runtimeWords(quoted, nil); err != nil {
		t.Fatalf("quoted runtime parameter-like text error = %v", err)
	}

	wordBudget := newDockerDiscovery('\\')
	large := strings.Repeat("a", maxDockerWordInputBytes)
	if _, err := wordBudget.normalizeDockerWord(large); err != nil {
		t.Fatal(err)
	}
	if _, err := wordBudget.normalizeDockerWord(large + "b"); !isResourceLimit(err, "normalization input limit") {
		t.Fatalf("Docker word input budget error = %v", err)
	}
	if _, err := wordBudget.normalizeDockerWord(strings.Repeat("b", maxDockerWordInputBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := wordBudget.normalizeDockerWord("distinct"); !isResourceLimit(err, "word-normalization work limit") {
		t.Fatalf("Docker word work budget error = %v", err)
	}

	normalizedBudget := newDockerDiscovery('\\')
	if err := normalizedBudget.accountNormalizedBytes(maxDockerNormalizedBytes); err != nil {
		t.Fatal(err)
	}
	if err := normalizedBudget.accountNormalizedBytes(1); !isResourceLimit(err, "normalized-word limit") {
		t.Fatalf("normalized Docker word budget error = %v", err)
	}
}

func TestDockerCandidateBudget(t *testing.T) {
	contents := "FROM alpine\n" + strings.Repeat("RUN echo golang\n", maxDockerCandidates+1)
	classification, _, dockerError := classifyDockerfile(contents)
	if classification != "invalid" || !strings.Contains(dockerError, "candidate limit") {
		t.Fatalf("classification = %q, error = %q, want invalid candidate limit", classification, dockerError)
	}
}

func TestDockerInstructionBudget(t *testing.T) {
	contents := "FROM alpine\n" + strings.Repeat("RUN true\n", maxDockerInstructions+1)
	classification, _, dockerError := classifyDockerfile(contents)
	if classification != "invalid" || !strings.Contains(dockerError, "instruction limit") {
		t.Fatalf("classification = %q, error = %q, want invalid instruction limit", classification, dockerError)
	}
}

func TestDockerParameterGrammarMatchesBuildKitUnicodeNames(t *testing.T) {
	lexer := shell.NewLex('\\')
	for _, value := range []string{
		"${名-golang}",
		"${٢:-golang}",
	} {
		result, _, err := lexer.ProcessWord(value, shell.EnvsFromSlice(nil))
		if err != nil {
			t.Fatalf("BuildKit rejected %q: %v", value, err)
		}
		if result != "golang" {
			t.Fatalf("BuildKit expanded %q to %q, want golang", value, result)
		}
		if !containsDockerGoToken(value) {
			t.Errorf("containsDockerGoToken(%q) = false, want true", value)
		}
	}
}

func TestDockerUnicodeParameterNameIsUnsupported(t *testing.T) {
	metadata, err := inspect(request{
		Path:     "Dockerfile",
		Contents: "FROM ${名-golang}:1.26 AS builder\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DockerClassification != "unsupported" {
		t.Fatalf("classification = %q, want unsupported", metadata.DockerClassification)
	}
}

func TestDockerGoTokenScanHasLinearBudget(t *testing.T) {
	const inputBytes = 3 << 20
	value := strings.Repeat("notgolangci ", inputBytes/len("notgolangci "))
	found, steps := scanDockerGoToken(value)
	if found {
		t.Fatal("substring-only input was classified as a Go image token")
	}
	if steps > 2*len(value) {
		t.Fatalf("scanner used %d steps for %d bytes", steps, len(value))
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

func TestMetadataResourceLimits(t *testing.T) {
	if _, err := inspect(request{Path: "workflow.yaml", Contents: strings.Repeat("x", maxInputBytes+1)}); !isResourceLimit(err, "input limit") {
		t.Fatalf("oversized input error = %v", err)
	}
	if _, err := inspect(request{Path: "workflow.yaml", Contents: strings.Repeat("---\na: 1\n", maxYAMLDocuments+1)}); !isResourceLimit(err, "document limit") {
		t.Fatalf("document limit error = %v", err)
	}
	if _, err := inspect(request{Path: "workflow.yaml", Contents: "value: " + strings.Repeat("x", maxYAMLScalarBytes+1)}); !isResourceLimit(err, "scalar limit") {
		t.Fatalf("scalar limit error = %v", err)
	}
	if _, err := inspect(request{Path: "workflow.yaml", Contents: "values:\n" + strings.Repeat("  - x\n", maxYAMLNodes)}); !isResourceLimit(err, "node traversal limit") {
		t.Fatalf("node limit error = %v", err)
	}
	aliasEdges := "anchor: &anchor x\nvalues:\n" + strings.Repeat("  - *anchor\n", 80000)
	if _, err := inspect(request{Path: "workflow.yaml", Contents: aliasEdges}); !isResourceLimit(err, "edge traversal limit") {
		t.Fatalf("edge limit error = %v", err)
	}
	aggregateNodes := strings.Repeat("---\nvalues:\n"+strings.Repeat("  - x\n", maxYAMLNodes/2), 2)
	if _, err := inspect(request{Path: "workflow.yaml", Contents: aggregateNodes}); !isResourceLimit(err, "node traversal limit") {
		t.Fatalf("aggregate node limit error = %v", err)
	}
	parserDepth := "deep: " + strings.Repeat("[", 10001) + "0" + strings.Repeat("]", 10001)
	if _, err := inspect(request{Path: "workflow.yaml", Contents: parserDepth}); !isResourceLimit(err, "exceeded max depth") {
		t.Fatalf("YAML parser depth error = %v", err)
	}

	deep := &yaml.Node{Kind: yaml.SequenceNode}
	cursor := deep
	for range maxYAMLDepth + 1 {
		child := &yaml.Node{Kind: yaml.SequenceNode}
		cursor.Content = []*yaml.Node{child}
		cursor = child
	}
	state := yamlTraversalState{
		visited:    map[*yaml.Node]bool{},
		scalarMemo: map[*yaml.Node]string{},
	}
	if err := walkYAML(deep, map[string][]string{}, new(bool), &state, 0); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("deep YAML error = %v", err)
	}
}

func isResourceLimit(err error, text string) bool {
	var limitError *resourceLimitError
	return errors.As(err, &limitError) && strings.Contains(err.Error(), text)
}

type repeatedByteReader struct {
	remaining int
}

func (reader *repeatedByteReader) Read(buffer []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	count := len(buffer)
	if count > reader.remaining {
		count = reader.remaining
	}
	for index := range count {
		buffer[index] = ' '
	}
	reader.remaining -= count
	return count, nil
}

func TestRequestEnvelopeIsBoundedBeforeJSONDecoding(t *testing.T) {
	_, err := decodeRequest(&repeatedByteReader{remaining: maxRequestEnvelopeBytes + 1})
	if !isResourceLimit(err, "envelope limit") {
		t.Fatalf("request envelope error = %v", err)
	}
	if _, err := decodeRequest(strings.NewReader(`{"path":"x"} {"path":"y"}`)); err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("trailing request error = %v", err)
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

func FuzzInspectNeverPanics(f *testing.F) {
	f.Add(byte(0), []byte("image: golang\n"))
	f.Add(byte(1), []byte("FROM golang:1.27.0 AS builder\n"))
	f.Add(byte(2), []byte("module example.com/test\n\ngo 1.27.0\n"))
	f.Add(byte(0), []byte("a: &a [{image: golang}]\nb: [*a,*a,*a]\n"))
	f.Add(byte(0), []byte("---\nimage: golang\n---\nuses: actions/setup-go@v7\n"))
	f.Fuzz(func(t *testing.T, kind byte, data []byte) {
		if len(data) > maxInputBytes+1 {
			t.Skip()
		}
		paths := []string{"workflow.yaml", "Dockerfile", "go.mod"}
		metadata, err := inspect(request{Path: paths[int(kind)%len(paths)], Contents: string(data)})
		if err == nil && metadata.DockerClassification != "" {
			switch metadata.DockerClassification {
			case "managed", "unsupported", "irrelevant", "invalid":
			default:
				t.Fatalf("invalid classification %q", metadata.DockerClassification)
			}
		}
	})
}
