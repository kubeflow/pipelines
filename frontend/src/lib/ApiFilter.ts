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

import { V2PredicateOperation } from 'src/apisv2/filter';
import type { V2Predicate } from 'src/apisv2/filter';

/** Compose list predicates before encoding once for the API query parameter. */
export function encodeNameFilter(name: string, additionalPredicates: V2Predicate[] = []): string {
  const predicates: V2Predicate[] = name
    ? [{ key: 'name', operation: V2PredicateOperation.IS_SUBSTRING, string_value: name }]
    : [];
  predicates.push(...additionalPredicates);
  return predicates.length ? encodeURIComponent(JSON.stringify({ predicates })) : '';
}
