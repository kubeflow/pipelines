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

import { ApiRunDetail, ApiRun } from '../apis/run';
import { CompareTableProps } from '../components/CompareTable';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { chain, flatten } from 'lodash';
import WorkflowParser from './WorkflowParser';
import { logger } from './Utils';
import RunUtils from './RunUtils';
import MetricUtils from './MetricUtils';
import { parseTaskDisplayNameByNodeId } from './ParserUtils';

export default class CompareUtils {
  /**
   * Given a list of run and worklow objects, returns CompareTableProps containing parameter
   * names on y axis, run names on x axis, and parameter values in rows.
   * runs and workflowObjects are required to be in the same order, so workflowObject[i]
   * is run[i]'s parsed workflow object
   */
  public static getParamsCompareProps(
    runs: ApiRunDetail[],
    workflowObjects: Workflow[],
  ): CompareTableProps {
    if (runs.length !== workflowObjects.length) {
      // Should never happen
      logger.error('Numbers of passed in runs and workflows do not match');
    }

    const yLabels = chain(flatten(workflowObjects.map(w => WorkflowParser.getParameters(w))))
      .countBy(p => p.name) // count by parameter name
      .map((k, v) => ({ name: v, count: k })) // convert to counter objects
      .orderBy('count', 'desc') // sort on count field, descending
      .map(o => o.name)
      .value();

    const rows = yLabels.map(name => {
      return workflowObjects.map(w => {
        const params = WorkflowParser.getParameters(w);
        const param = params.find(p => p.name === name);
        return param ? param.value || '' : '';
      });
    });

    return {
      rows,
      xLabels: runs.map(r => r.run!.name!),
      yLabels,
    };
  }

  public static multiRunMetricsCompareProps(runs: ApiRun[]): CompareTableProps {
    const metricMetadataMap = RunUtils.runsToMetricMetadataMap(runs);

    const yLabels = Array.from(metricMetadataMap.keys());

    const rows = yLabels.map(name =>
      runs.map(r =>
        // TODO(rjbauer): This logic isn't quite right. A single run can have multiple metrics
        // with the same name, but here we're stopping once we find one.
        MetricUtils.getMetricDisplayString((r.metrics || []).find(m => m.name === name)),
      ),
    );

    return {
      rows,
      xLabels: runs.map(r => r.name!),
      yLabels,
    };
  }

  /**
   * For a given run and its runtime workflow, a CompareTableProps object is returned containing:
   * xLabels: an array of unique meeric names produced during the run's execution
   * yLabels: an array of display names (falling back to node IDs) for steps of the execution which
   * produced metrics
   * rows: an array of arrays, each representing all of the metrics produced by a given step of the
   * execution.
   */
  public static singleRunToMetricsCompareProps(
    run?: ApiRun,
    workflow?: Workflow,
  ): CompareTableProps {
    let rows: string[][] = [];
    let xLabels: string[] = [];
    const yLabels: string[] = [];
    if (run) {
      const namesToNodesToValues: Map<string, Map<string, string>> = new Map();
      const nodeIds: Set<string> = new Set();

      (run.metrics || []).forEach(metric => {
        if (
          !metric.name ||
          !metric.node_id ||
          metric.number_value === undefined ||
          isNaN(metric.number_value)
        ) {
          return;
        }
        const nodeToValue = namesToNodesToValues.get(metric.name) || new Map();
        nodeToValue.set(metric.node_id, MetricUtils.getMetricDisplayString(metric));
        namesToNodesToValues.set(metric.name, nodeToValue);
        nodeIds.add(metric.node_id);
      });

      xLabels = Array.from(namesToNodesToValues.keys());

      rows = Array.from(nodeIds.keys()).map(nodeId => {
        yLabels.push(parseTaskDisplayNameByNodeId(nodeId, workflow));
        return xLabels.map(metricName => namesToNodesToValues.get(metricName)!.get(nodeId) || '');
      });
    }

    return {
      rows,
      xLabels,
      yLabels,
    };
  }
}
