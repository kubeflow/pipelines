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

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ArtifactArtifactType, V2beta1PipelineTask } from 'src/apisv2beta1/run';
import { RuntimeInputOutputTab } from './RuntimeInputOutputTab';

vi.mock('src/components/ArtifactPreview', () => ({
  default: ({ value }: { value: string }) => <div>Preview {value}</div>,
}));

describe('RuntimeInputOutputTab', () => {
  it('renders native parameters and artifact links for every input and output', () => {
    const task: V2beta1PipelineTask = {
      name: 'train',
      display_name: 'Train model',
      inputs: {
        parameters: [{ parameter_key: 'epochs', value: 5 as any }],
        artifacts: [
          {
            artifact_key: 'dataset',
            artifacts: [
              {
                artifact_id: 'dataset-1',
                name: 'training-data',
                type: ArtifactArtifactType.Dataset,
                uri: 's3://pipeline-root/data',
              },
            ],
          },
        ],
      },
      outputs: {
        parameters: [{ parameter_key: 'score', value: 0.98 as any }],
        artifacts: [
          {
            artifact_key: 'models',
            artifacts: [
              {
                artifact_id: 'model-1',
                name: 'trained-model',
                type: ArtifactArtifactType.Model,
                uri: 's3://pipeline-root/model-1',
              },
              {
                artifact_id: 'model-2',
                name: 'trained-model',
                type: ArtifactArtifactType.Model,
                uri: 's3://pipeline-root/model-2',
              },
            ],
          },
        ],
      },
    };

    render(
      <MemoryRouter>
        <RuntimeInputOutputTab task={task} namespace='team-a' />
      </MemoryRouter>,
    );

    screen.getByText('Train model');
    screen.getByText('Input Parameters');
    screen.getByText('epochs');
    screen.getByText('5');
    screen.getByText('Output Parameters');
    screen.getByText('score');
    screen.getByText('0.98');
    expect(screen.getByRole('link', { name: 'training-data' })).toHaveAttribute(
      'href',
      '/artifacts/dataset-1',
    );
    expect(screen.getByRole('link', { name: 'trained-model' })).toHaveAttribute(
      'href',
      '/artifacts/model-1',
    );
    expect(screen.getByRole('link', { name: 'trained-model (2)' })).toHaveAttribute(
      'href',
      '/artifacts/model-2',
    );
    screen.getByText('Preview s3://pipeline-root/data');
    screen.getByText('Preview s3://pipeline-root/model-1');
    screen.getByText('Preview s3://pipeline-root/model-2');
  });

  it('renders an explicit empty state', () => {
    render(
      <MemoryRouter>
        <RuntimeInputOutputTab task={{ name: 'empty-task' }} />
      </MemoryRouter>,
    );

    screen.getByText('There is no input/output parameter or artifact.');
  });
});
