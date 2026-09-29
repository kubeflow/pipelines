/*
 * Copyright 2018 The Kubeflow Authors
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

import { hasFinishedV2, statusBgColors, statusToBgColorV2 } from './StatusUtils';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';

describe('StatusUtils', () => {
  describe('hasFinishedV2', () => {
    it('treats a paused run as active', () => {
      expect(hasFinishedV2(V2beta1RuntimeState.PAUSED)).toBe(false);
    });
  });
  describe('hasFinishedV2', () => {
    [
      V2beta1RuntimeState.SUCCEEDED,
      V2beta1RuntimeState.FAILED,
      V2beta1RuntimeState.CANCELED,
      V2beta1RuntimeState.SKIPPED,
    ].forEach((state) => {
      it(`returns 'true' for finished state: ${state}`, () => {
        expect(hasFinishedV2(state)).toBe(true);
      });
    });

    [
      V2beta1RuntimeState.PENDING,
      V2beta1RuntimeState.RUNNING,
      V2beta1RuntimeState.CANCELING,
      V2beta1RuntimeState.PAUSED,
      V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED,
    ].forEach((state) => {
      it(`returns 'false' for non-finished state: ${state}`, () => {
        expect(hasFinishedV2(state)).toBe(false);
      });
    });

    it('does not throw for undefined state', () => {
      expect(() => hasFinishedV2(undefined)).not.toThrow();
    });
  });

  describe('statusToBgColorV2', () => {
    it("returns 'notStarted' color for PAUSED state", () => {
      expect(statusToBgColorV2(V2beta1RuntimeState.PAUSED)).toEqual(statusBgColors.notStarted);
    });

    it("returns 'running' color for RUNNING state", () => {
      expect(statusToBgColorV2(V2beta1RuntimeState.RUNNING)).toEqual(statusBgColors.running);
    });

    it("returns 'running' color for CANCELING state", () => {
      expect(statusToBgColorV2(V2beta1RuntimeState.CANCELING)).toEqual(statusBgColors.running);
    });

    it("returns 'succeeded' color for SUCCEEDED state", () => {
      expect(statusToBgColorV2(V2beta1RuntimeState.SUCCEEDED)).toEqual(statusBgColors.succeeded);
    });

    it("returns 'error' color for FAILED state", () => {
      expect(statusToBgColorV2(V2beta1RuntimeState.FAILED)).toEqual(statusBgColors.error);
    });

    [V2beta1RuntimeState.SKIPPED, V2beta1RuntimeState.CANCELED].forEach((state) => {
      it(`returns 'terminatedOrSkipped' color for state: ${state}`, () => {
        expect(statusToBgColorV2(state)).toEqual(statusBgColors.terminatedOrSkipped);
      });
    });
  });
});
