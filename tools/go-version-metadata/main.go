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
	"regexp/syntax"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// dockerInstructionMetadata is a bounded semantic projection of the pinned
// BuildKit AST. Policy callers use it instead of reimplementing Docker's
// physical-line, continuation, heredoc, and JSON parsing rules.
type dockerInstructionMetadata struct {
	Command string               `json:"command"`
	Line    int                  `json:"line"`
	Flags   []string             `json:"flags"`
	Stage   *dockerStageMetadata `json:"stage,omitempty"`
	Copy    *dockerCopyMetadata  `json:"copy,omitempty"`
	Run     *dockerRunMetadata   `json:"run,omitempty"`
}

type dockerParserDirectiveMetadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Line  int    `json:"line"`
}

type dockerStageMetadata struct {
	BaseName string `json:"baseName"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type dockerCopyMetadata struct {
	From          string   `json:"from"`
	Sources       []string `json:"sources"`
	Destination   string   `json:"destination"`
	InlineSources int      `json:"inlineSources"`
}

type dockerRunMetadata struct {
	Arguments    []string `json:"arguments"`
	PrependShell bool     `json:"prependShell"`
	HeredocFiles int      `json:"heredocFiles"`
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
	YAMLValues           map[string][]string             `json:"yamlValues,omitempty"`
	HasGoDownload        bool                            `json:"hasGoDownload,omitempty"`
	DockerClassification string                          `json:"dockerClassification,omitempty"`
	DockerCandidates     []dockerCandidate               `json:"dockerCandidates,omitempty"`
	DockerError          string                          `json:"dockerError,omitempty"`
	DockerInstructions   []dockerInstructionMetadata     `json:"dockerInstructions,omitempty"`
	DockerDirectives     []dockerParserDirectiveMetadata `json:"dockerDirectives,omitempty"`
	Module               *moduleMetadata                 `json:"module,omitempty"`
}

var goDownloadPattern = regexp.MustCompile(`(?i)^https://(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var goDownloadTextPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9+.-])https://(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var exactToolchainVersionPattern = regexp.MustCompile(`^1\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)
var canonicalDockerGoImagePattern = regexp.MustCompile(`^FROM golang:((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))(-[a-z0-9][a-z0-9._-]*)?@sha256:([0-9a-f]{64}) AS ([a-z0-9][a-z0-9_.-]*)$`)

const (
	maxInputBytes               = 4 << 20
	maxRequestEnvelopeBytes     = 32 << 20
	maxYAMLDocuments            = 64
	maxYAMLNodes                = 100000
	maxYAMLEdges                = 150000
	maxYAMLDepth                = 256
	maxYAMLScalarBytes          = 1 << 20
	maxDockerInstructions       = 100000
	maxDockerCandidates         = 10000
	maxDockerInstructionDepth   = 256
	maxShellASTNodes            = 100000
	maxShellASTDepth            = 256
	maxDockerNormalizedBytes    = 16 << 20
	maxDockerWordInputBytes     = 16 << 10
	maxDockerWordWorkBytes      = 32 << 10
	maxDockerWordVariables      = 6
	maxDockerWordAlternatives   = 729
	maxDockerAlternativeWork    = 1 << 20
	maxReferenceMachineStates   = 128
	maxReferenceTransformations = 2048
	maxSymbolicReferenceWork    = 1 << 20
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
		if classification != "invalid" {
			directives, err := projectDockerParserDirectives(input.Contents)
			if err != nil {
				return response{}, fmt.Errorf("%s: project Docker parser directives: %w", input.Path, err)
			}
			projected, err := projectDockerInstructions(input.Contents)
			if err != nil {
				return response{}, fmt.Errorf("%s: project Docker instructions: %w", input.Path, err)
			}
			metadata.DockerInstructions = projected
			metadata.DockerDirectives = directives
		}
	}
	return metadata, nil
}

func projectDockerParserDirectives(contents string) ([]dockerParserDirectiveMetadata, error) {
	projected := []dockerParserDirectiveMetadata{}
	if _, commandLine, locations, found := parser.DetectSyntax([]byte(contents)); found {
		line := 0
		if len(locations) > 0 {
			line = locations[0].Start.Line
		}
		projected = append(projected, dockerParserDirectiveMetadata{
			Name: "syntax", Value: commandLine, Line: line,
		})
	} else if directive, found := compatibleHashSyntaxDirective(contents); found {
		projected = append(projected, directive)
	}
	directiveParser := parser.DirectiveParser{}
	directives, err := directiveParser.ParseAll([]byte(contents))
	if err != nil {
		return nil, err
	}
	for _, directive := range directives {
		if directive.Name == "syntax" {
			continue
		}
		line := 0
		if len(directive.Location) > 0 {
			line = directive.Location[0].Start.Line
		}
		projected = append(projected, dockerParserDirectiveMetadata{
			Name: directive.Name, Value: directive.Value, Line: line,
		})
	}
	return projected, nil
}

// compatibleHashSyntaxDirective recognizes the initial hash-directive preamble
// accepted by newer supported builders but not by the pinned parser. It is a
// deliberately small, linear compatibility scanner: it never searches beyond
// the first blank, ordinary comment, unknown directive, indented line, or
// instruction. Managed files fail closed on this bounded superset so executor
// upgrades cannot silently select a different frontend.
func compatibleHashSyntaxDirective(contents string) (dockerParserDirectiveMetadata, bool) {
	contents = strings.TrimPrefix(contents, "\ufeff")
	lines := strings.Split(contents, "\n")
	start := 0
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSuffix(lines[0], "\r"), "#!") {
		start = 1
	}
	for index := start; index < len(lines); index++ {
		line := strings.TrimSuffix(lines[index], "\r")
		if line == "" || !strings.HasPrefix(line, "#") {
			return dockerParserDirectiveMetadata{}, false
		}
		body := strings.TrimLeftFunc(line[1:], unicode.IsSpace)
		nameEnd := 0
		if len(body) == 0 || !isASCIILetter(body[0]) {
			return dockerParserDirectiveMetadata{}, false
		}
		for nameEnd < len(body) {
			character := body[nameEnd]
			if !isASCIILetter(character) && !isASCIIDigit(character) {
				break
			}
			nameEnd++
		}
		name := strings.ToLower(body[:nameEnd])
		remainder := trimDockerDirectiveSpace(body[nameEnd:])
		if !strings.HasPrefix(remainder, "=") {
			return dockerParserDirectiveMetadata{}, false
		}
		value := trimDockerDirectiveSpace(remainder[1:])
		if value == "" {
			return dockerParserDirectiveMetadata{}, false
		}
		switch name {
		case "syntax":
			return dockerParserDirectiveMetadata{
				Name: name, Value: value, Line: index + 1,
			}, true
		case "escape", "check":
			continue
		default:
			return dockerParserDirectiveMetadata{}, false
		}
	}
	return dockerParserDirectiveMetadata{}, false
}

func trimDockerDirectiveSpace(value string) string {
	return strings.Trim(value, " \t\r\f")
}

func isASCIILetter(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z')
}

func isASCIIDigit(character byte) bool {
	return character >= '0' && character <= '9'
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
	deferred, err := discovery.exportedDeferredCandidates()
	if err != nil {
		return dockerValidationFailure(err)
	}
	seenUnsupported := make(map[dockerCandidate]bool, len(unsupported))
	for _, candidate := range unsupported {
		seenUnsupported[candidate] = true
	}
	for _, candidate := range deferred {
		if !seenUnsupported[candidate] {
			seenUnsupported[candidate] = true
			unsupported = append(unsupported, candidate)
		}
	}
	directives, err := projectDockerParserDirectives(contents)
	if err != nil {
		return "invalid", nil, err.Error()
	}
	for _, directive := range directives {
		if directive.Name != "syntax" {
			continue
		}
		candidate := dockerCandidate{
			Kind: "unsupported-frontend", Value: directive.Value, Line: directive.Line,
		}
		if !seenUnsupported[candidate] {
			seenUnsupported[candidate] = true
			unsupported = append(unsupported, candidate)
		}
	}
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

func projectDockerInstructions(contents string) ([]dockerInstructionMetadata, error) {
	parsed, err := parser.Parse(strings.NewReader(contents))
	if err != nil {
		return nil, err
	}
	projected := make([]dockerInstructionMetadata, 0, len(parsed.AST.Children))
	for _, node := range parsed.AST.Children {
		typed, err := instructions.ParseInstruction(node)
		if err != nil {
			return nil, err
		}
		record := dockerInstructionMetadata{
			Command: strings.ToLower(node.Value),
			Line:    node.StartLine,
			Flags:   slices.Clone(node.Flags),
		}
		if record.Flags == nil {
			record.Flags = []string{}
		}
		switch command := typed.(type) {
		case *instructions.Stage:
			record.Stage = &dockerStageMetadata{
				BaseName: command.BaseName,
				Name:     command.Name,
				Platform: command.Platform,
			}
		case *instructions.CopyCommand:
			record.Copy = &dockerCopyMetadata{
				From:          command.From,
				Sources:       slices.Clone(command.SourcePaths),
				Destination:   command.DestPath,
				InlineSources: len(command.SourceContents),
			}
			if record.Copy.Sources == nil {
				record.Copy.Sources = []string{}
			}
		case *instructions.RunCommand:
			record.Run = &dockerRunMetadata{
				Arguments:    slices.Clone(command.CmdLine),
				PrependShell: command.PrependShell,
				HeredocFiles: len(command.Files),
			}
			if record.Run.Arguments == nil {
				record.Run.Arguments = []string{}
			}
		}
		projected = append(projected, record)
	}
	return projected, nil
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
				if err := discovery.validateLiteralStageSource(command.From); err != nil {
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
				if err := discovery.validateLiteralStageSource(mount.From); err != nil {
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
			_, deferred, err := parseDeferredDockerInstruction(command.Expression, node.StartLine, discovery)
			if err != nil {
				return err
			}
			switch deferred := deferred.(type) {
			case *instructions.CopyCommand:
				if deferred.From != "" {
					if err := discovery.validateLiteralStageSource(deferred.From); err != nil {
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
						if err := discovery.validateLiteralStageSource(mount.From); err != nil {
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
	value       string
	escapeToken rune
}

type dockerWordResult struct {
	validation       string
	values           []symbolicValue
	symbolic         []symbolicValue
	variables        map[string]bool
	activeParameters []string
	fixedParameters  []string
	unknown          bool
	err              error
}

type runtimeWordsKey struct {
	value      string
	parameters string
}

type runtimeWordsResult struct {
	values []runtimeWord
	err    error
}

type symbolicSourceResult struct {
	image         bool
	imageKnown    bool
	download      bool
	downloadKnown bool
	err           error
}

type symbolicSearchKey struct {
	value        string
	target       string
	prefix       bool
	requireFixed bool
	foldCase     bool
}

type symbolicSearchResult struct {
	matched bool
	err     error
}

type compiledSymbolicAtom struct {
	literal       byte
	variableIndex int
}

type compiledSymbolicPattern struct {
	key                 string
	atoms               []compiledSymbolicAtom
	variableCount       int
	variableOccurrences [maxDockerWordVariables]int
	onlyVariables       bool
}

type symbolicSegment struct {
	literal  string
	variable string
}

type symbolicValue struct {
	segments   []symbolicSegment
	compiled   *compiledSymbolicPattern
	compileErr error
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
}

type dockerStageState struct {
	references []string
	baseName   string
	shell      runtimeShell
	triggers   []*deferredDockerInstruction
}

type dockerInstructionContext struct {
	shell             runtimeShell
	conservative      bool
	currentReferences []string
}

type dockerDiscovery struct {
	escapeToken         rune
	wordLexer           *shell.Lex
	wordMemo            map[dockerWordKey]dockerWordResult
	runtimeWordsMemo    map[runtimeWordsKey]runtimeWordsResult
	symbolicMemo        map[string]symbolicSourceResult
	symbolicPatternMemo map[string]*compiledSymbolicPattern
	symbolicSearchMemo  map[symbolicSearchKey]symbolicSearchResult
	instructions        int
	candidates          int
	normalizedBytes     int
	wordWorkBytes       int
	alternativeWork     int
	symbolicWork        int
	stageReferences     map[string]*dockerStageState
	baseReferences      map[string]*dockerStageState
	allStages           []*dockerStageState
	currentStage        *dockerStageState
	stageCount          int
}

func newDockerDiscovery(escapeToken rune) *dockerDiscovery {
	wordLexer := shell.NewLex(escapeToken)
	wordLexer.SkipUnsetEnv = true
	return &dockerDiscovery{
		escapeToken:         escapeToken,
		wordLexer:           wordLexer,
		wordMemo:            map[dockerWordKey]dockerWordResult{},
		runtimeWordsMemo:    map[runtimeWordsKey]runtimeWordsResult{},
		symbolicMemo:        map[string]symbolicSourceResult{},
		symbolicPatternMemo: map[string]*compiledSymbolicPattern{},
		symbolicSearchMemo:  map[symbolicSearchKey]symbolicSearchResult{},
		stageReferences:     map[string]*dockerStageState{},
		baseReferences:      map[string]*dockerStageState{},
	}
}

func (discovery *dockerDiscovery) dockerWordKey(value string) dockerWordKey {
	return dockerWordKey{value: value, escapeToken: discovery.escapeToken}
}

func (discovery *dockerDiscovery) withEscapeToken(escapeToken rune, evaluate func() error) error {
	previousToken := discovery.escapeToken
	previousLexer := discovery.wordLexer
	wordLexer := shell.NewLex(escapeToken)
	wordLexer.SkipUnsetEnv = true
	discovery.escapeToken = escapeToken
	discovery.wordLexer = wordLexer
	defer func() {
		discovery.escapeToken = previousToken
		discovery.wordLexer = previousLexer
	}()
	return evaluate()
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

func parseDeferredDockerInstruction(expression string, line int, discovery *dockerDiscovery) (*parser.Node, any, error) {
	// BuildKit stores ONBUILD expressions and reparses them as standalone
	// Dockerfiles with the default backslash parser. The caller selects the
	// consuming Dockerfile's escape token for typed word normalization; exported
	// triggers are evaluated once for each legal future child token.
	parsed, err := parser.Parse(strings.NewReader(expression))
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: parse ONBUILD trigger: %w", line, err)
	}
	if len(parsed.AST.Children) != 1 {
		return nil, nil, fmt.Errorf("line %d: ONBUILD must contain exactly one instruction", line)
	}
	node := parsed.AST.Children[0]
	node.StartLine = line
	node.EndLine = line
	typed, err := parseDockerInstruction(node, discovery)
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: ONBUILD trigger: %w", line, err)
	}
	return node, typed, nil
}

func (discovery *dockerDiscovery) validateLiteralStageSource(value string) error {
	normalized, err := discovery.normalizeDockerWord(value)
	if err != nil {
		return err
	}
	// BuildKit rejects COPY --from whenever its word lexer changes the parsed
	// flag value. This includes parameter expansion and escape normalization:
	// the Dockerfile parser can retain one escape that the word lexer removes.
	if normalized != value || discovery.dockerWordHasExpansion(value) {
		return fmt.Errorf("expanded stage source %q is not supported by BuildKit", value)
	}
	return nil
}

func (discovery *dockerDiscovery) validateDeferredStageSources(typed any) error {
	switch command := typed.(type) {
	case *instructions.CopyCommand:
		if command.From != "" {
			return discovery.validateLiteralStageSource(command.From)
		}
	case *instructions.RunCommand:
		for _, mount := range instructions.GetMounts(command) {
			if mount.From != "" {
				if err := discovery.validateLiteralStageSource(mount.From); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateDockerInstructionWords(typed any, discovery *dockerDiscovery, line int) error {
	validate := func(word string) (string, error) {
		if _, err := discovery.dockerWordAlternatives(word); err != nil {
			return "", err
		}
		result := discovery.wordMemo[discovery.dockerWordKey(word)]
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
		node, typed, err := parseDeferredDockerInstruction(trigger.expression, trigger.line, discovery)
		if err != nil {
			return nil, context.shell, err
		}
		if err := discovery.validateDeferredStageSources(typed); err != nil {
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
	appendUnsupportedParameterReference := func(value string) bool {
		result := discovery.wordMemo[discovery.dockerWordKey(value)]
		if len(result.fixedParameters) == 0 {
			return false
		}
		candidates = append(candidates, dockerCandidate{Kind: "unsupported-parameter-reference", Value: value, Line: node.StartLine})
		return true
	}
	appendImageCandidate := func(kind, value string, rejectCurrent, numericLocal bool) error {
		alternatives, err := discovery.dockerWordAlternatives(value)
		if err != nil {
			return err
		}
		if appendUnsupportedParameterReference(value) {
			return nil
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
		possible := false
		if !matched {
			var possibleErr error
			possible, possibleErr = discovery.dockerWordUnknownMayContainSource(value, false)
			if possibleErr != nil {
				return possibleErr
			}
		}
		if matched || possible {
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
			if isDockerWordFixedUnsetParameter(argument.Key) {
				candidates = append(candidates, dockerCandidate{Kind: "unsupported-parameter-name", Value: argument.Key, Line: node.StartLine})
			}
			if argument.Value == nil {
				continue
			}
			alternatives, err := discovery.dockerWordAlternatives(*argument.Value)
			if err != nil {
				return nil, err
			}
			if appendUnsupportedParameterReference(*argument.Value) {
				continue
			}
			matched := false
			for _, normalized := range alternatives {
				if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			possible := false
			if !matched {
				var possibleErr error
				possible, possibleErr = discovery.dockerWordUnknownMayContainSource(*argument.Value, false)
				if possibleErr != nil {
					return nil, possibleErr
				}
			}
			if matched || possible {
				candidates = append(candidates, dockerCandidate{Kind: "arg-default", Value: *argument.Value, Line: node.StartLine})
			}
		}
	case *instructions.EnvCommand:
		for _, environment := range command.Env {
			keyAlternatives, err := discovery.dockerWordAlternatives(environment.Key)
			if err != nil {
				return nil, err
			}
			unsupportedKey := discovery.dockerWordHasExpansion(environment.Key)
			for _, key := range keyAlternatives {
				unsupportedKey = unsupportedKey || isDockerWordFixedUnsetParameter(key)
			}
			if unsupportedKey {
				candidates = append(candidates, dockerCandidate{Kind: "unsupported-parameter-name", Value: environment.Key, Line: node.StartLine})
			}
			alternatives, err := discovery.dockerWordAlternatives(environment.Value)
			if err != nil {
				return nil, err
			}
			if appendUnsupportedParameterReference(environment.Value) {
				continue
			}
			matched := false
			for _, normalized := range alternatives {
				if containsDockerGoToken(normalized) || goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			possible := false
			if !matched {
				var possibleErr error
				possible, possibleErr = discovery.dockerWordUnknownMayContainSource(environment.Value, false)
				if possibleErr != nil {
					return nil, possibleErr
				}
			}
			if matched || possible {
				candidates = append(candidates, dockerCandidate{Kind: "env-value", Value: environment.Value, Line: node.StartLine})
			}
		}
	case *instructions.AddCommand:
		for _, source := range command.SourcePaths {
			alternatives, err := discovery.dockerWordAlternatives(source)
			if err != nil {
				return nil, err
			}
			if appendUnsupportedParameterReference(source) {
				continue
			}
			matched := false
			for _, normalized := range alternatives {
				if goDownloadPattern.MatchString(normalized) {
					matched = true
					break
				}
			}
			possible := false
			if !matched {
				var possibleErr error
				possible, possibleErr = discovery.dockerWordUnknownMayContainSource(source, true)
				if possibleErr != nil {
					return nil, possibleErr
				}
			}
			if matched || possible {
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
	wordResult := discovery.wordMemo[discovery.dockerWordKey(stage.BaseName)]
	for symbolicIndex := range wordResult.symbolic {
		externalPossible = true
		pattern, compileErr := discovery.compileSymbolic(&wordResult.symbolic[symbolicIndex])
		if compileErr != nil {
			return compileErr
		}
		for reference, base := range discovery.baseReferences {
			matched, matchErr := discovery.symbolicPatternMatch(pattern, reference, false, false, false)
			if matchErr != nil {
				return matchErr
			}
			if matched && !seenBases[base] {
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
	}
	return nil
}

func (discovery *dockerDiscovery) completeCurrentStage() error {
	if discovery.currentStage == nil {
		return nil
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

func (discovery *dockerDiscovery) stageIsExportable(index int, stage *dockerStageState) bool {
	if index == len(discovery.allStages)-1 {
		return true
	}
	if stage.baseName == "" {
		return false
	}
	return discovery.baseReferences[stage.baseName] == stage
}

func (discovery *dockerDiscovery) exportedDeferredCandidates() ([]dockerCandidate, error) {
	candidates := []dockerCandidate{}
	seen := map[dockerCandidate]bool{}
	for index, stage := range discovery.allStages {
		if len(stage.triggers) == 0 || !discovery.stageIsExportable(index, stage) {
			continue
		}
		context := dockerInstructionContext{
			shell:             stage.shell,
			conservative:      true,
			currentReferences: stage.references,
		}
		for _, escapeToken := range []rune{'\\', '`'} {
			var evaluated []dockerCandidate
			err := discovery.withEscapeToken(escapeToken, func() error {
				found, _, err := discovery.evaluateDeferred(stage.triggers, context)
				evaluated = found
				return err
			})
			if err != nil {
				return nil, err
			}
			for _, candidate := range evaluated {
				if !seen[candidate] {
					seen[candidate] = true
					candidates = append(candidates, candidate)
				}
			}
		}
	}
	return candidates, nil
}

func (discovery *dockerDiscovery) normalizeDockerWord(value string) (string, error) {
	_, err := discovery.dockerWordAlternatives(value)
	if err != nil {
		return "", err
	}
	return discovery.wordMemo[discovery.dockerWordKey(value)].validation, nil
}

func (discovery *dockerDiscovery) dockerWordAlternatives(value string) ([]string, error) {
	key := discovery.dockerWordKey(value)
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
	activeParameters := make([]string, 0, len(probe.Unmatched))
	fixedParameters := []string{}
	for name := range probe.Unmatched {
		activeParameters = append(activeParameters, name)
		// Docker's word lexer recognizes shell special and positional parameter
		// syntax, but Docker's ARG/ENV state cannot supply those parameters while
		// expanding Dockerfile metadata. BuildKit consequently gives them the one
		// deterministic unset value. Do not project the ordinary variable domain
		// (unset, empty, arbitrary) onto values that cannot occur.
		if isDockerWordFixedUnsetParameter(name) {
			fixedParameters = append(fixedParameters, name)
		} else {
			variables = append(variables, name)
		}
	}
	sort.Strings(variables)
	sort.Strings(activeParameters)
	sort.Strings(fixedParameters)
	validation := probe.Result
	if len(variables) > maxDockerWordVariables {
		err := resourceLimitf("Docker word references more than %d variables", maxDockerWordVariables)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return nil, err
	}
	unknownNames := make(map[string]bool, len(probe.Unmatched))
	for name := range probe.Unmatched {
		if !isDockerWordFixedUnsetParameter(name) {
			unknownNames[name] = true
		}
	}
	if _, unsupported := unsupportedDockerWordOperator(value, unknownNames, discovery.escapeToken); unsupported {
		// Content-sensitive operators are outside the contract. Do not feed the
		// private identity framing through an operator that can rewrite it; typed
		// validation will emit the policy diagnostic immediately after this call.
		if err := discovery.accountNormalizedBytes(len(validation)); err != nil {
			discovery.wordMemo[key] = dockerWordResult{err: err}
			return nil, err
		}
		concrete := symbolicValue{segments: []symbolicSegment{{literal: validation}}}
		discovery.wordMemo[key] = dockerWordResult{
			validation:       validation,
			values:           []symbolicValue{concrete},
			variables:        unknownNames,
			activeParameters: activeParameters,
			fixedParameters:  fixedParameters,
			unknown:          len(variables) > 0,
		}
		return []string{validation}, nil
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
	fixedValidationSet := false
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
		if len(fixedParameters) > 0 && !fixedValidationSet {
			// This is an exact BuildKit expansion with special/positional
			// parameters absent. Never model them as set-empty: the non-colon -
			// and + operators distinguish those states.
			validation = normalized
			fixedValidationSet = true
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
		validation:       validation,
		values:           typedLiteralValues(alternatives),
		symbolic:         symbolic,
		variables:        unknownNames,
		activeParameters: activeParameters,
		fixedParameters:  fixedParameters,
		unknown:          len(variables) > 0,
	}
	return alternatives, nil
}

func isDockerWordFixedUnsetParameter(name string) bool {
	if name == "" {
		return false
	}
	if character, size := utf8.DecodeRuneInString(name); size == len(name) && isShellSpecialParameter(character) {
		return true
	}
	// POSIX positional parameters use ASCII decimal digits. Unicode digits are
	// ordinary BuildKit variable identifiers and can be supplied by ARG/ENV.
	for index := 0; index < len(name); index++ {
		if name[index] < '0' || name[index] > '9' {
			return false
		}
	}
	return true
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
	return discovery.wordMemo[discovery.dockerWordKey(value)].unknown
}

// dockerWordHasExpansion reports active Docker parameter-expansion syntax,
// independently of whether the referenced name has an ordinary unknown value
// or BuildKit's deterministic unset value for special/positional parameters.
// Callers use this syntax fact for fields whose grammar forbids expansion.
func (discovery *dockerDiscovery) dockerWordHasExpansion(value string) bool {
	return len(discovery.wordMemo[discovery.dockerWordKey(value)].activeParameters) > 0
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

func (discovery *dockerDiscovery) dockerWordUnknownMayContainSource(value string, downloadOnly bool) (bool, error) {
	if !discovery.dockerWordHasUnknown(value) {
		return false, nil
	}
	wordResult := discovery.wordMemo[discovery.dockerWordKey(value)]
	for index := range wordResult.symbolic {
		sourceResult := discovery.symbolicSource(&wordResult.symbolic[index], !downloadOnly)
		if sourceResult.err != nil {
			return false, sourceResult.err
		}
		if sourceResult.download || !downloadOnly && sourceResult.image {
			return true, nil
		}
	}
	return false, nil
}

func (discovery *dockerDiscovery) accountSymbolicWork(count int) error {
	if count < 0 || discovery.symbolicWork > maxSymbolicReferenceWork-count {
		return resourceLimitf("Docker metadata exceeds the %d-step symbolic-reference search limit", maxSymbolicReferenceWork)
	}
	discovery.symbolicWork += count
	return nil
}

func (discovery *dockerDiscovery) compileSymbolic(value *symbolicValue) (*compiledSymbolicPattern, error) {
	if value.compiled != nil || value.compileErr != nil {
		return value.compiled, value.compileErr
	}
	// Charge a conservative bound for scanning input identities, constructing
	// the canonical key, and retaining atoms before doing any hashing or
	// allocation. The fixed per-segment allowance covers framing and decimal
	// lengths; literal bytes are charged once for the key and once for atoms.
	preprocessWork := 0
	for _, segment := range value.segments {
		contents := segment.literal
		atomCount := len(contents)
		if segment.variable != "" {
			contents = segment.variable
			atomCount = 1
		}
		increment := len(contents) + atomCount + 24
		if preprocessWork > maxSymbolicReferenceWork-increment {
			value.compileErr = resourceLimitf("Docker metadata exceeds the %d-step symbolic-reference search limit", maxSymbolicReferenceWork)
			return nil, value.compileErr
		}
		preprocessWork += increment
	}
	if err := discovery.accountSymbolicWork(preprocessWork); err != nil {
		value.compileErr = err
		return nil, err
	}

	variableIndexes := map[string]int{}
	var key strings.Builder
	for _, segment := range value.segments {
		kind, contents := byte('L'), segment.literal
		if segment.variable != "" {
			kind = 'V'
			variableIndex, found := variableIndexes[segment.variable]
			if !found {
				variableIndex = len(variableIndexes)
				variableIndexes[segment.variable] = variableIndex
			}
			contents = strconv.Itoa(variableIndex)
		}
		key.WriteByte(kind)
		key.WriteString(strconv.Itoa(len(contents)))
		key.WriteByte(':')
		key.WriteString(contents)
	}
	canonicalKey := key.String()
	if existing := discovery.symbolicPatternMemo[canonicalKey]; existing != nil {
		value.compiled = existing
		return existing, nil
	}
	if len(variableIndexes) > maxDockerWordVariables {
		value.compileErr = resourceLimitf("Docker metadata symbolic value references more than %d variable identities", maxDockerWordVariables)
		return nil, value.compileErr
	}

	pattern := &compiledSymbolicPattern{
		key:           canonicalKey,
		variableCount: len(variableIndexes),
		onlyVariables: true,
	}
	for _, segment := range value.segments {
		if segment.variable != "" {
			variableIndex := variableIndexes[segment.variable]
			pattern.atoms = append(pattern.atoms, compiledSymbolicAtom{variableIndex: variableIndex})
			pattern.variableOccurrences[variableIndex]++
			continue
		}
		if segment.literal != "" {
			pattern.onlyVariables = false
		}
		for index := 0; index < len(segment.literal); index++ {
			pattern.atoms = append(pattern.atoms, compiledSymbolicAtom{literal: segment.literal[index], variableIndex: -1})
		}
	}
	discovery.symbolicPatternMemo[canonicalKey] = pattern
	value.compiled = pattern
	return pattern, nil
}

func (discovery *dockerDiscovery) symbolicSource(value *symbolicValue, includeImage bool) symbolicSourceResult {
	pattern, err := discovery.compileSymbolic(value)
	if err != nil {
		return symbolicSourceResult{err: err}
	}
	result := discovery.symbolicMemo[pattern.key]
	if !result.downloadKnown && result.err == nil {
		for _, prefix := range []string{"https://go.dev/dl/go", "https://dl.google.com/go/go"} {
			var matched bool
			matched, result.err = discovery.symbolicPatternMatch(pattern, prefix, true, true, true)
			if result.err != nil || matched {
				result.download = matched
				break
			}
		}
		result.downloadKnown = result.err == nil
	}
	if includeImage && !result.imageKnown && result.err == nil {
		remaining := maxSymbolicReferenceWork - discovery.symbolicWork
		var work int
		result.image, work, result.err = symbolicGolangImagePossible(*value, remaining)
		result.imageKnown = true
		discovery.symbolicWork += work
	}
	discovery.symbolicMemo[pattern.key] = result
	return result
}

func (discovery *dockerDiscovery) symbolicCanStartWith(value *symbolicValue, prefix string) (bool, error) {
	pattern, err := discovery.compileSymbolic(value)
	if err != nil {
		return false, err
	}
	if pattern.onlyVariables {
		return false, nil
	}
	return discovery.symbolicPatternMatch(pattern, prefix, true, true, true)
}

func (discovery *dockerDiscovery) symbolicCanEqual(value *symbolicValue, target string) (bool, error) {
	pattern, err := discovery.compileSymbolic(value)
	if err != nil {
		return false, err
	}
	return discovery.symbolicPatternMatch(pattern, target, false, false, false)
}

func (discovery *dockerDiscovery) symbolicPatternMatch(pattern *compiledSymbolicPattern, target string, prefix, requireFixed, foldCase bool) (bool, error) {
	key := symbolicSearchKey{
		value:        pattern.key,
		target:       target,
		prefix:       prefix,
		requireFixed: requireFixed,
		foldCase:     foldCase,
	}
	if result, found := discovery.symbolicSearchMemo[key]; found {
		return result.matched, result.err
	}
	remaining := maxSymbolicReferenceWork - discovery.symbolicWork
	matched, work, err := matchSymbolicText(pattern, target, prefix, requireFixed, foldCase, remaining)
	discovery.symbolicWork += work
	result := symbolicSearchResult{matched: matched, err: err}
	discovery.symbolicSearchMemo[key] = result
	return result.matched, result.err
}

func matchSymbolicText(pattern *compiledSymbolicPattern, target string, prefix, requireFixed, foldCase bool, workLimit int) (bool, int, error) {
	atoms := pattern.atoms
	type matchState struct {
		patternIndex int
		targetIndex  int
		matchedFixed bool
		assigned     [maxDockerWordVariables]bool
		assignStarts [maxDockerWordVariables]int
		assignEnds   [maxDockerWordVariables]int
	}
	failed := map[matchState]bool{}
	work := 0
	exhausted := false
	var visit func(matchState) bool
	visit = func(state matchState) (matched bool) {
		if failed[state] {
			return false
		}
		defer func() {
			if !matched && !exhausted {
				failed[state] = true
			}
		}()
		if work >= workLimit {
			exhausted = true
			return false
		}
		work++
		if prefix && state.targetIndex == len(target) {
			return !requireFixed || state.matchedFixed
		}
		if state.patternIndex == len(atoms) {
			return state.targetIndex == len(target) && (!requireFixed || state.matchedFixed)
		}
		atom := atoms[state.patternIndex]
		state.patternIndex++
		if atom.variableIndex >= 0 {
			// A single-use identity has no correlation to preserve. Do not retain
			// its assignment in the memo state: all splits reaching the same next
			// atom and target offset are semantically identical. This keeps a long
			// local-stage name linear in its length instead of enumerating every
			// partition among adjacent unknowns.
			if pattern.variableOccurrences[atom.variableIndex] == 1 {
				if visit(state) { // The arbitrary value is empty.
					return true
				}
				if state.targetIndex < len(target) {
					// Consume one byte and remain on this atom. Memoization makes
					// this an O(atoms*target) wildcard search instead of enumerating
					// every partition among adjacent unknowns.
					state.targetIndex++
					state.patternIndex--
					return visit(state)
				}
				return false
			}
			variableIndex := atom.variableIndex
			if state.assigned[variableIndex] {
				assigned := target[state.assignStarts[variableIndex]:state.assignEnds[variableIndex]]
				if work > workLimit-len(assigned) {
					exhausted = true
					return false
				}
				work += len(assigned)
				if !strings.HasPrefix(target[state.targetIndex:], assigned) {
					return false
				}
				state.targetIndex += len(assigned)
				return visit(state)
			}
			state.assigned[variableIndex] = true
			state.assignStarts[variableIndex] = state.targetIndex
			for end := state.targetIndex; end <= len(target); end++ {
				state.assignEnds[variableIndex] = end
				branch := state
				branch.targetIndex = end
				if visit(branch) {
					return true
				}
			}
			return false
		}
		if state.targetIndex == len(target) {
			return false
		}
		matches := atom.literal == target[state.targetIndex]
		if foldCase {
			matches = strings.EqualFold(string(atom.literal), target[state.targetIndex:state.targetIndex+1])
		}
		if !matches {
			return false
		}
		state.targetIndex++
		state.matchedFixed = true
		return visit(state)
	}
	matched := visit(matchState{})
	if exhausted {
		return false, work, resourceLimitf("Docker metadata exceeds the %d-step symbolic-reference search limit", maxSymbolicReferenceWork)
	}
	return matched, work, nil
}

func symbolicGolangImagePossible(symbolic symbolicValue, workLimit int) (bool, int, error) {
	if len(symbolic.segments) == 0 || symbolic.onlyVariables() {
		return false, 0, nil
	}
	anchored := false
	for _, segment := range symbolic.segments {
		for _, character := range segment.literal {
			anchored = anchored || strings.ContainsRune("golang", character)
		}
	}
	if !anchored {
		return false, 0, nil
	}
	return golangReferenceMachine().matches(symbolic, workLimit)
}

type symbolicReferenceMachine struct {
	literalTransitions  [][]int
	variableTransitions [][]int
	accepting           []bool
	transformations     [][]int
}

var golangReferenceMachineState struct {
	sync.Once
	machine *symbolicReferenceMachine
}

func golangReferenceMachine() *symbolicReferenceMachine {
	golangReferenceMachineState.Do(func() {
		// These productions mirror distribution/reference's ReferenceRegexp,
		// with the remote name's final component fixed to the reserved source.
		// Matching the complete grammar keeps authority ports, path separators,
		// tags, and digests in their semantic positions. The tag maximum and
		// digest-encoding minimum are deliberately omitted: accepting additional
		// suffix lengths can only reject an invalid file conservatively, while
		// counters for those limits would make the transition monoid unboundedly
		// expensive for no source-discovery benefit.
		alphanumeric := `[a-z0-9]+`
		separator := `(?:[._]|__|[-]+)`
		pathComponent := alphanumeric + `(?:` + separator + alphanumeric + `)*`
		domainComponent := `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
		domainName := domainComponent + `(?:\.` + domainComponent + `)*`
		ipv6Address := `\[[a-fA-F0-9:]+\]`
		authority := `(?:` + domainName + `|` + ipv6Address + `)(?::[0-9]+)?`
		tag := `:[a-zA-Z0-9_][a-zA-Z0-9_.-]*`
		digest := `@[a-zA-Z][a-zA-Z0-9]*(?:[-_+.][a-zA-Z][a-zA-Z0-9]*)*:[a-fA-F0-9]+`
		pattern := `(?:docker://)?(?:` + authority + `/)?(?:` + pathComponent + `/)*(?P<reserved>golang)(?:` + tag + `)?(?:` + digest + `)?`
		golangReferenceMachineState.machine = compileSymbolicReferenceMachine(pattern)
	})
	return golangReferenceMachineState.machine
}

func compileSymbolicReferenceMachine(pattern string) *symbolicReferenceMachine {
	expression, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		panic(fmt.Sprintf("invalid internal image-reference grammar: %v", err))
	}
	program, err := syntax.Compile(expression.Simplify())
	if err != nil {
		panic(fmt.Sprintf("cannot compile internal image-reference grammar: %v", err))
	}

	targetCapture := 0
	for capture, name := range expression.CapNames() {
		if name == "reserved" {
			targetCapture = capture
		}
	}
	if targetCapture == 0 {
		panic("internal image-reference grammar lacks its reserved component capture")
	}
	type nfaState struct {
		pc           uint32
		insideTarget bool
		matchedFixed bool
	}
	type dfaState struct {
		nfaStates []nfaState
	}
	encodeNFAState := func(state nfaState) uint64 {
		encoded := uint64(state.pc) << 2
		if state.insideTarget {
			encoded |= 2
		}
		if state.matchedFixed {
			encoded |= 1
		}
		return encoded
	}
	closure := func(seed []nfaState) []nfaState {
		seen := make(map[uint64]bool, len(seed))
		pending := append([]nfaState(nil), seed...)
		result := []nfaState{}
		for len(pending) > 0 {
			state := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			encoded := encodeNFAState(state)
			if seen[encoded] {
				continue
			}
			seen[encoded] = true
			instruction := program.Inst[state.pc]
			switch instruction.Op {
			case syntax.InstAlt, syntax.InstAltMatch:
				left, right := state, state
				left.pc, right.pc = instruction.Out, instruction.Arg
				pending = append(pending, left, right)
			case syntax.InstCapture:
				state.pc = instruction.Out
				if instruction.Arg == uint32(2*targetCapture) {
					state.insideTarget = true
				} else if instruction.Arg == uint32(2*targetCapture+1) {
					state.insideTarget = false
				}
				pending = append(pending, state)
			case syntax.InstNop, syntax.InstEmptyWidth:
				state.pc = instruction.Out
				pending = append(pending, state)
			default:
				result = append(result, state)
			}
		}
		sort.Slice(result, func(left, right int) bool {
			return encodeNFAState(result[left]) < encodeNFAState(result[right])
		})
		return result
	}
	key := func(states []nfaState) string {
		encoded := make([]uint64, len(states))
		for index, state := range states {
			encoded[index] = encodeNFAState(state)
		}
		return fmt.Sprint(encoded)
	}
	states := []dfaState{{nfaStates: closure([]nfaState{{pc: uint32(program.Start)}})}}
	stateIndexes := map[string]int{key(states[0].nfaStates): 0}
	literalTransitions := [][]int{}
	variableTransitions := [][]int{}
	accepting := []bool{}
	for stateIndex := 0; stateIndex < len(states); stateIndex++ {
		state := states[stateIndex]
		literalRow := make([]int, 128)
		variableRow := make([]int, 128)
		isAccepting := false
		for _, nfaState := range state.nfaStates {
			isAccepting = isAccepting || nfaState.matchedFixed && program.Inst[nfaState.pc].Op == syntax.InstMatch
		}
		accepting = append(accepting, isAccepting)
		for character := 0; character < len(literalRow); character++ {
			literalSeed := []nfaState{}
			variableSeed := []nfaState{}
			for _, nfaState := range state.nfaStates {
				instruction := program.Inst[nfaState.pc]
				if (instruction.Op == syntax.InstRune || instruction.Op == syntax.InstRune1 || instruction.Op == syntax.InstRuneAny || instruction.Op == syntax.InstRuneAnyNotNL) && instruction.MatchRune(rune(character)) {
					nextState := nfaState
					nextState.pc = instruction.Out
					variableSeed = append(variableSeed, nextState)
					nextState.matchedFixed = nextState.matchedFixed || nextState.insideTarget
					literalSeed = append(literalSeed, nextState)
				}
			}
			for mode, next := range [][]nfaState{closure(literalSeed), closure(variableSeed)} {
				nextKey := key(next)
				nextIndex, found := stateIndexes[nextKey]
				if !found {
					nextIndex = len(states)
					if nextIndex >= maxReferenceMachineStates {
						panic("internal image-reference grammar exceeds its state budget")
					}
					stateIndexes[nextKey] = nextIndex
					states = append(states, dfaState{nfaStates: next})
				}
				if mode == 0 {
					literalRow[character] = nextIndex
				} else {
					variableRow[character] = nextIndex
				}
			}
		}
		literalTransitions = append(literalTransitions, literalRow)
		variableTransitions = append(variableTransitions, variableRow)
	}

	machine := &symbolicReferenceMachine{
		literalTransitions:  literalTransitions,
		variableTransitions: variableTransitions,
		accepting:           accepting,
	}
	machine.transformations = machine.transitionMonoid()
	return machine
}

func (machine *symbolicReferenceMachine) transitionMonoid() [][]int {
	identity := make([]int, len(machine.variableTransitions))
	for index := range identity {
		identity[index] = index
	}
	transformationKey := func(transformation []int) string {
		encoded := make([]byte, len(transformation)*2)
		for index, state := range transformation {
			encoded[2*index] = byte(state)
			encoded[2*index+1] = byte(state >> 8)
		}
		return string(encoded)
	}
	generators := [][]int{}
	seenGenerators := map[string]bool{}
	for character := 0; character < 128; character++ {
		generator := make([]int, len(machine.variableTransitions))
		for state := range generator {
			generator[state] = machine.variableTransitions[state][character]
		}
		generatorKey := transformationKey(generator)
		if !seenGenerators[generatorKey] {
			seenGenerators[generatorKey] = true
			generators = append(generators, generator)
		}
	}
	transformations := [][]int{identity}
	seen := map[string]bool{transformationKey(identity): true}
	for index := 0; index < len(transformations); index++ {
		current := transformations[index]
		for _, generator := range generators {
			next := make([]int, len(current))
			for state := range current {
				next[state] = generator[current[state]]
			}
			nextKey := transformationKey(next)
			if !seen[nextKey] {
				seen[nextKey] = true
				if len(transformations) >= maxReferenceTransformations {
					panic("internal image-reference grammar exceeds its transformation budget")
				}
				transformations = append(transformations, next)
			}
		}
	}
	return transformations
}

func (machine *symbolicReferenceMachine) matches(value symbolicValue, workLimit int) (bool, int, error) {
	variableIndexes := map[string]int{}
	variableOccurrences := map[string]int{}
	for _, segment := range value.segments {
		if segment.variable != "" {
			variableOccurrences[segment.variable]++
			if _, found := variableIndexes[segment.variable]; !found {
				variableIndexes[segment.variable] = len(variableIndexes)
			}
		}
	}
	if len(variableIndexes) > maxDockerWordVariables {
		return false, 0, resourceLimitf("Docker metadata symbolic value references more than %d variable identities", maxDockerWordVariables)
	}
	assignments := make([]int, len(variableIndexes))
	for index := range assignments {
		assignments[index] = -1
	}
	type searchState struct {
		segment     int
		state       int
		assignments [maxDockerWordVariables]int
	}
	failed := map[searchState]bool{}
	work := 0
	exhausted := false
	var visit func(int, int) bool
	visit = func(segmentIndex, state int) (matched bool) {
		memoKey := searchState{segment: segmentIndex, state: state}
		copy(memoKey.assignments[:], assignments)
		if failed[memoKey] {
			return false
		}
		defer func() {
			if !matched && !exhausted {
				failed[memoKey] = true
			}
		}()
		if work >= workLimit {
			exhausted = true
			return false
		}
		work++
		if segmentIndex == len(value.segments) {
			return machine.accepting[state]
		}
		segment := value.segments[segmentIndex]
		if segment.variable == "" {
			for index := 0; index < len(segment.literal); index++ {
				character := segment.literal[index]
				if character >= 128 {
					return false
				}
				state = machine.literalTransitions[state][character]
			}
			return visit(segmentIndex+1, state)
		}
		variableIndex := variableIndexes[segment.variable]
		if variableOccurrences[segment.variable] == 1 {
			seenStates := make([]bool, len(machine.variableTransitions))
			for _, transformation := range machine.transformations {
				nextState := transformation[state]
				if seenStates[nextState] {
					continue
				}
				seenStates[nextState] = true
				if visit(segmentIndex+1, nextState) {
					return true
				}
			}
			return false
		}
		if transformationIndex := assignments[variableIndex]; transformationIndex >= 0 {
			return visit(segmentIndex+1, machine.transformations[transformationIndex][state])
		}
		for transformationIndex, transformation := range machine.transformations {
			assignments[variableIndex] = transformationIndex
			if visit(segmentIndex+1, transformation[state]) {
				assignments[variableIndex] = -1
				return true
			}
		}
		assignments[variableIndex] = -1
		return false
	}
	matched := visit(0, 0)
	if exhausted {
		return false, work, resourceLimitf("Docker metadata exceeds the %d-step symbolic-reference search limit", maxSymbolicReferenceWork)
	}
	return matched, work, nil
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
	literal, download, scanErr := discovery.scanRuntimeWords(analyzed)
	if scanErr != nil {
		return nil, scanErr
	}
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

func (discovery *dockerDiscovery) scanRuntimeWords(words []runtimeWord) (bool, bool, error) {
	literal := false
	download := false
	for _, word := range words {
		if word.symbolic != nil {
			result := discovery.symbolicSource(word.symbolic, true)
			if result.err != nil {
				return false, false, result.err
			}
			literal = literal || result.image
			download = download || result.download
		} else {
			literal = literal || containsDockerGoToken(word.value)
			download = download || goDownloadPattern.MatchString(word.value)
		}
	}
	return literal, download, nil
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
