/*
 * Copyright 2018 The Kubeflow Authors
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

import 'brace';
import 'brace/ext/language_tools';
import 'brace/mode/yaml';
import 'brace/theme/github';
import { graphlib } from 'dagre';
import * as JsYaml from 'js-yaml';
import * as React from 'react';
import { FeatureKey, isFeatureEnabled } from 'src/features';
import { Apis } from 'src/lib/Apis';
import {
  convertFlowElements,
  convertSubDagToFlowElements,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { convertJsonToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { classes } from 'typestyle';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { ApiExperiment } from '../apis/experiment';
import { ApiGetTemplateResponse, ApiPipeline, ApiPipelineVersion } from '../apis/pipeline';
import { QUERY_PARAMS, RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import Buttons, { ButtonKeys } from '../lib/Buttons';
import RunUtils from '../lib/RunUtils';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import { compareGraphEdges, transitiveReduction } from '../lib/StaticGraphParser';
import { URLParser } from '../lib/URLParser';
import { logger } from '../lib/Utils';
import { Page } from './Page';
import PipelineDetailsV1 from './PipelineDetailsV1';
import PipelineDetailsV2 from './PipelineDetailsV2';

interface PipelineDetailsState {
  graph: dagre.graphlib.Graph | null;
  reducedGraph: dagre.graphlib.Graph | null;
  graphV2: PipelineFlowElement[] | null;
  pipeline: ApiPipeline | null;
  selectedNodeInfo: JSX.Element | null;
  selectedVersion?: ApiPipelineVersion;
  template?: Workflow;
  templateString?: string;
  versions: ApiPipelineVersion[];
}

class PipelineDetails extends Page<{}, PipelineDetailsState> {
  constructor(props: any) {
    super(props);

    this.state = {
      graph: null,
      reducedGraph: null,
      graphV2: null,
      pipeline: null,
      selectedNodeInfo: null,
      versions: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const fromRunId = new URLParser(this.props).get(QUERY_PARAMS.fromRunId);
    const pipelineIdFromParams = this.props.match.params[RouteParams.pipelineId];
    const pipelineVersionIdFromParams = this.props.match.params[RouteParams.pipelineVersionId];
    buttons
      .newRunFromPipelineVersion(
        () => {
          return pipelineIdFromParams ? pipelineIdFromParams : '';
        },
        () => {
          return pipelineVersionIdFromParams ? pipelineVersionIdFromParams : '';
        },
      )
      .newPipelineVersion('Upload version', () =>
        pipelineIdFromParams ? pipelineIdFromParams : '',
      );

    if (fromRunId) {
      return {
        actions: buttons.getToolbarActionMap(),
        breadcrumbs: [
          {
            displayName: fromRunId,
            href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, fromRunId),
          },
        ],
        pageTitle: 'Pipeline details',
      };
    } else {
      // Add buttons for creating experiment and deleting pipeline version
      buttons
        .newExperiment(() =>
          this.state.pipeline
            ? this.state.pipeline.id!
            : pipelineIdFromParams
            ? pipelineIdFromParams
            : '',
        )
        .delete(
          () => (pipelineVersionIdFromParams ? [pipelineVersionIdFromParams] : []),
          'pipeline version',
          this._deleteCallback.bind(this),
          pipelineVersionIdFromParams ? true : false /* useCurrentResource */,
        );
      return {
        actions: buttons.getToolbarActionMap(),
        breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
        pageTitle: this.props.match.params[RouteParams.pipelineId],
      };
    }
  }

  public render(): JSX.Element {
    const {
      pipeline,
      selectedVersion,
      templateString,
      versions,
      graph,
      graphV2,
      reducedGraph,
    } = this.state;

    const setLayers = (layers: string[]) => {
      if (!templateString) {
        console.warn('pipeline spec template is unknown.');
        return;
      }
      const pipelineSpec = convertJsonToV2PipelineSpec(templateString!);
      const newElements = convertSubDagToFlowElements(pipelineSpec!, layers);
      this.setStateSafe({ graphV2: newElements });
    };

    const showV2Pipeline =
      isFeatureEnabled(FeatureKey.V2) && graphV2 && graphV2.length > 0 && !graph;
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        {showV2Pipeline && (
          <PipelineDetailsV2
            templateString={templateString}
            pipelineFlowElements={graphV2!}
            setSubDagLayers={setLayers}
            apiPipeline={pipeline}
            selectedVersion={selectedVersion}
            versions={versions}
            handleVersionSelected={this.handleVersionSelected.bind(this)}
          />
        )}
        {!showV2Pipeline && (
          <PipelineDetailsV1
            pipeline={pipeline}
            templateString={templateString}
            graph={graph}
            reducedGraph={reducedGraph}
            updateBanner={this.props.updateBanner}
            selectedVersion={selectedVersion}
            versions={versions}
            handleVersionSelected={this.handleVersionSelected.bind(this)}
          />
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
    const fromRunId = new URLParser(this.props).get(QUERY_PARAMS.fromRunId);

    let pipeline: ApiPipeline | null = null;
    let version: ApiPipelineVersion | null = null;
    let templateString = '';
    let breadcrumbs: Array<{ displayName: string; href: string }> = [];
    const toolbarActions = this.props.toolbarProps.actions;
    let pageTitle = '';
    let selectedVersion: ApiPipelineVersion | undefined;
    let versions: ApiPipelineVersion[] = [];

    // If fromRunId is specified, load the run and get the pipeline template from it
    if (fromRunId) {
      try {
        const runDetails = await Apis.runServiceApi.getRun(fromRunId);

        // V1: Convert the run's pipeline spec to YAML to be displayed as the pipeline's source.
        // V2: Use the pipeline spec string directly because it can be translated in JSON format.
        try {
          const workflowManifestString = RunUtils.getWorkflowManifest(runDetails.run) || '';
          const workflowManifest = JSON.parse(workflowManifestString || '{}');
          try {
            if (WorkflowUtils.isPipelineSpec(workflowManifestString)) {
              templateString = workflowManifestString;
            } else {
              templateString = JsYaml.safeDump(workflowManifest);
            }
          } catch (err) {
            await this.showPageError(
              `Failed to parse pipeline spec from run with ID: ${runDetails.run!.id}.`,
              err,
            );
            logger.error(
              `Failed to convert pipeline spec YAML from run with ID: ${runDetails.run!.id}.`,
              err,
            );
          }
        } catch (err) {
          await this.showPageError(
            `Failed to parse pipeline spec from run with ID: ${runDetails.run!.id}.`,
            err,
          );
          logger.error(
            `Failed to parse pipeline spec JSON from run with ID: ${runDetails.run!.id}.`,
            err,
          );
        }

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
              href: RoutePage.EXPERIMENT_DETAILS.replace(
                ':' + RouteParams.experimentId,
                experiment.id!,
              ),
            },
          );
        } else {
          breadcrumbs.push({ displayName: 'All runs', href: RoutePage.RUNS });
        }
        breadcrumbs.push({
          displayName: runDetails.run!.name!,
          href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, fromRunId),
        });
        pageTitle = 'Pipeline details';
      } catch (err) {
        await this.showPageError('Cannot retrieve run details.', err);
        logger.error('Cannot retrieve run details.', err);
        return;
      }
    } else {
      // if fromRunId is not specified, then we have a full pipeline
      const pipelineId = this.props.match.params[RouteParams.pipelineId];

      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
      } catch (err) {
        await this.showPageError('Cannot retrieve pipeline details.', err);
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      const versionId = this.props.match.params[RouteParams.pipelineVersionId];

      try {
        // TODO(rjbauer): it's possible we might not have a version, even default
        if (versionId) {
          version = await Apis.pipelineServiceApi.getPipelineVersion(versionId);
        }
      } catch (err) {
        await this.showPageError('Cannot retrieve pipeline version.', err);
        logger.error('Cannot retrieve pipeline version.', err);
        return;
      }

      selectedVersion = versionId ? version! : pipeline.default_version;

      if (!selectedVersion) {
        // An empty pipeline, which doesn't have any version.
        pageTitle = pipeline.name!;
        const actions = this.props.toolbarProps.actions;
        actions[ButtonKeys.DELETE_RUN].disabled = true;
        this.props.updateToolbar({ actions });
      } else {
        // Fetch manifest for the selected version under this pipeline.
        pageTitle = pipeline.name!.concat(' (', selectedVersion!.name!, ')');
        try {
          // TODO(jingzhang36): pagination not proper here. so if many versions,
          // the page size value should be?
          versions =
            (
              await Apis.pipelineServiceApi.listPipelineVersions(
                'PIPELINE',
                pipelineId,
                50,
                undefined,
                'created_at desc',
              )
            ).versions || [];
        } catch (err) {
          await this.showPageError('Cannot retrieve pipeline versions.', err);
          logger.error('Cannot retrieve pipeline versions.', err);
          return;
        }
        templateString = await this._getTemplateString(pipelineId, versionId);
      }

      breadcrumbs = [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }];
    }

    this.props.updateToolbar({ breadcrumbs, actions: toolbarActions, pageTitle });

    const [graph, reducedGraph, graphV2] = await this._createGraph(templateString);

    if (isFeatureEnabled(FeatureKey.V2) && graphV2.length > 0) {
      this.setStateSafe({
        graph: undefined,
        reducedGraph: undefined,
        graphV2,
        pipeline,
        selectedVersion,
        templateString,
        versions,
      });
    } else {
      this.setStateSafe({
        graph,
        reducedGraph,
        graphV2: undefined,
        pipeline,
        selectedVersion,
        templateString,
        versions,
      });
    }
  }

  public async handleVersionSelected(versionId: string): Promise<void> {
    if (this.state.pipeline) {
      const selectedVersion = (this.state.versions || []).find(v => v.id === versionId);
      const selectedVersionPipelineTemplate = await this._getTemplateString(
        this.state.pipeline.id!,
        versionId,
      );
      this.props.history.replace({
        pathname: `/pipelines/details/${this.state.pipeline.id}/version/${versionId}`,
      });

      const [graph, reducedGraph, graphV2] = await this._createGraph(
        selectedVersionPipelineTemplate,
      );
      if (isFeatureEnabled(FeatureKey.V2) && graphV2.length > 0) {
        this.setStateSafe({
          graph: undefined,
          reducedGraph: undefined,
          graphV2,
          selectedVersion,
          templateString: selectedVersionPipelineTemplate,
        });
      } else {
        this.setStateSafe({
          graph,
          reducedGraph,
          graphV2: undefined,
          selectedVersion,
          templateString: selectedVersionPipelineTemplate,
        });
      }
    }
  }

  private async _getTemplateString(pipelineId: string, versionId: string): Promise<string> {
    try {
      let templateResponse: ApiGetTemplateResponse;
      if (versionId) {
        templateResponse = await Apis.pipelineServiceApi.getPipelineVersionTemplate(versionId);
      } else {
        templateResponse = await Apis.pipelineServiceApi.getTemplate(pipelineId);
      }
      return templateResponse.template || '';
    } catch (err) {
      await this.showPageError('Cannot retrieve pipeline template.', err);
      logger.error('Cannot retrieve pipeline details.', err);
    }
    return '';
  }

  private async _createGraph(
    templateString: string,
  ): Promise<
    [dagre.graphlib.Graph | null, dagre.graphlib.Graph | null | undefined, PipelineFlowElement[]]
  > {
    let graph: graphlib.Graph | null = null;
    let reducedGraph: graphlib.Graph | null | undefined = null;
    let graphV2: PipelineFlowElement[] = [];
    if (templateString) {
      try {
        const template = JsYaml.safeLoad(templateString);
        if (WorkflowUtils.isArgoWorkflowTemplate(template)) {
          graph = StaticGraphParser.createGraph(template!);

          reducedGraph = graph ? transitiveReduction(graph) : undefined;
          if (graph && reducedGraph && compareGraphEdges(graph, reducedGraph)) {
            reducedGraph = undefined; // disable reduction switch
          }
        } else if (isFeatureEnabled(FeatureKey.V2)) {
          const pipelineSpec = WorkflowUtils.convertJsonToV2PipelineSpec(templateString);
          graphV2 = convertFlowElements(pipelineSpec);
        } else {
          throw new Error(
            'Unable to convert string response from server to Argo workflow template' +
              ': https://argoproj.github.io/argo-workflows/workflow-templates/',
          );
        }
      } catch (err) {
        await this.showPageError('Error: failed to generate Pipeline graph.', err);
      }
    }
    return [graph, reducedGraph, graphV2];
  }

  private _deleteCallback(_: string[], success: boolean): void {
    if (success) {
      const breadcrumbs = this.props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length
        ? breadcrumbs[breadcrumbs.length - 1].href
        : RoutePage.PIPELINES;
      this.props.history.push(previousPage);
    }
  }
}

export default PipelineDetails;
