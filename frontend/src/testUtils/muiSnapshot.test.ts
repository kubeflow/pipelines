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

import { normalizeReactUseIdAttrs } from './muiSnapshot';

function fragmentFor(id: string): DocumentFragment {
  const template = document.createElement('template');
  template.innerHTML = `
    <label id="${id}-label" for="${id}">Run name</label>
    <input id="${id}" aria-labelledby="${id}-label heading"
      aria-describedby="${id}-helper-text" aria-controls="${id}-list" />
    <span id="${id}-helper-text">Required</span>
    <ul id="${id}-list"><li>Existing run</li></ul>
    <h2 id="heading">Create a run</h2>
    <a href="/runs/new">New run</a>
  `;
  return template.content;
}

describe('normalizeReactUseIdAttrs', () => {
  it.each(['_r_abc_', ':r1:'])('preserves label and description relationships for %s', (id) => {
    const fragment = fragmentFor(id);
    normalizeReactUseIdAttrs(fragment);

    const input = fragment.querySelector('input')!;
    expect(input.id).toBe('react-use-id-0');
    expect(fragment.querySelector('label')?.getAttribute('for')).toBe(input.id);
    expect(fragment.querySelector('label')?.getAttribute('id')).toBe(`${input.id}-label`);
    expect(input.getAttribute('aria-labelledby')).toBe(`${input.id}-label heading`);
    expect(input.getAttribute('aria-describedby')).toBe(`${input.id}-helper-text`);
    expect(fragment.querySelector('span')?.getAttribute('id')).toBe(`${input.id}-helper-text`);
    expect(input.getAttribute('aria-controls')).toBe(`${input.id}-list`);
    expect(fragment.querySelector('ul')?.getAttribute('id')).toBe(`${input.id}-list`);
    expect(fragment.querySelector('h2')?.getAttribute('id')).toBe('heading');
    expect(fragment.querySelector('a')?.getAttribute('href')).toBe('/runs/new');
    expect(fragment.querySelector('label')?.textContent).toBe('Run name');
  });

  it('normalizes changing generated IDs without changing ordinary identifiers', () => {
    const first = fragmentFor('_r_1_');
    const second = fragmentFor('_r_xyz_');
    normalizeReactUseIdAttrs(first);
    normalizeReactUseIdAttrs(second);
    expect(first.isEqualNode(second)).toBe(true);

    const ordinary = fragmentFor('run-name');
    const original = ordinary.cloneNode(true);
    normalizeReactUseIdAttrs(ordinary);
    expect(ordinary.isEqualNode(original)).toBe(true);
  });
});
