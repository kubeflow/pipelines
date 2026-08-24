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

import { CORE_SCHEMA, load as jsYamlLoad, loadAll as jsYamlLoadAll, mergeTag } from 'js-yaml';

/**
 * js-yaml 5 resolves merge keys only when `mergeTag` is present in the schema,
 * where js-yaml 4 resolved them by default. KFP parses Argo and Kubernetes
 * manifests that inherit fields through anchors, so without this a merged
 * mapping keeps a literal `<<` property and the inherited fields are lost.
 */
const MANIFEST_SCHEMA = CORE_SCHEMA.withTags(mergeTag);

/**
 * Parses manifest YAML, preserving merge key (`<<`) resolution.
 *
 * Restores the js-yaml 4 result for input that carries no document. js-yaml 5
 * throws `YAMLException: expected a document, but the input is empty` for a
 * blank string, comment-only text, or a bare `...` marker, where js-yaml 4
 * returned undefined for blank input and null otherwise. Call sites parse
 * arbitrary template strings during render, so throwing is reachable.
 *
 * loadAll reports zero documents for exactly those inputs while still raising
 * genuine syntax errors, which is what distinguishes the two cases without
 * matching on exception messages. Input holding more than one document is
 * handed back to `load` so it raises its own single-document error rather than
 * silently yielding the first.
 */
export function loadYaml(text: string): unknown {
  if (!text.trim()) {
    return undefined;
  }

  const documents: unknown[] = [];
  jsYamlLoadAll(text, (document) => documents.push(document), { schema: MANIFEST_SCHEMA });

  if (documents.length === 0) {
    return null;
  }
  if (documents.length === 1) {
    return documents[0];
  }
  return jsYamlLoad(text, { schema: MANIFEST_SCHEMA });
}
