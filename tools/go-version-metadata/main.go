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
	maxDockerNormalizedBytes  = maxInputBytes
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
		return "invalid", nil, err.Error()
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

type dockerWordKey struct {
	value string
}

type dockerWordResult struct {
	validation string
	values     []string
	symbolic   string
	unknown    bool
	err        error
}

type runtimeWordsKey struct {
	value      string
	parameters string
}

type runtimeWordsResult struct {
	values []string
	err    error
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
	normalizedBytes  int
	wordWorkBytes    int
	alternativeWork  int
	stageReferences  map[string]*dockerStageState
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
	}
}

func classifyDockerInstruction(node *parser.Node, allowManaged bool, depth int, discovery *dockerDiscovery, managed *[]dockerCandidate, unsupported *[]dockerCandidate) error {
	if depth > maxDockerInstructionDepth {
		return resourceLimitf("Docker metadata exceeds the %d-level instruction depth limit", maxDockerInstructionDepth)
	}
	discovery.instructions++
	if discovery.instructions > maxDockerInstructions {
		return resourceLimitf("Docker metadata exceeds the %d-instruction limit", maxDockerInstructions)
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
		*unsupported = append(*unsupported, candidates...)
	}
	if allowManaged {
		switch command := typed.(type) {
		case *instructions.ShellCommand:
			discovery.setRuntimeShell(command.Shell)
		}
	}
	if len(*managed)+len(*unsupported) > maxDockerCandidates {
		return resourceLimitf("Docker metadata exceeds the %d-candidate limit", maxDockerCandidates)
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
	if err := validateDockerInstructionWords(typed, discovery); err != nil {
		return nil, err
	}
	return typed, nil
}

func validateDockerInstructionWords(typed any, discovery *dockerDiscovery) error {
	validate := func(word string) (string, error) {
		if _, err := discovery.dockerWordAlternatives(word); err != nil {
			return "", err
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
		candidates = append(candidates, found...)
		if command, ok := typed.(*instructions.ShellCommand); ok {
			context.shell = runtimeShellFor(command.Shell)
		}
	}
	return candidates, context.shell, nil
}

func dockerInstructionCandidates(node *parser.Node, typed any, context dockerInstructionContext, discovery *dockerDiscovery) ([]dockerCandidate, error) {
	candidates := []dockerCandidate{}
	appendImageCandidate := func(kind, value string, rejectCurrent bool) error {
		alternatives, err := discovery.dockerWordAlternatives(value)
		if err != nil {
			return err
		}
		matched := false
		allLocal := len(alternatives) > 0
		for _, normalized := range alternatives {
			if rejectCurrent && context.isCurrentStageReference(normalized) {
				return fmt.Errorf("%s cannot reference the current stage %q", kind, normalized)
			}
			if !context.conservative && discovery.isStageReference(normalized) {
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
		if err := appendImageCandidate("from", command.BaseName, false); err != nil {
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
			if err := appendImageCandidate("copy-from", command.From, true); err != nil {
				return nil, err
			}
		}
	case *instructions.RunCommand:
		for _, mount := range instructions.GetMounts(command) {
			if mount.From == "" {
				continue
			}
			if err := appendImageCandidate("run-mount-from", mount.From, true); err != nil {
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

func (discovery *dockerDiscovery) beginStage(stage *instructions.Stage, unsupported *[]dockerCandidate) error {
	if stage.Name != "" {
		if _, duplicate := discovery.stageReferences[strings.ToLower(stage.Name)]; duplicate {
			return fmt.Errorf("duplicate stage name %q", stage.Name)
		}
	}
	base, local := discovery.stageReferences[strings.ToLower(stage.BaseName)]
	shellState := defaultRuntimeShell()
	if local {
		shellState = base.shell
	}
	index := strconv.Itoa(discovery.stageCount)
	discovery.stageCount++
	current := &dockerStageState{references: []string{index}, shell: shellState}
	if stage.Name != "" {
		current.references = append(current.references, strings.ToLower(stage.Name))
	}
	discovery.currentStage = current
	discovery.allStages = append(discovery.allStages, current)
	if local && len(base.triggers) > 0 {
		found, resultingShell, err := discovery.evaluateDeferred(base.triggers, discovery.currentContext())
		if err != nil {
			return err
		}
		current.shell = resultingShell
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
		return result.values, result.err
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
	unknownSentinel := "__KFP_DOCKER_UNKNOWN_7F3A__"
	for strings.Contains(value, unknownSentinel) {
		unknownSentinel += "_"
	}
	symbolicEnvironment := make([]string, 0, len(variables))
	for _, name := range variables {
		symbolicEnvironment = append(symbolicEnvironment, name+"="+unknownSentinel)
	}
	symbolic := probe.Result
	if len(variables) > 0 {
		discovery.alternativeWork += len(value)
		if discovery.alternativeWork > maxDockerAlternativeWork {
			err := resourceLimitf("Docker metadata exceeds the %d-byte alternative-expansion work limit", maxDockerAlternativeWork)
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
		if normalized, _, symbolicErr := discovery.wordLexer.ProcessWord(value, shell.EnvsFromSlice(symbolicEnvironment)); symbolicErr == nil {
			symbolic = strings.ReplaceAll(normalized, unknownSentinel, string(dockerUnknownMarker))
		}
	}
	alternatives := make([]string, 0, combinations)
	seen := map[string]bool{}
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
		for _, name := range variables {
			switch state % 3 {
			case 1:
				environment = append(environment, name+"=")
			case 2:
				environment = append(environment, name+"=x")
			}
			state /= 3
		}
		normalized, _, normalizeErr := discovery.wordLexer.ProcessWord(value, shell.EnvsFromSlice(environment))
		if normalizeErr != nil {
			if firstExpansionError == nil {
				firstExpansionError = normalizeErr
			}
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		alternatives = append(alternatives, normalized)
		if err := discovery.accountNormalizedBytes(len(normalized)); err != nil {
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
	}
	if len(alternatives) == 0 {
		err := fmt.Errorf("normalize Docker word %q: %w", value, firstExpansionError)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	discovery.wordMemo[key] = dockerWordResult{validation: probe.Result, values: alternatives, symbolic: symbolic, unknown: len(probe.Unmatched) > 0}
	return alternatives, nil
}

func (discovery *dockerDiscovery) dockerWordHasUnknown(value string) bool {
	return discovery.wordMemo[dockerWordKey{value: value}].unknown
}

const dockerUnknownMarker = byte(0)

func (discovery *dockerDiscovery) dockerWordUnknownMayContainSource(value string, downloadOnly bool) bool {
	if !discovery.dockerWordHasUnknown(value) {
		return false
	}
	symbolic := discovery.wordMemo[dockerWordKey{value: value}].symbolic
	if symbolicDownloadPrefixPossible(symbolic) {
		return true
	}
	return !downloadOnly && symbolicGolangImagePossible(symbolic)
}

func symbolicDownloadPrefixPossible(symbolic string) bool {
	if strings.Trim(symbolic, string(dockerUnknownMarker)) == "" {
		return false
	}
	for _, prefix := range []string{"https://go.dev/dl/go", "https://dl.google.com/go/go"} {
		if symbolicGlobCanStartWith(symbolic, prefix) {
			return true
		}
	}
	return false
}

func symbolicGlobCanStartWith(pattern, prefix string) bool {
	type state struct {
		pattern, prefix int
		matchedFixed    bool
	}
	seen := map[state]bool{}
	var visit func(int, int, bool) bool
	visit = func(patternIndex, prefixIndex int, matchedFixed bool) bool {
		if prefixIndex == len(prefix) {
			return matchedFixed
		}
		if patternIndex == len(pattern) {
			return false
		}
		current := state{patternIndex, prefixIndex, matchedFixed}
		if seen[current] {
			return false
		}
		seen[current] = true
		if pattern[patternIndex] == dockerUnknownMarker {
			return visit(patternIndex+1, prefixIndex, matchedFixed) || visit(patternIndex, prefixIndex+1, matchedFixed)
		}
		return strings.EqualFold(pattern[patternIndex:patternIndex+1], prefix[prefixIndex:prefixIndex+1]) && visit(patternIndex+1, prefixIndex+1, true)
	}
	return visit(0, 0, false)
}

func symbolicGolangImagePossible(symbolic string) bool {
	transportOffset := 0
	if strings.HasPrefix(strings.ToLower(symbolic), "docker://") {
		transportOffset = len("docker://")
	}
	repository := symbolic[transportOffset:]
	if strings.Trim(repository, string(dockerUnknownMarker)) == "" {
		return false
	}
	if digest := strings.IndexByte(repository, '@'); digest >= 0 {
		if strings.ContainsAny(repository[digest+1:], "/@") {
			return false
		}
		repository = repository[:digest]
	}
	lastSlash := strings.LastIndexByte(repository, '/')
	if tag := strings.LastIndexByte(repository, ':'); tag > lastSlash {
		if strings.ContainsAny(repository[tag+1:], "/:@") {
			return false
		}
		repository = repository[:tag]
	}
	if slash := strings.LastIndexByte(repository, '/'); slash >= 0 {
		repository = repository[slash+1:]
	}
	if strings.Trim(repository, string(dockerUnknownMarker)) == "" {
		return false
	}
	return symbolicGlobCanEqual(repository, "golang")
}

func symbolicGlobCanEqual(pattern, target string) bool {
	type state struct{ pattern, target int }
	seen := map[state]bool{}
	var visit func(int, int) bool
	visit = func(patternIndex, targetIndex int) bool {
		if patternIndex == len(pattern) {
			return targetIndex == len(target)
		}
		current := state{patternIndex, targetIndex}
		if seen[current] {
			return false
		}
		seen[current] = true
		if pattern[patternIndex] == dockerUnknownMarker {
			return visit(patternIndex+1, targetIndex) || targetIndex < len(target) && visit(patternIndex, targetIndex+1)
		}
		return targetIndex < len(target) && strings.EqualFold(pattern[patternIndex:patternIndex+1], target[targetIndex:targetIndex+1]) && visit(patternIndex+1, targetIndex+1)
	}
	return visit(0, 0)
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
	if command.PrependShell {
		if !runtime.known {
			return []dockerCandidate{{Kind: "unsupported-shell", Value: node.Original, Line: node.StartLine}}, nil
		}
		var err error
		script := dockerShellScript(command)
		words, err = discovery.runtimeWords(script, nil)
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
			words = scriptWords
		}
	}
	literal, download := scanRuntimeWords(words)
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

func (discovery *dockerDiscovery) runtimeWords(value string, parameters []string) ([]string, error) {
	key := runtimeWordsKey{value: value, parameters: strings.Join(parameters, "\x00")}
	if result, found := discovery.runtimeWordsMemo[key]; found {
		return result.values, result.err
	}
	words, err := parseRuntimeShellWords(value, parameters)
	if err == nil {
		length := 0
		for _, word := range words {
			length += len(word)
		}
		err = discovery.accountNormalizedBytes(length)
	}
	discovery.runtimeWordsMemo[key] = runtimeWordsResult{values: words, err: err}
	return words, err
}

func scanRuntimeWords(words []string) (bool, bool) {
	literal := false
	download := false
	for _, word := range words {
		literal = literal || containsDockerGoToken(word)
		download = download || goDownloadPattern.MatchString(word)
	}
	return literal, download
}

func parseRuntimeShellWords(value string, parameters []string) ([]string, error) {
	parsed, err := shsyntax.NewParser(shsyntax.Variant(shsyntax.LangPOSIX)).Parse(strings.NewReader(value), "runtime-shell")
	if err != nil {
		return nil, fmt.Errorf("parse POSIX runtime shell: %w", err)
	}
	words := []*shsyntax.Word{}
	excludedWords := map[*shsyntax.Word]bool{}
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
		if redirect, ok := node.(*shsyntax.Redirect); ok && (redirect.Op == shsyntax.Hdoc || redirect.Op == shsyntax.DashHdoc) {
			excludedWords[redirect.Word] = true
		}
		return true
	})
	if walkErr != nil {
		return nil, walkErr
	}

	result := []string{}
	seen := map[string]bool{}
	for _, environmentValue := range []string{"", "x"} {
		positionals := []string{}
		zero := "sh"
		lastBackgroundPID := ""
		if environmentValue != "" {
			lastBackgroundPID = "1"
		}
		if len(parameters) > 0 {
			zero = parameters[0]
			positionals = append(positionals, parameters[1:]...)
		}
		indexed := expand.Variable{Set: true, Kind: expand.Indexed, List: positionals}
		environment := &runtimeShellEnvironment{fallback: environmentValue, values: map[string]expand.Variable{
			"0":    {Set: true, Kind: expand.String, Str: zero},
			"@":    indexed,
			"*":    indexed,
			"#":    {Set: true, Kind: expand.String, Str: strconv.Itoa(len(positionals))},
			"?":    {Set: true, Kind: expand.String, Str: "0"},
			"$":    {Set: true, Kind: expand.String, Str: "1"},
			"!":    {Set: true, Kind: expand.String, Str: lastBackgroundPID},
			"-":    {Set: true, Kind: expand.String, Str: ""},
			"PPID": {Set: true, Kind: expand.String, Str: "1"},
		}}
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
			fields, expandErr := expand.Fields(config, word)
			if expandErr != nil {
				return nil, fmt.Errorf("expand POSIX runtime shell word: %w", expandErr)
			}
			for _, field := range fields {
				if !seen[field] {
					seen[field] = true
					result = append(result, field)
				}
			}
		}
	}
	return result, nil
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
