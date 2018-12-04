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

import 'codemirror/lib/codemirror.css';
import 'codemirror/mode/yaml/yaml.js';
import * as JsYaml from 'js-yaml';
import * as React from 'react';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import AddIcon from '@material-ui/icons/Add';
import Button from '@material-ui/core/Button';
import Graph from '../components/Graph';
import InfoIcon from '@material-ui/icons/InfoOutlined';
import MD2Tabs from '../atoms/MD2Tabs';
import Paper from '@material-ui/core/Paper';
import RunUtils from '../lib/RunUtils';
import SidePanel from '../components/SidePanel';
import StaticNodeDetails from '../components/StaticNodeDetails';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline, ApiGetTemplateResponse } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { UnControlled as CodeMirror } from 'react-codemirror2';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding, fontsize, fonts } from '../Css';
import { logger, errorToMessage, formatDateString } from '../lib/Utils';

interface PipelineDetailsState {
  graph?: dagre.graphlib.Graph;
  pipeline: ApiPipeline | null;
  selectedNodeId: string;
  selectedNodeInfo: JSX.Element | null;
  selectedTab: number;
  summaryShown: boolean;
  template?: Workflow;
  templateYaml?: string;
}

const summaryCardWidth = 500;

export const css = stylesheet({
  containerCss: {
    $nest: {
      '& .CodeMirror': {
        height: '100%',
        width: '80%',
      },

      '& .CodeMirror-gutters': {
        backgroundColor: '#f7f7f7',
      },
    },
    background: '#f7f7f7',
  },
  footer: {
    background: color.graphBg,
    display: 'flex',
    padding: '0 0 20px 20px',
  },
  footerInfoOffset: {
    marginLeft: summaryCardWidth + 40,
  },
  infoSpan: {
    color: color.lowContrast,
    fontFamily: fonts.secondary,
    fontSize: fontsize.small,
    letterSpacing: '0.21px',
    lineHeight: '24px',
    paddingLeft: 6,
  },
  summaryCard: {
    bottom: 20,
    left: 20,
    padding: 10,
    position: 'absolute',
    width: summaryCardWidth,
    zIndex: 1,
  },
  summaryKey: {
    color: color.strong,
    marginTop: 10,
  },
});

class PipelineDetails extends Page<{}, PipelineDetailsState> {

  constructor(props: any) {
    super(props);

    this.state = {
      pipeline: null,
      selectedNodeId: '',
      selectedNodeInfo: null,
      selectedTab: 0,
      summaryShown: true,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const fromRunId = new URLParser(this.props).get(QUERY_PARAMS.fromRunId);
    if (fromRunId) {
      return {
        actions: [],
        breadcrumbs: [
          { displayName: fromRunId, href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, fromRunId) }
        ],
        pageTitle: 'Pipeline details',
      };
    } else {
      return {
        actions: [{
          action: this._createNewExperiment.bind(this),
          icon: AddIcon,
          id: 'startNewExperimentBtn',
          primary: true,
          title: 'Start an experiment',
          tooltip: 'Create a new experiment beginning with this pipeline',
        }, {
          action: () => this.props.updateDialog({
            buttons: [
              { onClick: () => this._deleteDialogClosed(true), text: 'Delete' },
              { onClick: () => this._deleteDialogClosed(false), text: 'Cancel' },
            ],
            onClose: () => this._deleteDialogClosed(false),
            title: 'Delete this pipeline?',
          }),
          id: 'deleteBtn',
          title: 'Delete',
          tooltip: 'Delete this pipeline',
        }],
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: this.props.match.params[RouteParams.pipelineId],
      };
    }
  }

  public render(): JSX.Element {
    const { pipeline, selectedNodeId, selectedTab, summaryShown, templateYaml } = this.state;

    let selectedNodeInfo: StaticGraphParser.SelectedNodeInfo | null = null;
    if (this.state.graph && this.state.graph.node(selectedNodeId)) {
      selectedNodeInfo = this.state.graph.node(selectedNodeId).info;
      if (!!selectedNodeId && !selectedNodeInfo) {
        logger.error(`Node with ID: ${selectedNodeId} was not found in the graph`);
      }
    }

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>

        <div className={commonCss.page}>
          <MD2Tabs
            selectedTab={selectedTab}
            onSwitch={(tab: number) => this.setStateSafe({ selectedTab: tab })}
            tabs={['Graph', 'Source']}
          />
          <div className={commonCss.page}>
            {selectedTab === 0 && <div className={commonCss.page}>
              {this.state.graph && <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                {!!pipeline && summaryShown && (
                  <Paper className={css.summaryCard}>
                    <div style={{ alignItems: 'baseline', display: 'flex', justifyContent: 'space-between' }}>
                      <div className={commonCss.header}>
                        Summary
                        </div>
                      <Button onClick={() => this.setStateSafe({ summaryShown: false })} color='secondary'>
                        Hide
                        </Button>
                    </div>
                    <div className={css.summaryKey}>Uploaded on</div>
                    <div>{formatDateString(pipeline.created_at)}</div>
                    <div className={css.summaryKey}>Description</div>
                    <div>{pipeline.description}</div>
                  </Paper>
                )}

                <Graph graph={this.state.graph} selectedNodeId={selectedNodeId}
                  onClick={id => this.setStateSafe({ selectedNodeId: id })} />

                <SidePanel isOpen={!!selectedNodeId}
                  title={selectedNodeId} onClose={() => this.setStateSafe({ selectedNodeId: '' })}>
                  <div className={commonCss.page}>
                    {!selectedNodeInfo && (
                      <div className={commonCss.absoluteCenter}>Unable to retrieve node info</div>
                    )}
                    {!!selectedNodeInfo && <div className={padding(20, 'lr')}>
                      <StaticNodeDetails nodeInfo={selectedNodeInfo} />
                    </div>}
                  </div>
                </SidePanel>
                <div className={css.footer}>
                  {!summaryShown && (
                    <Button onClick={() => this.setStateSafe({ summaryShown: !summaryShown })} color='secondary'>
                      Show summary
                      </Button>
                  )}
                  <div className={classes(commonCss.flex, summaryShown && !!pipeline && css.footerInfoOffset)}>
                    <InfoIcon style={{ color: color.lowContrast, height: 16, width: 16 }} />
                    <span className={css.infoSpan}>Static pipeline graph</span>
                  </div>
                </div>
              </div>}
              {!this.state.graph && <span style={{ margin: '40px auto' }}>No graph to show</span>}
            </div>}
            {selectedTab === 1 && !!templateYaml &&
              <div className={css.containerCss}>
                <CodeMirror
                  value={templateYaml || ''}
                  editorDidMount={(editor) => editor.refresh()}
                  options={{
                    lineNumbers: true,
                    lineWrapping: true,
                    mode: 'text/yaml',
                    readOnly: true,
                    theme: 'default',
                  }}
                />
              </div>
            }
          </div>
        </div>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const fromRunId = new URLParser(this.props).get(QUERY_PARAMS.fromRunId);

    let pipeline: ApiPipeline | null = null;
    let templateYaml = '';
    let template: Workflow | undefined;
    let breadcrumbs: Array<{ displayName: string, href: string }> = [];
    const toolbarActions = [...this.props.toolbarProps.actions];
    let pageTitle = '';

    // If fromRunId is specified, load the run and get the pipeline template from it
    if (fromRunId) {
      try {
        const runDetails = await Apis.runServiceApi.getRun(fromRunId);
        templateYaml = RunUtils.getPipelineSpec(runDetails.run) || '';

        const relatedExperimentId = RunUtils.getFirstExperimentReferenceId(runDetails.run);
        let experiment: ApiExperiment | undefined;
        if (relatedExperimentId) {
          experiment = await Apis.experimentServiceApi.getExperiment(relatedExperimentId);
        }

        // Build the breadcrumbs, by adding experiment and run names
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
          displayName: runDetails.run!.name!,
          href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, fromRunId),
        });
        pageTitle = 'Pipeline details';
      } catch (err) {
        await this.showPageError('Cannot retrieve run details.', err);
        logger.error('Cannot retrieve run details.', err);
      }
    } else {
      // if fromRunId is not specified, then we have a full pipeline
      const pipelineId = this.props.match.params[RouteParams.pipelineId];
      let templateResponse: ApiGetTemplateResponse;

      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
        pageTitle = pipeline.name!;
      } catch (err) {
        await this.showPageError('Cannot retrieve pipeline details.', err);
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      try {
        templateResponse = await Apis.pipelineServiceApi.getTemplate(pipelineId);
        templateYaml = templateResponse.template || '';
      } catch (err) {
        await this.showPageError('Cannot retrieve pipeline template.', err);
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      breadcrumbs = [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }];
      toolbarActions[0].disabled = false;
    }

    this.props.updateToolbar({ breadcrumbs, actions: toolbarActions, pageTitle });

    let g: dagre.graphlib.Graph | undefined;
    try {
      template = JsYaml.safeLoad(templateYaml);
      g = StaticGraphParser.createGraph(template!);
    } catch (err) {
      await this.showPageError('Error: failed to generate Pipeline graph.', err);
    }

    this.setStateSafe({
      graph: g,
      pipeline,
      template,
      templateYaml,
    });
  }

  private _createNewExperiment(): void {
    const searchString = new URLParser(this.props).build({
      [QUERY_PARAMS.pipelineId]: this.state.pipeline!.id || ''
    });
    this.props.history.push(RoutePage.NEW_EXPERIMENT + searchString);
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      try {
        await Apis.pipelineServiceApi.deletePipeline(this.state.pipeline!.id!);
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `Successfully deleted pipeline: ${this.state.pipeline!.name}`,
          open: true,
        });
        this.props.history.push(RoutePage.PIPELINES);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: errorMessage,
          title: 'Failed to delete pipeline',
        });
        logger.error('Deleting pipeline failed with error:', err);
      }
    }
  }
}

export default PipelineDetails;
