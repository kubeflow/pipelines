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

import { ApiRunDetail } from '../apis/run';
import { CompareTableProps } from '../components/CompareTable';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { chain, flatten } from 'lodash';
import WorkflowParser from './WorkflowParser';
import { logger } from './Utils';

export default class CompareUtils {
  /**
   * Given a list of run and worklow objects, returns CompareTableProps containing parameter
   * names on y axis, run names on x axis, and parameter values in rows.
   * runs and workflowObjects are required to be in the same order, so workflowObject[i]
   * is run[i]'s parsed workflow object
   */
  public static getParamsCompareProps(runs: ApiRunDetail[], workflowObjects: Workflow[]): CompareTableProps {
    if (runs.length !== workflowObjects.length) {
      // Should never happen
      logger.error('Numbers of passed in runs and workflows do not match');
    }

    const xLabels = runs.map(r => r.run!.name!);

    const yLabels = chain(flatten(workflowObjects
      .map(w => WorkflowParser.getParameters(w))))
      .countBy(p => p.name)                         // count by parameter name
      .map((k, v) => ({ name: v, count: k }))       // convert to counter objects
      .orderBy('count', 'desc')                     // sort on count field, descending
      .map(o => o.name)
      .value();

    const rows = yLabels.map(name => {
      return workflowObjects
        .map(w => {
          const params = WorkflowParser.getParameters(w);
          const param = params.find(p => p.name === name);
          return param ? param.value || '' : '';
        });
    });

    return { rows, xLabels, yLabels };
  }
}
