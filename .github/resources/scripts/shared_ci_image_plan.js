'use strict';

const {applicable, loadLocalInventory} = require('./ci_expected_workflows');

const AMD64_CONSUMERS = new Set([
  'api-server-tests.yml', 'e2e-test-frontend.yml', 'e2e-test.yml',
  'kfp-kubernetes-native-migration-tests.yaml', 'kfp-sdk-client-tests.yml',
  'kfp-webhooks.yml',
]);

function architectures(inventory, eventName, event, files) {
  if (eventName === 'merge_group') {
    if (event.merge_group?.base_ref !== 'refs/heads/master') {
      throw new Error('Shared queue images require master');
    }
    return {amd64: true, arm64: true};
  }
  if (eventName === 'push') return {amd64: true, arm64: false};
  if (eventName === 'workflow_dispatch') return {amd64: true, arm64: true};
  if (eventName !== 'pull_request') throw new Error(`Unsupported event: ${eventName}`);
  const branch = event.pull_request.base.ref;
  const selected = {amd64: false, arm64: false};
  for (const workflow of inventory.workflows) {
    const name = workflow.path.split('/').pop();
    const architecture = name === 'arm64-presubmit.yml' ? 'arm64'
      : AMD64_CONSUMERS.has(name) ? 'amd64' : null;
    if (architecture && applicable(workflow.pull_request, branch, files)) {
      selected[architecture] = true;
    }
  }
  return selected;
}

// The files endpoint describes the live PR, not the immutable workflow event.
function samePullRequest(actual, expected) {
  if (actual.number !== expected.number) return false;
  for (const side of ['head', 'base']) {
    for (const field of ['sha', 'ref']) {
      if (!expected[side]?.[field] || actual[side]?.[field] !== expected[side][field]) return false;
    }
    for (const field of ['id', 'full_name']) {
      if (!expected[side]?.repo?.[field] || actual[side]?.repo?.[field] !== expected[side].repo[field]) return false;
    }
  }
  return true;
}

function pullRequestArchitectures(inventory, event, request, warn = console.warn) {
  try {
    const number = event.pull_request.number;
    if (!Number.isSafeInteger(number) || number <= 0) throw new Error('Invalid PR number');
    const endpoint = `pulls/${number}`;
    if (!samePullRequest(request(endpoint), event.pull_request)) {
      throw new Error('PR revision changed before file lookup');
    }
    const changes = request(`${endpoint}/files?per_page=100`, true).flat();
    if (!samePullRequest(request(endpoint), event.pull_request)) {
      throw new Error('PR revision changed during file lookup');
    }
    if (changes.length !== event.pull_request.changed_files || changes.length >= 3000) {
      throw new Error('Changed-file response is incomplete');
    }
    const files = changes.flatMap(file => [file.filename, file.previous_filename].filter(Boolean));
    return architectures(inventory, 'pull_request', event, files);
  } catch (error) {
    warn(`::warning::Cannot narrow shared image architectures: ${error.message}; building both.`);
    return {amd64: true, arm64: true};
  }
}

function main() {
  const fs = require('node:fs');
  const {execFileSync} = require('node:child_process');
  const event = JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
  const {inventory} = loadLocalInventory(process.cwd());
  const request = (endpoint, paginate = false) => JSON.parse(execFileSync('gh', [
    'api', ...(paginate ? ['--paginate', '--slurp'] : []),
    `repos/${process.env.GITHUB_REPOSITORY}/${endpoint}`,
  ], {encoding: 'utf8', maxBuffer: 10 * 1024 * 1024, timeout: 120000}));
  const selected = process.env.GITHUB_EVENT_NAME === 'pull_request'
    ? pullRequestArchitectures(inventory, event, request)
    : architectures(inventory, process.env.GITHUB_EVENT_NAME, event, []);
  for (const [architecture, enabled] of Object.entries(selected)) {
    fs.appendFileSync(process.env.GITHUB_OUTPUT, `${architecture}=${enabled}\n`);
  }
}

module.exports = {architectures, pullRequestArchitectures, AMD64_CONSUMERS};
if (require.main === module) main();
