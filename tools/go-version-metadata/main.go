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
var potentialDockerGoPattern = regexp.MustCompile(`(?i)g[^a-z0-9]*o[^a-z0-9]*l[^a-z0-9]*a[^a-z0-9]*n[^a-z0-9]*g`)
var potentialDockerGoDownloadPattern = regexp.MustCompile(
	`(?i)(?:` +
		`d[^a-z0-9]*l[^a-z0-9]*g[^a-z0-9]*o[^a-z0-9]*o[^a-z0-9]*g[^a-z0-9]*l[^a-z0-9]*e[^a-z0-9]*c[^a-z0-9]*o[^a-z0-9]*m[^a-z0-9]*g[^a-z0-9]*o` +
		`|g[^a-z0-9]*o[^a-z0-9]*d[^a-z0-9]*e[^a-z0-9]*v[^a-z0-9]*d[^a-z0-9]*l[^a-z0-9]*g[^a-z0-9]*o` +
		`)`)
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
	if err := validateDockerStructure(parsed.AST); err != nil {
		return "invalid", nil, err.Error()
	}

	managed := []dockerCandidate{}
	unsupported := []dockerCandidate{}
	discovery := newDockerDiscovery(parsed.EscapeToken)
	for _, node := range parsed.AST.Children {
		if err := classifyDockerInstruction(node, true, false, 0, discovery, &managed, &unsupported); err != nil {
			return "invalid", nil, err.Error()
		}
	}
	if len(unsupported) != 0 || len(managed) > 1 {
		return "unsupported", append(managed, unsupported...), ""
	}
	if len(managed) == 1 {
		return "managed", managed, ""
	}
	return "irrelevant", nil, ""
}

func validateDockerStructure(ast *parser.Node) error {
	hasStage := false
	for _, node := range ast.Children {
		typed, err := parseDockerInstruction(node)
		if err != nil {
			if !isRunMountExpansionError(node, err) {
				return err
			}
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
	return nil
}

func isRunMountExpansionError(node *parser.Node, parseError error) bool {
	return strings.EqualFold(node.Value, "run") && strings.Contains(parseError.Error(), "'from' doesn't support variable expansion")
}

type dockerWordKey struct {
	value string
}

type dockerWordResult struct {
	value string
	err   error
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

type dockerDiscovery struct {
	wordLexer        *shell.Lex
	wordMemo         map[dockerWordKey]dockerWordResult
	runtimeWordsMemo map[runtimeWordsKey]runtimeWordsResult
	instructions     int
	normalizedBytes  int
	stageReferences  map[string]runtimeShell
	currentStage     []string
	currentShell     runtimeShell
	stageCount       int
}

func newDockerDiscovery(escapeToken rune) *dockerDiscovery {
	wordLexer := shell.NewLex(escapeToken)
	wordLexer.SkipUnsetEnv = true
	return &dockerDiscovery{
		wordLexer:        wordLexer,
		wordMemo:         map[dockerWordKey]dockerWordResult{},
		runtimeWordsMemo: map[runtimeWordsKey]runtimeWordsResult{},
		stageReferences:  map[string]runtimeShell{},
		currentShell:     defaultRuntimeShell(),
	}
}

func classifyDockerInstruction(node *parser.Node, allowManaged, deferred bool, depth int, discovery *dockerDiscovery, managed *[]dockerCandidate, unsupported *[]dockerCandidate) error {
	if depth > maxDockerInstructionDepth {
		return resourceLimitf("Docker metadata exceeds the %d-level instruction depth limit", maxDockerInstructionDepth)
	}
	discovery.instructions++
	if discovery.instructions > maxDockerInstructions {
		return resourceLimitf("Docker metadata exceeds the %d-instruction limit", maxDockerInstructions)
	}
	typed, err := parseDockerInstruction(node)
	if err != nil {
		fallback, handled, fallbackErr := fallbackDockerInstructionCandidates(node, discovery, err)
		if fallbackErr != nil {
			return fallbackErr
		}
		if !handled {
			return err
		}
		*unsupported = append(*unsupported, fallback...)
		typed = nil
	}
	if !allowManaged {
		if _, ok := typed.(*instructions.Stage); ok {
			return fmt.Errorf("line %d: FROM is not permitted in ONBUILD", node.StartLine)
		}
	}
	if allowManaged {
		if stage, ok := typed.(*instructions.Stage); ok {
			discovery.beginStage(stage)
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
		candidates, err := dockerInstructionCandidates(node, typed, deferred, discovery)
		if err != nil {
			return fmt.Errorf("line %d: %w", node.StartLine, err)
		}
		*unsupported = append(*unsupported, candidates...)
	}
	if allowManaged {
		switch command := typed.(type) {
		case *instructions.Stage:
			discovery.recordCurrentStage()
		case *instructions.ShellCommand:
			discovery.setRuntimeShell(command.Shell)
		}
	}
	if len(*managed)+len(*unsupported) > maxDockerCandidates {
		return resourceLimitf("Docker metadata exceeds the %d-candidate limit", maxDockerCandidates)
	}
	if onbuild, ok := typed.(*instructions.OnbuildCommand); ok {
		trigger, err := parser.Parse(strings.NewReader(onbuild.Expression))
		if err != nil {
			return fmt.Errorf("line %d: parse ONBUILD trigger: %w", node.StartLine, err)
		}
		if len(trigger.AST.Children) != 1 {
			return fmt.Errorf("line %d: ONBUILD must contain exactly one instruction", node.StartLine)
		}
		if err := classifyDockerInstruction(trigger.AST.Children[0], false, true, depth+1, discovery, managed, unsupported); err != nil {
			return fmt.Errorf("line %d: ONBUILD trigger: %w", node.StartLine, err)
		}
	}
	return nil
}

func fallbackDockerInstructionCandidates(node *parser.Node, discovery *dockerDiscovery, parseError error) ([]dockerCandidate, bool, error) {
	if !isRunMountExpansionError(node, parseError) {
		return nil, false, nil
	}
	candidates := []dockerCandidate{}
	for _, flag := range node.Flags {
		mount, found := strings.CutPrefix(flag, "--mount=")
		if !found {
			continue
		}
		for _, field := range strings.Split(mount, ",") {
			value, found := strings.CutPrefix(field, "from=")
			if !found {
				continue
			}
			normalized, err := discovery.normalizeDockerWord(value)
			if err != nil {
				return nil, true, err
			}
			if !discovery.isStageReference(normalized) && containsDockerGoToken(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "run-mount-from", Value: value, Line: node.StartLine})
			}
		}
	}
	command := instructions.ShellDependantCmdLine{PrependShell: !node.Attributes["json"]}
	for value := node.Next; value != nil; value = value.Next {
		command.CmdLine = append(command.CmdLine, value.Value)
	}
	for _, heredoc := range node.Heredocs {
		command.Files = append(command.Files, instructions.ShellInlineFile{Data: heredoc.Content})
	}
	commandCandidates, err := discovery.commandCandidates(node, command)
	if err != nil {
		return nil, true, err
	}
	return append(candidates, commandCandidates...), true, nil
}

func parseDockerInstruction(node *parser.Node) (any, error) {
	return instructions.ParseInstruction(node)
}

func containsPotentialDockerGoReference(value string) bool {
	return potentialDockerGoPattern.MatchString(value) ||
		potentialDockerGoDownloadPattern.MatchString(value)
}

func dockerInstructionCandidates(node *parser.Node, typed any, deferred bool, discovery *dockerDiscovery) ([]dockerCandidate, error) {
	candidates := []dockerCandidate{}
	appendImageCandidate := func(kind, value string) error {
		normalized, err := discovery.normalizeDockerWord(value)
		if err != nil {
			return err
		}
		if !deferred && discovery.isStageReference(normalized) {
			return nil
		}
		if isGolangImage(normalized) || containsDockerGoToken(normalized) {
			candidates = append(candidates, dockerCandidate{Kind: kind, Value: value, Line: node.StartLine})
		}
		return nil
	}

	switch command := typed.(type) {
	case *instructions.Stage:
		if err := appendImageCandidate("from", command.BaseName); err != nil {
			return nil, err
		}
	case *instructions.ArgCommand:
		for _, argument := range command.Args {
			if argument.Value == nil {
				continue
			}
			if !containsPotentialDockerGoReference(*argument.Value) {
				continue
			}
			normalized, err := discovery.normalizeDockerWord(*argument.Value)
			if err != nil {
				return nil, err
			}
			if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "arg-default", Value: *argument.Value, Line: node.StartLine})
			}
		}
	case *instructions.EnvCommand:
		for _, environment := range command.Env {
			if !containsPotentialDockerGoReference(environment.Value) {
				continue
			}
			normalized, err := discovery.normalizeDockerWord(environment.Value)
			if err != nil {
				return nil, err
			}
			if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "env-value", Value: environment.Value, Line: node.StartLine})
			}
		}
	case *instructions.AddCommand:
		for _, source := range command.SourcePaths {
			if !containsPotentialDockerGoReference(source) {
				continue
			}
			normalized, err := discovery.normalizeDockerWord(source)
			if err != nil {
				return nil, err
			}
			if goDownloadPattern.MatchString(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "add-download", Value: source, Line: node.StartLine})
			}
		}
	case *instructions.CopyCommand:
		if command.From != "" {
			if err := appendImageCandidate("copy-from", command.From); err != nil {
				return nil, err
			}
		}
	case *instructions.RunCommand:
		for _, mount := range instructions.GetMounts(command) {
			if mount.From == "" {
				continue
			}
			if err := appendImageCandidate("run-mount-from", mount.From); err != nil {
				return nil, err
			}
		}
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, commandCandidates...)
	case *instructions.CmdCommand:
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, commandCandidates...)
	case *instructions.EntrypointCommand:
		commandCandidates, err := discovery.commandCandidates(node, command.ShellDependantCmdLine)
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
			commandCandidates, err := discovery.commandCandidates(node, commandLine)
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

func (discovery *dockerDiscovery) beginStage(stage *instructions.Stage) {
	inheritedShell, local := discovery.stageReferences[strings.ToLower(stage.BaseName)]
	if !local {
		inheritedShell = defaultRuntimeShell()
	}
	discovery.currentShell = inheritedShell
	index := strconv.Itoa(discovery.stageCount)
	discovery.stageCount++
	discovery.currentStage = []string{index}
	if stage.Name != "" {
		discovery.currentStage = append(discovery.currentStage, strings.ToLower(stage.Name))
	}
}

func (discovery *dockerDiscovery) recordCurrentStage() {
	for _, reference := range discovery.currentStage {
		discovery.stageReferences[reference] = discovery.currentShell
	}
}

func (discovery *dockerDiscovery) setRuntimeShell(command []string) {
	discovery.currentShell = runtimeShellFor(command)
	for _, reference := range discovery.currentStage {
		discovery.stageReferences[reference] = discovery.currentShell
	}
}

func (discovery *dockerDiscovery) isStageReference(value string) bool {
	_, found := discovery.stageReferences[strings.ToLower(value)]
	return found
}

func (discovery *dockerDiscovery) normalizeDockerWord(value string) (string, error) {
	key := dockerWordKey{value: value}
	if result, found := discovery.wordMemo[key]; found {
		return result.value, result.err
	}
	normalized, _, err := discovery.wordLexer.ProcessWord(value, shell.EnvsFromSlice(nil))
	if err != nil {
		err = fmt.Errorf("normalize Docker word %q: %w", value, err)
	}
	if err == nil {
		err = discovery.accountNormalizedBytes(len(normalized))
	}
	discovery.wordMemo[key] = dockerWordResult{value: normalized, err: err}
	return normalized, err
}

func (discovery *dockerDiscovery) accountNormalizedBytes(count int) error {
	discovery.normalizedBytes += count
	if discovery.normalizedBytes > maxDockerNormalizedBytes {
		return resourceLimitf("Docker metadata exceeds the %d-byte normalized-word limit", maxDockerNormalizedBytes)
	}
	return nil
}

func (discovery *dockerDiscovery) commandCandidates(node *parser.Node, command instructions.ShellDependantCmdLine) ([]dockerCandidate, error) {
	words := command.CmdLine
	if command.PrependShell {
		if !discovery.currentShell.known {
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
			parameters := append([]string{words[0]}, words[3:]...)
			if len(words) > 3 {
				parameters[0] = words[3]
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
		environment := &runtimeShellEnvironment{fallback: environmentValue, values: map[string]expand.Variable{}}
		if len(parameters) > 0 {
			positionals := append([]string(nil), parameters[1:]...)
			environment.values["@"] = expand.Variable{Set: true, Kind: expand.Indexed, List: positionals}
			environment.values["*"] = environment.values["@"]
			environment.values["#"] = expand.Variable{Set: true, Kind: expand.String, Str: strconv.Itoa(len(positionals))}
			for index, parameter := range parameters {
				environment.values[strconv.Itoa(index)] = expand.Variable{Set: true, Kind: expand.String, Str: parameter}
			}
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
