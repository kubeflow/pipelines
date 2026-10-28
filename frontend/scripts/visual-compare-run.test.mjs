// @vitest-environment node

/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { afterEach, beforeEach, expect, it } from 'vitest';

let root;
let wrapper;
let bin;

beforeEach(() => {
  root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-visual-wrapper-'));
  const scripts = path.join(root, 'frontend/scripts');
  bin = path.join(root, 'bin');
  fs.mkdirSync(scripts, { recursive: true });
  fs.mkdirSync(bin);
  fs.mkdirSync(path.join(root, 'baseline/frontend'), { recursive: true });
  fs.writeFileSync(path.join(root, 'frontend/.nvmrc'), '24.14.0');
  fs.writeFileSync(
    path.join(root, 'baseline/frontend/package.json'),
    '{"scripts":{"start": "vite"}}',
  );
  wrapper = path.join(scripts, 'visual-compare-run.sh');
  fs.copyFileSync(new URL('./visual-compare-run.sh', import.meta.url), wrapper);
  for (const command of ['curl', 'git', 'fnm']) {
    fs.writeFileSync(path.join(bin, command), '#!/bin/sh\nexit 0\n', { mode: 0o755 });
  }
  fs.writeFileSync(path.join(bin, 'node'), '#!/bin/sh\necho "${TEST_NODE_VERSION:-24.14.0}"\n', {
    mode: 0o755,
  });
  // Stub every npm operation: no real servers or browsers are launched.
  fs.writeFileSync(
    path.join(bin, 'npm'),
    `#!/bin/sh
if [ -n "$CAPTURE_ARGS" ]; then
  case "$4" in
    visual:baseline|visual:current|visual:diff) printf '%s\\n' "$*" >> "$CAPTURE_ARGS" ;;
  esac
fi
case "$4" in
  visual:baseline) echo baseline >> "$STEPS"; exit "$BASELINE_STATUS" ;;
  visual:current) echo current >> "$STEPS"; exit "$CURRENT_STATUS" ;;
  visual:diff)
    echo diff >> "$STEPS"
    echo 'comparison diagnostics' > "$REPORT"
    exit "$DIFF_STATUS"
    ;;
esac
exit 0
`,
    { mode: 0o755 },
  );
});

afterEach(() => {
  fs.rmSync(root, { recursive: true, force: true });
});

it.each([
  ['all successful', 0, 0, 0, 0],
  ['baseline failed', 7, 0, 0, 1],
  ['current failed', 0, 8, 0, 1],
  ['both captures failed', 7, 8, 0, 1],
  ['diff failed', 0, 0, 9, 1],
])(
  '%s: attempts both captures and the report, preserving failure',
  (_, baseline, current, diff, exit) => {
    const steps = path.join(root, 'steps');
    const report = path.join(root, 'frontend/.visual/report.html');
    const result = spawnSync('/bin/bash', [wrapper, 'test-base'], {
      cwd: root,
      encoding: 'utf8',
      timeout: 10000,
      env: {
        ...process.env,
        PATH: `${bin}:/usr/bin:/bin`,
        FNM_DIR: path.join(root, 'no-node-installation'),
        BASE_WORKTREE: path.join(root, 'baseline'),
        USE_MOCK: '0',
        SKIP_INSTALL: '1',
        STEPS: steps,
        REPORT: report,
        BASELINE_STATUS: String(baseline),
        CURRENT_STATUS: String(current),
        DIFF_STATUS: String(diff),
      },
    });
    expect(result.error).toBeUndefined();
    expect(fs.readFileSync(steps, 'utf8')).toBe('baseline\ncurrent\ndiff\n');
    expect(fs.readFileSync(report, 'utf8')).toContain('comparison diagnostics');
    expect(result.status, result.stderr).toBe(exit);
    if (baseline) expect(result.stderr).toContain(`Baseline capture failed (exit ${baseline})`);
    if (current) expect(result.stderr).toContain(`Current capture failed (exit ${current})`);
    if (diff) expect(result.stderr).toContain(`Visual diff/report failed (exit ${diff})`);
  },
);

it.each([
  ['fnm', 23],
  ['tr', 24],
])('%s failure prevents capture with fallback Node', (command, status) => {
  fs.writeFileSync(path.join(bin, command), `#!/bin/sh\nexit ${status}\n`, { mode: 0o755 });
  const steps = path.join(root, 'steps');
  const result = spawnSync('/bin/bash', [wrapper, 'test-base'], {
    encoding: 'utf8',
    timeout: 10000,
    env: {
      ...process.env,
      PATH: `${bin}:/usr/bin:/bin`,
      FNM_DIR: path.join(root, 'no-node-installation'),
      BASE_WORKTREE: path.join(root, 'baseline'),
      USE_MOCK: '0',
      SKIP_INSTALL: '1',
      STEPS: steps,
      REPORT: path.join(root, 'frontend/.visual/report.html'),
      BASELINE_STATUS: '0',
      CURRENT_STATUS: '0',
      DIFF_STATUS: '0',
    },
  });
  expect(result.error).toBeUndefined();
  expect(result.status, result.stderr).toBe(status);
  expect(fs.existsSync(steps)).toBe(false);
  expect(result.stderr.trim()).toBe(
    `Node toolchain setup failed (exit ${status}); comparison not started.`,
  );
  expect(fs.existsSync(path.join(root, 'frontend/.visual'))).toBe(false);
});

it.each([undefined, '', '2026-10-06T08:15:00.000Z'])(
  'resolves per-side routes and the selected capture clock (override: %s)',
  (fixedTime) => {
    // The baseline retains its own Node version; current setup must happen only
    // once even though the current server and three comparison commands use it.
    fs.writeFileSync(path.join(root, 'baseline/frontend/.nvmrc'), '22.15.0');
    fs.writeFileSync(
      path.join(bin, 'fnm'),
      `#!/bin/sh
if [ "$2" = "24.14.0" ]; then
  mkdir "$CURRENT_SETUP_ONCE" || exit 77
elif [ "$2" != "22.15.0" ]; then
  exit 78
fi
`,
      { mode: 0o755 },
    );
    const captureArgs = path.join(root, 'capture-args');
    const inheritedEnv = { ...process.env };
    delete inheritedEnv.FIXED_TIME;
    delete inheritedEnv.BASE_ROUTES;
    const result = spawnSync('/bin/bash', [wrapper, 'test-base'], {
      cwd: root,
      encoding: 'utf8',
      timeout: 10000,
      env: {
        ...inheritedEnv,
        PATH: `${bin}:/usr/bin:/bin`,
        FNM_DIR: path.join(root, 'no-node-installation'),
        BASE_WORKTREE: path.join(root, 'baseline'),
        USE_MOCK: '0',
        SKIP_INSTALL: '1',
        STEPS: path.join(root, 'steps'),
        REPORT: path.join(root, 'frontend/.visual/report.html'),
        BASELINE_STATUS: '0',
        CURRENT_STATUS: '0',
        DIFF_STATUS: '0',
        CAPTURE_ARGS: captureArgs,
        CURRENT_SETUP_ONCE: path.join(root, 'current-node-setup'),
        ...(fixedTime === undefined ? {} : { FIXED_TIME: fixedTime }),
        ROUTES: 'frontend/scripts/custom-routes.json',
        ...(fixedTime ? { BASE_ROUTES: 'frontend/scripts/legacy-routes.json' } : {}),
      },
    });
    expect(result.error).toBeUndefined();
    expect(result.status, result.stderr).toBe(0);
    const invocations = fs.readFileSync(captureArgs, 'utf8').trim().split('\n');
    expect(invocations).toHaveLength(3);
    for (const [index, mode] of ['baseline', 'current'].entries()) {
      expect(invocations[index]).toContain(`run visual:${mode} --`);
      expect(invocations[index]).toContain(`--out-dir ${root}/frontend/.visual/${mode}`);
      expect(invocations[index]).toContain(
        `--routes ${fs.realpathSync(root)}/frontend/scripts/${index === 0 && fixedTime ? 'legacy' : 'custom'}-routes.json`,
      );
      if (fixedTime === '') {
        expect(invocations[index]).not.toContain('--fixed-time');
      } else {
        expect(invocations[index]).toContain(
          `--fixed-time ${fixedTime ?? '2026-09-26T12:00:00.000Z'}`,
        );
      }
    }
    for (const [flag, relativePath] of [
      ['baseline-dir', 'baseline'],
      ['current-dir', 'current'],
      ['diff-dir', 'diff'],
      ['side-by-side-dir', 'side-by-side'],
      ['report', 'report.html'],
    ]) {
      expect(invocations[2]).toContain(`--${flag} ${root}/frontend/.visual/${relativePath}`);
    }
  },
);

it.each(['22.15.0', '24.1.0', 'not-a-version'])(
  'rejects unsupported actual Node %s before starting servers',
  (version) => {
    const result = spawnSync('/bin/bash', [wrapper, 'test-base'], {
      encoding: 'utf8',
      timeout: 10000,
      env: {
        ...process.env,
        PATH: `${bin}:/usr/bin:/bin`,
        FNM_DIR: path.join(root, 'no-node-installation'),
        TEST_NODE_VERSION: version,
      },
    });
    expect(result.error).toBeUndefined();
    expect(result.status).toBe(1);
    expect(result.stderr.trim()).toBe(
      `Node toolchain setup failed (exit 1); comparison not started. Node >=24.2.0 is required for capture/report; found ${version}.`,
    );
    expect(fs.existsSync(path.join(root, 'frontend/.visual'))).toBe(false);
  },
);
