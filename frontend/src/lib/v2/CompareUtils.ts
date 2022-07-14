/*
 * Copyright 2022 The Kubeflow Authors
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

import { CompareTableProps, xParentLabel } from 'src/components/CompareTable';
import { getArtifactName, getExecutionDisplayName, LinkedArtifact } from 'src/mlmd/MlmdUtils';
import { getMetadataValue } from 'src/mlmd/Utils';
import { Execution, Value } from 'src/third_party/mlmd';
import * as jspb from 'google-protobuf';
import { chain, flatten } from 'lodash';
import { ApiRunDetail } from 'src/apis/run';

export interface ExecutionArtifact {
  execution: Execution;
  linkedArtifacts: LinkedArtifact[];
}

export interface RunArtifact {
  run: ApiRunDetail;
  executionArtifacts: ExecutionArtifact[];
}

interface ScalarTableData {
  xLabels: string[];
  scalarMetricNames: string[];
  xParentLabels: xParentLabel[];
  dataMap: { [key: string]: string };
}

export const getCompareTableProps = (scalarMetricsArtifacts: RunArtifact[]): CompareTableProps => {
  const scalarTableData = getScalarTableData(scalarMetricsArtifacts);

  // Order the scalar metric names by decreasing count for table y-labels.
  const yLabels = chain(flatten(scalarTableData.scalarMetricNames))
    .countBy(p => p)
    .map((k, v) => ({ name: v, count: k }))
    .orderBy('count', 'desc')
    .map(o => o.name)
    .value();

  // Create a row of data items for each y-label.
  const rows: string[][] = yLabels.map(yLabel => {
    const row = [];
    for (let artifactIndex = 0; artifactIndex < scalarTableData.xLabels.length; artifactIndex++) {
      const key: string = `${yLabel}-${artifactIndex}`;
      row.push(scalarTableData.dataMap[key] || '');
    }
    return row;
  });

  return {
    xLabels: scalarTableData.xLabels,
    yLabels,
    xParentLabels: scalarTableData.xParentLabels,
    rows,
  } as CompareTableProps;
};

// Get different components needed to construct the scalar metrics table.
const getScalarTableData = (scalarMetricsArtifacts: RunArtifact[]): ScalarTableData => {
  const xLabels: string[] = [];
  const scalarMetricNames: string[] = [];
  const xParentLabels: xParentLabel[] = [];
  const dataMap: { [key: string]: string } = {};

  let artifactIndex = 0;
  for (const runArtifact of scalarMetricsArtifacts) {
    const runName = runArtifact.run.run?.name || '-';

    const newArtifactIndex = loadScalarExecutionArtifacts(
      runArtifact.executionArtifacts,
      xLabels,
      scalarMetricNames,
      dataMap,
      artifactIndex,
    );

    const xParentLabel: xParentLabel = {
      label: runName,
      colSpan: newArtifactIndex - artifactIndex,
    };
    xParentLabels.push(xParentLabel);
    artifactIndex = newArtifactIndex;
  }

  return {
    xLabels,
    scalarMetricNames,
    xParentLabels,
    dataMap,
  } as ScalarTableData;
};

// Load the data as well as row and column labels from execution artifacts.
const loadScalarExecutionArtifacts = (
  executionArtifacts: ExecutionArtifact[],
  xLabels: string[],
  scalarMetricNames: string[],
  dataMap: { [key: string]: string },
  artifactIndex: number,
): number => {
  for (const executionArtifact of executionArtifacts) {
    const executionText: string = getExecutionDisplayName(executionArtifact.execution) || '-';
    for (const linkedArtifact of executionArtifact.linkedArtifacts) {
      const linkedArtifactText: string = getArtifactName(linkedArtifact) || '-';
      const xLabel = `${executionText} > ${linkedArtifactText}`;
      xLabels.push(xLabel);

      const customProperties = linkedArtifact.artifact.getCustomPropertiesMap();
      addScalarDataItems(customProperties, scalarMetricNames, dataMap, artifactIndex);
      artifactIndex++;
    }
  }
  return artifactIndex;
};

// Add the scalar metric names and data items.
const addScalarDataItems = (
  customProperties: jspb.Map<string, Value>,
  scalarMetricNames: string[],
  dataMap: { [key: string]: string },
  artifactIndex: number,
) => {
  for (const entry of customProperties.getEntryList()) {
    const scalarMetricName: string = entry[0];
    if (scalarMetricName === 'display_name') {
      continue;
    }
    scalarMetricNames.push(scalarMetricName);

    // The scalar metric name and artifact index create a unique key per data item.
    const key: string = `${scalarMetricName}-${artifactIndex}`;
    dataMap[key] = JSON.stringify(getMetadataValue(customProperties.get(scalarMetricName)));
  }
};
