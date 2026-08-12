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

import { CircularProgress } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { useContext, useEffect, useEffectEvent, useMemo, useRef, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { Redirect } from 'react-router-dom';
import { V2beta1Artifact, V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import Separator from 'src/atoms/Separator';
import CollapseButtonSingle from 'src/components/CollapseButtonSingle';
import CompareTable, { CompareTableProps } from 'src/components/CompareTable';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import {
  createRuntimeArtifactComparisonSelectionState,
  RuntimeArtifactComparison,
  RuntimeArtifactComparisonKind,
  RuntimeArtifactComparisonSelectionState,
  RuntimeComparisonArtifact,
} from 'src/components/viewers/RuntimeArtifactComparison';
import { commonCss, padding, zIndex } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { useKeyedState } from 'src/hooks/useKeyedState';
import { Apis } from 'src/lib/Apis';
import Buttons from 'src/lib/Buttons';
import { NamespaceContext, useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage } from 'src/lib/Utils';
import {
  flattenArtifactGroups,
  formatParameterValue,
  getArtifactDisplayName,
  getScalarMetricValue,
  isScalarMetricArtifact,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { listAllRunTasks } from 'src/lib/v2/RunTaskUtils';
import { classes, stylesheet } from 'typestyle';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { PageProps } from './Page';
import RunList from './RunList';

const css = stylesheet({
  outputsRow: { marginLeft: 15 },
  outputsOverflow: { overflowX: 'auto' },
  relativeContainer: { height: '12rem', position: 'relative' },
});

export enum NativeMetricsTab {
  SCALAR,
  CLASSIFICATION,
  HTML,
  MARKDOWN,
}

const METRICS_TAB_NAMES = ['Scalar Metrics', 'Classification Metrics', 'HTML', 'Markdown'];

interface RunComparisonData {
  run: V2beta1Run;
  tasks: V2beta1PipelineTask[];
}

interface RunArtifactEntry {
  artifact: V2beta1Artifact;
  artifactKey: string;
  taskName: string;
}

export type CompareV2Props = PageProps & { namespace?: string };

function CompareTableSection({
  isLoading,
  compareTableProps,
  dataTypeName,
}: {
  isLoading?: boolean;
  compareTableProps?: CompareTableProps;
  dataTypeName: string;
}) {
  if (isLoading) {
    return (
      <div className={css.relativeContainer}>
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

export function CompareV2(props: CompareV2Props) {
  const { updateBanner, updateToolbar, namespace } = props;
  const runlistRef = useRef<RunList>(null);
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = useMemo(
    () => (queryParamRunIds ? queryParamRunIds.split(',').filter(Boolean) : []),
    [queryParamRunIds],
  );
  const runIdsKey = runIds.join(',');
  const [selectedIdsState, setSelectedIds] = useKeyedState<string[]>(runIdsKey, runIds);
  const [metricsTab, setMetricsTab] = useState(NativeMetricsTab.SCALAR);
  const [artifactComparisonSelection, setArtifactComparisonSelection] = useState(
    createRuntimeArtifactComparisonSelectionState,
  );
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);

  const {
    data: comparisonData,
    error,
    isError,
    isLoading,
    refetch,
  } = useQuery<RunComparisonData[], Error>({
    queryKey: queryKeys.v2RunComparison(runIds),
    queryFn: () =>
      Promise.all(
        runIds.map(async (runId) => {
          const [run, tasks] = await Promise.all([
            Apis.runServiceApiV2.getRun(runId),
            listAllRunTasks(runId),
          ]);
          return { run, tasks };
        }),
      ),
    staleTime: Infinity,
  });

  const selectedIds = useMemo(() => {
    if (!comparisonData) {
      return selectedIdsState;
    }
    const validRunIds = new Set(
      comparisonData.map(({ run }) => run.run_id).filter((id): id is string => !!id),
    );
    return selectedIdsState.filter((id) => validRunIds.has(id));
  }, [comparisonData, selectedIdsState]);

  const selectedData = useMemo(() => {
    const selectedIdSet = new Set(selectedIds);
    return (comparisonData || []).filter(({ run }) => selectedIdSet.has(run.run_id || ''));
  }, [comparisonData, selectedIds]);

  const paramsTableProps = useMemo(() => buildParamsTableProps(selectedData), [selectedData]);
  const scalarMetricsTableProps = useMemo(
    () => buildScalarMetricsTableProps(selectedData),
    [selectedData],
  );

  useEffect(() => {
    if (isLoading) {
      return;
    }
    if (isError) {
      updateBanner({
        additionalInfo: error?.message,
        message:
          'Cannot get native task and artifact data for the selected runs. Refresh the page to try again.',
        mode: 'error',
      });
    } else {
      updateBanner({});
    }
  }, [error, isError, isLoading, updateBanner]);

  const updateComparisonToolbar = useEffectEvent(() => {
    const refresh = async () => {
      await Promise.all([runlistRef.current?.refresh(), refetch()]);
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
  });

  useEffect(() => {
    updateComparisonToolbar();
  }, []);

  const showPageError = async (message: string, requestError: Error | undefined) => {
    const errorMessage = await errorToMessage(requestError);
    updateBanner({
      additionalInfo: errorMessage || undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
    });
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lrt'))}>
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
            onSelectionChange={setSelectedIds}
          />
        </div>
      )}

      <Separator orientation='vertical' />

      <CollapseButtonSingle
        sectionName={PARAMS_SECTION_NAME}
        collapseSection={isParamsCollapsed}
        collapseSectionUpdate={setIsParamsCollapsed}
      />
      {!isParamsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow, css.outputsOverflow)}>
          <Separator orientation='vertical' />
          <CompareTableSection
            isLoading={isLoading}
            compareTableProps={paramsTableProps}
            dataTypeName='parameters'
          />
        </div>
      )}

      <CollapseButtonSingle
        sectionName={METRICS_SECTION_NAME}
        collapseSection={isMetricsCollapsed}
        collapseSectionUpdate={setIsMetricsCollapsed}
      />
      {!isMetricsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <MD2Tabs tabs={METRICS_TAB_NAMES} selectedTab={metricsTab} onSwitch={setMetricsTab} />
          <div className={classes(padding(20, 'lrt'), css.outputsOverflow)}>
            {isError ? (
              <p>An error is preventing metrics from being displayed.</p>
            ) : metricsTab === NativeMetricsTab.SCALAR ? (
              <CompareTableSection
                isLoading={isLoading}
                compareTableProps={scalarMetricsTableProps}
                dataTypeName='scalar metrics artifacts'
              />
            ) : isLoading ? (
              <CompareTableSection isLoading={true} dataTypeName='artifacts' />
            ) : (
              <NativeArtifactComparison
                comparisonData={selectedData}
                metricsTab={metricsTab}
                namespace={namespace}
                selectionState={artifactComparisonSelection}
                setSelectionState={setArtifactComparisonSelection}
              />
            )}
          </div>
        </div>
      )}

      <Separator orientation='vertical' />
    </div>
  );
}

function NativeArtifactComparison({
  comparisonData,
  metricsTab,
  namespace,
  selectionState,
  setSelectionState,
}: {
  comparisonData: RunComparisonData[];
  metricsTab: NativeMetricsTab;
  namespace?: string;
  selectionState: RuntimeArtifactComparisonSelectionState;
  setSelectionState: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
}) {
  const artifacts = useMemo(
    () => collectRuntimeComparisonArtifacts(comparisonData, namespace),
    [comparisonData, namespace],
  );
  const kind: RuntimeArtifactComparisonKind =
    metricsTab === NativeMetricsTab.CLASSIFICATION
      ? 'classification'
      : metricsTab === NativeMetricsTab.HTML
        ? 'html'
        : 'markdown';
  return (
    <RuntimeArtifactComparison
      artifacts={artifacts}
      kind={kind}
      selectionState={selectionState}
      setSelectionState={setSelectionState}
    />
  );
}

function collectOutputArtifacts(tasks: V2beta1PipelineTask[]): RunArtifactEntry[] {
  return tasks.flatMap((task) =>
    flattenArtifactGroups(task.outputs?.artifacts).map(({ artifact, artifactKey }) => ({
      artifact,
      artifactKey,
      taskName: getTaskComparisonLabel(task),
    })),
  );
}

function getTaskComparisonLabel(task: V2beta1PipelineTask): string {
  const scope = task.scope_path?.replace(/^root\.?/, '');
  const baseLabel = scope || task.display_name || task.name || 'Task';
  return task.type_attributes?.iteration_index === undefined
    ? baseLabel
    : `${baseLabel} [iteration ${task.type_attributes.iteration_index}]`;
}

export function collectRuntimeComparisonArtifacts(
  comparisonData: RunComparisonData[],
  defaultNamespace?: string,
): RuntimeComparisonArtifact[] {
  return comparisonData.flatMap(({ run, tasks }) => {
    const runLabel = run.display_name || run.run_id || 'Run';
    return tasks.flatMap((task, taskIndex) =>
      flattenArtifactGroups(task.outputs?.artifacts).map(
        ({ artifact, artifactKey, group, index }) => ({
          artifact,
          key: [
            run.run_id || runLabel,
            task.task_id || task.name || taskIndex,
            artifactKey,
            index,
            artifact.artifact_id || artifact.uri || artifact.name || 'artifact',
          ].join(':'),
          label: `${runLabel} / ${getTaskComparisonLabel(task)} / ${getArtifactDisplayName(
            artifact,
            artifactKey,
            index,
            group.artifacts,
          )}`,
          namespace: artifact.namespace || defaultNamespace,
        }),
      ),
    );
  });
}

export function buildParamsTableProps(
  comparisonData: RunComparisonData[],
): CompareTableProps | undefined {
  const parameterNames = new Set<string>();
  comparisonData.forEach(({ run }) =>
    Object.keys(run.runtime_config?.parameters || {}).forEach((name) => parameterNames.add(name)),
  );
  if (!comparisonData.length || !parameterNames.size) {
    return undefined;
  }
  const yLabels = [...parameterNames];
  return {
    xLabels: comparisonData.map(({ run }) => run.display_name || run.run_id || 'Run'),
    yLabels,
    rows: yLabels.map((parameterName) =>
      comparisonData.map(({ run }) => {
        const value = run.runtime_config?.parameters?.[parameterName];
        return value === undefined ? '' : formatParameterValue(value);
      }),
    ),
  };
}

export function buildScalarMetricsTableProps(
  comparisonData: RunComparisonData[],
): CompareTableProps | undefined {
  const metricsByRun = comparisonData.map(({ tasks }) => {
    const metrics = new Map<string, string>();
    collectOutputArtifacts(tasks)
      .filter(({ artifact }) => isScalarMetricArtifact(artifact))
      .forEach(({ artifact, artifactKey, taskName }) => {
        const label = `${taskName} / ${artifact.name || artifactKey || 'Metric'}`;
        metrics.set(label, getScalarMetricValue(artifact));
      });
    return metrics;
  });
  const metricNames = new Set(metricsByRun.flatMap((metrics) => [...metrics.keys()]));
  if (!comparisonData.length || !metricNames.size) {
    return undefined;
  }
  const yLabels = [...metricNames];
  return {
    xLabels: comparisonData.map(({ run }) => run.display_name || run.run_id || 'Run'),
    yLabels,
    rows: yLabels.map((metricName) => metricsByRun.map((metrics) => metrics.get(metricName) || '')),
  };
}

function EnhancedCompareV2(props: PageProps) {
  const namespace = useContext(NamespaceContext);
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }
  return <CompareV2 namespace={namespace} {...props} />;
}

export default EnhancedCompareV2;
