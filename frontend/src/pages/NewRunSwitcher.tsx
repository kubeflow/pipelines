import React, { useState } from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from 'react-query';
import { QUERY_PARAMS } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import { NewRun } from './NewRun';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import { isTemplateV2 } from 'src/lib/v2/WorkflowUtils';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);

  const urlParser = new URLParser(props);
  // Currently using two query parameters to get Run ID.
  // because v1 has two different behavior with Run ID (clone a run / start a run)
  // Will keep clone run only in v2 if run ID is existing
  // runID query by cloneFromRun will be deprecated once v1 is deprecated.
  const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
  const embeddedRunId = urlParser.get(QUERY_PARAMS.fromRunId);
  const originalRecurringRunId = urlParser.get(QUERY_PARAMS.cloneFromRecurringRun);
  const [pipelineIdFromPipeline, setPipelineIdFromPipeline] = useState(
    urlParser.get(QUERY_PARAMS.pipelineId),
  );
  const experimentId = urlParser.get(QUERY_PARAMS.experimentId);
  const [pipelineVersionIdParam, setPipelineVersionIdParam] = useState(
    urlParser.get(QUERY_PARAMS.pipelineVersionId),
  );
  const existingRunId = originalRunId ? originalRunId : embeddedRunId;
  let pipelineIdFromRunOrRecurringRun;
  let pipelineVersionIdFromRunOrRecurringRun;

  // Retrieve v2 run details
  const { isSuccess: getV2RunSuccess, isFetching: v2RunIsFetching, data: v2Run } = useQuery<
    V2beta1Run,
    Error
  >(
    ['v2_run_details', existingRunId],
    () => {
      if (!existingRunId) {
        throw new Error('Run ID is missing');
      }
      return Apis.runServiceApiV2.getRun(existingRunId);
    },
    { enabled: !!existingRunId, staleTime: Infinity },
  );

  // Retrieve recurring run details
  const {
    isSuccess: getRecurringRunSuccess,
    isFetching: recurringRunIsFetching,
    data: recurringRun,
  } = useQuery<V2beta1RecurringRun, Error>(
    ['recurringRun', originalRecurringRunId],
    () => {
      if (!originalRecurringRunId) {
        throw new Error('Recurring Run ID is missing');
      }
      return Apis.recurringRunServiceApi.getRecurringRun(originalRecurringRunId);
    },
    { enabled: !!originalRecurringRunId, staleTime: Infinity },
  );

  if (v2Run !== undefined && recurringRun !== undefined) {
    throw new Error('The existence of run and recurring run should be exclusive.');
  }

  pipelineIdFromRunOrRecurringRun =
    v2Run?.pipeline_version_reference?.pipeline_id ||
    recurringRun?.pipeline_version_reference?.pipeline_id;
  pipelineVersionIdFromRunOrRecurringRun =
    v2Run?.pipeline_version_reference?.pipeline_version_id ||
    recurringRun?.pipeline_version_reference?.pipeline_version_id;

  // template string from cloned run / recurring run created by pipeline_spec
  let pipelineManifest: string | undefined;
  if (getV2RunSuccess && v2Run && v2Run.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(v2Run.pipeline_spec);
  }

  if (getRecurringRunSuccess && recurringRun && recurringRun.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(recurringRun.pipeline_spec);
  }

  const { isFetching: pipelineIsFetching, data: pipeline } = useQuery<V2beta1Pipeline, Error>(
    ['pipeline', pipelineIdFromPipeline],
    () => {
      if (!pipelineIdFromPipeline) {
        throw new Error('Pipeline ID is missing');
      }
      return Apis.pipelineServiceApiV2.getPipeline(pipelineIdFromPipeline);
    },
    { enabled: !!pipelineIdFromPipeline, staleTime: Infinity, cacheTime: 0 },
  );

  const pipelineId = pipelineIdFromPipeline || pipelineIdFromRunOrRecurringRun;
  const pipelineVersionId = pipelineVersionIdParam || pipelineVersionIdFromRunOrRecurringRun;

  const { isFetching: pipelineVersionIsFetching, data: pipelineVersion } = useQuery<
    V2beta1PipelineVersion,
    Error
  >(
    ['pipelineVersion', pipelineVersionIdParam],
    () => {
      if (!(pipelineId && pipelineVersionId)) {
        throw new Error('Pipeline id or pipeline Version ID is missing');
      }
      return Apis.pipelineServiceApiV2.getPipelineVersion(pipelineId, pipelineVersionId);
    },
    { enabled: !!pipelineId && !!pipelineVersionId, staleTime: Infinity, cacheTime: 0 },
  );
  const pipelineSpecInVersion = pipelineVersion?.pipeline_spec;
  const templateStrFromSpec = pipelineSpecInVersion ? JsYaml.safeDump(pipelineSpecInVersion) : '';

  const { isFetching: v1TemplateStrIsFetching, data: v1Template } = useQuery<string, Error>(
    ['v1PipelineVersionTemplate', pipelineVersionIdParam],
    async () => {
      if (!(pipelineId && pipelineVersionId)) {
        throw new Error('Pipeline id or pipeline Version ID is missing');
      }

      let v1TemplateResponse;
      if (pipelineVersionId) {
        v1TemplateResponse = await Apis.pipelineServiceApi.getPipelineVersionTemplate(
          pipelineVersionId,
        );
        return v1TemplateResponse.template || '';
      } else {
        v1TemplateResponse = await Apis.pipelineServiceApi.getTemplate(pipelineId);
      }
      return v1TemplateResponse.template || '';
    },
    { enabled: !!pipelineId || !!pipelineVersionId, staleTime: Infinity, cacheTime: 0 },
  );
  const v1TemplateStr = v1Template || '';

  const { isFetching: experimentIsFetching, data: experiment } = useQuery<V2beta1Experiment, Error>(
    ['experiment', experimentId],
    async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return Apis.experimentServiceApiV2.getExperiment(experimentId);
    },
    { enabled: !!experimentId, staleTime: Infinity },
  );

  // Three possible sources for template string
  // 1. pipelineManifest: pipeline_spec stored in run or recurring run created by SDK
  // 2. templateStrFromSpec: pipeline_spec stored in pipeline_version
  // 3. v1TemplateStr: pipelines created by v1 API (no pipeline_spec field)
  const templateString = pipelineManifest ?? (templateStrFromSpec || v1TemplateStr);

  if (
    v2RunIsFetching ||
    recurringRunIsFetching ||
    pipelineIsFetching ||
    pipelineVersionIsFetching ||
    v1TemplateStrIsFetching ||
    experimentIsFetching
  ) {
    return <div>Currently loading pipeline information</div>;
  }

  if (templateString && !isTemplateV2(templateString)) {
    return (
      <NewRun
        {...props}
        namespace={namespace}
        existingPipelineId={pipelineIdFromPipeline}
        handlePipelineIdChange={setPipelineIdFromPipeline}
        existingPipelineVersionId={pipelineVersionIdParam}
        handlePipelineVersionIdChange={setPipelineVersionIdParam}
      />
    );
  }

  return (
    <NewRunV2
      {...props}
      namespace={namespace}
      existingRunId={existingRunId}
      existingRun={v2Run}
      existingRecurringRunId={originalRecurringRunId}
      existingRecurringRun={recurringRun}
      existingPipeline={pipeline}
      handlePipelineIdChange={setPipelineIdFromPipeline}
      existingPipelineVersion={pipelineVersion}
      handlePipelineVersionIdChange={setPipelineVersionIdParam}
      templateString={templateString}
      chosenExperiment={experiment}
    />
  );
}

export default NewRunSwitcher;
