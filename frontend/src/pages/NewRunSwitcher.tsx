import React from 'react';
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

function NewRunSwitcher(props: PageProps) {
  const namespace = React.useContext(NamespaceContext);

  const urlParser = new URLParser(props);
  const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
  const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
  const pipelineVersionIdParam = urlParser.get(QUERY_PARAMS.pipelineVersionId);

  const {
    isSuccess: runIsSuccess,
    isFetching: runIsFetching,
    data: templateStrFromRunId,
  } = useQuery(
    [originalRunId],
    async () => {
      if (!originalRunId) {
        return '';
      }
      const originalRun = await Apis.runServiceApi.getRun(originalRunId);
      return originalRun.run?.pipeline_spec?.pipeline_manifest || '';
    },
    { staleTime: Infinity },
  );

  const {
    isSuccess: pipelineIsSuccess,
    isFetching: pipelineIsFetching,
    data: templateStrFromPipelineId,
  } = useQuery<string, Error>(
    [pipelineId, pipelineVersionIdParam],
    async () => {
      if (!pipelineId) {
        return '';
      }
      const pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);

      const pipelineVersionId = pipelineVersionIdParam || pipeline?.default_version?.id;
      if (!pipelineVersionId) {
        return '';
      }

      await Apis.pipelineServiceApi.getPipelineVersion(pipelineVersionId);
      const template = await Apis.pipelineServiceApi.getPipelineVersionTemplate(pipelineVersionId);
      return template?.template || '';
    },
    { staleTime: Infinity },
  );

  const templateString =
    templateStrFromRunId != '' ? templateStrFromRunId : templateStrFromPipelineId;

  if (isFeatureEnabled(FeatureKey.V2_ALPHA)) {
    if ((runIsSuccess || pipelineIsSuccess) && isTemplateV2(templateString || '')) {
      return <NewRunV2 {...props} namespace={namespace} />;
    }
  }

  if (pipelineIsFetching || runIsFetching) {
    return <div>Currently loading pipeline information</div>;
  }
  return <NewRun {...props} namespace={namespace} />;
}

export default NewRunSwitcher;
