const fs = require('fs');
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
