// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import assert from 'node:assert/strict';
import { afterEach, beforeEach, test } from 'node:test';
import { waitForSelectorDisplayed } from './test-helpers.js';

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
