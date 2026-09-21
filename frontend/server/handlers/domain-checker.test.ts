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
import { isAllowedDomain, isTrustedArtifactEndpoint } from './domain-checker.js';

describe('isAllowedDomain', () => {
  it('matches a host-only allowlist for a plain https URL', () => {
    expect(isAllowedDomain('https://example.com/path/to/artifact', '^example\\.com$')).toBe(true);
  });

  it('matches a host-only allowlist for an http URL with a port', () => {
    expect(isAllowedDomain('http://example.com:8080/path/to/artifact', '^example\\.com$')).toBe(
      true,
    );
  });

  it('matches a host-only allowlist when the URL contains user info', () => {
    expect(isAllowedDomain('https://user@example.com/path/to/artifact', '^example\\.com$')).toBe(
      true,
    );
  });

  it('rejects malformed input when the allowlist expects a hostname', () => {
    expect(isAllowedDomain('https:///broken', '^example\\.com$')).toBe(false);
  });

  it('rejects a subdomain when the allowlist expects the exact host', () => {
    expect(isAllowedDomain('https://sub.example.com/path/to/artifact', '^example\\.com$')).toBe(
      false,
    );
  });

  it('rejects a URL whose host does not match the allowlist', () => {
    expect(isAllowedDomain('https://evil.example/path/to/artifact', '^example\\.com$')).toBe(false);
  });

  it('accepts the default production allowlist for service URLs', () => {
    expect(isAllowedDomain('http://ml-pipeline.kubeflow:8888/artifacts', '^.*$')).toBe(true);
  });

  it('rejects non-http schemes before applying the allowlist', () => {
    expect(isAllowedDomain('file:///etc/passwd', '^.*$')).toBe(false);
  });
});

describe('isTrustedArtifactEndpoint', () => {
  it('trusts the exact configured endpoint', () => {
    expect(
      isTrustedArtifactEndpoint('http://seaweedfs.kubeflow:9000', [
        'http://seaweedfs.kubeflow:9000',
      ]),
    ).toBe(true);
  });

  it('rejects other hosts and ports', () => {
    expect(
      isTrustedArtifactEndpoint('http://169.254.169.254:80', ['http://seaweedfs.kubeflow:9000']),
    ).toBe(false);
    expect(
      isTrustedArtifactEndpoint('http://seaweedfs.kubeflow:9001', [
        'http://seaweedfs.kubeflow:9000',
      ]),
    ).toBe(false);
  });

  it('rejects endpoints containing user info', () => {
    expect(
      isTrustedArtifactEndpoint('http://attacker@seaweedfs.kubeflow:9000', [
        'http://seaweedfs.kubeflow:9000',
      ]),
    ).toBe(false);
  });
});
