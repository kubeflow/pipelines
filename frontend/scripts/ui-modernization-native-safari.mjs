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
  const tips = () =>
    command('POST', `${path}/elements`, { using: '-ios predicate string', value: tipPredicate });
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    const visibleTips = await tips();
    if (visibleTips.length) {
      await snapshot(await command('GET', `${path}/source`), 'safari-tip');
      // Use the nearest ancestor containing the tip's Close/Dismiss control.
      // Never click a generic browser Close button without this identified tip.
      const buttons = await command('POST', `${path}/elements`, {
        using: 'xpath',
        value: `//*[@visible="true" and contains(@label,"View Bookmarks") and contains(@label,"Open Tabs")]/ancestor-or-self::*[.//${closeButton}][1]//${closeButton}`,
      });
      assert.equal(buttons.length, 1, 'Safari onboarding tip needs one identifiable close control');
      await command('POST', `${path}/element/${buttons[0][elementKey]}/click`, {});
      assert.equal(
        (await tips()).length,
        0,
        'Safari onboarding tip remained after its native dismissal',
      );
      entry.actions.push('dismissed Safari bookmarks/share/tabs tip through native accessibility');
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
