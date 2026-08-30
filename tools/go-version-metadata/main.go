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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
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

type response struct {
	YAMLValues           map[string][]string `json:"yamlValues,omitempty"`
	HasGoDownload        bool                `json:"hasGoDownload,omitempty"`
	DockerGoStages       []string            `json:"dockerGoStages,omitempty"`
	DockerGoSources      []string            `json:"dockerGoSources,omitempty"`
	DockerRepositoryArgs []string            `json:"dockerRepositoryArgs,omitempty"`
	Module               *moduleMetadata     `json:"module,omitempty"`
}

var goDownloadPattern = regexp.MustCompile(`(?i)(?:dl\.google\.com/go/|go\.dev/dl/)go`)
var exactToolchainVersionPattern = regexp.MustCompile(`^1\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)

const maxYAMLNodes = 100000

func main() {
	var input request
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
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
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

func inspect(input request) (response, error) {
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
		stages, sources, arguments, err := inspectDockerfile(input.Contents)
		if err != nil {
			return response{}, fmt.Errorf("%s: %w", input.Path, err)
		}
		metadata.DockerGoStages = stages
		metadata.DockerGoSources = sources
		metadata.DockerRepositoryArgs = arguments
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
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false, err
		}
		visited := map[*yaml.Node]bool{}
		if err := walkYAML(&document, values, &hasDownload, visited); err != nil {
			return nil, false, err
		}
	}
	return values, hasDownload, nil
}

func walkYAML(node *yaml.Node, values map[string][]string, hasDownload *bool, visited map[*yaml.Node]bool) error {
	if node == nil || visited[node] {
		return nil
	}
	visited[node] = true
	if len(visited) > maxYAMLNodes {
		return fmt.Errorf("YAML metadata exceeds the %d-node traversal limit", maxYAMLNodes)
	}

	if node.Kind == yaml.AliasNode {
		return walkYAML(node.Alias, values, hasDownload, visited)
	}
	if node.Kind == yaml.ScalarNode && goDownloadPattern.MatchString(node.Value) {
		*hasDownload = true
	}
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := resolvedScalar(node.Content[index])
			valueNode := node.Content[index+1]
			if _, ok := values[key]; ok {
				if value := resolvedScalar(valueNode); value != "" {
					values[key] = append(values[key], value)
				}
			}
			if err := walkYAML(valueNode, values, hasDownload, visited); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := walkYAML(child, values, hasDownload, visited); err != nil {
			return err
		}
	}
	return nil
}

func resolvedScalar(node *yaml.Node) string {
	seen := map[*yaml.Node]bool{}
	for node != nil && node.Kind == yaml.AliasNode {
		if seen[node] {
			return ""
		}
		seen[node] = true
		node = node.Alias
	}
	if node != nil && node.Kind == yaml.ScalarNode {
		return strings.TrimSpace(node.Value)
	}
	return ""
}

func inspectDockerfile(contents string) ([]string, []string, []string, error) {
	parsed, err := parser.Parse(strings.NewReader(contents))
	if err != nil {
		return nil, nil, nil, err
	}
	stages, metaArgs, err := instructions.Parse(parsed.AST, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	environment := map[string]string{}
	lexer := shell.NewLex(parsed.EscapeToken)
	for _, command := range metaArgs {
		for _, argument := range command.Args {
			if argument.Value == nil {
				continue
			}
			result, err := lexer.ProcessWordWithMatches(*argument.Value, environmentGetter(environment))
			if err != nil {
				return nil, nil, nil, parser.WithLocation(err, command.Location())
			}
			environment[argument.Key] = result.Result
		}
	}

	repositoryArgs := []string{}
	goStages := []string{}
	goSources := []string{}
	for index, stage := range stages {
		result, err := lexer.ProcessWordWithMatches(stage.BaseName, environmentGetter(environment))
		if err != nil {
			return nil, nil, nil, parser.WithLocation(err, stage.Location)
		}
		_, isPriorStage := instructions.HasStage(stages[:index], result.Result)
		if !isPriorStage && (isGolangImage(result.Result) || isGolangImage(stage.BaseName)) {
			goStages = append(goStages, stage.BaseName)
			for argument := range result.Matched {
				repositoryArgs = appendUnique(repositoryArgs, argument)
			}
		}

		stageEnvironment := cloneEnvironment(environment)
		for _, command := range stage.Commands {
			switch command := command.(type) {
			case *instructions.ArgCommand:
				for _, argument := range command.Args {
					if argument.Value == nil {
						continue
					}
					expanded, err := lexer.ProcessWordWithMatches(*argument.Value, environmentGetter(stageEnvironment))
					if err != nil {
						return nil, nil, nil, parser.WithLocation(err, command.Location())
					}
					stageEnvironment[argument.Key] = expanded.Result
				}
			case *instructions.CopyCommand:
				if command.From != "" {
					source, err := resolveDockerSource(command.From, stageEnvironment, lexer, command.Location())
					if err != nil {
						return nil, nil, nil, err
					}
					if isExternalGolangSource(source, stages, command.From, true) {
						goSources = append(goSources, command.From)
					}
				}
			case *instructions.RunCommand:
				for _, mount := range instructions.GetMounts(command) {
					if mount.From == "" {
						continue
					}
					source, err := resolveDockerSource(mount.From, stageEnvironment, lexer, command.Location())
					if err != nil {
						return nil, nil, nil, err
					}
					if isExternalGolangSource(source, stages, mount.From, false) {
						goSources = append(goSources, mount.From)
					}
				}
			}
		}
	}
	return goStages, goSources, repositoryArgs, nil
}

func cloneEnvironment(environment map[string]string) map[string]string {
	cloned := make(map[string]string, len(environment))
	for key, value := range environment {
		cloned[key] = value
	}
	return cloned
}

func environmentGetter(environment map[string]string) shell.EnvGetter {
	values := make([]string, 0, len(environment))
	for key, value := range environment {
		values = append(values, key+"="+value)
	}
	return shell.EnvsFromSlice(values)
}

func resolveDockerSource(source string, environment map[string]string, lexer *shell.Lex, location []parser.Range) (string, error) {
	result, err := lexer.ProcessWordWithMatches(source, environmentGetter(environment))
	if err != nil {
		return "", parser.WithLocation(err, location)
	}
	return result.Result, nil
}

func isExternalGolangSource(resolved string, stages []instructions.Stage, original string, numericStageAllowed bool) bool {
	if _, isStage := instructions.HasStage(stages, resolved); isStage {
		return false
	}
	if numericStageAllowed {
		if index, err := strconv.Atoi(resolved); err == nil && index >= 0 && index < len(stages) {
			return false
		}
	}
	return isGolangImage(resolved) || isGolangImage(original)
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
