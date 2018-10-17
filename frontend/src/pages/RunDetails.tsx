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
import Banner, { BannerProps, Mode } from '../components/Banner';
import Button from '@material-ui/core/Button';
import CircularProgress from '@material-ui/core/CircularProgress';
import CloneIcon from '@material-ui/icons/FileCopy';
import CloseIcon from '@material-ui/icons/Close';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import Hr from '../atoms/Hr';
import LogViewer from '../components/LogViewer';
import MD2Tabs from '../atoms/MD2Tabs';
import RefreshIcon from '@material-ui/icons/Refresh';
import Resizable from 're-resizable';
import Slide from '@material-ui/core/Slide';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import ViewerContainer from '../components/viewers/ViewerContainer';
import { RouteComponentProps } from 'react-router';
import { RoutePage, RouteParams } from '../components/Router';
import { ViewerConfig } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { apiRun, apiJob } from '../../../frontend/src/api/job';
import { commonCss, color, padding } from '../Css';
import { formatDateString, getRunTime, logger } from '../lib/Utils';
import { loadOutputArtifacts } from '../lib/OutputArtifactLoader';
import { stylesheet, classes } from 'typestyle';

const css = stylesheet({
  closeButton: {
    color: color.inactive,
    margin: 15,
    minHeight: 0,
    minWidth: 0,
    padding: 0,
  },
  nodeName: {
    flexGrow: 1,
    textAlign: 'center',
  },
  sidepane: {
    backgroundColor: color.background,
    borderLeft: 'solid 1px #ddd',
    bottom: 0,
    display: 'flex',
    flexFlow: 'column',
    position: 'absolute !important' as any,
    right: 0,
    top: 0,
  },
});

enum SidePaneTab {
  ARTIFACTS,
  INPUT_OUTPUT,
  LOGS,
}

interface RunDetailsProps extends RouteComponentProps {
  runId?: string;
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface SelectedNodeDetails {
  id: string;
  logs?: string;
  phaseMessage?: string;
  viewerConfigs?: ViewerConfig[];
}

interface RunDetailsState {
  job?: apiJob;
  logsBannerAdditionalInfo: string;
  logsBannerMessage: string;
  logsBannerMode: Mode;
  graph?: dagre.graphlib.Graph;
  runMetadata?: apiRun;
  selectedTab: number;
  selectedNodeDetails: SelectedNodeDetails | null;
  sidepanelBusy: boolean;
  sidepanelSelectedTab: SidePaneTab;
  workflow?: Workflow;
}

class RunDetails extends React.Component<RunDetailsProps, RunDetailsState> {

  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: this._cloneRun.bind(this),
      disabled: false,
      icon: CloneIcon,
      id: 'cloneBtn',
      title: 'Clone',
      tooltip: 'Clone',
    },
    {
      action: this._loadRun.bind(this),
      disabled: false,
      icon: RefreshIcon,
      id: 'refreshBtn',
      title: 'Refresh',
      tooltip: 'Refresh',
    },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      logsBannerAdditionalInfo: '',
      logsBannerMessage: '',
      logsBannerMode: 'error',
      selectedNodeDetails: null,
      selectedTab: 0,
      sidepanelBusy: false,
      sidepanelSelectedTab: SidePaneTab.ARTIFACTS,
    };
  }

  public async componentWillMount(): Promise<void> {
    await this._loadRun();
    const { job, runMetadata } = this.state;
    const breadcrumbs = [
      { displayName: 'Jobs', href: RoutePage.JOBS },
    ];
    if (!runMetadata) {
      breadcrumbs.push(
        { displayName: this.props.runId!, href: '' },
      );
    } else {
      const jobDetailsHref = RoutePage.JOB_DETAILS.replace(
        ':' + RouteParams.jobId, runMetadata.job_id!);
      breadcrumbs.push(
        { displayName: job ? job.name! : runMetadata.job_id!, href: jobDetailsHref },
        { displayName: runMetadata.name!, href: '' },
      );
    }
    // TODO: run status next to page name
    this.props.updateToolbar({
      actions: this._toolbarActions,
      breadcrumbs,
    });
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    const { graph, selectedTab, selectedNodeDetails, sidepanelSelectedTab, workflow } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>

        {workflow && (
          <div className={commonCss.page}>
            <MD2Tabs selectedTab={selectedTab} tabs={['Graph', 'Config']}
              onSwitch={(tab: number) => this.setState({ selectedTab: tab })} />
            <div className={commonCss.page}>

              {selectedTab === 0 && <div className={commonCss.page}>
                {graph && <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                  <Graph graph={graph} selectedNodeId={selectedNodeDetails ? selectedNodeDetails.id : ''}
                    onClick={(id) => this._selectNode(id)} />
                  <Slide in={!!selectedNodeDetails} direction='left'>
                    <Resizable className={css.sidepane} defaultSize={{ width: '70%' }} maxWidth='90%'
                      minWidth={100} enable={{
                        bottom: false,
                        bottomLeft: false,
                        bottomRight: false,
                        left: true,
                        right: false,
                        top: false,
                        topLeft: false,
                        topRight: false,
                      }}>
                      {!!selectedNodeDetails && <div className={commonCss.page}>
                        <div className={commonCss.flex}>
                          <Button className={css.closeButton}
                            onClick={() => this.setState({ selectedNodeDetails: null })}>
                            <CloseIcon />
                          </Button>
                          <div className={css.nodeName}>{selectedNodeDetails.id}</div>
                        </div>
                        {this.state.selectedNodeDetails && this.state.selectedNodeDetails.phaseMessage && (
                          <Banner mode='warning'
                            message={this.state.selectedNodeDetails.phaseMessage} />
                        )}
                        <div className={commonCss.page}>
                          <MD2Tabs tabs={['Artifacts', 'Input/Output', 'Logs']}
                            selectedTab={sidepanelSelectedTab}
                            onSwitch={this._sidePaneTabSwitched.bind(this)} />

                          {this.state.sidepanelBusy &&
                            <CircularProgress size={30} className={commonCss.absoluteCenter} />}

                          <div className={commonCss.page}>
                            {sidepanelSelectedTab === SidePaneTab.ARTIFACTS &&
                              <div className={commonCss.page}>
                                {(selectedNodeDetails.viewerConfigs || []).map((config, i) => (
                                  <div key={i} className={padding(20, 'lrt')}>
                                    <ViewerContainer configs={[config]} />
                                    <Hr />
                                  </div>
                                ))}
                              </div>
                            }

                            {sidepanelSelectedTab === SidePaneTab.INPUT_OUTPUT &&
                              <div className={padding(20)}>
                                <div className={commonCss.header}>Input parameters</div>
                                <DetailsTable fields={WorkflowParser.getNodeInputOutputParams(
                                  workflow, selectedNodeDetails.id)[0]} />

                                <div className={commonCss.header}>Output parameters</div>
                                <DetailsTable fields={WorkflowParser.getNodeInputOutputParams(
                                  workflow, selectedNodeDetails.id)[1]} />
                              </div>
                            }

                            {sidepanelSelectedTab === SidePaneTab.LOGS &&
                              <div className={commonCss.page}>
                                {this.state.logsBannerMessage && (
                                  <Banner
                                    message={this.state.logsBannerMessage}
                                    mode={this.state.logsBannerMode}
                                    additionalInfo={this.state.logsBannerAdditionalInfo}
                                    refresh={this._loadSelectedNodeLogs.bind(this)} />
                                )}
                                {!this.state.logsBannerMessage && this.state.selectedNodeDetails && (
                                  <LogViewer logLines={(this.state.selectedNodeDetails.logs || '').split('\n')}
                                    classes={commonCss.page} />
                                )}
                              </div>
                            }
                          </div>
                        </div>
                      </div>}
                    </Resizable>
                  </Slide>
                </div>}
                {!graph && <span>No graph to show</span> /*TODO: proper error experience*/}
              </div>}

              {selectedTab === 1 && <div className={padding()}>
                <div className={commonCss.header}>Run details</div>
                <DetailsTable fields={[
                  ['Status', workflow.status.phase],
                  ['Created at', formatDateString(workflow.metadata.creationTimestamp)],
                  ['Started at', formatDateString(workflow.status.startedAt)],
                  ['Finished at', formatDateString(workflow.status.finishedAt)],
                  ['Duration', getRunTime(workflow)],
                ]} />

                <br />
                <br />

                {workflow.spec.arguments && workflow.spec.arguments.parameters && (<div>
                  <div className={commonCss.header}>Run parameters</div>
                  <DetailsTable fields={workflow.spec.arguments.parameters.map(p => [p.name, p.value || ''])} />
                </div>)}

              </div>}
            </div>
          </div>
        )}
      </div>
    );
  }

  private async _loadRun() {
    const runId = this.props.match.params[RouteParams.runId];

    try {
      const runDetail = await Apis.getRun(runId);
      const job = await Apis.getJob(runDetail.run!.job_id!);
      const [runMetadata, workflow] = [runDetail.run, JSON.parse(runDetail.workflow || '{}')];
      // Build runtime graph
      const graph = workflow && workflow.status && workflow.status.nodes ?
        WorkflowParser.createRuntimeGraph(workflow) : undefined;

      this.setState({
        graph,
        job,
        runMetadata,
        workflow,
      });
    } catch (err) {
      this.props.updateBanner({
        additionalInfo: err.message,
        message: `Error: failed to retrieve run: ${runId}. Click Details for more information.`,
        mode: 'error',
        refresh: this._loadRun.bind(this)
      });
      logger.error('Error loading run:', runId);
    }
  }

  private _selectNode(id: string) {
    this.setState({ selectedNodeDetails: { id } }, () =>
      this._sidePaneTabSwitched(this.state.sidepanelSelectedTab));
  }

  private async _sidePaneTabSwitched(tab: SidePaneTab) {
    const workflow = this.state.workflow;
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (workflow && workflow.status && workflow.status.nodes && selectedNodeDetails) {
      const node = workflow.status.nodes[selectedNodeDetails.id];
      if (node) {
        selectedNodeDetails.phaseMessage =
          `This step is in ${node.phase} state: ` + node.message;
        this.setState({ selectedNodeDetails });
      }
    }
    this.setState({ selectedNodeDetails, sidepanelSelectedTab: tab });

    switch (tab) {
      case SidePaneTab.ARTIFACTS:
        this._loadSelectedNodeOutputs();
        break;
      case SidePaneTab.LOGS:
        this._loadSelectedNodeLogs();
    }
  }

  private async _loadSelectedNodeOutputs() {
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (!selectedNodeDetails) {
      // This should never happen
      logger.error('Tried to load outputs for a node that is not selected');
      return;
    }
    this.setState({ sidepanelBusy: true });
    const workflow = this.state.workflow;
    if (workflow && workflow.status && workflow.status.nodes) {
      // Load runtime outputs from the selected Node
      const outputPaths = WorkflowParser.loadNodeOutputPaths(workflow.status.nodes[selectedNodeDetails.id]);

      // Load the viewer configurations from the output paths
      let viewerConfigs: ViewerConfig[] = [];
      for (const path of outputPaths) {
        viewerConfigs = viewerConfigs.concat(await loadOutputArtifacts(path));
      }

      selectedNodeDetails.viewerConfigs = viewerConfigs;
      this.setState({ selectedNodeDetails });
    }
    this.setState({ sidepanelBusy: false });
  }

  private async _loadSelectedNodeLogs() {
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (!selectedNodeDetails) {
      // This should never happen
      logger.error('Tried to load outputs for a node that is not selected');
      return;
    }
    this.setState({ sidepanelBusy: true });
    try {
      const logs = await Apis.getPodLogs(selectedNodeDetails.id);
      selectedNodeDetails.logs = logs;
      this.setState({ selectedNodeDetails });
    } catch (err) {
      this.setState({
        logsBannerAdditionalInfo: err.message,
        logsBannerMessage: `Error: failed to retrieve logs. Click Details for more information.`,
        logsBannerMode: 'error',
      });
      logger.error('Error loading logs for node:', selectedNodeDetails.id);
    } finally {
      this.setState({ sidepanelBusy: false });
    }
  }

  private _cloneRun() {
    if (this.state.runMetadata) {
      const searchString = UrlParser.from('search').build({
        [UrlParser.QUERY_PARAMS.cloneFromRun]: this.state.runMetadata.id || ''
      });
      this.props.history.push(RoutePage.NEW_JOB + searchString);
    }
  }
}

export default RunDetails;
