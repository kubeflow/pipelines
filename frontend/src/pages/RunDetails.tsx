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
import Banner, { Mode } from '../components/Banner';
import Button from '@material-ui/core/Button';
import CircularProgress from '@material-ui/core/CircularProgress';
import CloseIcon from '@material-ui/icons/Close';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import Hr from '../atoms/Hr';
import LogViewer from '../components/LogViewer';
import MD2Tabs from '../atoms/MD2Tabs';
import PlotCard from '../components/PlotCard';
import Resizable from 're-resizable';
import RunUtils from '../lib/RunUtils';
import Slide from '@material-ui/core/Slide';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiRun } from '../apis/run';
import { Apis } from '../lib/Apis';
import { NodePhase } from './Status';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { ViewerConfig } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { commonCss, color, padding } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { formatDateString, getRunTime, logger, errorToMessage } from '../lib/Utils';
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

interface SelectedNodeDetails {
  id: string;
  logs?: string;
  phaseMessage?: string;
  viewerConfigs?: ViewerConfig[];
}

interface RunDetailsProps {
  runId?: string;
}

interface RunDetailsState {
  experiment?: ApiExperiment;
  logsBannerAdditionalInfo: string;
  logsBannerMessage: string;
  logsBannerMode: Mode;
  graph?: dagre.graphlib.Graph;
  runMetadata?: ApiRun;
  selectedTab: number;
  selectedNodeDetails: SelectedNodeDetails | null;
  sidepanelBusy: boolean;
  sidepanelSelectedTab: SidePaneTab;
  workflow?: Workflow;
}

class RunDetails extends Page<RunDetailsProps, RunDetailsState> {

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

  public getInitialToolbarState() {
    return {
      actions: [{
        action: this._cloneRun.bind(this),
        id: 'cloneBtn',
        title: 'Clone',
        tooltip: 'Clone',
      }, {
        action: this.load.bind(this),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh',
      }],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: this.props.runId!, href: '' },
      ],
    };
  }

  public render() {
    const { graph, runMetadata, selectedTab, selectedNodeDetails, sidepanelSelectedTab,
      workflow } = this.state;

    const workflowParameters = WorkflowParser.getParameters(workflow);

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>

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
                                {(selectedNodeDetails.viewerConfigs || []).map((config, i) => {
                                  const title = componentMap[config.type].prototype.getDisplayName();
                                  return (
                                    <div key={i} className={padding(20, 'lrt')}>
                                      <PlotCard configs={[config]} title={title} maxDimension={500} />
                                      <Hr />
                                    </div>
                                  );
                                })}
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
                {!graph && <span style={{ margin: '40px auto' }}>No graph to show</span>}
              </div>}

              {selectedTab === 1 && <div className={padding()}>
                <div className={commonCss.header}>Run details</div>
                {/* TODO: show description */}
                <DetailsTable fields={[
                  ['Status', workflow.status.phase],
                  ['Description', runMetadata ? runMetadata!.description! : ''],
                  ['Created at', formatDateString(workflow.metadata.creationTimestamp)],
                  ['Started at', formatDateString(workflow.status.startedAt)],
                  ['Finished at', formatDateString(workflow.status.finishedAt)],
                  ['Duration', getRunTime(workflow)],
                ]} />

                {workflowParameters && workflowParameters.length && (<div>
                  <div className={commonCss.header}>Run parameters</div>
                  <DetailsTable fields={workflowParameters.map(p => [p.name, p.value || ''])} />
                </div>)}
              </div>}
            </div>
          </div>
        )}
      </div>
    );
  }

  public async load() {
    const runId = this.props.match.params[RouteParams.runId];

    try {
      const runDetail = await Apis.runServiceApi.getRun(runId);
      const relatedExperimentId = RunUtils.getFirstExperimentReferenceId(runDetail.run);
      let experiment: ApiExperiment | undefined;
      if (relatedExperimentId) {
        experiment = await Apis.experimentServiceApi.getExperiment(relatedExperimentId);
      }
      const workflow = JSON.parse(runDetail.pipeline_runtime!.workflow_manifest || '{}') as Workflow;
      const runMetadata = runDetail.run;

      // Show workflow errors
      const workflowError = WorkflowParser.getWorkflowError(workflow);
      if (workflowError) {
        this.showPageError(
          `Error: found errors when executing run: ${runId}.`,
          new Error(workflowError),
        );
      }

      // Build runtime graph
      const graph = workflow && workflow.status && workflow.status.nodes ?
        WorkflowParser.createRuntimeGraph(workflow) : undefined;

      const breadcrumbs: Array<{ displayName: string, href: string }> = [];
      if (experiment) {
        breadcrumbs.push(
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          {
            displayName: experiment.name!,
            href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experiment.id!)
          });
      } else {
        breadcrumbs.push(
          { displayName: 'All runs', href: RoutePage.RUNS }
        );
      }
      breadcrumbs.push({
        displayName: runMetadata ? runMetadata.name! : this.props.runId!,
        href: '',
      });

      // TODO: run status next to page name
      this.props.updateToolbar({ actions: this.props.toolbarProps.actions, breadcrumbs });

      this.setState({
        experiment,
        graph,
        runMetadata,
        workflow,
      });
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve run: ${runId}.`, err);
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
      if (node && node.message) {
        selectedNodeDetails.phaseMessage =
          `This step is in ${node.phase} state with this message: ` + node.message;
      }
      this.setState({ selectedNodeDetails, sidepanelSelectedTab: tab });

      switch (tab) {
        case SidePaneTab.ARTIFACTS:
          this._loadSelectedNodeOutputs();
          break;
        case SidePaneTab.LOGS:
          if (node.phase !== NodePhase.SKIPPED) {
            this._loadSelectedNodeLogs();
          } else {
            // Clear logs
            this.setState({ logsBannerAdditionalInfo: '', logsBannerMessage: '' });
          }
      }
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
      this.setState({ selectedNodeDetails, logsBannerAdditionalInfo: '', logsBannerMessage: '' });
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.setState({
        logsBannerAdditionalInfo: errorMessage,
        logsBannerMessage: 'Error: failed to retrieve logs.'
          + (errorMessage ? ' Click Details for more information.' : ''),
        logsBannerMode: 'error',
      });
      logger.error('Error loading logs for node:', selectedNodeDetails.id);
    } finally {
      this.setState({ sidepanelBusy: false });
    }
  }

  private _cloneRun() {
    if (this.state.runMetadata) {
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: this.state.runMetadata.id || ''
      });
      this.props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }
}

export default RunDetails;
