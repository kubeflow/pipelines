/*
 * Copyright 2025 The Kubeflow Authors
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

import { isReservedArtifactProperty } from './ReservedArtifactProperties';

describe('isReservedArtifactProperty', () => {
  it('returns true for store_session_info', () => {
    expect(isReservedArtifactProperty('store_session_info')).toBe(true);
  });

  it('returns false for non-reserved keys', () => {
    expect(isReservedArtifactProperty('user_metric')).toBe(false);
    expect(isReservedArtifactProperty('accuracy')).toBe(false);
  });

  it('returns false for display_name (filtered separately)', () => {
    expect(isReservedArtifactProperty('display_name')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(isReservedArtifactProperty('')).toBe(false);
  });

  it('returns false for undefined', () => {
    expect(isReservedArtifactProperty(undefined)).toBe(false);
  });

  it('is case-sensitive', () => {
    expect(isReservedArtifactProperty('STORE_SESSION_INFO')).toBe(false);
    expect(isReservedArtifactProperty('Store_Session_Info')).toBe(false);
  });
});
