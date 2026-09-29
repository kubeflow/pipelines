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

import {
  enabledDisplayStringV2,
  errorToMessage,
  formatDateString,
  logger,
  sanitizeExternalHref,
} from './Utils';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { expectErrors } from 'src/TestUtils';

describe('Utils', () => {
  describe('sanitizeExternalHref', () => {
    it('returns http and https URLs unchanged', () => {
      expect(sanitizeExternalHref('http://example.com/x')).toEqual('http://example.com/x');
      expect(sanitizeExternalHref('https://example.com/x')).toEqual('https://example.com/x');
      expect(sanitizeExternalHref('HTTPS://example.com/x')).toEqual('HTTPS://example.com/x');
    });

    it('rejects javascript: and other unsafe schemes', () => {
      // eslint-disable-next-line no-script-url
      expect(sanitizeExternalHref('javascript:alert(1)')).toBeUndefined();
      // eslint-disable-next-line no-script-url
      expect(sanitizeExternalHref('JAVASCRIPT:alert(1)')).toBeUndefined();
      expect(sanitizeExternalHref('data:text/html,<script>alert(1)</script>')).toBeUndefined();
      expect(sanitizeExternalHref('vbscript:msgbox(1)')).toBeUndefined();
      expect(sanitizeExternalHref('ftp://example.com/x')).toBeUndefined();
      expect(sanitizeExternalHref('//example.com')).toBeUndefined();
      expect(sanitizeExternalHref('not a URL')).toBeUndefined();
    });

    it('returns undefined for empty or missing input', () => {
      expect(sanitizeExternalHref('')).toBeUndefined();
      expect(sanitizeExternalHref(undefined)).toBeUndefined();
    });
  });

  describe('log', () => {
    it('logs to console', () => {
      // tslint:disable-next-line:no-console
      const backup = console.log;
      global.console.log = vi.fn();
      logger.verbose('something to console');
      // tslint:disable-next-line:no-console
      expect(console.log).toBeCalledWith('something to console');
      global.console.log = backup;
    });

    it('logs to console error', () => {
      // tslint:disable-next-line:no-console
      const backup = console.error;
      global.console.error = vi.fn();
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

  describe('errorToMessage', () => {
    it('handles an Error instance', async () => {
      expect(await errorToMessage(new Error('test error'))).toBe('test error');
    });

    it('handles object with text() method that returns a string', async () => {
      const mockResponse = {
        text: () => 'direct string response',
      };
      const result = await errorToMessage(mockResponse);
      expect(result).toBe('direct string response');
    });

    it('handles plain object input', async () => {
      const errorObj = { message: 'error message', code: 500 };
      const result = await errorToMessage(errorObj);
      expect(result).toBe(JSON.stringify(errorObj));
    });

    it('handles string input', async () => {
      expect(await errorToMessage('string error')).toBe('"string error"');
    });

    it('handles undefined input', async () => {
      expect(await errorToMessage(undefined)).toBe('');
    });

    it('handles array input', async () => {
      expect(await errorToMessage([1, 'error', { key: 'value' }])).toBe(
        '[1,"error",{"key":"value"}]',
      );
    });

    it('handles number input', async () => {
      expect(await errorToMessage(404)).toBe('404');
    });
  });
});
