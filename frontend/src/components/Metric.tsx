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

import * as React from 'react';
import { ApiRunMetric, RunMetricFormat } from '../apis/run';
import { MetricMetadata } from '../lib/RunUtils';
import { stylesheet } from 'typestyle';
import { logger } from '../lib/Utils';
import MetricUtils from '../lib/MetricUtils';

const css = stylesheet({
  metricContainer: {
    background: '#f6f7f9',
    marginLeft: 6,
    marginRight: 10,
  },
  metricFill: {
    background: '#cbf0f8',
    boxSizing: 'border-box',
    color: '#202124',
    fontFamily: 'Roboto',
    fontSize: 13,
    textIndent: 6,
  },
});

interface MetricProps {
  metadata?: MetricMetadata;
  metric?: ApiRunMetric;
}

class Metric extends React.PureComponent<MetricProps> {
  public render(): JSX.Element {
    const { metric, metadata } = this.props;
    if (!metric || metric.number_value === undefined) {
      return <div />;
    }

    const displayString = MetricUtils.getMetricDisplayString(metric);
    let width = '';

    if (metric.format === RunMetricFormat.PERCENTAGE) {
      width = `calc(${displayString})`;
    } else {
      // Non-percentage metrics must contain metadata
      if (!metadata) {
        return <div />;
      }

      const leftSpace = 6;

      if (metadata.maxValue === 0 && metadata.minValue === 0) {
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (metric.number_value - metadata.minValue < 0) {
        logger.error(
          `Metric ${metadata.name}'s value:` +
            ` (${metric.number_value}) was lower than the supposed minimum of` +
            ` (${metadata.minValue})`,
        );
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (metadata.maxValue - metric.number_value < 0) {
        logger.error(
          `Metric ${metadata.name}'s value:` +
            ` (${metric.number_value}) was greater than the supposed maximum of` +
            ` (${metadata.maxValue})`,
        );
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      const barWidth =
        ((metric.number_value - metadata.minValue) / (metadata.maxValue - metadata.minValue)) * 100;

      width = `calc(${barWidth}%)`;
    }
    return (
      <div className={css.metricContainer}>
        <div className={css.metricFill} style={{ width }}>
          {displayString}
        </div>
      </div>
    );
  }
}

export default Metric;
