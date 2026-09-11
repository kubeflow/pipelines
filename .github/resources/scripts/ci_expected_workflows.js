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

async function verifyExpectedWorkflows({github, owner, repo, pullRequest, inventory,
  workflowFiles, freshAfter = null}) {
  validateInventory(inventory, workflowFiles);
  const cutoff = freshAfter === null ? null : Date.parse(freshAfter);
  if (cutoff !== null && !Number.isFinite(cutoff)) throw new Error('Invalid CI freshness cutoff');
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
  const expected = inventory.workflows.filter(workflow => workflow.pull_request !== null &&
    applicable(workflow.pull_request, pullRequest.base.ref, paths));
  const reasons = [];
  if (expected.length === 0) reasons.push('No expected PR workflows; CI coverage cannot be established');
  // Fetch one head-scoped snapshot for all lanes, rather than one request
  // series per workflow on every constituent completion event.
  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    owner, repo, event: 'pull_request', head_sha: pullRequest.head.sha, per_page: 100,
  });
  if (runs.length >= 1000) throw new Error('Workflow run history truncated for this PR head');
  for (const workflow of expected) {
    const matching = runs.filter(run => run.path === workflow.path && run.event === 'pull_request' &&
      run.head_sha === pullRequest.head.sha && run.head_branch === pullRequest.head.ref &&
      run.head_repository?.full_name === pullRequest.head.repo.full_name);
    // A rerun updates an existing ID. Order by attempt start as well as run
    // creation so rerunning an older execution cannot hide behind a newer
    // run's prior success. Creation remains the separate base-freshness proof.
    const attemptTime = run => Math.max(Date.parse(run.created_at) || 0,
      Date.parse(run.run_started_at) || 0);
    matching.sort((left, right) => attemptTime(right) - attemptTime(left) || right.id - left.id);
    const run = matching[0];
    if (!run) {
      reasons.push(`${workflow.path}: expected workflow has not registered`);
      continue;
    }
    if (run.status !== 'completed' || run.conclusion !== 'success') {
      reasons.push(`${workflow.path}: latest run is ${run.status}/${run.conclusion}`);
    }
    // Rerunning an old run retains its original GITHUB_SHA/GITHUB_REF. Only
    // a new run created after retargeting can establish fresh base evidence.
    if (cutoff !== null && !(Date.parse(run.created_at) > cutoff)) {
      reasons.push(`${workflow.path}: trigger a new CI run after the base changed`);
    }
    // When GitHub supplies base evidence, reject a different base. Some real
    // PR runs have an empty pull_requests array; the durable retarget cutoff
    // supplied by the caller covers that case.
    const association = (run.pull_requests || []).find(pr => pr.number === pullRequest.number);
    if (association?.base?.ref && association.base.ref !== pullRequest.base.ref) {
      reasons.push(`${workflow.path}: workflow ran for a different base branch`);
    }
  }
  return {passed: reasons.length === 0, reasons, expected: expected.map(workflow => workflow.path)};
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
  verifyExpectedWorkflows, loadLocalInventory, triggerHeader};
