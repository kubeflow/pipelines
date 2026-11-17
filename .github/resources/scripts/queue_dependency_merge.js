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

const {execFile} = require('node:child_process');
const fs = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const {promisify} = require('node:util');

const exec = promisify(execFile);
const SHA = /^[0-9a-f]{40}$/;

async function probeDependencyConflict({context, base, entries, head}) {
  const {owner, repo} = context.repo;
  if (![owner, repo].every(name => typeof name === 'string' && /^[\w-][\w.-]*$/.test(name)) ||
      !SHA.test(base) || !SHA.test(head)) throw new Error('Invalid dependency merge probe target');

  let tip = base;
  const pending = [];
  for (const entry of entries) {
    if (!SHA.test(entry.pullRequest.headRefOid)) throw new Error('Invalid queued PR head');
    if (entry.headCommit) {
      // GitHub can build past an unmergeable entry. A candidate based on
      // the last materialized tip proves those intervening entries were skipped.
      if (entry.baseCommit?.oid !== tip || !SHA.test(entry.headCommit.oid)) return null;
      pending.length = 0;
      tip = entry.headCommit.oid;
    } else {
      pending.push(entry);
    }
  }
  // The advisory probe must not monopolize a recovery job on a long backlog.
  if (pending.length > 5 || (pending[0]?.baseCommit && pending[0].baseCommit.oid !== tip)) return null;
  const pendingHeads = pending.map(entry => entry.pullRequest.headRefOid);

  const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'kfp-queue-merge-'));
  const deadline = Date.now() + 45000;
  async function git(args, allowConflict = false) {
    const timeout = deadline - Date.now();
    if (timeout <= 0) throw new Error('Dependency merge probe timed out');
    try {
      const {stdout} = await exec('git', ['-c', 'protocol.allow=never',
        '-c', 'protocol.https.allow=always', ...args], {
        cwd: dir, timeout, maxBuffer: 1024 * 1024,
        env: {PATH: process.env.PATH, HOME: dir, GIT_CONFIG_NOSYSTEM: '1',
          GIT_CONFIG_GLOBAL: os.devNull, GIT_TERMINAL_PROMPT: '0', GIT_LFS_SKIP_SMUDGE: '1',
          GIT_AUTHOR_NAME: 'Queue probe', GIT_AUTHOR_EMAIL: 'probe@localhost',
          GIT_COMMITTER_NAME: 'Queue probe', GIT_COMMITTER_EMAIL: 'probe@localhost'},
      });
      return {stdout, conflict: false};
    } catch (error) {
      if (allowConflict && error.code === 1 && !error.killed) return {stdout: error.stdout, conflict: true};
      throw new Error(`Dependency merge probe failed during git ${args[0]}`);
    }
  }
  try {
    await git(['init', '--bare', '--template=', '.']);
    await git(['remote', 'add', 'origin', `https://github.com/${owner}/${repo}.git`]);
    await git(['config', 'remote.origin.promisor', 'true']);
    await git(['config', 'remote.origin.partialclonefilter', 'tree:0']);
    // Full commit ancestry gives merge-tree the real merge base. Fetch trees
    // and blobs on demand, without checking out or executing PR content.
    await git(['fetch', '--no-tags', '--no-recurse-submodules', '--filter=tree:0',
      'origin', ...new Set([tip, ...pendingHeads, head])]);
    const merge = async next => {
      const result = await git(['merge-tree', '--write-tree', '--name-only', '--no-messages', '-z', tip, next], true);
      const [tree, ...paths] = result.stdout.split('\0');
      if (!SHA.test(tree)) throw new Error('Invalid dependency merge-tree output');
      return {tree, conflict: result.conflict, paths: paths.filter(Boolean)};
    };
    for (const queuedHead of pendingHeads) {
      const result = await merge(queuedHead);
      if (result.conflict) return null;
      // Model the repository's squash merge. Each synthetic commit has only
      // the previous tip as parent, just like GitHub's materialized candidates.
      tip = (await git(['commit-tree', result.tree, '-p', tip, '-m', 'Queue merge probe'])).stdout.trim();
      if (!SHA.test(tip)) throw new Error('Invalid dependency merge probe commit');
    }
    const result = await merge(head);
    return result.conflict && result.paths.length ? {paths: result.paths} : null;
  } finally {
    await fs.rm(dir, {recursive: true, force: true});
  }
}

module.exports = {probeDependencyConflict};
