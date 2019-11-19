/*
 * Copyright 2018 Google LLC
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

import {
  NodePhase,
  hasFinished,
  statusBgColors,
  statusToBgColor,
  checkIfTerminated,
} from './StatusUtils';

describe('StatusUtils', () => {
  describe('hasFinished', () => {
    [
      NodePhase.ERROR,
      NodePhase.FAILED,
      NodePhase.SUCCEEDED,
      NodePhase.SKIPPED,
      NodePhase.TERMINATED,
    ].forEach(status => {
      it(`returns \'true\' if status is: ${status}`, () => {
        expect(hasFinished(status)).toBe(true);
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.UNKNOWN, NodePhase.TERMINATING].forEach(
      status => {
        it(`returns \'false\' if status is: ${status}`, () => {
          expect(hasFinished(status)).toBe(false);
        });
      },
    );

    it("returns 'false' if status is undefined", () => {
      expect(hasFinished(undefined)).toBe(false);
    });

    it("returns 'false' if status is invalid", () => {
      expect(hasFinished('bad phase' as any)).toBe(false);
    });
  });

  describe('statusToBgColor', () => {
    it('handles an invalid phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor('bad phase' as any)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it("handles an 'Unknown' phase", () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(NodePhase.UNKNOWN)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'Unknown');
    });

    it("returns color 'not started' if status is undefined", () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(undefined)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    it("returns color 'not started' if status is 'Pending'", () => {
      expect(statusToBgColor(NodePhase.PENDING)).toEqual(statusBgColors.notStarted);
    });

    [NodePhase.ERROR, NodePhase.FAILED].forEach(status => {
      it(`returns color \'error\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.error);
      });
    });

    [NodePhase.RUNNING, NodePhase.TERMINATING].forEach(status => {
      it(`returns color \'running\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.running);
      });
    });

    [NodePhase.SKIPPED, NodePhase.TERMINATED].forEach(status => {
      it(`returns color \'terminated or skipped\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.terminatedOrSkipped);
      });
    });

    it("returns color 'succeeded' if status is 'Succeeded'", () => {
      expect(statusToBgColor(NodePhase.SUCCEEDED)).toEqual(statusBgColors.succeeded);
    });
  });

  describe('checkIfTerminated', () => {
    it("returns status 'terminated' if status is 'failed' and error message is 'terminated'", () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'terminated')).toEqual(NodePhase.TERMINATED);
    });

    [
      NodePhase.SUCCEEDED,
      NodePhase.ERROR,
      NodePhase.SKIPPED,
      NodePhase.PENDING,
      NodePhase.RUNNING,
      NodePhase.TERMINATING,
      NodePhase.UNKNOWN,
    ].forEach(status => {
      it(`returns the original status, even if message is 'terminated', if status is: ${status}`, () => {
        expect(checkIfTerminated(status, 'terminated')).toEqual(status);
      });
    });

    it("returns 'failed' if status is 'failed' and no error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED)).toEqual(NodePhase.FAILED);
    });

    it("returns 'failed' if status is 'failed' and empty error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED, '')).toEqual(NodePhase.FAILED);
    });

    it("returns 'failed' if status is 'failed' and arbitrary error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'some random error')).toEqual(NodePhase.FAILED);
    });
  });
});
