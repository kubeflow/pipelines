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
'use strict';

// Scoped Node24+ preload. Observe transport diagnostics, without replacing any
// client's request implementation, retry policy, redirects or error handlers.
const diagnostics = require('node:diagnostics_channel');
const fs = require('node:fs');
const scopes = new Set(['artifact_primary', 'artifact_retry', 'all_checks', 'merged_all_checks',
  'prepare', 'finalize', 'recovery', 'merged_prepare', 'merged_finalize', 'revoke', 'merge_group', 'queue_admission']);
const methods = new Set(['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']);
const resources = new Set(['core', 'graphql', 'search', 'code_search', 'integration_manifest',
  'scim', 'actions_runner_registration', 'source_import']);
const states = new WeakMap();
const counts = new Map();
const rates = {};
let requests = 0;

function origin(value) {
  try { return new URL(value).origin; } catch { return ''; }
}
const apiOrigin = origin(process.env.GITHUB_API_URL || 'https://api.github.com');
const artifactOrigin = origin(process.env.ACTIONS_RESULTS_URL);

function category(url) {
  if (url.origin === apiOrigin) {
    if (/^\/repos\/[^/]+\/[^/]+\/?$/.test(url.pathname)) return 'repository_metadata';
    const path = url.pathname.replace(/^\/repos\/[^/]+\/[^/]+/, '');
    if (path === '/graphql') return 'graphql';
    if (/^\/actions\/(runs(?:\/[^/]+)?\/artifacts|artifacts)(?:\/|$)/.test(path)) return 'artifacts';
    if (/^\/actions\/runs\/[^/]+(?:\/attempts\/[^/]+)?\/jobs(?:\/|$)/.test(path)) return 'run_jobs';
    if (/^\/actions\/(runs|workflows\/[^/]+\/runs)(?:\/|$)/.test(path)) return 'workflow_runs';
    if (/^\/actions\/workflows\/[^/]+\/dispatches$/.test(path)) return 'workflow_dispatches';
    if (/^\/actions\/workflows(?:\/|$)/.test(path)) return 'workflow_definitions';
    if (/^\/commits\/[^/]+\/check-runs$/.test(path)) return 'check_runs';
    if (/^\/(statuses\/[^/]+|commits\/[^/]+\/status(?:es)?)$/.test(path)) return 'commit_statuses';
    if (/^\/pulls\/[^/]+\/files$/.test(path)) return 'pull_files';
    if (/^\/pulls(?:\/|$)/.test(path)) return 'pulls';
    if (/^\/issues\/[^/]+\/labels(?:\/|$)/.test(path)) return 'issue_labels';
    if (/^\/issues\/[^/]+\/timeline$/.test(path)) return 'issue_timeline';
    if (/^\/contents(?:\/|$)/.test(path)) return 'contents';
    if (/^\/rules\/branches\/.+/.test(path)) return 'branch_rules';
    if (/^\/(branches|git\/refs?)(?:\/|$)/.test(path)) return 'branches';
    return 'github_api_other';
  }
  if (artifactOrigin && url.origin === artifactOrigin) return 'artifact_service';
  // Blob URLs may use arbitrary storage hosts; don't classify all other hosts
  // as GitHub API calls or emit any hostname, path, or signed URL.
  return 'other_http';
}

function increment(state, status, amount) {
  const key = `${state.method} ${state.category} ${status}`;
  const count = (counts.get(key) || 0) + amount;
  if (count) counts.set(key, count); else counts.delete(key);
}

function start(request, url, method) {
  if (states.has(request)) return;
  const state = {method: methods.has(method) ? method : 'OTHER', category: category(url), api: url.origin === apiOrigin, done: false};
  states.set(request, state);
  requests++;
  increment(state, 'incomplete', 1);
}

function rateLimit(headers) {
  const values = {};
  // Undici supplies alternating header-name/value Buffers; native HTTP uses an
  // object. Read only the allowlisted rate-limit headers in either form.
  const get = name => {
    if (!Array.isArray(headers)) return headers?.[name];
    for (let i = 0; i < headers.length; i += 2) {
      if (String(headers[i]).toLowerCase() === name) return headers[i + 1];
    }
  };
  for (const field of ['limit', 'remaining', 'used', 'reset']) {
    const value = String(get(`x-ratelimit-${field}`) ?? '');
    if (/^\d{1,15}$/.test(value) && Number.isSafeInteger(Number(value))) values[field] = Number(value);
  }
  if (!Object.keys(values).length) return;
  const raw = String(get('x-ratelimit-resource') ?? '');
  const resource = resources.has(raw) ? raw : 'other';
  const previous = rates[resource];
  rates[resource] = {observations: (previous?.observations || 0) + 1,
    first: previous?.first || values, last: values};
}

function finish(request, code, headers) {
  const state = states.get(request);
  if (!state || state.done) return;
  state.done = true;
  increment(state, 'incomplete', -1);
  const status = Number.isInteger(code) && code >= 100 && code <= 599 ? String(code) : 'transport_error';
  increment(state, status, 1);
  if (state.api) rateLimit(headers);
}

function subscribe(name, observer) {
  diagnostics.subscribe(name, event => {
    // A diagnostics subscriber exception otherwise becomes an uncaught error.
    try { observer(event); } catch { /* Observability cannot alter the action. */ }
  });
}

const scope = process.env.CI_HTTP_ACCOUNTING_SCOPE;
const transportSupported = Number(process.versions.node.split('.')[0]) >= 24;
if (scopes.has(scope)) {
  if (transportSupported) {
    subscribe('undici:request:create', ({request}) => {
      start(request, new URL(request.path, request.origin), request.method);
    });
    subscribe('undici:request:headers', ({request, response}) => finish(request, response.statusCode, response.headers));
    subscribe('undici:request:error', ({request}) => finish(request));
    subscribe('http.client.request.created', ({request}) => {
      const url = new URL(request.path, `${request.protocol}//${request.getHeader('host')}`);
      start(request, url, request.method);
    });
    subscribe('http.client.response.finish', ({request, response}) => finish(request, response.statusCode, response.headers));
    subscribe('http.client.request.error', ({request}) => finish(request));
  }
  process.once('exit', () => {
    try {
      const summary = {event: 'github_request_accounting', version: 1,
        scope, measurement: 'node_http_attempts', transport_supported: transportSupported,
        coverage: 'This Node process only; HTTP attempts include redirects and retries. Other processes and forced termination are excluded. Do not add to Octokit logical-request summaries.',
        requests, counts: [...counts].sort(([a], [b]) => a.localeCompare(b)).map(([key, count]) => {
          const [method, category, status] = key.split(' ');
          return {method, category, status, count};
        }), rate_limits: rates};
      // Synchronous output also survives an action calling process.exit().
      fs.writeSync(1, `${JSON.stringify(summary)}\n`);
    } catch { /* Preserve the action exit status even if output is unavailable. */ }
  });
}
