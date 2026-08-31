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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	buildkitshell "github.com/moby/buildkit/frontend/dockerfile/shell"
)

type dockerContract struct {
	SchemaVersion           int                            `json:"schemaVersion"`
	BuildKitWordOracles     []buildKitWordOracle           `json:"buildkitWordOracles"`
	ShellOracles            []shellOracle                  `json:"shellOracles"`
	ExecutableCrossProducts []dockerExecutableCrossProduct `json:"executableCrossProducts"`
	Cases                   []dockerClassificationCase     `json:"cases"`
}

type dockerExecutableCrossProduct struct {
	ID           string                   `json:"id"`
	Sources      []dockerExecutableSource `json:"sources"`
	Instructions []string                 `json:"instructions"`
	Forms        []string                 `json:"forms"`
	Contexts     []string                 `json:"contexts"`
}

type dockerExecutableSource struct {
	ID            string `json:"id"`
	Value         string `json:"value"`
	CandidateKind string `json:"candidateKind"`
}

type buildKitWordOracle struct {
	ID     string            `json:"id"`
	Escape string            `json:"escape"`
	Word   string            `json:"word"`
	Env    map[string]string `json:"env"`
	Want   string            `json:"want"`
}

type shellOracle struct {
	ID       string            `json:"id"`
	Script   string            `json:"script"`
	Env      map[string]string `json:"env"`
	WantArgv []string          `json:"wantArgv"`
}

type dockerClassificationCase struct {
	ID         string                   `json:"id"`
	Dockerfile string                   `json:"dockerfile"`
	Generator  *dockerContractGenerator `json:"generator"`
	Want       dockerContractWant       `json:"want"`
}

type dockerContractGenerator struct {
	Kind        string `json:"kind"`
	Depth       int    `json:"depth"`
	Leaf        string `json:"leaf"`
	Instruction string `json:"instruction"`
	Count       int    `json:"count"`
	Prefix      string `json:"prefix"`
}

type dockerContractWant struct {
	Classification string   `json:"classification"`
	CandidateKinds []string `json:"candidateKinds"`
	ErrorContains  string   `json:"errorContains"`
}

func readDockerContract(t *testing.T) dockerContract {
	t.Helper()
	contents, err := os.ReadFile("testdata/docker-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract dockerContract
	if err := json.Unmarshal(contents, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.SchemaVersion != 1 {
		t.Fatalf("unsupported Docker contract schema %d", contract.SchemaVersion)
	}
	return contract
}

func TestDockerContractClassificationMatrix(t *testing.T) {
	contract := readDockerContract(t)
	seen := map[string]bool{}
	for _, testCase := range contract.Cases {
		testCase := testCase
		t.Run(testCase.ID, func(t *testing.T) {
			if testCase.ID == "" || seen[testCase.ID] {
				t.Fatalf("contract case ID %q is empty or duplicated", testCase.ID)
			}
			seen[testCase.ID] = true
			contents := testCase.Dockerfile
			if testCase.Generator != nil {
				contents = generateDockerContractInput(t, *testCase.Generator)
			}
			classification, candidates, parseError := classifyDockerfile(contents)
			if classification != testCase.Want.Classification {
				t.Fatalf("classification = %q, want %q (error %q)", classification, testCase.Want.Classification, parseError)
			}
			kinds := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				kinds = append(kinds, candidate.Kind)
			}
			if !slices.Equal(kinds, testCase.Want.CandidateKinds) {
				t.Fatalf("candidate kinds = %q, want %q", kinds, testCase.Want.CandidateKinds)
			}
			if testCase.Want.ErrorContains != "" && !strings.Contains(parseError, testCase.Want.ErrorContains) {
				t.Fatalf("error %q does not contain %q", parseError, testCase.Want.ErrorContains)
			}
		})
	}
}

func TestDockerContractExecutableCrossProducts(t *testing.T) {
	contract := readDockerContract(t)
	seen := map[string]bool{}
	for _, matrix := range contract.ExecutableCrossProducts {
		matrix := matrix
		for _, source := range matrix.Sources {
			for _, instruction := range matrix.Instructions {
				for _, form := range matrix.Forms {
					for _, context := range matrix.Contexts {
						id := strings.Join([]string{matrix.ID, source.ID, strings.ToLower(instruction), form, context}, "/")
						t.Run(id, func(t *testing.T) {
							if matrix.ID == "" || source.ID == "" || seen[id] {
								t.Fatalf("generated contract case ID %q is empty or duplicated", id)
							}
							seen[id] = true
							contents, valid := renderDockerExecutableCase(t, source.Value, instruction, form, context)
							classification, candidates, parseError := classifyDockerfile(contents)
							if !valid {
								if classification != "invalid" {
									t.Fatalf("classification = %q, want invalid (error %q)", classification, parseError)
								}
								return
							}
							if classification != "unsupported" {
								t.Fatalf("classification = %q, want unsupported (error %q)", classification, parseError)
							}
							kinds := make([]string, 0, len(candidates))
							for _, candidate := range candidates {
								kinds = append(kinds, candidate.Kind)
							}
							if want := []string{source.CandidateKind}; !slices.Equal(kinds, want) {
								t.Fatalf("candidate kinds = %q, want %q", kinds, want)
							}
						})
					}
				}
			}
		}
	}
}

func renderDockerExecutableCase(t *testing.T, source, instruction, form, context string) (string, bool) {
	t.Helper()
	prefix := instruction
	if instruction == "HEALTHCHECK" {
		prefix += " CMD"
	}
	var payload string
	switch form {
	case "shell":
		payload = fmt.Sprintf("%s echo %s", prefix, source)
	case "exec":
		encoded, err := json.Marshal([]string{"echo", source})
		if err != nil {
			t.Fatal(err)
		}
		payload = prefix + " " + string(encoded)
	case "heredoc":
		payload = fmt.Sprintf("%s <<EOF\necho %s\nEOF", prefix, source)
	default:
		t.Fatalf("unknown executable form %q", form)
	}
	switch context {
	case "top-level":
	case "onbuild":
		payload = "ONBUILD " + payload
	default:
		t.Fatalf("unknown executable context %q", context)
	}
	// BuildKit only defines executable heredoc files for RUN. The remaining
	// combinations stay in the product as negative grammar cases.
	valid := form != "heredoc" || instruction == "RUN"
	return "FROM alpine\n" + payload + "\n", valid
}

func generateDockerContractInput(t *testing.T, generator dockerContractGenerator) string {
	t.Helper()
	switch generator.Kind {
	case "nested-posix-default":
		return "FROM alpine\nRUN echo " + strings.Repeat("${A:-", generator.Depth) + generator.Leaf + strings.Repeat("}", generator.Depth) + "\n"
	case "repeat-instruction":
		var contents strings.Builder
		contents.WriteString(generator.Prefix)
		contents.WriteByte('\n')
		for range generator.Count {
			contents.WriteString(generator.Instruction)
			contents.WriteByte('\n')
		}
		return contents.String()
	default:
		t.Fatalf("unknown Docker contract generator %q", generator.Kind)
		return ""
	}
}

func TestDockerContractBuildKitWordOracles(t *testing.T) {
	for _, oracle := range readDockerContract(t).BuildKitWordOracles {
		oracle := oracle
		t.Run(oracle.ID, func(t *testing.T) {
			escapeRunes := []rune(oracle.Escape)
			if len(escapeRunes) != 1 {
				t.Fatalf("escape %q must contain exactly one rune", oracle.Escape)
			}
			escape := escapeRunes[0]
			lexer := buildkitshell.NewLex(escape)
			lexer.SkipUnsetEnv = true
			environment := make([]string, 0, len(oracle.Env))
			for key, value := range oracle.Env {
				environment = append(environment, key+"="+value)
			}
			got, _, err := lexer.ProcessWord(oracle.Word, buildkitshell.EnvsFromSlice(environment))
			if err != nil {
				t.Fatal(err)
			}
			if got != oracle.Want {
				t.Fatalf("normalized word = %q, want %q", got, oracle.Want)
			}
		})
	}
}

func TestDockerContractShellOracles(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh is not available")
	}
	for _, oracle := range readDockerContract(t).ShellOracles {
		oracle := oracle
		t.Run(oracle.ID, func(t *testing.T) {
			wrapper := "capture() { printf '%s\\n' \"$@\"; }\n" + oracle.Script
			command := exec.Command("/bin/sh", "-c", wrapper)
			command.Env = []string{"PATH=/usr/bin:/bin"}
			for key, value := range oracle.Env {
				command.Env = append(command.Env, key+"="+value)
			}
			output, err := command.Output()
			if err != nil {
				t.Fatalf("/bin/sh oracle failed: %v", err)
			}
			got := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
			if len(oracle.WantArgv) == 0 && len(got) == 1 && got[0] == "" {
				got = nil
			}
			if !slices.Equal(got, oracle.WantArgv) {
				t.Fatalf("/bin/sh argv = %q, want %q", got, oracle.WantArgv)
			}
		})
	}
}
