const { execFileSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { COMPONENTS } = require('./detect-changes.js');

const root = path.resolve(__dirname, '../../..');

describe('UI smoke backend build inventory', () => {
  it('references only existing backend sources and image targets', () => {
    const makefile = fs.readFileSync(path.join(root, 'backend/Makefile'), 'utf8');
    for (const component of COMPONENTS) {
      for (const source of component.paths) {
        expect(fs.existsSync(path.join(root, source)), source).toBe(true);
      }
      expect(makefile).toMatch(new RegExp(`^${component.makeTarget}:`, 'm'));
    }
  });

  it('retains the native v2 runtime and viewer components', () => {
    expect(COMPONENTS.map((component) => component.name)).toEqual(
      expect.arrayContaining([
        'apiserver',
        'persistence-agent',
        'scheduledworkflow',
        'viewercontroller',
        'driver',
        'launcher',
      ]),
    );
  });
});

describe('UI smoke change detection CLI', () => {
  let repository;

  beforeEach(() => {
    repository = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-base-'));
    const git = (...args) => execFileSync('git', args, { cwd: repository, stdio: 'pipe' });
    git('init', '--initial-branch=main');
    git(
      '-c',
      'user.name=Test Fixture',
      '-c',
      'user.email=fixture@example.invalid',
      'commit',
      '--allow-empty',
      '-m',
      'Base fixture',
    );
    git('checkout', '-b', 'feature');
    fs.mkdirSync(path.join(repository, 'frontend/src'), { recursive: true });
    fs.writeFileSync(path.join(repository, 'frontend/src/change.ts'), 'export {};\n');
    git('add', '.');
    git(
      '-c',
      'user.name=Test Fixture',
      '-c',
      'user.email=fixture@example.invalid',
      'commit',
      '-m',
      'Frontend fixture',
    );
  });

  afterEach(() => {
    fs.rmSync(repository, { recursive: true, force: true });
  });

  it('compares against main by default without a master branch', () => {
    const output = execFileSync(process.execPath, [path.join(__dirname, 'detect-changes.js')], {
      cwd: repository,
      encoding: 'utf8',
    });
    expect(output).toContain('Base ref: main');
    expect(output).toContain('Changed files: 1');
    expect(output).toContain('Frontend changed: true');
    expect(output).toContain('Backend changed: false');
  });

  it('honors an explicit base override', () => {
    const output = execFileSync(
      process.execPath,
      [path.join(__dirname, 'detect-changes.js'), '--base', 'HEAD'],
      {
        cwd: repository,
        encoding: 'utf8',
      },
    );
    expect(output).toContain('Base ref: HEAD');
    expect(output).toContain('Changed files: 0');
    expect(output).toContain('Frontend changed: false');
  });
});
