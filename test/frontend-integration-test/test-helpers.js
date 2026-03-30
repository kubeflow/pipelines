// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const assert = require('assert');
const path = require('path');
const URL = require('url').URL;

const defaultTimeout = 10000;
const screenshotDir = process.env.FRONTEND_INTEGRATION_SCREENSHOT_DIR || '/tmp';
const runPageLoadingText = 'Currently loading pipeline information';
const runDetailsLoadingText = 'Currently loading run information';

const legacyRunFormSelectors = {
  description: '#descriptionInput',
  message: 'input#newRunPipelineParam0',
  runName: '#runNameInput',
};

const v2RunFormSelectors = {
  description: '//label[normalize-space()="Description"]/following::*[self::textarea or self::input][1]',
  message: '#message',
  runName: '//label[starts-with(normalize-space(), "Run name")]/following::input[1]',
};

const defaultRunFormVariants = [
  { name: 'legacy', selectors: legacyRunFormSelectors },
  { name: 'v2', selectors: v2RunFormSelectors },
];

async function waitForCondition(condition, { timeout = defaultTimeout, timeoutMsg, interval } = {}) {
  const waitOptions = { timeout };
  if (timeoutMsg) {
    waitOptions.timeoutMsg = timeoutMsg;
  }
  if (interval) {
    waitOptions.interval = interval;
  }
  await browser.waitUntil(condition, waitOptions);
}

async function waitForHashPrefix(prefix, { timeout = defaultTimeout } = {}) {
  await waitForCondition(
    async () => new URL(await browser.getUrl()).hash.startsWith(prefix),
    {
      timeout,
      timeoutMsg: `expected URL hash to start with ${prefix}`,
    },
  );
}

async function pageContainsText(text) {
  return browser.execute((pageTextToFind) => {
    const pageText = document.body ? document.body.innerText : '';
    return pageText.includes(pageTextToFind);
  }, text);
}

async function isRunPageLoading() {
  return pageContainsText(runPageLoadingText);
}

async function isRunDetailsPageLoading() {
  return pageContainsText(runDetailsLoadingText);
}

async function waitForRunPageReady({
  timeout = defaultTimeout,
  requirePipelineVersion = true,
  timeoutMsg = 'expected a run creation form to load',
  variants = defaultRunFormVariants,
} = {}) {
  let matchedVariant;

  try {
    await waitForCondition(
      async () => {
        if (requirePipelineVersion) {
          const hash = new URL(await browser.getUrl()).hash;
          const query = hash.split('?')[1] || '';
          if (!new URLSearchParams(query).get('pipelineVersionId')) {
            return false;
          }
        }

        if (await isRunPageLoading()) {
          return false;
        }

        for (const variant of variants) {
          if (
            (await isSelectorDisplayed(variant.selectors.runName)) &&
            (await isSelectorDisplayed(variant.selectors.description))
          ) {
            matchedVariant = variant;
            return true;
          }
        }

        return false;
      },
      {
        timeout,
        timeoutMsg,
      },
    );
  } catch (error) {
    console.log('RUN_PAGE_LOADING', await isRunPageLoading());
    console.log('RUN_PAGE_URL', await browser.getUrl());
    await saveDebugScreenshot('run-page-ready');
    throw error;
  }

  return matchedVariant;
}

async function waitForRunDetailsPageReady({
  timeout = defaultTimeout,
  timeoutMsg = 'expected run details page to load',
} = {}) {
  try {
    await waitForHashPrefix('#/runs/details/', { timeout });
    await waitForCondition(
      async () => {
        if (await isRunDetailsPageLoading()) {
          return false;
        }

        return (
          (await isSelectorDisplayed('[data-testid="page-title"]')) &&
          (await isSelectorDisplayed('button=Graph')) &&
          (await isSelectorDisplayed('button=Config'))
        );
      },
      {
        timeout,
        timeoutMsg,
      },
    );
  } catch (error) {
    console.log('RUN_DETAILS_LOADING', await isRunDetailsPageLoading());
    console.log('RUN_DETAILS_URL', await browser.getUrl());
    await saveDebugScreenshot('run-details-ready');
    throw error;
  }
}

async function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key.
  const rowText = await $(`span=${key}`).$('..').getText();
  return rowText.substr(`${key}\n`.length);
}

async function clearDefaultInput() {
  await browser.keys(['Control', 'a']);
  await browser.keys('Backspace');
}

function buildTableRowSelector(rowLabel, { containerXPath = '' } = {}) {
  const rowSelector =
    '//*[@data-testid="table-row"][.//*[self::a or self::span][normalize-space()="' +
    rowLabel +
    '"]]';
  return containerXPath ? `${containerXPath}${rowSelector}` : rowSelector;
}

async function saveDebugScreenshot(name) {
  const screenshotPath = path.join(screenshotDir, `kfp-${name}-${Date.now()}.png`);
  await browser.saveScreenshot(screenshotPath);
  console.log('DEBUG_SCREENSHOT', screenshotPath);
  return screenshotPath;
}

async function isSelectorDisplayed(selector) {
  const element = await $(selector);
  return (await element.isExisting()) && (await element.isDisplayed());
}

async function waitForGraphNodeCount(expectedCount, { timeout = defaultTimeout } = {}) {
  await waitForCondition(
    async () => (await $$('.graphNode')).length === expectedCount,
    {
      timeout,
      timeoutMsg: `expected ${expectedCount} graph node(s) to be visible`,
    },
  );
  const nodes = await $$('.graphNode');
  assert(
    nodes.length === expectedCount,
    `should have a ${expectedCount}-node graph, instead has: ${nodes.length}`,
  );
}

async function waitForTableRows(
  selector = '[data-testid="table-row"]',
  { timeout = defaultTimeout, timeoutMsg } = {},
) {
  await waitForCondition(
    async () => (await $$(selector)).length > 0,
    {
      timeout,
      timeoutMsg: timeoutMsg || `expected at least one table row for selector ${selector}`,
    },
  );
  return await $$(selector);
}

function annotatePhaseError(phaseName, error) {
  if (error instanceof Error) {
    error.message = `[${phaseName}] ${error.message}`;
    return error;
  }
  return new Error(`[${phaseName}] ${String(error)}`);
}

function phaseNameToScreenshotName(phaseName) {
  return phaseName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

async function runPhase(phaseName, fn, { screenshotName } = {}) {
  console.log('TEST_PHASE', phaseName);
  try {
    return await fn();
  } catch (error) {
    try {
      await saveDebugScreenshot(screenshotName || phaseNameToScreenshotName(phaseName));
    } catch (screenshotError) {
      const screenshotMessage =
        screenshotError instanceof Error ? screenshotError.message : String(screenshotError);
      console.log('PHASE_SCREENSHOT_FAILED', phaseName, screenshotMessage);
    }
    throw annotatePhaseError(phaseName, error);
  }
}

module.exports = {
  buildTableRowSelector,
  clearDefaultInput,
  getValueFromDetailsTable,
  isSelectorDisplayed,
  runPhase,
  saveDebugScreenshot,
  waitForCondition,
  waitForGraphNodeCount,
  waitForHashPrefix,
  waitForRunDetailsPageReady,
  waitForRunPageReady,
  waitForTableRows,
};
