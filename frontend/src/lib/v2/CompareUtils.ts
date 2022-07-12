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
import { getArtifactName, getExecutionName } from 'src/mlmd/MlmdUtils';
import { getMetadataValue } from 'src/mlmd/Utils';
import { ExecutionArtifact, RunArtifact } from 'src/pages/CompareV2';
import { Value } from 'src/third_party/mlmd';
import * as jspb from 'google-protobuf';
import { chain, flatten } from 'lodash';

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

// Load the data as well as row and column labels from execution artifacts.
export const loadScalarExecutionArtifacts = (
  executionArtifacts: ExecutionArtifact[],
  xLabels: string[],
  scalarMetricNames: string[],
  dataMap: { [key: string]: string },
  artifactIndex: number,
): number => {
  for (const executionArtifact of executionArtifacts) {
    const executionText: string = getExecutionName(executionArtifact.execution) || '-';
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

// Get different components needed to construct the scalar metrics table.
const getScalarTableData = (
  scalarMetricsArtifacts: RunArtifact[],
): {
  xLabels: string[];
  scalarMetricNames: string[];
  xParentLabels: xParentLabel[];
  dataMap: { [key: string]: string };
} => {
  const xLabels: string[] = [];
  const scalarMetricNames: string[] = [];
  const xParentLabels: xParentLabel[] = [];
  const dataMap: { [key: string]: string } = {};

  let artifactIndex = 0;
  for (const runArtifact of scalarMetricsArtifacts) {
    const runName = runArtifact.run.run?.name || '-';
    const xParentLabel: xParentLabel = {
      label: runName,
      colSpan: runArtifact.executionArtifacts.length,
    };
    xParentLabels.push(xParentLabel);
    artifactIndex = loadScalarExecutionArtifacts(
      runArtifact.executionArtifacts,
      xLabels,
      scalarMetricNames,
      dataMap,
      artifactIndex,
    );
  }

  return {
    xLabels,
    scalarMetricNames,
    xParentLabels,
    dataMap,
  };
};

export const getCompareTableProps = (scalarMetricsArtifacts: RunArtifact[]) => {
  const { xLabels, scalarMetricNames, xParentLabels, dataMap } = getScalarTableData(
    scalarMetricsArtifacts,
  );

  // Order the scalar metric names by decreasing count for table y-labels.
  const yLabels = chain(flatten(scalarMetricNames))
    .countBy(p => p)
    .map((k, v) => ({ name: v, count: k }))
    .orderBy('count', 'desc')
    .map(o => o.name)
    .value();

  // Create a row of data items for each y-label.
  const rows: string[][] = yLabels.map(yLabel => {
    const row = [];
    for (let artifactIndex = 0; artifactIndex < xLabels.length; artifactIndex++) {
      const key: string = `${yLabel}-${artifactIndex}`;
      row.push(dataMap[key] || '');
    }
    return row;
  });

  return {
    xLabels,
    yLabels,
    xParentLabels,
    rows,
  } as CompareTableProps;
};
