// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	imageDigestResolveTimeout = 10 * time.Second
	dockerHubRegistry         = "docker.io"
	defaultDockerHubHost      = "registry-1.docker.io"
	defaultDockerHubAuthURL   = "https://auth.docker.io/token"
	defaultDockerHubService   = "registry.docker.io"
)

// ImageDigestResolver resolves a container image reference to a digest-pinned
// reference (e.g. repo@sha256:...). Used when CacheResolveImageDigest is enabled.
type ImageDigestResolver func(image string) (string, error)

// ImageDigestResolveConfig configures registry access for cache image digest resolution.
type ImageDigestResolveConfig struct {
	// DockerConfigPath is a path to a Docker config.json / .dockerconfigjson with
	// registry credentials for private registries. Empty disables credential lookup.
	DockerConfigPath string
	// DockerHubRegistryHost is the registry API host used when an image resolves to
	// docker.io (default: registry-1.docker.io). Useful for mirrors.
	DockerHubRegistryHost string
	// InsecureRegistries is a comma-separated list of registry hosts that should use
	// http:// instead of https:// (e.g. "localhost:5000,registry.local:5000").
	InsecureRegistries string
}

func (c ImageDigestResolveConfig) withDefaults() ImageDigestResolveConfig {
	out := c
	if out.DockerHubRegistryHost == "" {
		out.DockerHubRegistryHost = defaultDockerHubHost
	}
	return out
}

func (c ImageDigestResolveConfig) insecureSet() map[string]struct{} {
	result := map[string]struct{}{}
	for _, host := range strings.Split(c.InsecureRegistries, ",") {
		host = strings.TrimSpace(host)
		if host != "" {
			result[host] = struct{}{}
		}
	}
	return result
}

// resolveImageForCache returns the image string to include in the cache key.
// When CacheResolveImageDigest is false, the original image is returned unchanged.
// When enabled, resolution failures fall back to the original image so the run
// continues (fail-open). Callers should treat that fallback as equivalent to the
// historical mutable-tag fingerprint: cache hits may still occur after a tag is
// overwritten if the registry was unreachable or auth failed.
func resolveImageForCache(opts Options, image string) string {
	if !opts.CacheResolveImageDigest || image == "" {
		return image
	}
	// Dummy images used by Kubernetes platform ops are not registry images.
	if _, isDummy := dummyImages[image]; isDummy {
		return image
	}

	resolver := opts.ImageDigestResolver
	if resolver == nil {
		cfg := opts.CacheImageDigestConfig.withDefaults()
		client, err := newImageDigestHTTPClient(opts.CaCertPath)
		if err != nil {
			glog.Warningf("Failed to build HTTP client for image digest resolution (ca_cert_path=%q); using image reference as-is for cache key: %v", opts.CaCertPath, err)
			return image
		}
		resolver = func(img string) (string, error) {
			return resolveImageDigestHTTP(img, client, cfg)
		}
	}

	resolved, err := resolver(image)
	if err != nil {
		glog.Warningf("Failed to resolve image digest for %q; using image reference as-is for cache key: %v", image, err)
		return image
	}
	if resolved == "" {
		return image
	}
	if resolved != image {
		glog.Infof("Resolved image %q to digest-pinned reference %q for cache key", image, resolved)
	}
	return resolved
}

type parsedImageReference struct {
	registry   string
	repository string
	tag        string
	digest     string
}

func parseImageReference(image string) (parsedImageReference, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return parsedImageReference{}, fmt.Errorf("empty image reference")
	}

	ref := parsedImageReference{tag: "latest"}

	namePart := image
	if at := strings.LastIndex(image, "@"); at >= 0 {
		namePart = image[:at]
		ref.digest = image[at+1:]
		if ref.digest == "" {
			return parsedImageReference{}, fmt.Errorf("empty digest in image reference %q", image)
		}
	}

	tagPart := namePart
	// Tag is after the last ':' that is not part of a registry host:port.
	if colon := strings.LastIndex(namePart, ":"); colon >= 0 {
		maybeHost := namePart[:colon]
		maybeTag := namePart[colon+1:]
		// "ubuntu:22.04" -> tag
		// "localhost:5000/foo" -> no tag (port)
		// "localhost:5000/foo:bar" -> tag bar
		if !strings.Contains(maybeTag, "/") {
			tagPart = maybeHost
			ref.tag = maybeTag
		}
	}

	slash := strings.Index(tagPart, "/")
	switch {
	case slash < 0:
		ref.registry = dockerHubRegistry
		ref.repository = "library/" + tagPart
	default:
		first := tagPart[:slash]
		rest := tagPart[slash+1:]
		if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
			ref.registry = first
			ref.repository = rest
			if !strings.Contains(ref.repository, "/") && ref.registry == dockerHubRegistry {
				ref.repository = "library/" + ref.repository
			}
		} else {
			ref.registry = dockerHubRegistry
			ref.repository = tagPart
		}
	}

	if ref.repository == "" {
		return parsedImageReference{}, fmt.Errorf("invalid image reference %q", image)
	}
	return ref, nil
}

func (r parsedImageReference) digestPinned() string {
	return fmt.Sprintf("%s/%s@%s", r.registry, r.repository, r.digest)
}

func (r parsedImageReference) registryAPIHost(cfg ImageDigestResolveConfig) string {
	if r.registry == dockerHubRegistry {
		return cfg.DockerHubRegistryHost
	}
	return r.registry
}

func (c ImageDigestResolveConfig) schemeFor(host string) string {
	if _, ok := c.insecureSet()[host]; ok {
		return "http"
	}
	return "https"
}

// newImageDigestHTTPClient builds the HTTP client used for registry digest lookups.
// When caCertPath is set, the same custom CA bundle used for MLMD/API TLS is
// trusted for HTTPS registry connections. The default transport is cloned so
// proxy settings (HTTP_PROXY / HTTPS_PROXY / NO_PROXY) are preserved.
func newImageDigestHTTPClient(caCertPath string) (*http.Client, error) {
	client := &http.Client{Timeout: imageDigestResolveTimeout}
	if caCertPath == "" {
		return client, nil
	}
	tlsCfg, err := util.GetTLSConfig(caCertPath)
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = tlsCfg
		client.Transport = transport
	}
	return client, nil
}

type dockerConfigFile struct {
	Auths map[string]dockerConfigAuth `json:"auths"`
}

type dockerConfigAuth struct {
	Auth     string `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func loadDockerConfig(path string) (*dockerConfigFile, error) {
	if path == "" {
		return &dockerConfigFile{Auths: map[string]dockerConfigAuth{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read docker config %q: %w", path, err)
	}
	var cfg dockerConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse docker config %q: %w", path, err)
	}
	if cfg.Auths == nil {
		cfg.Auths = map[string]dockerConfigAuth{}
	}
	return &cfg, nil
}

// credentialsFor looks up dockerconfig credentials for registryHost.
// When includeDockerHubKeys is true (logical image registry is docker.io, possibly
// reached via CACHE_IMAGE_DIGEST_DOCKERHUB_REGISTRY_HOST mirror), also try the
// legacy Docker Hub auth keys such as https://index.docker.io/v1/.
func (c *dockerConfigFile) credentialsFor(registryHost string, includeDockerHubKeys bool) (username, password string, ok bool) {
	if c == nil {
		return "", "", false
	}
	candidates := []string{
		registryHost,
		"https://" + registryHost,
		"http://" + registryHost,
		"https://" + registryHost + "/",
		"http://" + registryHost + "/",
		"https://" + registryHost + "/v2/",
		"http://" + registryHost + "/v2/",
	}
	// Docker Hub stores credentials under the legacy index URL. Also apply when
	// talking to a configured Hub mirror so docker login credentials still work.
	if includeDockerHubKeys || registryHost == defaultDockerHubHost || registryHost == dockerHubRegistry || registryHost == "index.docker.io" {
		candidates = append(candidates,
			"https://index.docker.io/v1/",
			"https://index.docker.io/v1",
			"index.docker.io",
			dockerHubRegistry,
			defaultDockerHubHost,
		)
	}
	for _, key := range candidates {
		authEntry, exists := c.Auths[key]
		if !exists {
			continue
		}
		if authEntry.Username != "" || authEntry.Password != "" {
			return authEntry.Username, authEntry.Password, true
		}
		if authEntry.Auth == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(authEntry.Auth)
		if err != nil {
			continue
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			continue
		}
		return parts[0], parts[1], true
	}
	return "", "", false
}

type bearerChallenge struct {
	realm   string
	service string
	scope   string
}

func parseBearerChallenge(header string) (bearerChallenge, bool) {
	header = strings.TrimSpace(header)
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return bearerChallenge{}, false
	}
	params := header[len("Bearer "):]
	challenge := bearerChallenge{}
	for _, part := range strings.Split(params, ",") {
		part = strings.TrimSpace(part)
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(part[:eq]))
		value := strings.Trim(strings.TrimSpace(part[eq+1:]), `"`)
		switch key {
		case "realm":
			challenge.realm = value
		case "service":
			challenge.service = value
		case "scope":
			challenge.scope = value
		}
	}
	if challenge.realm == "" {
		return bearerChallenge{}, false
	}
	return challenge, true
}

func resolveImageDigestHTTP(image string, client *http.Client, cfg ImageDigestResolveConfig) (string, error) {
	cfg = cfg.withDefaults()
	ref, err := parseImageReference(image)
	if err != nil {
		return "", err
	}
	if ref.digest != "" {
		return ref.digestPinned(), nil
	}

	apiHost := ref.registryAPIHost(cfg)
	scheme := cfg.schemeFor(apiHost)
	manifestURL := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", scheme, apiHost, ref.repository, url.PathEscape(ref.tag))

	dockerCfg, err := loadDockerConfig(cfg.DockerConfigPath)
	if err != nil {
		return "", err
	}

	digest, authHeader, err := fetchManifestDigest(client, manifestURL, "")
	if err == nil {
		ref.digest = digest
		return ref.digestPinned(), nil
	}
	if authHeader == "" {
		return "", err
	}

	token, tokenErr := fetchRegistryPullToken(client, apiHost, ref.repository, authHeader, cfg, dockerCfg, ref.registry == dockerHubRegistry)
	if tokenErr != nil {
		// Some registries accept HTTP Basic directly without a token exchange.
		if username, password, ok := dockerCfg.credentialsFor(apiHost, ref.registry == dockerHubRegistry); ok {
			basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
			digest, _, basicErr := fetchManifestDigest(client, manifestURL, basic)
			if basicErr == nil {
				ref.digest = digest
				return ref.digestPinned(), nil
			}
			return "", fmt.Errorf("manifest fetch failed (%w); token fetch failed (%v); basic auth failed (%v)", err, tokenErr, basicErr)
		}
		return "", fmt.Errorf("manifest fetch failed (%w) and token fetch failed (%v)", err, tokenErr)
	}

	digest, _, err = fetchManifestDigest(client, manifestURL, "Bearer "+token)
	if err != nil {
		return "", err
	}
	ref.digest = digest
	return ref.digestPinned(), nil
}

// fetchManifestDigest returns the digest and, on 401, the WWW-Authenticate header value.
// It tries HEAD first, then falls back to GET when the registry rejects HEAD, omits
// Docker-Content-Digest on HEAD, or otherwise fails without an auth challenge.
func fetchManifestDigest(client *http.Client, manifestURL, authorization string) (digest string, wwwAuthenticate string, err error) {
	digest, wwwAuthenticate, err = doManifestDigestRequest(client, http.MethodHead, manifestURL, authorization)
	if err == nil {
		return digest, "", nil
	}
	// Auth challenges must be handled by the caller before retrying.
	if wwwAuthenticate != "" {
		return "", wwwAuthenticate, err
	}

	digest, wwwAuthenticate, getErr := doManifestDigestRequest(client, http.MethodGet, manifestURL, authorization)
	if getErr == nil {
		return digest, "", nil
	}
	if wwwAuthenticate != "" {
		return "", wwwAuthenticate, getErr
	}
	return "", "", fmt.Errorf("%v; GET fallback failed: %w", err, getErr)
}

func doManifestDigestRequest(client *http.Client, method, manifestURL, authorization string) (digest string, wwwAuthenticate string, err error) {
	req, err := http.NewRequest(method, manifestURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return "", resp.Header.Get("Www-Authenticate"), fmt.Errorf("registry returned %s for %s %s", resp.Status, method, manifestURL)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("registry returned %s for %s %s", resp.Status, method, manifestURL)
	}

	digest = resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", "", fmt.Errorf("registry response missing Docker-Content-Digest for %s %s", method, manifestURL)
	}
	return digest, "", nil
}

func fetchRegistryPullToken(
	client *http.Client,
	registryHost, repository, wwwAuthenticate string,
	cfg ImageDigestResolveConfig,
	dockerCfg *dockerConfigFile,
	includeDockerHubKeys bool,
) (string, error) {
	challenge, ok := parseBearerChallenge(wwwAuthenticate)
	if !ok {
		// No Bearer challenge: fall back to well-known Docker Hub auth for docker.io hosts.
		// Token realm/service for other registries (and Docker Hub mirrors) come from WWW-Authenticate.
		if registryHost == cfg.DockerHubRegistryHost || registryHost == dockerHubRegistry {
			challenge = bearerChallenge{
				realm:   defaultDockerHubAuthURL,
				service: defaultDockerHubService,
				scope:   "repository:" + repository + ":pull",
			}
		} else {
			return "", fmt.Errorf("registry did not return a Bearer WWW-Authenticate challenge")
		}
	}
	if challenge.scope == "" {
		challenge.scope = "repository:" + repository + ":pull"
	}
	if challenge.service == "" && (registryHost == cfg.DockerHubRegistryHost || registryHost == dockerHubRegistry || registryHost == defaultDockerHubHost) {
		challenge.service = defaultDockerHubService
	}

	tokenURL, err := url.Parse(challenge.realm)
	if err != nil {
		return "", fmt.Errorf("invalid token realm %q: %w", challenge.realm, err)
	}
	query := tokenURL.Query()
	if challenge.service != "" {
		query.Set("service", challenge.service)
	}
	query.Set("scope", challenge.scope)
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	if username, password, found := dockerCfg.credentialsFor(registryHost, includeDockerHubKeys); found {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned %s", resp.Status)
	}

	var payload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	token := payload.Token
	if token == "" {
		token = payload.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("token endpoint returned empty token")
	}
	return token, nil
}
