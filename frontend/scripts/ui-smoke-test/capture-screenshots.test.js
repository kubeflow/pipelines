const { PAGES, resolvePathTemplate } = require('./capture-screenshots.js');

describe('UI smoke screenshot routes', () => {
  it('loads pipeline detail graph captures from the seeded run spec', () => {
    const pipelineCaptures = PAGES.filter(({ name }) => name.startsWith('pipeline-details-seeded'));

    expect(pipelineCaptures).toHaveLength(2);
    for (const capture of pipelineCaptures) {
      expect(resolvePathTemplate(capture.path, { runId: 'run-1' })).toEqual({
        missing: [],
        resolvedPath: '/#/pipelines/details/?fromRun=run-1',
      });
    }
    const sidePanelCapture = pipelineCaptures.find(
      ({ name }) => name === 'pipeline-details-seeded-sidepanel',
    );
    expect(sidePanelCapture.actions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          selector: '.react-flow__node:visible, .graphNode:visible',
          type: 'click',
        }),
      ]),
    );
  });

  it('accepts the seeded run graph after its single visible task renders', () => {
    const runCaptures = PAGES.filter(({ name }) => name.startsWith('run-details-seeded'));
    const document = {
      querySelectorAll: (selector) =>
        selector === '.react-flow__node' ? [{ visibility: 'visible' }] : [],
    };
    const getComputedStyle = (node) => node;

    expect(runCaptures).toHaveLength(2);
    for (const capture of runCaptures) {
      const wait = capture.actions.find(({ type }) => type === 'waitForFunction');
      const predicate = new Function(
        'document',
        'getComputedStyle',
        `return (${wait.expression})();`,
      );
      expect(predicate(document, getComputedStyle)).toBe(true);
    }
  });
});
