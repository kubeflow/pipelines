/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { dump } from 'js-yaml';

import { loadYaml } from './YamlLoad';

describe('loadYaml', () => {
  it('resolves merge keys instead of leaving a literal << property', () => {
    const parsed = loadYaml(`
defaults: &defaults
  image: alpine
  imagePullPolicy: IfNotPresent
container:
  <<: *defaults
  command: ["echo"]
`) as any;

    expect(parsed.container.image).toBe('alpine');
    expect(parsed.container.imagePullPolicy).toBe('IfNotPresent');
    expect(parsed.container.command).toEqual(['echo']);
    expect(Object.prototype.hasOwnProperty.call(parsed.container, '<<')).toBe(false);
  });

  it('lets the merged mapping override inherited keys', () => {
    const parsed = loadYaml(`
base: &base
  image: alpine
step:
  <<: *base
  image: ubuntu
`) as any;

    expect(parsed.step.image).toBe('ubuntu');
  });

  it('parses documents without merge keys unchanged', () => {
    expect(loadYaml('a: 1\nb: two\n')).toEqual({ a: 1, b: 'two' });
  });

  // js-yaml 4 returned undefined for blank input and null for input that
  // carried no document. js-yaml 5 throws for both.
  it('returns undefined for blank input', () => {
    expect(loadYaml('')).toBeUndefined();
    expect(loadYaml('   ')).toBeUndefined();
    expect(loadYaml('\n')).toBeUndefined();
  });

  it('returns null for input that carries no document', () => {
    expect(loadYaml('# comment\n')).toBeNull();
    expect(loadYaml('# one\n# two\n')).toBeNull();
    expect(loadYaml('...\n')).toBeNull();
    expect(loadYaml('---\n')).toBeNull();
  });

  it('round trips no-document input through dump the way js-yaml 4 did', () => {
    expect(dump(loadYaml(''))).toBe('');
    expect(dump(loadYaml('# comment\n'))).toBe('null\n');
  });

  it('still raises genuine syntax errors', () => {
    expect(() => loadYaml('a: [1,2')).toThrow();
    expect(() => loadYaml('a:\n  - b\n\tc: d')).toThrow();
  });

  it('still raises on multiple documents rather than yielding the first', () => {
    expect(() => loadYaml('a: 1\n---\nb: 2\n')).toThrow();
  });
});
