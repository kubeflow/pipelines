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

package validation

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"golang.org/x/net/http/httpguts"
)

const (
	credentialTypeBearer = "bearer"
	credentialTypeBasic  = "basic"
	credentialTypeHeader = "header"

	supportedCredentialFields = "domain, type, token, username, password, headerName, headerValue"
)

// pipelineURLCredential is read from the environment variables
// PIPELINE_URL_CREDENTIAL_<name>_<field>, which envFrom with a prefix creates
// from the keys of one Secret.
type pipelineURLCredential struct {
	name        string
	domain      string
	authType    string
	token       string
	username    string
	password    string
	headerName  string
	headerValue string
}

// String omits secret values so a credential is safe to log.
func (c pipelineURLCredential) String() string {
	return fmt.Sprintf("{name: %s, domain: %s, type: %s}", c.name, c.domain, c.authType)
}

// setField matches field names case-insensitively and reports whether the
// field is supported.
func (c *pipelineURLCredential) setField(field, value string) bool {
	switch strings.ToLower(field) {
	case "domain":
		c.domain = value
	case "type":
		c.authType = value
	case "token":
		c.token = value
	case "username":
		c.username = value
	case "password":
		c.password = value
	case "headername":
		c.headerName = value
	case "headervalue":
		c.headerValue = value
	default:
		return false
	}
	return true
}

func (c pipelineURLCredential) validate() error {
	if c.domain == "" {
		return errors.New("domain is required")
	}
	if strings.ContainsAny(c.domain, "/:*@?#[] \t") {
		return fmt.Errorf("domain %q must be a bare hostname without a scheme, port, path, or wildcard", c.domain)
	}
	switch c.authType {
	case credentialTypeBearer:
		if c.token == "" || !httpguts.ValidHeaderFieldValue(c.token) {
			return errors.New("type bearer requires a token without control characters")
		}
	case credentialTypeBasic:
		if c.username == "" || c.password == "" {
			return errors.New("type basic requires username and password")
		}
		if strings.Contains(c.username, ":") {
			return errors.New("basic auth username must not contain a colon")
		}
	case credentialTypeHeader:
		if !httpguts.ValidHeaderFieldName(c.headerName) {
			return errors.New("type header requires a valid headerName")
		}
		if c.headerValue == "" || !httpguts.ValidHeaderFieldValue(c.headerValue) {
			return errors.New("type header requires a headerValue without control characters")
		}
	case "":
		return errors.New("type is required; use bearer, basic, or header")
	default:
		return fmt.Errorf("type %q is not supported; use bearer, basic, or header", c.authType)
	}
	return nil
}

func (c pipelineURLCredential) apply(request *http.Request) {
	switch c.authType {
	case credentialTypeBearer:
		request.Header.Set("Authorization", "Bearer "+c.token)
	case credentialTypeBasic:
		request.SetBasicAuth(c.username, c.password)
	case credentialTypeHeader:
		request.Header.Set(c.headerName, c.headerValue)
	}
}

type pipelineURLCredentials struct {
	byDomain map[string]pipelineURLCredential
}

func (c *pipelineURLCredentials) domains() []string {
	return slices.Sorted(maps.Keys(c.byDomain))
}

func (c *pipelineURLCredentials) forURL(target *url.URL) (pipelineURLCredential, bool) {
	if target.Scheme != "https" {
		return pipelineURLCredential{}, false
	}
	host, err := normalizeHost(target.Hostname())
	if err != nil {
		return pipelineURLCredential{}, false
	}
	credential, found := c.byDomain[host]
	return credential, found
}

// parsePipelineURLCredentials groups PIPELINE_URL_CREDENTIAL_<name>_<field>
// entries of environ by name. Invalid credentials are skipped and described in
// the returned problems, which never include credential values.
func parsePipelineURLCredentials(environ []string, domainAllowed func(string) bool) (*pipelineURLCredentials, []error) {
	var problems []error
	fieldsByName := map[string]map[string]string{}
	repeatedFields := map[string][]string{}
	for _, entry := range environ {
		key, value, _ := strings.Cut(entry, "=")
		suffix, found := strings.CutPrefix(key, common.PipelineURLCredentialEnvPrefix)
		if !found {
			continue
		}
		separator := strings.LastIndex(suffix, "_")
		if separator <= 0 || separator == len(suffix)-1 {
			problems = append(problems, fmt.Errorf("ignoring environment variable %s: name it %s<NAME>_<field>", key, common.PipelineURLCredentialEnvPrefix))
			continue
		}
		name, field := suffix[:separator], suffix[separator+1:]
		if fieldsByName[name] == nil {
			fieldsByName[name] = map[string]string{}
		}
		for existing := range fieldsByName[name] {
			if strings.EqualFold(existing, field) {
				repeatedFields[name] = append(repeatedFields[name], field)
			}
		}
		// Secrets created from files often end with a newline.
		fieldsByName[name][field] = strings.TrimRight(value, "\r\n")
	}

	matchesByDomain := map[string][]pipelineURLCredential{}
	for _, name := range slices.Sorted(maps.Keys(fieldsByName)) {
		if repeated := repeatedFields[name]; len(repeated) > 0 {
			problems = append(problems, fmt.Errorf("ignoring credential %s: field %s is set more than once; set each field once", name, strings.Join(repeated, ", ")))
			continue
		}
		credential := pipelineURLCredential{name: name}
		for _, field := range slices.Sorted(maps.Keys(fieldsByName[name])) {
			if !credential.setField(field, fieldsByName[name][field]) {
				problems = append(problems, fmt.Errorf("credential %s: ignoring unknown field %q; supported fields are %s", name, field, supportedCredentialFields))
			}
		}
		domain, err := normalizeHost(credential.domain)
		if err != nil {
			problems = append(problems, fmt.Errorf("ignoring credential %s: domain %w", name, err))
			continue
		}
		credential.domain = domain
		credential.authType = strings.ToLower(credential.authType)
		if err := credential.validate(); err != nil {
			problems = append(problems, fmt.Errorf("ignoring credential %s: %w", name, err))
			continue
		}
		if !domainAllowed(credential.domain) {
			problems = append(problems, fmt.Errorf("ignoring credential %s: domain %q is not allowed; add it to %s", name, credential.domain, common.PipelineURLAllowedDomains))
			continue
		}
		matchesByDomain[credential.domain] = append(matchesByDomain[credential.domain], credential)
	}

	credentials := &pipelineURLCredentials{byDomain: map[string]pipelineURLCredential{}}
	for _, domain := range slices.Sorted(maps.Keys(matchesByDomain)) {
		matches := matchesByDomain[domain]
		if len(matches) > 1 {
			names := make([]string, 0, len(matches))
			for _, match := range matches {
				names = append(names, match.name)
			}
			problems = append(problems, fmt.Errorf("ignoring credentials %s: they share domain %q; keep one credential per domain", strings.Join(names, ", "), domain))
			continue
		}
		credentials.byDomain[domain] = matches[0]
	}
	return credentials, problems
}

// credentialDomainAllowed applies the pipeline URL allowlist to a credential
// domain. Like the URL checks, it is bypassed when validation is disabled.
func credentialDomainAllowed(domain string) bool {
	if !common.GetBoolConfigWithDefault(common.PipelineURLValidationEnabled, true) {
		return true
	}
	initURLConfig()
	return isDomainAllowed(domain)
}

// withPipelineURLCredentials returns transport unchanged when no valid
// credentials are configured. Invalid credentials are logged and skipped, so a
// misconfiguration never blocks startup; downloads from their domains are sent
// without credentials.
func withPipelineURLCredentials(transport *http.Transport) http.RoundTripper {
	credentials, problems := parsePipelineURLCredentials(os.Environ(), credentialDomainAllowed)
	for _, problem := range problems {
		glog.Warningf("Pipeline URL credentials: %v", problem)
	}
	if len(credentials.byDomain) == 0 {
		return transport
	}
	glog.Infof("Pipeline URL downloads send credentials to: %s", strings.Join(credentials.domains(), ", "))
	return &credentialTransport{base: transport, credentials: credentials}
}

// credentialTransport adds the configured credential to HTTPS requests for its
// domain. It runs on every redirect hop, so a credential is never sent to a
// host it was not configured for.
type credentialTransport struct {
	base        http.RoundTripper
	credentials *pipelineURLCredentials
}

func (t *credentialTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	credential, found := t.credentials.forURL(request.URL)
	if !found {
		return t.base.RoundTrip(request)
	}
	authenticated := request.Clone(request.Context())
	credential.apply(authenticated)
	return t.base.RoundTrip(authenticated)
}
