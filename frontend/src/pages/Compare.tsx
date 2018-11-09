/*
 * Copyright 2018 Google LLC
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
import CollapseButton from '../components/CollapseButton';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import CompareTable, { CompareTableProps } from '../components/CompareTable';
import CompareUtils from '../lib/CompareUtils';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import Hr from '../atoms/Hr';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import RunList from './RunList';
import Separator from '../atoms/Separator';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiRunDetail } from '../apis/run';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { ViewerConfig, PlotType } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { loadOutputArtifacts } from '../lib/OutputArtifactLoader';
import { logger } from '../lib/Utils';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

interface TaggedViewerConfig {
  config: ViewerConfig;
  runId: string;
  runName: string;
}

interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  paramsCompareProps: CompareTableProps;
  runs: ApiRunDetail[];
  selectedIds: string[];
  viewersMap: Map<PlotType, TaggedViewerConfig[]>;
  workflowObjects: Workflow[];
}

const overviewSectionName = 'Run overview';
const paramsSectionName = 'Parameters';

class Compare extends Page<{}, CompareState> {

  constructor(props: any) {
    super(props);

    this.state = {
      collapseSections: {},
      fullscreenViewerConfig: null,
      paramsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: () => this.setState({ collapseSections: {} }),
        icon: ExpandIcon,
        id: 'expandBtn',
        title: 'Expand all',
        tooltip: 'Expand all sections',
      }, {
        action: this._collapseAllSections.bind(this),
        icon: CollapseIcon,
        id: 'collapseBtn',
        title: 'Collapse all',
        tooltip: 'Collapse all sections',
      }],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Compare runs', href: '' },
      ],
    };
  }

  public render(): JSX.Element {
    const { collapseSections, selectedIds, viewersMap } = this.state;

    const queryParamRunIds = new URLParser(this.props).get(QUERY_PARAMS.runlist);
    const runIds = queryParamRunIds ? queryParamRunIds.split(',') : [];

    const runsPerViewerType = (viewerType: PlotType) => {
      return viewersMap.get(viewerType) ? viewersMap
        .get(viewerType)!
        .filter(el => selectedIds.indexOf(el.runId) > -1) : [];
    };

    return (<div className={classes(commonCss.page, padding(20, 'lrt'))}>

      {/* Overview section */}
      <CollapseButton compareComponent={this} sectionName={overviewSectionName} />
      {!collapseSections[overviewSectionName] && (
        <div className={commonCss.noShrink}>
          <RunList onError={this.showPageError.bind(this)} {...this.props}
            selectedIds={selectedIds} runIdListMask={runIds} disablePaging={true}
            onSelectionChange={this._selectionChanged.bind(this)} />
        </div>
      )}

      <Separator orientation='vertical' />

      {/* Parameters section */}
      <CollapseButton compareComponent={this} sectionName={paramsSectionName} />
      {!collapseSections[paramsSectionName] && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <CompareTable {...this.state.paramsCompareProps} />
          <Hr />
        </div>
      )}

      <Separator orientation='vertical' />

      {Array.from(viewersMap.keys()).map((viewerType, i) => <div key={i}>
        <CollapseButton compareComponent={this}
          sectionName={componentMap[viewerType].prototype.getDisplayName()} />
        {!collapseSections[componentMap[viewerType].prototype.getDisplayName()] &&
          <React.Fragment>
            <div className={classes(commonCss.flex, css.outputsRow)}>
              {/* If the component allows aggregation, add one more card for
              its aggregated view. Only do this if there is more than one
              output, filtering out any unselected runs. */}
              {(componentMap[viewerType].prototype.isAggregatable() && (
                runsPerViewerType(viewerType).length > 1) && (
                  <PlotCard configs={
                    runsPerViewerType(viewerType).map(t => t.config)} maxDimension={400}
                    title='Aggregated view' />
                )
              )}

              {runsPerViewerType(viewerType).map((taggedConfig, c) => (
                <PlotCard key={c} configs={[taggedConfig.config]} title={taggedConfig.runName}
                  maxDimension={400} />
              ))}
              <Separator />

            </div>
            <Hr />
          </React.Fragment>
        }
        <Separator orientation='vertical' />
      </div>
      )}
    </div>);
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
    await Promise.all(runIds.map(async id => {
      try {
        const run = await Apis.runServiceApi.getRun(id);
        runs.push(run);
        workflowObjects.push(JSON.parse(run.pipeline_runtime!.workflow_manifest || '{}'));
      } catch (err) {
        failingRuns.push(id);
        lastError = err;
      }
    }));

    if (lastError) {
      await this.showPageError(
        `Error: failed loading ${failingRuns.length} runs.`,
        lastError,
      );
      logger.error(
        `Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`);
      return;
    }

    this.setState({
      runs,
      selectedIds: runs.map(r => r.run!.id!),
      workflowObjects,
    }, () => this._loadParameters());

    const outputPathsList = workflowObjects.map(
      workflow => WorkflowParser.loadAllOutputPaths(workflow));

    // Maps a viewer type (ROC, table.. etc) to a list of all viewer instances
    // of that type, each tagged with its parent run id
    const viewersMap = new Map<PlotType, TaggedViewerConfig[]>();

    await Promise.all(outputPathsList.map(async (pathList, i) => {
      for (const path of pathList) {
        const configs = await loadOutputArtifacts(path);
        configs.map(config => {
          const currentList: TaggedViewerConfig[] = viewersMap.get(config.type) || [];
          currentList.push({
            config,
            runId: runs[i].run!.id!,
            runName: runs[i].run!.name!,
          });
          viewersMap.set(config.type, currentList);
        });
      }
    }));

    // For each output artifact type, list all artifact instances in all runs
    this.setState({ viewersMap });
  }

  private _collapseAllSections(): void {
    const collapseSections = { [overviewSectionName]: true, [paramsSectionName]: true };
    Array.from(this.state.viewersMap.keys()).map(t => {
      const sectionName = componentMap[t].prototype.getDisplayName();
      collapseSections[sectionName] = true;
    });
    this.setState({
      collapseSections,
    });
  }

  private _selectionChanged(selectedIds: string[]): void {
    this.setState({ selectedIds }, () => this._loadParameters());
  }

  private _loadParameters(): void {
    const { runs, selectedIds, workflowObjects } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));
    const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1);
    const filteredWorkflows = workflowObjects.filter((_, i) => selectedIndices.indexOf(i) > -1);

    const paramsCompareProps = CompareUtils.getParamsCompareProps(
      filteredRuns, filteredWorkflows);

    this.setState({ paramsCompareProps });
  }
}

export default Compare;
