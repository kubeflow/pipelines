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

const {createHash} = require('node:crypto');
const {probeDependencyConflict} = require('./queue_dependency_merge');

const DEPENDENCIES = new Set([
  'package.json', 'package-lock.json', 'npm-shrinkwrap.json', 'pnpm-lock.yaml', 'yarn.lock',
  'pyproject.toml', 'uv.lock', 'setup.py', 'setup.cfg', 'Pipfile', 'Pipfile.lock', 'poetry.lock',
  'go.mod', 'go.sum', 'go.work', 'go.work.sum', 'Cargo.toml', 'Cargo.lock',
  'Gemfile', 'Gemfile.lock', 'pom.xml', 'build.gradle', 'build.gradle.kts',
  'gradle.lockfile', 'libs.versions.toml', 'actions-lock.json',
]);

function isDependencyPath(path) {
  if (typeof path !== 'string') return false;
  const name = path.split('/').pop();
  return DEPENDENCIES.has(name) || /^(?:[\w.-]+-)?requirements(?:[-.][\w.-]+)?\.(?:in|txt)$/.test(name);
}

const BUDGET = Symbol('dependency probe budget');

async function checkDependencyConflict({github, context, pr, entries, base, core,
  cache = new Map(), statuses, probe = probeDependencyConflict, now = Date.now}) {
  if (!entries.length) return false;
  const {owner, repo} = context.repo;
  const branch = pr.base.ref;
  const head = pr.head.sha;
  const statusContext = `queue-dependency-conflict/${pr.number}`;
  // Include null/unbuilt candidates and queue order. A new group invalidates an
  // earlier observation even if every individual PR still has the same head.
  const fingerprint = createHash('sha256').update(JSON.stringify(['v1', owner, repo, branch, base, pr.number, head,
    entries.map(e => [e.id, e.pullRequest.number, e.pullRequest.headRefOid, e.baseCommit?.oid ?? null, e.headCommit?.oid ?? null]),
  ])).digest('hex');
  const prefix = `https://github.com/${owner}/${repo}/actions/runs/`;
  const matches = url => {
    if (typeof url !== 'string' || !url.startsWith(prefix)) return false;
    const [run] = url.slice(prefix.length).split('#');
    return /^[1-9][0-9]*$/.test(run) && url === `${prefix}${run}#dependency-conflict=${fingerprint}`;
  };
  const receipt = statuses?.find(s => s.context === statusContext);
  if (receipt?.state === 'success' && receipt.creator?.type === 'Bot' &&
      receipt.creator.login === 'github-actions[bot]' && matches(receipt.target_url)) {
    core?.info?.(`Deferring #${pr.number}: ${receipt.description}`);
    return true;
  }
  if (!cache.has(BUDGET)) cache.set(BUDGET, {reads: 40, probeMs: 0});
  const budget = cache.get(BUDGET);
  function read() {
    if (budget.reads <= 0) throw new Error('sweep file discovery budget reached');
    budget.reads--;
  }
  async function pathsFor(number, sha) {
    const key = JSON.stringify([owner, repo, branch, number, base, sha]);
    if (cache.has(key)) return cache.get(key);
    read();
    const {data} = await github.rest.repos.compareCommitsWithBasehead({
      owner, repo, basehead: `${base}...${sha}`, per_page: 1,
    });
    if (!Array.isArray(data.files)) throw new Error('missing changed files');
    let files = data.files;
    // The comparison API truncates the combined file list at 300. Use the
    // PR files endpoint for large changes, checking the head after pagination.
    if (files.length >= 300) {
      files = [];
      for (let page = 1; page <= 30; page++) {
        read();
        const {data: batch} = await github.rest.pulls.listFiles({owner, repo, pull_number: number, page, per_page: 100});
        if (!Array.isArray(batch)) throw new Error('missing PR files page');
        files.push(...batch);
        if (batch.length < 100) break;
      }
      read();
      const {data: current} = await github.rest.pulls.get({owner, repo, pull_number: number});
      if (current.head?.sha !== sha || current.changed_files !== files.length || current.base?.ref !== branch) {
        throw new Error('changed or incomplete PR file list');
      }
    }
    const paths = new Set(files.flatMap(file => [file.filename, file.previous_filename]).filter(isDependencyPath));
    cache.set(key, paths);
    return paths;
  }
  let conflict;
  try {
    const incoming = await pathsFor(pr.number, head);
    if (!incoming.size) return false;
    const overlaps = new Map();
    // A built successor already excludes preceding PRs without a candidate.
    // The probe checks the materialized chain and models the unbuilt suffix.
    const lastBuilt = entries.findLastIndex(entry => entry.headCommit);
    for (const [index, entry] of entries.entries()) {
      if (!entry.headCommit && index < lastBuilt) continue;
      const queued = entry.pullRequest;
      for (const path of await pathsFor(queued.number, queued.headRefOid)) {
        if (incoming.has(path) && !overlaps.has(path)) overlaps.set(path, queued.number);
      }
      if (overlaps.size === incoming.size) break;
    }
    if (!overlaps.size) return false;
    // Each probe has a 45s timeout. Stop starting probes after 45s of Git
    // work so worst-case total time stays under 90s for this sweep.
    if (budget.probeMs >= 45000) throw new Error('sweep Git probe budget reached');
    const started = now();
    let result;
    try {
      result = await probe({github, context, base, entries, head});
    } finally {
      budget.probeMs += Math.max(0, now() - started);
    }
    const path = result?.paths.find(path => overlaps.has(path));
    if (!path) return false;
    conflict = `Queue conflict on ${JSON.stringify(path)} (also changed by #${overlaps.get(path)}).`;
  } catch (error) {
    core?.info?.(`Dependency conflict check for #${pr.number} inconclusive: ${error.message}`);
    return false;
  }
  core?.info?.(`Deferring #${pr.number}: ${conflict}`);
  // The optional status records a historical observation. Only a receipt for
  // this exact snapshot can suppress another probe; it is never a merge gate.
  try {
    await github.rest.repos.createCommitStatus({owner, repo, sha: head, context: statusContext,
      state: 'success', description: conflict.slice(0, 140),
      target_url: `${prefix}${context.runId}#dependency-conflict=${fingerprint}`});
  } catch (error) {
    core?.info?.(`Could not record dependency conflict for #${pr.number}: ${error.message}`);
  }
  return true;
}

module.exports = {checkDependencyConflict};
