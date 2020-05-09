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

import { NodePhase } from './StatusUtils';
import {
  enabledDisplayString,
  formatDateString,
  generateMinioArtifactUrl,
  getRunDuration,
  getRunDurationFromWorkflow,
  logger,
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

  describe('getRunDuration', () => {
    it('handles no run', () => {
      expect(getRunDuration()).toBe('-');
    });

    it('handles no status', () => {
      const run = {} as any;
      expect(getRunDuration(run)).toBe('-');
    });

    it('handles no created_at', () => {
      const run = {
        finished_at: 'some end',
      } as any;
      expect(getRunDuration(run)).toBe('-');
    });

    it('handles no finished_at', () => {
      const run = {
        created_at: 'some start',
      } as any;
      expect(getRunDuration(run)).toBe('-');
    });

    it('handles run which has not finished', () => {
      const run = {
        created_at: new Date(2018, 1, 2, 3, 55, 30).toISOString(),
        finished_at: new Date(2018, 1, 3, 3, 56, 25).toISOString(),
        status: NodePhase.RUNNING,
      } as any;
      expect(getRunDuration(run)).toBe('-');
    });

    it('computes seconds', () => {
      const run = {
        created_at: new Date(2018, 1, 3, 3, 55, 30).toISOString(),
        finished_at: new Date(2018, 1, 3, 3, 56, 25).toISOString(),
        status: NodePhase.SUCCEEDED,
      } as any;
      expect(getRunDuration(run)).toBe('0:00:55');
    });

    it('computes minutes/seconds', () => {
      const run = {
        created_at: new Date(2018, 1, 3, 3, 55, 10).toISOString(),
        finished_at: new Date(2018, 1, 3, 3, 59, 25).toISOString(),
        status: NodePhase.SUCCEEDED,
      } as any;
      expect(getRunDuration(run)).toBe('0:04:15');
    });

    it('computes hours/minutes/seconds', () => {
      const run = {
        created_at: new Date(2018, 1, 2, 3, 55, 10).toISOString(),
        finished_at: new Date(2018, 1, 3, 4, 55, 10).toISOString(),
        status: NodePhase.SUCCEEDED,
      } as any;
      expect(getRunDuration(run)).toBe('25:00:00');
    });

    it('computes padded hours/minutes/seconds', () => {
      const run = {
        created_at: new Date(2018, 1, 2, 3, 55, 10).toISOString(),
        finished_at: new Date(2018, 1, 3, 4, 56, 11).toISOString(),
        status: NodePhase.SUCCEEDED,
      } as any;
      expect(getRunDuration(run)).toBe('25:01:01');
    });

    it('shows negative sign if start date is after end date', () => {
      const run = {
        created_at: new Date(2018, 1, 2, 3, 55, 13).toISOString(),
        finished_at: new Date(2018, 1, 2, 3, 55, 11).toISOString(),
        status: NodePhase.SUCCEEDED,
      } as any;
      expect(getRunDuration(run)).toBe('-0:00:02');
    });
  });

  describe('getRunDurationFromWorkflow', () => {
    it('handles no workflow', () => {
      expect(getRunDurationFromWorkflow()).toBe('-');
    });

    it('handles no status', () => {
      const workflow = {} as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('-');
    });

    it('handles no start date', () => {
      const workflow = {
        status: {
          finishedAt: 'some end',
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('-');
    });

    it('handles no end date', () => {
      const workflow = {
        status: {
          startedAt: 'some end',
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('-');
    });

    it('computes seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 3, 56, 25).toISOString(),
          startedAt: new Date(2018, 1, 3, 3, 55, 30).toISOString(),
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('0:00:55');
    });

    it('computes minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 3, 59, 25).toISOString(),
          startedAt: new Date(2018, 1, 3, 3, 55, 10).toISOString(),
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('0:04:15');
    });

    it('computes hours/minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 4, 55, 10).toISOString(),
          startedAt: new Date(2018, 1, 3, 3, 55, 10).toISOString(),
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('1:00:00');
    });

    it('computes padded hours/minutes/seconds run time if status is provided', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 3, 4, 56, 11).toISOString(),
          startedAt: new Date(2018, 1, 3, 3, 55, 10).toISOString(),
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('1:01:01');
    });

    it('shows negative sign if start date is after end date', () => {
      const workflow = {
        status: {
          finishedAt: new Date(2018, 1, 2, 3, 55, 11).toISOString(),
          startedAt: new Date(2018, 1, 2, 3, 55, 13).toISOString(),
        },
      } as any;
      expect(getRunDurationFromWorkflow(workflow)).toBe('-0:00:02');
    });
  });

  describe('generateMinioArtifactUrl', () => {
    it('handles minio:// URIs', () => {
      expect(generateMinioArtifactUrl('minio://my-bucket/a/b/c')).toBe(
        'artifacts/get?source=minio&bucket=my-bucket&key=a%2Fb%2Fc',
      );
    });

    it('handles non-minio URIs', () => {
      expect(generateMinioArtifactUrl('minio://my-bucket-a-b-c')).toBe(undefined);
    });

    it('handles broken minio URIs', () => {
      expect(generateMinioArtifactUrl('ZZZ://my-bucket/a/b/c')).toBe(undefined);
    });
  });
});
