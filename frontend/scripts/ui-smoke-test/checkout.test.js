const { execFileSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { verifyCachedCheckout, createBaseWorktree } = require('./checkout');

let directory;
let repo;

function git(...args) {
  return execFileSync('git', args, {
    cwd: repo,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

beforeEach(() => {
  directory = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-smoke-checkout-'));
  repo = path.join(directory, 'repo');
  fs.mkdirSync(repo);
  git('init', '--initial-branch=main');
  // Identity is confined to disposable test fixtures, never the user's repository.
  git(
    '-c',
    'user.name=Fixture',
    '-c',
    'user.email=fixture@example.invalid',
    '-c',
    'commit.gpgsign=false',
    'commit',
    '--allow-empty',
    '-m',
    'base',
  );
});

afterEach(() => {
  fs.rmSync(directory, { recursive: true, force: true });
});

describe('UI smoke comparison checkouts', () => {
  it('creates a detached base while main is already checked out', () => {
    const target = path.join(directory, 'base');
    createBaseWorktree(repo, target, 'main');
    expect(
      execFileSync('git', ['rev-parse', 'HEAD'], {
        cwd: target,
        encoding: 'utf8',
      }).trim(),
    ).toBe(git('rev-parse', 'main'));
    expect(
      execFileSync('git', ['branch', '--show-current'], {
        cwd: target,
        encoding: 'utf8',
      }).trim(),
    ).toBe('');
    expect(git('branch', '--show-current')).toBe('main');
  });

  it('rejects a cached master clone when main is requested', () => {
    git('branch', '-m', 'main', 'master');
    expect(() => verifyCachedCheckout(repo, 'main')).toThrow(
      'Move this cache directory aside and rerun',
    );
    expect(git('branch', '--show-current')).toBe('master');
  });

  it('rejects a cache checked out at a different commit', () => {
    git('checkout', '-b', 'other');
    git(
      '-c',
      'user.name=Fixture',
      '-c',
      'user.email=fixture@example.invalid',
      '-c',
      'commit.gpgsign=false',
      'commit',
      '--allow-empty',
      '-m',
      'other',
    );
    expect(() => verifyCachedCheckout(repo, 'main')).toThrow('does not match main');
  });

  it('accepts matching main and explicit tag or SHA refs', () => {
    git('tag', '2.18.0');
    for (const ref of ['main', '2.18.0', git('rev-parse', 'HEAD')]) {
      expect(() => verifyCachedCheckout(repo, ref)).not.toThrow();
    }
  });

  it('preserves explicit remote refs when no local main branch exists', () => {
    const sha = git('rev-parse', 'HEAD');
    git('update-ref', 'refs/remotes/origin/main', sha);
    git('branch', '-m', 'main', 'feature');
    const target = path.join(directory, 'base');
    createBaseWorktree(repo, target, 'origin/main');
    expect(
      execFileSync('git', ['rev-parse', 'HEAD'], {
        cwd: target,
        encoding: 'utf8',
      }).trim(),
    ).toBe(sha);
  });
});

// Exercise the runner function without starting its CLI, servers, or cluster.
function loadCheckoutAndBuild(skipBuild, commands) {
  const source = fs.readFileSync(path.join(__dirname, 'smoke-test-runner.js'), 'utf8');
  const start = source.indexOf('async function checkoutAndBuild(');
  const end = source.indexOf('\nasync function serveBuild(', start);
  return require('vm').runInNewContext(`(${source.slice(start, end).trim()})`, {
    fs,
    path,
    verifyCachedCheckout,
    SKIP_BUILD: skipBuild,
    log() {},
    run(command) {
      commands.push(command);
      return { success: true };
    },
  });
}

describe('UI smoke cached build reuse', () => {
  it('rejects the wrong cached branch even with --skip-build', async () => {
    git('branch', '-m', 'main', 'master');
    const commands = [];
    await expect(loadCheckoutAndBuild(true, commands)('main', repo, 'main')).rejects.toThrow(
      'does not match main',
    );
    expect(commands).toEqual([]);
  });

  it('checks the cached ref before running npm', async () => {
    git('branch', '-m', 'main', 'master');
    const commands = [];
    await expect(loadCheckoutAndBuild(false, commands)('main', repo, 'main')).rejects.toThrow(
      'does not match main',
    );
    expect(commands).toEqual([]);
  });

  it('reuses a matching cached build when explicitly requested', async () => {
    git('tag', '2.18.0');
    const commands = [];
    await expect(loadCheckoutAndBuild(true, commands)('2.18.0', repo, 'base')).resolves.toBe(
      path.join(repo, 'frontend', 'build'),
    );
    expect(commands).toEqual([]);
  });
});
