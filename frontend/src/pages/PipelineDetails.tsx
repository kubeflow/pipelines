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
import CircularProgress from '@material-ui/core/CircularProgress';
import CloseIcon from '@material-ui/icons/Close';
import DeleteIcon from '@material-ui/icons/Delete';
import DetailsTable from '../components/DetailsTable';
import Graph from '../components/Graph';
import MD2Tabs from '../atoms/MD2Tabs';
import Resizable from 're-resizable';
import Slide from '@material-ui/core/Slide';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { BannerProps } from '../components/Banner';
import { DialogProps, RoutePage, RouteParams } from '../components/Router';
import { Paper, Collapse } from '@material-ui/core';
import { RouteComponentProps } from 'react-router';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { UnControlled as CodeMirror } from 'react-codemirror2';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { color, commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { classes, stylesheet } from 'typestyle';

interface PipelineDetailsProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

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
    maxHeight: 250,
    padding: 20,
    position: 'absolute',
    width: 250,
    zIndex: 1,
  },
  summaryKey: {
    color: color.strong,
    marginTop: 10,
  },
});

class PipelineDetails extends React.Component<PipelineDetailsProps, PipelineDetailsState> {

  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: () => this._newJobClicked(),
      disabled: true,
      disabledTitle: 'Must have a Pipeline to create a Job',
      icon: AddIcon,
      id: 'newJobBtn',
      title: 'Create new Job',
      tooltip: 'Create a new Job from this Pipeline',
    },
    {
      action: () => this.props.updateDialog({
        buttons: [
          { onClick: () => this._deleteDialogClosed(true), text: 'Delete' },
          { onClick: () => this._deleteDialogClosed(false), text: 'Cancel' },
        ],
        onClose: () => this._deleteDialogClosed(false),
        title: 'Delete this Pipeline?',
      }),
      disabled: false,
      icon: DeleteIcon,
      id: 'deleteBtn',
      title: 'Delete',
      tooltip: 'Delete this pipeline',
    },
  ];

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

  public componentWillMount() {
    const { pipeline } = this.state;
    this.props.updateToolbar({
      actions: this._toolbarActions,
      breadcrumbs: [
        { displayName: 'Pipelines', href: RoutePage.PIPELINES },
        { displayName: pipeline && pipeline.name ? pipeline.name : this.props.match.params[RouteParams.pipelineId], href: '' }
      ],
    });
  }

  public componentDidMount(): void {
    this._loadPipeline();
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render(): JSX.Element {
    const { pipeline, selectedNodeInfo, selectedNodeId, selectedTab, summaryShown, templateYaml } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>

        {pipeline && (
          <div className={commonCss.page}>
            <MD2Tabs
              selectedTab={selectedTab}
              onSwitch={(tab: number) => this.setState({ selectedTab: tab })}
              tabs={['Graph', 'Config']}
            />
            <div className={commonCss.page}>
              {selectedTab === 0 && <div className={commonCss.page}>
                {this.state.graph && <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                  <Paper className={css.summaryCard}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
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
                {!this.state.graph && <span style={{margin: '40px auto'}}>No graph to show</span>}
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

  private async _loadPipeline(): Promise<void> {
    const pipelineId = this.props.match.params[RouteParams.pipelineId];
    // TODO: Show spinner while waiting for responses
    await Promise.all([
      Apis.pipelineServiceApi.getPipeline(pipelineId),
      Apis.pipelineServiceApi.getTemplate(pipelineId)
    ]).then(([pipeline, templateResponse]) => {
      try {
        const template: Workflow = JsYaml.safeLoad(templateResponse.template!);
        let g: dagre.graphlib.Graph | undefined;
        try {
          g = StaticGraphParser.createGraph(template);
        } catch (err) {
          this._handlePageError('Error: failed to generate Pipeline graph.', err.message);
        }
        this.setState({
          graph: g,
          pipeline,
          template,
          templateYaml: templateResponse.template,
        });
      } catch (err) {
        this.props.updateBanner({
          additionalInfo: err.message + '\n\n\n' + templateResponse.template,
          message: 'Failed to parse pipeline yaml',
          mode: 'error',
        });
      }
      const toolbarActions = [...this.props.toolbarProps.actions];
      toolbarActions[0].disabled = false;
      this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    })
      .catch((err) => {
        this._handlePageError(
          `Error: failed to retrieve pipeline or template for ID: ${pipelineId}.`,
          err.message,
          this._loadPipeline.bind(this),
        );
        logger.error(`Error loading pipeline or template for ID: ${pipelineId}`, err);
      });
  }

  private _handlePageError(message: string, error?: Error, refreshFunc?: () => void): void {
    this.props.updateBanner({
      additionalInfo: error ? error.message : undefined,
      message: message + ((error && error.message) ? ' Click Details for more information.' : ''),
      mode: 'error',
      refresh: refreshFunc,
    });
  }

  private _selectNode(id: string): void {
    let nodeInfoJsx: JSX.Element = <div>Unable to retrieve node info</div>;
    const nodeInfo = StaticGraphParser.getNodeInfo(this.state.template, id);

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

  private _newJobClicked(): void {
    const searchString = this.state.pipeline && this.state.pipeline.id ?
      new URLParser(this.props).build({ [QUERY_PARAMS.pipelineId]: this.state.pipeline.id }) : '';
    this.props.history.push(RoutePage.NEW_JOB + searchString);
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      // TODO: Show spinner during wait.
      try {
        await Apis.pipelineServiceApi.deletePipeline(this.state.pipeline!.id!);
        // TODO: add success notification
        this.props.history.push(RoutePage.PIPELINES);
      } catch (err) {
        this.props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: err.message,
          title: 'Failed to delete pipeline',
        });
        logger.error('Deleting pipeline failed with error:', err);
      }
    }
  }
}

// tslint:disable-next-line:no-default-export
export default PipelineDetails;
