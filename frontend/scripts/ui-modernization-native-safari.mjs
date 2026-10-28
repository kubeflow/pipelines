/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

import assert from 'node:assert/strict';
import { setTimeout as delay } from 'node:timers/promises';

// A cold Appium alert probe can consume most of the 60s startup command bound.
// Allow another origin observation without extending ordinary UI readiness waits.
export async function waitForNativeSafariOrigin(
  execute,
  origin,
  { now = Date.now, sleep = delay } = {},
) {
  const deadline = now() + 120000;
  while (now() < deadline) {
    const url = await execute(
      (expectedOrigin) => (location.origin === expectedOrigin ? location.href : null),
      origin,
    );
    if (url) return url;
    if (now() < deadline) await sleep(100);
  }
  throw new Error('Timed out: initial Safari fixture origin');
}

const elementKey = 'element-6066-11e4-a52e-4f735466cecf';
const tipPredicate =
  'visible == 1 AND label CONTAINS "View Bookmarks" AND label CONTAINS "Open Tabs"';
const closeButton =
  'XCUIElementTypeButton[@visible="true" and not(ancestor::XCUIElementTypeWebView) and (@name="Close" or @label="Close" or @name="Dismiss" or @label="Dismiss")]';

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
    closeSelector: `//*[@visible="true" and not(ancestor::XCUIElementTypeWebView) and contains(@label,"View Bookmarks") and contains(@label,"Open Tabs")]/ancestor-or-self::*[.//${closeButton}][1]//${closeButton}`,
  },
  {
    name: 'Start Page',
    evidenceLabel: 'safari-start-page-tip',
    predicate:
      'visible == 1 AND name == "onboardingButton-CustomizeStartPage" AND label == "Customize Start Page"',
    closeSelector: safariStartPageCloseSelector,
  },
];

// One native query handles the common case where neither onboarding tip exists.
// Positive matches still use each tip's exact lookup and scoped dismissal below.
export const safariOnboardingTipPredicate = onboardingTips
  .map(({ predicate }) => `(${predicate})`)
  .join(' OR ');

export const safariKeyboardDoneSelector =
  '//XCUIElementTypeToolbar[@visible="true" and not(ancestor::XCUIElementTypeWebView)]' +
  '[.//XCUIElementTypeButton[@name="Previous"] and .//XCUIElementTypeButton[@name="Next"]]' +
  '//XCUIElementTypeButton[@name="Done" and @visible="true" and @enabled="true"]';

// iOS 27 can report the visible form-toolbar checkmark as visible=false.
// The fallback still requires its exact native identity and a positive rectangle
// fully inside the visible, browser-owned Previous/Next form toolbar.
export const safariKeyboardDoneGeometrySelector =
  '//XCUIElementTypeToolbar[@visible="true" and not(ancestor::XCUIElementTypeWebView)]' +
  '[.//XCUIElementTypeButton[@name="Previous"] and .//XCUIElementTypeButton[@name="Next"]]' +
  '//XCUIElementTypeButton[@name="Done" and @label="Done" and @enabled="true" and @accessible="true"]' +
  '[number(@width)>0 and number(@height)>0]' +
  '[number(@x)>=number(ancestor::XCUIElementTypeToolbar[1]/@x)]' +
  '[number(@y)>=number(ancestor::XCUIElementTypeToolbar[1]/@y)]' +
  '[number(@x)+number(@width)<=number(ancestor::XCUIElementTypeToolbar[1]/@x)+number(ancestor::XCUIElementTypeToolbar[1]/@width)]' +
  '[number(@y)+number(@height)<=number(ancestor::XCUIElementTypeToolbar[1]/@y)+number(ancestor::XCUIElementTypeToolbar[1]/@height)]';

// Captured iPadOS 26 keyboard: generic WDA dismissal fails even though this
// native button is visible. Match its exact identity outside the web content.
export const safariKeyboardHideSelector =
  '//XCUIElementTypeKeyboard[@visible="true" and not(ancestor::XCUIElementTypeWebView)]' +
  '//XCUIElementTypeButton[@name="Hide keyboard" and @label="Hide keyboard" and @visible="true" and @enabled="true"]';

export const safariActiveAddressSelector =
  '//XCUIElementTypeTextField[@label="Address" and starts-with(@name,"SearchFieldItemView?")]' +
  '[contains(concat(@name,"&"),"?isActive=true&") or contains(concat(@name,"&"),"&isActive=true&")]' +
  '[@visible="true" and @enabled="true" and not(ancestor::XCUIElementTypeWebView)]';

// Safari's first-launch tip and keyboard are outside the web context. They can
// intercept Appium's native calibration taps even when the page DOM is ready.
export async function prepareNativeSafariTap(command, session, evidence, snapshot, fixtureOrigin) {
  const path = `/session/${session}`;
  const context = await command('GET', `${path}/context`);
  assert.match(context, /^WEBVIEW_/, 'Native Safari preparation requires a selected web context');
  const currentUrl = await command('GET', `${path}/url`);
  let failure;
  const entry = { startedAt: new Date().toISOString(), actions: [] };
  evidence.push(entry);
  const mobile = (script, args = {}) =>
    command('POST', `${path}/execute/sync`, { script: `mobile: ${script}`, args: [args] });
  const tips = (predicate) =>
    command('POST', `${path}/elements`, { using: '-ios predicate string', value: predicate });
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    // Complete browser address editing first: on iPad its active editor can keep
    // the keyboard open even when a native Hide keyboard click reports success.
    const addresses = await command('POST', `${path}/elements`, {
      using: 'xpath',
      value: safariActiveAddressSelector,
    });
    assert.ok(addresses.length <= 1, 'Safari has ambiguous active browser Address fields');
    if (addresses.length) {
      const url = new URL(currentUrl);
      assert.ok(
        ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname),
        'Safari address completion requires a loopback fixture URL',
      );
      assert.equal(url.protocol, 'http:', 'Safari address completion requires an HTTP fixture');
      assert.equal(
        url.origin,
        fixtureOrigin,
        'Safari address completion must preserve the configured fixture origin',
      );
      assert.ok(!url.username && !url.password, 'Safari fixture URL must not contain credentials');
      entry.addressCompletion = {
        url: currentUrl,
        method: 'native Address clear/value and Return',
      };
      await snapshot(await command('GET', `${path}/source`), 'safari-address-edit');
      const addressPath = `${path}/element/${addresses[0][elementKey]}`;
      await command('POST', `${addressPath}/clear`, {});
      // XCUITest 12.13.3 native setValue passes newline through to WDA Return.
      await command('POST', `${addressPath}/value`, { text: `${currentUrl}\n` });
      let active = addresses;
      const deadline = Date.now() + 5000;
      do {
        active = await command('POST', `${path}/elements`, {
          using: 'xpath',
          value: safariActiveAddressSelector,
        });
        if (!active.length) break;
        if (Date.now() >= deadline) break;
        await delay(250);
      } while (Date.now() < deadline);
      assert.equal(active.length, 0, 'Safari address editor remained active after native Return');
      await snapshot(await command('GET', `${path}/source`), 'safari-address-completed');
      await command('POST', `${path}/context`, { name: context });
      const actualUrl = await command('GET', `${path}/url`);
      entry.addressCompletion.actualUrl = actualUrl;
      assert.equal(
        actualUrl,
        currentUrl,
        'Safari address completion changed the current fixture route',
      );
      entry.actions.push('completed active Safari Address through native typing and Return');
      await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    }
    if (await mobile('isKeyboardShown')) {
      // iPhone Safari exposes Done in its form accessory toolbar, outside the
      // keyboard subtree searched by WDA's generic keyboard dismissal.
      let doneButtons = await command('POST', `${path}/elements`, {
        using: 'xpath',
        value: safariKeyboardDoneSelector,
      });
      let doneSelection = 'visible';
      if (!doneButtons.length) {
        doneButtons = await command('POST', `${path}/elements`, {
          using: 'xpath',
          value: safariKeyboardDoneGeometrySelector,
        });
        doneSelection = 'toolbar-contained';
        if (doneButtons.length)
          await snapshot(await command('GET', `${path}/source`), 'safari-form-done-geometry');
      }
      assert.ok(doneButtons.length <= 1, 'Safari form toolbar has ambiguous Done controls');
      if (doneButtons.length) {
        entry.keyboardDone = { selection: doneSelection, method: 'native-element-click' };
        await command('POST', `${path}/element/${doneButtons[0][elementKey]}/click`, {});
        entry.actions.push('used Safari native form-toolbar Done');
      } else {
        const hideButtons = await command('POST', `${path}/elements`, {
          using: 'xpath',
          value: safariKeyboardHideSelector,
        });
        assert.ok(hideButtons.length <= 1, 'Safari keyboard has ambiguous Hide keyboard controls');
        if (hideButtons.length) {
          await snapshot(await command('GET', `${path}/source`), 'safari-keyboard-hide');
          entry.keyboardHide = { method: 'native-element-click' };
          await command('POST', `${path}/element/${hideButtons[0][elementKey]}/click`, {});
          entry.actions.push('used Safari native Hide keyboard');
        } else {
          await mobile('hideKeyboard', { keys: ['Done', 'Hide keyboard'] });
        }
      }
      assert.equal(
        await mobile('isKeyboardShown'),
        false,
        'Safari keyboard remained after native dismissal',
      );
      entry.actions.push('dismissed keyboard through native WebDriver');
    }
    const visibleTips = await tips(safariOnboardingTipPredicate);
    for (const tip of visibleTips.length ? onboardingTips : []) {
      if (!(await tips(tip.predicate)).length) continue;
      await snapshot(await command('GET', `${path}/source`), tip.evidenceLabel);
      const dismissal = { name: tip.name, timeoutMs: 5000, visibleCounts: [], attempts: [] };
      (entry.tipDismissals ||= []).push(dismissal);
      // A native click can succeed without dismissing Safari's own onboarding.
      // Retry once only after freshly identifying that same visible tip and its
      // unique close control. Application interactions never use this recovery.
      for (let attempt = 1; attempt <= 2; attempt++) {
        if (attempt === 2) {
          const remaining = await tips(tip.predicate);
          dismissal.visibleCounts.push(remaining.length);
          if (!remaining.length) break;
          assert.equal(remaining.length, 1, 'Safari onboarding tip became ambiguous before retry');
          await snapshot(await command('GET', `${path}/source`), `${tip.evidenceLabel}-retry`);
        }
        const buttons = await command('POST', `${path}/elements`, {
          using: 'xpath',
          value: tip.closeSelector,
        });
        assert.equal(
          buttons.length,
          1,
          'Safari onboarding tip needs one identifiable close control',
        );
        const observation = { attempt, visibleCounts: [] };
        dismissal.attempts.push(observation);
        await command('POST', `${path}/element/${buttons[0][elementKey]}/click`, {});
        const deadline = Date.now() + dismissal.timeoutMs;
        do {
          const count = (await tips(tip.predicate)).length;
          observation.visibleCounts.push(count);
          dismissal.visibleCounts.push(count);
          if (count === 0 || Date.now() >= deadline) break;
          await delay(250);
        } while (Date.now() < deadline);
        if (dismissal.visibleCounts.at(-1) === 0) break;
      }
      assert.equal(
        dismissal.visibleCounts.at(-1),
        0,
        'Safari onboarding tip remained after its native dismissal',
      );
      entry.actions.push(`dismissed Safari ${tip.name} tip through native accessibility`);
    }
    entry.status = 'passed';
  } catch (error) {
    failure = error;
    entry.status = 'failed';
    entry.error = String(error);
    try {
      // URL verification runs in web context; failure evidence must still be native XML.
      await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
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

function nativeNameLiteral(name, kind) {
  assert.equal(typeof name, 'string');
  assert.ok(name.length, `Native ${kind} needs its exact accessible name`);
  return name.includes('"')
    ? `concat(${name
        .split('"')
        .map((part) => `"${part}"`)
        .join(`, '"', `)})`
    : `"${name}"`;
}

export function nativeSafariLinkSelector(name) {
  return `//XCUIElementTypeWebView//XCUIElementTypeLink[@name=${nativeNameLiteral(name, 'link')} and @visible="true" and @enabled="true"]`;
}

export function nativeSafariRadioSelector(name) {
  // WDA 16.12.11 exposes wdTraits as the XPath traits attribute. The captured
  // radio is an Other/ToggleButton, distinct from its same-name StaticText.
  return `//XCUIElementTypeWebView//XCUIElementTypeOther[@name=${nativeNameLiteral(name, 'radio')} and @traits="ToggleButton" and @visible="true" and @enabled="true"]`;
}

// Appium 12.13.3 can map an input-zoomed web link to the wrong native coordinates.
// Selecting its native Link type also excludes the identically named child StaticText.
export async function clickNativeSafariLink(command, session, name, evidence) {
  return clickNativeSafariElement(
    command,
    session,
    name,
    evidence,
    'Link',
    nativeSafariLinkSelector(name),
  );
}

// Safari toolbar movement can invert Appium's calibration between samples.
// Activate the observed native radio itself rather than its separate text label.
export async function clickNativeSafariRadio(command, session, name, evidence) {
  return clickNativeSafariElement(
    command,
    session,
    name,
    evidence,
    'Radio',
    nativeSafariRadioSelector(name),
  );
}

async function clickNativeSafariElement(command, session, name, evidence, kind, selector) {
  const path = `/session/${session}`;
  const context = await command('GET', `${path}/context`);
  assert.match(
    context,
    /^WEBVIEW_/,
    `Native ${kind.toLowerCase()} click requires a selected web context`,
  );
  const entry = {
    startedAt: new Date().toISOString(),
    name,
    method: `native ${kind} element click`,
  };
  evidence.push(entry);
  let failure;
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    const elements = await command('POST', `${path}/elements`, {
      using: 'xpath',
      value: selector,
    });
    assert.equal(elements.length, 1, `Native Safari needs exactly one visible matching ${kind}`);
    await command('POST', `${path}/element/${elements[0][elementKey]}/click`, {});
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

// React Flow owns its transformed, overflow-hidden viewport. Scrolling a node
// itself also scrolls those hidden ancestors and can move it under graph chrome.
export function scrollMobileTargetIntoView(element) {
  const canvas = element.closest('.react-flow');
  const anchor = canvas || element;
  const before = canvas ? { left: canvas.scrollLeft, top: canvas.scrollTop } : null;
  anchor.scrollIntoView({ block: 'center', inline: 'center', behavior: 'instant' });
  return {
    anchor: canvas ? 'react-flow canvas' : 'target',
    canvasScrollBefore: before,
    canvasScrollAfter: canvas ? { left: canvas.scrollLeft, top: canvas.scrollTop } : null,
  };
}

// Geometry is measured in layout-viewport coordinates, including Safari's
// visual-viewport offset after it zooms a focused input.
export function inspectMobileTarget(element) {
  const view = element.ownerDocument.defaultView;
  const boundingBox = element.getBoundingClientRect();
  // Wrapped inline links have a union-box center between lines. Use the first
  // nonempty fragment for WebDriver-style hit testing, not that empty gap.
  const fragment = Array.from(element.getClientRects()).find(
    (rect) => rect.width > 0 && rect.height > 0,
  );
  const box = fragment || { x: boundingBox.x, y: boundingBox.y, width: 0, height: 0 };
  const visual = view.visualViewport;
  const viewport = {
    left: visual?.offsetLeft || 0,
    top: visual?.offsetTop || 0,
    width: visual?.width || view.innerWidth,
    height: visual?.height || view.innerHeight,
    scale: visual?.scale || 1,
  };
  const x = box.x + box.width / 2;
  const y = box.y + box.height / 2;
  const inViewport =
    box.width > 0 &&
    box.height > 0 &&
    x > viewport.left &&
    x < viewport.left + viewport.width &&
    y > viewport.top &&
    y < viewport.top + viewport.height;
  const hit = element.ownerDocument.elementFromPoint(x, y);
  const hitTarget = hit === element || element.contains(hit);
  return {
    rect: { x: box.x, y: box.y, width: box.width, height: box.height },
    boundingRect: {
      x: boundingBox.x,
      y: boundingBox.y,
      width: boundingBox.width,
      height: boundingBox.height,
    },
    viewport,
    center: { x, y },
    hitTag: hit?.tagName || null,
    hitLabel: hit?.getAttribute('aria-label') || null,
    inViewport,
    hitTarget,
    ready: inViewport && hitTarget && view.getComputedStyle(element).visibility === 'visible',
  };
}

// Only graph nodes clipped by the canvas/visual viewport need a pan. Start on a
// hit-tested blank pane so this gesture cannot drag a node or press graph controls.
export function planMobileGraphPan(element, geometry) {
  if (!element.closest('.react-flow__node')) return null;
  const canvas = element.closest('.react-flow');
  if (!canvas) return null;
  const box = canvas.getBoundingClientRect();
  const viewport = geometry.viewport;
  const bounds = {
    left: Math.max(box.x, viewport.left) + 24,
    top: Math.max(box.y, viewport.top) + 24,
    right: Math.min(box.right, viewport.left + viewport.width) - 24,
    bottom: Math.min(box.bottom, viewport.top + viewport.height) - 24,
  };
  const width = bounds.right - bounds.left;
  const height = bounds.bottom - bounds.top;
  if (width < 80 || height < 80) return null;
  const center = geometry.center;
  if (
    center.x >= bounds.left &&
    center.x <= bounds.right &&
    center.y >= bounds.top &&
    center.y <= bounds.bottom
  )
    return null;
  const clamp = (value, limit) => Math.max(-limit, Math.min(limit, value));
  const delta = {
    x: clamp((bounds.left + bounds.right) / 2 - center.x, width * 0.4),
    y: clamp((bounds.top + bounds.bottom) / 2 - center.y, height * 0.4),
  };
  const pane = canvas.querySelector('.react-flow__pane');
  if (!pane) return null;
  for (const fractionY of [0.5, 0.25, 0.75]) {
    for (const fractionX of [0.5, 0.25, 0.75]) {
      const from = { x: bounds.left + width * fractionX, y: bounds.top + height * fractionY };
      const to = { x: from.x + delta.x, y: from.y + delta.y };
      if (to.x < bounds.left || to.x > bounds.right || to.y < bounds.top || to.y > bounds.bottom)
        continue;
      if (element.ownerDocument.elementFromPoint(from.x, from.y) !== pane) continue;
      return {
        canvas: { x: box.x, y: box.y, width: box.width, height: box.height },
        bounds,
        targetCenter: center,
        from,
        to,
      };
    }
  }
  return null;
}

export const nativeSafariGraphSelector =
  '//XCUIElementTypeWebView//XCUIElementTypeOther[@name="Pipeline graph, web application" and @visible="true"]';

export async function panNativeSafariGraph(command, session, plan, evidence) {
  const path = `/session/${session}`;
  const context = await command('GET', `${path}/context`);
  assert.match(context, /^WEBVIEW_/, 'Native graph pan requires a selected web context');
  const entry = { startedAt: new Date().toISOString(), plan };
  evidence.push(entry);
  let failure;
  try {
    await command('POST', `${path}/context`, { name: 'NATIVE_APP' });
    const canvases = await command('POST', `${path}/elements`, {
      using: 'xpath',
      value: nativeSafariGraphSelector,
    });
    assert.equal(canvases.length, 1, 'Native graph pan needs exactly one visible canvas');
    const rect = await command('GET', `${path}/element/${canvases[0][elementKey]}/rect`);
    entry.nativeCanvas = rect;
    const scaleX = rect.width / plan.canvas.width;
    const scaleY = rect.height / plan.canvas.height;
    assert.ok(Number.isFinite(scaleX) && scaleX > 0 && Number.isFinite(scaleY) && scaleY > 0);
    assert.ok(
      Math.abs(scaleX / scaleY - 1) < 0.05,
      'Native and DOM graph rects must share an isotropic scale',
    );
    const nativePoint = (point) => ({
      x: rect.x + (point.x - plan.canvas.x) * scaleX,
      y: rect.y + (point.y - plan.canvas.y) * scaleY,
    });
    const from = nativePoint(plan.from);
    const to = nativePoint(plan.to);
    entry.from = from;
    entry.to = to;
    for (const point of [from, to]) {
      assert.ok(
        Number.isFinite(point.x) &&
          Number.isFinite(point.y) &&
          point.x > rect.x &&
          point.x < rect.x + rect.width &&
          point.y > rect.y &&
          point.y < rect.y + rect.height,
        'Native graph drag must stay inside its canvas',
      );
    }
    await command('POST', `${path}/execute/sync`, {
      script: 'mobile: dragFromToForDuration',
      args: [{ duration: 0.5, fromX: from.x, fromY: from.y, toX: to.x, toY: to.y }],
    });
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
