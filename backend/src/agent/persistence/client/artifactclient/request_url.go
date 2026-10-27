// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package artifactclient

import (
	"fmt"
	"net/url"
	"strings"
)

func validateArtifactSegment(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\%?#") {
		return fmt.Errorf("artifact route identifiers must be nonempty single path segments without encoding or delimiters")
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("artifact route identifiers must not contain control characters")
		}
	}
	return nil
}

func artifactRequestURL(base string, request *ReadArtifactRequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("artifact request is nil")
	}
	for _, value := range []string{request.RunID, request.NodeID, request.ArtifactName} {
		if err := validateArtifactSegment(value); err != nil {
			return "", err
		}
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid artifact API base URL")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(base, "#") {
		return "", fmt.Errorf("artifact API base must be an HTTP(S) URL without credentials, query or fragment")
	}
	prefix := strings.TrimSuffix(parsed.Path, "/")
	if prefix != "" {
		for _, segment := range strings.Split(strings.TrimPrefix(prefix, "/"), "/") {
			if err := validateArtifactSegment(segment); err != nil {
				return "", fmt.Errorf("artifact API base path must contain normalized segments")
			}
		}
	}
	parsed.Path = prefix + "/apis/v1beta1/runs/" + request.RunID + "/nodes/" + request.NodeID + "/artifacts/" + request.ArtifactName + ":read"
	// URL.String escapes the validated path without interpreting identifiers as URLs.
	parsed.RawPath = ""
	return parsed.String(), nil
}
