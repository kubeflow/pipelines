import React, { useState } from 'react';
import { useQuery } from 'react-query';
import { QUERY_PARAMS } from 'src/components/Router';
import { isFeatureEnabled, FeatureKey } from 'src/features';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { URLParser } from 'src/lib/URLParser';
import { NewRun } from './NewRun';
import NewRunV2 from './NewRunV2';
import { PageProps } from './Page';
import { isTemplateV2 } from 'src/lib/v2/WorkflowUtils';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { ApiRunDetail } from 'src/apis/run';
import { ApiExperiment } from 'src/apis/experiment';
import { ApiJob } from 'src/apis/job';

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
  const [pipelineId, setPipelineId] = useState(urlParser.get(QUERY_PARAMS.pipelineId));
  const experimentId = urlParser.get(QUERY_PARAMS.experimentId);
  const [pipelineVersionIdParam, setPipelineVersionIdParam] = useState(
    urlParser.get(QUERY_PARAMS.pipelineVersionId),
  );
  const existingRunId = originalRunId ? originalRunId : embeddedRunId;

  // Retrieve run details
  const { isSuccess: getRunSuccess, isFetching: runIsFetching, data: apiRun } = useQuery<
    ApiRunDetail,
    Error
  >(
    ['ApiRun', existingRunId],
    () => {
      if (!existingRunId) {
        throw new Error('Run ID is missing');
      }
      return Apis.runServiceApi.getRun(existingRunId);
    },
    { enabled: !!existingRunId, staleTime: Infinity },
  );

  // Retrieve recurring run details
  const {
    isSuccess: getRecurringRunSuccess,
    isFetching: recurringRunIsFetching,
    data: apiRecurringRun,
  } = useQuery<ApiJob, Error>(
    ['ApiRecurringRun', originalRecurringRunId],
    () => {
      if (!originalRecurringRunId) {
        throw new Error('Recurring Run ID is missing');
      }
      return Apis.jobServiceApi.getJob(originalRecurringRunId);
    },
    { enabled: !!originalRecurringRunId, staleTime: Infinity },
  );

  if (apiRun !== undefined && apiRecurringRun !== undefined) {
    throw new Error('The existence of run and recurring run should be exclusive.');
  }

  // template string from cloned object
  let pipelineManifest = apiRun?.run?.pipeline_spec?.pipeline_manifest;
  if (getRecurringRunSuccess && apiRecurringRun) {
    pipelineManifest = apiRecurringRun.pipeline_spec?.pipeline_manifest;
  }

  const { isFetching: pipelineIsFetching, data: apiPipeline } = useQuery<ApiPipeline, Error>(
    ['ApiPipeline', pipelineId],
    () => {
      if (!pipelineId) {
        throw new Error('Pipeline ID is missing');
      }
      return Apis.pipelineServiceApi.getPipeline(pipelineId);
    },
    { enabled: !!pipelineId, staleTime: Infinity, cacheTime: Infinity },
  );

  const { isFetching: pipelineVersionIsFetching, data: apiPipelineVersion } = useQuery<
    ApiPipelineVersion,
    Error
  >(
    ['ApiPipelineVersion', apiPipeline, pipelineVersionIdParam],
    () => {
      const pipelineVersionId = pipelineVersionIdParam || apiPipeline?.default_version?.id;
      if (!pipelineVersionId) {
        throw new Error('Pipeline Version ID is missing');
      }
      return Apis.pipelineServiceApi.getPipelineVersion(pipelineVersionId);
    },
    { enabled: !!apiPipeline, staleTime: Infinity, cacheTime: Infinity },
  );

  const {
    isSuccess: isTemplatePullSuccessFromPipeline,
    isFetching: pipelineTemplateStrIsFetching,
    data: templateStrFromPipelineId,
  } = useQuery<string, Error>(
    ['ApiPipelineVersionTemplate', apiPipeline, pipelineVersionIdParam],
    async () => {
      const pipelineVersionId = apiPipelineVersion?.id;
      if (!pipelineVersionId) {
        return '';
      }
      const template = await Apis.pipelineServiceApi.getPipelineVersionTemplate(pipelineVersionId);
      return template?.template || '';
    },
    { enabled: !!apiPipelineVersion, staleTime: Infinity, cacheTime: Infinity },
  );

  const { data: apiExperiment } = useQuery<ApiExperiment, Error>(
    ['experiment', experimentId],
    async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return Apis.experimentServiceApi.getExperiment(experimentId);
    },
    { enabled: !!experimentId, staleTime: Infinity },
  );

  const templateString = pipelineManifest ? pipelineManifest : templateStrFromPipelineId;

  if (isFeatureEnabled(FeatureKey.V2_ALPHA)) {
    if (
      (getRunSuccess || getRecurringRunSuccess || isTemplatePullSuccessFromPipeline) &&
      isTemplateV2(templateString || '')
    ) {
      return (
        <NewRunV2
          {...props}
          namespace={namespace}
          existingRunId={existingRunId}
          apiRun={apiRun}
          originalRecurringRunId={originalRecurringRunId}
          apiRecurringRun={apiRecurringRun}
          existingPipeline={apiPipeline}
          handlePipelineIdChange={setPipelineId}
          existingPipelineVersion={apiPipelineVersion}
          handlePipelineVersionIdChange={setPipelineVersionIdParam}
          templateString={templateString}
          chosenExperiment={apiExperiment}
        />
      );
    }
  }

  // Use experiment ID to create new run
  // Currently use NewRunV1 as default
  // TODO(jlyaoyuli): set v2 as default once v1 is deprecated.
  if (
    runIsFetching ||
    recurringRunIsFetching ||
    pipelineIsFetching ||
    pipelineVersionIsFetching ||
    pipelineTemplateStrIsFetching
  ) {
    return <div>Currently loading pipeline information</div>;
  }
  return (
    <NewRun
      {...props}
      namespace={namespace}
      existingPipelineId={pipelineId}
      handlePipelineIdChange={setPipelineId}
      existingPipelineVersionId={pipelineVersionIdParam}
      handlePipelineVersionIdChange={setPipelineVersionIdParam}
    />
  );
}

export default NewRunSwitcher;
