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

import * as Apis from '../lib/Apis';
import * as React from 'react';
import * as WorkflowParser from '../lib/WorkflowParser';
import Button from '@material-ui/core/Button';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import ExpandedIcon from '@material-ui/icons/ExpandLess';
import Hr from '../atoms/Hr';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import RunList from './RunList';
import Separator from '../atoms/Separator';
import { BannerProps } from '../components/Banner';
import { RouteComponentProps } from 'react-router';
import { RoutePage } from '../components/Router';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { ViewerConfig, PlotType } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { apiRunDetail } from '../../../frontend/src/api/run';
import { classes, stylesheet } from 'typestyle';
import { commonCss, fontsize, padding } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { loadOutputArtifacts } from '../lib/OutputArtifactLoader';
import { logger } from '../lib/Utils';

const css = stylesheet({
  collapseBtn: {
    fontSize: fontsize.title,
    fontWeight: 'lighter',
    padding: 5,
  },
  collapsed: {
    transform: 'rotate(180deg)',
  },
  outputsRow: {
    overflowX: 'auto',
  },
});

interface TaggedViewerConfig {
  config: ViewerConfig;
  runName: string;
}

interface CompareProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  runs: apiRunDetail[];
  viewersMap: Map<PlotType, TaggedViewerConfig[]>;
  workflowObjects: Workflow[];
}

const overviewSectionName = 'overview';

class Compare extends React.Component<CompareProps, CompareState> {

  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: () => this.setState({ collapseSections: {} }),
      disabled: false,
      icon: ExpandIcon,
      id: 'expandBtn',
      title: 'Expand all',
      tooltip: 'Expand all sections',
    },
    {
      action: this._collapseAllSections.bind(this),
      disabled: false,
      icon: CollapseIcon,
      id: 'collapseBtn',
      title: 'Collapse all',
      tooltip: 'Collapse all sections',
    },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      collapseSections: {},
      fullscreenViewerConfig: null,
      runs: [],
      viewersMap: new Map(),
      workflowObjects: [],
    };
  }

  public componentWillMount() {
    this.props.updateToolbar({
      actions: this._toolbarActions,
      breadcrumbs: [
        { displayName: 'Jobs', href: RoutePage.JOBS },
        { displayName: 'Compare runs', href: '' },
      ],
    });
  }

  public componentDidMount() {
    this._loadRuns();
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    const { collapseSections, viewersMap } = this.state;

    const runIds = new URLParser(this.props).get(QUERY_PARAMS.runlist).split(',');

    return (<div className={classes(commonCss.page, padding(20, 'lr'))}>

      <div>
        <Button onClick={() => {
          collapseSections[overviewSectionName] = !collapseSections[overviewSectionName];
          this.setState({ collapseSections });
        }} title='Expand/Collapse this section' className={css.collapseBtn}>
          <ExpandedIcon className={collapseSections[overviewSectionName] ? css.collapsed : ''}
            style={{ marginRight: 5, transition: 'transform 0.3s' }} />
          Run overview
        </Button>
      </div>

      {!collapseSections[overviewSectionName] && <div className={commonCss.noShrink}>
        <RunList handleError={this._handlePageError.bind(this)} {...this.props} runIdListMask={runIds} disablePaging={true} />
      </div>}

      <Separator orientation='vertical' />

      {Array.from(viewersMap.keys()).map((viewerType, i) => <div key={i}>
        <div>
          <Button onClick={() => {
            collapseSections[viewerType] = !collapseSections[viewerType];
            this.setState({ collapseSections });
          }} title='Expand/Collapse this section' className={css.collapseBtn}>
            <ExpandedIcon className={collapseSections[viewerType] ? css.collapsed : ''}
              style={{ marginRight: 5, transition: 'transform 0.3s' }} />
            {componentMap[viewerType].prototype.getDisplayName()}
          </Button>
        </div>

        {!collapseSections[viewerType] &&
          <div>
            <div className={classes(commonCss.flex, css.outputsRow)}>
              {/* If the component allows aggregation, add one more card for its
              aggregated view. Only do this if there is more than one output. */}
              {(componentMap[viewerType].prototype.isAggregatable() &&
                viewersMap.get(viewerType)!.length > 1) &&
                <PlotCard configs={viewersMap.get(viewerType)!.map(t => t.config)} maxDimension={400}
                  title='Aggregated view' />
              }

              {viewersMap.get(viewerType)!.map((taggedConfig, c) => (
                <PlotCard key={c} configs={[taggedConfig.config]} title={taggedConfig.runName}
                  maxDimension={400} />
              ))}
              <Separator />

            </div>
            <Hr />
          </div>
        }
        <Separator orientation='vertical' />
      </div>
      )}
    </div>);
  }

  private async _loadRuns() {
    const runIdsQuery = new URLParser(this.props).get(QUERY_PARAMS.runlist).split(',');
    const runs: apiRunDetail[] = [];
    const workflowObjects: Workflow[] = [];
    const failingRuns: string[] = [];
    let lastError = '';
    await Promise.all(runIdsQuery.map(async id => {
      try {
        const run = await Apis.getRun(id);
        runs.push(run);
        workflowObjects.push(JSON.parse(run.workflow || '{}'));
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
        refresh: this._loadRuns.bind(this),
      });
      logger.error(`Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`);
      return;
    }

    this.setState({ runs, workflowObjects });

    if (!runs || !workflowObjects) {
      return;
    }

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
    const collapseSections = { [overviewSectionName]: true };
    Array.from(this.state.viewersMap.keys()).map(t => collapseSections[t] = true);
    this.setState({
      collapseSections,
    });
  }

  private _handlePageError(message: string, error: Error): void {
    this.props.updateBanner({
      additionalInfo: error.message,
      message,
      mode: 'error',
      refresh: this._loadRuns.bind(this),
    });
  }
}

export default Compare;
