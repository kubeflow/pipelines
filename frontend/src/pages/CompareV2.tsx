/*
 * Copyright 2022 The Kubeflow Authors
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

import React, { useEffect } from 'react';
import { useQuery } from 'react-query';
import { QUERY_PARAMS } from 'src/components/Router';
import {
  getVerifiedClassificationMetricsArtifacts,
  getVerifiedMetricsArtifacts,
  getVertifiedHtmlArtifacts,
  getVertifiedMarkdownArtifacts,
  getV1VisualizationArtifacts,
} from 'src/components/viewers/MetricsVisualizations';
import { commonCss } from 'src/Css';
import { URLParser } from 'src/lib/URLParser';
import { Api } from 'src/mlmd/library';
import {
  getArtifactsFromContext,
  getArtifactTypes,
  getEventsByExecutions,
  getExecutionsFromContext,
  getKfpV2RunContext,
  getOutputLinkedArtifactsInExecution,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { ArtifactType, Event, Execution, GetEventsByArtifactIDsRequest } from 'src/third_party/mlmd';
import { PageProps } from './Page';
import { MlmdPackage } from './RunDetailsV2';

const EventType = Event.Type;

function CompareV2(props: PageProps) {
  const api = Api.getInstance();
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves MLMD states from the MLMD store.
  const { isSuccess, data } = useQuery<MlmdPackage[], Error>(
    ['mlmd_package', { ids: runIds }],
    () =>
      Promise.all(
        runIds.map(async id => {
          const context = await getKfpV2RunContext(id);
          const executions = await getExecutionsFromContext(context);
          const artifacts = await getArtifactsFromContext(context);
          const events = await getEventsByExecutions(executions);

          return { executions, artifacts, events };
        }),
      ),
    {
      staleTime: Infinity,
      onError: error =>
        props.updateBanner({
          message: 'Cannot get MLMD objects from Metadata store.',
          additionalInfo: error.message,
          mode: 'error',
        }),
      onSuccess: () => props.updateBanner({}),
    },
  );

  if (data) {
    console.log(data);
    for (const mlmdPackage of data) {
      console.log(mlmdPackage);
      const artifactIds = mlmdPackage.artifacts.map(artifact => artifact.getId());
      const eventsRequest = new GetEventsByArtifactIDsRequest();
      eventsRequest.setArtifactIdsList(artifactIds);
      // I know that not all code paths return a value, this is intentional code just for testing, for the moment.
      api.metadataStoreService.getEventsByArtifactIDs(eventsRequest).then(response => {
        if (!response) {
          console.log('Error in response.');
          return <></>;
        }
  
        const responseData = response.getEventsList();
        console.log(responseData);
        // The last output event is the event that produced current artifact.
        const outputEvents = responseData
          .filter(event => event.getType() === EventType.DECLARED_OUTPUT || event.getType() === EventType.OUTPUT);
        console.log(outputEvents);
        for (const outputEvent of outputEvents) {
          console.log(outputEvent?.getPath()?.getStepsList()[0].getKey());
          console.log(outputEvent.getPath());
        }
      }).catch(err => {
        console.log(err);
      });
    }
  }

  // // Retrieving a list of artifacts associated with this execution,
  // // so we can find the artifact for system metrics from there.
  // const {
  //   isLoading: isLoadingArtifacts,
  //   isSuccess: isSuccessArtifacts,
  //   error: errorArtifacts,
  //   data: linkedArtifactsList,
  // } = useQuery<LinkedArtifact[][], Error>(
  //   ['execution_output_artifact', { data }],
  //   async () => {
  //     const promises = [];
  //     if (data) {
  //       for (const mlmdPackage of data) {
  //         for (const execution of mlmdPackage.executions) {
  //           promises.push(getOutputLinkedArtifactsInExecution(execution));
  //         }
  //       }
  //     }
  //     return await Promise.all(promises);
  //   },
  //   { staleTime: Infinity },
  // );

  // // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // // so we can identify metrics artifact provided by system.
  // const {
  //   isLoading: isLoadingArtifactTypes,
  //   isSuccess: isSuccessArtifactTypes,
  //   error: errorArtifactTypes,
  //   data: artifactTypes,
  // } = useQuery<ArtifactType[], Error>(['artifact_types', {}], () => getArtifactTypes(), {
  //   staleTime: Infinity,
  // });

  if (!data) {
    return <></>;
  }

  // if (!linkedArtifactsList) {
  //   return <></>;
  // }

  // if (!artifactTypes) {
  //   return <></>;
  // }

  // // There can be multiple system.ClassificationMetrics or system.Metrics artifacts per execution.
  // // Get scalar metrics, confidenceMetrics and confusionMatrix from artifact.
  // // If there is no available metrics, show banner to notify users.
  // // Otherwise, Visualize all available metrics per artifact.
  // console.log(data);
  // console.log(linkedArtifactsList);
  // for (const linkedArtifacts of linkedArtifactsList) {
  //   const artifacts = linkedArtifacts.map(x => x.artifact);
  //   const classificationMetricsArtifacts = getVerifiedClassificationMetricsArtifacts(
  //     artifacts,
  //     artifactTypes,
  //   );
  //   const metricsArtifacts = getVerifiedMetricsArtifacts(artifacts, artifactTypes);
  //   const htmlArtifacts = getVertifiedHtmlArtifacts(linkedArtifacts, artifactTypes);
  //   const mdArtifacts = getVertifiedMarkdownArtifacts(linkedArtifacts, artifactTypes);
  //   const v1VisualizationArtifact = getV1VisualizationArtifacts(linkedArtifacts, artifactTypes);

  //   console.log('');
  //   console.log(classificationMetricsArtifacts);
  //   console.log(metricsArtifacts);
  //   console.log(htmlArtifacts);
  //   console.log(mdArtifacts);
  //   console.log(v1VisualizationArtifact);
  // }

  // const onSelectionChange = (elements: Elements<FlowElementDataBase> | null) => {
  //   if (!elements || elements?.length === 0) {
  //     setSelectedNode(null);
  //     return;
  //   }
  //   if (elements && elements.length === 1) {
  //     setSelectedNode(elements[0]);
  //     if (data) {
  //       setSelectedNodeMlmdInfo(
  //         getNodeMlmdInfo(elements[0], data.executions, data.events, data.artifacts),
  //       );
  //     }
  //   }
  // };

  return (
    <div className={commonCss.page}>
      <p>This is the V2 Run Comparison page.</p>
    </div>
  );
}

export default CompareV2;
