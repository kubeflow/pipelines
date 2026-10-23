/*
 * Copyright 2021 The Kubeflow Authors
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

import { render, screen } from '@testing-library/react';
import { testBestPractices } from 'src/TestUtils';
import { PipelineVersionCard } from './PipelineVersionCard';

const pipeline = {
  pipeline_id: 'pipeline-id',
  created_at: new Date('2021-11-24T20:58:23.000Z'),
  description: 'Pipeline **description**',
};
const selectedVersion = {
  pipeline_version_id: 'version-id',
  display_name: 'Version one',
  description: 'Version description',
  code_source_url: 'https://github.com/kubeflow/pipelines',
};

testBestPractices();
describe('PipelineVersionCard', () => {
  it('renders pipeline and version metadata with formatted descriptions', () => {
    render(<PipelineVersionCard pipeline={pipeline} selectedVersion={selectedVersion} />);
    expect(screen.getByRole('region', { name: 'Pipeline details' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Version details' })).toBeInTheDocument();
    expect(screen.getByText('pipeline-id')).toBeInTheDocument();
    expect(screen.getByText('Version one')).toBeInTheDocument();
    expect(screen.getByText('description').tagName).toBe('STRONG');
    expect(screen.getByRole('link', { name: 'Version source' })).toHaveAttribute(
      'href',
      selectedVersion.code_source_url,
    );
  });

  it('does not render an unsafe source link', () => {
    render(
      <PipelineVersionCard
        pipeline={pipeline}
        selectedVersion={{
          ...selectedVersion,
          // eslint-disable-next-line no-script-url
          code_source_url: 'javascript:alert(1)',
        }}
      />,
    );
    expect(screen.getByText('Version one')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Version source' })).not.toBeInTheDocument();
  });

  it('shows pipeline metadata without a selected version', () => {
    render(<PipelineVersionCard pipeline={pipeline} selectedVersion={undefined} />);
    expect(screen.getByText('pipeline-id')).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Version details' })).not.toBeInTheDocument();
  });

  it('does not invent metadata before the pipeline loads', () => {
    const { container } = render(
      <PipelineVersionCard pipeline={null} selectedVersion={undefined} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
