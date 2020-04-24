/*
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import MinioArtifactLink from './MinioArtifactLink';
import { render } from '@testing-library/react';

describe('MinioArtifactLink', () => {
  it('handles undefined artifact', () => {
    const { container } = render(<MinioArtifactLink artifact={undefined} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles null artifact', () => {
    const { container } = render(<MinioArtifactLink artifact={null} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles empty artifact', () => {
    const { container } = render(<MinioArtifactLink artifact={{}} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles invalid artifact: no bucket', () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: '',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactLink artifact={s3Artifact} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles invalid artifact: no key', () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: '',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactLink artifact={s3Artifact} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles s3 artifact', () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 's3.amazonaws.com',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactLink artifact={s3Artifact} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <a
          href="artifacts/get?source=s3&bucket=foo&key=bar"
          rel="noreferrer noopener"
          target="_blank"
          title="s3://foo/bar"
        >
          s3://foo/bar
        </a>
      </div>
    `);
  });

  it('handles minio artifact', () => {
    const minioArtifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactLink artifact={minioArtifact} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <a
          href="artifacts/get?source=minio&bucket=foo&key=bar"
          rel="noreferrer noopener"
          target="_blank"
          title="minio://foo/bar"
        >
          minio://foo/bar
        </a>
      </div>
    `);
  });
});
