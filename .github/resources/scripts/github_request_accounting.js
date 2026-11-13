// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Count requests visible to this Octokit client without recording request data.
'use strict';

const METHODS = new Set(['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']);
const SCOPES = new Set(['prepare', 'finalize', 'recovery', 'merged_prepare', 'merged_finalize', 'revoke']);
const RESOURCES = new Set(['core', 'graphql', 'search', 'code_search', 'integration_manifest',
  'scim', 'actions_runner_registration', 'source_import']);

function category(url) {
  // Discard query parameters before matching. Never include a URL in output.
  const path = String(url || '').split('?')[0].replace(/^https?:\/\/[^/]+/, '');
  if (path === '/graphql') return 'graphql';
  const repo = path.replace(/^\/repos\/[^/]+\/[^/]+/, '');
  if (/^\/actions\/(runs(?:\/[^/]+)?\/artifacts|artifacts)(?:\/|$)/.test(repo)) return 'artifacts';
  if (/^\/actions\/runs\/[^/]+\/jobs(?:\/|$)/.test(repo)) return 'run_jobs';
  if (/^\/actions\/(runs|workflows\/[^/]+\/runs)(?:\/|$)/.test(repo)) return 'workflow_runs';
  if (/^\/actions\/workflows(?:\/|$)/.test(repo)) return 'workflow_definitions';
  if (/^\/commits\/[^/]+\/check-runs$/.test(repo)) return 'check_runs';
  if (/^\/(statuses\/[^/]+|commits\/[^/]+\/status(?:es)?)$/.test(repo)) return 'commit_statuses';
  if (/^\/pulls\/[^/]+\/files$/.test(repo)) return 'pull_files';
  if (/^\/pulls(?:\/|$)/.test(repo)) return 'pulls';
  if (/^\/issues\/[^/]+\/labels(?:\/|$)/.test(repo)) return 'issue_labels';
  if (/^\/issues\/[^/]+\/timeline$/.test(repo)) return 'issue_timeline';
  if (/^\/contents(?:\/|$)/.test(repo)) return 'contents';
  if (/^\/(branches|git\/refs)(?:\/|$)/.test(repo)) return 'branches';
  return 'other';
}

function rateLimit(headers) {
  const result = {};
  for (const field of ['limit', 'remaining', 'used', 'reset']) {
    const value = String(headers?.[`x-ratelimit-${field}`] ?? '');
    if (/^\d{1,15}$/.test(value) && Number.isSafeInteger(Number(value))) result[field] = Number(value);
  }
  if (!Object.keys(result).length) return null;
  const resource = headers?.['x-ratelimit-resource'];
  return {resource: RESOURCES.has(resource) ? resource : 'other', value: result};
}

async function withRequestAccounting({github, core, scope}, operation) {
  const counts = new Map();
  const rates = {};
  let requests = 0;
  const available = typeof github.hook?.wrap === 'function' && typeof github.hook?.remove === 'function';
  const hook = async (request, options) => {
    let response;
    let status = 'transport_error';
    try {
      response = await request(options);
      status = response?.status;
      return response;
    } catch (error) {
      response = error?.response;
      // Octokit also assigns synthetic 500 statuses to transport errors.
      status = response?.status;
      throw error;
    } finally {
      // Instrumentation must never replace a response or its original error.
      try {
        requests++;
        const method = METHODS.has(options.method) ? options.method : 'OTHER';
        status = Number.isInteger(status) && status >= 100 && status <= 599 ? String(status) : 'transport_error';
        const key = `${method} ${category(options.url)} ${status}`;
        counts.set(key, (counts.get(key) || 0) + 1);
        const observation = rateLimit(response?.headers);
        if (observation) {
          const previous = rates[observation.resource];
          rates[observation.resource] = {observations: (previous?.observations || 0) + 1,
            first: previous?.first || observation.value, last: observation.value};
        }
      } catch { /* Accounting cannot affect the gate decision. */ }
    }
  };
  if (available) github.hook.wrap('request', hook);
  try {
    return await operation();
  } finally {
    try {
      if (available) github.hook.remove('request', hook);
    } catch { /* Hook cleanup must not replace the operation's error. */ }
    try {
      core.info(JSON.stringify({event: 'github_request_accounting', version: 1,
        scope: SCOPES.has(scope) ? scope : 'other', measurement: 'octokit_request_hook',
        coverage: 'This client only; opaque transport retries and other Actions clients are excluded.',
        hook_available: available, requests,
        counts: [...counts].sort(([a], [b]) => a.localeCompare(b)).map(([key, count]) => {
          const [method, category, status] = key.split(' ');
          return {method, category, status, count};
        }), rate_limits: rates}));
    } catch { /* Reporting must preserve the operation result, including errors. */ }
  }
}

module.exports = {withRequestAccounting};
