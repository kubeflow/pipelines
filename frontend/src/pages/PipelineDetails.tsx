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
import SidePanel from '../components/SidePanel';
import StaticNodeDetails from '../components/StaticNodeDetails';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { UnControlled as CodeMirror } from 'react-codemirror2';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding, fontsize } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';

interface PipelineDetailsState {
  graph?: dagre.graphlib.Graph;
  selectedNodeInfo: JSX.Element | null;
  pipeline: ApiPipeline | null;
  selectedTab: number;
  selectedNodeId: string;
  summaryShown: boolean;
  template?: Workflow;
  templateYaml?: string;
}

const css = stylesheet({
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
  infoSpan: {
    color: color.lowContrast,
    fontSize: fontsize.small,
    lineHeight: '24px',
    paddingLeft: 6,
  },
  summaryCard: {
    bottom: 20,
    left: 20,
    padding: 10,
    position: 'absolute',
    width: 500,
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
          title: 'Delete this Pipeline?',
        }),
        id: 'deleteBtn',
        title: 'Delete',
        tooltip: 'Delete this pipeline',
      }],
      breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
      pageTitle: this.props.match.params[RouteParams.pipelineId],
    };
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

        {pipeline && (
          <div className={commonCss.page}>
            <MD2Tabs
              selectedTab={selectedTab}
              onSwitch={(tab: number) => this.setState({ selectedTab: tab })}
              tabs={['Graph', 'Source']}
            />
            <div className={commonCss.page}>
              {selectedTab === 0 && <div className={commonCss.page}>
                {this.state.graph && <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                  {summaryShown && (
                    <Paper className={css.summaryCard}>
                      <div style={{ alignItems: 'baseline', display: 'flex', justifyContent: 'space-between' }}>
                        <div className={commonCss.header}>
                          Summary
                        </div>
                        <Button onClick={() => this.setState({ summaryShown: false })} color='secondary'>
                          Hide
                        </Button>
                      </div>
                      <div className={css.summaryKey}>Uploaded on</div>
                      <div>{new Date(pipeline.created_at!).toLocaleString()}</div>
                      <div className={css.summaryKey}>Description</div>
                      <div>{pipeline.description}</div>
                    </Paper>
                  )}

                  <Graph graph={this.state.graph} selectedNodeId={selectedNodeId}
                    onClick={id => this.setState({ selectedNodeId: id })} />

                  <SidePanel isOpen={!!selectedNodeId}
                    title={selectedNodeId} onClose={() => this.setState({ selectedNodeId: '' })}>
                    <div className={commonCss.page}>
                      {!selectedNodeInfo && <div>Unable to retrieve node info</div>}
                      {!!selectedNodeInfo && <div className={padding(20, 'lr')}>
                        <StaticNodeDetails nodeInfo={selectedNodeInfo} />
                      </div>}
                    </div>
                  </SidePanel>
                </div>}
                <div className={css.footer}>
                  {!summaryShown && (
                    <Button onClick={() => this.setState({ summaryShown: !summaryShown })} color='secondary'>
                      Show summary
                    </Button>
                  )}
                  <InfoIcon style={{ color: color.lowContrast, height: 24, width: 13 }} />
                  <span className={css.infoSpan}>Static pipeline graph</span>
                </div>
                {!this.state.graph && <span style={{ margin: '40px auto' }}>No graph to show</span>}
              </div>}
              {selectedTab === 1 &&
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
        )}
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
    const pipelineId = this.props.match.params[RouteParams.pipelineId];

    try {
      const [pipeline, templateResponse] = await Promise.all([
        Apis.pipelineServiceApi.getPipeline(pipelineId),
        Apis.pipelineServiceApi.getTemplate(pipelineId)
      ]);

      const template: Workflow = JsYaml.safeLoad(templateResponse.template!);
      let g: dagre.graphlib.Graph | undefined;
      try {
        g = StaticGraphParser.createGraph(template);
      } catch (err) {
        await this.showPageError('Error: failed to generate Pipeline graph.', err);
      }

      const breadcrumbs = [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }];
      const pageTitle = pipeline.name!;

      const toolbarActions = [...this.props.toolbarProps.actions];
      toolbarActions[0].disabled = false;
      this.props.updateToolbar({ breadcrumbs, actions: toolbarActions, pageTitle });

      this.setState({
        graph: g,
        pipeline,
        template,
        templateYaml: templateResponse.template,
      });
    }
    catch (err) {
      await this.showPageError(
        `Error: failed to retrieve pipeline or template for ID: ${pipelineId}.`,
        err,
      );
      logger.error(`Error loading pipeline or template for ID: ${pipelineId}`, err);
    }
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
