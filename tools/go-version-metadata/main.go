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
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
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

var goDownloadPattern = regexp.MustCompile(`(?i)(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var exactToolchainVersionPattern = regexp.MustCompile(`^1\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)
var canonicalDockerGoImagePattern = regexp.MustCompile(`^(?i:FROM)[ \t]+golang:((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))(-[a-z0-9][a-z0-9._-]*)?@sha256:([0-9a-f]{64})[ \t]+(?i:AS)[ \t]+([a-z0-9][a-z0-9_.-]*)[ \t]*$`)

const (
	maxInputBytes           = 4 << 20
	maxRequestEnvelopeBytes = 32 << 20
	maxYAMLDocuments        = 64
	maxYAMLNodes            = 100000
	maxYAMLEdges            = 150000
	maxYAMLDepth            = 256
	maxYAMLScalarBytes      = 1 << 20
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
	if node.Kind == yaml.ScalarNode && goDownloadPattern.MatchString(node.Value) {
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
	for _, node := range parsed.AST.Children {
		classifyDockerInstruction(node, true, &managed, &unsupported)
	}
	if len(unsupported) != 0 || len(managed) > 1 {
		return "unsupported", append(managed, unsupported...), ""
	}
	if len(managed) == 1 {
		return "managed", managed, ""
	}
	return "irrelevant", nil, ""
}

func classifyDockerInstruction(node *parser.Node, allowManaged bool, managed *[]dockerCandidate, unsupported *[]dockerCandidate) {
	original := strings.TrimSpace(node.Original)
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
		*unsupported = append(*unsupported, dockerInstructionCandidates(node)...)
	}
	for _, child := range node.Children {
		classifyDockerInstruction(child, false, managed, unsupported)
	}
	for value := node.Next; value != nil; value = value.Next {
		for _, child := range value.Children {
			classifyDockerInstruction(child, false, managed, unsupported)
		}
	}
}

func dockerInstructionCandidates(node *parser.Node) []dockerCandidate {
	candidates := []dockerCandidate{}
	appendImageCandidate := func(kind, value string) {
		if isGolangImage(value) || containsDockerGoToken(value) {
			candidates = append(candidates, dockerCandidate{Kind: kind, Value: value, Line: node.StartLine})
		}
	}

	for value := node.Next; value != nil; value = value.Next {
		if goDownloadPattern.MatchString(value.Value) {
			candidates = append(candidates, dockerCandidate{Kind: "download", Value: value.Value, Line: node.StartLine})
			break
		}
	}
	for _, heredoc := range node.Heredocs {
		if goDownloadPattern.MatchString(heredoc.Content) {
			candidates = append(candidates, dockerCandidate{Kind: "download", Value: heredoc.Content, Line: node.StartLine})
		}
	}
	switch strings.ToLower(node.Value) {
	case "from":
		if node.Next != nil {
			appendImageCandidate("from", node.Next.Value)
		}
	case "arg":
		if hasDockerGoLiteral(node) {
			candidates = append(candidates, dockerCandidate{
				Kind: "arg-default", Value: node.Next.Value, Line: node.StartLine,
			})
		}
	case "copy":
		for _, flag := range node.Flags {
			if value, found := strings.CutPrefix(flag, "--from="); found {
				appendImageCandidate("copy-from", value)
			}
		}
	case "run":
		for _, flag := range node.Flags {
			mount, found := strings.CutPrefix(flag, "--mount=")
			if !found {
				continue
			}
			for _, field := range strings.Split(mount, ",") {
				if value, found := strings.CutPrefix(field, "from="); found {
					appendImageCandidate("run-mount-from", value)
				}
			}
		}
	}
	if strings.EqualFold(node.Value, "from") {
		return candidates
	}
	if len(candidates) == 0 && hasDockerGoLiteral(node) {
		candidates = append(candidates, dockerCandidate{
			Kind: "literal", Value: node.Original, Line: node.StartLine,
		})
	}
	return candidates
}

func hasDockerGoLiteral(node *parser.Node) bool {
	for value := node.Next; value != nil; value = value.Next {
		if containsDockerGoToken(value.Value) {
			return true
		}
	}
	for _, heredoc := range node.Heredocs {
		if containsDockerGoToken(heredoc.Content) {
			return true
		}
	}
	return false
}

func containsDockerGoToken(value string) bool {
	found, _ := scanDockerGoToken(value)
	return found
}

func scanDockerGoToken(value string) (bool, int) {
	steps := 0
	for index := 0; index < len(value); index++ {
		steps++
		if value[index] == '$' && index+1 < len(value) && value[index+1] == '{' {
			wordStart, inspected, ok := parameterExpansionWordStart(value, index+2)
			steps += inspected
			if ok && hasDockerGoTokenAt(value, wordStart) && hasDockerGoTokenEnd(value, wordStart) {
				return true, steps
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

func parameterExpansionWordStart(value string, start int) (int, int, bool) {
	cursor := start
	inspected := 0
	if cursor >= len(value) {
		return 0, inspected, false
	}
	if isShellNameStart(value[cursor]) {
		for cursor < len(value) && isShellNameByte(value[cursor]) {
			cursor++
			inspected++
		}
	} else if isASCIIDigit(value[cursor]) {
		for cursor < len(value) && isASCIIDigit(value[cursor]) {
			cursor++
			inspected++
		}
	} else if isShellSpecialParameter(value[cursor]) {
		cursor++
		inspected++
	} else {
		return 0, inspected + 1, false
	}
	if cursor < len(value) && value[cursor] == ':' {
		cursor++
		inspected++
	}
	if cursor >= len(value) || !isShellParameterOperator(value[cursor]) {
		return 0, inspected + 1, false
	}
	return cursor + 1, inspected + 1, true
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

func isShellNameStart(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value == '_'
}

func isShellNameByte(value byte) bool {
	return isShellNameStart(value) || isASCIIDigit(value)
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isShellSpecialParameter(value byte) bool {
	return strings.ContainsRune("*@#?-$!", rune(value))
}

func isShellParameterOperator(value byte) bool {
	return value == '-' || value == '+' || value == '?' || value == '='
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
