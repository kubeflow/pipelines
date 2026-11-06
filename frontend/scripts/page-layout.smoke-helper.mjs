/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import assert from 'node:assert/strict';

// Check rendered geometry, including transient states, rather than just a CSS class.
export async function assertPageInsets(content) {
  await content.waitFor({ state: 'visible', timeout: 15000 });
  const geometry = await content.evaluate(async (element) => {
    await document.fonts.ready;
    await new Promise(requestAnimationFrame);
    const container = element.closest('[data-page-layout="padded"]');
    if (!container) return null;
    const box = container.getBoundingClientRect();
    const child = element.getBoundingClientRect();
    const style = getComputedStyle(container);
    return {
      left: child.left - box.left,
      right: box.right - child.right,
      paddingLeft: parseFloat(style.paddingLeft),
      paddingRight: parseFloat(style.paddingRight),
      containerRight: box.right,
      viewport: document.documentElement.clientWidth,
    };
  });
  assert.ok(geometry, 'page content must use a shared padded container');
  assert.ok(geometry.paddingLeft >= 16 && geometry.paddingRight >= 16,
    `page gutters must remain visible: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.left >= geometry.paddingLeft - 1 &&
    geometry.right >= geometry.paddingRight - 1,
    `content must stay inside page gutters: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.containerRight <= geometry.viewport + 1,
    `page container must fit the viewport: ${JSON.stringify(geometry)}`);
}

export const layoutViewports = [
  { width: 1440, height: 900 },
  { width: 390, height: 844 },
];
