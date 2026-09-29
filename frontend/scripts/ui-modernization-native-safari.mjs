/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

import assert from 'node:assert/strict';

const elementKey = 'element-6066-11e4-a52e-4f735466cecf';
const tipPredicate =
  'visible == 1 AND label CONTAINS "View Bookmarks" AND label CONTAINS "Open Tabs"';
const closeButton =
  'XCUIElementTypeButton[@visible="true" and (@name="Close" or @label="Close" or @name="Dismiss" or @label="Dismiss")]';

const startPageCloseButton =
  'XCUIElementTypeButton[@name="close" and @label="close" and @visible="true" and @enabled="true"]';
export const safariStartPageCloseSelector =
  '//XCUIElementTypeOther[@visible="true" and not(ancestor::XCUIElementTypeWebView)]' +
  '[./XCUIElementTypeOther/XCUIElementTypeButton[@name="onboardingButton-CustomizeStartPage" and @visible="true"]]' +
  `/${startPageCloseButton}`;
const onboardingTips = [
  {
    name: 'bookmarks/share/tabs',
    evidenceLabel: 'safari-bookmarks-tip',
    predicate: tipPredicate,
    closeSelector: `//*[@visible="true" and contains(@label,"View Bookmarks") and contains(@label,"Open Tabs")]/ancestor-or-self::*[.//${closeButton}][1]//${closeButton}`,
  },
  {
    name: 'Start Page',
    evidenceLabel: 'safari-start-page-tip',
    predicate:
      'visible == 1 AND name == "onboardingButton-CustomizeStartPage" AND label == "Customize Start Page"',
    closeSelector: safariStartPageCloseSelector,
  },
];

export const safariKeyboardDoneSelector =
  '//XCUIElementTypeToolbar[@visible="true" and not(ancestor::XCUIElementTypeWebView)]' +
  '[.//XCUIElementTypeButton[@name="Previous"] and .//XCUIElementTypeButton[@name="Next"]]' +
  '//XCUIElementTypeButton[@name="Done" and @visible="true" and @enabled="true"]';

// Safari's first-launch tip and keyboard are outside the web context. They can
// intercept Appium's native calibration taps even when the page DOM is ready.
export async function prepareNativeSafariTap(command, session, evidence, snapshot) {
  const path = `/session/${session}`;
  const context = await command('GET', `${path}/context`);
  assert.match(context, /^WEBVIEW_/, 'Native Safari preparation requires a selected web context');
  let failure;
  const entry = { startedAt: new Date().toISOString(), actions: [] };
  evidence.push(entry);
  const mobile = (script, args = {}) =>
    command('POST', `${path}/execute/sync`, { script: `mobile: ${script}`, args: [args] });
  const tips = (predicate) =>
    command('POST', `${path}/elements`, { using: '-ios predicate string', value: predicate });
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    for (const tip of onboardingTips) {
      if (!(await tips(tip.predicate)).length) continue;
      await snapshot(await command('GET', `${path}/source`), tip.evidenceLabel);
      // Use only the control scoped to the identified browser-owned tip.
      const buttons = await command('POST', `${path}/elements`, {
        using: 'xpath',
        value: tip.closeSelector,
      });
      assert.equal(buttons.length, 1, 'Safari onboarding tip needs one identifiable close control');
      await command('POST', `${path}/element/${buttons[0][elementKey]}/click`, {});
      assert.equal(
        (await tips(tip.predicate)).length,
        0,
        'Safari onboarding tip remained after its native dismissal',
      );
      entry.actions.push(`dismissed Safari ${tip.name} tip through native accessibility`);
    }
    if (await mobile('isKeyboardShown')) {
      // iPhone Safari exposes Done in its form accessory toolbar, outside the
      // keyboard subtree searched by WDA's generic keyboard dismissal.
      const doneButtons = await command('POST', `${path}/elements`, {
        using: 'xpath',
        value: safariKeyboardDoneSelector,
      });
      assert.ok(doneButtons.length <= 1, 'Safari form toolbar has ambiguous Done controls');
      if (doneButtons.length) {
        await command('POST', `${path}/element/${doneButtons[0][elementKey]}/click`, {});
        entry.actions.push('used Safari native form-toolbar Done');
      } else {
        await mobile('hideKeyboard', { keys: ['Done', 'Hide keyboard'] });
      }
      assert.equal(
        await mobile('isKeyboardShown'),
        false,
        'Safari keyboard remained after native dismissal',
      );
      entry.actions.push('dismissed keyboard through native WebDriver');
    }
    entry.status = 'passed';
  } catch (error) {
    failure = error;
    entry.status = 'failed';
    entry.error = String(error);
    try {
      await snapshot(await command('GET', `${path}/source`), 'safari-preparation-failed');
    } catch (diagnosticError) {
      entry.diagnosticError = String(diagnosticError);
    }
    throw error;
  } finally {
    try {
      await command('POST', `${path}/context`, { name: context });
    } catch (error) {
      entry.status = 'failed';
      entry.restoreError = String(error);
      if (!failure) throw error;
    }
  }
}

export function nativeSafariLinkSelector(name) {
  assert.equal(typeof name, 'string');
  assert.ok(name.length, 'Native link needs its exact accessible name');
  const literal = name.includes('"')
    ? `concat(${name
        .split('"')
        .map((part) => `"${part}"`)
        .join(`, '"', `)})`
    : `"${name}"`;
  return `//XCUIElementTypeWebView//XCUIElementTypeLink[@name=${literal} and @visible="true" and @enabled="true"]`;
}

// Appium 12.13.3 can map an input-zoomed web link to the wrong native coordinates.
// Selecting its native Link type also excludes the identically named child StaticText.
export async function clickNativeSafariLink(command, session, name, evidence) {
  const path = `/session/${session}`;
  const context = await command('GET', `${path}/context`);
  assert.match(context, /^WEBVIEW_/, 'Native link click requires a selected web context');
  const entry = { startedAt: new Date().toISOString(), name, method: 'native Link element click' };
  evidence.push(entry);
  let failure;
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    const links = await command('POST', `${path}/elements`, {
      using: 'xpath',
      value: nativeSafariLinkSelector(name),
    });
    assert.equal(links.length, 1, 'Native Safari needs exactly one visible matching Link');
    await command('POST', `${path}/element/${links[0][elementKey]}/click`, {});
    entry.status = 'passed';
  } catch (error) {
    failure = error;
    entry.status = 'failed';
    entry.error = String(error);
    throw error;
  } finally {
    try {
      await command('POST', `${path}/context`, { name: context });
    } catch (error) {
      entry.status = 'failed';
      entry.restoreError = String(error);
      if (!failure) throw error;
    }
  }
}
