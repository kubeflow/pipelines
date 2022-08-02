import { Array as ArrayRunType, Number, Record, ValidationError } from 'runtypes';
import { ROCCurveConfig } from './ROCCurve';
import { PlotType } from './Viewer';

type ConfidenceMetric = {
  confidenceThreshold: string;
  falsePositiveRate: number;
  recall: number;
};

const ConfidenceMetricRunType = Record({
  confidenceThreshold: Number,
  falsePositiveRate: Number,
  recall: Number,
});
const ConfidenceMetricArrayRunType = ArrayRunType(ConfidenceMetricRunType);
export function validateConfidenceMetrics(inputs: any): { error?: string } {
  try {
    ConfidenceMetricArrayRunType.check(inputs);
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(inputs) };
    }
  }
  return {};
}

export function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig {
  const arraytypesCheck = ConfidenceMetricArrayRunType.check(confidenceMetricsArray);
  return {
    type: PlotType.ROC,
    data: arraytypesCheck.map(metric => ({
      label: (metric.confidenceThreshold as unknown) as string,
      x: metric.falsePositiveRate,
      y: metric.recall,
    })),
  };
}
