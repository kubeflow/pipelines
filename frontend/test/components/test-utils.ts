/// <reference path="../../bower_components/paper-button/paper-button.d.ts" />
/// <reference path="../../bower_components/paper-input/paper-input.d.ts" />
/// <reference path="../../bower_components/paper-item/paper-item.d.ts" />
/// <reference path="../../bower_components/paper-listbox/paper-listbox.d.ts" />
/// <reference path="../../bower_components/paper-tabs/paper-tabs.d.ts" />
/// <reference path="../node_modules/@types/mocha/index.d.ts" />

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
    beforeAttach?: (el: any) => void, afterAttach?: (el: any) => void): Promise<void> {
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

// Stubs map stores replacements for HTML tags. This is part of the stubTag and
// restoreTag functions, which are loosely based on Polymer's replace addon
// that is part of web-component-tester:
// https://github.com/Polymer/web-component-tester/blob/master/browser/mocha/replace.ts
const stubMap = new Map<string, string>();

export function stubTag(oldTagName: string, tagName: string): any {
  // Standardizes our replacements map
  oldTagName = oldTagName.toLowerCase();
  tagName = tagName.toLowerCase();

  stubMap.set(oldTagName, tagName);

  // If the function is already a stub, restore it to original
  if ((document.importNode as any).isSinonProxy) {
    return;
  }

  // Keep a reference to the original `document.importNode`
  // implementation for later:
  const originalImportNode = document.importNode;

  // Use Sinon to stub `document.ImportNode`
  sinon.stub(document, 'importNode', (origContent: any, deep: boolean) => {
    const templateClone = document.createElement('template');
    const content = templateClone.content;
    const inertDoc = content.ownerDocument;

    // imports node from inertDoc which holds inert nodes.
    templateClone.content.appendChild(inertDoc.importNode(origContent, true));

    // Traverses the tree. A recently-replaced node will be put next, so if a
    // node is replaced, it will be checked if it needs to be replaced again.
    const nodeIterator = document.createNodeIterator(content, NodeFilter.SHOW_ELEMENT);
    let node = nodeIterator.nextNode() as Element;
    while (node) {
      const currentTagName = node.tagName.toLowerCase();

      if (stubMap.has(currentTagName)) {
        // Create a replacement
        const replacement = document.createElement(stubMap.get(currentTagName)!);

        // For all attributes in the original node, set that attribute on the replacement
        for (const attr of Array.from(node.attributes)) {
          replacement.setAttribute(attr.name, attr.value);
        }

        // Replace the original node with the replacement node:
        node.parentNode!.replaceChild(replacement, node);
      }
      node = nodeIterator.nextNode() as Element;
    }

    return originalImportNode.call(document, content, deep);
  });
}

export function restoreTag(tagName: string): void {
  // If there are no more replacements, restore the stubbed version of
  // `document.importNode`
  if (!stubMap.size) {
    const documentImportNode = document.importNode as any;
    if (documentImportNode.isSinonProxy) {
      documentImportNode.restore();
    }
  }

  // Remove the tag from the replacement map
  stubMap.delete(tagName);
}
