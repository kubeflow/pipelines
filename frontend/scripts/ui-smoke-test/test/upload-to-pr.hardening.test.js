const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const {
  commentMarker,
  findExistingComment,
  generateMarkdownSummary,
  parseCliArgs,
  postCommentToPR,
  runGh,
  validatePrNumber,
  validateRepository,
  validateScreenshotsDirectory,
  validateSummary,
} = require('../upload-to-pr');

function validSummary(overrides = {}) {
  const summary = {
    analysis: {
      looksSameClusterSize: 8,
      looksSameTolerance: 2.3,
    },
    fatalErrors: [],
    mainLabel: 'base: origin/master (aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)',
    passed: false,
    prLabel: 'PR #12756 (bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb)',
    results: [
      {
        diffPercent: 0.5,
        exceedsFailThreshold: false,
        hasVisualDiff: true,
        mainExists: true,
        page: 'pipelines',
        prExists: true,
        status: 'success',
      },
      {
        error: 'comparison failed | retry\nneeded',
        failureType: 'analysis',
        page: 'runs',
        status: 'failed',
      },
      {
        page: 'recurring-runs',
        reason: 'optional route unavailable',
        status: 'skipped',
      },
    ],
    stats: {
      analysisFailed: 1,
      corrupt: 0,
      dimensionMismatch: 0,
      failed: 1,
      integrity: 0,
      missing: 0,
      pagesExceedingFailThreshold: 0,
      pagesWithDiff: 1,
      skipped: 1,
      stale: 0,
      success: 1,
      total: 3,
    },
    thresholds: {
      diffThreshold: 0.25,
      failThreshold: 1,
    },
    schemaVersion: 2,
    sourceMode: 'manifest',
    timestamp: '2026-08-28T12:00:00.000Z',
    valid: false,
  };
  return { ...summary, ...overrides };
}

function createSummaryFixture(t, summary = validSummary()) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-upload-hardening-'));
  const comparisonDir = path.join(root, 'comparison');
  fs.mkdirSync(comparisonDir);
  fs.writeFileSync(path.join(comparisonDir, 'summary.json'), JSON.stringify(summary));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  return { comparisonDir, root };
}

test('CLI validation accepts expected inputs and rejects injection-shaped values', (t) => {
  const { comparisonDir, root } = createSummaryFixture(t);
  const parsed = parseCliArgs([
    '--pr',
    '12756',
    '--repo',
    'kubeflow/pipelines',
    '--screenshots',
    comparisonDir,
  ]);
  assert.equal(parsed.prNumber, '12756');
  assert.equal(parsed.repo, 'kubeflow/pipelines');
  assert.equal(parsed.directory, fs.realpathSync(comparisonDir));

  assert.throws(() => validatePrNumber('1;echo-owned'), /Invalid PR number/);
  assert.throws(() => validatePrNumber('0'), /Invalid PR number/);
  assert.throws(() => validateRepository('kubeflow/pipelines;echo-owned'), /Invalid GitHub/);
  assert.throws(() => validateRepository('not-an-owner-repo'), /owner\/name/);
  assert.throws(
    () => parseCliArgs(['--pr', '12', '--screenshots', path.join(root, 'missing')]),
    /does not exist/,
  );
});

test('summary.json must be a bounded regular file inside the comparison directory', (t) => {
  const { root } = createSummaryFixture(t);
  const directory = path.join(root, 'symlink-comparison');
  fs.mkdirSync(directory);
  const outsideSummary = path.join(root, 'outside-summary.json');
  fs.writeFileSync(outsideSummary, JSON.stringify(validSummary()));
  fs.symlinkSync(outsideSummary, path.join(directory, 'summary.json'));

  assert.throws(() => validateScreenshotsDirectory(directory), /inside the screenshots directory/);
});

test('summary validation rejects malformed schema and inconsistent counts', () => {
  assert.throws(
    () => validateSummary({ ...validSummary(), thresholds: undefined }),
    /summary.thresholds must be an object/,
  );
  assert.throws(
    () => validateSummary({ ...validSummary(), fatalErrors: ['capture failed'], valid: true }),
    /valid\/passed state is inconsistent/,
  );
  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        stats: { ...validSummary().stats, total: 99 },
      }),
    /must equal summary.results.length/,
  );
  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        results: [{ diffPercent: '1.2', page: 'pipelines', status: 'success' }],
        stats: {
          analysisFailed: 0,
          corrupt: 0,
          dimensionMismatch: 0,
          failed: 0,
          integrity: 0,
          missing: 0,
          pagesExceedingFailThreshold: 0,
          pagesWithDiff: 1,
          skipped: 0,
          stale: 0,
          success: 1,
          total: 1,
        },
      }),
    /diffPercent must be a number from 0 through 100/,
  );

  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        stats: { ...validSummary().stats, pagesWithDiff: 0 },
      }),
    /visual-difference statistics do not match/,
  );
  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        results: validSummary().results.map((result, index) =>
          index === 0 ? { ...result, exceedsFailThreshold: true } : result,
        ),
      }),
    /result flags do not match thresholds/,
  );
  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        thresholds: { ...validSummary().thresholds, diffThreshold: 101 },
      }),
    /0 through 100/,
  );
  assert.throws(
    () =>
      validateSummary({
        ...validSummary(),
        results: [{ page: 'pipelines', status: 'success' }],
        stats: {
          analysisFailed: 0,
          corrupt: 0,
          dimensionMismatch: 0,
          failed: 0,
          integrity: 0,
          missing: 0,
          pagesExceedingFailThreshold: 0,
          pagesWithDiff: 0,
          skipped: 0,
          stale: 0,
          success: 1,
          total: 1,
        },
      }),
    /diffPercent must be a number from 0 through 100/,
  );
});

test('markdown uses summary thresholds and handles failed or missing diffs', () => {
  const markdown = generateMarkdownSummary(
    validSummary({
      fatalErrors: ['base capture failed'],
    }),
    {
      prNumber: '12756',
      repo: 'kubeflow/pipelines',
    },
  );

  assert.match(markdown, /kubeflow-pipelines-ui-smoke-test:kubeflow\/pipelines#12756/);
  assert.match(markdown, /Diff Marker Threshold:\*\* 0\.25%/);
  assert.match(markdown, /Failure Threshold:\*\* 1%/);
  assert.match(markdown, /Base:\*\* <code>base: origin\/master \(a{40}\)<\/code>/);
  assert.match(markdown, /Head:\*\* <code>PR #12756 \(b{40}\)<\/code>/);
  assert.match(markdown, /Looks-same Color Tolerance \(ΔE\):\*\* 2\.3/);
  assert.match(markdown, /Overall:\*\* ❌ Invalid comparison/);
  assert.match(markdown, /pipelines.*0\.50%.*Visual change above 0\.25% marker threshold/);
  assert.match(markdown, /runs.*N\/A.*comparison failed \\| retry needed/);
  assert.match(markdown, /Skipped:\*\* 1/);
  assert.match(markdown, /recurring-runs.*⏭️.*N\/A.*optional route unavailable/);
  assert.doesNotMatch(markdown, /CI artifacts|ui-smoke-test-results artifact/);
  assert.match(markdown, /images are not attached by this tool/);
  assert.doesNotMatch(markdown, /skip-upload/);
  assert.ok(markdown.indexOf('### Fatal errors') < markdown.indexOf('### Page Results'));
  assert.ok(markdown.indexOf('### Page Results') < markdown.indexOf('| pipelines |'));
});

test('comment lookup paginates and only returns the authenticated author marker', () => {
  const repo = 'kubeflow/pipelines';
  const prNumber = '12756';
  const marker = commentMarker(repo, prNumber);
  const firstPage = Array.from({ length: 100 }, (_, index) => ({
    body: index === 10 ? marker : 'unrelated',
    id: index + 1,
    user: { login: index === 10 ? 'someone-else' : 'bot' },
  }));
  const secondPage = [
    { body: marker, id: 101, user: { login: 'viewer' } },
    { body: `${marker}\nnewer`, id: 102, user: { login: 'viewer' } },
    { body: `${marker} copied`, id: 103, user: { login: 'viewer' } },
    { body: `> ${marker}`, id: 104, user: { login: 'viewer' } },
    { body: `quoted report\n${marker}`, id: 105, user: { login: 'viewer' } },
  ];
  const calls = [];
  const fakeRunGh = (args) => {
    calls.push(args);
    const page = args[1].match(/[?&]page=(\d+)/)[1];
    return {
      stdout: JSON.stringify(page === '1' ? firstPage : secondPage),
      success: true,
    };
  };

  const comment = findExistingComment(repo, prNumber, 'viewer', marker, fakeRunGh);
  assert.equal(comment.id, 102);
  assert.equal(calls.length, 2);
  assert.ok(calls.every((args) => Array.isArray(args)));
  assert.match(calls[0][1], /per_page=100&page=1$/);
  assert.match(calls[1][1], /per_page=100&page=2$/);
});

test('comment lookup ignores quoted, embedded, or suffixed markers', () => {
  const repo = 'kubeflow/pipelines';
  const prNumber = '12756';
  const marker = commentMarker(repo, prNumber);
  const fakeRunGh = () => ({
    stdout: JSON.stringify([
      { body: `> ${marker}`, id: 1, user: { login: 'viewer' } },
      { body: `context\n${marker}`, id: 2, user: { login: 'viewer' } },
      { body: `${marker} from another report`, id: 3, user: { login: 'viewer' } },
    ]),
    success: true,
  });

  assert.equal(findExistingComment(repo, prNumber, 'viewer', marker, fakeRunGh), null);
});

test('posting updates only a matching viewer comment using argument arrays and stdin JSON', () => {
  const repo = 'kubeflow/pipelines';
  const prNumber = '12756';
  const marker = commentMarker(repo, prNumber);
  const calls = [];
  const fakeRunGh = (args, options = {}) => {
    calls.push({ args, options });
    if (args[0] === 'api' && args[1].includes('/comments?')) {
      return {
        stdout: JSON.stringify([
          { body: marker, id: 10, user: { login: 'someone-else' } },
          { body: marker, id: 20, user: { login: 'viewer' } },
        ]),
        success: true,
      };
    }
    return { stdout: '', success: true };
  };

  const result = postCommentToPR(
    `${marker}\nreport`,
    {
      prNumber,
      repo,
      viewerLogin: 'viewer',
    },
    fakeRunGh,
  );

  assert.deepEqual(result, { action: 'updated', commentId: '20' });
  assert.deepEqual(calls[1].args, [
    'api',
    'repos/kubeflow/pipelines/issues/comments/20',
    '--method',
    'PATCH',
    '--input',
    '-',
  ]);
  assert.deepEqual(JSON.parse(calls[1].options.input), { body: `${marker}\nreport` });
  assert.equal(
    calls.some((call) => call.args[0] === 'pr'),
    false,
  );
});

test('posting creates a comment when no authenticated marker exists', () => {
  const calls = [];
  const fakeRunGh = (args, options = {}) => {
    calls.push({ args, options });
    if (args[0] === 'api') {
      return { stdout: '[]', success: true };
    }
    return { stdout: '', success: true };
  };

  const result = postCommentToPR(
    'report body',
    {
      prNumber: '44',
      repo: 'kubeflow/pipelines',
      viewerLogin: 'viewer',
    },
    fakeRunGh,
  );
  assert.deepEqual(result, { action: 'created' });
  assert.deepEqual(calls[1].args, [
    'pr',
    'comment',
    '44',
    '--repo',
    'kubeflow/pipelines',
    '--body-file',
    '-',
  ]);
  assert.equal(calls[1].options.input, 'report body');
});

test('gh execution helper requires an argument array', () => {
  assert.throws(() => runGh('api user'), /string array/);
});
