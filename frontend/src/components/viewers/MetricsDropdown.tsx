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

import React, { useEffect, useState } from 'react';
import { color, commonCss, fontsize, zIndex } from 'src/Css';
import { classes, stylesheet } from 'typestyle';
import { LinkedArtifact, getArtifactName } from 'src/mlmd/MlmdUtils';
import TwoLevelDropdown, {
  DropdownItem,
  DropdownSubItem,
  SelectedItem,
} from 'src/components/TwoLevelDropdown';
import {
  ConfusionMatrixSection,
  getHtmlViewerConfig,
  getMarkdownViewerConfig,
} from 'src/components/viewers/MetricsVisualizations';
import PlotCard from 'src/components/PlotCard';
import { ViewerConfig } from 'src/components/viewers/Viewer';
import CircularProgress from '@material-ui/core/CircularProgress';
import Banner from 'src/components/Banner';
import { ExecutionArtifact, MetricsType, RunArtifact, SelectedArtifact } from 'src/pages/CompareV2';
import { useQuery } from 'react-query';
import { errorToMessage, logger } from 'src/lib/Utils';
import { Execution } from 'src/third_party/mlmd';
import { metricsTypeToString } from 'src/lib/v2/CompareUtils';

const css = stylesheet({
  leftCell: {
    borderRight: `3px solid ${color.divider}`,
  },
  rightCell: {
    borderLeft: `3px solid ${color.divider}`,
  },
  cell: {
    borderCollapse: 'collapse',
    padding: '1rem',
    verticalAlign: 'top',
  },
  errorBanner: {
    maxWidth: '40rem',
  },
  relativeContainer: {
    position: 'relative',
    height: '30rem',
  },
  visualizationPlaceholder: {
    width: '40rem',
    height: '30rem',
    backgroundColor: color.lightGrey,
    borderRadius: '1rem',
    margin: '1rem 0',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
  },
  visualizationPlaceholderText: {
    fontSize: fontsize.medium,
    textAlign: 'center',
    padding: '2rem',
  },
});

interface MetricsDropdownProps {
  filteredRunArtifacts: RunArtifact[];
  metricsTab: MetricsType;
  selectedArtifacts: SelectedArtifact[];
  updateSelectedArtifacts: (selectedArtifacts: SelectedArtifact[]) => void;
}

export default function MetricsDropdown(props: MetricsDropdownProps) {
  const { filteredRunArtifacts, metricsTab, selectedArtifacts, updateSelectedArtifacts } = props;
  const [firstSelectedItem, setFirstSelectedItem] = useState<SelectedItem>(
    selectedArtifacts[0].selectedItem,
  );
  const [secondSelectedItem, setSecondSelectedItem] = useState<SelectedItem>(
    selectedArtifacts[1].selectedItem,
  );
  const [firstSelectedNamespace, setFirstSelectedNamespace] = useState<string | undefined>();
  const [secondSelectedNamespace, setSecondSelectedNamespace] = useState<string | undefined>();

  const updateSelectedItemAndArtifact = (
    setSelectedItem: (selectedItem: SelectedItem) => void,
    setSelectedNamespace: (selectedNamespace: string | undefined) => void,
    panelIndex: number,
    selectedItem: SelectedItem,
  ): void => {
    setSelectedItem(selectedItem);
    selectedArtifacts[panelIndex].selectedItem = selectedItem;
    const linkedArtifact = getLinkedArtifactFromSelectedItem(filteredRunArtifacts, selectedItem);
    selectedArtifacts[panelIndex].linkedArtifact = linkedArtifact;
    setSelectedNamespace(getNamespace(selectedItem, filteredRunArtifacts));
    updateSelectedArtifacts(selectedArtifacts);
  };

  const dropdownItems: DropdownItem[] = getDropdownItems(filteredRunArtifacts);
  const metricsTabText = metricsTypeToString(metricsTab);

  if (dropdownItems.length === 0) {
    return <p>There are no {metricsTabText} artifacts available on the selected runs.</p>;
  }

  return (
    <table>
      <tbody>
        <tr>
          <td className={classes(css.cell, css.leftCell)}>
            <TwoLevelDropdown
              title={`Choose a first ${metricsTabText} artifact`}
              items={dropdownItems}
              selectedItem={firstSelectedItem}
              setSelectedItem={updateSelectedItemAndArtifact.bind(
                null,
                setFirstSelectedItem,
                setFirstSelectedNamespace,
                0,
              )}
            />
            <VisualizationPanelItem
              metricsTab={metricsTab}
              metricsTabText={metricsTabText}
              linkedArtifact={selectedArtifacts[0].linkedArtifact}
              namespace={firstSelectedNamespace}
            />
          </td>
          <td className={classes(css.cell, css.rightCell)}>
            <TwoLevelDropdown
              title={`Choose a second ${metricsTabText} artifact`}
              items={dropdownItems}
              selectedItem={secondSelectedItem}
              setSelectedItem={updateSelectedItemAndArtifact.bind(
                null,
                setSecondSelectedItem,
                setSecondSelectedNamespace,
                1,
              )}
            />
            <VisualizationPanelItem
              metricsTab={metricsTab}
              metricsTabText={metricsTabText}
              linkedArtifact={selectedArtifacts[1].linkedArtifact}
              namespace={secondSelectedNamespace}
            />
          </td>
        </tr>
      </tbody>
    </table>
  );
}

interface VisualizationPlaceholderProps {
  metricsTabText: string;
}

function VisualizationPlaceholder(props: VisualizationPlaceholderProps) {
  const { metricsTabText } = props;
  return (
    <div className={classes(css.visualizationPlaceholder)}>
      <p className={classes(css.visualizationPlaceholderText)}>
        The selected {metricsTabText} will be displayed here.
      </p>
    </div>
  );
}

interface VisualizationPanelItemProps {
  metricsTab: MetricsType;
  metricsTabText: string;
  linkedArtifact: LinkedArtifact | undefined;
  namespace: string | undefined;
}

function VisualizationPanelItem(props: VisualizationPanelItemProps) {
  const { metricsTab, metricsTabText, linkedArtifact, namespace } = props;
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [showError, setShowError] = useState<boolean>(false);

  const { isLoading, isError, error, data: viewerConfigs } = useQuery<ViewerConfig[], Error>(
    [
      'viewerConfig',
      {
        artifact: linkedArtifact?.artifact.getId(),
        namespace,
      },
    ],
    async () => {
      let viewerConfigs: ViewerConfig[] = [];
      if (linkedArtifact) {
        try {
          if (metricsTab === MetricsType.HTML) {
            viewerConfigs = await getHtmlViewerConfig([linkedArtifact], namespace);
          } else if (metricsTab === MetricsType.MARKDOWN) {
            viewerConfigs = await getMarkdownViewerConfig([linkedArtifact], namespace);
          }
        } catch (err) {
          throw err;
        }
      }
      return viewerConfigs;
    },
    { staleTime: Infinity },
  );

  useEffect(() => {
    if (isLoading) {
      return;
    }

    if (isError) {
      (async function() {
        const updatedMessage = await errorToMessage(error);
        setErrorMessage(updatedMessage);
        setShowError(true);
      })();
    } else {
      setShowError(false);
    }
  }, [isLoading, isError, error, setErrorMessage, setShowError]);

  if (!linkedArtifact) {
    return <VisualizationPlaceholder metricsTabText={metricsTabText} />;
  }

  if (metricsTab === MetricsType.CONFUSION_MATRIX) {
    return (
      <React.Fragment key={linkedArtifact.artifact.getId()}>
        <ConfusionMatrixSection artifact={linkedArtifact.artifact} />
      </React.Fragment>
    );
  }

  if (showError || isLoading) {
    return (
      <React.Fragment>
        {showError && (
          <div className={css.errorBanner}>
            <Banner
              message={`Error: failed loading ${metricsTabText} file.${errorMessage &&
                ' Click Details for more information.'}`}
              mode='error'
              additionalInfo={errorMessage}
              leftAlign
            />
          </div>
        )}
        {isLoading && (
          <div className={css.relativeContainer}>
            <CircularProgress
              size={25}
              className={commonCss.absoluteCenter}
              style={{ zIndex: zIndex.BUSY_OVERLAY }}
              role='circularprogress'
            />
          </div>
        )}
      </React.Fragment>
    );
  }

  if (viewerConfigs && (metricsTab === MetricsType.HTML || metricsTab === MetricsType.MARKDOWN)) {
    return <PlotCard configs={viewerConfigs} title={`Static ${metricsTabText}`} />;
  }

  return <></>;
}

const logDisplayNameWarning = (type: string, id: string) =>
  logger.warn(`Failed to fetch the display name of the ${type} with the following ID: ${id}`);

const getExecutionName = (execution: Execution) =>
  execution
    .getCustomPropertiesMap()
    .get('display_name')
    ?.getStringValue();

// Group each artifact name with its parent execution name.
function getDropdownSubLinkedArtifacts(linkedArtifacts: LinkedArtifact[], subItemName: string) {
  const executionLinkedArtifacts: DropdownSubItem[] = [];
  for (const linkedArtifact of linkedArtifacts) {
    const artifactName = getArtifactName(linkedArtifact);
    const artifactId = linkedArtifact.artifact.getId().toString();
    if (!artifactName) {
      logDisplayNameWarning('artifact', artifactId);
    }

    executionLinkedArtifacts.push({
      name: subItemName,
      secondaryName: artifactName || artifactId,
    } as DropdownSubItem);
  }
  return executionLinkedArtifacts;
}

// Combine execution names and artifact names into the same dropdown sub item list.
function getDropdownSubItems(executionArtifacts: ExecutionArtifact[]) {
  const subItems: DropdownSubItem[] = [];
  for (const executionArtifact of executionArtifacts) {
    const executionName = getExecutionName(executionArtifact.execution);
    const executionId = executionArtifact.execution.getId().toString();
    if (!executionName) {
      logDisplayNameWarning('execution', executionId);
    }

    const executionLinkedArtifacts: DropdownSubItem[] = getDropdownSubLinkedArtifacts(
      executionArtifact.linkedArtifacts,
      executionName || executionId,
    );
    subItems.push(...executionLinkedArtifacts);
  }
  return subItems;
}

function getDropdownItems(filteredRunArtifacts: RunArtifact[]) {
  const dropdownItems: DropdownItem[] = [];
  for (const runArtifact of filteredRunArtifacts) {
    const runName = runArtifact.run.run?.name;
    if (!runName) {
      logDisplayNameWarning('run', runArtifact.run.run!.id!);
      continue;
    }

    const subItems: DropdownSubItem[] = getDropdownSubItems(runArtifact.executionArtifacts);
    if (subItems.length > 0) {
      dropdownItems.push({
        name: runName,
        subItems,
      } as DropdownItem);
    }
  }

  return dropdownItems;
}

const getNamespace = (
  selectedItem: SelectedItem,
  filteredRunArtifacts: RunArtifact[],
): string | undefined => {
  const selectedRun = filteredRunArtifacts.find(
    runArtifact => runArtifact.run.run?.name === selectedItem.itemName,
  )?.run;
  let namespace: string | undefined;
  if (selectedRun) {
    // TODO(zpChris): Move away from workflow_manifest as this is V1 specific.
    const jsonWorkflow = JSON.parse(selectedRun.pipeline_runtime!.workflow_manifest || '{}');
    namespace = jsonWorkflow.metadata?.namespace;
  }
  return namespace;
};

function getLinkedArtifactFromSelectedItem(
  filteredRunArtifacts: RunArtifact[],
  selectedItem: SelectedItem,
): LinkedArtifact | undefined {
  const filteredRunArtifact = filteredRunArtifacts.find(
    runArtifact => runArtifact.run.run?.name === selectedItem.itemName,
  );

  const executionArtifact = filteredRunArtifact?.executionArtifacts.find(executionArtifact => {
    const executionText: string =
      getExecutionName(executionArtifact.execution) ||
      executionArtifact.execution.getId().toString();
    return executionText === selectedItem.subItemName;
  });

  const linkedArtifact = executionArtifact?.linkedArtifacts.find(linkedArtifact => {
    const linkedArtifactText: string =
      getArtifactName(linkedArtifact) || linkedArtifact.artifact.getId().toString();
    return linkedArtifactText === selectedItem.subItemSecondaryName;
  });

  return linkedArtifact;
}
