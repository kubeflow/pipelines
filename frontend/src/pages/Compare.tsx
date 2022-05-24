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
import { Redirect } from 'react-router-dom';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { stylesheet } from 'typestyle';
import { Workflow } from '../third_party/mlmd/argo_template';
import { ApiRunDetail } from '../apis/run';
import { CompareTableProps } from '../components/CompareTable';
import { PlotCardProps } from '../components/PlotCard';
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { componentMap } from '../components/viewers/ViewerContainer';
import { Apis } from '../lib/Apis';
import Buttons from '../lib/Buttons';
import CompareUtils from '../lib/CompareUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { URLParser } from '../lib/URLParser';
import { logger } from '../lib/Utils';
import WorkflowParser from '../lib/WorkflowParser';
import { Page, PageProps } from './Page';
import CompareV1 from './CompareV1';
import CompareV2 from './CompareV2';
import { FeatureKey, isFeatureEnabled } from 'src/features';

export interface TaggedViewerConfig {
  config: ViewerConfig;
  runId: string;
  runName: string;
}

export interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  paramsCompareProps: CompareTableProps;
  metricsCompareProps: CompareTableProps;
  runs: ApiRunDetail[];
  selectedIds: string[];
  viewersMap: Map<PlotType, TaggedViewerConfig[]>;
  workflowObjects: Workflow[];
}

const overviewSectionName = 'Run overview';
const paramsSectionName = 'Parameters';
const metricsSectionName = 'Metrics';

/**
 * TODO: How can I convert this into a functional component? I am copying the PR structure of
 * https://github.com/zpChris/pipelines/commit/6c8309d64450510450b7cdb182c0bd7ce37b6dab#diff-a5a58d21b3bf01ff61be32d1f1de444e130cde2abfbc6df68cd69e22e191703fR66.
 */
class Compare extends Page<{}, CompareState> {
  constructor(props: any) {
    super(props);

    this.state = {
      collapseSections: {},
      fullscreenViewerConfig: null,
      metricsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      paramsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .expandSections(() => this.setState({ collapseSections: {} }))
        .collapseSections(this._collapseAllSections.bind(this))
        .refresh(this.refresh.bind(this))
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Compare runs',
    };
  }

  public render(): JSX.Element {
    const { collapseSections, selectedIds, viewersMap } = this.state;

    const queryParamRunIds = new URLParser(this.props).get(QUERY_PARAMS.runlist);
    const runIds = queryParamRunIds ? queryParamRunIds.split(',') : [];

    const showV2Compare =
      isFeatureEnabled(FeatureKey.V2_ALPHA);

    if (!showV2Compare) {
      return (
        <CompareV1 {...this.props} />
      );
    } else {
      return (
        <CompareV2 />
      );
    }
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();

    const queryParamRunIds = new URLParser(this.props).get(QUERY_PARAMS.runlist);
    const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];
    const runs: ApiRunDetail[] = [];
    const workflowObjects: Workflow[] = [];
    const failingRuns: string[] = [];
    let lastError: Error | null = null;

    await Promise.all(
      runIds.map(async id => {
        try {
          const run = await Apis.runServiceApi.getRun(id);
          runs.push(run);
          workflowObjects.push(JSON.parse(run.pipeline_runtime!.workflow_manifest || '{}'));
        } catch (err) {
          failingRuns.push(id);
          lastError = err;
        }
      }),
    );

    if (lastError) {
      await this.showPageError(`Error: failed loading ${failingRuns.length} runs.`, lastError);
      logger.error(
        `Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`,
      );
      return;
    }

    const selectedIds = runs.map(r => r.run!.id!);
    this.setStateSafe({
      runs,
      selectedIds,
      workflowObjects,
    });
    this._loadParameters(selectedIds);
    this._loadMetrics(selectedIds);

    const outputPathsList = workflowObjects.map(workflow =>
      WorkflowParser.loadAllOutputPaths(workflow),
    );

    // Maps a viewer type (ROC, table.. etc) to a list of all viewer instances
    // of that type, each tagged with its parent run id
    const viewersMap = new Map<PlotType, TaggedViewerConfig[]>();

    await Promise.all(
      outputPathsList.map(async (pathList, i) => {
        for (const path of pathList) {
          const configs = await OutputArtifactLoader.load(
            path,
            workflowObjects[0]?.metadata?.namespace,
          );
          configs.forEach(config => {
            const currentList: TaggedViewerConfig[] = viewersMap.get(config.type) || [];
            currentList.push({
              config,
              runId: runs[i].run!.id!,
              runName: runs[i].run!.name!,
            });
            viewersMap.set(config.type, currentList);
          });
        }
      }),
    );

    // For each output artifact type, list all artifact instances in all runs
    this.setStateSafe({ viewersMap });
  }

  protected _selectionChanged(selectedIds: string[]): void {
    this.setState({ selectedIds });
    this._loadParameters(selectedIds);
    this._loadMetrics(selectedIds);
  }

  private _collapseAllSections(): void {
    const collapseSections = {
      [overviewSectionName]: true,
      [paramsSectionName]: true,
      [metricsSectionName]: true,
    };
    Array.from(this.state.viewersMap.keys()).forEach(t => {
      const sectionName = componentMap[t].prototype.getDisplayName();
      collapseSections[sectionName] = true;
    });
    this.setState({ collapseSections });
  }

  private _loadParameters(selectedIds: string[]): void {
    const { runs, workflowObjects } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));
    const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1);
    const filteredWorkflows = workflowObjects.filter((_, i) => selectedIndices.indexOf(i) > -1);

    const paramsCompareProps = CompareUtils.getParamsCompareProps(filteredRuns, filteredWorkflows);

    this.setState({ paramsCompareProps });
  }

  private _loadMetrics(selectedIds: string[]): void {
    const { runs } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));
    const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1).map(r => r.run!);

    const metricsCompareProps = CompareUtils.multiRunMetricsCompareProps(filteredRuns);

    this.setState({ metricsCompareProps });
  }
}

const EnhancedCompare: React.FC<PageProps> = props => {
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    // Compare page compares two runs, when namespace changes, the runs don't
    // exist in the new namespace, so we should redirect to experiment list page.
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }
  return <Compare {...props} />;
};

export default EnhancedCompare;

export const TEST_ONLY = {
  Compare,
};
