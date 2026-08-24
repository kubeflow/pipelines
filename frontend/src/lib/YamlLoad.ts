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

import { CORE_SCHEMA, load as jsYamlLoad, mergeTag } from 'js-yaml';

/**
 * js-yaml 5 resolves merge keys only when `mergeTag` is present in the schema,
 * where js-yaml 4 resolved them by default. KFP parses Argo and Kubernetes
 * manifests that inherit fields through anchors, so without this a merged
 * mapping keeps a literal `<<` property and the inherited fields are lost.
 */
const MANIFEST_SCHEMA = CORE_SCHEMA.withTags(mergeTag);

/** Parses manifest YAML, preserving merge key (`<<`) resolution. */
export function loadYaml(text: string): unknown {
  return jsYamlLoad(text, { schema: MANIFEST_SCHEMA });
}
