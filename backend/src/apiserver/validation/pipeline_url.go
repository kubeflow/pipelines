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
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
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

// Blocked CIDRs - private/loopback/link-local/metadata
var blockedCIDRs = []string{
	// Loopback
	"127.0.0.0/8",
	"::1/128",

	// Private
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",

	// Link-local
	"169.254.0.0/16",
	"fe80::/10",

	// Metadata
	"169.254.169.254/32",
	"fd00:ec2::254/128",

	// Invalid
	"0.0.0.0/8",
	"::/128",
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

// lookupIPAddr resolves a host to its IP addresses. It is a package-level
// variable (rather than a direct net.DefaultResolver.LookupIPAddr call) so
// tests can substitute a fake resolver instead of performing real DNS
// lookups.
var lookupIPAddr = net.DefaultResolver.LookupIPAddr

// initURLConfig initializes blocked networks and allowed domains
func initURLConfig() {
	urlConfigInit.Do(func() {
		// Parse CIDR strings into IPNet objects
		for _, cidr := range blockedCIDRs {
			if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
				blockedNets = append(blockedNets, ipNet)
			} else {
				glog.Warningf("Failed to parse blocked CIDR %q: %v", cidr, err)
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

		// Compile domain patterns into regex
		for _, domain := range allDomains {
			pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(domain), "\\*", ".*") + "$"
			if re, err := regexp.Compile(pattern); err == nil {
				allowedDomains = append(allowedDomains, re)
			} else {
				glog.Warningf("Failed to compile domain pattern %q: %v", pattern, err)
			}
		}
	})
}

// ValidatePipelineURL validates a pipeline URL is safe to fetch
// Can be disabled via PIPELINE_URL_VALIDATION_ENABLED=false (for testing purposes only)
func ValidatePipelineURL(urlStr string) error {
	// Allow disabling validation
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
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || !allowHTTP) {
		return util.NewInvalidInputError("URL scheme %q not allowed, use https", parsed.Scheme)
	}

	// Validate port
	port := parsed.Port()
	if port != "" && port != defaultPorts[parsed.Scheme] {
		return util.NewInvalidInputError(
			"%s requires port %s, got %q",
			strings.ToUpper(parsed.Scheme),
			defaultPorts[parsed.Scheme],
			port,
		)
	}

	// Validate domain against allowlist
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
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

const dnsLookupTimeout = 5 * time.Second

func validateResolvedIPs(host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dnsLookupTimeout)
	defer cancel()

	addrs, err := lookupIPAddr(ctx, host)
	if err != nil {
		return util.NewInvalidInputError("DNS resolution failed for %q: %v", host, err)
	}

	for _, addr := range addrs {
		if isBlockedIP(addr.IP) {
			return util.NewInvalidInputError("URL resolves to blocked IP range: %v", addr.IP)
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

// SafePipelineHTTPClient creates an HTTP client with timeout, redirect validation,
// and a custom dialer that enforces IP validation at connection time to prevent
// DNS rebinding (TOCTOU) attacks.
const minimumHTTPTimeout = 1

func SafePipelineHTTPClient() *http.Client {
	timeoutSeconds := common.GetIntConfigWithDefault(common.PipelineURLTimeout, 30)
	if timeoutSeconds < minimumHTTPTimeout {
		timeoutSeconds = 30
	}
	timeout := time.Duration(timeoutSeconds) * time.Second

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDialContext

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return util.NewInvalidInputError("too many redirects")
			}
			return ValidatePipelineURL(req.URL.String())
		},
	}
}

// safeDialContext resolves the host, validates all IPs against blocked ranges,
// and connects only to a validated IP. This closes the DNS TOCTOU gap where
// a host could resolve to a safe IP during validation but a blocked IP at
// connection time (DNS rebinding).
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %v", addr, err)
	}

	initURLConfig()

	addrs, err := lookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %q: %v", host, err)
	}

	for _, a := range addrs {
		if isBlockedIP(a.IP) {
			return nil, util.NewInvalidInputError("connection to blocked IP range denied")
		}
	}

	// Connect using the resolved IPs directly
	var dialer net.Dialer
	for _, a := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(a.IP.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("failed to connect to any resolved address for %q", host)
}
