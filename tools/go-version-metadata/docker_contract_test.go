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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	buildkitshell "github.com/moby/buildkit/frontend/dockerfile/shell"
)

type dockerContract struct {
	SchemaVersion           int                            `json:"schemaVersion"`
	BuildKitWordOracles     []buildKitWordOracle           `json:"buildkitWordOracles"`
	ShellOracles            []shellOracle                  `json:"shellOracles"`
	ExecutableCrossProducts []dockerExecutableCrossProduct `json:"executableCrossProducts"`
	WordExpansionProducts   []dockerWordExpansionProduct   `json:"wordExpansionCrossProducts"`
	DockerConformance       []dockerConformanceCase        `json:"dockerConformance"`
	Cases                   []dockerClassificationCase     `json:"cases"`
}

type dockerWordExpansionProduct struct {
	ID        string                        `json:"id"`
	Variable  string                        `json:"variable"`
	Sources   []dockerExecutableSource      `json:"sources"`
	Fields    []dockerWordExpansionField    `json:"fields"`
	Operators []dockerWordExpansionOperator `json:"operators"`
}

type dockerWordExpansionField struct {
	ID            string `json:"id"`
	Template      string `json:"template"`
	CandidateKind string `json:"candidateKind"`
}

type dockerWordExpansionOperator struct {
	Token string          `json:"token"`
	Want  map[string]bool `json:"wantSourceByState"`
}

type dockerConformanceCase struct {
	ID            string                   `json:"id"`
	Finding       int                      `json:"finding"`
	Domain        string                   `json:"domain"`
	Dockerfile    string                   `json:"dockerfile"`
	Generator     *dockerContractGenerator `json:"generator"`
	Accepted      bool                     `json:"accepted"`
	ErrorContains string                   `json:"errorContains"`
	Want          dockerContractWant       `json:"want"`
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
	Bytes       int    `json:"bytes"`
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
	validateDockerContractCoverage(t, contract)
	return contract
}

func validateDockerContractCoverage(t *testing.T, contract dockerContract) {
	t.Helper()
	if len(contract.Cases) == 0 {
		t.Error("classification cases must not be empty")
	}
	caseIDs := make([]string, 0, len(contract.Cases))
	for _, testCase := range contract.Cases {
		caseIDs = append(caseIDs, testCase.ID)
		if testCase.Want.Classification == "" || (testCase.Dockerfile == "") == (testCase.Generator == nil) {
			t.Errorf("classification case %q has an empty required field", testCase.ID)
		}
	}
	requireNonemptyUniqueIDs(t, "classification case", caseIDs)
	if len(contract.DockerConformance) == 0 {
		t.Error("Docker conformance cases must not be empty")
	}
	conformanceIDs := make([]string, 0, len(contract.DockerConformance))
	for _, testCase := range contract.DockerConformance {
		conformanceIDs = append(conformanceIDs, testCase.ID)
		if testCase.Domain == "" || testCase.Want.Classification == "" || (testCase.Dockerfile == "") == (testCase.Generator == nil) {
			t.Errorf("Docker conformance case %q has an empty required field", testCase.ID)
		}
	}
	requireNonemptyUniqueIDs(t, "Docker conformance case", conformanceIDs)
	requireExactIDs(t, "BuildKit word oracle", []string{
		"docker-backslash-concat",
		"docker-double-quote-concat",
		"docker-single-quote-concat",
		"docker-unicode-parameter-default",
		"docker-parameter-value",
		"alternate-docker-escape-does-not-make-backslash-a-shell-escape",
	}, slices.Collect(func(yield func(string) bool) {
		for _, oracle := range contract.BuildKitWordOracles {
			if oracle.Word == "" || oracle.Escape == "" || oracle.Want == "" {
				t.Errorf("BuildKit word oracle %q has an empty required field", oracle.ID)
			}
			yield(oracle.ID)
		}
	}))
	requireExactIDs(t, "shell oracle", []string{
		"sh-backslash-concat",
		"sh-double-quote-concat",
		"sh-single-quote-concat",
		"sh-backslash-newline",
		"sh-comment-to-newline",
		"sh-parameter-name",
		"sh-parameter-operand",
		"sh-parameter-set",
	}, slices.Collect(func(yield func(string) bool) {
		for _, oracle := range contract.ShellOracles {
			if oracle.Script == "" || oracle.WantArgv == nil {
				t.Errorf("shell oracle %q has an empty required field", oracle.ID)
			}
			yield(oracle.ID)
		}
	}))

	if len(contract.ExecutableCrossProducts) != 1 {
		t.Errorf("executable cross products = %d, want exactly 1", len(contract.ExecutableCrossProducts))
	} else {
		matrix := contract.ExecutableCrossProducts[0]
		if matrix.ID != "runtime-source" {
			t.Errorf("executable cross-product ID = %q, want runtime-source", matrix.ID)
		}
		requireExactIDs(t, "executable source", []string{"image", "download"}, executableSourceIDs(matrix.Sources))
		requireExactIDs(t, "executable instruction", []string{"RUN", "CMD", "ENTRYPOINT", "HEALTHCHECK"}, matrix.Instructions)
		requireExactIDs(t, "executable form", []string{"shell", "exec", "heredoc"}, matrix.Forms)
		requireExactIDs(t, "executable context", []string{"top-level", "onbuild"}, matrix.Contexts)
		if got, want := len(matrix.Sources)*len(matrix.Instructions)*len(matrix.Forms)*len(matrix.Contexts), 48; got != want {
			t.Errorf("executable cross-product cardinality = %d, want %d", got, want)
		}
		for _, source := range matrix.Sources {
			if source.Value == "" || source.CandidateKind == "" {
				t.Errorf("executable source %q has an empty required field", source.ID)
			}
		}
	}

	if len(contract.WordExpansionProducts) != 1 {
		t.Errorf("word-expansion cross products = %d, want exactly 1", len(contract.WordExpansionProducts))
	} else {
		product := contract.WordExpansionProducts[0]
		if product.ID != "conditional-source-branch" || product.Variable != "VALUE" {
			t.Errorf("word-expansion identity = %q/%q, want conditional-source-branch/VALUE", product.ID, product.Variable)
		}
		requireExactIDs(t, "word-expansion source", []string{"image", "download"}, executableSourceIDs(product.Sources))
		for _, source := range product.Sources {
			if source.Value == "" || source.CandidateKind == "" {
				t.Errorf("word-expansion source %q has an empty required field", source.ID)
			}
		}
		requireExactIDs(t, "word-expansion field", []string{"arg", "env"}, slices.Collect(func(yield func(string) bool) {
			for _, field := range product.Fields {
				if field.Template == "" || field.CandidateKind == "" {
					t.Errorf("word-expansion field %q has an empty required field", field.ID)
				}
				yield(field.ID)
			}
		}))
		requireExactIDs(t, "word-expansion operator", []string{":-", "-", ":+", "+"}, slices.Collect(func(yield func(string) bool) {
			for _, operator := range product.Operators {
				requireExactIDs(t, "operator "+operator.Token+" state", []string{"unset", "empty", "nonempty"}, slices.Collect(func(yieldState func(string) bool) {
					for state := range operator.Want {
						yieldState(state)
					}
				}))
				yield(operator.Token)
			}
		}))
		if got, want := len(product.Sources)*len(product.Fields)*len(product.Operators)*3, 48; got != want {
			t.Errorf("word-expansion cross-product cardinality = %d, want %d", got, want)
		}
	}
}

func requireNonemptyUniqueIDs(t *testing.T, domain string, ids []string) {
	t.Helper()
	if len(ids) == 0 {
		t.Errorf("%s IDs must not be empty", domain)
		return
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			t.Errorf("%s ID %q is empty or duplicated", domain, id)
		}
		seen[id] = true
	}
}

func executableSourceIDs(sources []dockerExecutableSource) []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.ID)
	}
	return ids
}

func requireExactIDs(t *testing.T, domain string, want, got []string) {
	t.Helper()
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("%s IDs/tokens = %q, want exactly %q", domain, got, want)
	}
}

func TestDockerContractStructuralCoverage(t *testing.T) {
	readDockerContract(t)
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

func TestDockerContractWordExpansionCrossProducts(t *testing.T) {
	for _, product := range readDockerContract(t).WordExpansionProducts {
		product := product
		if product.ID == "" || product.Variable == "" {
			t.Fatalf("matrix ID and variable must not be empty: %#v", product)
		}
		states := map[string][]string{
			"unset":    nil,
			"empty":    {product.Variable + "="},
			"nonempty": {product.Variable + "=x"},
		}
		for _, source := range product.Sources {
			for _, operator := range product.Operators {
				word := "${" + product.Variable + operator.Token + source.Value + "}"
				t.Run(strings.Join([]string{product.ID, source.ID, operator.Token, "exact-alternatives"}, "/"), func(t *testing.T) {
					lexer := buildkitshell.NewLex('\\')
					wantSet := map[string]bool{}
					for _, state := range []string{"unset", "empty", "nonempty"} {
						normalized, _, err := lexer.ProcessWord(word, buildkitshell.EnvsFromSlice(states[state]))
						if err != nil {
							t.Fatal(err)
						}
						wantSet[normalized] = true
					}
					want := make([]string, 0, len(wantSet))
					for value := range wantSet {
						want = append(want, value)
					}
					slices.Sort(want)
					discovery := newDockerDiscovery('\\')
					got, err := discovery.dockerWordAlternatives(word)
					if err != nil {
						t.Fatal(err)
					}
					slices.Sort(got)
					if !slices.Equal(got, want) {
						t.Fatalf("alternatives = %q, want exact BuildKit alternatives %q", got, want)
					}
				})
				for state, environment := range states {
					state, environment := state, environment
					t.Run(strings.Join([]string{product.ID, source.ID, operator.Token, state}, "/"), func(t *testing.T) {
						want, found := operator.Want[state]
						if !found {
							t.Fatalf("operator %q has no expectation for state %q", operator.Token, state)
						}
						lexer := buildkitshell.NewLex('\\')
						normalized, _, err := lexer.ProcessWord(word, buildkitshell.EnvsFromSlice(environment))
						if err != nil {
							t.Fatal(err)
						}
						if got := normalized == source.Value; got != want {
							t.Fatalf("source branch selected = %t, want %t; normalized value %q", got, want, normalized)
						}
					})
				}
				for _, field := range product.Fields {
					field := field
					t.Run(strings.Join([]string{product.ID, source.ID, operator.Token, field.ID}, "/"), func(t *testing.T) {
						instruction := strings.ReplaceAll(field.Template, "{{WORD}}", word)
						classification, candidates, parseError := classifyDockerfile("FROM alpine\n" + instruction + "\n")
						if classification != "unsupported" {
							t.Fatalf("classification = %q, want unsupported (error %q)", classification, parseError)
						}
						kinds := make([]string, 0, len(candidates))
						for _, candidate := range candidates {
							kinds = append(kinds, candidate.Kind)
						}
						if want := []string{field.CandidateKind}; !slices.Equal(kinds, want) {
							t.Fatalf("candidate kinds = %q, want %q", kinds, want)
						}
					})
				}
			}
		}
	}
}

func TestDockerContractAgainstDocker(t *testing.T) {
	if os.Getenv("KFP_RUN_DOCKER_CONFORMANCE") != "1" {
		t.Skip("set KFP_RUN_DOCKER_CONFORMANCE=1 to validate the contract with Docker")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}
	contract := readDockerContract(t)
	contextDirectory := t.TempDir()
	for _, testCase := range contract.DockerConformance {
		testCase := testCase
		t.Run(testCase.ID, func(t *testing.T) {
			contents := testCase.Dockerfile
			if testCase.Generator != nil {
				contents = generateDockerContractInput(t, *testCase.Generator)
			}
			assertDockerClassification(t, contents, testCase.Want)
			commandContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(commandContext, "docker", "build", "--check", "-f", "-", contextDirectory)
			// Conformance is about Dockerfile acceptance, not optional lint
			// warnings such as UndefinedVar on deliberately adversarial inputs.
			command.Stdin = strings.NewReader("# check=skip=all\n" + contents)
			output, err := command.CombinedOutput()
			if commandContext.Err() != nil {
				t.Fatalf("Docker conformance check timed out: %v", commandContext.Err())
			}
			if testCase.Accepted && err != nil {
				t.Fatalf("Docker rejected accepted input: %v\n%s", err, boundedDockerOutput(output))
			}
			if !testCase.Accepted && err == nil {
				t.Fatalf("Docker accepted rejected input:\n%s", boundedDockerOutput(output))
			}
			if testCase.ErrorContains != "" && !strings.Contains(string(output), testCase.ErrorContains) {
				t.Fatalf("Docker output does not contain %q:\n%s", testCase.ErrorContains, boundedDockerOutput(output))
			}
		})
	}
}

func TestDockerContractExecutableOracleCoverage(t *testing.T) {
	seen := map[string]bool{}
	domains := map[string]int{}
	for _, testCase := range readDockerContract(t).DockerConformance {
		if testCase.ID == "" || seen[testCase.ID] {
			t.Errorf("Docker conformance case ID %q is empty or duplicated", testCase.ID)
		}
		seen[testCase.ID] = true
		if testCase.Domain == "" {
			t.Errorf("%s: semantic domain must not be empty", testCase.ID)
		}
		domains[testCase.Domain]++
		if testCase.Want.Classification == "" {
			t.Errorf("%s: executable classification oracle is required", testCase.ID)
		}
		if (testCase.Dockerfile == "") == (testCase.Generator == nil) {
			t.Errorf("%s: exactly one of dockerfile or generator must be set", testCase.ID)
		}
		contents := testCase.Dockerfile
		if testCase.Generator != nil {
			contents = generateDockerContractInput(t, *testCase.Generator)
		}
		assertDockerClassification(t, contents, testCase.Want)
	}
	for _, domain := range []string{
		"numeric-from-external",
		"numeric-run-mount-external",
		"negative-copy-index",
		"from-alias-normalized-local",
		"from-alias-raw-external",
		"copy-alias-case-insensitive",
	} {
		if domains[domain] != 1 {
			t.Errorf("required Docker conformance domain %q has %d cases, want exactly 1", domain, domains[domain])
		}
	}
}

func assertDockerClassification(t *testing.T, contents string, want dockerContractWant) {
	t.Helper()
	classification, candidates, parseError := classifyDockerfile(contents)
	if classification != want.Classification {
		t.Fatalf("classification = %q, want %q (error %q)", classification, want.Classification, parseError)
	}
	if want.CandidateKinds != nil {
		kinds := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			kinds = append(kinds, candidate.Kind)
		}
		if !slices.Equal(kinds, want.CandidateKinds) {
			t.Fatalf("candidate kinds = %q, want %q", kinds, want.CandidateKinds)
		}
	}
	if want.ErrorContains != "" && !strings.Contains(parseError, want.ErrorContains) {
		t.Fatalf("error %q does not contain %q", parseError, want.ErrorContains)
	}
}

func boundedDockerOutput(output []byte) string {
	const limit = 8 << 10
	if len(output) <= limit {
		return string(output)
	}
	return string(output[:limit]) + fmt.Sprintf("\n... %d bytes omitted ...", len(output)-limit)
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
	case "distinct-nested-docker-defaults":
		var contents strings.Builder
		contents.WriteString("FROM alpine\n")
		for index := range generator.Count {
			fmt.Fprintf(&contents, "ARG A%d=%s%s%s%d\n", index,
				strings.Repeat("${A:-", generator.Depth), generator.Leaf,
				strings.Repeat("}", generator.Depth), index)
		}
		result := contents.String()
		if generator.Bytes != 0 && len(result) != generator.Bytes {
			t.Fatalf("generated Docker contract input is %d bytes, want %d", len(result), generator.Bytes)
		}
		return result
	case "near-limit-docker-word":
		if generator.Bytes < len(generator.Prefix) {
			t.Fatalf("Docker word size %d is shorter than prefix %q", generator.Bytes, generator.Prefix)
		}
		word := generator.Prefix + strings.Repeat("a", generator.Bytes-len(generator.Prefix))
		if len(word) != generator.Bytes {
			t.Fatalf("generated Docker word is %d bytes, want %d", len(word), generator.Bytes)
		}
		return "FROM alpine\nARG IMAGE=" + word + "\n"
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
