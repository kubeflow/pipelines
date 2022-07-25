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
import { commonCss, padding } from 'src/Css';
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
import { ConfidenceMetricsSection } from 'src/components/viewers/MetricsVisualizations';
import CompareTable, { CompareTableProps } from 'src/components/CompareTable';
import {
  ExecutionArtifact,
  FullArtifactPath,
  getCompareTableProps,
  getRocCurveId,
  getValidRocCurveLinkedArtifacts,
  MetricsType,
  RunArtifact,
  RunArtifactData,
} from 'src/lib/v2/CompareUtils';
import { NamespaceContext, useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import MetricsDropdown from 'src/components/viewers/MetricsDropdown';

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

interface CompareV2Namespace {
  namespace?: string;
}

export type CompareV2Props = PageProps & CompareV2Namespace;

function CompareV2(props: CompareV2Props) {
  const { updateBanner, updateToolbar, namespace } = props;

  const runlistRef = useRef<RunList>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [metricsTab, setMetricsTab] = useState(MetricsType.ROC_CURVE); // TODO(zpChris): Testing; change back to Scalar Metrics.
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);

  const [confusionMatrixRunArtifacts, setConfusionMatrixRunArtifacts] = useState<RunArtifact[]>([]);
  const [htmlRunArtifacts, setHtmlRunArtifacts] = useState<RunArtifact[]>([]);
  const [markdownRunArtifacts, setMarkdownRunArtifacts] = useState<RunArtifact[]>([]);

  const [rocCurveLinkedArtifacts, setRocCurveLinkedArtifacts] = useState<LinkedArtifact[]>([]);
  const [selectedRocCurveIds, setSelectedRocCurveIds] = useState<string[]>([]);

  const [fullArtifactPathMap, setFullArtifactPathMap] = useState<{
    [key: string]: FullArtifactPath;
  }>({});

  const [scalarMetricsArtifacts, setScalarMetricsArtifacts] = useState<RunArtifact[]>([]);
  const [scalarMetricsArtifactCount, setScalarMetricsArtifactCount] = useState<number>(0);
  const [scalarMetricsTableData, setScalarMetricsTableData] = useState<
    CompareTableProps | undefined
  >(undefined);
  const [selectedIdColorMap, setSelectedIdColorMap] = useState<{ [key: string]: string }>({});

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
      const scalarMetricsArtifactData = filterRunArtifactsByType(
        runArtifacts,
        artifactTypes,
        MetricsType.SCALAR_METRICS,
      );
      setScalarMetricsArtifacts(scalarMetricsArtifactData.runArtifacts);
      setScalarMetricsArtifactCount(scalarMetricsArtifactData.artifactCount);
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

      const { validLinkedArtifacts, fullArtifactPathMap } = getValidRocCurveLinkedArtifacts(
        rocCurveRunArtifacts,
      );

      setFullArtifactPathMap(fullArtifactPathMap);
      setRocCurveLinkedArtifacts(validLinkedArtifacts);
      setSelectedRocCurveIds(
        validLinkedArtifacts.map(linkedArtifact => getRocCurveId(linkedArtifact)).slice(0, 3),
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

  useEffect(() => {
    const compareTableProps: CompareTableProps = getCompareTableProps(
      scalarMetricsArtifacts,
      scalarMetricsArtifactCount,
    );
    if (compareTableProps.yLabels.length === 0) {
      setScalarMetricsTableData(undefined);
    } else {
      setScalarMetricsTableData(compareTableProps);
    }
  }, [scalarMetricsArtifacts, scalarMetricsArtifactCount]);

  const updateSelectedArtifacts = (newArtifacts: SelectedArtifact[]) => {
    selectedArtifactsMap[metricsTab] = newArtifacts;
    setSelectedArtifactsMap(selectedArtifactsMap);
  };

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
            {metricsTab === MetricsType.SCALAR_METRICS &&
              (scalarMetricsTableData ? (
                <CompareTable {...scalarMetricsTableData} />
              ) : (
                <p>There are no Scalar Metrics artifacts available on the selected runs.</p>
              ))}
            {metricsTab === MetricsType.CONFUSION_MATRIX && (
              <MetricsDropdown
                filteredRunArtifacts={confusionMatrixRunArtifacts}
                metricsTab={metricsTab}
                selectedArtifacts={selectedArtifactsMap[metricsTab]}
                updateSelectedArtifacts={updateSelectedArtifacts}
                namespace={namespace}
              />
            )}
            {metricsTab === MetricsType.ROC_CURVE &&
              (rocCurveLinkedArtifacts.length > 0 ? (
                <ConfidenceMetricsSection
                  linkedArtifacts={rocCurveLinkedArtifacts}
                  filter={{
                    selectedIds: selectedRocCurveIds,
                    setSelectedIds: setSelectedRocCurveIds,
                    fullArtifactPathMap,
                    selectedIdColorMap,
                    setSelectedIdColorMap,
                  }}
                />
              ) : (
                <p>There are no ROC Curve artifacts available on the selected runs.</p>
              ))}
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
