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

import React, { useEffect, useRef, useState } from 'react';
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import Hr from 'src/atoms/Hr';
import Separator from 'src/atoms/Separator';
import CollapseButtonSingle from 'src/components/CollapseButtonSingle';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { color, commonCss, fontsize, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import Buttons from 'src/lib/Buttons';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage, logger } from 'src/lib/Utils';
import { classes, stylesheet } from 'typestyle';
import {
  filterLinkedArtifactsByType,
  getArtifactTypes,
  getArtifactsFromContext,
  getEventsByExecutions,
  getExecutionsFromContext,
  getKfpV2RunContext,
  LinkedArtifact,
  getArtifactName,
} from 'src/mlmd/MlmdUtils';
import { Artifact, ArtifactType, Event, Execution } from 'src/third_party/mlmd';
import { PageProps } from './Page';
import RunList from './RunList';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import TwoLevelDropdown, {
  DropdownItem,
  DropdownSubItem,
  SelectedItem,
} from 'src/components/TwoLevelDropdown';
import MD2Tabs from 'src/atoms/MD2Tabs';
import {
  ConfusionMatrixSection,
  getHtmlViewerConfig,
  getMarkdownViewerConfig,
} from 'src/components/viewers/MetricsVisualizations';
import PlotCard from 'src/components/PlotCard';
import { ViewerConfig } from 'src/components/viewers/Viewer';

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
  outputsRow: {
    marginLeft: 15,
  },
  outputsOverflow: {
    overflowX: 'auto',
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

interface MlmdPackage {
  executions: Execution[];
  artifacts: Artifact[];
  events: Event[];
}

interface ExecutionArtifact {
  execution: Execution;
  linkedArtifacts: LinkedArtifact[];
}

interface RunArtifact {
  run: ApiRunDetail;
  executionArtifacts: ExecutionArtifact[];
}

enum MetricsType {
  SCALAR_METRICS,
  CONFUSION_MATRIX,
  ROC_CURVE,
  HTML,
  MARKDOWN,
}

const metricsTypeToString = (metricsType: MetricsType): string => {
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

const metricsTypeToFilter = (metricsType: MetricsType): string => {
  switch (metricsType) {
    case MetricsType.SCALAR_METRICS:
      return 'system.Metrics';
    case MetricsType.CONFUSION_MATRIX:
      return 'system.ClassificationMetrics';
    case MetricsType.ROC_CURVE:
      return 'system.ClassificationMetrics';
    case MetricsType.HTML:
      return 'system.HTML';
    case MetricsType.MARKDOWN:
      return 'system.Markdown';
    default:
      return '';
  }
};

// Include only the runs and executions which have artifacts of the specified type.
function filterRunArtifactsByType(
  runArtifacts: RunArtifact[],
  artifactTypes: ArtifactType[],
  metricsType: MetricsType,
): RunArtifact[] {
  const metricsFilter = metricsTypeToFilter(metricsType);
  const typeRuns: RunArtifact[] = [];
  for (const runArtifact of runArtifacts) {
    const typeExecutions: ExecutionArtifact[] = [];
    for (const e of runArtifact.executionArtifacts) {
      let typeArtifacts: LinkedArtifact[] = filterLinkedArtifactsByType(
        metricsFilter,
        artifactTypes,
        e.linkedArtifacts,
      );
      if (metricsType === MetricsType.CONFUSION_MATRIX) {
        typeArtifacts = typeArtifacts.filter(x =>
          x.artifact.getCustomPropertiesMap().has('confusionMatrix'),
        );
      } else if (metricsType === MetricsType.ROC_CURVE) {
        typeArtifacts = typeArtifacts.filter(x =>
          x.artifact.getCustomPropertiesMap().has('confidenceMetrics'),
        );
      }
      if (typeArtifacts.length > 0) {
        typeExecutions.push({
          execution: e.execution,
          linkedArtifacts: typeArtifacts,
        } as ExecutionArtifact);
      }
    }
    if (typeExecutions.length > 0) {
      typeRuns.push({
        run: runArtifact.run,
        executionArtifacts: typeExecutions,
      } as RunArtifact);
    }
  }
  return typeRuns;
}

function getRunArtifacts(runs: ApiRunDetail[], mlmdPackages: MlmdPackage[]): RunArtifact[] {
  return mlmdPackages.map((mlmdPackage, index) => {
    const events = mlmdPackage.events.filter(e => e.getType() === Event.Type.OUTPUT);

    // Match artifacts to executions.
    const artifactMap = new Map();
    mlmdPackage.artifacts.forEach(artifact => artifactMap.set(artifact.getId(), artifact));
    const executionArtifacts = mlmdPackage.executions.map(execution => {
      const executionEvents = events.filter(e => e.getExecutionId() === execution.getId());
      const linkedArtifacts: LinkedArtifact[] = [];
      for (const event of executionEvents) {
        const artifactId = event.getArtifactId();
        const artifact = artifactMap.get(artifactId);
        if (artifact) {
          linkedArtifacts.push({
            event,
            artifact,
          } as LinkedArtifact);
        } else {
          logger.warn(`The artifact with the following ID was not found: ${artifactId}`);
        }
      }
      return {
        execution,
        linkedArtifacts,
      } as ExecutionArtifact;
    });
    return {
      run: runs[index],
      executionArtifacts,
    } as RunArtifact;
  });
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

  const { data: viewerConfigs } = useQuery<ViewerConfig[], Error>(
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

  // TODO(zpChris): Show loading and error pages for HTML and Markdown.
  if (viewerConfigs && (metricsTab === MetricsType.HTML || metricsTab === MetricsType.MARKDOWN)) {
    return <PlotCard configs={viewerConfigs} title={`Static ${metricsTabText}`} />;
  }

  return <></>;
}

interface SelectedArtifact {
  selectedItem: SelectedItem;
  linkedArtifact?: LinkedArtifact;
}

interface MetricsDropdownProps {
  filteredRunArtifacts: RunArtifact[];
  metricsTab: MetricsType;
  metricsTabText: string;
  selectedArtifacts: SelectedArtifact[];
  updateSelectedArtifacts: (selectedArtifacts: SelectedArtifact[]) => void;
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

const getNamespace = (selectedItem: SelectedItem, filteredRunArtifacts: RunArtifact[]) => {
  const selectedRun = filteredRunArtifacts.find(
    runArtifact => runArtifact.run.run?.name === selectedItem.itemName,
  )?.run;
  if (selectedRun) {
    // TODO(zpChris): Move away from workflow_manifest as this is V1 specific.
    const jsonWorkflow = JSON.parse(selectedRun.pipeline_runtime!.workflow_manifest || '{}');
    return jsonWorkflow.metadata?.namespace;
  }
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

function MetricsDropdown(props: MetricsDropdownProps) {
  const {
    filteredRunArtifacts,
    metricsTab,
    metricsTabText,
    selectedArtifacts,
    updateSelectedArtifacts,
  } = props;
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

function CompareV2(props: PageProps) {
  const { updateBanner, updateToolbar } = props;

  const runlistRef = useRef<RunList>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [metricsTab, setMetricsTab] = useState(MetricsType.SCALAR_METRICS);
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);

  const [confusionMatrixArtifacts, setConfusionMatrixArtifacts] = useState<RunArtifact[]>([]);
  const [htmlArtifacts, setHtmlArtifacts] = useState<RunArtifact[]>([]);
  const [markdownArtifacts, setMarkdownArtifacts] = useState<RunArtifact[]>([]);

  // Selected artifacts for two-panel layout.
  const createSelectedArtifactArray = (count: number): SelectedArtifact[] => {
    const array: SelectedArtifact[] = [];
    for (let i = 0; i < count; i++) {
      array.push({
        selectedItem: { itemName: '', subItemName: '' },
      });
    }
    return array;
  };
  const [selectedArtifactsMap, setSelectedArtifactsMap] = useState<{
    [key: string]: SelectedArtifact[];
  }>({
    [MetricsType.CONFUSION_MATRIX]: createSelectedArtifactArray(2),
    [MetricsType.HTML]: createSelectedArtifactArray(2),
    [MetricsType.MARKDOWN]: createSelectedArtifactArray(2),
  });

  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details.
  const {
    isLoading: isLoadingRunDetails,
    isError: isErrorRunDetails,
    error: errorRunDetails,
    data: runs,
    refetch,
  } = useQuery<ApiRunDetail[], Error>(
    ['run_details', { ids: runIds }],
    () => Promise.all(runIds.map(async id => await Apis.runServiceApi.getRun(id))),
    {
      staleTime: Infinity,
    },
  );

  // Retrieves MLMD states (executions and linked artifacts) from the MLMD store.
  const {
    data: mlmdPackages,
    isLoading: isLoadingMlmdPackages,
    isError: isErrorMlmdPackages,
    error: errorMlmdPackages,
  } = useQuery<MlmdPackage[], Error>(
    ['run_artifacts', { runIds }],
    () =>
      Promise.all(
        runIds.map(async runId => {
          const context = await getKfpV2RunContext(runId);
          const executions = await getExecutionsFromContext(context);
          const artifacts = await getArtifactsFromContext(context);
          const events = await getEventsByExecutions(executions);
          return {
            executions,
            artifacts,
            events,
          } as MlmdPackage;
        }),
      ),
    {
      staleTime: Infinity,
    },
  );

  // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // so we can identify metrics artifact provided by system.
  const {
    data: artifactTypes,
    isLoading: isLoadingArtifactTypes,
    isError: isErrorArtifactTypes,
    error: errorArtifactTypes,
  } = useQuery<ArtifactType[], Error>(['artifact_types', {}], () => getArtifactTypes(), {
    staleTime: Infinity,
  });

  useEffect(() => {
    if (runs && mlmdPackages && artifactTypes) {
      const runArtifacts: RunArtifact[] = getRunArtifacts(runs, mlmdPackages);
      setConfusionMatrixArtifacts(
        filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.CONFUSION_MATRIX),
      );
      setHtmlArtifacts(filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.HTML));
      setMarkdownArtifacts(
        filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.MARKDOWN),
      );
    }
  }, [runs, mlmdPackages, artifactTypes]);

  useEffect(() => {
    if (isLoadingRunDetails || isLoadingMlmdPackages || isLoadingArtifactTypes) {
      return;
    }

    if (isErrorRunDetails) {
      (async function() {
        const errorMessage = await errorToMessage(errorRunDetails);
        updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message: `Error: failed loading ${runIds.length} runs. Click Details for more information.`,
          mode: 'error',
        });
      })();
    } else if (isErrorMlmdPackages) {
      updateBanner({
        message: 'Cannot get MLMD objects from Metadata store.',
        additionalInfo: errorMlmdPackages ? errorMlmdPackages.message : undefined,
        mode: 'error',
      });
    } else if (isErrorArtifactTypes) {
      updateBanner({
        message: 'Cannot get Artifact Types for MLMD.',
        additionalInfo: errorArtifactTypes ? errorArtifactTypes.message : undefined,
        mode: 'error',
      });
    } else {
      updateBanner({});
    }
  }, [
    runIds.length,
    isLoadingRunDetails,
    isLoadingMlmdPackages,
    isLoadingArtifactTypes,
    isErrorRunDetails,
    isErrorMlmdPackages,
    isErrorArtifactTypes,
    errorRunDetails,
    errorMlmdPackages,
    errorArtifactTypes,
    updateBanner,
  ]);

  useEffect(() => {
    const refresh = async () => {
      if (runlistRef.current) {
        await runlistRef.current.refresh();
      }
      await refetch();
    };

    const buttons = new Buttons(props, refresh);
    updateToolbar({
      actions: buttons
        .expandSections(() => {
          setIsOverviewCollapsed(false);
          setIsParamsCollapsed(false);
          setIsMetricsCollapsed(false);
        })
        .collapseSections(() => {
          setIsOverviewCollapsed(true);
          setIsParamsCollapsed(true);
          setIsMetricsCollapsed(true);
        })
        .refresh(refresh)
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Compare runs',
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (runs) {
      setSelectedIds(runs.map(r => r.run!.id!));
    }
  }, [runs]);

  const showPageError = async (message: string, error: Error | undefined) => {
    const errorMessage = await errorToMessage(error);
    updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
    });
  };

  const selectionChanged = (selectedIds: string[]): void => {
    setSelectedIds(selectedIds);
  };

  const updateSelectedArtifacts = (newArtifacts: SelectedArtifact[]) => {
    selectedArtifactsMap[metricsTab] = newArtifacts;
    setSelectedArtifactsMap(selectedArtifactsMap);
  };

  const metricsTabText = metricsTypeToString(metricsTab);
  return (
    <div className={classes(commonCss.page, padding(20, 'lrt'))}>
      {/* Overview section */}
      <CollapseButtonSingle
        sectionName={OVERVIEW_SECTION_NAME}
        collapseSection={isOverviewCollapsed}
        collapseSectionUpdate={setIsOverviewCollapsed}
      />
      {!isOverviewCollapsed && (
        <div className={commonCss.noShrink}>
          <RunList
            onError={showPageError}
            {...props}
            selectedIds={selectedIds}
            ref={runlistRef}
            runIdListMask={runIds}
            disablePaging={true}
            onSelectionChange={selectionChanged}
          />
        </div>
      )}

      <Separator orientation='vertical' />

      {/* Parameters section */}
      <CollapseButtonSingle
        sectionName={PARAMS_SECTION_NAME}
        collapseSection={isParamsCollapsed}
        collapseSectionUpdate={setIsParamsCollapsed}
      />
      {!isParamsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow, css.outputsOverflow)}>
          <Separator orientation='vertical' />
          <p>Parameter Section V2</p>
          <Hr />
        </div>
      )}

      {/* Metrics section */}
      <CollapseButtonSingle
        sectionName={METRICS_SECTION_NAME}
        collapseSection={isMetricsCollapsed}
        collapseSectionUpdate={setIsMetricsCollapsed}
      />
      {!isMetricsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <MD2Tabs
            tabs={['Scalar Metrics', 'Confusion Matrix', 'ROC Curve', 'HTML', 'Markdown']}
            selectedTab={metricsTab}
            onSwitch={setMetricsTab}
          />
          <div className={classes(padding(20, 'lrt'), css.outputsOverflow)}>
            {metricsTab === MetricsType.SCALAR_METRICS && <p>This is the Scalar Metrics tab.</p>}
            {metricsTab === MetricsType.CONFUSION_MATRIX && (
              <MetricsDropdown
                filteredRunArtifacts={confusionMatrixArtifacts}
                metricsTab={metricsTab}
                metricsTabText={metricsTabText}
                selectedArtifacts={selectedArtifactsMap[metricsTab]}
                updateSelectedArtifacts={updateSelectedArtifacts}
              />
            )}
            {metricsTab === MetricsType.ROC_CURVE && <p>This is the {metricsTabText} tab.</p>}
            {metricsTab === MetricsType.HTML && (
              <MetricsDropdown
                filteredRunArtifacts={htmlArtifacts}
                metricsTab={metricsTab}
                metricsTabText={metricsTabText}
                selectedArtifacts={selectedArtifactsMap[metricsTab]}
                updateSelectedArtifacts={updateSelectedArtifacts}
              />
            )}
            {metricsTab === MetricsType.MARKDOWN && (
              <MetricsDropdown
                filteredRunArtifacts={markdownArtifacts}
                metricsTab={metricsTab}
                metricsTabText={metricsTabText}
                selectedArtifacts={selectedArtifactsMap[metricsTab]}
                updateSelectedArtifacts={updateSelectedArtifacts}
              />
            )}
          </div>
        </div>
      )}

      <Separator orientation='vertical' />
    </div>
  );
}

export default CompareV2;
