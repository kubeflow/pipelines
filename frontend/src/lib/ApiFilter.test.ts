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

import { encodeNameFilter } from './ApiFilter';
import { V2beta1PredicateOperation } from 'src/apisv2beta1/filter';

it('omits an empty filter', () => {
  expect(encodeNameFilter('')).toBe('');
});

it('encodes special characters once and composes predicates without mutating them', () => {
  const types = [
    { key: 'type', operation: V2beta1PredicateOperation.IN, int_values: { values: [6, 7] } },
  ];
  const name = '100% / & \"quoted\" 雪';
  expect(JSON.parse(decodeURIComponent(encodeNameFilter(name, types)))).toEqual({
    predicates: [{ key: 'name', operation: 'IS_SUBSTRING', string_value: name }, ...types],
  });
  expect(types).toHaveLength(1);
  expect(JSON.parse(decodeURIComponent(encodeNameFilter('', types)))).toEqual({
    predicates: types,
  });
});
