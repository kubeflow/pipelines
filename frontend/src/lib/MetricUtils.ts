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

import { ApiRunMetric, RunMetricFormat } from '../apis/run';

function getMetricDisplayString(metric?: ApiRunMetric, decimalPlaces = 3): string {
  if (!metric || metric.number_value === undefined) {
    return '';
  }

  if (metric.format === RunMetricFormat.PERCENTAGE) {
    return (metric.number_value * 100).toFixed(decimalPlaces) + '%';
  }

  return metric.number_value.toFixed(decimalPlaces);
}

export default {
  getMetricDisplayString,
};
