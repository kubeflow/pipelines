/// <reference path="../../bower_components/paper-button/paper-button.d.ts" />
/// <reference path="../../bower_components/paper-input/paper-input.d.ts" />
/// <reference path="../../bower_components/paper-item/paper-item.d.ts" />
/// <reference path="../../bower_components/paper-listbox/paper-listbox.d.ts" />
/// <reference path="../../bower_components/paper-tabs/paper-tabs.d.ts" />
/// <reference path="../../node_modules/@types/mocha/index.d.ts" />

import * as sinon from 'sinon';
import * as Utils from '../../src/lib/utils';

export const notificationStub = sinon.stub(Utils, 'showNotification');
export const dialogStub = sinon.stub(Utils, 'showDialog');

// https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/offsetParent
// Works only for non-fixed display elements.
export function isVisible(el: HTMLElement): boolean {
  return el && !!el.offsetParent;
}

// Recreates the fixture test element with the given tag, and optionally calls
// callback functions before and after attaching the new node, and waits on them.
export async function resetFixture(testTag: string,
    beforeAttach?: (el: Element) => void, afterAttach?: (el: Element) => void): Promise<void> {
  const old = document.querySelector(testTag);
  if (old) {
    document.body.removeChild(old);
  }
  const newNode = document.createElement(testTag);
  if (beforeAttach) {
    beforeAttach(newNode);
  }
  document.body.appendChild(newNode);
  if (afterAttach) {
    await afterAttach(newNode);
  }
  Polymer.flush();
}
