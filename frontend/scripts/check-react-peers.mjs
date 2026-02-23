#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const REACT_PEER_KEYS = ['react', 'react-dom'];

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

function compareVersions(a, b) {
  if (a.major !== b.major) {
    return a.major - b.major;
  }
  if (a.minor !== b.minor) {
    return a.minor - b.minor;
  }
  return a.patch - b.patch;
}

function normalizeRange(range) {
  return String(range)
    .replace(/\u00a0/g, ' ')
    .replace(/,/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function splitDisjuncts(range) {
  return normalizeRange(range)
    .split('||')
    .map(part => part.trim())
    .filter(Boolean);
}

function expandHyphenRanges(disjunct) {
  return disjunct.replace(
    /(\d+(?:\.\d+){0,2}(?:-[0-9A-Za-z.-]+)?)\s*-\s*(\d+(?:\.\d+){0,2}(?:-[0-9A-Za-z.-]+)?)/g,
    '>=$1 <=$2',
  );
}

function parseVersionPattern(raw) {
  const cleaned = String(raw).trim().replace(/^v/i, '');
  const parts = cleaned.split('.');
  const isWildcard = value => value === 'x' || value === 'X' || value === '*';

  const parsed = [];
  for (const part of parts) {
    const bare = part.split('-')[0];
    if (isWildcard(bare)) {
      parsed.push('x');
      continue;
    }
    if (!/^\d+$/.test(bare)) {
      return null;
    }
    parsed.push(Number(bare));
  }

  while (parsed.length < 3) {
    parsed.push('x');
  }
  return parsed.slice(0, 3);
}

function parseExactVersion(raw, missingDefault = 0) {
  const cleaned = String(raw).trim().replace(/^v/i, '');
  const parts = cleaned.split('.');
  if (!parts.length || parts.length > 3) {
    return null;
  }
  const parsed = [];
  for (const part of parts) {
    const bare = part.split('-')[0];
    if (!/^\d+$/.test(bare)) {
      return null;
    }
    parsed.push(Number(bare));
  }
  while (parsed.length < 3) {
    parsed.push(missingDefault);
  }
  return { major: parsed[0], minor: parsed[1], patch: parsed[2] };
}

function nextMajor(version) {
  return { major: version.major + 1, minor: 0, patch: 0 };
}

function nextMinor(version) {
  return { major: version.major, minor: version.minor + 1, patch: 0 };
}

function nextPatch(version) {
  return { major: version.major, minor: version.minor, patch: version.patch + 1 };
}

function comparatorsFromToken(token) {
  if (token === '*' || token.toLowerCase() === 'x') {
    return [{ type: 'any' }];
  }

  const match = token.match(/^(\^|~|>=|<=|>|<|=)?(.+)$/);
  if (!match) {
    return null;
  }

  const op = match[1] ?? '';
  const versionRaw = match[2].trim();
  const hasExplicitWildcard = /(^|\.)(x|X|\*)($|\.|-)/.test(versionRaw);

  if (op === '' || op === '=') {
    const pattern = parseVersionPattern(versionRaw);
    if (!pattern) {
      return null;
    }
    const [major, minor, patch] = pattern;
    const hasWildcard = [major, minor, patch].includes('x');
    if (hasWildcard) {
      if (major === 'x') {
        return [{ type: 'any' }];
      }
      if (minor === 'x') {
        const low = { major, minor: 0, patch: 0 };
        const high = { major: major + 1, minor: 0, patch: 0 };
        return [
          { type: 'cmp', op: '>=', version: low },
          { type: 'cmp', op: '<', version: high },
        ];
      }
      const low = { major, minor, patch: 0 };
      const high = { major, minor: minor + 1, patch: 0 };
      return [
        { type: 'cmp', op: '>=', version: low },
        { type: 'cmp', op: '<', version: high },
      ];
    }
    const version = { major, minor, patch };
    return [{ type: 'cmp', op: '=', version }];
  }

  if (hasExplicitWildcard) {
    return null;
  }

  const version = parseExactVersion(versionRaw, 0);
  if (!version) {
    return null;
  }

  if (op === '^') {
    let upper;
    if (version.major > 0) {
      upper = nextMajor(version);
    } else if (version.minor > 0) {
      upper = nextMinor(version);
    } else {
      upper = nextPatch(version);
    }
    return [
      { type: 'cmp', op: '>=', version },
      { type: 'cmp', op: '<', version: upper },
    ];
  }
  if (op === '~') {
    return [
      { type: 'cmp', op: '>=', version },
      { type: 'cmp', op: '<', version: nextMinor(version) },
    ];
  }
  return [{ type: 'cmp', op, version }];
}

function tighterLower(current, candidate) {
  if (!current) {
    return candidate;
  }
  const cmp = compareVersions(candidate.version, current.version);
  if (cmp > 0) {
    return candidate;
  }
  if (cmp < 0) {
    return current;
  }
  if (!candidate.inclusive && current.inclusive) {
    return candidate;
  }
  return current;
}

function tighterUpper(current, candidate) {
  if (!current) {
    return candidate;
  }
  const cmp = compareVersions(candidate.version, current.version);
  if (cmp < 0) {
    return candidate;
  }
  if (cmp > 0) {
    return current;
  }
  if (!candidate.inclusive && current.inclusive) {
    return candidate;
  }
  return current;
}

function intervalHasValues(interval) {
  if (!interval.lower || !interval.upper) {
    return true;
  }
  const cmp = compareVersions(interval.lower.version, interval.upper.version);
  if (cmp < 0) {
    return true;
  }
  if (cmp > 0) {
    return false;
  }
  return interval.lower.inclusive && interval.upper.inclusive;
}

function disjunctSupportsMajor(disjunct, targetMajor) {
  const expanded = expandHyphenRanges(disjunct);
  const rawTokens = expanded.split(' ').map(token => token.trim()).filter(Boolean);
  const tokens = [];
  for (let i = 0; i < rawTokens.length; i++) {
    const token = rawTokens[i];
    if (['>', '>=', '<', '<=', '=', '^', '~'].includes(token) && i + 1 < rawTokens.length) {
      tokens.push(`${token}${rawTokens[i + 1]}`);
      i += 1;
      continue;
    }
    tokens.push(token);
  }

  let interval = {
    lower: { version: { major: targetMajor, minor: 0, patch: 0 }, inclusive: true },
    upper: { version: { major: targetMajor + 1, minor: 0, patch: 0 }, inclusive: false },
  };

  for (const token of tokens) {
    const comparators = comparatorsFromToken(token);
    if (!comparators) {
      return false;
    }
    for (const comparator of comparators) {
      if (comparator.type === 'any') {
        continue;
      }
      const { op, version } = comparator;
      if (op === '>') {
        interval.lower = tighterLower(interval.lower, { version, inclusive: false });
      } else if (op === '>=') {
        interval.lower = tighterLower(interval.lower, { version, inclusive: true });
      } else if (op === '<') {
        interval.upper = tighterUpper(interval.upper, { version, inclusive: false });
      } else if (op === '<=') {
        interval.upper = tighterUpper(interval.upper, { version, inclusive: true });
      } else if (op === '=') {
        interval.lower = tighterLower(interval.lower, { version, inclusive: true });
        interval.upper = tighterUpper(interval.upper, { version, inclusive: true });
      } else {
        return false;
      }
      if (!intervalHasValues(interval)) {
        return false;
      }
    }
  }

  return intervalHasValues(interval);
}

function rangeSupportsMajor(range, targetMajor) {
  if (!range || String(range).trim() === '') {
    return true;
  }
  const disjuncts = splitDisjuncts(range);
  if (!disjuncts.length) {
    return false;
  }
  return disjuncts.some(disjunct => disjunctSupportsMajor(disjunct, targetMajor));
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
  const data = readJson(filePath);
  const targetEntries = data?.[String(targetMajor)];
  if (!Array.isArray(targetEntries)) {
    return new Set();
  }
  const normalized = targetEntries
    .map(value => String(value).trim())
    .filter(Boolean);
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

  const packageJson = readJson(packageJsonPath);
  const lockfile = readJson(lockfilePath);
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
      console.error(
        `- ${entry.packageName}@${entry.version} (${entry.packagePath}) -> ${entry.peerSignature}`,
      );
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
