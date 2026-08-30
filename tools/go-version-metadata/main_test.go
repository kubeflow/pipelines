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
	"reflect"
	"strings"
	"testing"

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
			name: "comments heredocs and command text",
			contents: "FROM alpine\n# FROM golang:latest\n" +
				"RUN <<EOF\necho golang\nEOF\nRUN echo golang\n",
			classification: "irrelevant",
		},
		{
			name:           "literal unsupported from",
			contents:       "FROM golang:latest AS builder\n",
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
			name: "external copy and run mount",
			contents: "FROM alpine\n" +
				"COPY --from=golang:1.27.0 /go /go\n" +
				"RUN --mount=type=bind,from=golang:1.26.0,target=/go true\n",
			classification: "unsupported",
			candidateKinds: []string{"copy-from", "run-mount-from"},
		},
		{
			name: "interpolated external sources are unsupported",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"COPY --from=${IMAGE:-golang} /go /go\n" +
				"RUN --mount=type=bind,from=golang${TAG},target=/go true\n",
			classification: "unsupported",
			candidateKinds: []string{"from", "copy-from", "run-mount-from"},
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
