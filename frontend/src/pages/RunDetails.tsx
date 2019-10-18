/*
 * Copyright 2018-2019 Google LLC
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
import Buttons, { ButtonKeys } from '../lib/Buttons';
import CircularProgress from '@material-ui/core/CircularProgress';
import CompareTable from '../components/CompareTable';
import CompareUtils from '../lib/CompareUtils';
import DetailsTable from '../components/DetailsTable';
import MinioArtifactLink from '../components/MinioArtifactLink';
import Graph from '../components/Graph';
import Hr from '../atoms/Hr';
import InfoIcon from '@material-ui/icons/InfoOutlined';
import LogViewer from '../components/LogViewer';
import MD2Tabs from '../atoms/MD2Tabs';
import PlotCard from '../components/PlotCard';
import RunUtils from '../lib/RunUtils';
import Separator from '../atoms/Separator';
import SidePanel from '../components/SidePanel';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiRun, RunStorageState } from '../apis/run';
import { Apis } from '../lib/Apis';
import { NodePhase, hasFinished } from '../lib/StatusUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { KeyValue } from '../lib/StaticGraphParser';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { ViewerConfig, PlotType } from '../components/viewers/Viewer';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, color, fonts, fontsize } from '../Css';
import { componentMap } from '../components/viewers/ViewerContainer';
import { flatten } from 'lodash';
import { formatDateString, getRunDurationFromWorkflow, logger, errorToMessage } from '../lib/Utils';
import { statusToIcon } from './Status';
import VisualizationCreator, { VisualizationCreatorConfig } from '../components/viewers/VisualizationCreator';
import { ApiVisualization, ApiVisualizationType } from '../apis/visualization';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';

enum SidePaneTab {
  ARTIFACTS,
  INPUT_OUTPUT,
  VOLUMES,
  MANIFEST,
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

interface GeneratedVisualization {
  config: HTMLViewerConfig;
  nodeId: string;
}

interface RunDetailsState {
  allArtifactConfigs: AnnotatedConfig[];
  allowCustomVisualizations: boolean;
  experiment?: ApiExperiment;
  generatedVisualizations: GeneratedVisualization[];
  isGeneratingVisualization: boolean;
  legacyStackdriverUrl: string;
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
  stackdriverK8sLogsUrl: string;
  workflow?: Workflow;
}

export const css = stylesheet({
  footer: {
    background: color.graphBg,
    display: 'flex',
    padding: '0 0 20px 20px',
  },
  graphPane: {
    backgroundColor: color.graphBg,
    overflow: 'hidden',
    position: 'relative',
  },
  infoSpan: {
    color: color.lowContrast,
    fontFamily: fonts.secondary,
    fontSize: fontsize.small,
    letterSpacing: '0.21px',
    lineHeight: '24px',
    paddingLeft: 6,
  },
  link: {
    color: '#77abda'
  },
  outputTitle: {
    color: color.strong,
    fontSize: fontsize.title,
    fontWeight: 'bold',
    paddingLeft: 20,
  },
});

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
      allowCustomVisualizations: false,
      generatedVisualizations: [],
      isGeneratingVisualization: false,
      legacyStackdriverUrl: '',
      logsBannerAdditionalInfo: '',
      logsBannerMessage: '',
      logsBannerMode: 'error',
      runFinished: false,
      selectedNodeDetails: null,
      selectedTab: 0,
      sidepanelBusy: false,
      sidepanelSelectedTab: SidePaneTab.ARTIFACTS,
      stackdriverK8sLogsUrl: '',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .retryRun(
            () => this.state.runMetadata? [this.state.runMetadata!.id!] : [],
            true, () => this.retry())
        .cloneRun(() => this.state.runMetadata ? [this.state.runMetadata!.id!] : [], true)
        .terminateRun(
          () => this.state.runMetadata ? [this.state.runMetadata!.id!] : [],
          true,
          () => this.refresh()
        )
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: this.props.runId!,
    };
  }

  public render(): JSX.Element {
    const {
      allArtifactConfigs,
      allowCustomVisualizations,
      graph,
      isGeneratingVisualization,
      legacyStackdriverUrl,
      runFinished,
      runMetadata,
      selectedTab,
      selectedNodeDetails,
      sidepanelSelectedTab,
      stackdriverK8sLogsUrl,
      workflow
    } = this.state;
    const selectedNodeId = selectedNodeDetails ? selectedNodeDetails.id : '';

    const workflowParameters = WorkflowParser.getParameters(workflow);
    const {inputParams, outputParams} = WorkflowParser.getNodeInputOutputParams(workflow, selectedNodeId);
    const {inputArtifacts, outputArtifacts} = WorkflowParser.getNodeInputOutputArtifacts(workflow, selectedNodeId);
    const hasMetrics = runMetadata && runMetadata.metrics && runMetadata.metrics.length > 0;
    const visualizationCreatorConfig: VisualizationCreatorConfig = {
      allowCustomVisualizations,
      isBusy: isGeneratingVisualization,
      onGenerate: (visualizationArguments: string, source: string, type: ApiVisualizationType) => {
        this._onGenerate(visualizationArguments, source, type);
      },
      type: PlotType.VISUALIZATION_CREATOR,
    };

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>

        {!!workflow && (
          <div className={commonCss.page}>
            <MD2Tabs selectedTab={selectedTab} tabs={['Graph', 'Run output', 'Config']}
              onSwitch={(tab: number) => this.setStateSafe({ selectedTab: tab })} />
            <div className={commonCss.page}>

              {/* Graph tab */}
              {selectedTab === 0 && <div className={classes(commonCss.page, css.graphPane)}>
                {graph && <div className={commonCss.page}>
                  <Graph graph={graph} selectedNodeId={selectedNodeId}
                    onClick={(id) => this._selectNode(id)} />

                  <SidePanel isBusy={this.state.sidepanelBusy} isOpen={!!selectedNodeDetails}
                    onClose={() => this.setStateSafe({ selectedNodeDetails: null })} title={selectedNodeId}>
                    {!!selectedNodeDetails && (<React.Fragment>
                      {!!selectedNodeDetails.phaseMessage && (
                        <Banner mode='warning' message={selectedNodeDetails.phaseMessage} />
                      )}
                      <div className={commonCss.page}>
                        <MD2Tabs tabs={['Artifacts', 'Input/Output', 'Volumes', 'Manifest', 'Logs']}
                          selectedTab={sidepanelSelectedTab}
                          onSwitch={this._loadSidePaneTab.bind(this)} />

                        <div className={commonCss.page}>
                          {sidepanelSelectedTab === SidePaneTab.ARTIFACTS && (
                            <div className={commonCss.page}>
                              <div className={padding(20, 'lrt')}>
                                <PlotCard
                                  configs={[visualizationCreatorConfig]}
                                  title={VisualizationCreator.prototype.getDisplayName()}
                                  maxDimension={500} />
                                <Hr />
                              </div>
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
                                fields={inputParams} />

                              <DetailsTable title='Input artifacts'
                                fields={inputArtifacts}
                                valueComponent={MinioArtifactLink} />

                              <DetailsTable title='Output parameters'
                                fields={outputParams} />

                              <DetailsTable title='Output artifacts'
                                fields={outputArtifacts}
                                valueComponent={MinioArtifactLink} />
                            </div>
                          )}

                          {sidepanelSelectedTab === SidePaneTab.VOLUMES && (
                            <div className={padding(20)}>
                              <DetailsTable title='Volume Mounts'
                                fields={WorkflowParser.getNodeVolumeMounts(
                                  workflow, selectedNodeId)} />
                            </div>
                          )}

                          {sidepanelSelectedTab === SidePaneTab.MANIFEST && (
                            <div className={padding(20)}>
                              <DetailsTable title='Resource Manifest'
                                fields={WorkflowParser.getNodeManifest(
                                  workflow, selectedNodeId)} />
                            </div>
                          )}

                          {sidepanelSelectedTab === SidePaneTab.LOGS && (
                            <div className={commonCss.page}>
                              {this.state.logsBannerMessage && (
                                <React.Fragment>
                                  <Banner
                                    message={this.state.logsBannerMessage}
                                    mode={this.state.logsBannerMode}
                                    additionalInfo={this.state.logsBannerAdditionalInfo}
                                    refresh={this._loadSelectedNodeLogs.bind(this)} />
                                  {(legacyStackdriverUrl && stackdriverK8sLogsUrl) && (
                                    <div className={padding(20, 'blr')}>
                                      Logs can still be viewed in either <a href={legacyStackdriverUrl} target='_blank' className={classes(css.link, commonCss.unstyled)}>Legacy Stackdriver
                                    </a> or in <a href={stackdriverK8sLogsUrl} target='_blank' className={classes(css.link, commonCss.unstyled)}>Stackdriver Kubernetes Monitoring</a>
                                    </div>
                                  )}
                                </React.Fragment>
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

                  <div className={css.footer}>
                    <div className={commonCss.flex}>
                      <InfoIcon className={commonCss.infoIcon} />
                      <span className={css.infoSpan}>
                        Runtime execution graph. Only steps that are currently running or have already completed are shown.
                    </span>
                    </div>
                  </div>
                </div>}
                {!graph && (
                  <div>
                    {runFinished && (
                      <span style={{ margin: '40px auto' }}>
                        No graph to show
                      </span>
                    )}
                    {!runFinished && (
                      <CircularProgress size={30} className={commonCss.absoluteCenter} />
                    )}
                  </div>
                )}
              </div>}

              {/* Run outputs tab */}
              {selectedTab === 1 && (
                <div className={padding()}>
                  {hasMetrics && (
                    <div>
                      <div className={css.outputTitle}>Metrics</div>
                      <div className={padding(20, 'lt')}>
                        <CompareTable {...CompareUtils.singleRunToMetricsCompareProps(runMetadata, workflow)} />
                      </div>
                    </div>
                  )}
                  {!hasMetrics && (
                    <span>
                      No metrics found for this run.
                    </span>
                  )}

                  <Separator orientation='vertical' />
                  <Hr />

                  {allArtifactConfigs.map((annotatedConfig, i) => <div key={i}>
                    <PlotCard key={i} configs={[annotatedConfig.config]}
                      title={annotatedConfig.stepName} maxDimension={400} />
                    <Separator orientation='vertical' />
                    <Hr />
                  </div>
                  )}
                  {!allArtifactConfigs.length && (
                    <span>
                      No output artifacts found for this run.
                    </span>
                  )}
                </div>
              )}

              {/* Config tab */}
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
    this.clearBanner();
  }

  public async retry(): Promise<void>{
    const runFinished = false;
    this.setStateSafe({
      runFinished,
    });

    await this._startAutoRefresh();
  }

  public async refresh(): Promise<void> {
    await this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const runId = this.props.match.params[RouteParams.runId];

    try {
      const allowCustomVisualizations = await Apis.areCustomVisualizationsAllowed();
      this.setState({ allowCustomVisualizations });
    } catch (err) {
      this.showPageError('Error: Unable to enable custom visualizations.', err);
    }

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
        if (workflowError === 'terminated') {
          this.props.updateBanner({
            additionalInfo: `This run's workflow included the following message: ${workflowError}`,
            message: 'This run was terminated',
            mode: 'warning',
            refresh: undefined,
          });
        } else {
          this.showPageError(
            `Error: found errors when executing run: ${runId}.`,
            new Error(workflowError),
          );
        }
      }

      // Build runtime graph
      const graph = workflow && workflow.status && workflow.status.nodes ?
        WorkflowParser.createRuntimeGraph(workflow) : undefined;

      const breadcrumbs: Array<{ displayName: string, href: string }> = [];
      // If this is an archived run, only show Archive in breadcrumbs, otherwise show
      // the full path, including the experiment if any.
      if (runMetadata.storage_state === RunStorageState.ARCHIVED) {
        breadcrumbs.push({ displayName: 'Archive', href: RoutePage.ARCHIVE });
      } else {
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
      }
      const pageTitle = <div className={commonCss.flex}>
        {statusToIcon(runMetadata.status as NodePhase, runDetail.run!.created_at)}
        <span style={{ marginLeft: 10 }}>{runMetadata.name!}</span>
      </div>;

      // Update the Archive/Restore button based on the storage state of this run
      const buttons = new Buttons(this.props, this.refresh.bind(this), this.getInitialToolbarState().actions);
      const idGetter = () => runMetadata ? [runMetadata!.id!] : [];
      runMetadata!.storage_state === RunStorageState.ARCHIVED ?
        buttons.restore(idGetter, true, () => this.refresh()) :
        buttons.archive(idGetter, true, () => this.refresh());
      const actions = buttons.getToolbarActionMap();
      actions[ButtonKeys.TERMINATE_RUN].disabled =
          (runMetadata.status as NodePhase) === NodePhase.TERMINATING || runFinished;
      actions[ButtonKeys.RETRY].disabled =
          (runMetadata.status as NodePhase) !== NodePhase.FAILED && (runMetadata.status as NodePhase) !== NodePhase.ERROR ;
      this.props.updateToolbar({ actions, breadcrumbs, pageTitle, pageTitleTooltip: runMetadata.name });

      this.setStateSafe({
        experiment,
        graph,
        legacyStackdriverUrl: '', // Reset legacy Stackdriver logs URL
        runFinished,
        runMetadata,
        stackdriverK8sLogsUrl: '', // Reset Kubernetes Stackdriver logs URL
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

      // Reset interval to indicate that a new one can be set
      this._interval = undefined;
    }
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

  private _getDetailsFields(workflow: Workflow, runMetadata?: ApiRun): Array<KeyValue<string>> {
    return !workflow.status ? [] : [
      ['Status', workflow.status.phase],
      ['Description', runMetadata ? runMetadata!.description! : ''],
      ['Created at', workflow.metadata ? formatDateString(workflow.metadata.creationTimestamp) : '-'],
      ['Started at', formatDateString(workflow.status.startedAt)],
      ['Finished at', formatDateString(workflow.status.finishedAt)],
      ['Duration', getRunDurationFromWorkflow(workflow)],
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
    const { generatedVisualizations, selectedNodeDetails } = this.state;
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
      const generatedConfigs = generatedVisualizations
        .filter(visualization => visualization.nodeId === selectedNodeDetails.id)
        .map(visualization => visualization.config);
      viewerConfigs = viewerConfigs.concat(generatedConfigs);

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
      try {
        const projectId = await Apis.getProjectId();
        const clusterName = await Apis.getClusterName();
        this.setStateSafe({
          legacyStackdriverUrl: `https://console.cloud.google.com/logs/viewer?project=${projectId}&interval=NO_LIMIT&advancedFilter=resource.type%3D"container"%0Aresource.labels.cluster_name:"${clusterName}"%0Aresource.labels.pod_id:"${selectedNodeDetails.id}"`,
          logsBannerMessage: 'Warning: failed to retrieve pod logs. Possible reasons include cluster autoscaling or pod preemption',
          logsBannerMode: 'warning',
          stackdriverK8sLogsUrl: `https://console.cloud.google.com/logs/viewer?project=${projectId}&interval=NO_LIMIT&advancedFilter=resource.type%3D"k8s_container"%0Aresource.labels.cluster_name:"${clusterName}"%0Aresource.labels.pod_name:"${selectedNodeDetails.id}"`,
        });
      } catch (fetchSystemInfoErr) {
        const errorMessage = await errorToMessage(err);
        this.setStateSafe({
          logsBannerAdditionalInfo: errorMessage,
          logsBannerMessage: 'Error: failed to retrieve pod logs.'
            + (errorMessage ? ' Click Details for more information.' : ''),
          logsBannerMode: 'error',
        });
      }
      logger.error('Error loading logs for node:', selectedNodeDetails.id);
    } finally {
      this.setStateSafe({ sidepanelBusy: false });
    }
  }

  private async _onGenerate(visualizationArguments: string, source: string, type: ApiVisualizationType): Promise<void> {
    const nodeId = this.state.selectedNodeDetails ? this.state.selectedNodeDetails.id : '';
    if (nodeId.length === 0) {
      this.showPageError('Unable to generate visualization, no component selected.');
      return;
    }

    if (visualizationArguments.length) {
      try {
        // Attempts to validate JSON, if attempt fails an error is displayed.
        JSON.parse(visualizationArguments);
      } catch (err) {
        this.showPageError('Unable to generate visualization, invalid JSON provided.', err);
        return;
      }
    }
    this.setState({ isGeneratingVisualization: true });
    const visualizationData: ApiVisualization = {
      arguments: visualizationArguments,
      source,
      type,
    };
    try {
      const config = await Apis.buildPythonVisualizationConfig(visualizationData);
      const { generatedVisualizations, selectedNodeDetails } = this.state;
      const generatedVisualization: GeneratedVisualization = {
        config,
        nodeId,
      };
      generatedVisualizations.push(generatedVisualization);
      if (selectedNodeDetails) {
        const viewerConfigs = selectedNodeDetails.viewerConfigs || [];
        viewerConfigs.push(generatedVisualization.config);
        selectedNodeDetails.viewerConfigs = viewerConfigs;
      }
      this.setState({ generatedVisualizations, selectedNodeDetails });
    } catch (err) {
      this.showPageError('Unable to generate visualization, an unexpected error was encountered.', err);
    } finally {
      this.setState({ isGeneratingVisualization: false });
    }
  }
}

export default RunDetails;
