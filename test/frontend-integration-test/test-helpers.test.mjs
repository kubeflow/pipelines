// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import assert from 'node:assert/strict';
import { afterEach, beforeEach, test } from 'node:test';
import { waitForPipelineListReady, waitForSelectorDisplayed } from './test-helpers.js';

let originalBrowser;
let originalSelector;

beforeEach(() => {
  originalBrowser = globalThis.browser;
  originalSelector = globalThis.$;
  globalThis.browser = {
    async waitUntil(condition, { timeoutMsg }) {
      for (let attempt = 0; attempt < 3; attempt++) {
        if (await condition()) return;
      }
      throw new Error(timeoutMsg || 'condition timed out');
    },
  };
});

afterEach(() => {
  if (originalBrowser === undefined) delete globalThis.browser;
  else globalThis.browser = originalBrowser;
  if (originalSelector === undefined) delete globalThis.$;
  else globalThis.$ = originalSelector;
});

function element(displayed, exists = true) {
  return {
    isExisting: async () => exists,
    isDisplayed: async () => displayed,
    // Like WebdriverIO, the wait stays bound to this particular element.
    async waitForDisplayed(options) {
      await globalThis.browser.waitUntil(() => this.isDisplayed(), options);
    },
  };
}

test('selector visibility wait survives a hidden element being replaced during navigation', async () => {
  const outgoingFilter = element(false);
  const incomingFilter = element(true);
  let navigated = false;
  globalThis.$ = () => {
    const current = navigated ? incomingFilter : outgoingFilter;
    navigated = true;
    return current;
  };

  await waitForSelectorDisplayed('#tableFilterBox', { timeout: 5000 });
});

test('selector visibility wait allows the destination element to mount later', async () => {
  const mountingStates = [element(false, false), element(true)];
  globalThis.$ = () => mountingStates.shift() || element(true);

  await waitForSelectorDisplayed('#tableFilterBox', { timeout: 5000 });
});

test('selector visibility wait still fails if the destination never becomes visible', async () => {
  globalThis.$ = () => element(false);

  await assert.rejects(
    waitForSelectorDisplayed('#tableFilterBox', { timeout: 5000 }),
    /expected selector #tableFilterBox to be displayed/,
  );
});

function pipelinePage(label, visible, hash = '#/pipelines') {
  return {
    location: { hash },
    document: {
      getElementById(id) {
        assert.equal(id, 'tableFilterBox');
        return label === null
          ? null
          : {
              labels: [{ textContent: label }],
              checkVisibility: () => visible,
            };
      },
    },
  };
}

async function executeInPage(fn, page) {
  const { runInNewContext } = await import('node:vm');
  return runInNewContext(`(${fn.toString()})()`, page);
}

function usePages(pages) {
  let reads = 0;
  browser.execute = async (fn) => executeInPage(fn, pages[Math.min(reads++, pages.length - 1)]);
  browser.getUrl = async () => `http://localhost/${pages[0].location.hash}`;
  // Model the old WebdriverIO wait: capture a node once, then poll that node.
  globalThis.$ = () => {
    const input = pages[0].document.getElementById('tableFilterBox');
    return element(input?.checkVisibility() || false, !!input);
  };
  return () => reads;
}

test('pipeline readiness rejects outgoing visible filter then observes the replacement', async () => {
  const pages = [
    pipelinePage('Filter experiments', true),
    pipelinePage(null, false),
    pipelinePage('Filter pipelines', true),
  ];
  const reads = usePages(pages);
  await waitForPipelineListReady({ timeout: 5000 });
  assert.equal(reads(), 3);
});

test('pipeline readiness accepts query parameters on the exact list route', async () => {
  usePages([pipelinePage('Filter pipelines', true, '#/pipelines?namespace=team')]);
  await waitForPipelineListReady();
});

for (const [name, page] of [
  ['outgoing page', pipelinePage('Filter experiments', true)],
  ['hidden destination', pipelinePage('Filter pipelines', false)],
  ['pipeline details route', pipelinePage('Filter pipelines', true, '#/pipelines/details/id')],
]) {
  test(`pipeline readiness fails for ${name}`, async () => {
    usePages([page]);
    await assert.rejects(
      waitForPipelineListReady({ timeout: 5000 }),
      /Pipelines list filter|condition timed out/,
    );
  });
}
