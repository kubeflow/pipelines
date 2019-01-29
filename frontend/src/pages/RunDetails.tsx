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
import Buttons from '../lib/Buttons';
import CircularProgress from '@material-ui/core/CircularProgress';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import Hr from '../atoms/Hr';
import LogViewer from '../components/LogViewer';
import MD2Tabs from '../atoms/MD2Tabs';
import PlotCard from '../components/PlotCard';
import RunUtils from '../lib/RunUtils';
import Separator from '../atoms/Separator';
import SidePanel from '../components/SidePanel';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiRun } from '../apis/run';
import { Apis } from '../lib/Apis';
import { NodePhase, statusToIcon, hasFinished } from './Status';
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
import { flatten } from 'lodash';
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

interface AnnotatedConfig {
  config: ViewerConfig;
  stepName: string;
}

interface RunDetailsState {
  allArtifactConfigs: AnnotatedConfig[];
  experiment?: ApiExperiment;
  logsBannerAdditionalInfo: string;
  logsBannerMessage: string;
  logsBannerMode: Mode;
  graph?: dagre.graphlib.Graph;
  runFinished: boolean;
  runMetadata?: ApiRun;
  selectedTab: number;
  selectedNodeDetails: SelectedNodeDetails | null;
  sidepanelBusy: boolean;
  sidepanelSelectedTab: SidePaneTab;
  workflow?: Workflow;
}

class RunDetails extends Page<RunDetailsProps, RunDetailsState> {
  private _onBlur: EventListener;
  private _onFocus: EventListener;
  private readonly AUTO_REFRESH_INTERVAL = 5000;

  private _interval?: NodeJS.Timeout;

  constructor(props: any) {
    super(props);

    this._onBlur = this.onBlurHandler.bind(this);
    this._onFocus = this.onFocusHandler.bind(this);

    this.state = {
      allArtifactConfigs: [],
      logsBannerAdditionalInfo: '',
      logsBannerMessage: '',
      logsBannerMode: 'error',
      runFinished: false,
      selectedNodeDetails: null,
      selectedTab: 0,
      sidepanelBusy: false,
      sidepanelSelectedTab: SidePaneTab.ARTIFACTS,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [
        Buttons.cloneRun(this._cloneRun.bind(this)),
        Buttons.refresh(this.refresh.bind(this)),
      ],
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: this.props.runId!,
    };
  }

  public render(): JSX.Element {
    const { allArtifactConfigs, graph, runMetadata, selectedTab, selectedNodeDetails,
      sidepanelSelectedTab, workflow } = this.state;
    const selectedNodeId = selectedNodeDetails ? selectedNodeDetails.id : '';

    const workflowParameters = WorkflowParser.getParameters(workflow);

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>

        {!!workflow && (
          <div className={commonCss.page}>
            <MD2Tabs selectedTab={selectedTab} tabs={['Graph', 'Run output', 'Config']}
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

              {selectedTab === 1 && (
                <div className={padding()}>
                  {!allArtifactConfigs.length && (
                    <span className={commonCss.absoluteCenter}>
                      No output artifacts found for this run.
                    </span>
                  )}

                  {allArtifactConfigs.map((annotatedConfig, i) => <div key={i}>
                    <PlotCard key={i} configs={[annotatedConfig.config]}
                      title={annotatedConfig.stepName} maxDimension={400} />
                    <Hr />
                    <Separator orientation='vertical' />
                  </div>
                  )}
                </div>
              )}

              {selectedTab === 2 && (
                <div className={padding()}>
                  <DetailsTable title='Run details' fields={this._getDetailsFields(workflow, runMetadata)} />

                  {workflowParameters && !!workflowParameters.length && (<div>
                    <DetailsTable title='Run parameters'
                      fields={workflowParameters.map(p => [p.name, p.value || ''])} />
                  </div>)}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    );
  }

  public async componentDidMount(): Promise<void> {
    window.addEventListener('focus', this._onFocus);
    window.addEventListener('blur', this._onBlur);
    await this._startAutoRefresh();
  }

  public onBlurHandler(): void {
    this._stopAutoRefresh();
  }

  public async onFocusHandler(): Promise<void> {
    await this._startAutoRefresh();
  }

  public componentWillUnmount(): void {
    this._stopAutoRefresh();
    window.removeEventListener('focus', this._onFocus);
    window.removeEventListener('blur', this._onBlur);
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
      const runMetadata = runDetail.run!;

      let runFinished = this.state.runFinished;
      // If the run has finished, stop auto refreshing
      if (hasFinished(runMetadata.status as NodePhase)) {
        this._stopAutoRefresh();
        // This prevents other events, such as onFocus, from resuming the autorefresh
        runFinished = true;
      }

      const workflow = JSON.parse(runDetail.pipeline_runtime!.workflow_manifest || '{}') as Workflow;

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
        {statusToIcon(runMetadata.status as NodePhase, runDetail.run!.created_at)}
        <span style={{ marginLeft: 10 }}>{runMetadata.name!}</span>
      </div>;

      this.props.updateToolbar({ breadcrumbs, pageTitle, pageTitleTooltip: runMetadata.name });

      this.setStateSafe({
        experiment,
        graph,
        runFinished,
        runMetadata,
        workflow,
      });
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve run: ${runId}.`, err);
      logger.error('Error loading run:', runId);
    }

    // Make sure logs and artifacts in the side panel are refreshed when
    // the user hits "Refresh", either in the top toolbar or in an error banner.
    await this._loadSidePaneTab(this.state.sidepanelSelectedTab);

    // Load all run's outputs
    await this._loadAllOutputs();
  }

  private async _startAutoRefresh(): Promise<void> {
    // If the run was not finished last time we checked, check again in case anything changed
    // before proceeding to set auto-refresh interval
    if (!this.state.runFinished) {
      // refresh() updates runFinished's value
      await this.refresh();
    }

    // Only set interval if run has not finished, and verify that the interval is undefined to
    // avoid setting multiple intervals
    if (!this.state.runFinished && this._interval === undefined) {
      this._interval = setInterval(
        () => this.refresh(),
        this.AUTO_REFRESH_INTERVAL
      );
    }
  }

  private _stopAutoRefresh(): void {
    if (this._interval !== undefined) {
      clearInterval(this._interval);
    }
    // Reset interval to indicate that a new one can be set
    this._interval = undefined;
  }

  private async _loadAllOutputs(): Promise<void> {
    const workflow = this.state.workflow;

    if (!workflow) {
      return;
    }

    const outputPathsList = WorkflowParser.loadAllOutputPathsWithStepNames(workflow);

    const configLists =
      await Promise.all(outputPathsList.map(({ stepName, path }) => OutputArtifactLoader.load(path)
        .then(configs => configs.map(config => ({ config, stepName })))));
    const allArtifactConfigs = flatten(configLists);

    this.setStateSafe({ allArtifactConfigs });
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

  private async _selectNode(id: string): Promise<void> {
    this.setStateSafe({ selectedNodeDetails: { id } }, async () =>
      await this._loadSidePaneTab(this.state.sidepanelSelectedTab));
  }

  private async _loadSidePaneTab(tab: SidePaneTab): Promise<void> {
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
          await this._loadSelectedNodeOutputs();
          break;
        case SidePaneTab.LOGS:
          if (node.phase !== NodePhase.SKIPPED) {
            await this._loadSelectedNodeLogs();
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
    this.setStateSafe({ sidepanelBusy: true });
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
