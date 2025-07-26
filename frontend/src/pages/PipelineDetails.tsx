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
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import Buttons from 'src/lib/Buttons';
import * as StaticGraphParser from 'src/lib/StaticGraphParser';
import { transitiveReduction } from 'src/lib/StaticGraphParser';
import { useURLParser } from 'src/lib/URLParser';
import { ensureError, logger } from 'src/lib/Utils';
import { PageProps } from './Page';
import PipelineDetailsV1 from './PipelineDetailsV1';
import PipelineDetailsV2 from './PipelineDetailsV2';
import { ApiRunDetail } from 'src/apis/run';
import { ApiJob } from 'src/apis/job';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { AdditionalNodeData } from 'src/components/Graph';

type Origin = {
  isRecurring: boolean;
  runId: string | null;
  recurringRunId: string | null;
  v1Run?: ApiRunDetail;
  v2Run?: V2beta1Run;
  v1RecurringRun?: ApiJob;
  v2RecurringRun?: V2beta1RecurringRun;
};

function PipelineDetails(props: PageProps) {
  const urlParser = useURLParser();

  // State hooks
  const [graph, setGraph] = React.useState<dagre.graphlib.Graph<AdditionalNodeData> | null>(null);
  const [reducedGraph, setReducedGraph] = React.useState<dagre.graphlib.Graph<
    AdditionalNodeData
  > | null>(null);
  const [graphV2, setGraphV2] = React.useState<PipelineFlowElement[] | null>(null);
  const [graphIsLoading, setGraphIsLoading] = React.useState<boolean>(true);
  const [v1Pipeline, setV1Pipeline] = React.useState<ApiPipeline | null>(null);
  const [v2Pipeline, setV2Pipeline] = React.useState<V2beta1Pipeline | null>(null);
  const [v1SelectedVersion, setV1SelectedVersion] = React.useState<
    ApiPipelineVersion | undefined
  >();
  const [v2SelectedVersion, setV2SelectedVersion] = React.useState<
    V2beta1PipelineVersion | undefined
  >();
  const [templateString, setTemplateString] = React.useState<string | undefined>();
  const [v1Versions, setV1Versions] = React.useState<ApiPipelineVersion[]>([]);
  const [v2Versions, setV2Versions] = React.useState<V2beta1PipelineVersion[]>([]);

  // Get route parameters
  const pipelineIdFromParams = React.useMemo(() => {
    // Access route params through location pathname
    const pathSegments = props.location.pathname.split('/');
    const detailsIndex = pathSegments.indexOf('details');
    return detailsIndex !== -1 && pathSegments[detailsIndex + 1]
      ? pathSegments[detailsIndex + 1]
      : '';
  }, [props.location.pathname]);

  const pipelineVersionIdFromParams = React.useMemo(() => {
    const pathSegments = props.location.pathname.split('/');
    const versionIndex = pathSegments.indexOf('version');
    return versionIndex !== -1 && pathSegments[versionIndex + 1]
      ? pathSegments[versionIndex + 1]
      : '';
  }, [props.location.pathname]);

  // Utility functions
  const clearBanner = React.useCallback(() => {
    props.updateBanner({});
  }, [props]);

  const showPageError = React.useCallback(
    async (message: string, error: Error) => {
      props.updateBanner({
        additionalInfo: error.message,
        message,
        mode: 'error',
      });
    },
    [props],
  );

  const getOrigin = React.useCallback((): Origin | undefined => {
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
  }, [urlParser]);

  const getTempStrFromRunOrRecurringRun = React.useCallback(
    async (existingObj: V2beta1Run | V2beta1RecurringRun) => {
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

      return pipelineManifest ? pipelineManifest : templateStrFromOrigin;
    },
    [],
  );

  const getSelectedVersion = React.useCallback(
    async (pipelineId: string, versionId?: string): Promise<V2beta1PipelineVersion> => {
      if (versionId) {
        return await Apis.pipelineServiceApiV2.getPipelineVersion(pipelineId, versionId);
      } else {
        const listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
          pipelineId,
          undefined,
          1,
          'created_at desc',
        );
        if (listVersionsResponse.pipeline_versions?.length) {
          return listVersionsResponse.pipeline_versions[0];
        } else {
          throw new Error(`No pipeline versions found for pipeline: ${pipelineId}.`);
        }
      }
    },
    [],
  );

  const createGraph = React.useCallback(
    async (
      templateString: string,
    ): Promise<
      [
        dagre.graphlib.Graph<AdditionalNodeData> | null,
        dagre.graphlib.Graph<AdditionalNodeData> | null,
        PipelineFlowElement[],
      ]
    > => {
      let graph: dagre.graphlib.Graph<AdditionalNodeData> | null = null;
      let reducedGraph: dagre.graphlib.Graph<AdditionalNodeData> | null = null;
      let graphV2: PipelineFlowElement[] = [];

      if (templateString) {
        const isV2Pipeline = WorkflowUtils.isTemplateV2(templateString);
        if (isV2Pipeline) {
          try {
            const v2PipelineSpec = convertYamlToV2PipelineSpec(templateString);
            graphV2 = convertFlowElements(v2PipelineSpec);
          } catch (err) {
            await showPageError('Error: failed to generate workflow graph.', ensureError(err));
          }
        } else {
          const workflowObject = JsYaml.safeLoad(templateString) as Workflow;
          if (workflowObject) {
            graph = StaticGraphParser.createGraph(workflowObject!);
            if (graph) {
              reducedGraph = transitiveReduction(graph) || null;
            }
          } else {
            await showPageError(
              'Error: failed to generate workflow graph.',
              new Error('workflowObject is empty'),
            );
          }
        }
      }

      return [graph, reducedGraph, graphV2];
    },
    [showPageError],
  );

  const handleVersionSelected = React.useCallback(
    async (versionId: string): Promise<void> => {
      setGraphIsLoading(true);
      clearBanner();

      const origin = getOrigin();
      let templateString = '';

      // If fromRunId or fromRecurringRunId is specified, the list of version is not available in this page
      if (origin) {
        return;
      } else {
        // The normal case where a pipeline + version is selected
        const pipelineId = pipelineIdFromParams;
        let v1Version: ApiPipelineVersion | null = null;
        let v2SelectedVersion: V2beta1PipelineVersion | undefined;

        try {
          v1Version = await Apis.pipelineServiceApi.getPipelineVersion(versionId);
        } catch (err) {
          // This is expected for v2 pipeline versions
        }

        v2SelectedVersion = await getSelectedVersion(pipelineId, versionId);

        if (v1Version) {
          setV1SelectedVersion(v1Version);
        }
        setV2SelectedVersion(v2SelectedVersion);

        // Get template string
        if (v2SelectedVersion.pipeline_spec) {
          templateString = JsYaml.safeDump(v2SelectedVersion.pipeline_spec);
        } else if (v1Version) {
          const templateResponse = await Apis.pipelineServiceApi.getPipelineVersionTemplate(
            versionId,
          );
          templateString = templateResponse.template || '';
        }
      }

      setTemplateString(templateString);
      const [newGraph, newReducedGraph, newGraphV2] = await createGraph(templateString);

      // Currently, we allow upload v1 version to v2 pipeline or vice versa,
      // so v1 and v2 field is "non-exclusive".
      if (isFeatureEnabled(FeatureKey.V2_ALPHA) && newGraphV2.length > 0) {
        setGraph(null);
        setGraphV2(newGraphV2);
        setReducedGraph(null);
      } else {
        setGraph(newGraph);
        setGraphV2(null);
        setReducedGraph(newReducedGraph);
      }
      setGraphIsLoading(false);
    },
    [getOrigin, pipelineIdFromParams, getSelectedVersion, createGraph, clearBanner],
  );

  const load = React.useCallback(async (): Promise<void> => {
    clearBanner();
    const origin = getOrigin();

    let newV1Pipeline: ApiPipeline | null = null;
    let v1Version: ApiPipelineVersion | null = null;
    let newV1SelectedVersion: ApiPipelineVersion | undefined;
    let newV1Versions: ApiPipelineVersion[] = [];

    let newV2Pipeline: V2beta1Pipeline | null = null;
    let newV2SelectedVersion: V2beta1PipelineVersion | undefined;
    let newV2Versions: V2beta1PipelineVersion[] = [];

    let newTemplateString = '';
    let breadcrumbs: Array<{ displayName: string; href: string }> = [];
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
          ? await getTempStrFromRunOrRecurringRun(origin.v2RecurringRun!)
          : await getTempStrFromRunOrRecurringRun(origin.v2Run!);

        // V1: Convert the run's pipeline spec to YAML to be displayed as the pipeline's source.
        // V2: Directly Use the template string from existing run or recurring run
        // because it can be translated in JSON format.
        if (isFeatureEnabled(FeatureKey.V2_ALPHA) && templateStrFromOrigin) {
          newTemplateString = templateStrFromOrigin;
        } else {
          try {
            // TODO(jlyaoyuli): remove when v1 is deprecated
            // V1: Convert the run's pipeline spec to YAML to be displayed as the pipeline's source.
            const workflow = origin.isRecurring
              ? JSON.parse(origin.v1RecurringRun!.pipeline_spec!.workflow_manifest || '{}')
              : JSON.parse(origin.v1Run!.pipeline_runtime!.workflow_manifest || '{}');
            newTemplateString = JsYaml.safeDump(workflow);
          } catch (err) {
            await showPageError(
              `Error: failed to parse ${msgRunOrRecurringRun} template.`,
              ensureError(err),
            );
            logger.error(`Error: failed to parse ${msgRunOrRecurringRun} template`, err);
            return;
          }
        }

        pageTitle = origin.isRecurring
          ? `Pipeline details (from recurring run)`
          : `Pipeline details (from run)`;
        breadcrumbs = [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }];
      } catch (err) {
        setGraphIsLoading(false);
        await showPageError(`Cannot retrieve ${msgRunOrRecurringRun} details.`, ensureError(err));
        logger.error(`Cannot retrieve ${msgRunOrRecurringRun} details.`, err);
        return;
      }
    } else {
      // if fromRunId or fromRecurringRunId is not specified, then we have a full pipeline
      const pipelineId = pipelineIdFromParams;
      const versionId = pipelineVersionIdFromParams;

      try {
        newV1Pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
        newV2Pipeline = await Apis.pipelineServiceApiV2.getPipeline(pipelineId);
      } catch (err) {
        setGraphIsLoading(false);
        await showPageError('Cannot retrieve pipeline details.', ensureError(err));
        logger.error('Cannot retrieve pipeline details.', err);
        return;
      }

      try {
        // TODO(rjbauer): it's possible we might not have a version, even default
        if (versionId) {
          v1Version = await Apis.pipelineServiceApi.getPipelineVersion(versionId);
        }
      } catch (err) {
        setGraphIsLoading(false);
        await showPageError('Cannot retrieve pipeline version.', ensureError(err));
        logger.error('Cannot retrieve pipeline version.', err);
        return;
      }

      newV1SelectedVersion = versionId ? v1Version! : newV1Pipeline.default_version;
      newV2SelectedVersion = await getSelectedVersion(pipelineId, versionId);

      try {
        const v1VersionsResponse = await Apis.pipelineServiceApi.listPipelineVersions(
          'PIPELINE',
          pipelineId,
          100,
          undefined,
          'created_at desc',
        );
        newV1Versions = v1VersionsResponse.versions || [];
      } catch (err) {
        logger.error('Cannot retrieve pipeline versions.', err);
      }

      try {
        const v2VersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
          pipelineId,
          undefined,
          100,
          'created_at desc',
        );
        newV2Versions = v2VersionsResponse.pipeline_versions || [];
      } catch (err) {
        logger.error('Cannot retrieve pipeline versions.', err);
      }

      // Get template string
      if (newV2SelectedVersion.pipeline_spec) {
        newTemplateString = JsYaml.safeDump(newV2SelectedVersion.pipeline_spec);
      } else if (newV1SelectedVersion) {
        const versionIdToUse = newV1SelectedVersion.id || newV1Pipeline.default_version?.id;
        if (versionIdToUse) {
          const templateResponse = await Apis.pipelineServiceApi.getPipelineVersionTemplate(
            versionIdToUse,
          );
          newTemplateString = templateResponse.template || '';
        }
      }

      breadcrumbs = [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }];
      pageTitle = newV2Pipeline?.display_name || newV1Pipeline?.name || 'Pipeline details';
    }

    const [newGraph, newReducedGraph, newGraphV2] = await createGraph(newTemplateString);

    // Currently, we allow upload v1 version to v2 pipeline or vice versa,
    // so v1 and v2 field is "non-exclusive".
    // TODO(jlyaoyuli): If we decide not to support "mix versions",
    // v1 and v2 should be "exclusive"
    if (isFeatureEnabled(FeatureKey.V2_ALPHA) && newGraphV2.length > 0) {
      setV1Pipeline(newV1Pipeline);
      setV2Pipeline(newV2Pipeline);
      setV1SelectedVersion(newV1SelectedVersion);
      setV2SelectedVersion(newV2SelectedVersion);
      setV1Versions(newV1Versions);
      setV2Versions(newV2Versions);
      setGraph(null);
      setGraphV2(newGraphV2);
      setGraphIsLoading(false);
      setReducedGraph(null);
      setTemplateString(newTemplateString);
    } else {
      setV1Pipeline(newV1Pipeline);
      setV2Pipeline(newV2Pipeline);
      setV1SelectedVersion(newV1SelectedVersion);
      setV2SelectedVersion(newV2SelectedVersion);
      setV1Versions(newV1Versions);
      setV2Versions(newV2Versions);
      setGraph(newGraph);
      setGraphV2(null);
      setGraphIsLoading(false);
      setReducedGraph(newReducedGraph);
      setTemplateString(newTemplateString);
    }

    // Update toolbar
    const buttons = new Buttons(props, load, { get: urlParser.get, build: urlParser.build });

    if (origin) {
      const getOriginIdList = () => [origin.isRecurring ? origin.recurringRunId! : origin.runId!];
      origin.isRecurring
        ? buttons.cloneRecurringRun(getOriginIdList, true)
        : buttons.cloneRun(getOriginIdList, true);
    } else {
      buttons
        .newRun(() => pipelineIdFromParams)
        .newRunFromPipelineVersion(
          () => pipelineIdFromParams,
          () => newV2SelectedVersion?.pipeline_version_id || newV1SelectedVersion?.id || '',
        );
    }

    const toolbarProps: ToolbarProps = {
      actions: buttons.getToolbarActionMap(),
      breadcrumbs,
      pageTitle,
    };
    props.updateToolbar(toolbarProps);
  }, [
    clearBanner,
    getOrigin,
    createGraph,
    props,
    urlParser.get,
    urlParser.build,
    getTempStrFromRunOrRecurringRun,
    showPageError,
    pipelineIdFromParams,
    pipelineVersionIdFromParams,
    getSelectedVersion,
  ]);

  // Load data on mount
  React.useEffect(() => {
    load();
  }, [load]);

  const setLayers = (layers: string[]) => {
    if (!templateString) {
      console.warn('pipeline spec template is unknown.');
      return;
    }
    const pipelineSpec = convertYamlToV2PipelineSpec(templateString!);
    const newElements = convertSubDagToFlowElements(pipelineSpec!, layers);
    setGraphV2(newElements);
    setGraphIsLoading(false);
  };

  const showV2Pipeline =
    isFeatureEnabled(FeatureKey.V2_ALPHA) && graphV2 && graphV2.length > 0 && !graph;

  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      {graphIsLoading && <div>Currently loading pipeline information</div>}
      {!graphIsLoading && showV2Pipeline && (
        <PipelineDetailsV2
          templateString={templateString}
          pipelineFlowElements={graphV2!}
          setSubDagLayers={setLayers}
          pipeline={v2Pipeline}
          selectedVersion={v2SelectedVersion}
          versions={v2Versions}
          handleVersionSelected={handleVersionSelected}
        />
      )}
      {!graphIsLoading && !showV2Pipeline && (
        <PipelineDetailsV1
          pipeline={v1Pipeline}
          templateString={templateString}
          graph={graph}
          reducedGraph={reducedGraph}
          updateBanner={props.updateBanner}
          selectedVersion={v1SelectedVersion}
          versions={v1Versions}
          handleVersionSelected={handleVersionSelected}
        />
      )}
    </div>
  );
}

export default PipelineDetails;
