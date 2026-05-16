#!/usr/bin/env node
/**
 * Coverage baseline capture and compare for issue #12895.
 *
 * Capture: npm run coverage:baseline
 *   Runs coverage and saves summary to .coverage-baseline.json
 *
 * Compare: npm run coverage:compare
 *   Runs coverage and fails if line/branch coverage decreased from baseline
 */
import { readFileSync, writeFileSync, existsSync } from 'fs';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, '..');
const BASELINE_PATH = join(ROOT, '.coverage-baseline.json');

function runCoverage() {
  execSync('npm run test:ui:coverage', {
    cwd: ROOT,
    stdio: 'inherit',
  });
}

function readCoverageSummary() {
  const v8Path = join(ROOT, 'coverage', 'vitest', 'coverage-summary.json');
  if (!existsSync(v8Path)) {
    throw new Error(
      'Coverage summary not found. Run "npm run test:ui:coverage" first.',
    );
  }
  return JSON.parse(readFileSync(v8Path, 'utf8'));
}

function extractTotals(summary) {
  const total = summary.total;
  return {
    lines: total?.lines?.pct ?? 0,
    branches: total?.branches?.pct ?? 0,
    functions: total?.functions?.pct ?? 0,
    statements: total?.statements?.pct ?? 0,
  };
}

if (process.argv[2] === 'capture') {
  runCoverage();
  const summary = readCoverageSummary();
  const totals = extractTotals(summary);
  writeFileSync(BASELINE_PATH, JSON.stringify(totals, null, 2));
  console.log('Baseline saved to .coverage-baseline.json:', totals);
} else if (process.argv[2] === 'compare') {
  if (!existsSync(BASELINE_PATH)) {
    console.error(
      'No baseline found. Run "npm run coverage:baseline" first.',
    );
    process.exit(1);
  }
  runCoverage();
  const baseline = JSON.parse(readFileSync(BASELINE_PATH, 'utf8'));
  const current = extractTotals(readCoverageSummary());
  let failed = false;
  for (const [key, value] of Object.entries(baseline)) {
    const curr = current[key];
    if (curr < value) {
      console.error(
        `Coverage decreased: ${key} ${value}% -> ${curr}%`,
      );
      failed = true;
    }
  }
  if (failed) {
    process.exit(1);
  }
  console.log('Coverage check passed:', current);
} else {
  console.log('Usage: node coverage-baseline.mjs capture|compare');
  process.exit(1);
}
