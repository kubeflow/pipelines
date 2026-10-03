const { PAGES, resolvePathTemplate } = require('./capture-screenshots.js');

describe('UI smoke screenshot routes', () => {
  it('loads pipeline graphs from the uploaded v2 pipeline and version fixtures', () => {
    const captures = PAGES.filter(({ name }) => name.startsWith('pipeline-details-seeded'));
    expect(captures).toHaveLength(2);
    for (const capture of captures) {
      expect(resolvePathTemplate(capture.path, { pipelineId: 'pipeline-1' })).toEqual({
        missing: [],
        resolvedPath: '/#/pipelines/details/pipeline-1',
      });
      expect(capture.actions).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ type: 'waitForFunction', predicate: expect.any(Function) }),
        ]),
      );
    }
    expect(captures.find(({ name }) => name.endsWith('sidepanel')).actions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ type: 'click', selector: expect.any(String) }),
      ]),
    );
  });

  it('captures native run detail graphs using the seeded run ID', () => {
    const captures = PAGES.filter(({ name }) => name.startsWith('run-details-seeded'));
    expect(captures).toHaveLength(2);
    for (const capture of captures) {
      expect(resolvePathTemplate(capture.path, { runId: 'run-1' }).resolvedPath).toContain('run-1');
      expect(capture.actions).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ type: 'waitForFunction', predicate: expect.any(Function) }),
        ]),
      );
    }
  });
});
