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
import CircularProgress from '@material-ui/core/CircularProgress';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import Hr from '../atoms/Hr';
import LogViewer from '../components/LogViewer';
import MD2Tabs from '../atoms/MD2Tabs';
import PlotCard from '../components/PlotCard';
import RunUtils from '../lib/RunUtils';
import SidePanel from '../components/SidePanel';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiRun } from '../apis/run';
import { Apis } from '../lib/Apis';
import { NodePhase, statusToIcon } from './Status';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { Page } from './Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { ViewerConfig } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { formatDateString, getRunTime, logger, errorToMessage } from '../lib/Utils';

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

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: this._cloneRun.bind(this),
        id: 'cloneBtn',
        title: 'Clone',
        tooltip: 'Clone',
      }, {
        action: this.refresh.bind(this),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh',
      }],
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: this.props.runId!,
    };
  }

  public render(): JSX.Element {
    const { graph, runMetadata, selectedTab, selectedNodeDetails, sidepanelSelectedTab,
      workflow } = this.state;
    const selectedNodeId = selectedNodeDetails ? selectedNodeDetails.id : '';

    const workflowParameters = WorkflowParser.getParameters(workflow);

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>

        {!!workflow && (
          <div className={commonCss.page}>
            <MD2Tabs selectedTab={selectedTab} tabs={['Graph', 'Config']}
              onSwitch={(tab: number) => this.setStateSafe({ selectedTab: tab })} />
            <div className={commonCss.page}>

              {selectedTab === 0 && <div className={commonCss.page}>
                {graph && <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                  <Graph graph={graph} selectedNodeId={selectedNodeId}
                    onClick={(id) => this._selectNode(id)} />

                  <SidePanel isBusy={this.state.sidepanelBusy} isOpen={!!selectedNodeDetails}
                    onClose={() => this.setStateSafe({ selectedNodeDetails: null })} title={selectedNodeId}>
                    {!!selectedNodeDetails && (<React.Fragment>
                      {!!selectedNodeDetails.phaseMessage && (
                        <Banner mode='warning'
                          message={selectedNodeDetails.phaseMessage} />
                      )}
                      <div className={commonCss.page}>
                        <MD2Tabs tabs={['Artifacts', 'Input/Output', 'Logs']}
                          selectedTab={sidepanelSelectedTab}
                          onSwitch={this._loadSidePaneTab.bind(this)} />

                        {this.state.sidepanelBusy &&
                          <CircularProgress size={30} className={commonCss.absoluteCenter} />}

                        <div className={commonCss.page}>
                          {sidepanelSelectedTab === SidePaneTab.ARTIFACTS && (
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
                          )}

                          {sidepanelSelectedTab === SidePaneTab.INPUT_OUTPUT && (
                            <div className={padding(20)}>
                              <DetailsTable title='Input parameters'
                                fields={WorkflowParser.getNodeInputOutputParams(
                                  workflow, selectedNodeId)[0]} />

                              <DetailsTable title='Output parameters'
                                fields={WorkflowParser.getNodeInputOutputParams(
                                  workflow, selectedNodeId)[1]} />
                            </div>
                          )}

                          {sidepanelSelectedTab === SidePaneTab.LOGS && (
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
                          )}
                        </div>
                      </div>
                    </React.Fragment>)}
                  </SidePanel>
                </div>}
                {!graph && <span style={{ margin: '40px auto' }}>No graph to show</span>}
              </div>}

              {selectedTab === 1 && <div className={padding()}>
                <DetailsTable title='Run details' fields={this._getDetailsFields(workflow, runMetadata)} />

                {workflowParameters && !!workflowParameters.length && (<div>
                  <DetailsTable title='Run parameters'
                    fields={workflowParameters.map(p => [p.name, p.value || ''])} />
                </div>)}
              </div>}
            </div>
          </div>
        )}
      </div>
    );
  }

  public async componentDidMount(): Promise<void> {
    await this.load();
  }

  public async refresh(): Promise<void> {
    await this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const runId = this.props.match.params[RouteParams.runId];

    try {
      const runDetail = await Apis.runServiceApi.getRun(runId);
      const relatedExperimentId = RunUtils.getFirstExperimentReferenceId(runDetail.run);
      let experiment: ApiExperiment | undefined;
      if (relatedExperimentId) {
        experiment = await Apis.experimentServiceApi.getExperiment(relatedExperimentId);
      }
      const workflow = JSON.parse(runDetail.pipeline_runtime!.workflow_manifest || '{}') as Workflow;
      const runMetadata = runDetail.run!;

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
      const pageTitle = <div className={commonCss.flex}>
        {statusToIcon(runMetadata.status as NodePhase)}
        <span style={{ marginLeft: 10 }}>{runMetadata.name!}</span>
      </div>;

      this.props.updateToolbar({ breadcrumbs, pageTitle, pageTitleTooltip: runMetadata.name });

      this.setStateSafe({
        experiment,
        graph,
        runMetadata,
        workflow,
      });
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve run: ${runId}.`, err);
      logger.error('Error loading run:', runId);
    }

    // Make sure logs and artifacts in the side panel are refreshed when
    // the user hits "Refresh", either in the top toolbar or in an error banner.
    this._loadSidePaneTab(this.state.sidepanelSelectedTab);
  }

  private _getDetailsFields(workflow: Workflow, runMetadata?: ApiRun): string[][] {
    return !workflow.status ? [] : [
      ['Status', workflow.status.phase],
      ['Description', runMetadata ? runMetadata!.description! : ''],
      ['Created at', workflow.metadata ? formatDateString(workflow.metadata.creationTimestamp) : '-'],
      ['Started at', formatDateString(workflow.status.startedAt)],
      ['Finished at', formatDateString(workflow.status.finishedAt)],
      ['Duration', getRunTime(workflow)],
    ];
  }

  private _selectNode(id: string): void {
    this.setStateSafe({ selectedNodeDetails: { id } }, () =>
      this._loadSidePaneTab(this.state.sidepanelSelectedTab));
  }

  private _loadSidePaneTab(tab: SidePaneTab): void {
    const workflow = this.state.workflow;
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (workflow && workflow.status && workflow.status.nodes && selectedNodeDetails) {
      const node = workflow.status.nodes[selectedNodeDetails.id];
      if (node) {
        selectedNodeDetails.phaseMessage = (node && node.message) ?
          `This step is in ${node.phase} state with this message: ` + node.message :
          undefined;
      }
      this.setStateSafe({ selectedNodeDetails, sidepanelSelectedTab: tab });

      switch (tab) {
        case SidePaneTab.ARTIFACTS:
          this._loadSelectedNodeOutputs();
          break;
        case SidePaneTab.LOGS:
          if (node.phase !== NodePhase.SKIPPED) {
            this._loadSelectedNodeLogs();
          } else {
            // Clear logs
            this.setStateSafe({ logsBannerAdditionalInfo: '', logsBannerMessage: '' });
          }
      }
    }
  }

  private async _loadSelectedNodeOutputs(): Promise<void> {
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (!selectedNodeDetails) {
      return;
    }
    this.setStateSafe({ sidepanelBusy: true }, async () => {
      const workflow = this.state.workflow;
      if (workflow && workflow.status && workflow.status.nodes) {
        // Load runtime outputs from the selected Node
        const outputPaths = WorkflowParser.loadNodeOutputPaths(workflow.status.nodes[selectedNodeDetails.id]);

        // Load the viewer configurations from the output paths
        let viewerConfigs: ViewerConfig[] = [];
        for (const path of outputPaths) {
          viewerConfigs = viewerConfigs.concat(await OutputArtifactLoader.load(path));
        }

        selectedNodeDetails.viewerConfigs = viewerConfigs;
        this.setStateSafe({ selectedNodeDetails });
      }
      this.setStateSafe({ sidepanelBusy: false });
    });
  }

  private async _loadSelectedNodeLogs(): Promise<void> {
    const selectedNodeDetails = this.state.selectedNodeDetails;
    if (!selectedNodeDetails) {
      return;
    }
    this.setStateSafe({ sidepanelBusy: true });
    try {
      const logs = await Apis.getPodLogs(selectedNodeDetails.id);
      selectedNodeDetails.logs = logs;
      this.setStateSafe({ selectedNodeDetails, logsBannerAdditionalInfo: '', logsBannerMessage: '' });
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.setStateSafe({
        logsBannerAdditionalInfo: errorMessage,
        logsBannerMessage: 'Error: failed to retrieve logs.'
          + (errorMessage ? ' Click Details for more information.' : ''),
        logsBannerMode: 'error',
      });
      logger.error('Error loading logs for node:', selectedNodeDetails.id);
    } finally {
      this.setStateSafe({ sidepanelBusy: false });
    }
  }

  private _cloneRun(): void {
    if (this.state.runMetadata) {
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: this.state.runMetadata.id || ''
      });
      this.props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }
}

export default RunDetails;
