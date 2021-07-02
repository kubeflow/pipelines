/*
 * Copyright 2021 The Kubeflow Authors
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

/**
 * This file provides common types used by the Lineage Explorer components
 */
import { Artifact, Execution } from './library';

export type LineageCardType = 'artifact' | 'execution';
export interface LineageRow {
  prev?: boolean;
  next?: boolean;
  typedResource: LineageTypedResource;
  resourceDetailsPageRoute: string;
}

export type LineageResource = Artifact | Execution;
export type LineageTypedResource =
  | { type: 'artifact'; resource: Artifact }
  | { type: 'execution'; resource: Execution };

export const DEFAULT_LINEAGE_CARD_TYPE = 'artifact' as LineageCardType;
