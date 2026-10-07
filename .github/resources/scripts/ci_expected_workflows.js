// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

// Support only the glob subset used in this repository. Reject unsupported
// syntax instead of silently omitting an expected workflow.
function globRegex(pattern) {
  if (typeof pattern !== 'string' || !pattern || /[?+\[\]{}()\\!]/.test(pattern)) {
    throw new Error(`Unsupported CI trigger glob: ${pattern}`);
  }
  let source = '^';
  for (let index = 0; index < pattern.length; index++) {
    if (pattern[index] === '*') {
      if (pattern[index + 1] === '*') {
        if (pattern[index + 2] === '*') throw new Error(`Unsupported glob: ${pattern}`);
        index++;
        if (pattern[index + 1] === '/') {
          source += '(?:.*/)?';
          index++;
        } else {
          source += '.*';
        }
      } else {
        source += '[^/]*';
      }
    } else {
      source += pattern[index].replace(/[.$^|]/g, '\\$&');
    }
  }
  return new RegExp(`${source}$`);
}

function matchesPatterns(value, patterns) {
  if (!Array.isArray(patterns) || patterns.length === 0) {
    throw new Error('CI trigger filters must be nonempty arrays');
  }
  let matched = false;
  for (const pattern of patterns) {
    if (typeof pattern !== 'string') throw new Error('CI trigger patterns must be strings');
    const negative = pattern.startsWith('!');
    const regex = globRegex(negative ? pattern.slice(1) : pattern);
    if (regex.test(value)) matched = !negative;
  }
  return matched;
}

function applicable(trigger, branch, files) {
  if (!trigger || typeof trigger !== 'object' || Array.isArray(trigger)) {
    throw new Error('Unsupported pull_request trigger');
  }
  const supported = new Set(['branches', 'branches-ignore', 'paths', 'paths-ignore']);
  for (const key of Object.keys(trigger)) {
    if (!supported.has(key)) throw new Error(`Unsupported pull_request trigger key: ${key}`);
  }
  if ((trigger.branches && trigger['branches-ignore']) ||
      (trigger.paths && trigger['paths-ignore'])) {
    throw new Error('Conflicting CI trigger filters');
  }
  // Validate every pattern, including those in a currently irrelevant lane.
  for (const patterns of Object.values(trigger)) matchesPatterns('', patterns);
  if (trigger.branches && !matchesPatterns(branch, trigger.branches)) return false;
  if (trigger['branches-ignore'] && matchesPatterns(branch, trigger['branches-ignore'])) return false;
  if (trigger.paths && !files.some(file => matchesPatterns(file, trigger.paths))) return false;
  if (trigger['paths-ignore'] && files.every(file => matchesPatterns(file, trigger['paths-ignore']))) return false;
  return true;
}

function validateInventory(inventory, workflowFiles) {
  if (inventory.version !== 1 || !Array.isArray(inventory.workflows)) {
    throw new Error('Invalid CI workflow inventory; regenerate it');
  }
  const actual = new Map(workflowFiles.filter(file => /\.ya?ml$/.test(file.path))
    .map(file => [file.path, file.header_sha256]));
  const recorded = new Set();
  for (const workflow of inventory.workflows) {
    if (recorded.has(workflow.path) || !workflow.header_sha256 ||
        actual.get(workflow.path) !== workflow.header_sha256) {
      throw new Error(`CI workflow inventory is stale for ${workflow.path}; regenerate it`);
    }
    recorded.add(workflow.path);
  }
  if (actual.size !== recorded.size) throw new Error('CI workflow inventory is incomplete; regenerate it');
}

function invalidRunMetadata(run) {
  for (const field of ['id', 'run_attempt']) {
    if (!Number.isSafeInteger(run[field]) || run[field] <= 0) return field;
  }
  const timestamp = value => typeof value === 'string' && Number.isFinite(Date.parse(value));
  if (!timestamp(run.created_at)) return 'created_at';
  const unstarted = run.run_attempt === 1 && run.run_started_at === null &&
    ['queued', 'requested', 'waiting', 'pending'].includes(run.status) && run.conclusion == null;
  if (!unstarted && !timestamp(run.run_started_at)) return 'run_started_at';
  return null;
}

const MISSING_BASE_RECOVERY_DOCS =
  'https://github.com/kubeflow/pipelines/blob/master/docs/agents/ci-passed.md#missing-base-merge-record';

async function baseArrivedAt({github, owner, repo, baseSha}) {
  // A run tests the base revision that was live when it was created. For a
  // pull_request run, Actions checks out a synthetic merge commit whose SHA is
  // NOT exposed on the run object, so it cannot be compared directly. The
  // merge commit contains the validated base iff the run was created at or
  // after that base became reachable on the base branch. A commit's own
  // timestamps do not record that arrival: a commit can be prepared well
  // before it is pushed (fast-forward integration), so neither committer.date
  // nor author.date proves when the branch advanced to it. The authoritative
  // provenance is the merge record: when the base revision landed through a
  // pull request, that PR's merged_at is when the ref advanced, regardless of
  // merge method. A direct push of a pre-existing commit has no merged PR, so
  // its arrival cannot be established and the caller fails closed to pending
  // rather than ever passing stale evidence as fresh.
  // Returns a discriminated result so the caller can tell a lookup failure, a
  // missing merge record, and an unparseable merge timestamp apart, instead of
  // collapsing all three to null:
  //   {status: 'arrived', arrivedAt}  - a merged PR with a parseable merged_at
  //   {status: 'missing'}             - no merged PR (e.g. a direct push)
  //   {status: 'invalid'}             - a merged PR whose merged_at is unparseable
  //   {status: 'error'}               - the lookup API call failed
  if (!baseSha) return {status: 'missing'};
  try {
    const pulls = await github.paginate(github.rest.repos.listPullRequestsAssociatedWithCommit, {
      owner, repo, commit_sha: baseSha, per_page: 100,
    });
    for (const pr of pulls) {
      if (pr.merged_at) {
        const arrived = Date.parse(pr.merged_at);
        if (Number.isFinite(arrived)) return {status: 'arrived', arrivedAt: arrived};
        return {status: 'invalid'};
      }
    }
    return {status: 'missing'};
  } catch (error) {
    return {status: 'error'};
  }
}

async function verifyExpectedWorkflows({github, owner, repo, pullRequest, baseSha, liveBase, inventory,
  workflowFiles, freshAfter = null, registrationStartedAt = null, now = Date.now()}) {
  validateInventory(inventory, workflowFiles);
  const cutoff = freshAfter === null ? null : Date.parse(freshAfter);
  if (cutoff !== null && !Number.isFinite(cutoff)) throw new Error('Invalid CI freshness cutoff');
  const registrationStart = registrationStartedAt === null ? null : Date.parse(registrationStartedAt);
  if ((registrationStartedAt !== null && typeof registrationStartedAt !== 'string') ||
      (registrationStart !== null && !Number.isFinite(registrationStart)) || !Number.isFinite(now)) {
    throw new Error('Invalid CI registration timestamp; inspect CI Check and retry');
  }
  const registrationExpired = registrationStart !== null && now - registrationStart >= 15 * 60 * 1000;
  const files = await github.paginate(github.rest.pulls.listFiles, {
    owner, repo, pull_number: pullRequest.number, per_page: 100,
  });
  // GitHub caps this endpoint at 3,000 files. Never approve truncated evidence.
  if (!Number.isInteger(pullRequest.changed_files) ||
      files.length !== pullRequest.changed_files || files.length >= 3000) {
    throw new Error('Incomplete PR file list; CI coverage cannot be established');
  }
  const paths = files.flatMap(file => file.previous_filename ?
    [file.filename, file.previous_filename] : [file.filename]);
  if (paths.some(path => typeof path !== 'string')) throw new Error('Invalid changed-file path');
  const applicableWorkflows = inventory.workflows.filter(workflow => workflow.pull_request !== null &&
    applicable(workflow.pull_request, pullRequest.base.ref, paths));
  const disabled = applicableWorkflows.filter(workflow =>
    workflow.path === '.github/workflows/upgrade-test.yml' &&
    workflow.disabled_for_migration === true)
    .map(workflow => ({path: workflow.path, reason: 'Upgrade workflow is paused in the trusted base pending #14029'}));
  const expected = applicableWorkflows.filter(workflow => !disabled.some(item => item.path === workflow.path));
  const failures = [];
  const pending = [];
  const missing = [];
  let documentation = null;
  if (expected.length === 0) failures.push('No expected PR workflows; CI coverage cannot be established');
  // Fetch one head-scoped snapshot for all lanes, rather than one request
  // series per workflow on every constituent completion event.
  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    owner, repo, event: 'pull_request', head_sha: pullRequest.head.sha, per_page: 100,
  });
  if (runs.length >= 1000) throw new Error('Workflow run history truncated for this PR head');
  // Freshness keys off the LIVE base tip. A run tested the validated base iff
  // it was created at or after that base became reachable; resolve the arrival
  // time once here (it depends only on baseSha, not on any individual run).
  // When the base is frozen (release branches), skip the check entirely to
  // preserve existing behavior.
  const baseArrived = liveBase && baseSha ?
    await baseArrivedAt({github, owner, repo, baseSha}) : null;
  for (const workflow of expected) {
    const matching = runs.filter(run => run.path === workflow.path && run.event === 'pull_request' &&
      run.head_sha === pullRequest.head.sha && run.head_branch === pullRequest.head.ref &&
      run.head_repository?.full_name === pullRequest.head.repo.full_name);
    // Invalid metadata can change which execution sorts latest. Reject the
    // workflow instead of discarding a malformed run and reusing an old pass.
    const invalidField = matching.map(invalidRunMetadata).find(field => field !== null);
    if (invalidField) {
      failures.push(`${workflow.path}: invalid workflow run ${invalidField}; inspect CI Check and retry`);
      continue;
    }
    // A rerun updates an existing ID. Order by attempt start as well as run
    // creation so rerunning an older execution cannot hide behind a newer
    // run's prior success. Creation remains the separate base-freshness proof.
    const attemptTime = run => run.run_started_at === null ? Date.parse(run.created_at) :
      Math.max(Date.parse(run.created_at), Date.parse(run.run_started_at));
    matching.sort((left, right) => attemptTime(right) - attemptTime(left) || right.id - left.id);
    const run = matching[0];
    if (!run) {
      missing.push(workflow.path);
      if (registrationExpired) {
        failures.push(`${workflow.path}: expected workflow has not registered after 15 minutes; inspect its trigger and approval state`);
      } else {
        pending.push(`${workflow.path}: expected workflow has not registered`);
      }
      continue;
    }
    const waiting = ['queued', 'requested', 'waiting', 'in_progress', 'pending'].includes(run.status);
    if (waiting && run.conclusion == null) {
      pending.push(`${workflow.path}: latest run is ${run.status}`);
    } else if (run.status !== 'completed' || run.conclusion !== 'success') {
      failures.push(`${workflow.path}: latest run is ${run.status}/${run.conclusion}`);
    }
    // Rerunning an old run retains its original GITHUB_SHA/GITHUB_REF. Only
    // a new run created after retargeting can establish fresh base evidence.
    if (cutoff !== null && !(Date.parse(run.created_at) > cutoff)) {
      failures.push(`${workflow.path}: trigger a new CI run after the base changed`);
    }
    // When GitHub supplies base evidence, reject a different base. Some real
    // PR runs have an empty pull_requests array; the durable retarget cutoff
    // supplied by the caller covers that case.
    const association = (run.pull_requests || []).find(pr => pr.number === pullRequest.number);
    if (association?.base?.ref && association.base.ref !== pullRequest.base.ref) {
      failures.push(`${workflow.path}: workflow ran for a different base branch`);
    }
    // The run-scoped base association and pullRequest.base.sha are MUTABLE
    // (GitHub rewrites them to the PR's current base), so neither proves which
    // base a run tested. Instead, a run is fresh for the live base only when it
    // was created after that base became reachable on the base branch; the
    // run's immutable creation time is compared against the push event that
    // introduced the base (run.created_at is validated by invalidRunMetadata
    // above). Both values are second-granularity, so an equal instant is
    // ambiguous: the run may have been created sub-second before the base
    // arrived and tested an older base. Resolve the tie fail-closed (<=) so the
    // merge gate can never pass stale evidence as fresh; the PR stays pending
    // until a strictly later run.
    if (liveBase && baseSha) {
      if (baseArrived.status === 'error') {
        pending.push(`${workflow.path}: Cannot establish master arrival for base ${baseSha}: lookup failed. Retrying.`);
      } else if (baseArrived.status === 'missing' || baseArrived.status === 'invalid') {
        // A missing record and an unparseable merged_at are both permanent: no
        // fresh CI run can establish arrival, so they need maintainer
        // investigation. The base's provenance is repo-wide, not per-workflow,
        // so the reason omits the workflow path and stays within the
        // 140-character status description. The recovery docs link rides in
        // the result documentation field, which reaches the CI Check logs
        // without bloating the status description.
        documentation = MISSING_BASE_RECOVERY_DOCS;
        if (baseArrived.status === 'missing') {
          pending.push(`no merged PR for base ${baseSha}. Maintainer investigation required.`);
        } else {
          pending.push(`malformed merged-PR record for base ${baseSha}. Maintainer investigation required.`);
        }
      } else if (baseArrived.status === 'arrived' && Date.parse(run.created_at) <= baseArrived.arrivedAt) {
        pending.push(`${workflow.path}: awaiting a fresh CI run against the new base`);
      }
    }
  }
  const state = failures.length ? 'failure' : pending.length ? 'pending' : 'success';
  return {state, passed: state === 'success', reasons: [...failures, ...pending],
    expected: expected.map(workflow => workflow.path), disabled, missing, documentation};
}

async function loadBaseInventory({github, owner, repo, pullRequest, baseSha, root}) {
  const base = pullRequest.base;
  const fullName = `${owner}/${repo}`;
  const sha = baseSha ?? base?.sha;
  if (!/^[0-9a-f]{40}$/.test(sha || '') ||
      base.repo?.full_name?.toLowerCase() !== fullName.toLowerCase()) {
    throw new Error('Workflow inventory requires an immutable trusted base repository SHA');
  }
  // Read the complete directory in one request. Never check out or execute
  // anything from the PR or its base: the selected blobs are YAML data only.
  const result = await github.graphql(`query BaseWorkflows($owner: String!, $repo: String!, $expression: String!) {
    repository(owner: $owner, name: $repo) {
      nameWithOwner
      object(expression: $expression) {
        __typename
        ... on Tree { entries { name type mode object {
          __typename
          ... on Blob { text byteSize isBinary isTruncated }
        } } }
      }
    }
  }`, {owner, repo, expression: `${sha}:.github/workflows`});
  const repository = result?.repository;
  const tree = repository?.object;
  if (repository?.nameWithOwner?.toLowerCase() !== fullName.toLowerCase() ||
      tree?.__typename !== 'Tree' || !Array.isArray(tree.entries) ||
      tree.entries.length === 0 || tree.entries.length >= 1000) {
    throw new Error('Incomplete trusted base workflow tree');
  }
  const names = new Set();
  const records = [];
  let totalBytes = 0;
  for (const entry of tree.entries) {
    if (typeof entry?.name !== 'string' || !/^[A-Za-z0-9_.-]+$/.test(entry.name) ||
        ['.', '..'].includes(entry.name) || names.has(entry.name)) {
      throw new Error('Invalid or duplicate trusted workflow entry');
    }
    names.add(entry.name);
    if (!/\.ya?ml$/.test(entry.name)) continue;
    const blob = entry.object;
    if (entry.type !== 'blob' || ![33188, 33261].includes(entry.mode) ||
        blob?.__typename !== 'Blob' || blob.isBinary !== false || blob.isTruncated !== false ||
        typeof blob.text !== 'string' || !Number.isSafeInteger(blob.byteSize) ||
        blob.byteSize < 1 || blob.byteSize > 1024 * 1024 ||
        Buffer.byteLength(blob.text, 'utf8') !== blob.byteSize) {
      throw new Error(`Incomplete or invalid trusted workflow blob: ${entry.name}`);
    }
    totalBytes += blob.byteSize;
    if (totalBytes > 8 * 1024 * 1024) throw new Error('Trusted workflow data exceeds size limit');
    records.push({path: `.github/workflows/${entry.name}`, content: blob.text});
  }
  const {spawnSync} = require('node:child_process');
  const path = require('node:path');
  const parsed = spawnSync('python3', [path.join(root,
    '.github/resources/scripts/generate_ci_workflow_inventory.py'), '--stdin'], {
    input: JSON.stringify(records), encoding: 'utf8', timeout: 30000, maxBuffer: 16 * 1024 * 1024,
  });
  if (parsed.error || parsed.status !== 0) {
    throw new Error(`Cannot parse trusted base workflows: ${parsed.error?.message || parsed.stderr}`);
  }
  const inventory = JSON.parse(parsed.stdout);
  validateInventory(inventory.inventory, inventory.workflowFiles);
  return inventory;
}

// Hash just top-level name/on blocks. Dependency bumps within jobs must not
// require regeneration. This extracts text; PyYAML alone interprets triggers.
function triggerHeader(content) {
  const blocks = [];
  const keys = new Set();
  let selected = false;
  for (const line of content.split(/\r?\n/)) {
    if (line && !/^\s/.test(line) && !line.startsWith('#')) {
      const match = line.match(/^(?:([a-z_]+)|"([a-z_]+)"|'([a-z_]+)')\s*:/);
      const key = match ? match.slice(1).find(Boolean) : '';
      selected = key === 'name' || key === 'on';
      if (selected) {
        if (keys.has(key)) throw new Error(`Duplicate workflow header: ${key}`);
        keys.add(key);
      }
    }
    if (selected) blocks.push(line);
  }
  if (!keys.has('name') || !keys.has('on')) {
    throw new Error('Workflow must have top-level name and on headers');
  }
  return blocks.join('\n');
}

function loadLocalInventory(root) {
  const fs = require('fs');
  const path = require('path');
  const crypto = require('crypto');
  const inventory = JSON.parse(fs.readFileSync(
    path.join(root, '.github/resources/ci-workflow-inventory.json'), 'utf8'));
  const directory = path.join(root, '.github/workflows');
  const workflowFiles = fs.readdirSync(directory).filter(name => /\.ya?ml$/.test(name)).map(name => {
    const content = fs.readFileSync(path.join(directory, name));
    return {
      path: `.github/workflows/${name}`,
      header_sha256: crypto.createHash('sha256').update(triggerHeader(content.toString('utf8'))).digest('hex'),
    };
  });
  validateInventory(inventory, workflowFiles);
  return {inventory, workflowFiles};
}

module.exports = {globRegex, matchesPatterns, applicable, validateInventory,
  verifyExpectedWorkflows, loadLocalInventory, loadBaseInventory, triggerHeader};
