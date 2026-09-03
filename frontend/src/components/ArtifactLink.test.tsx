/*
 * Copyright 2026 The Kubeflow Authors
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

  it('renders a clickable link for gs:// URIs with correct href', () => {
    render(<ArtifactLink artifactUri='gs://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noreferrer noopener');
    expect(link).toHaveAttribute(
      'href',
      'https://console.cloud.google.com/storage/browser/my-bucket/my-object',
    );
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

  it('renders a clickable link for s3:// URIs with generated href', () => {
    render(<ArtifactLink artifactUri='s3://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveTextContent('s3://my-bucket/my-object');
    expect(link).toHaveAttribute('href');
    expect(link.getAttribute('href')).not.toBe('');
  });

  it('renders malformed S3 URIs as plain text', () => {
    render(<ArtifactLink artifactUri='s3:not-a-storage-path' namespace='team-a' />);

    expect(screen.getByText('s3:not-a-storage-path')).toBeInTheDocument();
    expect(screen.queryByRole('link')).toBeNull();
  });

  it('separates the S3 artifact URI query from the object key and includes the namespace', () => {
    render(
      <ArtifactLink
        artifactUri='s3://my-bucket/my-object?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph'
        namespace='team-a'
      />,
    );

    const link = screen.getByRole('link');
    const [path, query] = (link.getAttribute('href') || '').split('?');
    expect(path).toBe('artifacts/s3/my-bucket/my-object');
    const params = new URLSearchParams(query);
    expect(params.get('namespace')).toBe('team-a');
    expect(params.get('artifactUriQuery')).toBe(
      'endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
    );
    expect(params.has('providerInfo')).toBe(false);
  });

  it('renders a clickable link for minio:// URIs with generated href', () => {
    render(<ArtifactLink artifactUri='minio://my-bucket/my-object' />);
    const link = screen.getByRole('link');
    expect(link).toHaveTextContent('minio://my-bucket/my-object');
    expect(link).toHaveAttribute('href');
    expect(link.getAttribute('href')).not.toBe('');
  });

  it('removes GCS provider parameters from the external console URL', () => {
    render(<ArtifactLink artifactUri='gs://my-bucket/my-object?endpoint=storage.example' />);

    expect(screen.getByRole('link')).toHaveAttribute(
      'href',
      'https://console.cloud.google.com/storage/browser/my-bucket/my-object',
    );
  });

  it('renders an empty string when artifactUri is empty', () => {
    const { container } = render(<ArtifactLink artifactUri='' />);
    expect(container.textContent).toBe('');
    expect(screen.queryByRole('link')).toBeNull();
  });
});
