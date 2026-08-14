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
import { useQueries, useQueryClient, type UseQueryResult } from '@tanstack/react-query';
import {
  useCallback,
  useContext,
  useEffect,
  useEffectEvent,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { Redirect } from 'react-router-dom';
import { V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
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
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import {
  flattenArtifactGroups,
  formatParameterValue,
  getArtifactDisplayName,
  getScalarMetricEntries,
  isScalarMetricArtifact,
  isTaskFinished,
  type RuntimeArtifactEntry,
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
export const ACTIVE_COMPARISON_REFRESH_INTERVAL = 10_000;
const ACTIVE_COMPARISON_STALE_TIME = ACTIVE_COMPARISON_REFRESH_INTERVAL;
const TERMINAL_COMPARISON_STALE_TIME = 60_000;

interface RunComparisonData {
  run: V2beta1Run;
  runError?: Error;
  taskError?: Error;
  tasks: V2beta1PipelineTask[];
  terminalTaskReconciliationPending?: boolean;
}

interface RunComparisonFailure {
  runId: string;
  source: 'run' | 'tasks';
  error: Error;
}

type RunComparisonQueryResult = Pick<
  UseQueryResult<RunComparisonData, Error>,
  'data' | 'error' | 'isPending' | 'refetch'
>;

interface RunArtifactEntry extends RuntimeArtifactEntry {
  sourceFinished: boolean;
  taskKey: string;
  taskName: string;
}

interface RunScalarMetricEntry extends RunArtifactEntry {
  metricName: string;
  metricValue: string;
}

export type CompareV2Props = PageProps & { namespace?: string };

async function loadRunComparisonData(
  runId: string,
  previousData?: RunComparisonData,
): Promise<RunComparisonData> {
  const [runResult, tasksResult] = await Promise.allSettled([
    Apis.runServiceApiV2.getRun(runId),
    listAllRunTasks(runId),
  ]);
  let run: V2beta1Run;
  let runError: Error | undefined;
  if (runResult.status === 'rejected') {
    const cachedRun = previousData?.run;
    if (
      cachedRun?.state === undefined ||
      !hasFinishedV2(cachedRun.state) ||
      previousData?.terminalTaskReconciliationPending !== true
    ) {
      throw toError(runResult.reason);
    }
    // The cached terminal state is sufficient for the single bounded reconciliation. Do not use
    // this fallback for later manual refreshes, where the run may have been retried and active.
    run = cachedRun;
    runError = toError(runResult.reason);
  } else {
    run = runResult.value;
  }
  const tasks = tasksResult.status === 'fulfilled' ? tasksResult.value : previousData?.tasks || [];
  const taskError = tasksResult.status === 'rejected' ? toError(tasksResult.reason) : undefined;
  const runIsTerminal = run.state !== undefined && hasFinishedV2(run.state);
  const previousState = previousData?.run.state;
  const previousRunWasTerminal = previousState !== undefined && hasFinishedV2(previousState);
  const taskSnapshotIsIncomplete =
    taskError !== undefined || tasks.some((task) => !isTaskFinished(task.state));
  return {
    run,
    runError,
    tasks,
    taskError,
    // Fail-fast runs can retain non-terminal sibling task rows permanently. Reconcile once when
    // terminal run data first arrives incomplete, then let the run state remain authoritative.
    terminalTaskReconciliationPending:
      runIsTerminal && !previousRunWasTerminal && taskSnapshotIsIncomplete,
  };
}

function toError(value: unknown): Error {
  return value instanceof Error ? value : new Error(String(value));
}

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
  const queryClient = useQueryClient();
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = useMemo(
    () => (queryParamRunIds ? queryParamRunIds.split(',').filter(Boolean) : []),
    [queryParamRunIds],
  );
  const runIdsKey = runIds.join(',');
  const [selectedIds, setSelectedIds] = useKeyedState<string[]>(runIdsKey, runIds);
  const [metricsTab, setMetricsTab] = useState(NativeMetricsTab.SCALAR);
  const [artifactComparisonSelection, setArtifactComparisonSelection] = useState(
    createRuntimeArtifactComparisonSelectionState,
  );
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);

  const comparisonQueryOptions = useMemo(
    () =>
      runIds.map((runId) => ({
        queryKey: queryKeys.v2RunComparison(runId),
        queryFn: () =>
          loadRunComparisonData(
            runId,
            queryClient.getQueryData<RunComparisonData>(queryKeys.v2RunComparison(runId)),
          ),
        retry: false,
        staleTime: (query: { state: { data?: RunComparisonData } }) => {
          const state = query.state.data?.run.state;
          return state !== undefined && hasFinishedV2(state)
            ? TERMINAL_COMPARISON_STALE_TIME
            : ACTIVE_COMPARISON_STALE_TIME;
        },
        refetchInterval: (query: { state: { data?: RunComparisonData } }) => {
          const data = query.state.data;
          const state = data?.run.state;
          const runIsActive = state !== undefined && !hasFinishedV2(state);
          const reconciliationIsPending = data?.terminalTaskReconciliationPending === true;
          return runIsActive || reconciliationIsPending
            ? ACTIVE_COMPARISON_REFRESH_INTERVAL
            : false;
        },
      })),
    [queryClient, runIds],
  );
  const combineComparisonQueries = useCallback(
    (results: RunComparisonQueryResult[]) => {
      const comparisonData: RunComparisonData[] = [];
      const failures: RunComparisonFailure[] = [];
      results.forEach((result, index) => {
        if (result.data) {
          comparisonData.push(result.data);
          if (result.data.runError) {
            failures.push({ runId: runIds[index], source: 'run', error: result.data.runError });
          }
          if (result.data.taskError) {
            failures.push({ runId: runIds[index], source: 'tasks', error: result.data.taskError });
          }
        }
        if (result.error) {
          failures.push({ runId: runIds[index], source: 'run', error: result.error });
        }
      });
      return {
        comparisonData,
        failures,
        isLoading: results.some((result) => result.isPending),
        refetch: () => Promise.all(results.map((result) => result.refetch())),
      };
    },
    [runIds],
  );
  const { comparisonData, failures, isLoading, refetch } = useQueries({
    queries: comparisonQueryOptions,
    combine: combineComparisonQueries,
  });

  const selectedData = useMemo(() => {
    const selectedIdSet = new Set(selectedIds);
    return comparisonData.filter(({ run }) => selectedIdSet.has(run.run_id || ''));
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
    if (failures.length) {
      const failedRunLabel = failures.length === 1 ? 'run' : 'runs';
      updateBanner({
        additionalInfo: failures
          .map(
            ({ runId, source, error }) =>
              `${runId}${source === 'tasks' ? ' tasks' : ''}: ${error.message}`,
          )
          .join('\n'),
        message: comparisonData.length
          ? `Cannot get comparison data for ${failures.length} selected ${failedRunLabel}. Available runs are still shown. Refresh the page to try again.`
          : 'Cannot get comparison data for the selected runs. Refresh the page to try again.',
        mode: comparisonData.length ? 'warning' : 'error',
      });
    } else {
      updateBanner({});
    }
  }, [comparisonData.length, failures, isLoading, updateBanner]);

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
            {metricsTab === NativeMetricsTab.SCALAR ? (
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
  return tasks.flatMap((task, taskIndex) =>
    flattenArtifactGroups(task.outputs?.artifacts).map((entry) => ({
      ...entry,
      sourceFinished: isTaskFinished(task.state),
      taskKey: task.task_id || task.name || String(taskIndex),
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
  return comparisonData.flatMap(({ run, tasks, terminalTaskReconciliationPending }) => {
    const runLabel = run.display_name || run.run_id || 'Run';
    // After the bounded reconciliation, a terminal run is the authoritative signal that artifact
    // production has stopped even when fail-fast leaves a sibling task row marked RUNNING.
    const terminalRunFinishedSources =
      run.state !== undefined &&
      hasFinishedV2(run.state) &&
      terminalTaskReconciliationPending !== true;
    return collectOutputArtifacts(tasks).map(
      ({ artifact, artifactKey, group, index, sourceFinished, taskKey, taskName }) => ({
        artifact,
        key: [
          run.run_id || runLabel,
          taskKey,
          artifactKey,
          index,
          artifact.artifact_id || artifact.uri || artifact.name || 'artifact',
        ].join(':'),
        label: `${runLabel} / ${taskName} / ${getArtifactDisplayName(
          artifact,
          artifactKey,
          index,
          group.artifacts,
        )}`,
        namespace: artifact.namespace || defaultNamespace,
        sourceFinished: sourceFinished || terminalRunFinishedSources,
      }),
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
  const scalarMetricsByRun = comparisonData.map(({ tasks }) =>
    collectOutputArtifacts(tasks)
      .filter(({ artifact }) => isScalarMetricArtifact(artifact))
      .flatMap((entry) =>
        getScalarMetricEntries(entry.artifact).map(({ name, value }) => ({
          ...entry,
          metricName: name,
          metricValue: value,
        })),
      ),
  );
  const labelsNeedingArtifactKey = new Set<string>();
  const artifactKeysByLabel = new Map<string, Set<string>>();
  scalarMetricsByRun.forEach((entries) => {
    entries.forEach((entry) => {
      const label = getScalarMetricBaseLabel(entry);
      const artifactKeys = artifactKeysByLabel.get(label) || new Set<string>();
      artifactKeys.add(entry.artifactKey || '');
      artifactKeysByLabel.set(label, artifactKeys);
    });
  });
  artifactKeysByLabel.forEach((artifactKeys, label) => {
    if (artifactKeys.size > 1) {
      labelsNeedingArtifactKey.add(label);
    }
  });

  const metricsByRun = scalarMetricsByRun.map((entries) => {
    const metrics = new Map<string, string>();
    const labelOccurrences = new Map<string, number>();
    entries.forEach((entry) => {
      const { artifactKey, metricName, metricValue, taskName } = entry;
      const baseLabel = getScalarMetricBaseLabel(entry);
      const disambiguatedLabel =
        labelsNeedingArtifactKey.has(baseLabel) && artifactKey
          ? `${taskName} / ${artifactKey} / ${metricName}`
          : baseLabel;
      const occurrence = (labelOccurrences.get(disambiguatedLabel) || 0) + 1;
      labelOccurrences.set(disambiguatedLabel, occurrence);
      const label = occurrence === 1 ? disambiguatedLabel : `${disambiguatedLabel} (${occurrence})`;
      metrics.set(label, metricValue);
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

function getScalarMetricBaseLabel(entry: RunScalarMetricEntry): string {
  return `${entry.taskName} / ${entry.metricName}`;
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
