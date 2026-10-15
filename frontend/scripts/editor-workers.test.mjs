// @vitest-environment node

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

import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { createContext, runInContext } from 'node:vm';
import { describe, expect, it } from 'vitest';

const require = createRequire(import.meta.url);

// Exercise the packaged worker protocol without launching or installing a browser.
describe('packaged Ace workers', () => {
  for (const [mode, classname, valid, invalid] of [
    ['yaml', 'YamlWorker', 'name: example\nitems:\n  - first\n', 'name: [unfinished'],
    ['json', 'JsonWorker', '{"name":"example","items":["first"]}', '{"name": }'],
  ]) {
    it(`${mode} preserves its complete model and reports syntax errors`, async () => {
      const messages = [];
      const timers = new Map();
      let timerId = 0;
      const context = createContext({
        postMessage: (message) => messages.push(structuredClone(message)),
        setTimeout: (callback) => {
          timers.set(++timerId, callback);
          return timerId;
        },
        clearTimeout: (id) => timers.delete(id),
        importScripts: () => {
          throw new Error('Packaged worker requested an external script');
        },
      });
      const code = await readFile(
        require.resolve(`ace-builds/src-min-noconflict/worker-${mode}.js`),
        'utf8',
      );
      runInContext(code, context, { timeout: 1000 });
      const send = (data) => context.onmessage({ data });
      send({ init: true, module: `ace/mode/${mode}_worker`, classname });
      for (const [model, invalidSyntax] of [
        [valid, false],
        [invalid, true],
        [valid, false],
      ]) {
        messages.length = 0;
        send({ command: 'setValue', args: [model] });
        send({ command: 'getValue', args: [42] });
        expect(messages.find((message) => message.type === 'call')).toMatchObject({
          id: 42,
          data: model,
        });
        context.main.deferredUpdate.call();
        const annotations = messages.find((message) => message.name === 'annotate');
        expect(annotations).toBeDefined();
        expect(annotations.data.length > 0).toBe(invalidSyntax);
        expect(messages.filter((message) => message.type === 'error')).toEqual([]);
      }
      expect(timers.size).toBe(0);
    });
  }
});
