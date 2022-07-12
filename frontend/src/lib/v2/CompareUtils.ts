import { xParentLabel } from 'src/components/CompareTable';
import { getArtifactName, getExecutionName } from 'src/mlmd/MlmdUtils';
import { getMetadataValue } from 'src/mlmd/Utils';
import { ExecutionArtifact, RunArtifact } from 'src/pages/CompareV2';
import { Value } from 'src/third_party/mlmd';
import * as jspb from 'google-protobuf';

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
const loadScalarExecutionArtifacts = (
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
export const getScalarTableData = (
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
