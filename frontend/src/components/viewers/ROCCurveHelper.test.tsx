/*
 * Copyright 2026 The Kubeflow Authors
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

import { validateConfidenceMetrics, buildRocCurveConfig } from './ROCCurveHelper';
import { PlotType } from './Viewer';

describe('validateConfidenceMetrics', () => {
  it.each([
    [
      'valid confidence metrics',
      [
        { confidenceThreshold: 0.9, falsePositiveRate: 0.1, recall: 0.95 },
        { confidenceThreshold: 0.5, falsePositiveRate: 0.3, recall: 0.8 },
      ],
      false,
    ],
    ['an empty array', [], false],
    [
      'invalid data types',
      [{ confidenceThreshold: 'not-a-number', falsePositiveRate: 'bad', recall: 'invalid' }],
      true,
    ],
    ['missing fields', [{ confidenceThreshold: 0.9 }], true],
    ['non-array input', 'not-an-array', true],
    ['array containing null', [null], true],
  ] as const)('validates %s', (_label, input, expectError) => {
    const result = validateConfidenceMetrics(input);
    expect(result.error).toEqual(expectError ? expect.anything() : undefined);
  });
});

describe('buildRocCurveConfig', () => {
  // The ConfidenceMetric TypeScript type declares confidenceThreshold as string,
  // but the runtypes runtime validator (ConfidenceMetricArrayRunType) expects Number
  // for all three fields. We use `as any` to match the runtime expectation.
  it('builds ROC curve config from valid metrics', () => {
    const metrics = [
      { confidenceThreshold: 0.9, falsePositiveRate: 0.1, recall: 0.95 },
      { confidenceThreshold: 0.5, falsePositiveRate: 0.3, recall: 0.8 },
    ] as any;
    const config = buildRocCurveConfig(metrics);
    expect(config.type).toBe(PlotType.ROC);
    expect(config.data).toHaveLength(2);
    expect(config.data[0]).toEqual({ label: 0.9, x: 0.1, y: 0.95 });
    expect(config.data[1]).toEqual({ label: 0.5, x: 0.3, y: 0.8 });
  });

  it('returns empty data array for empty metrics', () => {
    const config = buildRocCurveConfig([] as any);
    expect(config.type).toBe(PlotType.ROC);
    expect(config.data).toHaveLength(0);
  });

  it('maps falsePositiveRate to x and recall to y', () => {
    const metrics = [{ confidenceThreshold: 0.7, falsePositiveRate: 0.2, recall: 0.85 }] as any;
    const config = buildRocCurveConfig(metrics);
    expect(config.data[0].x).toBe(0.2);
    expect(config.data[0].y).toBe(0.85);
  });
});
