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
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { classes } from 'typestyle';
import { Workflow } from 'src/third_party/mlmd/argo_template';
import { ApiGetTemplateResponse, ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import {
  V2beta1ListPipelineVersionsResponse,
  V2beta1Pipeline,
  V2beta1PipelineVersion,
} from 'src/apisv2beta1/pipeline';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import RunUtils from 'src/lib/RunUtils';
import * as StaticGraphParser from 'src/lib/StaticGraphParser';
import { compareGraphEdges, transitiveReduction } from 'src/lib/StaticGraphParser';
import { URLParser } from 'src/lib/URLParser';
import { logger } from 'src/lib/Utils';
import { Page } from './Page';
import PipelineDetailsV1 from './PipelineDetailsV1';
import PipelineDetailsV2 from './PipelineDetailsV2';
import { ApiRunDetail } from 'src/apis/run';
import { ApiJob } from 'src/apis/job';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

interface PipelineDetailsState {
  graph: dagre.graphlib.Graph | null;
  reducedGraph: dagre.graphlib.Graph | null;
  graphV2: PipelineFlowElement[] | null;
  graphIsLoading: boolean;
  v1Pipeline: ApiPipeline | null;
  v2Pipeline: V2beta1Pipeline | null;
  selectedNodeInfo: JSX.Element | null;
  v1SelectedVersion?: ApiPipelineVersion;
  v2SelectedVersion?: V2beta1PipelineVersion;
  template?: Workflow;
  templateString?: string;
  v1Versions: ApiPipelineVersion[];
  v2Versions: V2beta1PipelineVersion[];
}

type Origin = {
  isRecurring: boolean;
  runId: string | null;
  recurringRunId: string | null;
  v1Run?: ApiRunDetail;
  v2Run?: V2beta1Run;
  v1RecurringRun?: ApiJob;
  v2RecurringRun?: V2beta1RecurringRun;
};

class PipelineDetails extends Page<{}, PipelineDetailsState> {
  constructor(props: any) {
    super(props);

    this.state = {
      graph: null,
      reducedGraph: null,
      graphV2: null,
      graphIsLoading: true,
      v1Pipeline: null,
      v2Pipeline: null,
      selectedNodeInfo: null,
      v1Versions: [],
      v2Versions: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const origin = this.getOrigin();
    const pipelineIdFromParams = this.props.match.params[RouteParams.pipelineId];
    const pipelineVersionIdFromParams = this.props.match.params[RouteParams.pipelineVersionId];

    if (origin) {
      const getOriginIdList = () => [origin.isRecurring ? origin.recurringRunId! : origin.runId!];
      origin.isRecurring
        ? buttons.cloneRecurringRun(getOriginIdList, true)
        : buttons.cloneRun(getOriginIdList, true);

      return {
        actions: buttons.getToolbarActionMap(),
        breadcrumbs: [
          {
            displayName: origin.isRecurring ? origin.recurringRunId! : origin.runId!,
            href: origin.isRecurring
              ? RoutePage.RECURRING_RUN_DETAILS.replace(
                  ':' + RouteParams.recurringRunId,
                  origin.recurringRunId!,
                )
              : RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, origin.runId!),
          },
        ],
        pageTitle: 'Pipeline details',
      };
    } else {
      // Add buttons for creating experiment and deleting pipeline version
      buttons
        .newRunFromPipelineVersion(
          () => {
            return this.state.v2Pipeline
              ? this.state.v2Pipeline.pipeline_id
              : pipelineIdFromParams
              ? pipelineIdFromParams
              : '';
          },
          () => {
            return this.state.v2SelectedVersion
              ? this.state.v2SelectedVersion.pipeline_version_id
              : pipelineVersionIdFromParams
              ? pipelineVersionIdFromParams
              : '';
          },
        )
        .newPipelineVersion('Upload version', () =>
          pipelineIdFromParams ? pipelineIdFromParams : '',
        )
        .newExperiment(() =>
          this.state.v1Pipeline
            ? this.state.v1Pipeline.id!
            : pipelineIdFromParams
            ? pipelineIdFromParams
            : '',
        )
        .deletePipelineVersion(
          () =>
            pipelineIdFromParams && pipelineVersionIdFromParams
              ? new Map<string, string>([[pipelineVersionIdFromParams, pipelineIdFromParams]])
              : new Map<string, string>(),
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
      v1Pipeline,
      v2Pipeline,
      v1SelectedVersion,
      v2SelectedVersion,
      v1Versions,
      v2Versions,
      graph,
      graphV2,
      reducedGraph,
      templateString,
    } = this.state;

    const setLayers = (layers: string[]) => {
      if (!templateString) {
        console.warn('pipeline spec template is unknown.');
        return;
      }
      const pipelineSpec = convertYamlToV2PipelineSpec(templateString!);
      const newElements = convertSubDagToFlowElements(pipelineSpec!, layers);
      this.setStateSafe({ graphV2: newElements, graphIsLoading: false });
    };

    const showV2Pipeline =
      isFeatureEnabled(FeatureKey.V2_ALPHA) && graphV2 && graphV2.length > 0 && !graph;
    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        {this.state.graphIsLoading && <div>Currently loading pipeline information</div>}
        {!this.state.graphIsLoading && showV2Pipeline && (
          <PipelineDetailsV2
            templateString={templateString}
            pipelineFlowElements={graphV2!}
            setSubDagLayers={setLayers}
            pipeline={v2Pipeline}
            selectedVersion={v2SelectedVersion}
            versions={v2Versions}
            handleVersionSelected={this.handleVersionSelected.bind(this)}
          />
        )}
        {!this.state.graphIsLoading && !showV2Pipeline && (
          <PipelineDetailsV1
            pipeline={v1Pipeline}
            templateString={templateString}
            graph={graph}
            reducedGraph={reducedGraph}
            updateBanner={this.props.updateBanner}
            selectedVersion={v1SelectedVersion}
            versions={v1Versions}
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

  private getOrigin() {
    const urlParser = new URLParser(this.props);
    const fromRunId = urlParser.get(QUERY_PARAMS.fromRunId);
    const fromRecurringRunId = urlParser.get(QUERY_PARAMS.fromRecurringRunId);

    if (fromRunId && fromRecurringRunId) {
      throw new Error('The existence of run and recurring run should be exclusive.');
    }

    let origin: Origin = {
      isRecurring: !!fromRecurringRunId,
      runId: fromRunId,
      recurringRunId: fromRecurringRunId,
    };
    return fromRunId || fromRecurringRunId ? origin : undefined;
  }

  private async getTempStrFromRunOrRecurringRun(existingObj: V2beta1Run | V2beta1RecurringRun) {
    // existing run or recurring run have two kinds of resources that can provide template string

    // 1. Pipeline and pipeline version id
    const pipelineId = existingObj.pipeline_version_reference?.pipeline_id;
    const pipelineVersionId = existingObj.pipeline_version_reference?.pipeline_version_id;
    let templateStrFromOrigin: string | undefined;
    if (pipelineId && pipelineVersionId) {
      const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
        pipelineId,
        pipelineVersionId,
      );
      const pipelineSpecFromVersion = pipelineVersion.pipeline_spec;
      templateStrFromOrigin = pipelineSpecFromVersion
        ? JsYaml.safeDump(pipelineSpecFromVersion)
        : '';
    }

    // 2. Pipeline_spec
    let pipelineManifest: string | undefined;
    if (existingObj.pipeline_spec) {
      pipelineManifest = JsYaml.safeDump(existingObj.pipeline_spec);
    }

    return pipelineManifest ?? templateStrFromOrigin;
  }

  // We don't have default version in v2 pipeline proto, choose the latest version instead.
  private async getSelectedVersion(pipelineId: string, versionId?: string) {
    let selectedVersion: V2beta1PipelineVersion;
    // Get specific version if version id is provided
    if (versionId) {
      try {
        selectedVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(pipelineId, versionId);
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError('Cannot retrieve pipeline version.', err);
        logger.error('Cannot retrieve pipeline version.', err);
        return undefined;
      }
    } else {
      // Get the latest version if no version id
      let listVersionsResponse: V2beta1ListPipelineVersionsResponse;
      try {
        listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
          pipelineId,
          undefined,
          1, // Only need the latest one
          'created_at desc',
        );

        if (
          listVersionsResponse.pipeline_versions &&
          listVersionsResponse.pipeline_versions.length > 0
        ) {
          selectedVersion = listVersionsResponse.pipeline_versions[0];
        } else {
          return undefined;
        }
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError('Cannot retrieve pipeline version list.', err);
        logger.error('Cannot retrieve pipeline version list.', err);
        return undefined;
      }
    }
    return selectedVersion;
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const origin = this.getOrigin();

    let v1Pipeline: ApiPipeline | null = null;
    let v1Version: ApiPipelineVersion | null = null;
    let v1SelectedVersion: ApiPipelineVersion | undefined;
    let v1Versions: ApiPipelineVersion[] = [];

    let v2Pipeline: V2beta1Pipeline | null = null;
    let v2SelectedVersion: V2beta1PipelineVersion | undefined;
    let v2Versions: V2beta1PipelineVersion[] = [];

    let templateString = '';
    let breadcrumbs: Array<{ displayName: string; href: string }> = [];
    const toolbarActions = this.props.toolbarProps.actions;
    let pageTitle = '';

    // If fromRunId or fromRecurringRunId is specified,
    // then load the run and get the pipeline template from it
    if (origin) {
      const msgRunOrRecurringRun = origin.isRecurring ? 'recurring run' : 'run';
      try {
        // TODO(jlyaoyuli): change to v2 API after v1 is deprecated
        // Note: v2 getRecurringRun() api can retrieve v1 job
        // (ApiParameter can be only retrieve in the response of v1 getJob() api)
        // Experiment ID and pipeline version id are preserved.
        if (origin.isRecurring) {
          origin.v1RecurringRun = await Apis.jobServiceApi.getJob(origin.recurringRunId!);
          origin.v2RecurringRun = await Apis.recurringRunServiceApi.getRecurringRun(
            origin.recurringRunId!,
          );
        } else {
          origin.v1Run = await Apis.runServiceApi.getRun(origin.runId!);
          origin.v2Run = await Apis.runServiceApiV2.getRun(origin.runId!);
        }

        // If v2 run or recurring is existing, get template string from it
        const templateStrFromOrigin = origin.isRecurring
          ? await this.getTempStrFromRunOrRecurringRun(origin.v2RecurringRun!)
          : await this.getTempStrFromRunOrRecurringRun(origin.v2Run!);

        // V1: Convert the run's pipeline spec to YAML to be displayed as the pipeline's source.
        // V2: Directly Use the template string from existing run or recurring run
        // because it can be translated in JSON format.
        if (isFeatureEnabled(FeatureKey.V2_ALPHA) && templateStrFromOrigin) {
          templateString = templateStrFromOrigin;
        } else {
          try {
            const workflowManifestString =
              RunUtils.getWorkflowManifest(
                origin.isRecurring ? origin.v1RecurringRun : origin.v1Run!.run,
              ) || '';
            const workflowManifest = JSON.parse(workflowManifestString || '{}');
            try {
              templateString = WorkflowUtils.isPipelineSpec(workflowManifestString)
                ? workflowManifestString
                : JsYaml.safeDump(workflowManifest);
            } catch (err) {
              this.setStateSafe({ graphIsLoading: false });
              await this.showPageError(
                `Failed to parse pipeline spec from ${msgRunOrRecurringRun} with ID: ${
                  origin.isRecurring ? origin.v1RecurringRun!.id : origin.v1Run!.run!.id
                }.`,
                err,
              );
              logger.error(
                `Failed to convert pipeline spec YAML from ${msgRunOrRecurringRun} with ID: ${
                  origin.isRecurring ? origin.v1RecurringRun!.id : origin.v1Run!.run!.id
                }.`,
                err,
              );
            }
          } catch (err) {
            this.setStateSafe({ graphIsLoading: false });
            await this.showPageError(
              `Failed to parse pipeline spec from ${msgRunOrRecurringRun} with ID: ${
                origin.isRecurring ? origin.v1RecurringRun!.id : origin.v1Run!.run!.id
              }.`,
              err,
            );
            logger.error(
              `Failed to parse pipeline spec JSON from ${msgRunOrRecurringRun} with ID: ${
                origin.isRecurring ? origin.v1RecurringRun!.id : origin.v1Run!.run!.id
              }.`,
              err,
            );
          }
        }

        // We have 2 options to get experiment id (resource_ref in v1, experiment_id in v2)
        // which is used in getExperiment(). Getting the experiment id from v2 API works well
        // for any runs created via v1 api v2 APIs. Therefore, choosing v2 API is feasible and also
        // makes the API integration more comprehensively.
        const relatedExperimentId = origin.isRecurring
          ? origin.v2RecurringRun?.experiment_id
          : origin.v2Run?.experiment_id;
        let experiment: V2beta1Experiment | undefined;
        if (relatedExperimentId) {
          experiment = await Apis.experimentServiceApiV2.getExperiment(relatedExperimentId);
        }

        // Build the breadcrumbs, by adding experiment and run names
        if (experiment) {
          breadcrumbs.push(
            { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: experiment.display_name!,
              href: RoutePage.EXPERIMENT_DETAILS.replace(
                ':' + RouteParams.experimentId,
                experiment.experiment_id!,
              ),
            },
          );
        } else {
          breadcrumbs.push({
            displayName: `All ${msgRunOrRecurringRun}s`,
            href: origin.isRecurring ? RoutePage.RECURRING_RUNS : RoutePage.RUNS,
          });
        }
        breadcrumbs.push({
          displayName: origin.isRecurring
            ? origin.v2RecurringRun!.display_name!
            : origin.v2Run!.display_name!,
          href: origin.isRecurring
            ? RoutePage.RECURRING_RUN_DETAILS.replace(
                ':' + RouteParams.recurringRunId,
                origin.recurringRunId!,
              )
            : RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, origin.runId!),
        });
        pageTitle = 'Pipeline details';
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError(`Cannot retrieve ${msgRunOrRecurringRun} details.`, err);
        logger.error(`Cannot retrieve ${msgRunOrRecurringRun} details.`, err);
        return;
      }
    } else {
      // if fromRunId or fromRecurringRunId is not specified, then we have a full pipeline
      const pipelineId = this.props.match.params[RouteParams.pipelineId];
      const versionId = this.props.match.params[RouteParams.pipelineVersionId];

      try {
        v1Pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
        v2Pipeline = await Apis.pipelineServiceApiV2.getPipeline(pipelineId);
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError('Cannot retrieve pipeline details.', err);
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      try {
        // TODO(rjbauer): it's possible we might not have a version, even default
        if (versionId) {
          v1Version = await Apis.pipelineServiceApi.getPipelineVersion(versionId);
        }
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError('Cannot retrieve pipeline version.', err);
        logger.error('Cannot retrieve pipeline version.', err);
        return;
      }

      v1SelectedVersion = versionId ? v1Version! : v1Pipeline.default_version;
      v2SelectedVersion = await this.getSelectedVersion(pipelineId, versionId);

      if (!v1SelectedVersion && !v2SelectedVersion) {
        // An empty pipeline, which doesn't have any version.
        pageTitle = v2Pipeline.display_name!;
        const actions = this.props.toolbarProps.actions;
        actions[ButtonKeys.DELETE_RUN].disabled = true;
        this.props.updateToolbar({ actions });
      } else {
        // Fetch manifest for the selected version under this pipeline.
        // Basically, v1 and v2 selectedVersion are existing simultanesouly
        // Set v2 has higher priority is only for full migration after v1 is deprecated
        pageTitle = v2SelectedVersion
          ? v2Pipeline.display_name!.concat(' (', v2SelectedVersion!.display_name!, ')')
          : v1Pipeline.name!.concat(' (', v1SelectedVersion!.name!, ')');
        try {
          // TODO(jingzhang36): pagination not proper here. so if many versions,
          // the page size value should be?
          v1Versions =
            (
              await Apis.pipelineServiceApi.listPipelineVersions(
                'PIPELINE',
                pipelineId,
                50,
                undefined,
                'created_at desc',
              )
            ).versions || [];

          v2Versions =
            (
              await Apis.pipelineServiceApiV2.listPipelineVersions(
                pipelineId,
                undefined,
                50,
                'created_at desc',
              )
            ).pipeline_versions || [];
        } catch (err) {
          this.setStateSafe({ graphIsLoading: false });
          await this.showPageError('Cannot retrieve pipeline versions.', err);
          logger.error('Cannot retrieve pipeline versions.', err);
          return;
        }
        templateString = await this._getTemplateString(v2SelectedVersion);
      }

      breadcrumbs = [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }];
    }

    this.props.updateToolbar({ breadcrumbs, actions: toolbarActions, pageTitle });

    const [graph, reducedGraph, graphV2] = await this._createGraph(templateString);

    // Currently, we allow upload v1 version to v2 pipeline or vice versa,
    // so v1 and v2 field is "non-exclusive".
    // TODO(jlyaoyuli): If we decide not to support "mix versions",
    // v1 and v2 should be "exclusive"
    if (isFeatureEnabled(FeatureKey.V2_ALPHA) && graphV2.length > 0) {
      this.setStateSafe({
        v1Pipeline,
        v2Pipeline,
        v1SelectedVersion,
        v2SelectedVersion,
        v1Versions,
        v2Versions,
        graph: undefined,
        graphV2,
        graphIsLoading: false,
        reducedGraph: undefined,
        templateString,
      });
    } else {
      this.setStateSafe({
        v1Pipeline,
        v2Pipeline,
        v1SelectedVersion,
        v2SelectedVersion,
        v1Versions,
        v2Versions,
        graph,
        graphV2: undefined,
        graphIsLoading: false,
        reducedGraph,
        templateString,
      });
    }
  }

  public async handleVersionSelected(versionId: string): Promise<void> {
    if (this.state.v2Pipeline) {
      const v1SelectedVersion = (this.state.v1Versions || []).find(v => v.id === versionId);
      const v2SelectedVersion = (this.state.v2Versions || []).find(
        v => v.pipeline_version_id === versionId,
      );
      const pageTitle = this.state.v2Pipeline.display_name?.concat(
        ' (',
        v2SelectedVersion?.display_name!,
        ')',
      );

      const selectedVersionPipelineTemplate = await this._getTemplateString(v2SelectedVersion);
      this.props.history.replace({
        pathname: `/pipelines/details/${this.state.v2Pipeline.pipeline_id}/version/${versionId}`,
      });
      this.props.updateToolbar(this.getInitialToolbarState());
      this.props.updateToolbar({ pageTitle });

      const [graph, reducedGraph, graphV2] = await this._createGraph(
        selectedVersionPipelineTemplate,
      );
      if (isFeatureEnabled(FeatureKey.V2_ALPHA) && graphV2.length > 0) {
        this.setStateSafe({
          graph: undefined,
          reducedGraph: undefined,
          graphV2,
          graphIsLoading: false,
          v2SelectedVersion,
          templateString: selectedVersionPipelineTemplate,
        });
      } else {
        this.setStateSafe({
          graph,
          reducedGraph,
          graphV2: undefined,
          graphIsLoading: false,
          v1SelectedVersion,
          templateString: selectedVersionPipelineTemplate,
        });
      }
    }
  }

  private async _getTemplateString(pipelineVersion?: V2beta1PipelineVersion): Promise<string> {
    if (pipelineVersion?.pipeline_spec) {
      return JsYaml.safeDump(pipelineVersion.pipeline_spec);
    }

    // Handle v1 pipelines created by v1 API (no pipeline_spec field)
    let v1TemplateResponse: ApiGetTemplateResponse;
    if (pipelineVersion?.pipeline_version_id) {
      v1TemplateResponse = await Apis.pipelineServiceApi.getPipelineVersionTemplate(
        pipelineVersion.pipeline_version_id,
      );
      return v1TemplateResponse.template || '';
    } else if (pipelineVersion?.pipeline_id) {
      v1TemplateResponse = await Apis.pipelineServiceApi.getTemplate(pipelineVersion.pipeline_id);
      return v1TemplateResponse.template || '';
    } else {
      logger.error('No template string is found');
      return '';
    }
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
        } else if (isFeatureEnabled(FeatureKey.V2_ALPHA)) {
          const pipelineSpec = WorkflowUtils.convertYamlToV2PipelineSpec(templateString);
          graphV2 = convertFlowElements(pipelineSpec);
        } else {
          throw new Error(
            'Unable to convert string response from server to Argo workflow template' +
              ': https://argoproj.github.io/argo-workflows/workflow-templates/',
          );
        }
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
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
