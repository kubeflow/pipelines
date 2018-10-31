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
  logger,
  formatDateString,
  enabledDisplayString,
  getRunTime,
} from './Utils';

describe('Utils', () => {
  describe('log', () => {
    it('logs to console', () => {
      // tslint:disable-next-line:no-console
      const backup = console.log;
      global.console.log = jest.fn();
      logger.verbose('something to console');
      // tslint:disable-next-line:no-console
      expect(console.log).toBeCalledWith('something to console');
      global.console.log = backup;
    });

    it('logs to console error', () => {
      // tslint:disable-next-line:no-console
      const backup = console.error;
      global.console.error = jest.fn();
      logger.error('something to console error');
      // tslint:disable-next-line:no-console
      expect(console.error).toBeCalledWith('something to console error');
      global.console.error = backup;
    });
  });

  describe('formatDateString', () => {
    it('handles an ISO format date string', () => {
      const d = new Date(2018, 1, 13, 9, 55);
      expect(formatDateString(d.toISOString())).toBe(d.toLocaleString());
    });

    it('handles a locale format date string', () => {
      const d = new Date(2018, 1, 13, 9, 55);
      expect(formatDateString(d.toLocaleString())).toBe(d.toLocaleString());
    });

    it('handles a date', () => {
      const d = new Date(2018, 1, 13, 9, 55);
      expect(formatDateString(d)).toBe(d.toLocaleString());
    });

    it('handles undefined', () => {
      expect(formatDateString(undefined)).toBe('-');
    });
  });

  describe('enabledDisplayString', () => {
    it('handles undefined', () => {
      expect(enabledDisplayString(undefined, true)).toBe('-');
      expect(enabledDisplayString(undefined, false)).toBe('-');
    });

    it('handles a trigger according to the enabled flag', () => {
      expect(enabledDisplayString({}, true)).toBe('Yes');
      expect(enabledDisplayString({}, false)).toBe('No');
    });
  });

  describe('getRunTime', () => {
    it('handles no workflow', () => {
      expect(getRunTime()).toBe('-');
    });

    it('handles no status', () => {
      const workflow = {} as any;
      expect(getRunTime(workflow)).toBe('-');
    });

    it('handles no phase', () => {
      const workflow = {
        status: {
          finishedAt: 'some end',
          startedAt: 'some start',
        }
      } as any;
      expect(getRunTime(workflow)).toBe('-');
    });

    it('handles no start date', () => {
      const workflow = {
        status: {
          finishedAt: 'some end',
          phase: 'Succeeded',
        }
      } as any;
      expect(getRunTime(workflow)).toBe('-');
    });

    it('handles no end date', () => {
      const workflow = {
        status: {
          phase: 'Succeeded',
          startedAt: 'some end',
        }
      } as any;
      expect(getRunTime(workflow)).toBe('-');
    });

    it('computes seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 3, 56, 25).toISOString(),
          phase: 'Succeeded',
          startedAt: new Date(2018, 1, 2, 3, 55, 30).toISOString(),
        }
      } as any;
      expect(getRunTime(workflow)).toBe('0:00:55');
    });

    it('computes minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 3, 59, 25).toISOString(),
          phase: 'Succeeded',
          startedAt: new Date(2018, 1, 2, 3, 55, 10).toISOString(),
        }
      } as any;
      expect(getRunTime(workflow)).toBe('0:04:15');
    });

    it('computes days/minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 4, 55, 10).toISOString(),
          phase: 'Succeeded',
          startedAt: new Date(2018, 1, 2, 3, 55, 10).toISOString(),
        }
      } as any;
      expect(getRunTime(workflow)).toBe('1:00:00');
    });

    it('computes padded days/minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 4, 56, 11).toISOString(),
          phase: 'Succeeded',
          startedAt: new Date(2018, 1, 2, 3, 55, 10).toISOString(),
        }
      } as any;
      expect(getRunTime(workflow)).toBe('1:01:01');
    });

    it('shows negative sign if start date is after end date', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 2, 3, 55, 11).toISOString(),
          phase: 'Succeeded',
          startedAt: new Date(2018, 1, 2, 3, 55, 13).toISOString(),
        }
      } as any;
      expect(getRunTime(workflow)).toBe('-0:00:02');
    });
  });
});
