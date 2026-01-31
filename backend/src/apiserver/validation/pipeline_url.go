// Copyright 2025 The Kubeflow Authors
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
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Default allowed domains for pipeline URLs
var defaultAllowedDomains = []string{
	"storage.googleapis.com",
	"s3.amazonaws.com",
	"github.com",
	"raw.githubusercontent.com",
}

// Blocked CIDRs - private/loopback/link-local/metadata (non-configurable)
var blockedCIDRs = []string{
	// Loopback
	"127.0.0.0/8",
	"::1/128",

	// Private IPv4
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",

	// Private IPv6 (ULA)
	"fc00::/7",

	// Link-local
	"169.254.0.0/16",
	"fe80::/10",

	// Cloud metadata (explicit)
	"169.254.169.254/32",
	"169.254.170.2/32",   // AWS ECS
	"100.100.100.200/32", // Alibaba

	// Reserved / special
	"0.0.0.0/8",
	"224.0.0.0/4",
	"::/128",
	"ff00::/8",
}

var defaultPorts = map[string]string{
	"https": "443",
	"http":  "80",
}

var (
	blockedNets    []*net.IPNet
	allowedDomains []*regexp.Regexp
	urlConfigInit  sync.Once
)

// initURLConfig initializes blocked networks and allowed domains (called once)
func initURLConfig() {
	urlConfigInit.Do(func() {
		// Parse blocked CIDRs
		for _, cidr := range blockedCIDRs {
			if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
				blockedNets = append(blockedNets, ipNet)
			}
		}

		// Build allowed domains: defaults + user-configured
		allDomains := append([]string{}, defaultAllowedDomains...)
		userDomains := common.GetStringConfigWithDefault(common.PipelineURLAllowedDomains, "")
		if userDomains != "" {
			for _, d := range strings.Split(userDomains, ",") {
				if d = strings.TrimSpace(d); d != "" {
					allDomains = append(allDomains, d)
				}
			}
		}

		// Compile domain patterns (support wildcards like *.s3.amazonaws.com)
		for _, domain := range allDomains {
			pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(domain), "\\*", ".*") + "$"
			if re, err := regexp.Compile(pattern); err == nil {
				allowedDomains = append(allowedDomains, re)
			}
		}
	})
}

// ValidatePipelineURL validates a pipeline URL is safe to fetch.
// Returns InvalidInputError if validation fails.
// Can be disabled via PIPELINE_URL_VALIDATION_ENABLED=false (for testing).
func ValidatePipelineURL(urlStr string) error {
	// Allow disabling validation (primarily for tests)
	if !common.GetBoolConfigWithDefault(common.PipelineURLValidationEnabled, true) {
		return nil
	}

	initURLConfig()

	parsed, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return util.NewInvalidInputError("invalid URL: %v", err)
	}

	// Validate scheme
	allowHTTP := common.GetBoolConfigWithDefault(common.PipelineURLAllowHTTP, false)
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return util.NewInvalidInputError("URL scheme %q not allowed, use https", parsed.Scheme)
	}

	// Validate port

	port := parsed.Port()
	if port == "" {
		port = defaultPorts[parsed.Scheme]
	}

	if expected, ok := defaultPorts[parsed.Scheme]; ok && port != expected {
		return util.NewInvalidInputError(
			"%s requires port %s, got %q",
			strings.ToUpper(parsed.Scheme),
			expected,
			port,
		)
	}

	// Validate domain against allowlist
	host := parsed.Hostname()
	if !isDomainAllowed(host) {
		return util.NewInvalidInputError("URL domain %q not in allowlist", host)
	}

	// Resolve DNS and validate IPs are not in blocked ranges
	if err := validateResolvedIPs(host); err != nil {
		return err
	}

	return nil
}

func isDomainAllowed(host string) bool {
	for _, re := range allowedDomains {
		if re.MatchString(host) {
			return true
		}
	}
	return false
}

func validateResolvedIPs(host string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		return util.NewInvalidInputError("DNS resolution failed for %q: %v", host, err)
	}

	for _, ip := range ips {
		if isBlockedIP(ip) {
			return util.NewInvalidInputError("URL resolves to blocked IP range: %v", ip)
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	for _, ipNet := range blockedNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// SafePipelineHTTPClient creates an HTTP client with timeout and redirect validation.
func SafePipelineHTTPClient() *http.Client {
	initURLConfig()
	timeout := time.Duration(common.GetIntConfigWithDefault(common.PipelineURLTimeout, 30)) * time.Second

	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return util.NewInvalidInputError("too many redirects")
			}
			return ValidatePipelineURL(req.URL.String())
		},
	}
}
