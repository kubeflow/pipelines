/*
 * Copyright 2019 Google LLC
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

import MetricUtils from './MetricUtils';
import { RunMetricFormat } from '../apis/run';

describe('MetricUtils', () => {
  describe('getMetricDisplayString', () => {
    it('returns an empty string when no metric is provided', () => {
      expect(MetricUtils.getMetricDisplayString()).toEqual('');
    });

    it('returns an empty string when metric has no number value', () => {
      expect(MetricUtils.getMetricDisplayString({})).toEqual('');
    });

    it('returns a formatted string to three decimal places when metric has number value', () => {
      expect(MetricUtils.getMetricDisplayString({ number_value: 0.1234 })).toEqual('0.123');
    });

    it('returns a formatted string to specified decimal places', () => {
      expect(MetricUtils.getMetricDisplayString({ number_value: 0.1234 }, 2)).toEqual('0.12');
    });

    it('returns a formatted string when metric has format percentage and has number value', () => {
      expect(
        MetricUtils.getMetricDisplayString({
          format: RunMetricFormat.PERCENTAGE,
          number_value: 0.12341234,
        }),
      ).toEqual('12.341%');
    });

    it('returns a formatted string to specified decimal places when metric has format percentage', () => {
      expect(
        MetricUtils.getMetricDisplayString(
          {
            format: RunMetricFormat.PERCENTAGE,
            number_value: 0.12341234,
          },
          1,
        ),
      ).toEqual('12.3%');
    });
  });
});
