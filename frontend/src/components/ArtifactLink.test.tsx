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

import * as React from 'react';
import { render, screen } from '@testing-library/react';
import { ArtifactLink } from './ArtifactLink';

describe('ArtifactLink', () => {
  it('renders nothing when artifactUri is undefined', () => {
    const { container } = render(<ArtifactLink />);
    expect(container.textContent).toBe('');
  });

  it('renders plain text when artifactUri is not a recognized scheme', () => {
    render(<ArtifactLink artifactUri='some/local/path' />);
    expect(screen.getByText('some/local/path')).toBeInTheDocument();
    expect(screen.queryByRole('link')).toBeNull();
  });

  it('renders a clickable link for gs:// URIs', () => {
    render(<ArtifactLink artifactUri='gs://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noreferrer noopener');
    expect(link).toHaveTextContent('gs://my-bucket/my-object');
  });

  it('renders a clickable link for http:// URIs', () => {
    render(<ArtifactLink artifactUri='http://example.com/artifact' />);
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', 'http://example.com/artifact');
    expect(link).toHaveTextContent('http://example.com/artifact');
  });

  it('renders a clickable link for https:// URIs', () => {
    render(<ArtifactLink artifactUri='https://example.com/artifact' />);
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', 'https://example.com/artifact');
  });

  it('renders a clickable link for s3:// URIs', () => {
    render(<ArtifactLink artifactUri='s3://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveTextContent('s3://my-bucket/my-object');
  });

  it('renders a clickable link for minio:// URIs', () => {
    render(<ArtifactLink artifactUri='minio://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveTextContent('minio://my-bucket/my-object');
  });

  it('renders an empty string when artifactUri is empty', () => {
    const { container } = render(<ArtifactLink artifactUri='' />);
    expect(container.textContent).toBe('');
    expect(screen.queryByRole('link')).toBeNull();
  });
});
