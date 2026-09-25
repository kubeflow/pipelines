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

import 'ace-builds/src-noconflict/ace';
import 'ace-builds/src-noconflict/ext-language_tools';
import 'ace-builds/src-noconflict/mode-yaml';
import 'ace-builds/src-noconflict/theme-github';
import type * as React from 'react';
import { CircularProgress } from '@mui/material';
import * as JsYaml from 'js-yaml';
import { Apis } from 'src/lib/Apis';
import {
  convertFlowElements,
  convertSubDagToFlowElements,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { classes } from 'typestyle';
import {
  V2beta1ListPipelineVersionsResponse,
  V2beta1Pipeline,
  V2beta1PipelineVersion,
} from 'src/apisv2beta1/pipeline';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { URLParser } from 'src/lib/URLParser';
import { logger } from 'src/lib/Utils';
import { Page } from './Page';
import PipelineDetailsV2 from './PipelineDetailsV2';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

interface PipelineDetailsState {
  graphV2: PipelineFlowElement[] | null;
  graphIsLoading: boolean;
  v2Pipeline: V2beta1Pipeline | null;
  v2SelectedVersion?: V2beta1PipelineVersion;
  templateString?: string;
  v2Versions: V2beta1PipelineVersion[];
}

type Origin = {
  isRecurring: boolean;
  runId: string | null;
  recurringRunId: string | null;
  v2Run?: V2beta1Run;
  v2RecurringRun?: V2beta1RecurringRun;
};

class PipelineDetails extends Page<{}, PipelineDetailsState> {
  constructor(props: any) {
    super(props);

    this.state = {
      graphV2: null,
      graphIsLoading: true,
      v2Pipeline: null,
      v2Versions: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const origin = this.getOrigin();
    const pipelineIdFromParams = this.props.params[RouteParams.pipelineId] ?? '';
    const pipelineVersionIdFromParams = this.props.params[RouteParams.pipelineVersionId] ?? '';

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
              ? (this.state.v2Pipeline.pipeline_id ?? '')
              : pipelineIdFromParams
                ? pipelineIdFromParams
                : '';
          },
          () => {
            return this.state.v2SelectedVersion
              ? (this.state.v2SelectedVersion.pipeline_version_id ?? '')
              : pipelineVersionIdFromParams
                ? pipelineVersionIdFromParams
                : '';
          },
        )
        .newPipelineVersion('Upload version', () =>
          pipelineIdFromParams ? pipelineIdFromParams : '',
        )
        .newExperiment(() => this.state.v2Pipeline?.pipeline_id || pipelineIdFromParams)
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
        pageTitle: this.props.params[RouteParams.pipelineId] ?? '',
      };
    }
  }

  public render(): React.JSX.Element {
    const { v2Pipeline, v2SelectedVersion, v2Versions, graphV2, templateString } = this.state;

    const setLayers = (layers: string[]) => {
      if (!templateString) {
        console.warn('pipeline spec template is unknown.');
        return;
      }
      const pipelineSpec = convertYamlToV2PipelineSpec(templateString!);
      const newElements = convertSubDagToFlowElements(pipelineSpec!, layers);
      this.setStateSafe({ graphV2: newElements, graphIsLoading: false });
    };

    return (
      <div className={classes(commonCss.page, padding(20, 't'))}>
        {this.state.graphIsLoading && (
          <div style={{ textAlign: 'center', paddingTop: 40 }}>
            <CircularProgress />
            <div>Currently loading pipeline information</div>
          </div>
        )}
        {!this.state.graphIsLoading && (
          <PipelineDetailsV2
            key={v2SelectedVersion?.pipeline_version_id ?? templateString}
            templateString={templateString}
            pipelineFlowElements={graphV2 || []}
            setSubDagLayers={setLayers}
            pipeline={v2Pipeline}
            selectedVersion={v2SelectedVersion}
            versions={v2Versions}
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
    this._isMounted = true;
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
    if (existingObj.pipeline_spec) return JsYaml.dump(existingObj.pipeline_spec);

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
      templateStrFromOrigin = pipelineSpecFromVersion ? JsYaml.dump(pipelineSpecFromVersion) : '';
    }

    return templateStrFromOrigin;
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

    let v2Pipeline: V2beta1Pipeline | null = null;
    let v2SelectedVersion: V2beta1PipelineVersion | undefined;
    let v2Versions: V2beta1PipelineVersion[] = [];

    let templateString = '';
    let breadcrumbs: Array<{ displayName: string; href: string }> = [];
    const toolbarActions = this.props.toolbarProps.actions;
    let pageTitle: string;

    // If fromRunId or fromRecurringRunId is specified,
    // then load the run and get the pipeline template from it
    if (origin) {
      const msgRunOrRecurringRun = origin.isRecurring ? 'recurring run' : 'run';
      try {
        if (origin.isRecurring) {
          origin.v2RecurringRun = await Apis.recurringRunServiceApi.getRecurringRun(
            origin.recurringRunId!,
          );
        } else {
          origin.v2Run = await Apis.runServiceApiV2.getRun(origin.runId!);
        }

        // If v2 run or recurring is existing, get template string from it
        const templateStrFromOrigin = origin.isRecurring
          ? await this.getTempStrFromRunOrRecurringRun(origin.v2RecurringRun!)
          : await this.getTempStrFromRunOrRecurringRun(origin.v2Run!);

        templateString = templateStrFromOrigin || '';

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
      const pipelineId = this.props.params[RouteParams.pipelineId] ?? '';
      const versionId = this.props.params[RouteParams.pipelineVersionId] ?? '';

      try {
        v2Pipeline = await Apis.pipelineServiceApiV2.getPipeline(pipelineId);
      } catch (err) {
        this.setStateSafe({ graphIsLoading: false });
        await this.showPageError('Cannot retrieve pipeline details.', err);
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      v2SelectedVersion = await this.getSelectedVersion(pipelineId, versionId);

      if (!v2SelectedVersion) {
        // An empty pipeline, which doesn't have any version.
        pageTitle = v2Pipeline.display_name!;
        const actions = this.props.toolbarProps.actions;
        actions[ButtonKeys.DELETE_RUN].disabled = true;
        this.props.updateToolbar({ actions });
      } else {
        pageTitle = `${v2Pipeline.display_name} (${v2SelectedVersion.display_name})`;
        try {
          // TODO(jingzhang36): pagination not proper here. so if many versions,
          // the page size value should be?
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

    const graphV2 = await this._createGraph(templateString);
    this.setStateSafe({
      v2Pipeline,
      v2SelectedVersion,
      v2Versions,
      graphV2,
      graphIsLoading: false,
      templateString,
    });
  }

  public async handleVersionSelected(versionId: string): Promise<void> {
    if (this.state.v2Pipeline) {
      const v2SelectedVersion = (this.state.v2Versions || []).find(
        (v) => v.pipeline_version_id === versionId,
      );
      const pageTitle = this.state.v2Pipeline.display_name?.concat(
        ' (',
        v2SelectedVersion?.display_name!,
        ')',
      );

      this.clearBanner();
      const selectedVersionPipelineTemplate = await this._getTemplateString(v2SelectedVersion);
      this.props.navigate(
        {
          pathname: `/pipelines/details/${this.state.v2Pipeline.pipeline_id}/version/${versionId}`,
        },
        { replace: true },
      );
      this.props.updateToolbar(this.getInitialToolbarState());
      this.props.updateToolbar({ pageTitle });

      const graphV2 = await this._createGraph(selectedVersionPipelineTemplate);
      this.setStateSafe({
        graphV2,
        graphIsLoading: false,
        v2SelectedVersion,
        templateString: selectedVersionPipelineTemplate,
      });
    }
  }

  private async _getTemplateString(pipelineVersion?: V2beta1PipelineVersion): Promise<string> {
    if (pipelineVersion && !pipelineVersion.pipeline_spec) {
      await this.showPageError(
        'This pipeline version has no pipeline spec. Legacy formats are no longer supported; upload a version compiled with the KFP v2 SDK.',
        undefined,
        'warning',
      );
    }
    return pipelineVersion?.pipeline_spec ? JsYaml.dump(pipelineVersion.pipeline_spec) : '';
  }

  private async _createGraph(templateString: string): Promise<PipelineFlowElement[]> {
    if (!templateString) return [];
    try {
      return convertFlowElements(WorkflowUtils.convertYamlToV2PipelineSpec(templateString));
    } catch (err) {
      await this.showPageError('Error: failed to generate Pipeline graph.', err);
      return [];
    }
  }

  private _deleteCallback(_: string[], success: boolean): void {
    if (success) {
      const breadcrumbs = this.props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length
        ? breadcrumbs[breadcrumbs.length - 1].href
        : RoutePage.PIPELINES;
      this.props.navigate(previousPage);
    }
  }
}

export default PipelineDetails;
