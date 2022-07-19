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
import { ApiRunDetail } from 'src/apis/run';
import { stylesheet } from 'typestyle';

export const compareCss = stylesheet({
  relativeContainer: {
    position: 'relative',
    height: '30rem',
  },
});

export interface ExecutionArtifact {
  execution: Execution;
  linkedArtifacts: LinkedArtifact[];
}

export interface RunArtifact {
  run: ApiRunDetail;
  executionArtifacts: ExecutionArtifact[];
}

interface ScalarRowData {
  row: string[];
  dataCount: number;
}

interface ScalarTableData {
  xLabels: string[];
  xParentLabels: xParentLabel[];
  dataMap: { [key: string]: ScalarRowData };
}

export interface RunArtifactData {
  runArtifacts: RunArtifact[];
  artifactCount: number;
}

export const getCompareTableProps = (
  scalarMetricsArtifacts: RunArtifact[],
  artifactCount: number,
): CompareTableProps => {
  const scalarTableData = getScalarTableData(scalarMetricsArtifacts, artifactCount);

  // Sort by decreasing data item count.
  const sortedDataList = Object.entries(scalarTableData.dataMap).sort(
    (a, b) => b[1].dataCount - a[1].dataCount,
  );
  const yLabels: string[] = [];
  const rows: string[][] = [];
  for (const sortedDataItem of sortedDataList) {
    yLabels.push(sortedDataItem[0]);
    rows.push(sortedDataItem[1].row);
  }
  return {
    xLabels: scalarTableData.xLabels,
    yLabels,
    xParentLabels: scalarTableData.xParentLabels,
    rows,
  } as CompareTableProps;
};

// Get different components needed to construct the scalar metrics table.
const getScalarTableData = (
  scalarMetricsArtifacts: RunArtifact[],
  artifactCount: number,
): ScalarTableData => {
  const xLabels: string[] = [];
  const xParentLabels: xParentLabel[] = [];
  const dataMap: { [key: string]: ScalarRowData } = {};

  let artifactIndex = 0;
  for (const runArtifact of scalarMetricsArtifacts) {
    const runName = runArtifact.run.run?.name || '-';

    const newArtifactIndex = loadScalarExecutionArtifacts(
      runArtifact.executionArtifacts,
      xLabels,
      dataMap,
      artifactIndex,
      artifactCount,
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
    xParentLabels,
    dataMap,
  } as ScalarTableData;
};

// Load the data as well as row and column labels from execution artifacts.
const loadScalarExecutionArtifacts = (
  executionArtifacts: ExecutionArtifact[],
  xLabels: string[],
  dataMap: { [key: string]: ScalarRowData },
  artifactIndex: number,
  artifactCount: number,
): number => {
  for (const executionArtifact of executionArtifacts) {
    const executionText: string = getExecutionDisplayName(executionArtifact.execution) || '-';
    for (const linkedArtifact of executionArtifact.linkedArtifacts) {
      const linkedArtifactText: string = getArtifactName(linkedArtifact) || '-';
      const xLabel = `${executionText} > ${linkedArtifactText}`;
      xLabels.push(xLabel);

      const customProperties = linkedArtifact.artifact.getCustomPropertiesMap();
      addScalarDataItems(customProperties, dataMap, artifactIndex, artifactCount);
      artifactIndex++;
    }
  }
  return artifactIndex;
};

// Add the scalar metric names and data items.
const addScalarDataItems = (
  customProperties: jspb.Map<string, Value>,
  dataMap: { [key: string]: ScalarRowData },
  artifactIndex: number,
  artifactCount: number,
) => {
  for (const entry of customProperties.getEntryList()) {
    const scalarMetricName: string = entry[0];
    if (scalarMetricName === 'display_name') {
      continue;
    }

    if (!dataMap[scalarMetricName]) {
      dataMap[scalarMetricName] = {
        row: Array(artifactCount).fill(''),
        dataCount: 0,
      };
    }

    dataMap[scalarMetricName].row[artifactIndex] = JSON.stringify(
      getMetadataValue(customProperties.get(scalarMetricName)),
    );
    dataMap[scalarMetricName].dataCount++;
  }
};

export enum MetricsType {
  SCALAR_METRICS,
  CONFUSION_MATRIX,
  ROC_CURVE,
  HTML,
  MARKDOWN,
}

export const metricsTypeToString = (metricsType: MetricsType): string => {
  switch (metricsType) {
    case MetricsType.SCALAR_METRICS:
      return 'Scalar Metrics';
    case MetricsType.CONFUSION_MATRIX:
      return 'Confusion Matrix';
    case MetricsType.ROC_CURVE:
      return 'ROC Curve';
    case MetricsType.HTML:
      return 'HTML';
    case MetricsType.MARKDOWN:
      return 'Markdown';
    default:
      return '';
  }
};
