const fs = require('fs');
const path = require('path');
const { COMPONENTS } = require('./detect-changes.js');
const { componentsForRevision, revisionUsesMetadataService } = require('./smoke-test-runner.js');

const root = path.resolve(__dirname, '../../..');

describe('UI smoke backend build inventory', () => {
  it('selects only image sources deployed by the current native revision', () => {
    const components = componentsForRevision(root);
    for (const component of components) {
      expect(fs.existsSync(path.join(root, component.dockerfile)), component.name).toBe(true);
    }
    for (const removed of ['cache-server', 'metadata-writer', 'metadata-envoy', 'visualization']) {
      expect(components.map(({ name }) => name)).not.toContain(removed);
    }
    expect(revisionUsesMetadataService(root)).toBe(false);
  });

  it('retains native v2 runtime and viewer components plus historical build descriptors', () => {
    expect(componentsForRevision(root).map(({ name }) => name)).toEqual(
      expect.arrayContaining([
        'apiserver',
        'persistence-agent',
        'scheduledworkflow',
        'viewercontroller',
        'driver',
        'launcher',
      ]),
    );
    expect(COMPONENTS.map(({ name }) => name)).toEqual(
      expect.arrayContaining([
        'cache-server',
        'metadata-writer',
        'metadata-envoy',
        'visualization',
      ]),
    );
  });
});
