// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import assert from 'node:assert/strict';
import { afterEach, beforeEach, test } from 'node:test';
import { openNewExperiment, selectPipelineCard, waitForLogViewerText, waitForSelectorDisplayed } from './test-helpers.js';

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

test('pipeline card selection waits for an enabled named checkbox after filtering', async () => {
  const pipelineName = 'uploaded-pipeline';
  const filterSelector = 'input[type="search"][placeholder="Search pipelines"]';
  const checkboxSelector = `[role="checkbox"][aria-label="Select pipeline ${pipelineName}"]`;
  let filterValue = '';
  let readinessChecks = 0;
  let selected = false;
  let clicks = 0;
  const filter = {
    ...element(true),
    async setValue(value) {
      filterValue = value;
    },
  };
  const checkbox = {
    ...element(true),
    // WebDriver considers Base UI's span enabled; aria-disabled owns its busy state.
    isEnabled: async () => true,
    async getAttribute(name) {
      if (name === 'aria-disabled') {
        assert.equal(filterValue, pipelineName);
        return String(++readinessChecks === 1);
      }
      assert.equal(name, 'aria-checked');
      return String(selected);
    },
    async click() {
      assert.ok(readinessChecks > 1, 'must wait until filtering finishes before selecting');
      selected = true;
      clicks++;
    },
  };
  globalThis.$ = (selector) => {
    if (selector === filterSelector) return filter;
    assert.equal(selector, checkboxSelector);
    return checkbox;
  };

  await selectPipelineCard(pipelineName, { timeout: 5000 });
  assert.equal(selected, true);
  assert.equal(clicks, 1);

  await selectPipelineCard(pipelineName, { timeout: 5000 });
  assert.equal(selected, true, 'an already selected pipeline must not be toggled off');
  assert.equal(clicks, 1);
});

test('pipeline card selection fails when clicking does not select the pipeline', async () => {
  globalThis.$ = (selector) =>
    selector.startsWith('input')
      ? { ...element(true), setValue: async () => {} }
      : {
          ...element(true),
          isEnabled: async () => true,
          getAttribute: async (name) => (name === 'aria-disabled' ? null : 'false'),
          click: async () => {},
        };

  await assert.rejects(
    selectPipelineCard('uploaded-pipeline', { timeout: 5000 }),
    /expected pipeline uploaded-pipeline to be selected/,
  );
});

test('experiment creation navigates to its owning list before using the toolbar', async () => {
  let hash = '#/pipelines/details/uploaded';
  const clicks = [];
  globalThis.browser.getUrl = async () => `http://localhost/${hash}`;
  globalThis.$ = (selector) => ({
    async waitForDisplayed() {
      assert.equal(hash, '#/experiments', 'the creation action only exists on the list');
    },
    async click() {
      clicks.push(selector);
      if (selector === 'a[aria-label="Experiments"]') {
        hash = '#/experiments';
      } else {
        assert.equal(selector, '#newExperimentBtn');
        assert.equal(hash, '#/experiments');
        hash = '#/experiments/new';
      }
    },
  });
  await openNewExperiment({ timeout: 5000 });
  assert.deepEqual(clicks, ['a[aria-label="Experiments"]', '#newExperimentBtn']);
  assert.equal(hash, '#/experiments/new');
});


test('log wait reads component output instead of driver logs and waits for expected text', async () => {
  let reads = 0;
  globalThis.browser.execute = async () => false;
  globalThis.browser.getUrl = async () => "http://localhost/#/runs/details/test";
  globalThis.browser.saveScreenshot = async () => {};
  globalThis.$ = (selector) => {
    if (selector === '.kfp-log-viewer[aria-label="Component logs"]') {
      return {
        ...element(true),
        getText: async () => ++reads === 1 ? 'still running' : 'Hello world from node: A',
      };
    }
    return element(false, false);
  };
  await waitForLogViewerText('Hello world from node: A', { timeout: 5000 });
  assert.equal(reads, 2, 'visible logs without the expected output must not pass');
});
