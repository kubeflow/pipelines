const { execFileSync } = require('child_process');
const { realpathSync } = require('fs');

function verifyCachedCheckout(targetDir, branchRef) {
  const options = { cwd: targetDir, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] };
  try {
    const root = execFileSync('git', ['rev-parse', '--show-toplevel'], options).trim();
    if (realpathSync(root) !== realpathSync(targetDir)) {
      throw new Error('Cache directory is not a checkout root');
    }
    const head = execFileSync('git', ['rev-parse', '--verify', 'HEAD'], options).trim();
    const expected = execFileSync(
      'git',
      ['rev-parse', '--verify', '--end-of-options', `${branchRef}^{commit}`],
      options,
    ).trim();
    if (head === expected) return;
  } catch {
    // Missing refs and invalid cached clones need the same recovery action.
  }
  throw new Error(
    `Cached checkout at ${targetDir} does not match ${branchRef}. ` +
      'Move this cache directory aside and rerun to clone the requested branch.',
  );
}

function createBaseWorktree(repoRoot, targetDir, baseRef) {
  execFileSync('git', ['worktree', 'add', '--detach', targetDir, baseRef], {
    cwd: repoRoot,
    stdio: ['ignore', 'pipe', 'pipe'],
  });
}

module.exports = { verifyCachedCheckout, createBaseWorktree };
