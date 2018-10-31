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
import * as WorkflowParser from '../lib/WorkflowParser';
import Button from '@material-ui/core/Button';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import CompareTable from '../components/CompareTable';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import ExpandedIcon from '@material-ui/icons/ArrowDropUp';
import Hr from '../atoms/Hr';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import RunList from './RunList';
import Separator from '../atoms/Separator';
import { ApiRunDetail } from '../apis/run';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage } from '../components/Router';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { ViewerConfig, PlotType } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, fontsize, padding, color } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { countBy, flatten } from 'lodash';
import { loadOutputArtifacts } from '../lib/OutputArtifactLoader';
import { logger } from '../lib/Utils';

const css = stylesheet({
  collapseBtn: {
    color: color.strong,
    fontSize: fontsize.title,
    fontWeight: 'bold',
    padding: 5,
  },
  collapsed: {
    transform: 'rotate(180deg)',
  },
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

interface CollapseButtonProps {
  compareComponent: Compare;
  sectionName: string;
}

// tslint:disable-next-line:variable-name
const CollapseButton = (props: CollapseButtonProps) => {
  const collapseSections = props.compareComponent.state.collapseSections;
  const sectionName = props.sectionName;
  return <div>
    <Button onClick={() => {
      collapseSections[sectionName] = !collapseSections[sectionName];
      props.compareComponent.setState({ collapseSections });
    }} title='Expand/Collapse this section' className={css.collapseBtn}>
      <ExpandedIcon className={collapseSections[sectionName] ? css.collapsed : ''}
        style={{ marginRight: 5, transition: 'transform 0.3s' }} />
      {sectionName}
    </Button>
  </div>;
};

interface TaggedViewerConfig {
  config: ViewerConfig;
  runId: string;
  runName: string;
}

interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  paramsTableRows: string[][];
  paramsTableXLabels: string[];
  paramsTableYLabels: string[];
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
      paramsTableRows: [],
      paramsTableXLabels: [],
      paramsTableYLabels: [],
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    };
  }

  public getInitialToolbarState() {
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

  public render() {
    const { collapseSections, paramsTableRows, paramsTableXLabels, paramsTableYLabels,
      selectedIds, viewersMap } = this.state;

    const runIds = new URLParser(this.props).get(QUERY_PARAMS.runlist).split(',');

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
          <RunList onError={this._handlePageError.bind(this)} {...this.props}
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
          <CompareTable rows={paramsTableRows} xLabels={paramsTableXLabels} yLabels={paramsTableYLabels} />
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

  public async load() {
    const runIdsQuery = new URLParser(this.props).get(QUERY_PARAMS.runlist).split(',');
    const runs: ApiRunDetail[] = [];
    const workflowObjects: Workflow[] = [];
    const failingRuns: string[] = [];
    let lastError = '';
    await Promise.all(runIdsQuery.map(async id => {
      try {
        const run = await Apis.runServiceApi.getRun(id);
        runs.push(run);
        workflowObjects.push(JSON.parse(run.pipeline_runtime!.workflow_manifest || '{}'));
      } catch (err) {
        failingRuns.push(id);
        lastError = err.message;
      }
    }));

    if (lastError) {
      this.props.updateBanner({
        additionalInfo: `The last error was:\n\n${lastError}`,
        message: `Error: failed loading ${failingRuns.length} runs. Click Details for more information.`,
        mode: 'error',
        refresh: this.load.bind(this),
      });
      logger.error(`Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`);
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

  private _collapseAllSections() {
    const collapseSections = { [overviewSectionName]: true, [paramsSectionName]: true };
    Array.from(this.state.viewersMap.keys()).map(t => {
      const sectionName = componentMap[t].prototype.getDisplayName();
      collapseSections[sectionName] = true;
    });
    this.setState({
      collapseSections,
    });
  }

  private _selectionChanged(selectedIds: string[]) {
    this.setState({ selectedIds }, () => this._loadParameters());
  }

  private _loadParameters() {
    const { runs, selectedIds, workflowObjects } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));

    const parameterNames = flatten(workflowObjects
      .filter((_, i) => selectedIndices.indexOf(i) > -1)
      .map(workflow => ((workflow.spec.arguments || {}).parameters || []).map(p => p.name)));

    const paramsTableXLabels = runs
      .filter((_, i) => selectedIndices.indexOf(i) > -1)
      .map(r => r.run!.name!);
    const paramsTableYLabels = Object.keys(countBy(parameterNames));

    const paramsTableRows = paramsTableYLabels.map(name => {
      return workflowObjects
        .filter((_, i) => selectedIndices.indexOf(i) > -1)
        .map(w => {
          const param =
            ((w.spec.arguments || {}).parameters || []).find(p => p.name === name);
          return param ? param.value || '' : '';
        });
    });

    this.setState({ paramsTableRows, paramsTableXLabels, paramsTableYLabels });
  }

  private _handlePageError(message: string, error: Error): void {
    this.props.updateBanner({
      additionalInfo: error.message,
      message,
      mode: 'error',
      refresh: this.load.bind(this),
    });
  }
}

export default Compare;
