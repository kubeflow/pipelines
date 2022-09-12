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

import React, { useContext, useEffect, useRef, useState } from 'react';
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import Separator from 'src/atoms/Separator';
import CollapseButtonSingle from 'src/components/CollapseButtonSingle';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { commonCss, padding, zIndex } from 'src/Css';
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
} from 'src/mlmd/MlmdUtils';
import { Artifact, ArtifactType, Event, Execution } from 'src/third_party/mlmd';
import { PageProps } from './Page';
import RunList from './RunList';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { SelectedItem } from 'src/components/TwoLevelDropdown';
import MD2Tabs from 'src/atoms/MD2Tabs';
import {
  ConfidenceMetricsFilter,
  ConfidenceMetricsSection,
} from 'src/components/viewers/MetricsVisualizations';
import CompareTable, { CompareTableProps } from 'src/components/CompareTable';
import {
  compareCss,
  ExecutionArtifact,
  FullArtifactPathMap,
  getScalarTableProps,
  getParamsTableProps,
  getRocCurveId,
  getValidRocCurveArtifactData,
  MetricsType,
  metricsTypeToString,
  RocCurveColorMap,
  RunArtifact,
  RunArtifactData,
} from 'src/lib/v2/CompareUtils';
import { NamespaceContext, useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import MetricsDropdown from 'src/components/viewers/MetricsDropdown';
import CircularProgress from '@material-ui/core/CircularProgress';
import { lineColors } from 'src/components/viewers/ROCCurve';
import Hr from 'src/atoms/Hr';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
  },
  outputsOverflow: {
    overflowX: 'auto',
  },
});

interface MlmdPackage {
  executions: Execution[];
  artifacts: Artifact[];
  events: Event[];
}

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
): RunArtifactData {
  const metricsFilter = metricsTypeToFilter(metricsType);
  const typeRuns: RunArtifact[] = [];
  let artifactCount: number = 0;
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
        artifactCount += typeArtifacts.length;
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
  return {
    runArtifacts: typeRuns,
    artifactCount,
  };
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

export interface SelectedArtifact {
  selectedItem: SelectedItem;
  linkedArtifact?: LinkedArtifact;
}

interface CompareTableSectionParams {
  isLoading?: boolean;
  compareTableProps?: CompareTableProps;
  dataTypeName: string;
}

function CompareTableSection(props: CompareTableSectionParams) {
  const { isLoading, compareTableProps, dataTypeName } = props;

  if (isLoading) {
    return (
      <div className={compareCss.smallRelativeContainer}>
        <CircularProgress
          size={25}
          className={commonCss.absoluteCenter}
          style={{ zIndex: zIndex.BUSY_OVERLAY }}
          role='circularprogress'
        />
      </div>
    );
  }

  if (!compareTableProps) {
    return <p>There are no {dataTypeName} available on the selected runs.</p>;
  }

  return <CompareTable {...compareTableProps} />;
}

interface RocCurveMetricsParams {
  linkedArtifacts: LinkedArtifact[];
  filter: ConfidenceMetricsFilter;
}

function RocCurveMetrics(props: RocCurveMetricsParams) {
  const { linkedArtifacts, filter } = props;

  if (linkedArtifacts.length === 0) {
    return <p>There are no ROC Curve artifacts available on the selected runs.</p>;
  }

  return <ConfidenceMetricsSection linkedArtifacts={linkedArtifacts} filter={filter} />;
}

interface CompareV2Namespace {
  namespace?: string;
}

export type CompareV2Props = PageProps & CompareV2Namespace;

function CompareV2(props: CompareV2Props) {
  const { updateBanner, updateToolbar, namespace } = props;

  const runlistRef = useRef<RunList>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [metricsTab, setMetricsTab] = useState(MetricsType.SCALAR_METRICS);
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);
  const [isLoadingArtifacts, setIsLoadingArtifacts] = useState<boolean>(true);
  const [paramsTableProps, setParamsTableProps] = useState<CompareTableProps | undefined>();
  const [isInitialArtifactsLoad, setIsInitialArtifactsLoad] = useState<boolean>(true);

  // Scalar Metrics
  const [scalarMetricsTableData, setScalarMetricsTableData] = useState<
    CompareTableProps | undefined
  >(undefined);

  // ROC Curve
  const [rocCurveLinkedArtifacts, setRocCurveLinkedArtifacts] = useState<LinkedArtifact[]>([]);
  const [selectedRocCurveIds, setSelectedRocCurveIds] = useState<string[]>([]);
  const [selectedIdColorMap, setSelectedIdColorMap] = useState<RocCurveColorMap>({});
  const [lineColorsStack, setLineColorsStack] = useState<string[]>([...lineColors].reverse());
  const [fullArtifactPathMap, setFullArtifactPathMap] = useState<FullArtifactPathMap>({});

  // Two-panel display artifacts
  const [confusionMatrixRunArtifacts, setConfusionMatrixRunArtifacts] = useState<RunArtifact[]>([]);
  const [htmlRunArtifacts, setHtmlRunArtifacts] = useState<RunArtifact[]>([]);
  const [markdownRunArtifacts, setMarkdownRunArtifacts] = useState<RunArtifact[]>([]);

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
    ['run_artifacts', { runs }],
    () =>
      Promise.all(
        runIds.map(async runId => {
          // TODO(zijianjoy): MLMD query is limited to 100 artifacts per run.
          // https://github.com/google/ml-metadata/blob/5757f09d3b3ae0833078dbfd2d2d1a63208a9821/ml_metadata/proto/metadata_store.proto#L733-L737
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

  // Ensure that the two-panel selected artifacts are present in selected valid run list.
  const getVerifiedTwoPanelSelection = (
    runArtifacts: RunArtifact[],
    selectedArtifacts: SelectedArtifact[],
  ) => {
    const artifactsPresent: boolean[] = new Array(2).fill(false);
    for (const runArtifact of runArtifacts) {
      const runName = runArtifact.run.run?.name;
      if (runName === selectedArtifacts[0].selectedItem.itemName) {
        artifactsPresent[0] = true;
      } else if (runName === selectedArtifacts[1].selectedItem.itemName) {
        artifactsPresent[1] = true;
      }
    }

    for (let i: number = 0; i < artifactsPresent.length; i++) {
      if (!artifactsPresent[i]) {
        selectedArtifacts[i] = {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        };
      }
    }

    return [...selectedArtifacts];
  };

  useEffect(() => {
    if (runs && selectedIds && mlmdPackages && artifactTypes) {
      const selectedIdsSet = new Set(selectedIds);
      const runArtifacts: RunArtifact[] = getRunArtifacts(runs, mlmdPackages).filter(runArtifact =>
        selectedIdsSet.has(runArtifact.run.run!.id!),
      );
      const scalarMetricsArtifactData = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.SCALAR_METRICS,
      );
      setScalarMetricsTableData(
        getScalarTableProps(
          scalarMetricsArtifactData.runArtifacts,
          scalarMetricsArtifactData.artifactCount,
        ),
      );

      // Filter and set the two-panel layout run artifacts.
      const confusionMatrixRunArtifacts: RunArtifact[] = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.CONFUSION_MATRIX,
      ).runArtifacts;
      const htmlRunArtifacts: RunArtifact[] = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.HTML,
      ).runArtifacts;
      const markdownRunArtifacts: RunArtifact[] = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.MARKDOWN,
      ).runArtifacts;
      setConfusionMatrixRunArtifacts(confusionMatrixRunArtifacts);
      setHtmlRunArtifacts(htmlRunArtifacts);
      setMarkdownRunArtifacts(markdownRunArtifacts);

      // Iterate through selected runs, remove current selection if not present among runs.
      setSelectedArtifactsMap({
        [MetricsType.CONFUSION_MATRIX]: getVerifiedTwoPanelSelection(
          confusionMatrixRunArtifacts,
          selectedArtifactsMap[MetricsType.CONFUSION_MATRIX],
        ),
        [MetricsType.HTML]: getVerifiedTwoPanelSelection(
          htmlRunArtifacts,
          selectedArtifactsMap[MetricsType.HTML],
        ),
        [MetricsType.MARKDOWN]: getVerifiedTwoPanelSelection(
          markdownRunArtifacts,
          selectedArtifactsMap[MetricsType.MARKDOWN],
        ),
      });

      // Set ROC Curve run artifacts and get all valid ROC curve data for plot visualization.
      const rocCurveRunArtifacts: RunArtifact[] = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.ROC_CURVE,
      ).runArtifacts;
      updateRocCurveDisplay(rocCurveRunArtifacts);
      setIsLoadingArtifacts(false);
      setIsInitialArtifactsLoad(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [runs, selectedIds, mlmdPackages, artifactTypes]);

  // Update the ROC Curve colors and selection.
  const updateRocCurveDisplay = (runArtifacts: RunArtifact[]) => {
    const {
      validLinkedArtifacts,
      fullArtifactPathMap,
      validRocCurveIdSet,
    } = getValidRocCurveArtifactData(runArtifacts);

    setFullArtifactPathMap(fullArtifactPathMap);
    setRocCurveLinkedArtifacts(validLinkedArtifacts);

    // Remove all newly invalid ROC Curves from the selection (if run selection changes).
    const removedRocCurveIds: Set<string> = new Set();
    for (const oldSelectedId of Object.keys(selectedIdColorMap)) {
      if (!validRocCurveIdSet.has(oldSelectedId)) {
        removedRocCurveIds.add(oldSelectedId);
      }
    }

    // If initial load, choose first three artifacts; ow, remove artifacts from de-selected runs.
    let updatedRocCurveIds: string[] = selectedRocCurveIds;
    if (isInitialArtifactsLoad) {
      updatedRocCurveIds = validLinkedArtifacts
        .map(linkedArtifact => getRocCurveId(linkedArtifact))
        .slice(0, 3);
      updatedRocCurveIds.forEach(rocCurveId => {
        selectedIdColorMap[rocCurveId] = lineColorsStack.pop()!;
      });
    } else {
      updatedRocCurveIds = updatedRocCurveIds.filter(rocCurveId => {
        if (removedRocCurveIds.has(rocCurveId)) {
          lineColorsStack.push(selectedIdColorMap[rocCurveId]);
          delete selectedIdColorMap[rocCurveId];
          return false;
        }
        return true;
      });
    }
    setSelectedRocCurveIds(updatedRocCurveIds);
    setLineColorsStack(lineColorsStack);
    setSelectedIdColorMap(selectedIdColorMap);
  };

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

  useEffect(() => {
    if (runs) {
      const selectedIdsSet = new Set(selectedIds);
      const selectedRuns: ApiRunDetail[] = runs.filter(run => selectedIdsSet.has(run.run!.id!));
      setParamsTableProps(getParamsTableProps(selectedRuns));
    } else {
      setParamsTableProps(undefined);
    }
  }, [runs, selectedIds]);

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

  const isErrorArtifacts = isErrorRunDetails || isErrorMlmdPackages || isErrorArtifactTypes;
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
          <CompareTableSection
            isLoading={isLoadingRunDetails}
            compareTableProps={paramsTableProps}
            dataTypeName='Parameters'
          />
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
            {isErrorArtifacts ? (
              <p>An error is preventing the {metricsTabText} from being displayed.</p>
            ) : isLoadingArtifacts ? (
              <div className={compareCss.relativeContainer}>
                <CircularProgress
                  size={25}
                  className={commonCss.absoluteCenter}
                  style={{ zIndex: zIndex.BUSY_OVERLAY }}
                  role='circularprogress'
                />
              </div>
            ) : (
              <>
                {metricsTab === MetricsType.SCALAR_METRICS && (
                  <CompareTableSection
                    compareTableProps={scalarMetricsTableData}
                    dataTypeName='Scalar Metrics artifacts'
                  />
                )}
                {metricsTab === MetricsType.CONFUSION_MATRIX && (
                  <MetricsDropdown
                    filteredRunArtifacts={confusionMatrixRunArtifacts}
                    metricsTab={metricsTab}
                    selectedArtifacts={selectedArtifactsMap[metricsTab]}
                    updateSelectedArtifacts={updateSelectedArtifacts}
                    namespace={namespace}
                  />
                )}
                {metricsTab === MetricsType.ROC_CURVE && (
                  <RocCurveMetrics
                    linkedArtifacts={rocCurveLinkedArtifacts}
                    filter={{
                      selectedIds: selectedRocCurveIds,
                      setSelectedIds: setSelectedRocCurveIds,
                      fullArtifactPathMap,
                      selectedIdColorMap,
                      setSelectedIdColorMap,
                      lineColorsStack,
                      setLineColorsStack,
                    }}
                  />
                )}
                {metricsTab === MetricsType.HTML && (
                  <MetricsDropdown
                    filteredRunArtifacts={htmlRunArtifacts}
                    metricsTab={metricsTab}
                    selectedArtifacts={selectedArtifactsMap[metricsTab]}
                    updateSelectedArtifacts={updateSelectedArtifacts}
                    namespace={namespace}
                  />
                )}
                {metricsTab === MetricsType.MARKDOWN && (
                  <MetricsDropdown
                    filteredRunArtifacts={markdownRunArtifacts}
                    metricsTab={metricsTab}
                    selectedArtifacts={selectedArtifactsMap[metricsTab]}
                    updateSelectedArtifacts={updateSelectedArtifacts}
                    namespace={namespace}
                  />
                )}
              </>
            )}
          </div>
        </div>
      )}

      <Separator orientation='vertical' />
    </div>
  );
}

function EnhancedCompareV2(props: PageProps) {
  const namespace: string | undefined = useContext(NamespaceContext);
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    // Run Comparison page compares multiple runs, when namespace changes, the runs don't
    // exist in the new namespace, so we should redirect to experiment list page.
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }

  return <CompareV2 namespace={namespace} {...props} />;
}

export default EnhancedCompareV2;

export const TEST_ONLY = {
  CompareV2,
};
