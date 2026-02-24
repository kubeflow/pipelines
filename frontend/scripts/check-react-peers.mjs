#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import semver from 'semver';

const REACT_PEER_KEYS = ['react', 'react-dom'];
const SEMVER_OPTIONS = { loose: true, includePrerelease: true };

function parseArgs(argv) {
  let targetMajor = null;
  let lockPath = 'package-lock.json';
  let allowlistPath = 'docs/react-peer-allowlist.json';
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--target' && i + 1 < argv.length) {
      targetMajor = Number(argv[++i]);
      continue;
    }
    if (arg.startsWith('--target=')) {
      targetMajor = Number(arg.slice('--target='.length));
      continue;
    }
    if (arg === '--lock' && i + 1 < argv.length) {
      lockPath = argv[++i];
      continue;
    }
    if (arg.startsWith('--lock=')) {
      lockPath = arg.slice('--lock='.length);
      continue;
    }
    if (arg === '--allowlist' && i + 1 < argv.length) {
      allowlistPath = argv[++i];
      continue;
    }
    if (arg.startsWith('--allowlist=')) {
      allowlistPath = arg.slice('--allowlist='.length);
      continue;
    }
    if (arg === '-h' || arg === '--help') {
      printHelp();
      process.exit(0);
    }
    throw new Error(`Unknown argument: ${arg}`);
  }

  if (!Number.isInteger(targetMajor) || targetMajor <= 0) {
    throw new Error('Missing or invalid --target value (expected positive integer major).');
  }

  return { targetMajor, lockPath, allowlistPath };
}

function printHelp() {
  console.log(`
Check whether all lockfile packages with React peer dependencies support a target React major.

Usage:
  node scripts/check-react-peers.mjs --target <major> [--lock package-lock.json] [--allowlist docs/react-peer-allowlist.json]

Examples:
  node scripts/check-react-peers.mjs --target 17
  node scripts/check-react-peers.mjs --target 17 --allowlist docs/react-peer-allowlist.json
  node scripts/check-react-peers.mjs --target 18 --lock frontend/package-lock.json
  `.trim());
}

function normalizeRange(range) {
  return String(range)
    .replace(/\u00a0/g, ' ')
    .replace(/,/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function normalizeSemverRange(range) {
  return normalizeRange(range).replace(/([<>~^]=?|=)\s+(\d)/g, '$1$2');
}

function rangeSupportsMajor(range, targetMajor) {
  if (!range || String(range).trim() === '') {
    return true;
  }

  const normalizedRange = normalizeSemverRange(range);
  const validRange = semver.validRange(normalizedRange, SEMVER_OPTIONS);
  if (!validRange) {
    return false;
  }

  const targetMajorRange = `>=${targetMajor}.0.0 <${targetMajor + 1}.0.0`;
  return semver.intersects(validRange, targetMajorRange, SEMVER_OPTIONS);
}

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function packageNameFromPath(packagePath, fallbackName) {
  if (fallbackName) {
    return fallbackName;
  }
  if (!packagePath || packagePath === '') {
    return '(root)';
  }
  if (!packagePath.startsWith('node_modules/')) {
    return packagePath;
  }
  const withoutTopLevelPrefix = packagePath.slice('node_modules/'.length);
  const segments = withoutTopLevelPrefix.split('/node_modules/');
  const remainder = segments[segments.length - 1];
  if (remainder.startsWith('@')) {
    const parts = remainder.split('/');
    return parts.length >= 2 ? `${parts[0]}/${parts[1]}` : remainder;
  }
  return remainder.split('/')[0];
}

function normalizePeerSignature(failingPeers) {
  return failingPeers
    .map(peer => `${peer.key}=${peer.range}`)
    .sort()
    .join(', ');
}

function readAllowlist(filePath, targetMajor) {
  if (!fs.existsSync(filePath)) {
    return new Set();
  }
  let data;
  try {
    data = readJson(filePath);
  } catch (error) {
    console.error(`ERROR: Failed to parse allowlist JSON at ${filePath}: ${error.message}`);
    process.exit(2);
  }
  const targetEntries = data?.[String(targetMajor)];
  if (!Array.isArray(targetEntries)) {
    return new Set();
  }
  const normalized = targetEntries.map(value => String(value).trim()).filter(Boolean);
  return new Set(normalized);
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

  const cwd = process.cwd();
  const lockfilePath = path.resolve(cwd, args.lockPath);
  const packageJsonPath = path.resolve(cwd, 'package.json');
  const allowlistFilePath = path.resolve(cwd, args.allowlistPath);

  if (!fs.existsSync(lockfilePath)) {
    console.error(`ERROR: Lockfile not found at ${lockfilePath}`);
    process.exit(2);
  }
  if (!fs.existsSync(packageJsonPath)) {
    console.error(`ERROR: package.json not found at ${packageJsonPath}`);
    process.exit(2);
  }

  let packageJson;
  try {
    packageJson = readJson(packageJsonPath);
  } catch (error) {
    console.error(`ERROR: Failed to parse package.json at ${packageJsonPath}: ${error.message}`);
    process.exit(2);
  }

  let lockfile;
  try {
    lockfile = readJson(lockfilePath);
  } catch (error) {
    console.error(`ERROR: Failed to parse lockfile at ${lockfilePath}: ${error.message}`);
    process.exit(2);
  }

  const allowlist = readAllowlist(allowlistFilePath, args.targetMajor);
  const directDependencies = new Set([
    ...Object.keys(packageJson.dependencies || {}),
    ...Object.keys(packageJson.devDependencies || {}),
  ]);

  const unsupported = [];
  const lockPackages = lockfile.packages || {};

  for (const [packagePath, meta] of Object.entries(lockPackages)) {
    if (!meta || !meta.peerDependencies) {
      continue;
    }
    const peerDependencies = meta.peerDependencies;
    const failingPeers = [];
    for (const key of REACT_PEER_KEYS) {
      if (!(key in peerDependencies)) {
        continue;
      }
      const peerRange = peerDependencies[key];
      if (!rangeSupportsMajor(peerRange, args.targetMajor)) {
        failingPeers.push({ key, range: peerRange });
      }
    }

    if (!failingPeers.length) {
      continue;
    }

    const packageName = packageNameFromPath(packagePath, meta.name);
    unsupported.push({
      packagePath,
      packageName,
      version: meta.version || '(unknown)',
      direct: directDependencies.has(packageName) && packagePath === `node_modules/${packageName}`,
      failingPeers,
    });
  }

  unsupported.sort((a, b) => a.packagePath.localeCompare(b.packagePath));

  const allowed = [];
  const blocked = [];
  for (const entry of unsupported) {
    const peerSignature = normalizePeerSignature(entry.failingPeers);
    const allowlistKey = `${entry.packageName}@${entry.version}::${peerSignature}`;
    if (allowlist.has(allowlistKey)) {
      allowed.push({ ...entry, peerSignature, allowlistKey });
    } else {
      blocked.push({ ...entry, peerSignature, allowlistKey });
    }
  }

  const directUnsupported = blocked.filter(entry => entry.direct);
  const transitiveUnsupported = blocked.filter(entry => !entry.direct);

  const usedAllowlistKeys = new Set(allowed.map(entry => entry.allowlistKey));
  const staleAllowlist = [...allowlist].filter(key => !usedAllowlistKeys.has(key)).sort();

  if (!blocked.length) {
    console.log(
      `PASS: all non-allowlisted lockfile React peer ranges support React ${args.targetMajor} (checked ${Object.keys(lockPackages).length} package entries).`,
    );
    if (allowed.length) {
      console.log(`INFO: ${allowed.length} unsupported entries are allowlisted in ${args.allowlistPath}.`);
    }
    if (staleAllowlist.length) {
      console.log(`INFO: ${staleAllowlist.length} allowlist entries are stale and can be removed.`);
    }
    return;
  }

  console.error(
    `FAIL: found ${blocked.length} non-allowlisted package entries with React peer ranges excluding React ${args.targetMajor}.`,
  );
  if (allowed.length) {
    console.error(`INFO: ${allowed.length} additional unsupported entries are allowlisted.`);
  }
  if (directUnsupported.length) {
    console.error(`\nDirect dependencies (${directUnsupported.length}):`);
    for (const entry of directUnsupported) {
      console.error(`- ${entry.packageName}@${entry.version} (${entry.packagePath}) -> ${entry.peerSignature}`);
    }
  }

  if (transitiveUnsupported.length) {
    const grouped = new Map();
    for (const entry of transitiveUnsupported) {
      const key = `${entry.packageName}@${entry.version}::${entry.peerSignature}`;
      if (!grouped.has(key)) {
        grouped.set(key, {
          packageName: entry.packageName,
          version: entry.version,
          peerSignature: entry.peerSignature,
          occurrences: 0,
        });
      }
      grouped.get(key).occurrences += 1;
    }

    const rows = [...grouped.values()].sort((a, b) =>
      `${a.packageName}@${a.version}`.localeCompare(`${b.packageName}@${b.version}`),
    );
    console.error(`\nTransitive dependencies (${rows.length} unique signatures):`);
    for (const row of rows) {
      console.error(
        `- ${row.packageName}@${row.version} -> ${row.peerSignature} (occurrences: ${row.occurrences})`,
      );
    }
  }

  if (staleAllowlist.length) {
    console.error(`\nStale allowlist entries (${staleAllowlist.length}):`);
    for (const value of staleAllowlist) {
      console.error(`- ${value}`);
    }
  }

  process.exit(1);
}

main();
