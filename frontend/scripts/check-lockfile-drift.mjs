#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const DEP_FIELDS = [
  'dependencies',
  'devDependencies',
  'optionalDependencies',
  'overrides',
  'workspaces',
];

const TOOLCHAIN_PACKAGES = ['prettier', 'eslint', 'typescript', 'vitest'];
const TOOLCHAIN_PREFIXES = ['@typescript-eslint/'];
const SILENT_STDIO = ['pipe', 'pipe', 'pipe'];

function parseArgs(argv) {
  let baseRef = null;
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--base-ref' && i + 1 < argv.length) {
      baseRef = argv[++i];
      continue;
    }
    if (arg.startsWith('--base-ref=')) {
      baseRef = arg.slice('--base-ref='.length);
      continue;
    }
    if (arg === '-h' || arg === '--help') {
      printHelp();
      process.exit(0);
    }
    throw new Error(`Unknown argument: ${arg}`);
  }
  if (!baseRef) {
    throw new Error('Missing --base-ref argument.');
  }
  return { baseRef };
}

function printHelp() {
  console.log(
    `Detect CI-sensitive toolchain version drift in package-lock.json.

Compares the resolved versions of critical CI tooling packages (prettier,
eslint, typescript, vitest, @typescript-eslint/*) between the current
working tree and a base git ref. Fails only when those versions differ
and no dependency-bearing field in package.json was intentionally changed
by this branch (checked against the merge base, not the base branch tip).

Run from the directory containing the package.json and package-lock.json
to check (typically frontend/).

Usage:
  node scripts/check-lockfile-drift.mjs --base-ref <git-ref>

Examples:
  node scripts/check-lockfile-drift.mjs --base-ref origin/master`.trim(),
  );
}

function gitShow(ref, filePath) {
  try {
    return execFileSync('git', ['show', `${ref}:${filePath}`], {
      encoding: 'utf8',
      stdio: SILENT_STDIO,
    });
  } catch {
    return null;
  }
}

function extractDepFields(packageJson) {
  const result = {};
  for (const field of DEP_FIELDS) {
    if (packageJson[field] !== undefined) {
      result[field] = packageJson[field];
    }
  }
  return result;
}

function isToolchainPackage(name) {
  if (TOOLCHAIN_PACKAGES.includes(name)) return true;
  return TOOLCHAIN_PREFIXES.some((prefix) => name.startsWith(prefix));
}

function extractToolchainVersions(lockfile) {
  const versions = {};
  const packages = lockfile.packages || {};
  for (const [packagePath, meta] of Object.entries(packages)) {
    if (!packagePath.startsWith('node_modules/')) continue;
    const name = packagePath.slice('node_modules/'.length);
    if (name.includes('node_modules/')) continue;
    if (isToolchainPackage(name) && meta.version) {
      versions[name] = meta.version;
    }
  }
  return versions;
}

function parseJsonOrExit(content, label) {
  try {
    return JSON.parse(content);
  } catch (error) {
    console.error(`ERROR: failed to parse ${label}: ${error.message}`);
    process.exit(2);
  }
}

function formatDriftedPackages(packages) {
  return packages.map(
    ({ name, baseVersion, currentVersion }) => `  ${name}: ${baseVersion} \u2192 ${currentVersion}`,
  );
}

function computeMergeBase(baseRef) {
  try {
    return execFileSync('git', ['merge-base', baseRef, 'HEAD'], {
      encoding: 'utf8',
      stdio: SILENT_STDIO,
    }).trim();
  } catch {
    return null;
  }
}

function verifyLockfileSync(lockfilePath) {
  const original = fs.readFileSync(lockfilePath, 'utf8');
  try {
    execFileSync('npm', ['install', '--package-lock-only', '--ignore-scripts'], {
      stdio: SILENT_STDIO,
      cwd: path.dirname(lockfilePath),
    });
  } catch {
    fs.writeFileSync(lockfilePath, original);
    return { ok: true, skipped: true };
  }
  const regenerated = fs.readFileSync(lockfilePath, 'utf8');
  fs.writeFileSync(lockfilePath, original);
  return { ok: original === regenerated, skipped: false };
}

function main() {
  let args;
  try {
    args = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(`ERROR: ${error.message}`);
    printHelp();
    process.exit(2);
  }

  const { baseRef } = args;
  const cwd = process.cwd();
  const packageJsonPath = path.resolve(cwd, 'package.json');
  const lockfilePath = path.resolve(cwd, 'package-lock.json');

  if (!fs.existsSync(packageJsonPath)) {
    console.error(`ERROR: package.json not found at ${packageJsonPath}`);
    process.exit(2);
  }
  if (!fs.existsSync(lockfilePath)) {
    console.error(`ERROR: package-lock.json not found at ${lockfilePath}`);
    process.exit(2);
  }

  const currentPkg = parseJsonOrExit(fs.readFileSync(packageJsonPath, 'utf8'), 'package.json');
  const currentLock = parseJsonOrExit(fs.readFileSync(lockfilePath, 'utf8'), 'package-lock.json');

  const repoRoot = execFileSync('git', ['rev-parse', '--show-toplevel'], {
    encoding: 'utf8',
  }).trim();
  const gitPrefix = path.relative(repoRoot, cwd);
  const gitPkgPath = gitPrefix ? `${gitPrefix}/package.json` : 'package.json';
  const gitLockPath = gitPrefix ? `${gitPrefix}/package-lock.json` : 'package-lock.json';

  const baseLockContent = gitShow(baseRef, gitLockPath);

  if (!baseLockContent) {
    console.log('SKIP: could not read base branch lockfile. Nothing to compare.');
    process.exit(0);
  }

  const baseLock = parseJsonOrExit(baseLockContent, 'base branch package-lock.json');

  const mergeBase = computeMergeBase(baseRef);
  const depsRef = mergeBase || baseRef;
  if (!mergeBase) {
    console.log(
      'WARNING: could not compute merge base (shallow clone?). ' +
        'Falling back to base branch tip for deps comparison.',
    );
  }

  const depsRefPkgContent = gitShow(depsRef, gitPkgPath);
  if (!depsRefPkgContent) {
    console.log('SKIP: could not read package.json from deps baseline. Nothing to compare.');
    process.exit(0);
  }

  const depsRefPkg = parseJsonOrExit(depsRefPkgContent, 'baseline package.json');

  const currentDeps = extractDepFields(currentPkg);
  const baselineDeps = extractDepFields(depsRefPkg);
  const depsChanged = JSON.stringify(currentDeps) !== JSON.stringify(baselineDeps);

  const currentToolchain = extractToolchainVersions(currentLock);
  const baseToolchain = extractToolchainVersions(baseLock);

  const allToolchainNames = new Set([
    ...Object.keys(currentToolchain),
    ...Object.keys(baseToolchain),
  ]);

  const driftedPackages = [];
  for (const name of [...allToolchainNames].sort()) {
    const baseVersion = baseToolchain[name] || '(not present)';
    const currentVersion = currentToolchain[name] || '(not present)';
    if (baseVersion !== currentVersion) {
      driftedPackages.push({ name, baseVersion, currentVersion });
    }
  }

  if (depsChanged) {
    const sync = verifyLockfileSync(lockfilePath);
    if (sync.skipped) {
      console.log(
        'WARNING: could not run lockfile sync verification (npm install failed). Skipping.',
      );
    } else if (!sync.ok) {
      console.error('FAIL: package-lock.json is out of sync with package.json.');
      console.error("Run 'npm install' to regenerate your lockfile, then commit the result.");
      process.exit(1);
    }

    if (driftedPackages.length) {
      console.log(
        'PASS: toolchain versions changed alongside dependency-bearing manifest fields (intentional update).',
      );
      console.log('Changed toolchain packages:');
      formatDriftedPackages(driftedPackages).forEach((line) => console.log(line));
    } else {
      console.log('PASS: dependency-bearing fields changed; lockfile is in sync.');
    }
    return;
  }

  if (!driftedPackages.length) {
    console.log('PASS: no CI-sensitive toolchain version drift detected in package-lock.json.');
    return;
  }

  console.error(
    'FAIL: CI-sensitive toolchain versions in package-lock.json have drifted from the base branch',
  );
  console.error('without a corresponding change to dependency-bearing fields in package.json.');
  console.error('');
  console.error(
    'This can cause silent formatting/linting/testing mismatches between CI and local development.',
  );
  console.error('');
  console.error('Drifted packages:');
  formatDriftedPackages(driftedPackages).forEach((line) => console.error(line));
  console.error('');
  console.error(
    `To fix: rebase onto ${baseRef.replace(/^origin\//, '')} and run 'npm ci' to sync your lockfile.`,
  );
  process.exit(1);
}

main();
