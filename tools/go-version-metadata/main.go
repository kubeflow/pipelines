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
	"unicode"
	"unicode/utf8"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
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
	maxDockerParameterDepth   = 256
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
	discovery := newDockerDiscovery(parsed.EscapeToken)
	for _, node := range parsed.AST.Children {
		if err := classifyDockerInstruction(node, true, 0, discovery, &managed, &unsupported); err != nil {
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

type dockerWordKey struct {
	value string
	json  bool
}

type dockerWordResult struct {
	value string
	err   error
}

type dockerDiscovery struct {
	wordLexer       *shell.Lex
	wordMemo        map[dockerWordKey]dockerWordResult
	runWordMemo     map[dockerWordKey]dockerWordResult
	escapeToken     byte
	instructions    int
	normalizedBytes int
}

func newDockerDiscovery(escapeToken rune) *dockerDiscovery {
	wordLexer := shell.NewLex(escapeToken)
	wordLexer.SkipUnsetEnv = true
	return &dockerDiscovery{
		wordLexer:   wordLexer,
		wordMemo:    map[dockerWordKey]dockerWordResult{},
		runWordMemo: map[dockerWordKey]dockerWordResult{},
		escapeToken: byte(escapeToken),
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
		candidates, err := dockerInstructionCandidates(node, discovery)
		if err != nil {
			return fmt.Errorf("line %d: %w", node.StartLine, err)
		}
		*unsupported = append(*unsupported, candidates...)
	}
	if len(*managed)+len(*unsupported) > maxDockerCandidates {
		return resourceLimitf("Docker metadata exceeds the %d-candidate limit", maxDockerCandidates)
	}
	for _, child := range node.Children {
		if err := classifyDockerInstruction(child, false, depth+1, discovery, managed, unsupported); err != nil {
			return err
		}
	}
	for value := node.Next; value != nil; value = value.Next {
		for _, child := range value.Children {
			if err := classifyDockerInstruction(child, false, depth+1, discovery, managed, unsupported); err != nil {
				return err
			}
		}
	}
	return nil
}

func dockerInstructionCandidates(node *parser.Node, discovery *dockerDiscovery) ([]dockerCandidate, error) {
	candidates := []dockerCandidate{}
	appendImageCandidate := func(kind, value string) error {
		normalized, err := discovery.normalizeDockerWord(value, node.Attributes["json"])
		if err != nil {
			return err
		}
		if isGolangImage(normalized) || containsDockerGoToken(normalized) {
			candidates = append(candidates, dockerCandidate{Kind: kind, Value: value, Line: node.StartLine})
		}
		return nil
	}

	switch strings.ToLower(node.Value) {
	case "from":
		if node.Next != nil {
			if err := appendImageCandidate("from", node.Next.Value); err != nil {
				return nil, err
			}
		}
	case "arg":
		hasLiteral, err := hasDockerGoLiteral(dockerArgDefaults(node), node.Attributes["json"], discovery)
		if err != nil {
			return nil, err
		}
		if hasLiteral {
			candidates = append(candidates, dockerCandidate{
				Kind: "arg-default", Value: node.Original, Line: node.StartLine,
			})
		}
	case "copy":
		for _, flag := range node.Flags {
			if value, found := strings.CutPrefix(flag, "--from="); found {
				if err := appendImageCandidate("copy-from", value); err != nil {
					return nil, err
				}
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
					if err := appendImageCandidate("run-mount-from", value); err != nil {
						return nil, err
					}
				}
			}
		}
		for _, value := range dockerNodeValues(node) {
			normalized, err := discovery.normalizeDockerRunWord(value, node.Attributes["json"])
			if err != nil {
				return nil, err
			}
			if goDownloadPattern.MatchString(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "download", Value: value, Line: node.StartLine})
			}
			if containsDockerGoToken(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "literal", Value: node.Original, Line: node.StartLine})
			}
		}
		for _, heredoc := range node.Heredocs {
			normalized, err := discovery.normalizeDockerRunWord(heredoc.Content, false)
			if err != nil {
				return nil, err
			}
			if goDownloadPattern.MatchString(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "download", Value: heredoc.Content, Line: node.StartLine})
			}
			if containsDockerGoToken(normalized) {
				candidates = append(candidates, dockerCandidate{Kind: "literal", Value: node.Original, Line: node.StartLine})
			}
		}
	}
	return candidates, nil
}

func dockerArgDefaults(node *parser.Node) []string {
	values := []string{}
	for argument := node.Next; argument != nil; argument = argument.Next {
		_, defaultValue, hasDefault := strings.Cut(argument.Value, "=")
		if hasDefault {
			values = append(values, defaultValue)
		}
	}
	return values
}

func dockerNodeValues(node *parser.Node) []string {
	values := []string{}
	for value := node.Next; value != nil; value = value.Next {
		values = append(values, value.Value)
	}
	return values
}

func (discovery *dockerDiscovery) normalizeDockerWord(value string, json bool) (string, error) {
	key := dockerWordKey{value: value, json: json}
	if result, found := discovery.wordMemo[key]; found {
		return result.value, result.err
	}
	if dockerParameterExpansionDepth(value) > maxDockerParameterDepth {
		err := resourceLimitf("Docker word exceeds the %d-level parameter expansion depth limit", maxDockerParameterDepth)
		discovery.wordMemo[key] = dockerWordResult{err: err}
		return "", err
	}
	var normalized string
	var err error
	if json {
		normalized = value
	} else {
		normalized, _, err = discovery.wordLexer.ProcessWord(value, shell.EnvsFromSlice(nil))
		if err != nil {
			err = fmt.Errorf("normalize Docker word %q: %w", value, err)
		}
	}
	if err == nil {
		discovery.normalizedBytes += len(normalized)
		if discovery.normalizedBytes > maxDockerNormalizedBytes {
			err = resourceLimitf("Docker metadata exceeds the %d-byte normalized-word limit", maxDockerNormalizedBytes)
			normalized = ""
		}
	}
	discovery.wordMemo[key] = dockerWordResult{value: normalized, err: err}
	return normalized, err
}

func (discovery *dockerDiscovery) normalizeDockerRunWord(value string, json bool) (string, error) {
	if json {
		return discovery.normalizeDockerWord(value, true)
	}
	key := dockerWordKey{value: value}
	if result, found := discovery.runWordMemo[key]; found {
		return result.value, result.err
	}
	normalized := projectDockerShellWord(value, discovery.escapeToken)
	discovery.normalizedBytes += len(normalized)
	var err error
	if discovery.normalizedBytes > maxDockerNormalizedBytes {
		err = resourceLimitf("Docker metadata exceeds the %d-byte normalized-word limit", maxDockerNormalizedBytes)
		normalized = ""
	}
	discovery.runWordMemo[key] = dockerWordResult{value: normalized, err: err}
	return normalized, err
}

func projectDockerShellWord(value string, escapeToken byte) string {
	var projected strings.Builder
	projected.Grow(len(value))
	quote := byte(0)
	atWordStart := true
	for index := 0; index < len(value); index++ {
		character := value[index]
		switch quote {
		case '\'':
			if character == '\'' {
				quote = 0
			} else {
				projected.WriteByte(character)
			}
		case '"':
			switch {
			case character == '"':
				quote = 0
			case character == escapeToken && index+1 < len(value) &&
				(value[index+1] == '"' || value[index+1] == '$' || value[index+1] == escapeToken):
				index++
				projected.WriteByte(value[index])
			default:
				projected.WriteByte(character)
			}
		default:
			switch {
			case character == '#' && atWordStart:
				return projected.String()
			case character == '\'' || character == '"':
				quote = character
				atWordStart = false
			case character == escapeToken && index+1 < len(value):
				index++
				projected.WriteByte(value[index])
				atWordStart = false
			default:
				projected.WriteByte(character)
				atWordStart = character == ' ' || character == '\t' || character == '\r' || character == '\n'
			}
		}
	}
	return projected.String()
}

func hasDockerGoLiteral(values []string, json bool, discovery *dockerDiscovery) (bool, error) {
	for _, value := range values {
		normalized, err := discovery.normalizeDockerWord(value, json)
		if err != nil {
			return false, err
		}
		if containsDockerGoToken(normalized) {
			return true, nil
		}
	}
	return false, nil
}

func dockerParameterExpansionDepth(value string) int {
	depth := 0
	maximum := 0
	for index := 0; index+1 < len(value); index++ {
		switch {
		case value[index] == '$' && value[index+1] == '{':
			depth++
			if depth > maximum {
				maximum = depth
			}
			index++
		case value[index] == '}' && depth > 0:
			depth--
		}
	}
	return maximum
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
