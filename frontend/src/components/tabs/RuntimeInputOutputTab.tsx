// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { V2beta1PipelineTask } from 'src/apisv2beta1/run';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import { buildRuntimeArtifactRows } from 'src/components/RuntimeArtifactRows';
import { commonCss, padding } from 'src/Css';
import { formatParameters } from 'src/lib/v2/RuntimeArtifactUtils';
import { getTaskDisplayName } from 'src/lib/v2/RunTaskUtils';

export interface RuntimeInputOutputTabProps {
  task: V2beta1PipelineTask;
  namespace?: string;
}

export function RuntimeInputOutputTab({ task, namespace }: RuntimeInputOutputTabProps) {
  const inputParameters = formatParameters(task.inputs?.parameters);
  const outputParameters = formatParameters(task.outputs?.parameters);
  const inputArtifacts = buildRuntimeArtifactRows(task.inputs?.artifacts);
  const outputArtifacts = buildRuntimeArtifactRows(task.outputs?.artifacts);
  const isEmpty =
    !inputParameters.length &&
    !outputParameters.length &&
    !inputArtifacts.length &&
    !outputArtifacts.length;

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <h3>{getTaskDisplayName(task)}</h3>
          {isEmpty && (
            <Banner message='There is no input/output parameter or artifact.' mode='info' />
          )}
          {!!inputParameters.length && (
            <DetailsTable title='Input Parameters' fields={inputParameters} />
          )}
          {!!inputArtifacts.length && (
            <DetailsTable
              title='Input Artifacts'
              fields={inputArtifacts}
              valueComponent={ArtifactPreview}
              valueComponentProps={{ namespace }}
            />
          )}
          {!!outputParameters.length && (
            <DetailsTable title='Output Parameters' fields={outputParameters} />
          )}
          {!!outputArtifacts.length && (
            <DetailsTable
              title='Output Artifacts'
              fields={outputArtifacts}
              valueComponent={ArtifactPreview}
              valueComponentProps={{ namespace }}
            />
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}

export default RuntimeInputOutputTab;
