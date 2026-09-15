// Copyright 2023 The Kubeflow Authors
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

export function isAllowedDomain(urlStr: string, allowedDomain: string): boolean {
  const allowedRegExp = new RegExp(allowedDomain);
  const domain = domain_from_url(urlStr);
  const allowed = domain.length > 0 && allowedRegExp.test(domain);
  if (!allowed) {
    console.log(`Domain not allowed: ${urlStr}`);
  }
  return allowed;
}

/**
 * Trusts an object-store endpoint only when its effective origin exactly
 * matches an endpoint configured by the server operator.
 */
export function isTrustedArtifactEndpoint(
  endpoint: string,
  configuredEndpoints: string[],
): boolean {
  const normalizedEndpoint = normalizeEndpoint(endpoint);
  if (!normalizedEndpoint) {
    return false;
  }

  return configuredEndpoints.some(
    (configuredEndpoint) => normalizeEndpoint(configuredEndpoint) === normalizedEndpoint,
  );
}

function normalizeEndpoint(endpoint: string): string | undefined {
  try {
    const parsedUrl = new URL(endpoint.includes('://') ? endpoint : `https://${endpoint}`);
    if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
      return undefined;
    }
    if (parsedUrl.username || parsedUrl.password) {
      return undefined;
    }
    const port = parsedUrl.port || (parsedUrl.protocol === 'https:' ? '443' : '80');
    return `${parsedUrl.protocol}//${parsedUrl.hostname.toLowerCase()}:${port}`;
  } catch {
    return undefined;
  }
}

function domain_from_url(url: string): string {
  try {
    const parsedUrl = new URL(url.includes('://') ? url : `http://${url}`);
    if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
      return '';
    }
    return parsedUrl.hostname;
  } catch {
    return '';
  }
}
