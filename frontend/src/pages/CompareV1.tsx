/*
 * Copyright 2018 The Kubeflow Authors
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

import * as React from 'react';
import { Navigate } from 'react-router-dom';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { classes, stylesheet } from 'typestyle';
import { Workflow } from '../third_party/mlmd/argo_template';
import { ApiRunDetail } from '../apis/run';
import Hr from '../atoms/Hr';
import Separator from '../atoms/Separator';
import CollapseButton from '../components/CollapseButton';
import CompareTable, { CompareTableProps } from '../components/CompareTable';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { componentMap } from '../components/viewers/ViewerContainer';
import { commonCss, padding } from '../Css';
import { Apis } from '../lib/Apis';
import Buttons from '../lib/Buttons';
import CompareUtils from '../lib/CompareUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { useURLParser } from '../lib/URLParser';
import { ensureError, logger } from '../lib/Utils';
import WorkflowParser from '../lib/WorkflowParser';
import { PageProps } from './Page';
import RunList from './RunList';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

type TaggedViewerConfig = ViewerConfig & {
  runId: string;
  runName: string;
};

function CompareV1(props: PageProps) {
  const urlParser = useURLParser();
  const runlistRef = React.useRef<RunList>(null);

  // State hooks
  const [collapseSections, setCollapseSections] = React.useState<Record<string, boolean>>({});
  const [fullscreenViewerConfig, setFullscreenViewerConfig] = React.useState<PlotCardProps | null>(
    null,
  );
  const [paramsCompareProps, setParamsCompareProps] = React.useState<CompareTableProps>({
    rows: [],
    xLabels: [],
    yLabels: [],
  });
  const [metricsCompareProps, setMetricsCompareProps] = React.useState<CompareTableProps>({
    rows: [],
    xLabels: [],
    yLabels: [],
  });
  const [runs, setRuns] = React.useState<ApiRunDetail[]>([]);
  const [selectedIds, setSelectedIds] = React.useState<string[]>([]);
  const [viewersMap, setViewersMap] = React.useState<Map<PlotType, TaggedViewerConfig[]>>(
    new Map(),
  );
  const [workflowObjects, setWorkflowObjects] = React.useState<Workflow[]>([]);

  // Utility functions
  const showPageError = React.useCallback(
    async (message: string, error: Error) => {
      props.updateBanner({
        additionalInfo: error.message,
        message,
        mode: 'error',
      });
    },
    [props],
  );

  const clearBanner = React.useCallback(() => {
    props.updateBanner({});
  }, [props]);

  const refresh = React.useCallback(async (): Promise<void> => {
    await load();
  }, []);

  const collapseSectionsUpdate = React.useCallback(
    (newCollapseSections: Record<string, boolean>) => {
      setCollapseSections(newCollapseSections);
    },
    [],
  );

  const collapseAllSections = React.useCallback(() => {
    const newCollapseSections: Record<string, boolean> = {};
    newCollapseSections[OVERVIEW_SECTION_NAME] = true;
    newCollapseSections[PARAMS_SECTION_NAME] = true;
    newCollapseSections[METRICS_SECTION_NAME] = true;
    Array.from(viewersMap.keys()).forEach(plotType => {
      newCollapseSections[componentMap[plotType].prototype.getDisplayName()] = true;
    });
    setCollapseSections(newCollapseSections);
  }, [viewersMap]);

  const loadParameters = React.useCallback(
    (selectedRunIds: string[]) => {
      const selectedIndices = selectedRunIds.map(id => runs.findIndex(r => r.run!.id === id));
      const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1);
      const filteredWorkflows = workflowObjects.filter((_, i) => selectedIndices.indexOf(i) > -1);

      const newParamsCompareProps = CompareUtils.getParamsCompareProps(
        filteredRuns,
        filteredWorkflows,
      );
      setParamsCompareProps(newParamsCompareProps);
    },
    [runs, workflowObjects],
  );

  const loadMetrics = React.useCallback(
    (selectedRunIds: string[]) => {
      const selectedIndices = selectedRunIds.map(id => runs.findIndex(r => r.run!.id === id));
      const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1).map(r => r.run!);

      const newMetricsCompareProps = CompareUtils.multiRunMetricsCompareProps(filteredRuns);
      setMetricsCompareProps(newMetricsCompareProps);
    },
    [runs],
  );

  const selectionChanged = React.useCallback(
    (newSelectedIds: string[]) => {
      setSelectedIds(newSelectedIds);
      loadParameters(newSelectedIds);
      loadMetrics(newSelectedIds);
    },
    [loadParameters, loadMetrics],
  );

  const load = React.useCallback(async (): Promise<void> => {
    clearBanner();

    const queryParamRunIds = urlParser.get(QUERY_PARAMS.runlist);
    const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];
    const loadedRuns: ApiRunDetail[] = [];
    const loadedWorkflowObjects: Workflow[] = [];
    const failingRuns: string[] = [];
    let lastError: Error | null = null;

    await Promise.all(
      runIds.map(async id => {
        try {
          const run = await Apis.runServiceApi.getRun(id);
          loadedRuns.push(run);
          loadedWorkflowObjects.push(JSON.parse(run.pipeline_runtime!.workflow_manifest || '{}'));
        } catch (err) {
          failingRuns.push(id);
          lastError = ensureError(err);
        }
      }),
    );

    if (lastError) {
      await showPageError(`Error: failed loading ${failingRuns.length} runs.`, lastError);
      logger.error(
        `Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`,
      );
      return;
    } else if (
      loadedRuns.length > 0 &&
      loadedRuns.every(runDetail =>
        runDetail.run?.pipeline_spec?.hasOwnProperty('pipeline_manifest'),
      )
    ) {
      props.updateBanner({
        additionalInfo:
          'The selected runs are all V2, but the V2_ALPHA feature flag is disabled.' +
          ' The V1 page will not show any useful information for these runs.',
        message:
          'Info: enable the V2_ALPHA feature flag in order to view the updated Run Comparison page.',
        mode: 'info',
      });
    }

    const newSelectedIds = loadedRuns.map(r => r.run!.id!);
    setRuns(loadedRuns);
    setSelectedIds(newSelectedIds);
    setWorkflowObjects(loadedWorkflowObjects);

    // Load parameters and metrics for the new runs
    const selectedIndices = newSelectedIds.map(id => loadedRuns.findIndex(r => r.run!.id === id));
    const filteredRuns = loadedRuns.filter((_, i) => selectedIndices.indexOf(i) > -1);
    const filteredWorkflows = loadedWorkflowObjects.filter(
      (_, i) => selectedIndices.indexOf(i) > -1,
    );

    const newParamsCompareProps = CompareUtils.getParamsCompareProps(
      filteredRuns,
      filteredWorkflows,
    );
    setParamsCompareProps(newParamsCompareProps);

    const newMetricsCompareProps = CompareUtils.multiRunMetricsCompareProps(
      filteredRuns.map(r => r.run!),
    );
    setMetricsCompareProps(newMetricsCompareProps);

    const outputPathsList = loadedWorkflowObjects.map(workflow =>
      WorkflowParser.loadAllOutputPaths(workflow),
    );

    // Maps a viewer type (ROC, table.. etc) to a list of all viewer instances
    // of that type, each tagged with its parent run id
    const newViewersMap = new Map<PlotType, TaggedViewerConfig[]>();

    await Promise.all(
      outputPathsList.map(async (pathList, i) => {
        for (const path of pathList) {
          const configs = await OutputArtifactLoader.load(
            path,
            loadedWorkflowObjects[0]?.metadata?.namespace,
          );
          configs.forEach(config => {
            const currentList: TaggedViewerConfig[] = newViewersMap.get(config.type) || [];
            currentList.push({
              config,
              runId: loadedRuns[i].run!.id!,
              runName: loadedRuns[i].run!.name!,
            });
            newViewersMap.set(config.type, currentList);
          });
        }
      }),
    );

    // For each output artifact type, list all artifact instances in all runs
    setViewersMap(newViewersMap);
  }, [urlParser, clearBanner, showPageError, props]);

  // Initialize toolbar
  React.useEffect(() => {
    const buttons = new Buttons(props, refresh, { get: urlParser.get, build: urlParser.build });
    const toolbarProps: ToolbarProps = {
      actions: buttons
        .expandSections(() => setCollapseSections({}))
        .collapseSections(collapseAllSections)
        .refresh(refresh)
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Compare runs',
    };
    props.updateToolbar(toolbarProps);
  }, [props, refresh, collapseAllSections]);

  // Load data on mount
  React.useEffect(() => {
    load();
  }, [load]);

  const queryParamRunIds = urlParser.get(QUERY_PARAMS.runlist);
  const runIds = queryParamRunIds ? queryParamRunIds.split(',') : [];

  const runsPerViewerType = (viewerType: PlotType) => {
    return viewersMap.get(viewerType)
      ? viewersMap.get(viewerType)!.filter(el => selectedIds.indexOf(el.runId) > -1)
      : [];
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lrt'))}>
      {/* Overview section */}
      <CollapseButton
        sectionName={OVERVIEW_SECTION_NAME}
        collapseSections={collapseSections}
        collapseSectionsUpdate={collapseSectionsUpdate}
      />
      {!collapseSections[OVERVIEW_SECTION_NAME] && (
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
      <CollapseButton
        sectionName={PARAMS_SECTION_NAME}
        collapseSections={collapseSections}
        collapseSectionsUpdate={collapseSectionsUpdate}
      />
      {!collapseSections[PARAMS_SECTION_NAME] && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <CompareTable {...paramsCompareProps} />
          <Hr />
        </div>
      )}

      {/* Metrics section */}
      <CollapseButton
        sectionName={METRICS_SECTION_NAME}
        collapseSections={collapseSections}
        collapseSectionsUpdate={collapseSectionsUpdate}
      />
      {!collapseSections[METRICS_SECTION_NAME] && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <CompareTable {...metricsCompareProps} />
          <Hr />
        </div>
      )}

      <Separator orientation='vertical' />

      {Array.from(viewersMap.keys()).map(
        (viewerType, i) =>
          !!runsPerViewerType(viewerType).length && (
            <div key={i}>
              <CollapseButton
                collapseSections={collapseSections}
                collapseSectionsUpdate={collapseSectionsUpdate}
                sectionName={componentMap[viewerType].prototype.getDisplayName()}
              />
              {!collapseSections[componentMap[viewerType].prototype.getDisplayName()] && (
                <React.Fragment>
                  <div className={classes(commonCss.flex, css.outputsRow)}>
                    {/* If the component allows aggregation, add one more card for
                its aggregated view. Only do this if there is more than one
                output, filtering out any unselected runs. */}
                    {componentMap[viewerType].prototype.isAggregatable() &&
                      runsPerViewerType(viewerType).length > 1 && (
                        <PlotCard
                          configs={runsPerViewerType(viewerType).map(t => t.config)}
                          maxDimension={400}
                          title='Aggregated view'
                        />
                      )}

                    {runsPerViewerType(viewerType).map((taggedConfig, c) => (
                      <PlotCard
                        key={c}
                        configs={[taggedConfig.config]}
                        title={taggedConfig.runName}
                        maxDimension={400}
                      />
                    ))}
                    <Separator />
                  </div>
                  <Hr />
                </React.Fragment>
              )}
              <Separator orientation='vertical' />
            </div>
          ),
      )}
    </div>
  );
}

const EnhancedCompareV1: React.FC<PageProps> = props => {
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    // Compare page compares two runs, when namespace changes, the runs don't
    // exist in the new namespace, so we should redirect to experiment list page.
    return <Navigate to={RoutePage.EXPERIMENTS} replace />;
  }
  return <CompareV1 {...props} />;
};

export default EnhancedCompareV1;

// Export for testing
export const TEST_ONLY = { CompareV1 };
