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
import * as UrlParser from '../lib/UrlParser';
import * as WorkflowParser from '../lib/WorkflowParser';
import { BannerProps } from '../components/Banner';
import Button from '@material-ui/core/Button';
import CloseIcon from '@material-ui/icons/Close';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import Dialog from '@material-ui/core/Dialog';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import ExpandedIcon from '@material-ui/icons/ExpandLess';
import Hr from '../atoms/Hr';
import Paper from '@material-ui/core/Paper';
import PopOutIcon from '@material-ui/icons/Launch';
import RunList from './RunList';
import Separator from '../atoms/Separator';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import Tooltip from '@material-ui/core/Tooltip';
import ViewerContainer, { componentMap } from '../components/viewers/ViewerContainer';
import { RouteComponentProps } from 'react-router';
import { RoutePage } from '../components/Router';
import { ViewerConfig, PlotType } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { apiRunDetail } from '../../../frontend/src/api/run';
import { classes, stylesheet } from 'typestyle';
import { commonCss, color, fontsize, padding } from '../Css';
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
  container: {
    flexGrow: 1,
    justifyContent: 'space-evenly',
  },
  dialogTitle: {
    color: color.strong,
    fontSize: fontsize.large,
    width: '100%',
  },
  fullscreenCloseBtn: {
    minHeight: 0,
    minWidth: 0,
    padding: 3,
  },
  fullscreenDialog: {
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: '80%',
    minWidth: '80%',
    padding: 20,
  },
  fullscreenViewerContainer: {
    alignItems: 'center',
    boxSizing: 'border-box',
    display: 'flex',
    flexFlow: 'column',
    flexGrow: 1,
    height: '100%',
    justifyContent: 'center',
    margin: 20,
    overflow: 'auto',
    width: '100%',
  },
  outputsRow: {
    overflowX: 'auto',
  },
  plotCard: {
    flexShrink: 0,
    margin: 20,
    minWidth: 250,
    padding: 20,
    width: 'min-content',
  },
  plotHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    overflow: 'hidden',
    paddingBottom: 10,
  },
  plotTitle: {
    color: color.strong,
    fontSize: 12,
    fontWeight: 'bold',
  },
  popoutIcon: {
    fontSize: 18,
  },
});

interface PlotCardProps {
  title: string;
  configs: ViewerConfig[];
  CompareInstance?: Compare;
}

class PlotCard extends React.Component<PlotCardProps> {
  public render() {
    const { title, configs, CompareInstance, ...otherProps } = this.props;

    return <Paper {...otherProps} className={css.plotCard}>
      <div className={css.plotHeader}>
        <div className={classes(css.plotTitle)} title={title}>{title}</div>
        <div>
          <Button onClick={CompareInstance ?
            () => CompareInstance!.setState({
              fullscreenViewerConfig: {
                configs,
                title,
              }
            }) :
            () => null
          }
            style={{ padding: 4, minHeight: 0, minWidth: 0 }}>
            <Tooltip title='Pop out'>
              <PopOutIcon classes={{ root: css.popoutIcon }} />
            </Tooltip>
          </Button>
        </div>
      </div>
      <ViewerContainer configs={configs} maxDimension={400} />
    </Paper>;
  }
}

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

    const runIds = UrlParser.from('search').get(UrlParser.QUERY_PARAMS.runlist).split(',');

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
                <PlotCard configs={viewersMap.get(viewerType)!.map(t => t.config)}
                  title='Aggregated view' CompareInstance={this} />
              }

              {viewersMap.get(viewerType)!.map((taggedConfig, c) => (
                <PlotCard key={c} configs={[taggedConfig.config]} title={taggedConfig.runName}
                  CompareInstance={this} />
              ))}
              <Separator />

            </div>
            <Hr />
          </div>
        }
        <Separator orientation='vertical' />
      </div>
      )}

      {!!this.state.fullscreenViewerConfig &&
        <Dialog open={!!this.state.fullscreenViewerConfig} classes={{ paper: css.fullscreenDialog }}
          onClose={() => this.setState({ fullscreenViewerConfig: null })}>
          <div className={css.dialogTitle}>
            <Button onClick={() => this.setState({ fullscreenViewerConfig: null })}
              className={css.fullscreenCloseBtn}>
              <CloseIcon />
            </Button>
            {componentMap[this.state.fullscreenViewerConfig.configs[0].type].prototype.getDisplayName()}
            <Separator />
            <span style={{ color: color.inactive }}>({this.state.fullscreenViewerConfig!.title})</span>
          </div>
          <div className={css.fullscreenViewerContainer}>
            <ViewerContainer configs={this.state.fullscreenViewerConfig!.configs} />
          </div>
        </Dialog>
      }
    </div>);
  }

  private async _loadRuns() {
    const runIdsQuery = UrlParser.from('search')
      .get(UrlParser.QUERY_PARAMS.runlist).split(',');
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
