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
import Button from '@material-ui/core/Button';
import CircularProgress from '@material-ui/core/CircularProgress';
import CloseIcon from '@material-ui/icons/Close';
import Collapse from '@material-ui/core/Collapse';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import MD2Tabs from '../atoms/MD2Tabs';
import Paper from '@material-ui/core/Paper';
import Resizable from 're-resizable';
import Slide from '@material-ui/core/Slide';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { UnControlled as CodeMirror } from 'react-codemirror2';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';

interface PipelineDetailsState {
  graph?: dagre.graphlib.Graph;
  selectedNodeInfo: JSX.Element | null;
  pipeline: ApiPipeline | null;
  selectedTab: number;
  selectedNodeId: string;
  sidepanelBusy: boolean;
  summaryShown: boolean;
  template?: Workflow;
  templateYaml?: string;
}

const css = stylesheet({
  closeButton: {
    color: color.inactive,
    margin: 15,
    minHeight: 0,
    minWidth: 0,
    padding: 0,
  },
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
    zIndex: 2,
  },
  summaryCard: {
    backgroundColor: color.background,
    bottom: 20,
    left: 20,
    maxHeight: 350,
    overflow: 'auto',
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
      sidepanelBusy: false,
      summaryShown: true,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: this._createNewExperiment.bind(this),
        id: 'startNewExperimentBtn',
        // TODO: should be primary.
        outlined: true,
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
      breadcrumbs: [
        { displayName: 'Pipelines', href: RoutePage.PIPELINES },
        { displayName: this.props.match.params[RouteParams.pipelineId], href: '' }
      ],
    };
  }

  public render(): JSX.Element {
    const { pipeline, selectedNodeInfo, selectedNodeId, selectedTab, summaryShown, templateYaml } = this.state;

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
                  <Paper className={css.summaryCard}>
                    <div style={{ alignItems: 'baseline', display: 'flex', justifyContent: 'space-between' }}>
                      <div className={commonCss.header}>
                        Summary
                      </div>
                      <Button onClick={() => this.setState({ summaryShown: !summaryShown })} color='secondary'>
                        {summaryShown ? 'Hide' : 'Show'}
                      </Button>
                    </div>
                    <Collapse in={summaryShown}>
                      <div className={css.summaryKey}>Uploaded on</div>
                      <div>{new Date(pipeline.created_at!).toLocaleString()}</div>
                      <div className={css.summaryKey}>Description</div>
                      <div>{pipeline.description}</div>
                    </Collapse>
                  </Paper>
                  <Graph graph={this.state.graph} selectedNodeId={selectedNodeId} onClick={(id) => this._selectNode(id)} />
                  <Slide in={!!selectedNodeId} direction='left'>
                    <Resizable className={css.sidepane} defaultSize={{ width: '40%' }} maxWidth='90%'
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
                      {!!selectedNodeId && <div className={commonCss.page}>
                        <div className={commonCss.flex}>
                          <Button className={css.closeButton}
                            onClick={() => this.setState({ selectedNodeId: '' })}>
                            <CloseIcon />
                          </Button>
                          <div className={css.nodeName}>{selectedNodeId}</div>
                        </div>
                        <div className={commonCss.page}>

                          {this.state.sidepanelBusy &&
                            <CircularProgress size={30} className={commonCss.absoluteCenter} />}

                          <div className={commonCss.page}>
                            {selectedNodeInfo && <div className={padding(20, 'lr')}>
                              {this.state.selectedNodeInfo}
                            </div>}
                          </div>
                        </div>
                      </div>}
                    </Resizable>
                  </Slide>
                </div>}
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
                      readOnly: 'nocursor',
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

    // TODO: Show spinner while waiting for responses
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

      const breadcrumbs = [
        { displayName: 'Pipelines', href: RoutePage.PIPELINES },
        { displayName: pipeline.name!, href: '' },
      ];

      const toolbarActions = [...this.props.toolbarProps.actions];
      toolbarActions[0].disabled = false;
      this.props.updateToolbar({ breadcrumbs, actions: toolbarActions });

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

  private _selectNode(id: string): void {
    let nodeInfoJsx: JSX.Element = <div>Unable to retrieve node info</div>;
    const nodeInfo = StaticGraphParser.getNodeInfo(this.state.template, id);

    if (!nodeInfo) {
      logger.error(`Node with ID: ${id} was not found in the graph`);
      return;
    }

    switch (nodeInfo.nodeType) {
      case 'container':
        if (nodeInfo.containerInfo) {
          // TODO: The headers for these DetailsTables should just be a part of DetailsTables
          nodeInfoJsx =
            <div>
              <div className={commonCss.header}>Input parameters</div>
              <DetailsTable fields={nodeInfo.containerInfo.inputs} />

              <div className={commonCss.header}>Output parameters</div>
              <DetailsTable fields={nodeInfo.containerInfo.outputs} />

              <div className={commonCss.header}>Arguments</div>
              {nodeInfo.containerInfo.args.map((arg, i) =>
                <div key={i} style={{ fontFamily: 'mono' }}>{arg}</div>)}

              <div className={commonCss.header}>Command</div>
              {nodeInfo.containerInfo.command.map((c, i) => <div key={i}>{c}</div>)}

              <div className={commonCss.header}>Image</div>
              <div>{nodeInfo.containerInfo.image}</div>
            </div>;
        }
        break;
      case 'steps':
        if (nodeInfo.stepsInfo) {
          nodeInfoJsx =
            <div>
              <div className={commonCss.header}>Conditional</div>
              <div>{nodeInfo.stepsInfo.conditional}</div>

              <div className={commonCss.header}>Parameters</div>
              <DetailsTable fields={nodeInfo.stepsInfo.parameters} />
            </div>;
        }
        break;
      default:
        // TODO: display using error banner within side panel.
        nodeInfoJsx = <div>{`Node ${id} has unknown node type.`}</div>;
        logger.error(`Node ${id} has unknown node type.`);
    }

    this.setState({
      selectedNodeId: id,
      selectedNodeInfo: nodeInfoJsx,
    });
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      // TODO: Show spinner during wait.
      try {
        await Apis.pipelineServiceApi.deletePipeline(this.state.pipeline!.id!);
        // TODO: add success notification
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
