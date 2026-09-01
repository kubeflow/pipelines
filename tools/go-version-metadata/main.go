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

// go-version-metadata parses repository metadata using the same syntax-aware
// libraries as the formats it enforces. It reads JSON from stdin and writes
// JSON to stdout.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/expand"
	shsyntax "mvdan.cc/sh/v3/syntax"
)

type request struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

type moduleMetadata struct {
	Go        string `json:"go"`
	Toolchain string `json:"toolchain,omitempty"`
}

type dockerCandidate struct {
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	Line    int    `json:"line"`
	Version string `json:"version,omitempty"`
	Flavor  string `json:"flavor,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Alias   string `json:"alias,omitempty"`
}

type response struct {
	YAMLValues           map[string][]string `json:"yamlValues,omitempty"`
	HasGoDownload        bool                `json:"hasGoDownload,omitempty"`
	DockerClassification string              `json:"dockerClassification,omitempty"`
	DockerCandidates     []dockerCandidate   `json:"dockerCandidates,omitempty"`
	DockerError          string              `json:"dockerError,omitempty"`
	Module               *moduleMetadata     `json:"module,omitempty"`
}

var goDownloadPattern = regexp.MustCompile(`(?i)^https://(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var goDownloadTextPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9+.-])https://(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var exactToolchainVersionPattern = regexp.MustCompile(`^1\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)
var canonicalDockerGoImagePattern = regexp.MustCompile(`^FROM golang:((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))(-[a-z0-9][a-z0-9._-]*)?@sha256:([0-9a-f]{64}) AS ([a-z0-9][a-z0-9_.-]*)$`)

const (
	maxInputBytes             = 4 << 20
	maxRequestEnvelopeBytes   = 32 << 20
	maxYAMLDocuments          = 64
	maxYAMLNodes              = 100000
	maxYAMLEdges              = 150000
	maxYAMLDepth              = 256
	maxYAMLScalarBytes        = 1 << 20
	maxDockerInstructions     = 100000
	maxDockerCandidates       = 10000
	maxDockerInstructionDepth = 256
	maxShellASTNodes          = 100000
	maxShellASTDepth          = 256
	maxDockerNormalizedBytes  = 16 << 20
	maxDockerWordInputBytes   = 16 << 10
	maxDockerWordWorkBytes    = 32 << 10
	maxDockerWordVariables    = 6
	maxDockerWordAlternatives = 729
	maxDockerAlternativeWork  = 1 << 20
)

type resourceLimitError struct {
	message string
}

func (err *resourceLimitError) Error() string {
	return err.message
}

func resourceLimitf(format string, arguments ...any) error {
	return &resourceLimitError{message: fmt.Sprintf(format, arguments...)}
}

type unsupportedDockerPolicyError struct {
	candidate dockerCandidate
	message   string
}

func (err *unsupportedDockerPolicyError) Error() string {
	return err.message
}

func unsupportedDockerPolicyf(kind, value string, line int, format string, arguments ...any) error {
	return &unsupportedDockerPolicyError{
		candidate: dockerCandidate{Kind: kind, Value: value, Line: line},
		message:   fmt.Sprintf(format, arguments...),
	}
}

func main() {
	input, err := decodeRequest(os.Stdin)
	if err != nil {
		fail(fmt.Errorf("decode request: %w", err))
	}
	metadata, err := inspect(input)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(metadata); err != nil {
		fail(fmt.Errorf("encode response: %w", err))
	}
}

func fail(err error) {
	var limitError *resourceLimitError
	if errors.As(err, &limitError) {
		fmt.Fprintf(os.Stderr, "resource limit: %s\n", limitError)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

func decodeRequest(reader io.Reader) (request, error) {
	encoded, err := io.ReadAll(io.LimitReader(reader, maxRequestEnvelopeBytes+1))
	if err != nil {
		return request{}, err
	}
	if len(encoded) > maxRequestEnvelopeBytes {
		return request{}, resourceLimitf("request exceeds the %d-byte envelope limit", maxRequestEnvelopeBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var input request
	if err := decoder.Decode(&input); err != nil {
		return request{}, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return request{}, fmt.Errorf("request contains multiple JSON values")
		}
		return request{}, fmt.Errorf("request contains trailing data: %w", err)
	}
	return input, nil
}

func inspect(input request) (response, error) {
	if len(input.Contents) > maxInputBytes {
		return response{}, resourceLimitf("%s exceeds the %d-byte metadata input limit", input.Path, maxInputBytes)
	}
	name := filepath.Base(input.Path)
	ext := strings.ToLower(filepath.Ext(name))
	metadata := response{}

	if name == "go.mod" {
		parsed, err := modfile.Parse(input.Path, []byte(input.Contents), nil)
		if err != nil {
			return response{}, fmt.Errorf("%s: %w", input.Path, err)
		}
		if parsed.Go == nil {
			return response{}, fmt.Errorf("%s must contain exactly one go directive", input.Path)
		}
		metadata.Module = &moduleMetadata{Go: parsed.Go.Version}
		if parsed.Toolchain != nil {
			toolchainVersion := strings.TrimPrefix(parsed.Toolchain.Name, "go")
			if !exactToolchainVersionPattern.MatchString(toolchainVersion) {
				return response{}, fmt.Errorf("%s contains an invalid toolchain directive %q", input.Path, parsed.Toolchain.Name)
			}
			metadata.Module.Toolchain = toolchainVersion
		}
		return metadata, nil
	}

	if ext == ".yaml" || ext == ".yml" {
		values, hasDownload, err := inspectYAML(input.Contents)
		if err != nil {
			return response{}, fmt.Errorf("%s: %w", input.Path, err)
		}
		metadata.YAMLValues = values
		metadata.HasGoDownload = hasDownload
		return metadata, nil
	}

	if isContainerRecipe(name) {
		classification, candidates, parseError := classifyDockerfile(input.Contents)
		metadata.DockerClassification = classification
		metadata.DockerCandidates = candidates
		metadata.DockerError = parseError
	}
	return metadata, nil
}

func isContainerRecipe(name string) bool {
	return strings.HasPrefix(name, "Dockerfile") || strings.HasPrefix(name, "Containerfile")
}

func inspectYAML(contents string) (map[string][]string, bool, error) {
	values := map[string][]string{
		"container": {},
		"image":     {},
		"uses":      {},
	}
	decoder := yaml.NewDecoder(strings.NewReader(contents))
	hasDownload := false
	state := yamlTraversalState{
		visited:    map[*yaml.Node]bool{},
		scalarMemo: map[*yaml.Node]string{},
	}
	documents := 0
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "exceeded max depth") {
				return nil, false, resourceLimitf("YAML parser %s", err)
			}
			return nil, false, err
		}
		documents++
		if documents > maxYAMLDocuments {
			return nil, false, resourceLimitf("YAML metadata exceeds the %d-document limit", maxYAMLDocuments)
		}
		if err := walkYAML(&document, values, &hasDownload, &state, 0); err != nil {
			return nil, false, err
		}
	}
	return values, hasDownload, nil
}

type yamlTraversalState struct {
	visited    map[*yaml.Node]bool
	scalarMemo map[*yaml.Node]string
	nodes      int
	edges      int
}

func walkYAML(node *yaml.Node, values map[string][]string, hasDownload *bool, state *yamlTraversalState, depth int) error {
	if node == nil || state.visited[node] {
		return nil
	}
	if depth > maxYAMLDepth {
		return resourceLimitf("YAML metadata exceeds the %d-level depth limit", maxYAMLDepth)
	}
	state.visited[node] = true
	state.nodes++
	if state.nodes > maxYAMLNodes {
		return resourceLimitf("YAML metadata exceeds the %d-node traversal limit", maxYAMLNodes)
	}
	state.edges += len(node.Content)
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		state.edges++
	}
	if state.edges > maxYAMLEdges {
		return resourceLimitf("YAML metadata exceeds the %d-edge traversal limit", maxYAMLEdges)
	}
	if node.Kind == yaml.ScalarNode && len(node.Value) > maxYAMLScalarBytes {
		return resourceLimitf("YAML metadata exceeds the %d-byte scalar limit", maxYAMLScalarBytes)
	}

	if node.Kind == yaml.AliasNode {
		return walkYAML(node.Alias, values, hasDownload, state, depth+1)
	}
	if node.Kind == yaml.ScalarNode && goDownloadTextPattern.MatchString(node.Value) {
		*hasDownload = true
	}
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := resolvedScalar(node.Content[index], state)
			valueNode := node.Content[index+1]
			if _, ok := values[key]; ok {
				if value := resolvedScalar(valueNode, state); value != "" {
					values[key] = appendUnique(values[key], value)
				}
			}
			if err := walkYAML(valueNode, values, hasDownload, state, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := walkYAML(child, values, hasDownload, state, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func resolvedScalar(node *yaml.Node, state *yamlTraversalState) string {
	original := node
	if value, ok := state.scalarMemo[original]; ok {
		return value
	}
	seen := map[*yaml.Node]bool{}
	for node != nil && node.Kind == yaml.AliasNode {
		if seen[node] {
			return ""
		}
		seen[node] = true
		node = node.Alias
	}
	if node != nil && node.Kind == yaml.ScalarNode {
		value := strings.TrimSpace(node.Value)
		state.scalarMemo[original] = value
		return value
	}
	state.scalarMemo[original] = ""
	return ""
}

func classifyDockerfile(contents string) (string, []dockerCandidate, string) {
	parsed, err := parser.Parse(strings.NewReader(contents))
	if err != nil {
		return "invalid", nil, err.Error()
	}
	managed := []dockerCandidate{}
	unsupported := []dockerCandidate{}
	discovery := newDockerDiscovery(parsed.EscapeToken)
	if err := validateDockerStructure(parsed.AST, discovery); err != nil {
		return dockerValidationFailure(err)
	}
	if err := validateDockerStageGraph(parsed.AST, discovery); err != nil {
		return dockerValidationFailure(err)
	}
	for _, node := range parsed.AST.Children {
		if strings.EqualFold(node.Value, "from") {
			if err := discovery.completeCurrentStage(); err != nil {
				return "invalid", nil, err.Error()
			}
		}
		if err := classifyDockerInstruction(node, true, 0, discovery, &managed, &unsupported); err != nil {
			return "invalid", nil, err.Error()
		}
	}
	if err := discovery.completeCurrentStage(); err != nil {
		return "invalid", nil, err.Error()
	}
	unsupported = append(unsupported, discovery.unconsumedDeferredCandidates()...)
	sort.SliceStable(unsupported, func(left, right int) bool {
		return unsupported[left].Line < unsupported[right].Line
	})
	if len(unsupported) != 0 || len(managed) > 1 {
		return "unsupported", append(managed, unsupported...), ""
	}
	if len(managed) == 1 {
		return "managed", managed, ""
	}
	return "irrelevant", nil, ""
}

func dockerValidationFailure(err error) (string, []dockerCandidate, string) {
	var unsupported *unsupportedDockerPolicyError
	if errors.As(err, &unsupported) {
		return "unsupported", []dockerCandidate{unsupported.candidate}, ""
	}
	return "invalid", nil, err.Error()
}

func validateDockerStructure(ast *parser.Node, discovery *dockerDiscovery) error {
	hasStage := false
	for _, node := range ast.Children {
		typed, err := parseDockerInstruction(node, discovery)
		if err != nil {
			return err
		}
		if _, ok := typed.(*instructions.Stage); ok {
			hasStage = true
			continue
		}
		if _, ok := typed.(*instructions.ArgCommand); ok && !hasStage {
			continue
		}
		if !hasStage {
			return fmt.Errorf("line %d: %s requires a preceding FROM instruction", node.StartLine, node.Value)
		}
	}
	if !hasStage {
		return fmt.Errorf("dockerfile must contain at least one FROM instruction")
	}
	return nil
}

func validateDockerStageGraph(ast *parser.Node, discovery *dockerDiscovery) error {
	type validationStage struct {
		name         string
		dependencies map[int]bool
	}
	stages := []*validationStage{}
	finalNames := map[string]int{}
	for _, node := range ast.Children {
		typed, err := parseDockerInstruction(node, discovery)
		if err != nil {
			return err
		}
		if stage, ok := typed.(*instructions.Stage); ok {
			index := len(stages)
			stages = append(stages, &validationStage{name: stage.Name, dependencies: map[int]bool{}})
			if stage.Name != "" {
				finalNames[strings.ToLower(stage.Name)] = index
			}
		}
	}
	resolveNamed := func(value string, names map[string]int) (int, bool) {
		index, found := names[strings.ToLower(value)]
		return index, found
	}
	validateLiteralSource := func(value string) error {
		if _, err := discovery.dockerWordAlternatives(value); err != nil {
			return err
		}
		if discovery.dockerWordHasUnknown(value) {
			return fmt.Errorf("expanded stage source %q is not supported by BuildKit", value)
		}
		return nil
	}
	current := -1
	priorNames := map[string]int{}
	for _, node := range ast.Children {
		typed, err := parseDockerInstruction(node, discovery)
		if err != nil {
			return err
		}
		switch command := typed.(type) {
		case *instructions.Stage:
			current++
			alternatives, err := discovery.dockerWordAlternatives(command.BaseName)
			if err != nil {
				return err
			}
			for _, base := range alternatives {
				// FROM resolves only previously declared named aliases. Numeric
				// values are external image names, not stage indices.
				if dependency, local := priorNames[base]; local {
					stages[current].dependencies[dependency] = true
				}
			}
			if command.Name != "" {
				priorNames[command.Name] = current
			}
		case *instructions.CopyCommand:
			if command.From != "" {
				if err := validateLiteralSource(command.From); err != nil {
					return fmt.Errorf("line %d: %w", node.StartLine, err)
				}
				dependency, local := 0, false
				if index, err := strconv.Atoi(command.From); err == nil {
					if index < 0 {
						return fmt.Errorf("line %d: invalid stage index %d", node.StartLine, index)
					}
					if index >= current {
						return fmt.Errorf("line %d: invalid stage index %d", node.StartLine, index)
					}
					dependency, local = index, true
				} else {
					dependency, local = resolveNamed(command.From, finalNames)
				}
				if local && dependency >= current {
					return fmt.Errorf("line %d: stage source %q must reference a prior stage", node.StartLine, command.From)
				}
				if local {
					stages[current].dependencies[dependency] = true
				}
			}
		case *instructions.RunCommand:
			for _, mount := range instructions.GetMounts(command) {
				if mount.From == "" {
					continue
				}
				if err := validateLiteralSource(mount.From); err != nil {
					return fmt.Errorf("line %d: %w", node.StartLine, err)
				}
				// RUN --mount=from uses named local aliases. A numeric token is
				// an external image name, unlike COPY --from's stage-index form.
				dependency, local := resolveNamed(mount.From, finalNames)
				if local && dependency >= current {
					return fmt.Errorf("line %d: stage source %q must reference a prior stage", node.StartLine, mount.From)
				}
				if local {
					stages[current].dependencies[dependency] = true
				}
			}
		case *instructions.OnbuildCommand:
			contents := fmt.Sprintf("# escape=%c\n%s", discovery.escapeToken, command.Expression)
			parsed, err := parser.Parse(strings.NewReader(contents))
			if err != nil || len(parsed.AST.Children) != 1 {
				continue
			}
			deferred, err := parseDockerInstruction(parsed.AST.Children[0], discovery)
			if err != nil {
				return err
			}
			switch deferred := deferred.(type) {
			case *instructions.CopyCommand:
				if deferred.From != "" {
					if err := validateLiteralSource(deferred.From); err != nil {
						return fmt.Errorf("line %d: ONBUILD COPY --from: %w", node.StartLine, err)
					}
				}
				if deferred.From != "" && deferredStageSourceIsOutsideContract(deferred.From, current, finalNames) {
					return unsupportedDockerPolicyf("copy-from", deferred.From, node.StartLine,
						"line %d: deferred COPY --from resolution is outside the offline stage-graph contract", node.StartLine)
				}
			case *instructions.RunCommand:
				for _, mount := range instructions.GetMounts(deferred) {
					if mount.From != "" {
						if err := validateLiteralSource(mount.From); err != nil {
							return fmt.Errorf("line %d: ONBUILD RUN --mount=from: %w", node.StartLine, err)
						}
					}
					if mount.From != "" && deferredStageSourceIsOutsideContract(mount.From, current, finalNames) {
						return unsupportedDockerPolicyf("run-mount-from", mount.From, node.StartLine,
							"line %d: deferred RUN --mount=from resolution is outside the offline stage-graph contract", node.StartLine)
					}
				}
			}
		}
	}
	visiting := make([]bool, len(stages))
	visited := make([]bool, len(stages))
	var visit func(int) error
	visit = func(index int) error {
		if visiting[index] {
			return fmt.Errorf("circular dependency detected on stage %d", index)
		}
		if visited[index] {
			return nil
		}
		visiting[index] = true
		for dependency := range stages[index].dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[index] = false
		visited[index] = true
		return nil
	}
	for index := range stages {
		if err := visit(index); err != nil {
			return err
		}
	}
	return nil
}

func deferredStageSourceIsOutsideContract(value string, definingStage int, finalNames map[string]int) bool {
	if _, err := strconv.Atoi(value); err == nil {
		return true
	}
	dependency, local := finalNames[strings.ToLower(value)]
	return local && dependency >= definingStage
}

type dockerWordKey struct {
	value string
}

type dockerWordResult struct {
	validation string
	values     []symbolicValue
	symbolic   []symbolicValue
	variables  map[string]bool
	unknown    bool
	err        error
}

type runtimeWordsKey struct {
	value      string
	parameters string
}

type runtimeWordsResult struct {
	values []runtimeWord
	err    error
}

type symbolicSegment struct {
	literal  string
	variable string
}

type symbolicValue struct {
	segments []symbolicSegment
}

type runtimeWord struct {
	value    string
	symbolic *symbolicValue
}

type runtimeShell struct {
	known bool
}

type deferredDockerInstruction struct {
	expression string
	line       int
	consumed   bool
}

type dockerStageState struct {
	references  []string
	baseName    string
	shell       runtimeShell
	triggers    []*deferredDockerInstruction
	provisional []dockerCandidate
}

type dockerInstructionContext struct {
	shell             runtimeShell
	conservative      bool
	currentReferences []string
}

type dockerDiscovery struct {
	escapeToken      rune
	wordLexer        *shell.Lex
	wordMemo         map[dockerWordKey]dockerWordResult
	runtimeWordsMemo map[runtimeWordsKey]runtimeWordsResult
	instructions     int
	candidates       int
	normalizedBytes  int
	wordWorkBytes    int
	alternativeWork  int
	stageReferences  map[string]*dockerStageState
	baseReferences   map[string]*dockerStageState
	allStages        []*dockerStageState
	currentStage     *dockerStageState
	stageCount       int
}

func newDockerDiscovery(escapeToken rune) *dockerDiscovery {
	wordLexer := shell.NewLex(escapeToken)
	wordLexer.SkipUnsetEnv = true
	return &dockerDiscovery{
		escapeToken:      escapeToken,
		wordLexer:        wordLexer,
		wordMemo:         map[dockerWordKey]dockerWordResult{},
		runtimeWordsMemo: map[runtimeWordsKey]runtimeWordsResult{},
		stageReferences:  map[string]*dockerStageState{},
		baseReferences:   map[string]*dockerStageState{},
	}
}

func classifyDockerInstruction(node *parser.Node, allowManaged bool, depth int, discovery *dockerDiscovery, managed *[]dockerCandidate, unsupported *[]dockerCandidate) error {
	if depth > maxDockerInstructionDepth {
		return resourceLimitf("Docker metadata exceeds the %d-level instruction depth limit", maxDockerInstructionDepth)
	}
	if err := discovery.admitInstructions(1); err != nil {
		return err
	}
	typed, err := parseDockerInstruction(node, discovery)
	if err != nil {
		return err
	}
	if !allowManaged {
		if _, ok := typed.(*instructions.Stage); ok {
			return fmt.Errorf("line %d: FROM is not permitted in ONBUILD", node.StartLine)
		}
	}
	if allowManaged {
		if stage, ok := typed.(*instructions.Stage); ok {
			if err := discovery.beginStage(stage, unsupported); err != nil {
				return err
			}
		}
	}
	original := node.Original
	canonical := canonicalDockerGoImagePattern.FindStringSubmatch(original)
	if allowManaged && strings.EqualFold(node.Value, "from") && node.StartLine == node.EndLine && canonical != nil {
		if err := discovery.admitCandidates(1); err != nil {
			return err
		}
		*managed = append(*managed, dockerCandidate{
			Kind:    "from",
			Value:   original,
			Line:    node.StartLine,
			Version: canonical[1],
			Flavor:  canonical[2],
			Digest:  "sha256:" + canonical[3],
			Alias:   canonical[4],
		})
	} else {
		candidates, err := dockerInstructionCandidates(node, typed, discovery.currentContext(), discovery)
		if err != nil {
			return fmt.Errorf("line %d: %w", node.StartLine, err)
		}
		if err := discovery.admitCandidates(len(candidates)); err != nil {
			return err
		}
		*unsupported = append(*unsupported, candidates...)
	}
	if allowManaged {
		if command, ok := typed.(*instructions.ShellCommand); ok {
			discovery.setRuntimeShell(command.Shell)
		}
	}
	if onbuild, ok := typed.(*instructions.OnbuildCommand); ok {
		if allowManaged {
			discovery.currentStage.triggers = append(discovery.currentStage.triggers, &deferredDockerInstruction{expression: onbuild.Expression, line: node.StartLine})
		} else {
			return fmt.Errorf("line %d: ONBUILD is not permitted in ONBUILD", node.StartLine)
		}
	}
	return nil
}

func parseDockerInstruction(node *parser.Node, discovery *dockerDiscovery) (any, error) {
	typed, err := instructions.ParseInstruction(node)
	if err != nil {
		return nil, err
	}
	if err := validateDockerInstructionWords(typed, discovery, node.StartLine); err != nil {
		return nil, err
	}
	return typed, nil
}

func validateDockerInstructionWords(typed any, discovery *dockerDiscovery, line int) error {
	validate := func(word string) (string, error) {
		if _, err := discovery.dockerWordAlternatives(word); err != nil {
			return "", err
		}
		result := discovery.wordMemo[dockerWordKey{value: word}]
		if operator, found := unsupportedDockerWordOperator(word, result.variables, discovery.escapeToken); found {
			return "", unsupportedDockerPolicyf("unsupported-word", word, line,
				"line %d: Docker parameter operator %q is outside the bounded word-expansion contract", line, operator)
		}
		return word, nil
	}
	if stage, ok := typed.(*instructions.Stage); ok {
		for _, word := range []string{stage.BaseName, stage.Platform} {
			if _, err := validate(word); err != nil {
				return err
			}
		}
	}
	if command, ok := typed.(instructions.SupportsSingleWordExpansion); ok {
		if err := command.Expand(validate); err != nil {
			return err
		}
	}
	if command, ok := typed.(instructions.SupportsSingleWordExpansionRaw); ok {
		if err := command.ExpandRaw(validate); err != nil {
			return err
		}
	}
	return nil
}

func (discovery *dockerDiscovery) evaluateDeferred(triggers []*deferredDockerInstruction, context dockerInstructionContext) ([]dockerCandidate, runtimeShell, error) {
	candidates := []dockerCandidate{}
	for _, trigger := range triggers {
		if err := discovery.admitInstructions(1); err != nil {
			return nil, context.shell, err
		}
		contents := fmt.Sprintf("# escape=%c\n%s", discovery.escapeToken, trigger.expression)
		parsed, err := parser.Parse(strings.NewReader(contents))
		if err != nil {
			return nil, context.shell, fmt.Errorf("line %d: parse ONBUILD trigger: %w", trigger.line, err)
		}
		if len(parsed.AST.Children) != 1 {
			return nil, context.shell, fmt.Errorf("line %d: ONBUILD must contain exactly one instruction", trigger.line)
		}
		node := parsed.AST.Children[0]
		node.StartLine = trigger.line
		typed, err := parseDockerInstruction(node, discovery)
		if err != nil {
			return nil, context.shell, fmt.Errorf("line %d: ONBUILD trigger: %w", trigger.line, err)
		}
		if _, forbidden := typed.(*instructions.Stage); forbidden {
			return nil, context.shell, fmt.Errorf("line %d: FROM is not permitted in ONBUILD", trigger.line)
		}
		if _, forbidden := typed.(*instructions.OnbuildCommand); forbidden {
			return nil, context.shell, fmt.Errorf("line %d: ONBUILD is not permitted in ONBUILD", trigger.line)
		}
		found, err := dockerInstructionCandidates(node, typed, context, discovery)
		if err != nil {
			return nil, context.shell, fmt.Errorf("line %d: ONBUILD trigger: %w", trigger.line, err)
		}
		if err := discovery.admitCandidates(len(found)); err != nil {
			return nil, context.shell, err
		}
		candidates = append(candidates, found...)
		if command, ok := typed.(*instructions.ShellCommand); ok {
			context.shell = runtimeShellFor(command.Shell)
		}
	}
	return candidates, context.shell, nil
}

func (discovery *dockerDiscovery) admitInstructions(count int) error {
	discovery.instructions += count
	if discovery.instructions > maxDockerInstructions {
		return resourceLimitf("Docker metadata exceeds the %d-instruction limit", maxDockerInstructions)
	}
	return nil
}

func (discovery *dockerDiscovery) admitCandidates(count int) error {
	discovery.candidates += count
	if discovery.candidates > maxDockerCandidates {
		return resourceLimitf("Docker metadata exceeds the %d-candidate limit", maxDockerCandidates)
	}
	return nil
}

func dockerInstructionCandidates(node *parser.Node, typed any, context dockerInstructionContext, discovery *dockerDiscovery) ([]dockerCandidate, error) {
	candidates := []dockerCandidate{}
	appendImageCandidate := func(kind, value string, rejectCurrent, numericLocal bool) error {
		alternatives, err := discovery.dockerWordAlternatives(value)
		if err != nil {
			return err
		}
		matched := false
		allLocal := len(alternatives) > 0
		for _, normalized := range alternatives {
			numeric := isNonnegativeDecimal(normalized)
			if rejectCurrent && (numericLocal || !numeric) && context.isCurrentStageReference(normalized) {
				return fmt.Errorf("%s cannot reference the current stage %q", kind, normalized)
			}
			local := discovery.isStageReference(normalized)
			if kind == "from" {
				// BuildKit normalizes declared aliases, but FROM compares its raw
				// base token against that normalized namespace. COPY --from and
				// RUN --mount=from, by contrast, fold their lookup keys.
				local = discovery.isBaseReference(normalized)
			}
			if !context.conservative && (numericLocal || !numeric) && local {
				continue
			}
			allLocal = false
			if isGolangImage(normalized) || containsDockerGoToken(normalized) {
				matched = true
				break
			}
		}
		if allLocal && !discovery.dockerWordHasUnknown(value) {
			return nil
		}
		if matched || discovery.dockerWordUnknownMayContainSource(value, false) || containsDockerGoToken(value) {
			candidates = append(candidates, dockerCandidate{Kind: kind, Value: value, Line: node.StartLine})
		}
		return nil
	}

	switch command := typed.(type) {
	case *instructions.Stage:
		if err := appendImageCandidate("from", command.BaseName, false, false); err != nil {
			return nil, err
		}
	case *instructions.ArgCommand:
		for _, argument := range command.Args {
			if argument.Value == nil {
				continue
			}
			alternatives, err := discovery.dockerWordAlternatives(*argument.Value)
			if err != nil {
				return nil, err
			}
			matched := false
			for _, normalized := range alternatives {
				if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			if matched || discovery.dockerWordUnknownMayContainSource(*argument.Value, false) || containsDockerGoToken(*argument.Value) || goDownloadTextPattern.MatchString(*argument.Value) {
				candidates = append(candidates, dockerCandidate{Kind: "arg-default", Value: *argument.Value, Line: node.StartLine})
			}
		}
	case *instructions.EnvCommand:
		for _, environment := range command.Env {
			alternatives, err := discovery.dockerWordAlternatives(environment.Value)
			if err != nil {
				return nil, err
			}
			matched := false
			for _, normalized := range alternatives {
				if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			if matched || discovery.dockerWordUnknownMayContainSource(environment.Value, false) || containsDockerGoToken(environment.Value) || goDownloadTextPattern.MatchString(environment.Value) {
				candidates = append(candidates, dockerCandidate{Kind: "env-value", Value: environment.Value, Line: node.StartLine})
			}
		}
	case *instructions.AddCommand:
		for _, source := range command.SourcePaths {
			alternatives, err := discovery.dockerWordAlternatives(source)
			if err != nil {
				return nil, err
			}
			matched := false
			for _, normalized := range alternatives {
				if goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			if matched || discovery.dockerWordUnknownMayContainSource(source, true) || goDownloadTextPattern.MatchString(source) {
				candidates = append(candidates, dockerCandidate{Kind: "add-download", Value: source, Line: node.StartLine})
			}
		}
	case *instructions.CopyCommand:
		if command.From != "" {
			if err := appendImageCandidate("copy-from", command.From, true, true); err != nil {
				return nil, err
			}
		}
	case *instructions.RunCommand:
		for _, mount := range instructions.GetMounts(command) {
			if mount.From == "" {
				continue
			}
			if err := appendImageCandidate("run-mount-from", mount.From, true, false); err != nil {
				return nil, err
			}
		}
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine, context.shell)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, commandCandidates...)
	case *instructions.CmdCommand:
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine, context.shell)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, commandCandidates...)
	case *instructions.EntrypointCommand:
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine, context.shell)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, commandCandidates...)
	case *instructions.HealthCheckCommand:
		if command.Health != nil && len(command.Health.Test) > 1 {
			commandLine := instructions.ShellDependantCmdLine{
				CmdLine:      command.Health.Test[1:],
				PrependShell: command.Health.Test[0] == "CMD-SHELL",
			}
			commandCandidates, err := discovery.commandCandidates(node, commandLine, context.shell)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, commandCandidates...)
		}
	}
	return candidates, nil
}

func defaultRuntimeShell() runtimeShell {
	return runtimeShell{known: true}
}

func runtimeShellFor(command []string) runtimeShell {
	if len(command) != 2 || command[1] != "-c" {
		return runtimeShell{}
	}
	if isPOSIXShellExecutable(command[0]) {
		return defaultRuntimeShell()
	}
	return runtimeShell{}
}

func isPOSIXShellExecutable(command string) bool {
	return command == "sh" || command == "/bin/sh"
}

func isNonnegativeDecimal(value string) bool {
	index, err := strconv.Atoi(value)
	return err == nil && index >= 0
}

func (discovery *dockerDiscovery) beginStage(stage *instructions.Stage, unsupported *[]dockerCandidate) error {
	baseAlternatives, err := discovery.dockerWordAlternatives(stage.BaseName)
	if err != nil {
		return err
	}
	baseStates := []*dockerStageState{}
	seenBases := map[*dockerStageState]bool{}
	externalPossible := false
	for _, alternative := range baseAlternatives {
		if isNonnegativeDecimal(alternative) {
			externalPossible = true
			continue
		}
		base, local := discovery.baseReferences[alternative]
		if !local {
			externalPossible = true
			continue
		}
		if !seenBases[base] {
			seenBases[base] = true
			baseStates = append(baseStates, base)
		}
	}
	for _, symbolic := range discovery.wordMemo[dockerWordKey{value: stage.BaseName}].symbolic {
		externalPossible = true
		for reference, base := range discovery.baseReferences {
			if symbolicGlobCanEqual(symbolic, reference) && !seenBases[base] {
				seenBases[base] = true
				baseStates = append(baseStates, base)
			}
		}
	}
	shellState := defaultRuntimeShell()
	if !externalPossible && len(baseStates) > 0 {
		shellState = baseStates[0].shell
	}
	for _, base := range baseStates {
		shellState.known = shellState.known && base.shell.known
	}
	index := strconv.Itoa(discovery.stageCount)
	discovery.stageCount++
	current := &dockerStageState{references: []string{index}, baseName: strings.ToLower(stage.Name), shell: shellState}
	if stage.Name != "" {
		current.references = append(current.references, strings.ToLower(stage.Name))
	}
	discovery.currentStage = current
	discovery.allStages = append(discovery.allStages, current)
	for _, base := range baseStates {
		if len(base.triggers) == 0 {
			continue
		}
		context := discovery.currentContext()
		context.shell = base.shell
		found, resultingShell, err := discovery.evaluateDeferred(base.triggers, context)
		if err != nil {
			return err
		}
		current.shell.known = current.shell.known && resultingShell.known
		*unsupported = append(*unsupported, found...)
		for _, trigger := range base.triggers {
			trigger.consumed = true
		}
	}
	return nil
}

func (discovery *dockerDiscovery) completeCurrentStage() error {
	if discovery.currentStage == nil {
		return nil
	}
	if len(discovery.currentStage.triggers) > 0 {
		context := dockerInstructionContext{shell: discovery.currentStage.shell, conservative: true}
		candidates, _, err := discovery.evaluateDeferred(discovery.currentStage.triggers, context)
		if err != nil {
			return err
		}
		discovery.currentStage.provisional = candidates
	}
	for _, reference := range discovery.currentStage.references {
		discovery.stageReferences[reference] = discovery.currentStage
	}
	if discovery.currentStage.baseName != "" {
		discovery.baseReferences[discovery.currentStage.baseName] = discovery.currentStage
	}
	discovery.currentStage = nil
	return nil
}

func (discovery *dockerDiscovery) setRuntimeShell(command []string) {
	discovery.currentStage.shell = runtimeShellFor(command)
}

func (discovery *dockerDiscovery) isStageReference(value string) bool {
	_, found := discovery.stageReferences[strings.ToLower(value)]
	return found
}

func (discovery *dockerDiscovery) isBaseReference(value string) bool {
	_, found := discovery.baseReferences[value]
	return found
}

func (context dockerInstructionContext) isCurrentStageReference(value string) bool {
	value = strings.ToLower(value)
	for _, reference := range context.currentReferences {
		if value == reference {
			return true
		}
	}
	return false
}

func (discovery *dockerDiscovery) currentContext() dockerInstructionContext {
	if discovery.currentStage == nil {
		return dockerInstructionContext{shell: defaultRuntimeShell()}
	}
	return dockerInstructionContext{
		shell:             discovery.currentStage.shell,
		currentReferences: discovery.currentStage.references,
	}
}

func (discovery *dockerDiscovery) unconsumedDeferredCandidates() []dockerCandidate {
	var candidates []dockerCandidate
	for _, stage := range discovery.allStages {
		unconsumed := false
		for _, trigger := range stage.triggers {
			unconsumed = unconsumed || !trigger.consumed
		}
		if unconsumed {
			candidates = append(candidates, stage.provisional...)
		}
	}
	return candidates
}

func (discovery *dockerDiscovery) normalizeDockerWord(value string) (string, error) {
	_, err := discovery.dockerWordAlternatives(value)
	if err != nil {
		return "", err
	}
	return discovery.wordMemo[dockerWordKey{value: value}].validation, nil
}

func (discovery *dockerDiscovery) dockerWordAlternatives(value string) ([]string, error) {
	key := dockerWordKey{value: value}
	if result, found := discovery.wordMemo[key]; found {
		return concreteSymbolicValues(result.values), result.err
	}
	if len(value) > maxDockerWordInputBytes {
		err := resourceLimitf("Docker word exceeds the %d-byte normalization input limit", maxDockerWordInputBytes)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	discovery.wordWorkBytes += len(value)
	if discovery.wordWorkBytes > maxDockerWordWorkBytes {
		err := resourceLimitf("Docker metadata exceeds the %d-byte word-normalization work limit", maxDockerWordWorkBytes)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	discovery.wordLexer.SkipUnsetEnv = true
	probe, err := discovery.wordLexer.ProcessWordWithMatches(value, shell.EnvsFromSlice(nil))
	if err != nil {
		err = fmt.Errorf("normalize Docker word %q: %w", value, err)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	if strings.ContainsRune(probe.Result, '\x00') {
		err := fmt.Errorf("normalize Docker word %q: NUL is not supported", value)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	variables := make([]string, 0, len(probe.Unmatched))
	for name := range probe.Unmatched {
		variables = append(variables, name)
	}
	sort.Strings(variables)
	if len(variables) > maxDockerWordVariables {
		err := resourceLimitf("Docker word references more than %d variables", maxDockerWordVariables)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	unknownNames := make(map[string]bool, len(probe.Unmatched))
	for name := range probe.Unmatched {
		unknownNames[name] = true
	}
	if _, unsupported := unsupportedDockerWordOperator(value, unknownNames, discovery.escapeToken); unsupported {
		// Content-sensitive operators are outside the contract. Do not feed the
		// private identity framing through an operator that can rewrite it; typed
		// validation will emit the policy diagnostic immediately after this call.
		if err := discovery.accountNormalizedBytes(len(probe.Result)); err != nil {
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
		concrete := symbolicValue{segments: []symbolicSegment{{literal: probe.Result}}}
		discovery.wordMemo[key] = dockerWordResult{
			validation: probe.Result,
			values:     []symbolicValue{concrete},
			variables:  unknownNames,
			unknown:    len(probe.Unmatched) > 0,
		}
		return []string{probe.Result}, nil
	}
	combinations := 1
	for range variables {
		combinations *= 3
	}
	if combinations > maxDockerWordAlternatives {
		err := resourceLimitf("Docker word exceeds the %d-alternative expansion limit", maxDockerWordAlternatives)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	discovery.wordLexer.SkipUnsetEnv = false
	sentinels := make([]string, len(variables))
	for index := range variables {
		sentinels[index] = fmt.Sprintf("\x00D%d\x00", index)
	}
	alternatives := make([]string, 0, combinations)
	symbolic := make([]symbolicValue, 0, combinations)
	seenAlternatives := map[string]bool{}
	seenSymbolic := map[string]bool{}
	var firstExpansionError error
	for combination := 0; combination < combinations; combination++ {
		discovery.alternativeWork += len(value)
		if discovery.alternativeWork > maxDockerAlternativeWork {
			err := resourceLimitf("Docker metadata exceeds the %d-byte alternative-expansion work limit", maxDockerAlternativeWork)
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
		state := combination
		environment := make([]string, 0, len(variables))
		for index, name := range variables {
			switch state % 3 {
			case 1:
				environment = append(environment, name+"=")
			case 2:
				environment = append(environment, name+"="+sentinels[index])
			}
			state /= 3
		}
		normalizedSymbolic, _, normalizeErr := discovery.wordLexer.ProcessWord(value, shell.EnvsFromSlice(environment))
		if normalizeErr != nil {
			if firstExpansionError == nil {
				firstExpansionError = normalizeErr
			}
			continue
		}
		normalized := normalizedSymbolic
		for _, sentinel := range sentinels {
			normalized = strings.ReplaceAll(normalized, sentinel, "x")
		}
		if err := discovery.accountNormalizedBytes(len(normalizedSymbolic)); err != nil {
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
		if !seenAlternatives[normalized] {
			seenAlternatives[normalized] = true
			alternatives = append(alternatives, normalized)
		}
		typedSymbolic, parseErr := parseSymbolicValue(normalizedSymbolic)
		if parseErr != nil {
			discovery.wordMemo[key] = dockerWordResult{err: parseErr}
			return nil, parseErr
		}
		if typedSymbolic.hasVariables() && !seenSymbolic[normalizedSymbolic] {
			seenSymbolic[normalizedSymbolic] = true
			symbolic = append(symbolic, typedSymbolic)
		}
	}
	if len(alternatives) == 0 {
		err := fmt.Errorf("normalize Docker word %q: %w", value, firstExpansionError)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	discovery.wordMemo[key] = dockerWordResult{
		validation: probe.Result,
		values:     typedLiteralValues(alternatives),
		symbolic:   symbolic,
		variables:  unknownNames,
		unknown:    len(probe.Unmatched) > 0,
	}
	return alternatives, nil
}

func typedLiteralValues(values []string) []symbolicValue {
	typed := make([]symbolicValue, 0, len(values))
	for _, value := range values {
		typed = append(typed, symbolicValue{segments: []symbolicSegment{{literal: value}}})
	}
	return typed
}

func concreteSymbolicValues(values []symbolicValue) []string {
	concrete := make([]string, 0, len(values))
	for _, value := range values {
		var literal strings.Builder
		for _, segment := range value.segments {
			if segment.variable != "" {
				panic("internal symbolic alternative is not concrete")
			}
			literal.WriteString(segment.literal)
		}
		concrete = append(concrete, literal.String())
	}
	return concrete
}

func (discovery *dockerDiscovery) dockerWordHasUnknown(value string) bool {
	return discovery.wordMemo[dockerWordKey{value: value}].unknown
}

func symbolicMarkerAt(value string, index int) (string, int, bool) {
	if index >= len(value) || value[index] != '\x00' {
		return "", 0, false
	}
	relativeEnd := strings.IndexByte(value[index+1:], '\x00')
	if relativeEnd < 0 {
		return "", 0, false
	}
	end := index + 1 + relativeEnd
	identity := value[index+1 : end]
	if len(identity) < 2 || identity[0] != 'D' && identity[0] != 'R' {
		return "", 0, false
	}
	for _, digit := range identity[1:] {
		if digit < '0' || digit > '9' {
			return "", 0, false
		}
	}
	return identity, end - index + 1, true
}

func parseSymbolicValue(encoded string) (symbolicValue, error) {
	segments := []symbolicSegment{}
	literalStart := 0
	appendLiteral := func(end int) {
		if literalStart < end {
			segments = append(segments, symbolicSegment{literal: encoded[literalStart:end]})
		}
	}
	for index := 0; index < len(encoded); {
		if encoded[index] != '\x00' {
			index++
			continue
		}
		identity, width, found := symbolicMarkerAt(encoded, index)
		if !found {
			return symbolicValue{}, fmt.Errorf("malformed internal symbolic value")
		}
		appendLiteral(index)
		segments = append(segments, symbolicSegment{variable: identity})
		index += width
		literalStart = index
	}
	appendLiteral(len(encoded))
	return symbolicValue{segments: segments}, nil
}

func (value symbolicValue) hasVariables() bool {
	for _, segment := range value.segments {
		if segment.variable != "" {
			return true
		}
	}
	return false
}

func (value symbolicValue) onlyVariables() bool {
	for _, segment := range value.segments {
		if segment.literal != "" {
			return false
		}
	}
	return true
}

type symbolicAtom struct {
	literal  byte
	variable string
}

func (value symbolicValue) atoms() []symbolicAtom {
	atoms := []symbolicAtom{}
	for _, segment := range value.segments {
		if segment.variable != "" {
			atoms = append(atoms, symbolicAtom{variable: segment.variable})
			continue
		}
		for index := 0; index < len(segment.literal); index++ {
			atoms = append(atoms, symbolicAtom{literal: segment.literal[index]})
		}
	}
	return atoms
}

func (discovery *dockerDiscovery) dockerWordUnknownMayContainSource(value string, downloadOnly bool) bool {
	if !discovery.dockerWordHasUnknown(value) {
		return false
	}
	for _, symbolic := range discovery.wordMemo[dockerWordKey{value: value}].symbolic {
		if symbolicDownloadPrefixPossible(symbolic) || !downloadOnly && symbolicGolangImagePossible(symbolic) {
			return true
		}
	}
	return false
}

func symbolicDownloadPrefixPossible(symbolic symbolicValue) bool {
	if symbolic.onlyVariables() {
		return false
	}
	for _, prefix := range []string{"https://go.dev/dl/go", "https://dl.google.com/go/go"} {
		if symbolicGlobCanStartWith(symbolic, prefix) {
			return true
		}
	}
	return false
}

func symbolicGlobCanStartWith(pattern symbolicValue, prefix string) bool {
	atoms := pattern.atoms()
	var visit func(int, int, bool, map[string]string) bool
	visit = func(patternIndex, prefixIndex int, matchedFixed bool, assignments map[string]string) bool {
		if prefixIndex == len(prefix) {
			return matchedFixed
		}
		if patternIndex == len(atoms) {
			return false
		}
		atom := atoms[patternIndex]
		if atom.variable != "" {
			marker := atom.variable
			if assigned, found := assignments[marker]; found {
				return strings.HasPrefix(prefix[prefixIndex:], assigned) &&
					visit(patternIndex+1, prefixIndex+len(assigned), matchedFixed, assignments)
			}
			for end := prefixIndex; end <= len(prefix); end++ {
				branch := make(map[string]string, len(assignments)+1)
				for identity, value := range assignments {
					branch[identity] = value
				}
				branch[marker] = prefix[prefixIndex:end]
				if visit(patternIndex+1, end, matchedFixed, branch) {
					return true
				}
			}
			return false
		}
		return strings.EqualFold(string(atom.literal), prefix[prefixIndex:prefixIndex+1]) && visit(patternIndex+1, prefixIndex+1, true, assignments)
	}
	return visit(0, 0, false, map[string]string{})
}

func symbolicGolangImagePossible(symbolic symbolicValue) bool {
	atoms := symbolic.atoms()
	dockerPrefix := "docker://"
	if len(atoms) >= len(dockerPrefix) {
		matched := true
		for index := range dockerPrefix {
			matched = matched && atoms[index].variable == "" && strings.EqualFold(string(atoms[index].literal), dockerPrefix[index:index+1])
		}
		if matched {
			atoms = atoms[len(dockerPrefix):]
		}
	}
	if len(atoms) == 0 {
		return false
	}
	onlyVariables := true
	for _, atom := range atoms {
		onlyVariables = onlyVariables && atom.variable != ""
	}
	if onlyVariables {
		return false
	}
	starts := []int{0}
	ends := []int{len(atoms)}
	inSuffix := false
	portCompatible := false
	type injectedStart struct {
		index  int
		prefix string
	}
	injectedStarts := []injectedStart{}
	for index, atom := range atoms {
		if atom.variable != "" {
			// An expansion can introduce a slash and start a new repository
			// component. After a colon that is only possible when the text
			// since the colon can still be a registry port.
			if !inSuffix {
				starts = append(starts, index)
			} else if portCompatible {
				injectedStarts = append(injectedStarts, injectedStart{index: index, prefix: "0/"})
			}
			ends = append(ends, index+1)
			continue
		}
		switch atom.literal {
		case '/':
			starts = append(starts, index+1)
			inSuffix = false
			portCompatible = false
		case ':', '@':
			if !inSuffix {
				ends = append(ends, index)
				inSuffix = true
				portCompatible = atom.literal == ':'
			}
		default:
			if inSuffix && (atom.literal < '0' || atom.literal > '9') {
				portCompatible = false
			}
		}
	}
	for _, start := range starts {
		for _, end := range ends {
			if start < end && symbolicGlobCanEqualAnchored(symbolicValueFromAtoms(atoms[start:end]), "golang") {
				return true
			}
		}
	}
	for _, start := range injectedStarts {
		// The injected slash starts a fresh repository component, so delimiters
		// after it must be interpreted independently of the earlier colon that
		// may have become a registry port.
		injectedEnds := []int{len(atoms)}
		for index := start.index + 1; index < len(atoms); index++ {
			if atoms[index].variable == "" && (atoms[index].literal == ':' || atoms[index].literal == '@') {
				injectedEnds = append(injectedEnds, index)
			}
		}
		for _, end := range injectedEnds {
			if start.index < end && symbolicGlobCanEqualAfterInjectedSlash(
				symbolicValueFromAtoms(atoms[start.index:end]), "golang", start.prefix,
			) {
				return true
			}
		}
	}
	return false
}

// symbolicGlobCanEqualAfterInjectedSlash checks a component that begins inside
// the first variable. injectedPrefix is the witness text before that component
// (a slash, optionally preceded by a registry port). Keeping the full assignment
// in the map preserves repeated-variable identity at later occurrences.
func symbolicGlobCanEqualAfterInjectedSlash(pattern symbolicValue, target, injectedPrefix string) bool {
	atoms := pattern.atoms()
	if len(atoms) == 0 || atoms[0].variable == "" {
		return false
	}
	// Keep the bounded policy's pure-variable exclusion: an unknown may
	// introduce the registry-port/path boundary, but some literal source text
	// must still anchor the reserved final component.
	hasLiteral := false
	for _, atom := range atoms {
		hasLiteral = hasLiteral || atom.variable == ""
	}
	if !hasLiteral {
		return false
	}
	first := atoms[0].variable
	for end := 0; end <= len(target); end++ {
		assignment := injectedPrefix + target[:end]
		assignments := map[string]string{first: assignment}
		if symbolicAtomsCanEqual(atoms[1:], target[end:], assignments) {
			return true
		}
	}
	return false
}

func symbolicAtomsCanEqual(atoms []symbolicAtom, target string, assignments map[string]string) bool {
	if len(atoms) == 0 {
		return target == ""
	}
	atom := atoms[0]
	if atom.variable != "" {
		assigned, found := assignments[atom.variable]
		if found {
			return strings.HasPrefix(target, assigned) &&
				symbolicAtomsCanEqual(atoms[1:], target[len(assigned):], assignments)
		}
		for end := 0; end <= len(target); end++ {
			branch := make(map[string]string, len(assignments)+1)
			for identity, value := range assignments {
				branch[identity] = value
			}
			branch[atom.variable] = target[:end]
			if symbolicAtomsCanEqual(atoms[1:], target[end:], branch) {
				return true
			}
		}
		return false
	}
	return len(target) > 0 && strings.EqualFold(string(atom.literal), target[:1]) &&
		symbolicAtomsCanEqual(atoms[1:], target[1:], assignments)
}

func symbolicValueFromAtoms(atoms []symbolicAtom) symbolicValue {
	segments := []symbolicSegment{}
	for _, atom := range atoms {
		if atom.variable != "" {
			segments = append(segments, symbolicSegment{variable: atom.variable})
			continue
		}
		if len(segments) > 0 && segments[len(segments)-1].variable == "" {
			segments[len(segments)-1].literal += string(atom.literal)
		} else {
			segments = append(segments, symbolicSegment{literal: string(atom.literal)})
		}
	}
	return symbolicValue{segments: segments}
}

func symbolicGlobCanEqual(pattern symbolicValue, target string) bool {
	return symbolicGlobCanEqualState(pattern, target, false, false)
}

func symbolicGlobCanEqualAnchored(pattern symbolicValue, target string) bool {
	return symbolicGlobCanEqualState(pattern, target, true, true)
}

func symbolicGlobCanEqualState(pattern symbolicValue, target string, requireFixed, foldCase bool) bool {
	atoms := pattern.atoms()
	var visit func(int, int, bool, map[string]string) bool
	visit = func(patternIndex, targetIndex int, matchedFixed bool, assignments map[string]string) bool {
		if patternIndex == len(atoms) {
			return targetIndex == len(target) && (!requireFixed || matchedFixed)
		}
		atom := atoms[patternIndex]
		if atom.variable != "" {
			marker := atom.variable
			if assigned, found := assignments[marker]; found {
				return strings.HasPrefix(target[targetIndex:], assigned) &&
					visit(patternIndex+1, targetIndex+len(assigned), matchedFixed, assignments)
			}
			for end := targetIndex; end <= len(target); end++ {
				branch := make(map[string]string, len(assignments)+1)
				for identity, value := range assignments {
					branch[identity] = value
				}
				branch[marker] = target[targetIndex:end]
				if visit(patternIndex+1, end, matchedFixed, branch) {
					return true
				}
			}
			return false
		}
		matches := targetIndex < len(target) && atom.literal == target[targetIndex]
		if foldCase && targetIndex < len(target) {
			matches = strings.EqualFold(string(atom.literal), target[targetIndex:targetIndex+1])
		}
		return matches && visit(patternIndex+1, targetIndex+1, true, assignments)
	}
	return visit(0, 0, false, map[string]string{})
}

func (discovery *dockerDiscovery) accountNormalizedBytes(count int) error {
	discovery.normalizedBytes += count
	if discovery.normalizedBytes > maxDockerNormalizedBytes {
		return resourceLimitf("Docker metadata exceeds the %d-byte normalized-word limit", maxDockerNormalizedBytes)
	}
	return nil
}

func (discovery *dockerDiscovery) commandCandidates(node *parser.Node, command instructions.ShellDependantCmdLine, runtime runtimeShell) ([]dockerCandidate, error) {
	words := command.CmdLine
	analyzed := []runtimeWord{}
	if command.PrependShell {
		script := dockerShellScript(command)
		// NUL frames are reserved exclusively for the internal symbolic-word
		// serialization. Reject them before shell selection so an unknown runtime
		// cannot turn malformed input into an unsupported-shell classification.
		if strings.ContainsRune(script, '\x00') {
			return nil, fmt.Errorf("runtime shell contains unsupported NUL byte")
		}
		if !runtime.known {
			return []dockerCandidate{{Kind: "unsupported-shell", Value: node.Original, Line: node.StartLine}}, nil
		}
		var err error
		analyzed, err = discovery.runtimeWords(script, nil)
		if err != nil {
			var limitError *resourceLimitError
			if errors.As(err, &limitError) {
				return nil, err
			}
			return []dockerCandidate{{Kind: "unsupported-shell", Value: node.Original, Line: node.StartLine}}, nil
		}
	} else {
		length := 0
		for _, word := range words {
			length += len(word)
		}
		if err := discovery.accountNormalizedBytes(length); err != nil {
			return nil, err
		}
		if len(words) >= 3 && isPOSIXShellExecutable(words[0]) && words[1] == "-c" {
			if strings.ContainsRune(words[2], '\x00') {
				return nil, fmt.Errorf("runtime shell contains unsupported NUL byte")
			}
			parameters := []string{words[0]}
			if len(words) > 3 {
				parameters = append([]string{words[3]}, words[4:]...)
			}
			scriptWords, err := discovery.runtimeWords(words[2], parameters)
			if err != nil {
				var limitError *resourceLimitError
				if errors.As(err, &limitError) {
					return nil, err
				}
				return []dockerCandidate{{Kind: "unsupported-shell", Value: node.Original, Line: node.StartLine}}, nil
			}
			analyzed = scriptWords
		} else {
			for _, word := range words {
				analyzed = append(analyzed, runtimeWord{value: word})
			}
		}
	}
	literal, download := scanRuntimeWords(analyzed)
	candidates := []dockerCandidate{}
	if download {
		candidates = append(candidates, dockerCandidate{Kind: "download", Value: node.Original, Line: node.StartLine})
	}
	if literal {
		candidates = append(candidates, dockerCandidate{Kind: "literal", Value: node.Original, Line: node.StartLine})
	}
	return candidates, nil
}

func dockerShellScript(command instructions.ShellDependantCmdLine) string {
	script := strings.Join(command.CmdLine, " ")
	if len(command.Files) == 1 && len(command.CmdLine) == 1 && parser.MustParseHeredoc(command.CmdLine[0]) != nil {
		return command.Files[0].Data
	}
	for _, file := range command.Files {
		script += "\n" + file.Data
		if !strings.HasSuffix(file.Data, "\n") {
			script += "\n"
		}
		script += file.Name + "\n"
	}
	return script
}

func (discovery *dockerDiscovery) runtimeWords(value string, parameters []string) ([]runtimeWord, error) {
	key := runtimeWordsKey{value: value, parameters: strings.Join(parameters, "\x00")}
	if result, found := discovery.runtimeWordsMemo[key]; found {
		return result.values, result.err
	}
	words, err := parseRuntimeShellWordsWithBudget(value, parameters, discovery.accountNormalizedBytes)
	discovery.runtimeWordsMemo[key] = runtimeWordsResult{values: words, err: err}
	return words, err
}

func scanRuntimeWords(words []runtimeWord) (bool, bool) {
	literal := false
	download := false
	for _, word := range words {
		if word.symbolic != nil {
			literal = literal || symbolicGolangImagePossible(*word.symbolic)
			download = download || symbolicDownloadPrefixPossible(*word.symbolic)
		} else {
			literal = literal || containsDockerGoToken(word.value)
			download = download || goDownloadPattern.MatchString(word.value)
		}
	}
	return literal, download
}

func parseRuntimeShellWords(value string, parameters []string) ([]runtimeWord, error) {
	return parseRuntimeShellWordsWithBudget(value, parameters, func(int) error { return nil })
}

func parseRuntimeShellWordsWithBudget(value string, parameters []string, account func(int) error) ([]runtimeWord, error) {
	if strings.ContainsRune(value, '\x00') {
		return nil, fmt.Errorf("runtime shell contains unsupported NUL byte")
	}
	parsed, err := shsyntax.NewParser(shsyntax.Variant(shsyntax.LangPOSIX)).Parse(strings.NewReader(value), "runtime-shell")
	if err != nil {
		return nil, fmt.Errorf("parse POSIX runtime shell: %w", err)
	}
	words := []*shsyntax.Word{}
	excludedWords := map[*shsyntax.Word]bool{}
	ordinaryVariables := map[string]bool{}
	patternExpansions := []*shsyntax.ParamExp{}
	nodes := 0
	depth := 0
	var walkErr error
	shsyntax.Walk(parsed, func(node shsyntax.Node) bool {
		if node == nil {
			depth--
			return true
		}
		if walkErr != nil {
			return false
		}
		nodes++
		depth++
		if nodes > maxShellASTNodes {
			depth--
			walkErr = resourceLimitf("runtime shell exceeds the %d-node parsed-work limit", maxShellASTNodes)
			return false
		}
		if depth > maxShellASTDepth {
			depth--
			walkErr = resourceLimitf("runtime shell exceeds the %d-level shell AST depth limit", maxShellASTDepth)
			return false
		}
		if arithmetic, ok := node.(*shsyntax.ArithmExp); ok {
			markArithmeticIdentifierWords(arithmetic.X, excludedWords)
		}
		if word, ok := node.(*shsyntax.Word); ok {
			words = append(words, word)
		}
		if parameter, ok := node.(*shsyntax.ParamExp); ok && parameter.Param != nil {
			name := parameter.Param.Value
			if !isRuntimeSpecialParameter(name) {
				ordinaryVariables[name] = true
			}
			if parameter.Exp != nil {
				switch parameter.Exp.Op {
				case shsyntax.RemSmallPrefix, shsyntax.RemLargePrefix, shsyntax.RemSmallSuffix, shsyntax.RemLargeSuffix:
					patternExpansions = append(patternExpansions, parameter)
				}
			}
		}
		if redirect, ok := node.(*shsyntax.Redirect); ok && (redirect.Op == shsyntax.Hdoc || redirect.Op == shsyntax.DashHdoc) {
			excludedWords[redirect.Word] = true
		}
		return true
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if len(patternExpansions) > 0 {
		rewritten, err := rewriteRuntimePatternExpansions(value, patternExpansions)
		if err != nil {
			return nil, err
		}
		return parseRuntimeShellWordsWithBudget(rewritten, parameters, account)
	}

	variables := make([]string, 0, len(ordinaryVariables))
	for name := range ordinaryVariables {
		variables = append(variables, name)
	}
	sort.Strings(variables)
	sentinels := make([]string, len(variables))
	for index := range variables {
		sentinels[index] = fmt.Sprintf("\x00R%d\x00", index)
	}

	result := []runtimeWord{}
	seen := map[string]bool{}
	expandEnvironment := func(ordinary map[string]expand.Variable) error {
		for _, lastBackgroundPID := range []string{"", "1"} {
			positionals := []string{}
			zero := "sh"
			if len(parameters) > 0 {
				zero = parameters[0]
				positionals = append(positionals, parameters[1:]...)
			}
			indexed := expand.Variable{Set: true, Kind: expand.Indexed, List: positionals}
			values := map[string]expand.Variable{
				"0":    {Set: true, Kind: expand.String, Str: zero},
				"@":    indexed,
				"*":    indexed,
				"#":    {Set: true, Kind: expand.String, Str: strconv.Itoa(len(positionals))},
				"?":    {Set: true, Kind: expand.String, Str: "0"},
				"$":    {Set: true, Kind: expand.String, Str: "1"},
				"!":    {Set: true, Kind: expand.String, Str: lastBackgroundPID},
				"-":    {Set: true, Kind: expand.String, Str: "s"},
				"PPID": {Set: true, Kind: expand.String, Str: "1"},
			}
			for name, variable := range ordinary {
				values[name] = variable
			}
			environment := &runtimeShellEnvironment{values: values}
			for index, parameter := range positionals {
				environment.values[strconv.Itoa(index+1)] = expand.Variable{Set: true, Kind: expand.String, Str: parameter}
			}
			config := &expand.Config{
				Env: environment,
				CmdSubst: func(io.Writer, *shsyntax.CmdSubst) error {
					return nil
				},
			}
			for _, word := range words {
				if excludedWords[word] {
					continue
				}
				span := int(word.End().Offset() - word.Pos().Offset())
				if span < 1 {
					span = 1
				}
				if err := account(span); err != nil {
					return err
				}
				fields, expandErr := expand.Fields(config, word)
				if expandErr != nil {
					return fmt.Errorf("expand POSIX runtime shell word: %w", expandErr)
				}
				for _, field := range fields {
					if err := account(len(field)); err != nil {
						return err
					}
					if !seen[field] {
						seen[field] = true
						word := runtimeWord{value: field}
						symbolic, symbolicErr := parseSymbolicValue(field)
						if symbolicErr != nil {
							return symbolicErr
						}
						if symbolic.hasVariables() {
							word.value = ""
							word.symbolic = &symbolic
						}
						result = append(result, word)
					}
				}
			}
		}
		return nil
	}
	states := make([]uint8, len(variables))
	for {
		ordinary := map[string]expand.Variable{}
		for index, state := range states {
			switch state {
			case 1:
				ordinary[variables[index]] = expand.Variable{Set: true, Exported: true, Kind: expand.String, Str: ""}
			case 2:
				ordinary[variables[index]] = expand.Variable{Set: true, Exported: true, Kind: expand.String, Str: sentinels[index]}
			}
		}
		if err := expandEnvironment(ordinary); err != nil {
			return nil, err
		}
		index := 0
		for index < len(states) {
			states[index]++
			if states[index] < 3 {
				break
			}
			states[index] = 0
			index++
		}
		if index == len(states) {
			break
		}
	}
	return result, nil
}

func rewriteRuntimePatternExpansions(value string, parameters []*shsyntax.ParamExp) (string, error) {
	type span struct{ start, end int }
	spans := []span{}
	for _, parameter := range parameters {
		start := int(parameter.Pos().Offset())
		end := int(parameter.End().Offset())
		if start < 0 || end > len(value) || start >= end {
			return "", fmt.Errorf("runtime-shell parameter span is outside its source")
		}
		spans = append(spans, span{start: start, end: end})
	}
	sort.Slice(spans, func(left, right int) bool { return spans[left].start < spans[right].start })
	prefix := "__KFP_PATTERN_RESULT_"
	for strings.Contains(value, prefix) {
		prefix += "_"
	}
	var rewritten strings.Builder
	cursor := 0
	identities := map[string]int{}
	for _, current := range spans {
		if current.start < cursor {
			continue
		}
		rewritten.WriteString(value[cursor:current.start])
		expression := value[current.start:current.end]
		identity, found := identities[expression]
		if !found {
			identity = len(identities)
			identities[expression] = identity
		}
		fmt.Fprintf(&rewritten, "${%s%d}", prefix, identity)
		cursor = current.end
	}
	if len(identities) == 0 {
		return "", fmt.Errorf("runtime-shell pattern expansion rewrite made no progress")
	}
	rewritten.WriteString(value[cursor:])
	return rewritten.String(), nil
}

func isRuntimeSpecialParameter(name string) bool {
	if name == "@" || name == "*" || name == "#" || name == "?" || name == "$" || name == "!" || name == "-" || name == "PPID" {
		return true
	}
	_, err := strconv.Atoi(name)
	return err == nil
}

func markArithmeticIdentifierWords(expression shsyntax.ArithmExpr, excluded map[*shsyntax.Word]bool) {
	shsyntax.Walk(expression, func(node shsyntax.Node) bool {
		if node == nil {
			return true
		}
		if _, commandSubstitution := node.(*shsyntax.CmdSubst); commandSubstitution {
			return false
		}
		if word, ok := node.(*shsyntax.Word); ok {
			excluded[word] = true
		}
		return true
	})
}

type runtimeShellEnvironment struct {
	fallback string
	values   map[string]expand.Variable
}

func (environment *runtimeShellEnvironment) Get(name string) expand.Variable {
	if value, found := environment.values[name]; found {
		return value
	}
	if environment.fallback == "" {
		return expand.Variable{}
	}
	return expand.Variable{Set: true, Exported: true, Kind: expand.String, Str: environment.fallback}
}

func (environment *runtimeShellEnvironment) Each(yield func(string, expand.Variable) bool) {
	for name, value := range environment.values {
		if !yield(name, value) {
			return
		}
	}
}

func (environment *runtimeShellEnvironment) Set(name string, value expand.Variable) error {
	environment.values[name] = value
	return nil
}

func containsDockerGoToken(value string) bool {
	found, _ := scanDockerGoToken(value)
	return found
}

func scanDockerGoToken(value string) (bool, int) {
	steps := 0
	for index := 0; index < len(value); index++ {
		steps++
		if value[index] == '$' && index+1 < len(value) {
			if value[index+1] == '{' {
				if end, inspected, ok := prefixedParameterNameEnd(value, index+2); ok {
					steps += inspected
					index = end
					continue
				}
				wordStart, nameEnd, inspected, ok := parameterExpansionWordStart(value, index+2)
				steps += inspected
				if ok && hasDockerGoTokenAt(value, wordStart) && hasDockerGoTokenEnd(value, wordStart) {
					return true, steps
				}
				if nameEnd > index+2 {
					index = nameEnd - 1
					continue
				}
			} else {
				nameEnd, inspected, ok := shellParameterNameEnd(value, index+1)
				steps += inspected
				if ok {
					index = nameEnd - 1
					continue
				}
			}
		}
		if !hasDockerGoTokenAt(value, index) {
			continue
		}
		if (index == 0 || !isDockerNameByte(value[index-1])) && hasDockerGoTokenEnd(value, index) {
			return true, steps
		}
		index += len("golang") - 1
	}
	return false, steps
}

func prefixedParameterNameEnd(value string, start int) (int, int, bool) {
	if start >= len(value) || value[start] != '#' && value[start] != '!' {
		return start, 0, false
	}
	nameEnd, inspected, ok := shellParameterNameEnd(value, start+1)
	inspected++
	if !ok || nameEnd >= len(value) || value[nameEnd] != '}' {
		return start, inspected, false
	}
	return nameEnd, inspected + 1, true
}

func parameterExpansionWordStart(value string, start int) (int, int, int, bool) {
	cursor, inspected, ok := shellParameterNameEnd(value, start)
	if !ok {
		return 0, cursor, inspected, false
	}
	nameEnd := cursor
	nullIsUnset := false
	if cursor < len(value) && value[cursor] == ':' {
		cursor++
		inspected++
		nullIsUnset = true
	}
	if cursor >= len(value) || !isShellParameterOperator(value[cursor], nullIsUnset) {
		return 0, nameEnd, inspected + 1, false
	}
	return cursor + 1, nameEnd, inspected + 1, true
}

// unsupportedDockerWordOperator identifies parameter operators whose result
// depends on value contents rather than only setness/emptiness. The bounded
// policy deliberately supports direct substitution and -, :-, +, and :+;
// accepted Docker words using other operators fail closed as unsupported.
func unsupportedDockerWordOperator(value string, activeVariables map[string]bool, escapeToken rune) (string, bool) {
	quote := byte(0)
	for index := 0; index+2 < len(value); index++ {
		if quote == '\'' {
			if value[index] == quote {
				quote = 0
			}
			continue
		}
		if value[index] == byte(escapeToken) {
			index++
			continue
		}
		if value[index] == '$' && value[index+1] == '$' {
			// Docker's word lexer reduces $$ to a literal dollar. Consume the
			// pair so the second byte cannot be mistaken for an active ${...}.
			index++
			continue
		}
		if value[index] == '\'' || value[index] == '"' {
			switch quote {
			case 0:
				quote = value[index]
			case value[index]:
				quote = 0
			}
			continue
		}
		if value[index] != '$' || value[index+1] != '{' {
			continue
		}
		nameStart := index + 2
		cursor, _, ok := shellParameterNameEnd(value, nameStart)
		if !ok || cursor >= len(value) {
			continue
		}
		if !activeVariables[value[nameStart:cursor]] {
			continue
		}
		operatorStart := cursor
		if value[cursor] == ':' {
			cursor++
			if cursor >= len(value) {
				continue
			}
		}
		switch value[cursor] {
		case '-', '+':
			continue
		case '}', ':':
			continue
		default:
			end := cursor + 1
			if end < len(value) && value[end] == value[cursor] && (value[cursor] == '#' || value[cursor] == '%') {
				end++
			}
			return value[operatorStart:end], true
		}
	}
	return "", false
}

// shellParameterNameEnd mirrors BuildKit shellWord.processName. In particular,
// it uses Unicode letters and digits, while an initial digit consumes only a
// positional-parameter digit run and a special parameter consumes one rune.
func shellParameterNameEnd(value string, start int) (int, int, bool) {
	if start >= len(value) {
		return start, 0, false
	}
	cursor := start
	inspected := 0
	first, firstSize := utf8.DecodeRuneInString(value[cursor:])
	switch {
	case unicode.IsDigit(first):
		for cursor < len(value) {
			character, size := utf8.DecodeRuneInString(value[cursor:])
			if !unicode.IsDigit(character) {
				break
			}
			cursor += size
			inspected += size
		}
	case isShellSpecialParameter(first):
		cursor += firstSize
		inspected += firstSize
	default:
		for cursor < len(value) {
			character, size := utf8.DecodeRuneInString(value[cursor:])
			if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '_' {
				break
			}
			cursor += size
			inspected += size
		}
	}
	return cursor, inspected, cursor > start
}

func hasDockerGoTokenAt(value string, start int) bool {
	const token = "golang"
	if start < 0 || start+len(token) > len(value) {
		return false
	}
	for offset := range len(token) {
		character := value[start+offset]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		if character != token[offset] {
			return false
		}
	}
	return true
}

func hasDockerGoTokenEnd(value string, start int) bool {
	end := start + len("golang")
	return end == len(value) || !isDockerNameByte(value[end]) && value[end] != '/'
}

func isShellSpecialParameter(value rune) bool {
	switch value {
	case '@', '*', '#', '?', '-', '$', '!', '0':
		return true
	default:
		return false
	}
}

func isShellParameterOperator(value byte, nullIsUnset bool) bool {
	switch value {
	case '-', '+', '?':
		return true
	case '#', '%':
		return !nullIsUnset
	default:
		return false
	}
}

func isDockerNameByte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '.' || value == '_' || value == '-'
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func isGolangImage(value string) bool {
	image := strings.TrimSpace(value)
	image = strings.TrimPrefix(image, "docker://")
	component := image
	if slash := strings.LastIndex(component, "/"); slash >= 0 {
		component = component[slash+1:]
	}
	if separator := strings.IndexAny(component, ":@"); separator >= 0 {
		component = component[:separator]
	}
	return strings.EqualFold(component, "golang")
}
