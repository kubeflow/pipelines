// @vitest-environment node

/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { JSDOM, VirtualConsole } from 'jsdom';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

describe('generated Tailwind CSS in the test DOM', () => {
  let dom;
  const errors = [];

  beforeAll(async () => {
    const css = await readFile(path.resolve('src/build/tailwind.output.css'), 'utf8');
    const virtualConsole = new VirtualConsole();
    virtualConsole.on('jsdomError', ({ type, message }) => errors.push({ type, message }));
    dom = new JSDOM('<!doctype html><html><head></head><body></body></html>', {
      virtualConsole,
    });
    const style = dom.window.document.createElement('style');
    style.textContent = css;
    dom.window.document.head.append(style);
  });

  afterAll(() => dom?.window.close());

  it('retains the stylesheet without parser errors', () => {
    expect(errors).toEqual([]);
    expect(dom.window.document.styleSheets).toHaveLength(1);
  });

  // Graph boxes now use Graph.css and shared dimensions, covered by node tests
  // and the production graph harness; only active utilities belong in this gate.
  it.each([
    {
      name: 'navigation layout',
      classes: 'flex flex-row flex-shrink-0',
      expected: { display: 'flex', flexDirection: 'row', flexShrink: '0' },
    },
    {
      name: 'shared button layout',
      classes: 'inline-flex shrink-0 items-center justify-center whitespace-nowrap h-[34px]',
      expected: {
        display: 'inline-flex',
        flexShrink: '0',
        justifyContent: 'center',
        alignItems: 'center',
        whiteSpace: 'nowrap',
        height: '34px',
      },
    },
  ])('preserves the active $name utilities', ({ classes, expected }) => {
    const element = dom.window.document.createElement('div');
    element.className = classes;
    dom.window.document.body.append(element);
    const style = dom.window.getComputedStyle(element);
    for (const [property, value] of Object.entries(expected)) {
      expect(style[property], property).toBe(value);
    }
    element.remove();
  });
});
