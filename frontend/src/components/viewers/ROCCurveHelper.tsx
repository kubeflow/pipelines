import { Array as ArrayRunType, Failure, Number, Object as ObjectRunType, Static } from 'runtypes';
import { ROCCurveConfig } from './ROCCurve';
import { PlotType } from './Viewer';

const ConfidenceMetricRunType = ObjectRunType({
  confidenceThreshold: Number,
  falsePositiveRate: Number,
  recall: Number,
});
type ConfidenceMetric = Static<typeof ConfidenceMetricRunType>;
const ConfidenceMetricArrayRunType = ArrayRunType(ConfidenceMetricRunType);

function formatValidationFailure(failure: Failure, path = 'confidenceMetrics'): string {
  if ('details' in failure) {
    return Object.entries(failure.details)
      .map(([key, detail]) => formatValidationFailure(detail, `${path}.${key}`))
      .join('; ');
  }
  return `${path}: ${failure.message}`;
}

export function validateConfidenceMetrics(inputs: unknown): { error?: string } {
  const result = ConfidenceMetricArrayRunType.inspect(inputs);
  if (!result.success) {
    return { error: formatValidationFailure(result) + '. Data: ' + JSON.stringify(inputs) };
  }
  return {};
}

export function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig {
  const arraytypesCheck = ConfidenceMetricArrayRunType.check(confidenceMetricsArray);
  return {
    type: PlotType.ROC,
    data: arraytypesCheck.map((metric) => ({
      label: metric.confidenceThreshold as unknown as string,
      x: metric.falsePositiveRate,
      y: metric.recall,
    })),
  };
}
