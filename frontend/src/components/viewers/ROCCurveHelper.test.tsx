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
  it('returns no error for valid confidence metrics', () => {
    const validMetrics = [
      { confidenceThreshold: 0.9, falsePositiveRate: 0.1, recall: 0.95 },
      { confidenceThreshold: 0.5, falsePositiveRate: 0.3, recall: 0.8 },
    ];
    const result = validateConfidenceMetrics(validMetrics);
    expect(result.error).toBeUndefined();
  });

  it('returns an error for invalid data types', () => {
    const invalidMetrics = [
      { confidenceThreshold: 'not-a-number', falsePositiveRate: 'bad', recall: 'invalid' },
    ];
    const result = validateConfidenceMetrics(invalidMetrics);
    expect(result.error).toBeDefined();
  });

  it('returns an error for missing fields', () => {
    const incompleteMetrics = [{ confidenceThreshold: 0.9 }];
    const result = validateConfidenceMetrics(incompleteMetrics);
    expect(result.error).toBeDefined();
  });

  it('returns no error for an empty array', () => {
    const result = validateConfidenceMetrics([]);
    expect(result.error).toBeUndefined();
  });

  it('returns an error for non-array input', () => {
    const result = validateConfidenceMetrics('not-an-array');
    expect(result.error).toBeDefined();
  });

  it('returns an error when array contains null', () => {
    const result = validateConfidenceMetrics([null]);
    expect(result.error).toBeDefined();
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
