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
			name: "environment and label metadata are not runtime sources",
			contents: "FROM alpine\n" +
				"ENV golang=alpine TOOLCHAIN=golang:latest\n" +
				"LABEL golang=alpine toolchain=golang:latest\n",
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
			name: "runtime shell modifiers are not Docker expansion errors",
			contents: "FROM alpine\n" +
				"RUN echo ${A=alpine} ${A^^}\n",
			classification: "irrelevant",
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
			name: "interpolated external sources are unsupported",
			contents: "FROM golang:1.27.0@sha256:" + digest + " AS builder\n" +
				"COPY --from=${IMAGE:-golang} /go /go\n" +
				"RUN --mount=type=bind,from=golang${TAG},target=/go true\n",
			classification: "unsupported",
			candidateKinds: []string{"from", "copy-from", "run-mount-from"},
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
			name: "nested canonical form is never managed",
			contents: "FROM alpine\nONBUILD FROM golang:1.27.0@sha256:" + digest +
				" AS hidden\n",
			classification: "unsupported",
			candidateKinds: []string{"from"},
		},
		{
			name: "decoded JSON and heredoc run literals are unsupported",
			contents: "FROM alpine\n" +
				"RUN [\"docker\",\"pull\",\"gol\\u0061ng:latest\"]\n" +
				"ENV GO_BUILDER=golang:latest\n" +
				"RUN <<EOF\ndocker pull golang:latest\nEOF\n",
			classification: "unsupported",
			candidateKinds: []string{"literal", "literal"},
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
		got, err := discovery.normalizeDockerWord(input, false)
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
		} else if projected := projectDockerShellWord(input, '\\'); projected != buildKit {
			t.Errorf("projectDockerShellWord(%q) = %q, BuildKit = %q", input, projected, buildKit)
		}
	}
}

func TestDockerWordNormalizationBudgetsAndMemoization(t *testing.T) {
	discovery := newDockerDiscovery('\\')
	value := strings.Repeat("alpine", 100)
	if _, err := discovery.normalizeDockerWord(value, false); err != nil {
		t.Fatal(err)
	}
	bytesAfterFirst := discovery.normalizedBytes
	if _, err := discovery.normalizeDockerWord(value, false); err != nil {
		t.Fatal(err)
	}
	if discovery.normalizedBytes != bytesAfterFirst {
		t.Fatalf("memoized normalization increased byte budget from %d to %d", bytesAfterFirst, discovery.normalizedBytes)
	}

	deep := strings.Repeat("${A:-", maxDockerParameterDepth+1) + "alpine" + strings.Repeat("}", maxDockerParameterDepth+1)
	if _, err := discovery.normalizeDockerWord(deep, false); !isResourceLimit(err, "parameter expansion depth") {
		t.Fatalf("deep Docker word error = %v", err)
	}

	byteBudget := newDockerDiscovery('\\')
	large := strings.Repeat("a", maxDockerNormalizedBytes/2+1)
	if _, err := byteBudget.normalizeDockerWord(large, false); err != nil {
		t.Fatal(err)
	}
	if _, err := byteBudget.normalizeDockerWord(large+"b", false); !isResourceLimit(err, "normalized-word limit") {
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
