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
import Hr from 'src/atoms/Hr';
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
  getCompareTableProps,
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

interface ScalarMetricsTableParams {
  scalarMetricsTableData: CompareTableProps | undefined;
}

function ScalarMetricsTable(props: ScalarMetricsTableParams) {
  const { scalarMetricsTableData } = props;

  if (!scalarMetricsTableData) {
    return <p>There are no Scalar Metrics artifacts available on the selected runs.</p>;
  }

  return <CompareTable {...scalarMetricsTableData} />;
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

  // Two-panel display artifacts
  const [confusionMatrixRunArtifacts, setConfusionMatrixRunArtifacts] = useState<RunArtifact[]>([]);
  const [htmlRunArtifacts, setHtmlRunArtifacts] = useState<RunArtifact[]>([]);
  const [markdownRunArtifacts, setMarkdownRunArtifacts] = useState<RunArtifact[]>([]);

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
          // TODO(zpChris): MLMD query is limited to 100 artifacts per run.
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

  // TODO(zpChris): Make clear this is also called on re-render, and for roc curves state must be saved.
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
      const compareTableProps: CompareTableProps = getCompareTableProps(
        scalarMetricsArtifactData.runArtifacts,
        scalarMetricsArtifactData.artifactCount,
      );
      if (compareTableProps.yLabels.length === 0) {
        setScalarMetricsTableData(undefined);
      } else {
        setScalarMetricsTableData(compareTableProps);
      }

      setConfusionMatrixRunArtifacts(
        filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.CONFUSION_MATRIX)
          .runArtifacts,
      );
      setHtmlRunArtifacts(
        filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.HTML).runArtifacts,
      );
      setMarkdownRunArtifacts(
        filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.MARKDOWN).runArtifacts,
      );

      const rocCurveRunArtifacts: RunArtifact[] = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.ROC_CURVE,
      ).runArtifacts;

      const { validLinkedArtifacts, fullArtifactPathMap, updatedIdColorMap } = getValidRocCurveArtifactData(
        rocCurveRunArtifacts,
        selectedIdColorMap,
        lineColorsStack,
      );

      setFullArtifactPathMap(fullArtifactPathMap);
      setRocCurveLinkedArtifacts(validLinkedArtifacts);

      // I could clear the map to refresh the colors completely.
      // I could get all of the Object.keys(initialIdColorMap), and then look through each one. For each one
      // I would look through all of the selected run artifacts linked artifacts, and see if it matches? If not,
      // then find that corresponding color, add it back onto the stack, and delete that entry. Ooof - maybe I can do this while looping through the getValidRocCurveArtifactData?
      // Such as get the new initialIdColorMap?
      // What if I also stored the run ID on this color map? Then on update, I could check which runs correspond to the colors.
      // This requires work on the updated IDs side of MetricsVisualizations, so I'll stick to the first solution.
      console.log(Object.keys(selectedIdColorMap));

      // So the colors do indeed update.

      const updatedRocCurveIds = validLinkedArtifacts.map(linkedArtifact => getRocCurveId(linkedArtifact)).slice(0, 3);
      setSelectedRocCurveIds(updatedRocCurveIds);

      // Populate the color map on the initial render.

      // Ok, so I don't have to base it off of the initial stack. I can base it off what I know that value to be.
      // However, how do I modify this when it *does* have to be based off the initial value?
      // What we have to do then is find all of the runs whose artifacts are being used; then,
      // pop those colors off the stack.
      // Or, I can not include the lineColorsStack in this list?
      // Note: deselecting and re-selecting a run will not maintain those selections. I personally am OK with that, knowing the complexity that adds.
      // const initialIdColorMap: { [key: string]: string } = {};
      // updatedRocCurveIds.forEach(selectedId => {
      //   initialIdColorMap[selectedId] = lineColorsStack.pop()!;
      // });
      setLineColorsStack(lineColorsStack);
      setSelectedIdColorMap(updatedIdColorMap);
      setIsLoadingArtifacts(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [runs, selectedIds, mlmdPackages, artifactTypes]);

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

  const selectionChanged = (newSelectedIds: string[]): void => {
    console.log(selectedIds);
    console.log(newSelectedIds);
    // Get all of the removed ids. (None of the ones that are added will update the plot.)
    // From that, we find all of the runs where 
    setSelectedIds(newSelectedIds);
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
                  <ScalarMetricsTable scalarMetricsTableData={scalarMetricsTableData} />
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
